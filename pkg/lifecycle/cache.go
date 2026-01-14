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

// CacheMessage 缓存广播消息
type CacheMessage struct {
	Service   string    `json:"service"`   // 来源服务
	Module    string    `json:"module"`    // 功能模块
	Key       string    `json:"key"`       // 缓存键
	Data      string    `json:"data"`      // JSON序列化后的数据
	Action    string    `json:"action"`    // 操作类型: set, delete, clear
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// CacheHandler 缓存更新处理器
type CacheHandler func(msg *CacheMessage)

// CacheSpace 缓存空间 - 每个模块独立的缓存存储
type CacheSpace struct {
	service  string
	module   string
	data     map[string]string // key -> raw JSON string
	mu       sync.RWMutex
	handlers map[string][]CacheHandler // key -> handlers
	globalHandlers []CacheHandler      // 监听该模块所有key变更
}

// NewCacheSpace 创建缓存空间
func NewCacheSpace(service, module string) *CacheSpace {
	return &CacheSpace{
		service:        service,
		module:         module,
		data:           make(map[string]string),
		handlers:       make(map[string][]CacheHandler),
		globalHandlers: make([]CacheHandler, 0),
	}
}

// Set 设置缓存（仅本地，不广播）
func (cs *CacheSpace) Set(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}
	
	cs.mu.Lock()
	cs.data[key] = string(data)
	cs.mu.Unlock()
	
	return nil
}

// Get 获取缓存并反序列化
func (cs *CacheSpace) Get(key string, dest any) error {
	cs.mu.RLock()
	raw, ok := cs.data[key]
	cs.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("cache key not found: %s", key)
	}
	
	return json.Unmarshal([]byte(raw), dest)
}

// GetRaw 获取原始JSON字符串
func (cs *CacheSpace) GetRaw(key string) (string, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	raw, ok := cs.data[key]
	return raw, ok
}

// Delete 删除缓存
func (cs *CacheSpace) Delete(key string) {
	cs.mu.Lock()
	delete(cs.data, key)
	cs.mu.Unlock()
}

// Clear 清空所有缓存
func (cs *CacheSpace) Clear() {
	cs.mu.Lock()
	cs.data = make(map[string]string)
	cs.mu.Unlock()
}

// Keys 获取所有缓存键
func (cs *CacheSpace) Keys() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	keys := make([]string, 0, len(cs.data))
	for k := range cs.data {
		keys = append(keys, k)
	}
	return keys
}

// OnKeyChange 监听特定key变更
func (cs *CacheSpace) OnKeyChange(key string, handler CacheHandler) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.handlers[key] = append(cs.handlers[key], handler)
}

// OnAnyChange 监听任意key变更
func (cs *CacheSpace) OnAnyChange(handler CacheHandler) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.globalHandlers = append(cs.globalHandlers, handler)
}

// notifyHandlers 通知处理器
func (cs *CacheSpace) notifyHandlers(msg *CacheMessage) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	// 调用key特定处理器
	if handlers, ok := cs.handlers[msg.Key]; ok {
		for _, h := range handlers {
			go h(msg)
		}
	}
	
	// 调用全局处理器
	for _, h := range cs.globalHandlers {
		go h(msg)
	}
}

// updateFromMessage 从消息更新本地缓存
func (cs *CacheSpace) updateFromMessage(msg *CacheMessage) {
	switch msg.Action {
	case "set":
		cs.mu.Lock()
		cs.data[msg.Key] = msg.Data
		cs.mu.Unlock()
	case "delete":
		cs.Delete(msg.Key)
	case "clear":
		cs.Clear()
	}
	
	// 通知处理器进行业务更新
	cs.notifyHandlers(msg)
}

// CacheBroadcaster 缓存广播器
type CacheBroadcaster struct {
	service string
	redis   *redis.Client
	spaces  map[string]*CacheSpace // module -> CacheSpace
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	pubsub  *redis.PubSub
}

const cacheChannel = "service:cache"

