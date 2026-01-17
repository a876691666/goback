package main

import (
	"fmt"
	"os"

	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/logger"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/goback/services/redis/internal/redis"
	"go.uber.org/zap"
)

const (
	serviceName = "redis-service"
	servicePort = 28090
	basePath    = "cache"
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

	// 服务地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, servicePort)

	// 创建 Redis 服务
	svc := redis.NewService(serviceName)

	// 创建 PubSub 服务
	pubsubSvc := redis.NewPubSubService()

	// 创建 BaseApp（Redis 服务使用内存注册中心，因为它是基础设施服务）
	app := core.NewBaseApp(core.BaseAppConfig{
		ServiceName:    serviceName,
		ServiceVersion: "v1.0.0",
		ServiceAddress: addr,
		BasePath:       basePath,
		Registry:       pkgRegistry.NewMemoryRegistry(), // 使用内存注册中心
		DisablePubSub:  true,                            // Redis 服务是 PubSub 中心，不需要 PubSub 客户端
	})

	// 启动时初始化缓存服务
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := svc.Start(); err != nil {
			return fmt.Errorf("缓存服务启动失败: %w", err)
		}
		return e.Next()
	})

	// 路由注册
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// 注册缓存路由
		svc.RegisterRoutes(e.Router)
		// 注册 PubSub 路由
		pubsubSvc.RegisterRoutes(e.Router)

		return e.Next()
	})

	// 服务就绪事件
	app.OnServiceReady().BindFunc(func(e *core.LifecycleEvent) error {
		logger.Info("Redis 缓存服务已启动",
			zap.String("mode", "memory"),
			zap.String("addr", addr),
		)
		return e.Next()
	})

	// 服务停止事件
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		logger.Info("正在关闭服务...")
		pubsubSvc.Stop()
		if err := svc.Stop(); err != nil {
			logger.Error("停止服务失败", zap.Error(err))
		}
		logger.Info("Redis 缓存服务已停止")
		return e.Next()
	})

	// 启动服务
	if err := app.Serve(core.ServeConfig{HttpAddr: addr, ShowStartBanner: true}); err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
