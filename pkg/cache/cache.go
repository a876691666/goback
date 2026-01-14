package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Item 缓存项
type Item struct {
	Value      []byte
	Expiration int64 // Unix时间戳，0表示永不过期
}

// Expired 检查是否过期
func (item *Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// Cache 内存缓存
type Cache struct {
	items map[string]*Item
	mu    sync.RWMutex
	
	// 清理相关
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// New 创建新的缓存实例
func New() *Cache {
	return NewWithCleanup(5 * time.Minute)
}

// NewWithCleanup 创建带定期清理的缓存
func NewWithCleanup(cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items:           make(map[string]*Item),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}
	
	if cleanupInterval > 0 {
		go c.cleanupLoop()
	}
	
	return c
}

// cleanupLoop 定期清理过期项
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// Set 设置缓存（永不过期）
func (c *Cache) Set(key string, value interface{}) error {
	return c.SetWithExpiration(key, value, 0)
}

// SetWithExpiration 设置带过期时间的缓存
func (c *Cache) SetWithExpiration(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	
	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}
	
	c.mu.Lock()
	c.items[key] = &Item{
		Value:      data,
		Expiration: exp,
	}
	c.mu.Unlock()
	
	return nil
}

// SetRaw 设置原始字节数据
func (c *Cache) SetRaw(key string, value []byte, expiration time.Duration) {
	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}
	
	c.mu.Lock()
	c.items[key] = &Item{
		Value:      value,
		Expiration: exp,
	}
	c.mu.Unlock()
}

// Get 获取缓存并反序列化
func (c *Cache) Get(key string, dest interface{}) error {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}
	
	if item.Expired() {
		c.Delete(key)
		return fmt.Errorf("key expired: %s", key)
	}
	
	return json.Unmarshal(item.Value, dest)
}

// GetRaw 获取原始字节数据
func (c *Cache) GetRaw(key string) ([]byte, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	
	if !ok || item.Expired() {
		return nil, false
	}
	
	return item.Value, true
}

// GetString 获取字符串值
func (c *Cache) GetString(key string) (string, bool) {
	data, ok := c.GetRaw(key)
	if !ok {
		return "", false
	}
	
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return string(data), true
	}
	return s, true
}

// Delete 删除缓存
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// DeletePrefix 删除指定前缀的所有缓存
func (c *Cache) DeletePrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	count := 0
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
			count++
		}
	}
	return count
}

// DeleteExpired 删除所有过期项
func (c *Cache) DeleteExpired() {
	now := time.Now().UnixNano()
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for key, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			delete(c.items, key)
		}
	}
}

// Exists 检查key是否存在
func (c *Cache) Exists(key string) bool {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	
	if !ok {
		return false
	}
	
	if item.Expired() {
		c.Delete(key)
		return false
	}
	
	return true
}

// Keys 获取所有key
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	keys := make([]string, 0, len(c.items))
	now := time.Now().UnixNano()
	
	for key, item := range c.items {
		if item.Expiration == 0 || now <= item.Expiration {
			keys = append(keys, key)
		}
	}
	
	return keys
}

// Count 获取缓存数量
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear 清空所有缓存
func (c *Cache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]*Item)
	c.mu.Unlock()
}

// Close 关闭缓存（停止清理协程）
func (c *Cache) Close() {
	if c.cleanupInterval > 0 {
		close(c.stopCleanup)
	}
}

// GetOrSet 获取缓存，如果不存在则设置
func (c *Cache) GetOrSet(key string, dest interface{}, setter func() (interface{}, error), expiration time.Duration) error {
	// 先尝试获取
	if err := c.Get(key, dest); err == nil {
		return nil
	}
	
	// 不存在，调用setter获取值
	value, err := setter()
	if err != nil {
		return err
	}
	
	// 设置缓存
	if err := c.SetWithExpiration(key, value, expiration); err != nil {
		return err
	}
	
	// 重新获取（确保dest被正确赋值）
	return c.Get(key, dest)
}

// SetNX 仅当key不存在时设置
func (c *Cache) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if item, ok := c.items[key]; ok && !item.Expired() {
		return false, nil
	}
	
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	
	var exp int64
	if expiration > 0 {
		exp = time.Now().Add(expiration).UnixNano()
	}
	
	c.items[key] = &Item{
		Value:      data,
		Expiration: exp,
	}
	
	return true, nil
}

// Incr 自增（仅支持int64）
func (c *Cache) Incr(key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var val int64 = 0
	
	if item, ok := c.items[key]; ok && !item.Expired() {
		if err := json.Unmarshal(item.Value, &val); err != nil {
			return 0, fmt.Errorf("value is not int64")
		}
	}
	
	val++
	data, _ := json.Marshal(val)
	
	if item, ok := c.items[key]; ok {
		item.Value = data
	} else {
		c.items[key] = &Item{Value: data}
	}
	
	return val, nil
}

// Expire 设置key的过期时间
func (c *Cache) Expire(key string, expiration time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	item, ok := c.items[key]
	if !ok {
		return false
	}
	
	if expiration > 0 {
		item.Expiration = time.Now().Add(expiration).UnixNano()
	} else {
		item.Expiration = 0
	}
	
	return true
}
