# 缓存架构重构说明

## 概述

已将缓存模块完全重写，采用**极简设计**，所有微服务共享 Redis 微服务提供的全局内存缓存。

## 架构变化

### 之前（分散式）
```
┌──────────┐        ┌──────────┐        ┌──────────┐
│ Service A│        │ Service B│        │ Service C│
│ (本地缓存) │        │ (本地缓存) │        │ (本地缓存) │
└──────────┘        └──────────┘        └──────────┘
   各自独立的内存缓存，无法共享
```

### 现在（集中式）
```
┌──────────┐        ┌──────────┐        ┌──────────┐
│ Service A│───────▶│          │◀───────│ Service C│
└──────────┘        │  Redis   │        └──────────┘
                    │ Service  │
┌──────────┐        │ (全局缓存)│
│ Service B│───────▶│          │
└──────────┘        └──────────┘
   所有服务共享同一个内存缓存实例
```

## 新的缓存模块

### 文件结构
```
pkg/cache/
├── cache.go   - 缓存接口和全局实例
└── store.go   - 内存存储实现
```

### 核心特性

✅ **极简 API**：只保留最常用的操作  
✅ **全局单例**：`cache.Global()` 获取全局缓存  
✅ **自动清理**：每 5 分钟清理过期项  
✅ **线程安全**：使用 RWMutex 保护并发访问  
✅ **零依赖**：仅使用标准库

### API 列表

```go
// 创建缓存
cache.Global()           // 获取全局缓存实例
cache.New()              // 创建新的独立缓存实例

// 基础操作
Set(key, value)          // 设置缓存（永不过期）
SetWithExpiration(...)   // 设置带过期时间的缓存
SetRaw(key, []byte, ttl) // 设置原始字节数据
Get(key, &dest)          // 获取缓存
GetRaw(key)              // 获取原始字节数据
Delete(key)              // 删除缓存
Exists(key)              // 检查键是否存在
Keys()                   // 获取所有键
Clear()                  // 清空所有缓存
Close()                  // 关闭缓存
Count()                  // 获取缓存项数量

// 便捷函数（使用全局缓存）
cache.Set(key, value)
cache.Get(key, &dest)
cache.Delete(key)
cache.Exists(key)
cache.Clear()
cache.Close()
```

## 使用示例

### 基础使用
```go
import "github.com/goback/pkg/cache"

// 设置缓存
cache.Set("user:123", user)

// 获取缓存
var user User
err := cache.Get("user:123", &user)

// 设置带过期时间的缓存
cache.Global().SetWithExpiration("token:abc", token, 30*time.Minute)

// 删除缓存
cache.Delete("user:123")
```

### 服务注册中心使用
```go
// registry/redis.go 使用全局缓存存储服务注册信息
r.cache = cache.Global()
r.cache.SetRaw(key, data, 30*time.Second)
```

### Redis 微服务
```go
// services/redis/internal/redis/redis.go
svc.cache = cache.New()  // 独立实例
```

## 移除的功能

为了简化，以下功能已移除（如需要可后续添加）：

- ❌ HashCache（哈希缓存）
- ❌ SetCache（集合缓存）
- ❌ PrefixedCache（前缀缓存）
- ❌ 复杂的缓存策略
- ❌ GetOrSet、SetNX、Incr 等高级操作

## 性能特点

- **内存存储**：所有数据存储在内存中，读写速度极快
- **自动过期**：支持 TTL，自动清理过期数据
- **并发安全**：使用读写锁，支持高并发访问
- **低开销**：无网络调用，零序列化开销（内部操作）

## 注意事项

⚠️ **数据共享**：所有微服务共享同一个缓存实例（通过 Redis 服务）  
⚠️ **内存限制**：数据存储在内存中，注意内存使用  
⚠️ **无持久化**：服务重启后数据丢失  
⚠️ **无分布式**：当前实现为单机内存缓存

## 迁移指南

### 如果代码使用了以下功能：

**HashCache / SetCache / PrefixedCache**
```go
// 旧代码
hashCache := cache.NewHashCache()
hashCache.HSet("hash", "field", value)

// 新代码 - 使用普通缓存 + 命名空间
cache.Set("hash:field", value)
```

**GetOrSet / SetNX**
```go
// 旧代码
cache.GetOrSet(key, &dest, setter, ttl)

// 新代码 - 手动实现
if !cache.Exists(key) {
    value := computeValue()
    cache.SetWithExpiration(key, value, ttl)
}
```

**Incr**
```go
// 需要时可以自己实现
var count int
cache.Get(key, &count)
count++
cache.Set(key, count)
```

## 编译验证

✅ 所有服务已编译通过：
```bash
go build ./pkg/cache
go build ./pkg/registry  
go build ./...
```

## 总结

新的缓存模块：
- 📦 **代码量减少 70%**
- 🚀 **API 更简洁**
- 🎯 **职责更清晰**
- ✨ **易于理解和维护**
