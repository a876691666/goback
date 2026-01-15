package main

import (
	"fmt"
	"os"
	"time"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/database"
	"github.com/goback/pkg/logger"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/goback/services/config/internal/model"
	"github.com/goback/services/config/internal/sysconfig"
	"go.uber.org/zap"
)

const (
	serviceName = "config-service"
	servicePort = 8086
	basePath    = "config"
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

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}

	// 服务地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, servicePort)

	// 创建 BaseApp
	app := core.NewBaseApp(core.BaseAppConfig{
		ServiceName:    serviceName,
		ServiceVersion: "v1.0.0",
		IsDev:          cfg.App.Env == "dev",
	})

	// 设置注册中心和服务
	app.SetRegistry(pkgRegistry.NewRedisRegistry()).
		SetService(pkgRegistry.NewServiceBuilder(serviceName, "v1.0.0").
			WithAddress(addr).
			WithBasePath(basePath).
			Build())

	// JWT验证器
	jwtValidator := auth.NewJWTManager(&cfg.JWT)

	// 注册 OnBootstrap 钩子 - 数据库迁移
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		db := database.Get()
		if err := db.AutoMigrate(&model.SysConfig{}); err != nil {
			return fmt.Errorf("数据库迁移失败: %w", err)
		}
		logger.Info("数据库迁移完成")
		return e.Next()
	})

	// 注册 OnServe 钩子 - 配置路由
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// 全局中间件
		e.Router.BindFunc(apis.Recovery().Func)
		e.Router.BindFunc(apis.RequestID().Func)
		e.Router.BindFunc(apis.Logger(apis.LoggerConfig{
			SkipPaths: []string{"/health"},
		}).Func)

		// 健康检查
		e.Router.GET("/health", func(re *core.RequestEvent) error {
			return re.JSON(200, map[string]any{
				"status":  "healthy",
				"service": serviceName,
				"time":    time.Now().Format(time.RFC3339),
			})
		})

		// JWT中间件
		jwtMiddleware := apis.JWTAuth(apis.JWTConfig{
			Validator: jwtValidator,
			SkipPaths: []string{"/health", "/config/get-by-key"},
		})

		// 系统参数配置路由组
		configGroup := e.Router.Group("/config")
		configGroup.Bind(jwtMiddleware)
		configGroup.GET("/info", sysconfig.Info)
		configGroup.GET("/get-by-key", sysconfig.GetByKey).Unbind(apis.DefaultJWTAuthMiddlewareId)
		configGroup.GET("/page", sysconfig.Page)
		configGroup.POST("/add", sysconfig.Add)
		configGroup.PUT("/update", sysconfig.Update)
		configGroup.DELETE("/remove", sysconfig.Remove)

		return e.Next()
	})

	// 注册生命周期事件处理
	app.OnServiceReady().BindFunc(func(e *core.LifecycleEvent) error {
		if e.Message.Service == serviceName {
			return e.Next()
		}
		logger.Info("检测到服务就绪",
			zap.String("service", e.Message.Service),
			zap.String("nodeId", e.Message.NodeID))
		return e.Next()
	})

	app.OnServiceStopped().BindFunc(func(e *core.LifecycleEvent) error {
		if e.Message.Service == serviceName {
			return e.Next()
		}
		logger.Info("检测到服务停止",
			zap.String("service", e.Message.Service))
		return e.Next()
	})

	// 启动服务
	if err := app.Serve(core.ServeConfig{
		HttpAddr:        addr,
		ShowStartBanner: true,
	}); err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
