package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/goback/pkg/config"
	"github.com/goback/pkg/database"
	"github.com/goback/pkg/lifecycle"
	"github.com/goback/pkg/logger"
	"github.com/goback/pkg/middleware"
	"github.com/goback/services/gateway/internal/gateway"
	"github.com/gofiber/fiber/v2"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
)

const (
	serviceName = "gateway-service"
)

func main() {
	// 加载配置
	if err := config.Init(""); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}
	cfg := config.Get()

	// 初始化日志
	logger.Init(&cfg.Log)
	defer logger.Sync()

	// 初始化Redis（生命周期系统依赖）
	if err := database.InitRedis(&cfg.Redis); err != nil {
		logger.Fatal("初始化Redis失败", zap.Error(err))
	}
	defer database.CloseRedis()

	// 服务地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)

	// 创建mDNS注册中心
	reg := registry.NewMDNSRegistry()

	// 创建网关
	gw := gateway.NewGateway(reg, cfg)

	// 创建Fiber应用
	app := fiber.New()

	// 中间件
	app.Use(middleware.Recovery())
	app.Use(middleware.Cors())
	app.Use(middleware.RequestID())

	// 限流
	rateLimiter := middleware.NewRateLimiter(1000, 100)
	app.Use(rateLimiter.Middleware())

	// 健康检查
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"status":  "healthy",
			"service": serviceName,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 服务状态
	app.Get("/services", gw.GetServicesStatus)

	// API 代理
	app.All("/api/*", gw.GetHandler())

	// 声明服务变量（用于闭包）
	var svc *lifecycle.Service

	// 创建服务
	svc = lifecycle.NewBuilder(serviceName).
		WithNodeID(serviceName + "-1").
		WithAddress(addr).
		WithApp(app).
		OnStart(func(ctx *lifecycle.ServiceContext) error {
			// 同步已有服务的路由
			if err := gw.SyncRoutes(); err != nil {
				logger.Warn("同步服务路由失败", zap.Error(err))
			}

			// 监听服务变化，自动注册/注销路由
			if err := gw.WatchServices(); err != nil {
				return fmt.Errorf("启动服务监听失败: %w", err)
			}

			return nil
		}).
		OnReady(func(ctx *lifecycle.ServiceContext) error {
			// 订阅RBAC权限更新（网关可用于鉴权）
			cache := ctx.Cache()
			cache.Subscribe("rbac-service", lifecycle.ModuleRBAC, func(msg *lifecycle.CacheMessage) {
				logger.Info("收到RBAC缓存更新",
					zap.String("key", msg.Key),
					zap.String("action", msg.Action),
				)
				// 网关可以在这里更新权限缓存用于请求鉴权
			})

			logger.Info("网关服务就绪", zap.String("addr", addr))
			return nil
		}).
		OnStop(func(ctx *lifecycle.ServiceContext) error {
			ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := gw.Shutdown(ctx2); err != nil {
				logger.Error("网关关闭异常", zap.Error(err))
			}

			logger.Info("网关服务正在清理资源...")
			return nil
		}).
		Build()

	// 监听其他服务的生命周期
	svc.Lifecycle().OnEvent(lifecycle.EventReady, func(msg *lifecycle.LifecycleMessage) {
		if msg.Service == serviceName {
			return
		}
		logger.Info("检测到服务就绪，更新路由表", zap.String("service", msg.Service))
		// 服务就绪后重新同步路由
		if err := gw.SyncRoutes(); err != nil {
			logger.Warn("同步路由失败", zap.Error(err))
		}
	})

	svc.Lifecycle().OnEvent(lifecycle.EventStopping, func(msg *lifecycle.LifecycleMessage) {
		if msg.Service == serviceName {
			return
		}
		logger.Info("检测到服务停止，更新路由表", zap.String("service", msg.Service))
	})

	// 运行服务
	if err := svc.Run(); err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
