package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goback/pkg/config"
	"github.com/goback/pkg/logger"
	"github.com/goback/pkg/middleware"
	"github.com/goback/services/gateway/internal/gateway"
	"github.com/gofiber/fiber/v2"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
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

	// 创建mDNS注册中心 (go-micro v5) - 无需外部服务
	reg := registry.NewMDNSRegistry()

	// 创建网关
	gw := gateway.NewGateway(reg, cfg)

	// 同步已有服务的路由
	if err := gw.SyncRoutes(); err != nil {
		logger.Warn("同步服务路由失败", zap.Error(err))
	}

	// 监听服务变化，自动注册/注销路由
	if err := gw.WatchServices(); err != nil {
		logger.Fatal("启动服务监听失败", zap.Error(err))
	}

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
	app.Get("/health", gw.HealthCheck)

	// 服务状态
	app.Get("/services", gw.GetServicesStatus)

	// API 代理
	app.All("/api/*", gw.GetHandler())

	// 启动服务
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)
	go func() {
		logger.Info("网关服务启动",
			zap.String("addr", addr),
			zap.String("env", cfg.App.Env),
		)
		if err := app.Listen(addr); err != nil {
			logger.Fatal("启动服务失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(); err != nil {
		logger.Error("服务关闭异常", zap.Error(err))
	}

	if err := gw.Shutdown(ctx); err != nil {
		logger.Error("网关关闭异常", zap.Error(err))
	}

	logger.Info("服务已关闭")
}