// NewCacheBroadcaster 创建缓存广播器
func NewCacheBroadcaster(service string) *CacheBroadcaster {
	ctx, cancel := context.WithCancel(context.Background())
	return &CacheBroadcaster{
		service: service,
		redis:   database.GetRedis(),
		spaces:  make(map[string]*CacheSpace),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// GetSpace 获取或创建缓存空间
func (cb *CacheBroadcaster) GetSpace(module string) *CacheSpace {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if space, ok := cb.spaces[module]; ok {
		return space
	}
	
	space := NewCacheSpace(cb.service, module)
	cb.spaces[module] = space
	return space
}

// Broadcast 广播缓存更新
func (cb *CacheBroadcaster) Broadcast(module, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}
	
	msg := &CacheMessage{
		Service:   cb.service,
		Module:    module,
		Key:       key,
		Data:      string(data),
		Action:    "set",
		Timestamp: time.Now(),
	}
	
	// 更新本地缓存
	space := cb.GetSpace(module)
	space.mu.Lock()
	space.data[key] = string(data)
	space.mu.Unlock()
	
	// 广播到其他节点
	return cb.publish(msg)
}

// BroadcastDelete 广播删除缓存
func (cb *CacheBroadcaster) BroadcastDelete(module, key string) error {
	msg := &CacheMessage{
		Service:   cb.service,
		Module:    module,
		Key:       key,
		Action:    "delete",
		Timestamp: time.Now(),
	}
	
	// 删除本地缓存
	space := cb.GetSpace(module)
	space.Delete(key)
	
	return cb.publish(msg)
}

// BroadcastClear 广播清空模块缓存
func (cb *CacheBroadcaster) BroadcastClear(module string) error {
	msg := &CacheMessage{
		Service:   cb.service,
		Module:    module,
		Action:    "clear",
		Timestamp: time.Now(),
	}
	
	// 清空本地缓存
	space := cb.GetSpace(module)
	space.Clear()
	
	return cb.publish(msg)
}

// publish 发布消息到Redis
func (cb *CacheBroadcaster) publish(msg *CacheMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return cb.redis.Publish(cb.ctx, cacheChannel, data).Err()
}

// Subscribe 订阅指定服务和模块的缓存更新
// 支持通配符: service="*" 订阅所有服务, module="*" 订阅所有模块
func (cb *CacheBroadcaster) Subscribe(service, module string, handler CacheHandler) {
	space := cb.GetSpace(module)
	
	// 包装handler，添加过滤逻辑
	wrappedHandler := func(msg *CacheMessage) {
		// 服务过滤
		if service != "*" && msg.Service != service {
			return
		}
		// 模块过滤
		if module != "*" && msg.Module != module {
			return
		}
		handler(msg)
	}
	
	space.OnAnyChange(wrappedHandler)
}

// Start 启动缓存广播监听
func (cb *CacheBroadcaster) Start() error {
	cb.pubsub = cb.redis.Subscribe(cb.ctx, cacheChannel)
	
	_, err := cb.pubsub.Receive(cb.ctx)
	if err != nil {
		return fmt.Errorf("subscribe cache channel: %w", err)
	}
	
	go cb.listen()
	
	logger.Info("缓存广播器已启动", zap.String("service", cb.service))
	return nil
}

// listen 监听缓存消息
func (cb *CacheBroadcaster) listen() {
	ch := cb.pubsub.Channel()
	
	for {
		select {
		case <-cb.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			cb.handleMessage(msg.Payload)
		}
	}
}

// handleMessage 处理缓存消息
func (cb *CacheBroadcaster) handleMessage(payload string) {
	var msg CacheMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		logger.Error("解析缓存消息失败", zap.Error(err))
		return
	}
	
	// 获取对应的缓存空间并更新
	space := cb.GetSpace(msg.Module)
	space.updateFromMessage(&msg)
	
	logger.Debug("收到缓存更新",
		zap.String("from_service", msg.Service),
		zap.String("module", msg.Module),
		zap.String("key", msg.Key),
		zap.String("action", msg.Action),
	)
}

// Stop 停止缓存广播
func (cb *CacheBroadcaster) Stop() error {
	cb.cancel()
	if cb.pubsub != nil {
		return cb.pubsub.Close()
	}
	return nil
}
