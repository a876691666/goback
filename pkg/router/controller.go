package router

import (
	"github.com/goback/pkg/lifecycle"
	"github.com/gofiber/fiber/v2"
)

// BaseController 控制器基类
// 所有控制器都应该嵌入此基类以获得生命周期管理能力
// 使用时需要在创建控制器时传入 ServiceContext
type BaseController struct {
	svc *lifecycle.ServiceContext
}

// NewBaseController 创建基础控制器
func NewBaseController(svc *lifecycle.ServiceContext) BaseController {
	return BaseController{svc: svc}
}

// ServiceContext 获取服务上下文
func (c *BaseController) ServiceContext() *lifecycle.ServiceContext {
	return c.svc
}

// Cache 获取缓存广播器
func (c *BaseController) Cache() *lifecycle.CacheBroadcaster {
	if c.svc == nil {
		return nil
	}
	return c.svc.Cache()
}

// Broadcast 广播缓存更新
func (c *BaseController) Broadcast(module, key string, value any) error {
	if c.svc == nil {
		return nil
	}
	return c.svc.Broadcast(module, key, value)
}

// BroadcastDelete 广播删除缓存
func (c *BaseController) BroadcastDelete(module, key string) error {
	if c.svc == nil {
		return nil
	}
	return c.svc.BroadcastDelete(module, key)
}

// GetCacheSpace 获取缓存空间
func (c *BaseController) GetCacheSpace(module string) *lifecycle.CacheSpace {
	if c.svc == nil {
		return nil
	}
	return c.svc.GetCacheSpace(module)
}

// Controller 控制器接口
type Controller interface {
	Prefix() string
	Routes(middlewares map[string]fiber.Handler) []Route
}
