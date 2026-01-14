package router

import (
	"github.com/goback/pkg/lifecycle"
	"github.com/gofiber/fiber/v2"
)

// BaseController 控制器基类
// 所有控制器都应该嵌入此基类以获得生命周期管理能力
// 使用时需要在创建控制器时传入 Service
type BaseController struct {
	svc *lifecycle.Service
}

// NewBaseController 创建基础控制器
func NewBaseController(svc *lifecycle.Service) BaseController {
	return BaseController{svc: svc}
}

// Service 获取服务实例
func (c *BaseController) Service() *lifecycle.Service {
	return c.svc
}

// Controller 控制器接口
type Controller interface {
	Prefix() string
	Routes(middlewares map[string]fiber.Handler) []Route
}
