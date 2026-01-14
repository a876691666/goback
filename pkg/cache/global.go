package cache

import (
	"sync"
)

// 全局缓存实例
var (
	globalCache     *Cache
	globalHashCache *HashCache
	globalSetCache  *SetCache
	globalOnce      sync.Once
)

// initGlobal 初始化全局缓存
func initGlobal() {
	globalOnce.Do(func() {
		globalCache = New()
		globalHashCache = NewHashCache()
		globalSetCache = NewSetCache()
	})
}

// Global 获取全局缓存实例
func Global() *Cache {
	initGlobal()
	return globalCache
}

// GlobalHash 获取全局哈希缓存实例
func GlobalHash() *HashCache {
	initGlobal()
	return globalHashCache
}

// GlobalSet 获取全局集合缓存实例
func GlobalSet() *SetCache {
	initGlobal()
	return globalSetCache
}

// Close 关闭全局缓存
func Close() {
	if globalCache != nil {
		globalCache.Close()
	}
}
