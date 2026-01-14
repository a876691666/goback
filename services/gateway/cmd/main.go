package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/goback/pkg/cache"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/lifecycle"
	"github.com/goback/pkg/logger"
	"github.com/goback/pkg/middleware"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/goback/services/gateway/internal/gateway"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const serviceName = "gateway-service"

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

	// 初始化 Redis 缓存客户端
	cache.Init(cfg.Redis.Host, cfg.Redis.Port)
	logger.Info("缓存客户端已初始化",
		zap.String("host", cfg.Redis.Host),
		zap.Int("port", cfg.Redis.Port),
	)

	// 服务地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)

	// 创建 Redis 注册中心
	reg := pkgRegistry.NewRedisRegistry()

	// 构建服务注册信息
	svcInfo := pkgRegistry.NewServiceBuilder(serviceName, "v1.0.0").
		WithAddress(addr).
		Build()

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

	// 创建并运行服务
	err := lifecycle.New(serviceName).
		Node(serviceName+"-1").
		Addr(addr).
		Registry(reg).
		RegInfo(svcInfo).
		App(app).
		OnStart(func(s *lifecycle.Service) error {
			if err := gw.SyncRoutes(); err != nil {
				logger.Warn("同步服务路由失败", zap.Error(err))
			}
			if err := gw.WatchServices(); err != nil {
				return fmt.Errorf("启动服务监听失败: %w", err)
			}
			return nil
		}).
		UseRBAC().
		OnReady(func(s *lifecycle.Service) error {
			logger.Info("网关服务就绪", zap.String("addr", addr))
			return nil
		}).
		OnStop(func(s *lifecycle.Service) error {
			ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := gw.Shutdown(ctx2); err != nil {
				logger.Error("网关关闭异常", zap.Error(err))
			}
			logger.Info("网关服务正在清理资源...")
			return nil
		}).
		On(lifecycle.EventReady, func(msg *lifecycle.EventMessage, s *lifecycle.Service) {
			if msg.Service == serviceName {
				return
			}
			logger.Info("检测到服务就绪，更新路由表", zap.String("service", msg.Service))
			if err := gw.SyncRoutes(); err != nil {
				logger.Warn("同步路由失败", zap.Error(err))
			}
		}).
		On(lifecycle.EventStopping, func(msg *lifecycle.EventMessage, s *lifecycle.Service) {
			if msg.Service == serviceName {
				return
			}
			logger.Info("检测到服务停止，更新路由表", zap.String("service", msg.Service))
		}).
		Run()

	if err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
