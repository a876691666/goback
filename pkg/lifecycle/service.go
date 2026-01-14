package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goback/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
)

// ServiceOptions 服务配置选项
type ServiceOptions struct {
	Name     string            // 服务名称
	NodeID   string            // 节点ID
	Address  string            // 服务地址
	Registry registry.Registry // 服务注册中心
	Service  *registry.Service // 服务注册信息
}

// Service 微服务包装器
type Service struct {
	opts      *ServiceOptions
	app       *fiber.App
	lifecycle *Manager
	cache     *CacheBroadcaster
	ctx       *ServiceContext // 服务上下文

	// 钩子函数
	onStart []func(*ServiceContext) error
	onReady []func(*ServiceContext) error
	onStop  []func(*ServiceContext) error
}

// NewService 创建微服务
func NewService(opts *ServiceOptions) *Service {
	s := &Service{
		opts:      opts,
		lifecycle: NewManager(opts.Name, opts.NodeID),
		cache:     NewCacheBroadcaster(opts.Name),
		onStart:   make([]func(*ServiceContext) error, 0),
		onReady:   make([]func(*ServiceContext) error, 0),
		onStop:    make([]func(*ServiceContext) error, 0),
	}
	// 创建服务上下文
	s.ctx = newServiceContext(s)
	return s
}

// SetApp 设置Fiber应用
func (s *Service) SetApp(app *fiber.App) {
	s.app = app
}

// Lifecycle 获取生命周期管理器
func (s *Service) Lifecycle() *Manager {
	return s.lifecycle
}

// Cache 获取缓存广播器
func (s *Service) Cache() *CacheBroadcaster {
	return s.cache
}

// Context 获取服务上下文
func (s *Service) Context() *ServiceContext {
	return s.ctx
}

// OnStart 注册启动钩子
func (s *Service) OnStart(fn func(*ServiceContext) error) {
	s.onStart = append(s.onStart, fn)
}

// OnReady 注册就绪钩子
func (s *Service) OnReady(fn func(*ServiceContext) error) {
	s.onReady = append(s.onReady, fn)
}

// OnStop 注册停止钩子
func (s *Service) OnStop(fn func(*ServiceContext) error) {
	s.onStop = append(s.onStop, fn)
}

// Run 运行服务
func (s *Service) Run() error {
	// 启动生命周期监听
	if err := s.lifecycle.Start(); err != nil {
		return fmt.Errorf("start lifecycle manager: %w", err)
	}

	// 启动缓存广播监听
	if err := s.cache.Start(); err != nil {
		return fmt.Errorf("start cache broadcaster: %w", err)
	}

	// 发布启动中事件
	s.lifecycle.EmitStarting()

	// 执行启动钩子
	for _, fn := range s.onStart {
		if err := fn(s.ctx); err != nil {
			return fmt.Errorf("start hook: %w", err)
		}
	}

	// 注册服务
	if s.opts.Registry != nil && s.opts.Service != nil {
		if err := s.opts.Registry.Register(s.opts.Service); err != nil {
			return fmt.Errorf("register service: %w", err)
		}
	}

	// 发布已启动事件
	s.lifecycle.EmitStarted()

	// 启动HTTP服务
	errCh := make(chan error, 1)
	go func() {
		logger.Info("服务启动",
			zap.String("service", s.opts.Name),
			zap.String("address", s.opts.Address),
		)
		if err := s.app.Listen(s.opts.Address); err != nil {
			errCh <- err
		}
	}()

	// 等待服务启动
	time.Sleep(100 * time.Millisecond)

	// 执行就绪钩子
	for _, fn := range s.onReady {
		if err := fn(s.ctx); err != nil {
			return fmt.Errorf("ready hook: %w", err)
		}
	}

	// 发布就绪事件
	s.lifecycle.EmitReady()

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

// Shutdown 优雅关闭服务
func (s *Service) Shutdown() error {
	// 发布停止中事件
	s.lifecycle.EmitStopping()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx

	// 执行停止钩子
	for _, fn := range s.onStop {
		if err := fn(s.ctx); err != nil {
			logger.Error("停止钩子执行失败", zap.Error(err))
		}
	}

	// 注销服务
	if s.opts.Registry != nil && s.opts.Service != nil {
		if err := s.opts.Registry.Deregister(s.opts.Service); err != nil {
			logger.Error("注销服务失败", zap.Error(err))
		}
	}

	// 关闭HTTP服务
	if s.app != nil {
		if err := s.app.Shutdown(); err != nil {
			logger.Error("关闭HTTP服务失败", zap.Error(err))
		}
	}

	// 停止缓存广播
	if err := s.cache.Stop(); err != nil {
		logger.Error("停止缓存广播失败", zap.Error(err))
	}

	// 发布已停止事件
	s.lifecycle.EmitStopped()

	// 停止生命周期监听
	if err := s.lifecycle.Stop(); err != nil {
		logger.Error("停止生命周期监听失败", zap.Error(err))
	}

	logger.Info("服务已关闭", zap.String("service", s.opts.Name))
	return nil
}
