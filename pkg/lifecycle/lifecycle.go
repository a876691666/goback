package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/goback/pkg/database"
	"github.com/goback/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Event 生命周期事件类型
type Event string

const (
	EventStarting  Event = "starting"   // 服务启动中
	EventStarted   Event = "started"    // 服务已启动
	EventReady     Event = "ready"      // 服务就绪（可接收请求）
	EventStopping  Event = "stopping"   // 服务停止中
	EventStopped   Event = "stopped"    // 服务已停止
	EventHealthy   Event = "healthy"    // 健康检查通过
	EventUnhealthy Event = "unhealthy"  // 健康检查失败
)

// LifecycleMessage 生命周期消息
type LifecycleMessage struct {
	Service   string    `json:"service"`    // 服务名称
	NodeID    string    `json:"node_id"`    // 节点ID
	Event     Event     `json:"event"`      // 事件类型
	Timestamp time.Time `json:"timestamp"`  // 时间戳
	Metadata  any       `json:"metadata"`   // 附加元数据
}

// LifecycleHandler 生命周期事件处理器
type LifecycleHandler func(msg *LifecycleMessage)

// Manager 生命周期管理器
type Manager struct {
	service   string
	nodeID    string
	redis     *redis.Client
	handlers  map[Event][]LifecycleHandler
	allHandlers []LifecycleHandler // 监听所有事件的处理器
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	pubsub    *redis.PubSub
}

const lifecycleChannel = "service:lifecycle"

// NewManager 创建生命周期管理器
func NewManager(service, nodeID string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		service:     service,
		nodeID:      nodeID,
		redis:       database.GetRedis(),
		handlers:    make(map[Event][]LifecycleHandler),
		allHandlers: make([]LifecycleHandler, 0),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// OnEvent 监听特定生命周期事件
func (m *Manager) OnEvent(event Event, handler LifecycleHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[event] = append(m.handlers[event], handler)
}

// OnAnyEvent 监听所有生命周期事件
func (m *Manager) OnAnyEvent(handler LifecycleHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allHandlers = append(m.allHandlers, handler)
}

// Emit 发布生命周期事件
func (m *Manager) Emit(event Event, metadata any) error {
	msg := &LifecycleMessage{
		Service:   m.service,
		NodeID:    m.nodeID,
		Event:     event,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal lifecycle message: %w", err)
	}

	return m.redis.Publish(m.ctx, lifecycleChannel, data).Err()
}

// Start 启动生命周期监听
func (m *Manager) Start() error {
	m.pubsub = m.redis.Subscribe(m.ctx, lifecycleChannel)

	// 等待订阅确认
	_, err := m.pubsub.Receive(m.ctx)
	if err != nil {
		return fmt.Errorf("subscribe lifecycle channel: %w", err)
	}

	go m.listen()

	logger.Info("生命周期管理器已启动",
		zap.String("service", m.service),
		zap.String("node_id", m.nodeID),
	)

	return nil
}

// listen 监听生命周期消息
func (m *Manager) listen() {
	ch := m.pubsub.Channel()

	for {
		select {
		case <-m.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			m.handleMessage(msg.Payload)
		}
	}
}

// handleMessage 处理生命周期消息
func (m *Manager) handleMessage(payload string) {
	var msg LifecycleMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		logger.Error("解析生命周期消息失败", zap.Error(err))
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 调用特定事件处理器
	if handlers, ok := m.handlers[msg.Event]; ok {
		for _, handler := range handlers {
			go handler(&msg)
		}
	}

	// 调用全局处理器
	for _, handler := range m.allHandlers {
		go handler(&msg)
	}
}

// Stop 停止生命周期监听
func (m *Manager) Stop() error {
	m.cancel()
	if m.pubsub != nil {
		return m.pubsub.Close()
	}
	return nil
}

// EmitStarting 发布启动中事件
func (m *Manager) EmitStarting() error {
	return m.Emit(EventStarting, nil)
}

// EmitStarted 发布已启动事件
func (m *Manager) EmitStarted() error {
	return m.Emit(EventStarted, nil)
}

// EmitReady 发布就绪事件
func (m *Manager) EmitReady() error {
	return m.Emit(EventReady, nil)
}

// EmitStopping 发布停止中事件
func (m *Manager) EmitStopping() error {
	return m.Emit(EventStopping, nil)
}

// EmitStopped 发布已停止事件
func (m *Manager) EmitStopped() error {
	return m.Emit(EventStopped, nil)
}
