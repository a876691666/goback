package redis

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/goback/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

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
func (s *Service) RegisterRoutes(app *fiber.App) {
	app.Post("/cache/set", s.handleSet)
	app.Post("/cache/get", s.handleGet)
	app.Post("/cache/delete", s.handleDelete)
	app.Post("/cache/exists", s.handleExists)
	app.Get("/cache/keys", s.handleKeys)
	app.Post("/cache/clear", s.handleClear)
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

func (s *Service) handleSet(c *fiber.Ctx) error {
	var req SetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	var exp int64
	if req.TTL > 0 {
		exp = time.Now().Add(time.Duration(req.TTL) * time.Second).UnixNano()
	}

	s.mu.Lock()
	s.items[req.Key] = &item{Value: req.Value, Expiration: exp}
	s.mu.Unlock()

	return c.JSON(fiber.Map{"ok": true})
}

func (s *Service) handleGet(c *fiber.Ctx) error {
	var req GetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	s.mu.RLock()
	it, ok := s.items[req.Key]
	s.mu.RUnlock()

	if !ok || it.expired() {
		return c.JSON(GetResponse{Found: false})
	}

	return c.JSON(GetResponse{Value: it.Value, Found: true})
}

func (s *Service) handleDelete(c *fiber.Ctx) error {
	var req GetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	s.mu.Lock()
	delete(s.items, req.Key)
	s.mu.Unlock()

	return c.JSON(fiber.Map{"ok": true})
}

func (s *Service) handleExists(c *fiber.Ctx) error {
	var req GetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	s.mu.RLock()
	it, ok := s.items[req.Key]
	s.mu.RUnlock()

	exists := ok && !it.expired()
	return c.JSON(fiber.Map{"exists": exists})
}

func (s *Service) handleKeys(c *fiber.Ctx) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.items))
	now := time.Now().UnixNano()
	for k, v := range s.items {
		if v.Expiration == 0 || now <= v.Expiration {
			keys = append(keys, k)
		}
	}

	return c.JSON(fiber.Map{"keys": keys})
}

func (s *Service) handleClear(c *fiber.Ctx) error {
	s.mu.Lock()
	s.items = make(map[string]*item)
	s.mu.Unlock()

	return c.JSON(fiber.Map{"ok": true})
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
		return fiber.ErrNotFound
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
