package redis

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/app/tools/router"
	"github.com/goback/pkg/logger"
	"go.uber.org/zap"
)

// PubSubMessage 发布/订阅消息
type PubSubMessage struct {
	Topic     string    `json:"topic"`
	Sender    string    `json:"sender"`
	Payload   []byte    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// Subscriber 订阅者信息
type Subscriber struct {
	Service      string    `json:"service"`
	CallbackAddr string    `json:"callback_addr"`
	Topics       []string  `json:"topics"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	Service      string   `json:"service"`
	CallbackAddr string   `json:"callback_addr"`
	Topics       []string `json:"topics"`
}

// PublishRequest 发布请求
type PublishRequest struct {
	Topic   string `json:"topic"`
	Sender  string `json:"sender"`
	Payload []byte `json:"payload"`
}

// PubSubService 发布/订阅服务
type PubSubService struct {
	subscribers map[string]*Subscriber // service -> Subscriber
	topicIndex  map[string][]string    // topic -> []service names
	mu          sync.RWMutex
	client      *http.Client
	stopCleanup chan struct{}
}

// NewPubSubService 创建 PubSub 服务
func NewPubSubService() *PubSubService {
	ps := &PubSubService{
		subscribers: make(map[string]*Subscriber),
		topicIndex:  make(map[string][]string),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		stopCleanup: make(chan struct{}),
	}
	go ps.cleanupLoop()
	return ps
}

// cleanupLoop 定期清理过期订阅
func (ps *PubSubService) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ps.cleanupExpired()
		case <-ps.stopCleanup:
			return
		}
	}
}

// cleanupExpired 清理过期订阅（超过 2 分钟未更新）
func (ps *PubSubService) cleanupExpired() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	expireTime := time.Now().Add(-2 * time.Minute)
	for service, sub := range ps.subscribers {
		if sub.UpdatedAt.Before(expireTime) {
			logger.Debug("removing expired subscriber", zap.String("service", service))
			delete(ps.subscribers, service)
			// 从 topicIndex 中移除
			for topic := range ps.topicIndex {
				ps.topicIndex[topic] = removeFromSlice(ps.topicIndex[topic], service)
			}
		}
	}
}

// RegisterRoutes 注册 HTTP 路由
func (ps *PubSubService) RegisterRoutes(r *router.Router[*core.RequestEvent]) {
	r.POST("/pubsub/subscribe", ps.handleSubscribe)
	r.POST("/pubsub/publish", ps.handlePublish)
	r.GET("/pubsub/subscribers", ps.handleListSubscribers)
}

// handleSubscribe 处理订阅请求
func (ps *PubSubService) handleSubscribe(e *core.RequestEvent) error {
	var req SubscribeRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	if req.Service == "" || req.CallbackAddr == "" {
		return apis.Error(e, 400, "service and callback_addr are required")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// 更新或创建订阅者
	existing, exists := ps.subscribers[req.Service]
	if exists {
		// 更新现有订阅者
		existing.CallbackAddr = req.CallbackAddr
		existing.UpdatedAt = time.Now()

		// 合并主题列表
		topicSet := make(map[string]bool)
		for _, t := range existing.Topics {
			topicSet[t] = true
		}
		for _, t := range req.Topics {
			topicSet[t] = true
		}
		existing.Topics = make([]string, 0, len(topicSet))
		for t := range topicSet {
			existing.Topics = append(existing.Topics, t)
		}
	} else {
		// 创建新订阅者
		ps.subscribers[req.Service] = &Subscriber{
			Service:      req.Service,
			CallbackAddr: req.CallbackAddr,
			Topics:       req.Topics,
			UpdatedAt:    time.Now(),
		}
	}

	// 更新主题索引
	for _, topic := range req.Topics {
		if !containsString(ps.topicIndex[topic], req.Service) {
			ps.topicIndex[topic] = append(ps.topicIndex[topic], req.Service)
		}
	}

	logger.Debug("subscriber registered",
		zap.String("service", req.Service),
		zap.String("callback", req.CallbackAddr),
		zap.Strings("topics", req.Topics),
	)

	return e.JSON(200, map[string]any{"ok": true})
}

// handlePublish 处理发布请求
func (ps *PubSubService) handlePublish(e *core.RequestEvent) error {
	var req PublishRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	if req.Topic == "" {
		return apis.Error(e, 400, "topic is required")
	}

	msg := &PubSubMessage{
		Topic:     req.Topic,
		Sender:    req.Sender,
		Payload:   req.Payload,
		Timestamp: time.Now(),
	}

	// 获取订阅该主题的服务列表
	ps.mu.RLock()
	services := ps.topicIndex[req.Topic]
	subscribers := make([]*Subscriber, 0, len(services))
	for _, svc := range services {
		if sub, ok := ps.subscribers[svc]; ok {
			// 不发送给发送者自己
			if sub.Service != req.Sender {
				subscribers = append(subscribers, sub)
			}
		}
	}
	ps.mu.RUnlock()

	// 异步推送给所有订阅者
	data, err := json.Marshal(msg)
	if err != nil {
		return apis.Error(e, 500, "marshal message failed")
	}

	for _, sub := range subscribers {
		go ps.pushToSubscriber(sub, data)
	}

	logger.Debug("message published",
		zap.String("topic", req.Topic),
		zap.String("sender", req.Sender),
		zap.Int("subscribers", len(subscribers)),
	)

	return e.JSON(200, map[string]any{
		"ok":          true,
		"subscribers": len(subscribers),
	})
}

// handleListSubscribers 列出所有订阅者
func (ps *PubSubService) handleListSubscribers(e *core.RequestEvent) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make([]*Subscriber, 0, len(ps.subscribers))
	for _, sub := range ps.subscribers {
		result = append(result, sub)
	}

	return e.JSON(200, map[string]any{
		"subscribers": result,
		"topics":      ps.topicIndex,
	})
}

// pushToSubscriber 推送消息给订阅者
func (ps *PubSubService) pushToSubscriber(sub *Subscriber, data []byte) {
	url := "http://" + sub.CallbackAddr + "/_pubsub"
	resp, err := ps.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Debug("push to subscriber failed",
			zap.String("service", sub.Service),
			zap.String("callback", sub.CallbackAddr),
			zap.Error(err),
		)
		return
	}
	resp.Body.Close()
}

// Stop 停止服务
func (ps *PubSubService) Stop() {
	close(ps.stopCleanup)
}

// 辅助函数
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeFromSlice(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
