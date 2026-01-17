package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go-micro.dev/v5/registry"
)

// PubSubMessage 发布/订阅消息
type PubSubMessage struct {
	Topic     string    `json:"topic"`
	Sender    string    `json:"sender"`    // 发送者服务名
	Payload   []byte    `json:"payload"`   // 消息内容
	Timestamp time.Time `json:"timestamp"` // 发送时间
}

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	Service      string   `json:"service"`       // 订阅服务名
	CallbackAddr string   `json:"callback_addr"` // 回调地址（HTTP）
	Topics       []string `json:"topics"`        // 订阅的主题列表
}

// PublishRequest 发布请求
type PublishRequest struct {
	Topic   string `json:"topic"`
	Sender  string `json:"sender"`
	Payload []byte `json:"payload"`
}

// PubSubHandler 消息处理函数
type PubSubHandler func(msg *PubSubMessage)

// PubSubOption 配置选项
type PubSubOption func(*PubSub)

// PubSub 基于 Redis 服务的发布/订阅客户端
type PubSub struct {
	service      string            // 本服务名
	callbackAddr string            // 本服务回调地址
	redisAddr    string            // Redis 服务地址
	registry     registry.Registry // 服务注册中心（用于动态发现 Redis 服务）
	handlers     map[string][]PubSubHandler
	mu           sync.RWMutex
	client       *http.Client
	logger       *slog.Logger
	started      bool
	stopCh       chan struct{}
}

// NewPubSub 创建 PubSub 客户端
func NewPubSub(service, callbackAddr string, opts ...PubSubOption) *PubSub {
	ps := &PubSub{
		service:      service,
		callbackAddr: callbackAddr,
		handlers:     make(map[string][]PubSubHandler),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: slog.Default(),
		stopCh: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(ps)
	}

	return ps
}

// WithRedisAddr 设置 Redis 服务地址（静态配置）
func WithRedisAddr(addr string) PubSubOption {
	return func(ps *PubSub) {
		ps.redisAddr = addr
	}
}

// WithPubSubRegistry 设置服务注册中心（动态发现 Redis 服务）
func WithPubSubRegistry(reg registry.Registry) PubSubOption {
	return func(ps *PubSub) {
		ps.registry = reg
	}
}

// WithPubSubLogger 设置日志
func WithPubSubLogger(logger *slog.Logger) PubSubOption {
	return func(ps *PubSub) {
		ps.logger = logger
	}
}

// Start 启动 PubSub 客户端
func (ps *PubSub) Start() error {
	ps.mu.Lock()
	if ps.started {
		ps.mu.Unlock()
		return nil
	}
	ps.started = true
	topics := make([]string, 0, len(ps.handlers))
	for topic := range ps.handlers {
		topics = append(topics, topic)
	}
	ps.mu.Unlock()

	// 向 Redis 服务注册订阅
	if len(topics) > 0 {
		if err := ps.registerSubscription(topics); err != nil {
			ps.logger.Warn("initial subscription registration failed, will retry", "error", err)
		}
	}

	// 启动心跳/重新注册协程
	go ps.heartbeatLoop()

	ps.logger.Info("pubsub client started",
		"service", ps.service,
		"callback", ps.callbackAddr,
	)
	return nil
}

// Stop 停止 PubSub 客户端
func (ps *PubSub) Stop() error {
	ps.mu.Lock()
	if !ps.started {
		ps.mu.Unlock()
		return nil
	}
	ps.started = false
	ps.mu.Unlock()

	close(ps.stopCh)
	ps.logger.Info("pubsub client stopped", "service", ps.service)
	return nil
}

// Subscribe 订阅主题
func (ps *PubSub) Subscribe(topic string, handler PubSubHandler) error {
	ps.mu.Lock()
	ps.handlers[topic] = append(ps.handlers[topic], handler)
	started := ps.started
	ps.mu.Unlock()

	// 如果已启动，立即向 Redis 服务注册
	if started {
		if err := ps.registerSubscription([]string{topic}); err != nil {
			ps.logger.Warn("subscription registration failed", "topic", topic, "error", err)
		}
	}

	return nil
}

// Publish 发布消息
func (ps *PubSub) Publish(topic string, payload []byte) error {
	redisAddr := ps.getRedisAddr()
	if redisAddr == "" {
		return fmt.Errorf("redis service not available")
	}

	req := PublishRequest{
		Topic:   topic,
		Sender:  ps.service,
		Payload: payload,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal publish request: %w", err)
	}

	url := fmt.Sprintf("http://%s/pubsub/publish", redisAddr)
	resp, err := ps.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("publish to redis service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish failed with status: %d", resp.StatusCode)
	}

	return nil
}

// PublishJSON 发布 JSON 消息
func (ps *PubSub) PublishJSON(topic string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	return ps.Publish(topic, payload)
}

// Handler 返回 HTTP 处理器（用于接收 Redis 服务推送的消息）
func (ps *PubSub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var msg PubSubMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 不处理自己发送的消息
		if msg.Sender == ps.service {
			w.WriteHeader(http.StatusOK)
			return
		}

		ps.handleMessage(&msg)
		w.WriteHeader(http.StatusOK)
	}
}

// handleMessage 处理接收到的消息
func (ps *PubSub) handleMessage(msg *PubSubMessage) {
	ps.mu.RLock()
	handlers := ps.handlers[msg.Topic]
	ps.mu.RUnlock()

	for _, handler := range handlers {
		go handler(msg)
	}
}

// HandleMessage 公开的消息处理方法（供外部调用）
func (ps *PubSub) HandleMessage(msg *PubSubMessage) {
	ps.handleMessage(msg)
}

// registerSubscription 向 Redis 服务注册订阅
func (ps *PubSub) registerSubscription(topics []string) error {
	redisAddr := ps.getRedisAddr()
	if redisAddr == "" {
		return fmt.Errorf("redis service not available")
	}

	req := SubscribeRequest{
		Service:      ps.service,
		CallbackAddr: ps.callbackAddr,
		Topics:       topics,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal subscribe request: %w", err)
	}

	url := fmt.Sprintf("http://%s/pubsub/subscribe", redisAddr)
	resp, err := ps.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("subscribe to redis service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscribe failed with status: %d", resp.StatusCode)
	}

	return nil
}

// getRedisAddr 获取 Redis 服务地址
func (ps *PubSub) getRedisAddr() string {
	// 优先使用静态配置
	if ps.redisAddr != "" {
		return ps.redisAddr
	}

	// 从注册中心发现
	if ps.registry != nil {
		services, err := ps.registry.GetService("redis-service")
		if err == nil && len(services) > 0 && len(services[0].Nodes) > 0 {
			return services[0].Nodes[0].Address
		}
	}

	return ""
}

// heartbeatLoop 心跳循环，定期重新注册订阅
func (ps *PubSub) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ps.mu.RLock()
			topics := make([]string, 0, len(ps.handlers))
			for topic := range ps.handlers {
				topics = append(topics, topic)
			}
			ps.mu.RUnlock()

			if len(topics) > 0 {
				if err := ps.registerSubscription(topics); err != nil {
					ps.logger.Debug("heartbeat subscription failed", "error", err)
				}
			}
		case <-ps.stopCh:
			return
		}
	}
}

// Service 返回服务名
func (ps *PubSub) Service() string {
	return ps.service
}
