package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/goback/pkg/config"
	"github.com/goback/pkg/logger"
	"github.com/goback/services/redis/internal/redis"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	serviceName = "redis-service"
	servicePort = 28090
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

	logger.Info("启动 Redis 缓存服务", zap.String("service", serviceName))

	// 创建 Redis 服务
	svc := redis.NewService(serviceName)

	// 启动服务
	if err := svc.Start(); err != nil {
		logger.Fatal("服务启动失败", zap.Error(err))
	}

	// 创建 Fiber 应用
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// 健康检查
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": serviceName})
	})

	// 注册缓存路由
	svc.RegisterRoutes(app)

	// 启动 HTTP 服务
	go func() {
		addr := fmt.Sprintf(":%d", servicePort)
		if err := app.Listen(addr); err != nil {
			logger.Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	logger.Info("Redis 缓存服务已启动",
		zap.String("mode", "memory"),
		zap.Int("port", servicePort),
	)

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")

	// 停止服务
	app.Shutdown()
	if err := svc.Stop(); err != nil {
		logger.Error("停止服务失败", zap.Error(err))
	}

	logger.Info("Redis 缓存服务已停止")
}
