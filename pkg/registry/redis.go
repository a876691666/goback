package registry

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/goback/pkg/cache"
	"github.com/goback/pkg/logger"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
)

const (
	// Redis key 前缀
	servicePrefix = "registry:service:"
	ttlDuration   = 30 * time.Second
)

// RedisRegistry 基于 Redis 的服务注册中心
type RedisRegistry struct {
	cache     *cache.Cache
	mu        sync.RWMutex
	heartbeat map[string]*time.Ticker
}

// NewRedisRegistry 创建基于 Redis 的注册中心
func NewRedisRegistry() registry.Registry {
	return &RedisRegistry{
		cache:     cache.Global(),
		heartbeat: make(map[string]*time.Ticker),
	}
}

// Init 初始化
func (r *RedisRegistry) Init(opts ...registry.Option) error {
	return nil
}

// Options 获取选项
func (r *RedisRegistry) Options() registry.Options {
	return registry.Options{}
}

// Register 注册服务
func (r *RedisRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if s == nil || len(s.Nodes) == 0 {
		return fmt.Errorf("service or nodes cannot be empty")
	}

	key := servicePrefix + s.Name

	// 序列化服务信息
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal service: %w", err)
	}

	// 存储到 Redis（带过期时间）
	if err := r.cache.SetRaw(key, data, ttlDuration); err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	logger.Debug("服务已注册",
		zap.String("key", key),
		zap.String("service", s.Name),
		zap.Int("nodes", len(s.Nodes)),
	)

	// 启动心跳保活
	r.startHeartbeat(s)

	return nil
}

// Deregister 注销服务
func (r *RedisRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if s == nil {
		return fmt.Errorf("service cannot be nil")
	}

	key := servicePrefix + s.Name
	r.cache.Delete(key)

	// 停止心跳
	r.stopHeartbeat(s.Name)

	return nil
}

// GetService 获取服务
func (r *RedisRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	key := servicePrefix + name
	data, ok := r.cache.GetRaw(key)
	if !ok {
		return nil, registry.ErrNotFound
	}

	var svc registry.Service
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("unmarshal service: %w", err)
	}

	return []*registry.Service{&svc}, nil
}

// ListServices 列出所有服务
func (r *RedisRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	keys := r.cache.Keys()
	services := make([]*registry.Service, 0)

	logger.Debug("ListServices - 获取所有key", zap.Int("total_keys", len(keys)), zap.Strings("keys", keys))

	for _, key := range keys {
		// 只处理服务注册的 key
		if !strings.HasPrefix(key, servicePrefix) {
			continue
		}

		data, ok := r.cache.GetRaw(key)
		if !ok {
			logger.Debug("ListServices - key不存在", zap.String("key", key))
			continue
		}

		var svc registry.Service
		if err := json.Unmarshal(data, &svc); err != nil {
			logger.Warn("ListServices - 反序列化失败", zap.String("key", key), zap.Error(err))
			continue
		}

		services = append(services, &svc)
	}

	logger.Debug("ListServices - 返回服务列表", zap.Int("count", len(services)))
	return services, nil
}

// Watch 监听服务变化（简化实现）
func (r *RedisRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return &redisWatcher{
		registry: r,
		exit:     make(chan bool),
	}, nil
}

// String 返回注册中心名称
func (r *RedisRegistry) String() string {
	return "redis"
}

// startHeartbeat 启动心跳保活
func (r *RedisRegistry) startHeartbeat(s *registry.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 如果已存在心跳，先停止
	if ticker, ok := r.heartbeat[s.Name]; ok {
		ticker.Stop()
	}

	// 创建新的心跳定时器
	ticker := time.NewTicker(ttlDuration / 3)
	r.heartbeat[s.Name] = ticker

	go func() {
		for range ticker.C {
			key := servicePrefix + s.Name
			data, err := json.Marshal(s)
			if err != nil {
				continue
			}
			_ = r.cache.SetRaw(key, data, ttlDuration)
		}
	}()
}

// stopHeartbeat 停止心跳
func (r *RedisRegistry) stopHeartbeat(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ticker, ok := r.heartbeat[name]; ok {
		ticker.Stop()
		delete(r.heartbeat, name)
	}
}

// redisWatcher Redis 监听器（简化实现）
type redisWatcher struct {
	registry *RedisRegistry
	exit     chan bool
}

func (w *redisWatcher) Next() (*registry.Result, error) {
	<-w.exit
	return nil, registry.ErrWatcherStopped
}

func (w *redisWatcher) Stop() {
	select {
	case <-w.exit:
		return
	default:
		close(w.exit)
	}
}
