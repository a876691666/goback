package main

import (
	"fmt"
	"os"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/database"
	"github.com/goback/pkg/logger"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/goback/services/log/internal/loginlog"
	"github.com/goback/services/log/internal/model"
	"github.com/goback/services/log/internal/operationlog"
	"go.uber.org/zap"
)

const (
	serviceName = "log-service"
	servicePort = 8084
	basePath    = "logs"
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

	// 创建 BaseApp（自动创建 Registry、PubSub、Service）
	app := core.NewBaseApp(core.BaseAppConfig{
		ServiceName:    serviceName,
		ServiceVersion: "v1.0.0",
		ServiceAddress: addr,
		BasePath:       basePath,
		Registry:       pkgRegistry.NewRedisRegistry(),
		RedisAddr:      fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
	})

	// JWT管理器
	jwtValidator := auth.NewJWTManager(&cfg.JWT)

	// 启动时数据库迁移
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		db := database.Get()
		if err := db.AutoMigrate(&model.OperationLog{}, &model.LoginLog{}); err != nil {
			return fmt.Errorf("数据库迁移失败: %w", err)
		}
		logger.Info("数据库迁移完成")
		return e.Next()
	})

	// 路由注册
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// JWT验证中间件
		jwtMiddleware := apis.JWTAuth(apis.JWTConfig{
			Validator: jwtValidator,
		})

		// 操作日志路由
		opLogGroup := e.Router.Group("/operation-logs")
		opLogGroup.Bind(jwtMiddleware)
		opLogGroup.GET("", operationlog.List)
		opLogGroup.DELETE("/{ids}", operationlog.Delete)
		opLogGroup.DELETE("/clear", operationlog.Clear)

		// 登录日志路由
		loginLogGroup := e.Router.Group("/login-logs")
		loginLogGroup.Bind(jwtMiddleware)
		loginLogGroup.GET("", loginlog.List)
		loginLogGroup.DELETE("/{ids}", loginlog.Delete)
		loginLogGroup.DELETE("/clear", loginlog.Clear)

		return e.Next()
	})

	// 服务就绪事件
	app.OnServiceReady().BindFunc(func(e *core.LifecycleEvent) error {
		logger.Info("日志服务就绪", zap.String("addr", addr))
		return e.Next()
	})

	// 服务停止事件
	app.OnServiceStopped().BindFunc(func(e *core.LifecycleEvent) error {
		logger.Info("日志服务正在清理资源...")
		return e.Next()
	})

	// 启动服务
	if err := app.Serve(core.ServeConfig{HttpAddr: addr, ShowStartBanner: true}); err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
