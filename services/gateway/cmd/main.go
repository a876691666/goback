package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/cache"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/logger"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/goback/services/gateway/internal/gateway"
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

	// 创建网关
	gw := gateway.NewGateway(reg, cfg)

	// 创建应用
	app := core.NewBaseApp(core.BaseAppConfig{
		ServiceName:    serviceName,
		ServiceVersion: "v1.0.0",
	})

	// 设置注册中心和服务信息
	app.SetRegistry(reg).
		SetService(pkgRegistry.NewServiceBuilder(serviceName, "v1.0.0").
			WithAddress(addr).
			Build())

	// 启动时同步路由并监听服务
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := gw.SyncRoutes(); err != nil {
			logger.Warn("同步服务路由失败", zap.Error(err))
		}
		if err := gw.WatchServices(); err != nil {
			return fmt.Errorf("启动服务监听失败: %w", err)
		}
		return e.Next()
	})

	// 路由注册
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// 服务状态
		e.Router.GET("/services", gw.GetServicesStatus)

		// API 代理 - 使用通配符路由
		e.Router.GET("/api/{path...}", gw.GetHandler())
		e.Router.POST("/api/{path...}", gw.GetHandler())
		e.Router.PUT("/api/{path...}", gw.GetHandler())
		e.Router.DELETE("/api/{path...}", gw.GetHandler())
		e.Router.PATCH("/api/{path...}", gw.GetHandler())

		return e.Next()
	})

	// 服务就绪事件
	app.OnServiceReady().BindFunc(func(e *core.LifecycleEvent) error {
		logger.Info("网关服务就绪", zap.String("addr", addr))
		return e.Next()
	})

	// 服务停止事件
	app.OnServiceStopped().BindFunc(func(e *core.LifecycleEvent) error {
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := gw.Shutdown(ctx2); err != nil {
			logger.Error("网关关闭异常", zap.Error(err))
		}
		logger.Info("网关服务正在清理资源...")
		return e.Next()
	})

	// 启动服务
	if err := app.Serve(core.ServeConfig{HttpAddr: addr, ShowStartBanner: true}); err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
