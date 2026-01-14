package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// redisServiceURL Redis 服务地址
var (
	redisServiceURL = "http://localhost:28090"
	httpClient      = &http.Client{Timeout: 3 * time.Second}
	mu              sync.RWMutex
	initialized     bool
)

// Init 从配置初始化 Redis 服务地址
func Init(host string, port int) {
	mu.Lock()
	defer mu.Unlock()
	if host != "" && port > 0 {
		redisServiceURL = fmt.Sprintf("http://%s:%d", host, port)
	}
	initialized = true
}

// SetRedisServiceURL 设置 Redis 服务地址
func SetRedisServiceURL(url string) {
	mu.Lock()
	redisServiceURL = url
	initialized = true
	mu.Unlock()
}

// getURL 获取当前 Redis 服务地址
func getURL() string {
	mu.RLock()
	defer mu.RUnlock()
	return redisServiceURL
}

// Cache 缓存客户端
type Cache struct {
	baseURL string
}

// Global 获取全局缓存客户端
func Global() *Cache {
	return &Cache{baseURL: getURL()}
}

// New 创建新的缓存客户端
func New() *Cache {
	return &Cache{baseURL: getURL()}
}

// NewWithURL 创建指定 URL 的缓存客户端
func NewWithURL(url string) *Cache {
	return &Cache{baseURL: url}
}

// setRequest 设置请求体
type setRequest struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
	TTL   int64  `json:"ttl"`
}

// getRequest 获取请求体
type getRequest struct {
	Key string `json:"key"`
}

// getResponse 获取响应体
type getResponse struct {
	Value []byte `json:"value"`
	Found bool   `json:"found"`
}

// keysResponse 键列表响应
type keysResponse struct {
	Keys []string `json:"keys"`
}

// Set 设置缓存（永不过期）
func (c *Cache) Set(key string, value any) error {
	return c.SetWithExpiration(key, value, 0)
}

// SetWithExpiration 设置带过期时间的缓存
func (c *Cache) SetWithExpiration(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	return c.SetRaw(key, data, expiration)
}

// SetRaw 设置原始字节数据
func (c *Cache) SetRaw(key string, value []byte, expiration time.Duration) error {
	req := setRequest{
		Key:   key,
		Value: value,
		TTL:   int64(expiration.Seconds()),
	}
	body, _ := json.Marshal(req)

	resp, err := httpClient.Post(c.baseURL+"/cache/set", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("redis service unavailable (%s): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("redis service error (%s): status %d", c.baseURL, resp.StatusCode)
	}
	return nil
}

// Get 获取缓存
func (c *Cache) Get(key string, dest any) error {
	data, ok := c.GetRaw(key)
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}
	return json.Unmarshal(data, dest)
}

// GetRaw 获取原始字节数据
func (c *Cache) GetRaw(key string) ([]byte, bool) {
	req := getRequest{Key: key}
	body, _ := json.Marshal(req)

	resp, err := httpClient.Post(c.baseURL+"/cache/get", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, false
	}

	respBody, _ := io.ReadAll(resp.Body)
	var result getResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, false
	}

	return result.Value, result.Found
}

// Delete 删除缓存
func (c *Cache) Delete(key string) {
	req := getRequest{Key: key}
	body, _ := json.Marshal(req)

	resp, err := httpClient.Post(c.baseURL+"/cache/delete", "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	resp.Body.Close()
}

// Exists 检查键是否存在
func (c *Cache) Exists(key string) bool {
	req := getRequest{Key: key}
	body, _ := json.Marshal(req)

	resp, err := httpClient.Post(c.baseURL+"/cache/exists", "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]bool
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false
	}

	return result["exists"]
}

// Keys 获取所有键
func (c *Cache) Keys() []string {
	resp, err := httpClient.Get(c.baseURL + "/cache/keys")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result keysResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil
	}

	return result.Keys
}

// Clear 清空所有缓存
func (c *Cache) Clear() {
	resp, err := httpClient.Post(c.baseURL+"/cache/clear", "application/json", nil)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// Close 关闭（兼容接口，无实际操作）
func (c *Cache) Close() {}

// Count 获取缓存项数量
func (c *Cache) Count() int {
	return len(c.Keys())
}

// 全局便捷函数
func Set(key string, value any) error {
	return Global().Set(key, value)
}

func Get(key string, dest any) error {
	return Global().Get(key, dest)
}

func Delete(key string) {
	Global().Delete(key)
}

func Exists(key string) bool {
	return Global().Exists(key)
}

func Clear() {
	Global().Clear()
}

func Close() {
	// no-op
}
