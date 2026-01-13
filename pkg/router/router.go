package router

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Route 路由配置
type Route struct {
	Method      string           // HTTP方法
	Path        string           // 路径(相对路径或以/开头的绝对路径)
	Handler     fiber.Handler    // 处理函数
	Middlewares *[]fiber.Handler // 路由级中间件
}

// RouteFunc 路由配置函数类型
type RouteFunc func(middlewares map[string]fiber.Handler) []Route

// Registrar 路由注册器接口
type Registrar interface {
	// Prefix 返回路由前缀
	Prefix() string
	// Routes 返回路由配置列表,接收中间件作为参数
	Routes(middlewares map[string]fiber.Handler) []Route
}

// Register 自动注册路由
func Register(app fiber.Router, middlewares map[string]fiber.Handler, controllers ...Registrar) {
	for _, ctrl := range controllers {
		prefix := ctrl.Prefix()

		// 创建路由组
		g := app.Group(prefix)

		for _, route := range ctrl.Routes(middlewares) {
			handlers := buildHandlers(route)
			if strings.HasPrefix(route.Path, "/") && !strings.HasPrefix(route.Path, prefix) {
				// 绝对路径,直接注册到app
				app.Add(route.Method, route.Path, handlers...)
			} else {
				// 相对路径,注册到组
				g.Add(route.Method, route.Path, handlers...)
			}
		}
	}
}

// buildHandlers 构建处理器链(中间件 + 处理函数)
func buildHandlers(route Route) []fiber.Handler {
	if route.Middlewares == nil || len(*route.Middlewares) == 0 {
		return []fiber.Handler{route.Handler}
	}
	handlers := make([]fiber.Handler, 0, len(*route.Middlewares)+1)
	handlers = append(handlers, *route.Middlewares...)
	handlers = append(handlers, route.Handler)
	return handlers
}
