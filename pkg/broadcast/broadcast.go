package broadcast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/goback/pkg/logger"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
)

// Message 广播消息
type Message struct {
	Topic     string    `json:"topic"`
	Service   string    `json:"service"`   // 发送者服务名
	NodeID    string    `json:"node_id"`   // 发送者节点ID
	Target    string    `json:"target"`    // 目标服务名，空表示广播所有
	Payload   []byte    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// Handler 消息处理器
type Handler func(msg *Message)

// Broadcaster 广播器
type Broadcaster struct {
	service     string
	nodeID      string
	registry    registry.Registry
	subscribers map[string][]Handler
	mu          sync.RWMutex
	client      *http.Client
	ctx         context.Context
	cancel      context.CancelFunc
}

// New 创建广播器
func New(service, nodeID string, reg registry.Registry) *Broadcaster {
	ctx, cancel := context.WithCancel(context.Background())
	return &Broadcaster{
		service:     service,
		nodeID:      nodeID,
		registry:    reg,
		subscribers: make(map[string][]Handler),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Subscribe 订阅 topic
func (b *Broadcaster) Subscribe(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[topic] = append(b.subscribers[topic], handler)
}

// Send 发送消息（核心方法）
// target 为空时广播到所有服务，否则只发送到指定服务
func (b *Broadcaster) Send(ctx context.Context, topic string, payload []byte, target string) error {
	msg := &Message{
		Topic:     topic,
		Service:   b.service,
		NodeID:    b.nodeID,
		Target:    target,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	// 本地先处理（仅广播时，定向发送不处理本地）
	if target == "" {
		b.handleMessage(msg)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// 获取目标节点
	nodes := b.getTargetNodes(target)
	for _, node := range nodes {
		go b.sendToNode(node.Address, data)
	}

	return nil
}

// Publish 广播到所有服务
func (b *Broadcaster) Publish(ctx context.Context, topic string, payload []byte) error {
	return b.Send(ctx, topic, payload, "")
}

// SendTo 发送到指定服务
func (b *Broadcaster) SendTo(ctx context.Context, topic string, payload []byte, target string) error {
	return b.Send(ctx, topic, payload, target)
}

// getTargetNodes 获取目标节点列表
func (b *Broadcaster) getTargetNodes(target string) []*registry.Node {
	var nodes []*registry.Node

	if target != "" {
		// 定向发送：只获取目标服务的节点
		services, err := b.registry.GetService(target)
		if err != nil {
			return nil
		}
		for _, svc := range services {
			nodes = append(nodes, svc.Nodes...)
		}
	} else {
		// 广播：获取所有服务的节点
		services, err := b.registry.ListServices()
		if err != nil {
			return nil
		}
		for _, svc := range services {
			fullSvc, err := b.registry.GetService(svc.Name)
			if err != nil {
				continue
			}
			for _, s := range fullSvc {
				nodes = append(nodes, s.Nodes...)
			}
		}
	}

	// 过滤掉自己
	result := make([]*registry.Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Id != b.nodeID {
			result = append(result, node)
		}
	}
	return result
}

// sendToNode 发送消息到指定节点
func (b *Broadcaster) sendToNode(addr string, data []byte) {
	url := fmt.Sprintf("http://%s/_broadcast", addr)
	resp, err := b.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Debug("发送消息失败", zap.String("addr", addr), zap.Error(err))
		return
	}
	resp.Body.Close()
}

// Handler 返回 HTTP 处理器
func (b *Broadcaster) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		b.handleMessage(&msg)
		w.WriteHeader(http.StatusOK)
	}
}

// handleMessage 处理消息
func (b *Broadcaster) handleMessage(msg *Message) {
	// 目标过滤：如果指定了目标且不是自己，忽略
	if msg.Target != "" && msg.Target != b.service {
		return
	}

	b.mu.RLock()
	handlers := b.subscribers[msg.Topic]
	b.mu.RUnlock()

	for _, handler := range handlers {
		go handler(msg)
	}
}

// Start 启动广播器
func (b *Broadcaster) Start() error {
	logger.Info("广播器已启动",
		zap.String("service", b.service),
		zap.String("node_id", b.nodeID),
	)
	return nil
}

// Stop 停止广播器
func (b *Broadcaster) Stop() error {
	b.cancel()
	return nil
}

// PublishJSON 广播 JSON 消息
func (b *Broadcaster) PublishJSON(ctx context.Context, topic string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return b.Publish(ctx, topic, payload)
}

// SendJSON 发送 JSON 消息到指定服务
func (b *Broadcaster) SendJSON(ctx context.Context, topic string, data any, target string) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return b.SendTo(ctx, topic, payload, target)
}
