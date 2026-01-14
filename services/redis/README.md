# Redis 缓存微服务

## 概述

这是一个**极简**的内存缓存微服务，作为整个系统的中心缓存服务。**最先启动**，为其他微服务提供缓存支持。

## 特性

- ✅ **内存模式**：所有数据存储在内存中，性能极佳
- ✅ **Pub/Sub**：支持基础的发布/订阅功能
- ✅ **内网访问**：仅供内部微服务调用，无对外接口
- ✅ **极简设计**：无 controller、无生命周期管理、无多余中间件
- ✅ **自动清理**：定期清理过期缓存
- ✅ **最先启动**：在 docker-compose 中配置为最先启动的微服务

## 架构

```
redis-service
├── 内存缓存 (pkg/cache)
└── Pub/Sub (pkg/broadcast)
```

**注意**：此服务不注册到服务发现中心，其他服务通过全局缓存实例直接访问。

## 使用方式

### 1. 全局缓存访问

通过 `pkg/cache/global.go` 提供的全局缓存实例：

```go
import "github.com/goback/pkg/cache"

// 设置缓存
cache.Set("key", value)

// 获取缓存
var result MyType
err := cache.Get("key", &result)
```

### 2. 缓存操作

支持的操作：

- `Set/Get`: 基础读写
- `SetWithExpiration`: 带过期时间
- `Delete`: 删除键
- `Exists`: 检查键是否存在
- `Keys`: 获取所有键
- `Clear`: 清空缓存
- `Incr`: 自增计数器

### 3. Pub/Sub

通过 `pkg/broadcast` 包实现：

```go
// 发布消息
svc.Publish("channel", []byte("message"))

// 订阅频道
svc.Subscribe("channel", func(msg []byte) {
    // 处理消息
})
```

## 部署

### Docker 部署

```bash
# 单独启动
docker-compose up -d redis-service

# 完整启动（redis-service 会最先启动）
docker-compose up -d
```

### 本地运行

```bash
cd services/redis/cmd
go run main.go
```

## 配置

运行模式：内存模式（无持久化）  
网络：仅内网访问  
依赖：仅依赖 etcd（用于服务间通信）

## 启动顺序

在 `docker-compose.yml` 中，所有其他微服务都配置了对 `redis-service` 的依赖，确保：

1. **redis-service** 最先启动
2. 其他服务启动时缓存已就绪
3. 避免缓存未就绪导致的启动失败

## 注意事项

⚠️ **数据持久化**：所有数据仅存储在内存中，服务重启后数据会丢失  
⚠️ **内网访问**：该服务不对外暴露端口，仅供内部微服务使用  
⚠️ **无认证**：默认无认证机制，依赖内网隔离保证安全  
⚠️ **无服务注册**：不注册到服务发现中心，通过全局缓存实例访问
