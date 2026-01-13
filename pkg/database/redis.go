package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/goback/pkg/config"
	"github.com/redis/go-redis/v9"
)

var (
	redisOnce   sync.Once
	redisClient *redis.Client
	miniRedis   *miniredis.Miniredis // 内存模式的 Redis
)

// InitRedis 初始化Redis连接
func InitRedis(cfg *config.RedisConfig) error {
	var err error
	redisOnce.Do(func() {
		if cfg.Mode == "memory" {
			// 使用内存模式（miniredis）
			miniRedis, err = miniredis.Run()
			if err != nil {
				return
			}
			redisClient = redis.NewClient(&redis.Options{
				Addr: miniRedis.Addr(),
			})
		} else {
			// 使用外部 Redis 服务
			redisClient = redis.NewClient(&redis.Options{
				Addr:     cfg.Addr(),
				Password: cfg.Password,
				DB:       cfg.DB,
				PoolSize: cfg.PoolSize,
			})

			// 测试连接
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err = redisClient.Ping(ctx).Result()
		}
	})
	return err
}

// GetRedis 获取Redis客户端
func GetRedis() *redis.Client {
	if redisClient == nil {
		panic("redis not initialized, call InitRedis first")
	}
	return redisClient
}

// CloseRedis 关闭Redis连接
func CloseRedis() error {
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			return err
		}
	}
	if miniRedis != nil {
		miniRedis.Close()
	}
	return nil
}

// Cache Redis缓存操作封装
type Cache struct {
	client *redis.Client
	prefix string
}

// NewCache 创建缓存实例
func NewCache(prefix string) *Cache {
	return &Cache{
		client: GetRedis(),
		prefix: prefix,
	}
}

// key 生成带前缀的key
func (c *Cache) key(key string) string {
	if c.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

// Set 设置缓存
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, c.key(key), value, expiration).Err()
}

// Get 获取缓存
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.key(key)).Result()
}

// Del 删除缓存
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	fullKeys := make([]string, len(keys))
	for i, k := range keys {
		fullKeys[i] = c.key(k)
	}
	return c.client.Del(ctx, fullKeys...).Err()
}

// Exists 检查key是否存在
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.key(key)).Result()
	return n > 0, err
}

// SetNX 设置缓存(不存在时)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, c.key(key), value, expiration).Result()
}

// Incr 自增
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, c.key(key)).Result()
}

// Expire 设置过期时间
func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, c.key(key), expiration).Err()
}

// HSet 哈希设置
func (c *Cache) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, c.key(key), values...).Err()
}

// HGet 哈希获取
func (c *Cache) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, c.key(key), field).Result()
}

// HGetAll 获取所有哈希字段
func (c *Cache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, c.key(key)).Result()
}

// HDel 删除哈希字段
func (c *Cache) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, c.key(key), fields...).Err()
}

// SAdd 集合添加
func (c *Cache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, c.key(key), members...).Err()
}

// SMembers 获取集合成员
func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, c.key(key)).Result()
}

// SIsMember 检查是否是集合成员
func (c *Cache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.client.SIsMember(ctx, c.key(key), member).Result()
}
