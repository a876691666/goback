package redis

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/app/tools/router"
	"github.com/goback/pkg/logger"
	"go.uber.org/zap"
)

// ErrNotFound 未找到错误
var ErrNotFound = errors.New("key not found")

// item 缓存项
type item struct {
	Value      []byte
	Expiration int64
}

func (i *item) expired() bool {
	return i.Expiration > 0 && time.Now().UnixNano() > i.Expiration
}

// Service Redis 缓存服务
type Service struct {
	name        string
	items       map[string]*item
	mu          sync.RWMutex
	stopCleanup chan struct{}
}

// NewService 创建 Redis 服务
func NewService(name string) *Service {
	s := &Service{
		name:        name,
		items:       make(map[string]*item),
		stopCleanup: make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

// cleanupLoop 定期清理过期项
func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.deleteExpired()
		case <-s.stopCleanup:
			return
		}
	}
}

func (s *Service) deleteExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixNano()
	for k, v := range s.items {
		if v.Expiration > 0 && now > v.Expiration {
			delete(s.items, k)
		}
	}
}

// Start 启动服务
func (s *Service) Start() error {
	logger.Info("Redis 缓存服务初始化完成",
		zap.String("service", s.name),
		zap.String("mode", "memory"),
	)
	return nil
}

// Stop 停止服务
func (s *Service) Stop() error {
	logger.Info("正在停止 Redis 缓存服务")
	close(s.stopCleanup)
	logger.Info("Redis 缓存服务已停止")
	return nil
}

// RegisterRoutes 注册 HTTP 路由
func (s *Service) RegisterRoutes(r *router.Router[*core.RequestEvent]) {
	r.POST("/cache/set", s.handleSet)
	r.POST("/cache/get", s.handleGet)
	r.POST("/cache/delete", s.handleDelete)
	r.POST("/cache/exists", s.handleExists)
	r.GET("/cache/keys", s.handleKeys)
	r.POST("/cache/clear", s.handleClear)
}

// SetRequest 设置请求
type SetRequest struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
	TTL   int64  `json:"ttl"` // 秒
}

// GetRequest 获取请求
type GetRequest struct {
	Key string `json:"key"`
}

// GetResponse 获取响应
type GetResponse struct {
	Value []byte `json:"value"`
	Found bool   `json:"found"`
}

func (s *Service) handleSet(e *core.RequestEvent) error {
	var req SetRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	var exp int64
	if req.TTL > 0 {
		exp = time.Now().Add(time.Duration(req.TTL) * time.Second).UnixNano()
	}

	s.mu.Lock()
	s.items[req.Key] = &item{Value: req.Value, Expiration: exp}
	s.mu.Unlock()

	return e.JSON(200, map[string]any{"ok": true})
}

func (s *Service) handleGet(e *core.RequestEvent) error {
	var req GetRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	s.mu.RLock()
	it, ok := s.items[req.Key]
	s.mu.RUnlock()

	if !ok || it.expired() {
		return e.JSON(200, GetResponse{Found: false})
	}

	return e.JSON(200, GetResponse{Value: it.Value, Found: true})
}

func (s *Service) handleDelete(e *core.RequestEvent) error {
	var req GetRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	s.mu.Lock()
	delete(s.items, req.Key)
	s.mu.Unlock()

	return e.JSON(200, map[string]any{"ok": true})
}

func (s *Service) handleExists(e *core.RequestEvent) error {
	var req GetRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	s.mu.RLock()
	it, ok := s.items[req.Key]
	s.mu.RUnlock()

	exists := ok && !it.expired()
	return e.JSON(200, map[string]any{"exists": exists})
}

func (s *Service) handleKeys(e *core.RequestEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.items))
	now := time.Now().UnixNano()
	for k, v := range s.items {
		if v.Expiration == 0 || now <= v.Expiration {
			keys = append(keys, k)
		}
	}

	return e.JSON(200, map[string]any{"keys": keys})
}

func (s *Service) handleClear(e *core.RequestEvent) error {
	s.mu.Lock()
	s.items = make(map[string]*item)
	s.mu.Unlock()

	return e.JSON(200, map[string]any{"ok": true})
}

// --- 直接访问方法（供本地调用） ---

func (s *Service) Set(key string, value any, ttl int64) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(time.Duration(ttl) * time.Second).UnixNano()
	}
	s.mu.Lock()
	s.items[key] = &item{Value: data, Expiration: exp}
	s.mu.Unlock()
	return nil
}

func (s *Service) SetRaw(key string, value []byte, ttl int64) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(time.Duration(ttl) * time.Second).UnixNano()
	}
	s.mu.Lock()
	s.items[key] = &item{Value: value, Expiration: exp}
	s.mu.Unlock()
}

func (s *Service) Get(key string, dest any) error {
	s.mu.RLock()
	it, ok := s.items[key]
	s.mu.RUnlock()
	if !ok || it.expired() {
		return ErrNotFound
	}
	return json.Unmarshal(it.Value, dest)
}

func (s *Service) GetRaw(key string) ([]byte, bool) {
	s.mu.RLock()
	it, ok := s.items[key]
	s.mu.RUnlock()
	if !ok || it.expired() {
		return nil, false
	}
	return it.Value, true
}

func (s *Service) Delete(key string) {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
}

func (s *Service) Exists(key string) bool {
	s.mu.RLock()
	it, ok := s.items[key]
	s.mu.RUnlock()
	return ok && !it.expired()
}

func (s *Service) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.items))
	now := time.Now().UnixNano()
	for k, v := range s.items {
		if v.Expiration == 0 || now <= v.Expiration {
			keys = append(keys, k)
		}
	}
	return keys
}
