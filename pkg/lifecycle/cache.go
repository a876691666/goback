package lifecycle

import (
	"encoding/json"
	"fmt"
	"sync"
)

// CacheSpace 缓存空间 - 每个模块独立的缓存存储
type CacheSpace struct {
	module string
	data   map[string]string // key -> raw JSON string
	mu     sync.RWMutex
}

// NewCacheSpace 创建缓存空间
func NewCacheSpace(module string) *CacheSpace {
	return &CacheSpace{
		module: module,
		data:   make(map[string]string),
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
