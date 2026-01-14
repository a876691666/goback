package cache

import (
	"fmt"
	"sync"
)

// HashCache 哈希缓存（类似Redis Hash）
type HashCache struct {
	data map[string]map[string][]byte
	mu   sync.RWMutex
}

// NewHashCache 创建哈希缓存
func NewHashCache() *HashCache {
	return &HashCache{
		data: make(map[string]map[string][]byte),
	}
}

// HSet 设置哈希字段
func (h *HashCache) HSet(key, field string, value []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.data[key] == nil {
		h.data[key] = make(map[string][]byte)
	}
	h.data[key][field] = value
}

// HGet 获取哈希字段
func (h *HashCache) HGet(key, field string) ([]byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.data[key] == nil {
		return nil, false
	}
	val, ok := h.data[key][field]
	return val, ok
}

// HGetAll 获取所有哈希字段
func (h *HashCache) HGetAll(key string) map[string][]byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.data[key] == nil {
		return nil
	}
	
	result := make(map[string][]byte)
	for k, v := range h.data[key] {
		result[k] = v
	}
	return result
}

// HDel 删除哈希字段
func (h *HashCache) HDel(key string, fields ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.data[key] == nil {
		return
	}
	
	for _, field := range fields {
		delete(h.data[key], field)
	}
}

// HExists 检查哈希字段是否存在
func (h *HashCache) HExists(key, field string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.data[key] == nil {
		return false
	}
	_, ok := h.data[key][field]
	return ok
}

// HKeys 获取所有字段名
func (h *HashCache) HKeys(key string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.data[key] == nil {
		return nil
	}
	
	keys := make([]string, 0, len(h.data[key]))
	for k := range h.data[key] {
		keys = append(keys, k)
	}
	return keys
}

// HLen 获取字段数量
func (h *HashCache) HLen(key string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.data[key] == nil {
		return 0
	}
	return len(h.data[key])
}

// Del 删除整个哈希
func (h *HashCache) Del(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.data, key)
}

// SetCache 集合缓存（类似Redis Set）
type SetCache struct {
	data map[string]map[string]struct{}
	mu   sync.RWMutex
}

// NewSetCache 创建集合缓存
func NewSetCache() *SetCache {
	return &SetCache{
		data: make(map[string]map[string]struct{}),
	}
}

// SAdd 添加成员
func (s *SetCache) SAdd(key string, members ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.data[key] == nil {
		s.data[key] = make(map[string]struct{})
	}
	
	added := 0
	for _, member := range members {
		if _, ok := s.data[key][member]; !ok {
			s.data[key][member] = struct{}{}
			added++
		}
	}
	return added
}

// SRem 删除成员
func (s *SetCache) SRem(key string, members ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.data[key] == nil {
		return 0
	}
	
	removed := 0
	for _, member := range members {
		if _, ok := s.data[key][member]; ok {
			delete(s.data[key], member)
			removed++
		}
	}
	return removed
}

// SMembers 获取所有成员
func (s *SetCache) SMembers(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.data[key] == nil {
		return nil
	}
	
	members := make([]string, 0, len(s.data[key]))
	for member := range s.data[key] {
		members = append(members, member)
	}
	return members
}

// SIsMember 检查是否是成员
func (s *SetCache) SIsMember(key, member string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.data[key] == nil {
		return false
	}
	_, ok := s.data[key][member]
	return ok
}

// SCard 获取成员数量
func (s *SetCache) SCard(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.data[key] == nil {
		return 0
	}
	return len(s.data[key])
}

// Del 删除整个集合
func (s *SetCache) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// PrefixedCache 带前缀的缓存包装器
type PrefixedCache struct {
	cache  *Cache
	prefix string
}

// NewPrefixedCache 创建带前缀的缓存
func NewPrefixedCache(cache *Cache, prefix string) *PrefixedCache {
	return &PrefixedCache{
		cache:  cache,
		prefix: prefix,
	}
}

func (p *PrefixedCache) key(key string) string {
	if p.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", p.prefix, key)
}

// Set 设置缓存
func (p *PrefixedCache) Set(key string, value interface{}) error {
	return p.cache.Set(p.key(key), value)
}

// Get 获取缓存
func (p *PrefixedCache) Get(key string, dest interface{}) error {
	return p.cache.Get(p.key(key), dest)
}

// Delete 删除缓存
func (p *PrefixedCache) Delete(key string) {
	p.cache.Delete(p.key(key))
}

// Exists 检查是否存在
func (p *PrefixedCache) Exists(key string) bool {
	return p.cache.Exists(p.key(key))
}

// Clear 清除所有带前缀的缓存
func (p *PrefixedCache) Clear() int {
	return p.cache.DeletePrefix(p.prefix + ":")
}
