package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/goback/pkg/broadcast"
	"github.com/goback/pkg/cache"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/logger"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/gofiber/fiber/v2"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
)

// Event 生命周期事件类型
type Event string

const (
	EventStarting  Event = "starting"
	EventStarted   Event = "started"
	EventReady     Event = "ready"
	EventStopping  Event = "stopping"
	EventStopped   Event = "stopped"
	EventHealthy   Event = "healthy"
	EventUnhealthy Event = "unhealthy"
)

const lifecycleTopic = "service:lifecycle"

// EventMessage 生命周期事件消息
type EventMessage struct {
	Service   string    `json:"service"`
	NodeID    string    `json:"node_id"`
	Event     Event     `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  any       `json:"metadata,omitempty"`
}

// EventHandler 事件处理器
type EventHandler func(*EventMessage, *Service)

// Hook 钩子函数
type Hook func(*Service) error

// Service 微服务 - 统一的服务生命周期管理
type Service struct {
	name    string
	nodeID  string
	address string

	app         *fiber.App
	registry    registry.Registry
	regService  *registry.Service
	broadcaster *broadcast.Broadcaster

	onStart []Hook
	onReady []Hook
	onStop  []Hook

	rbac *RBACData

	eventHandlers    map[Event][]EventHandler
	allEventHandlers []EventHandler
	mu               sync.RWMutex
}

// New 创建服务
func New(name string) *Service {
	return &Service{
		name:             name,
		nodeID:           name + "-1",
		eventHandlers:    make(map[Event][]EventHandler),
		allEventHandlers: make([]EventHandler, 0),
	}
}

// --- 链式配置方法 ---

// Node 设置节点ID
func (s *Service) Node(id string) *Service {
	s.nodeID = id
	return s
}

// Addr 设置服务地址
func (s *Service) Addr(addr string) *Service {
	s.address = addr
	return s
}

// App 设置Fiber应用
func (s *Service) App(app *fiber.App) *Service {
	s.app = app
	return s
}

// Registry 设置服务注册中心
func (s *Service) Registry(reg registry.Registry) *Service {
	s.registry = reg
	return s
}

// RegInfo 设置服务注册信息
func (s *Service) RegInfo(svc *registry.Service) *Service {
	s.regService = svc
	return s
}

// --- 钩子注册 ---

// OnStart 启动时执行
func (s *Service) OnStart(fn Hook) *Service {
	s.onStart = append(s.onStart, fn)
	return s
}

// OnReady 就绪时执行
func (s *Service) OnReady(fn Hook) *Service {
	s.onReady = append(s.onReady, fn)
	return s
}

// OnStop 停止时执行
func (s *Service) OnStop(fn Hook) *Service {
	s.onStop = append(s.onStop, fn)
	return s
}

// --- 工具函数 ---

// UseRBAC 使用RBAC功能，订阅RBAC数据更新
func (s *Service) UseRBAC() *Service {
	s.OnReady(func(s *Service) error {
		s.Broadcaster().Subscribe(KeyRBACData, func(msg *broadcast.Message) {
			var data RBACData
			if err := json.Unmarshal(msg.Payload, &data); err != nil {
				logger.Error("解析RBAC数据失败", zap.Error(err))
				return
			}
			s.rbac = &data
		})
		return nil
	})
	return s
}

// --- 事件监听 ---

// On 监听特定事件
func (s *Service) On(event Event, handler EventHandler) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandlers[event] = append(s.eventHandlers[event], handler)
	return s
}

// OnAny 监听所有事件
func (s *Service) OnAny(handler EventHandler) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allEventHandlers = append(s.allEventHandlers, handler)
	return s
}

// --- 广播能力 ---

// Broadcaster 获取广播器
func (s *Service) Broadcaster() *broadcast.Broadcaster {
	return s.broadcaster
}

// Emit 发布事件
func (s *Service) Emit(event Event, metadata any) error {
	if s.broadcaster == nil {
		return nil
	}
	msg := &EventMessage{
		Service:   s.name,
		NodeID:    s.nodeID,
		Event:     event,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.broadcaster.Send(lifecycleTopic, data, "")
}

// --- 核心方法 ---

// Run 启动服务
func (s *Service) Run() error {
	// 初始化 Redis 缓存客户端
	cfg := config.Get()
	if cfg != nil && cfg.Redis.Host != "" {
		cache.Init(cfg.Redis.Host, cfg.Redis.Port)
		logger.Debug("缓存客户端已初始化",
			zap.String("host", cfg.Redis.Host),
			zap.Int("port", cfg.Redis.Port),
		)
	}

	// 初始化注册中心
	if s.registry == nil {
		s.registry = pkgRegistry.NewRedisRegistry()
	}

	// 初始化广播器
	s.broadcaster = broadcast.New(s.name, s.nodeID, s.registry)
	s.mountBroadcastRoute()
	s.broadcaster.Subscribe(lifecycleTopic, s.handleBroadcast)
	if err := s.broadcaster.Start(); err != nil {
		return fmt.Errorf("start broadcaster: %w", err)
	}

	// 发布启动中事件
	s.Emit(EventStarting, nil)

	// 执行启动钩子
	for _, fn := range s.onStart {
		if err := fn(s); err != nil {
			return fmt.Errorf("start hook: %w", err)
		}
	}

	// 注册服务
	if s.registry != nil && s.regService != nil {
		if err := s.registry.Register(s.regService); err != nil {
			return fmt.Errorf("register service: %w", err)
		}
	}

	// 发布已启动事件
	s.Emit(EventStarted, nil)

	// 启动HTTP服务
	errCh := make(chan error, 1)
	go func() {
		logger.Info("服务启动", zap.String("service", s.name), zap.String("address", s.address))
		if err := s.app.Listen(s.address); err != nil {
			errCh <- err
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// 执行就绪钩子
	for _, fn := range s.onReady {
		if err := fn(s); err != nil {
			return fmt.Errorf("ready hook: %w", err)
		}
	}

	// 发布就绪事件
	s.Emit(EventReady, nil)

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("收到退出信号，正在关闭服务...")
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	return s.Shutdown()
}

// Shutdown 优雅关闭
func (s *Service) Shutdown() error {
	s.Emit(EventStopping, nil)

	// 执行停止钩子
	for _, fn := range s.onStop {
		if err := fn(s); err != nil {
			logger.Error("停止钩子执行失败", zap.Error(err))
		}
	}

	// 注销服务
	if s.registry != nil && s.regService != nil {
		if err := s.registry.Deregister(s.regService); err != nil {
			logger.Error("注销服务失败", zap.Error(err))
		}
	}

	// 关闭HTTP服务
	if s.app != nil {
		if err := s.app.Shutdown(); err != nil {
			logger.Error("关闭HTTP服务失败", zap.Error(err))
		}
	}

	s.Emit(EventStopped, nil)

	logger.Info("服务已关闭", zap.String("service", s.name))
	return nil
}

// --- 内部方法 ---

func (s *Service) mountBroadcastRoute() {
	if s.app == nil {
		return
	}
	s.app.Post("/_broadcast", func(c *fiber.Ctx) error {
		var msg broadcast.Message
		if err := c.BodyParser(&msg); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		// 统一由广播器处理所有消息（包括生命周期和其他订阅）
		s.broadcaster.HandleMessage(&msg)
		return c.SendStatus(fiber.StatusOK)
	})
}

func (s *Service) handleBroadcast(msg *broadcast.Message) {
	var evt EventMessage
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		logger.Error("解析事件消息失败", zap.Error(err))
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if handlers, ok := s.eventHandlers[evt.Event]; ok {
		for _, h := range handlers {
			go h(&evt, s)
		}
	}
	for _, h := range s.allEventHandlers {
		go h(&evt, s)
	}
}
