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
	"github.com/goback/services/dict/internal/dictdata"
	"github.com/goback/services/dict/internal/dicttype"
	"github.com/goback/services/dict/internal/model"
	"go.uber.org/zap"
)

const (
	serviceName = "dict-service"
	servicePort = 8085
	basePath    = "dicts"
)

func main() {
	// 加载配置（自动设置 SQLite 数据库文件名为 data/<serviceName>.db）
	if err := config.InitWithService("", serviceName); err != nil {
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
		IsDev:          cfg.App.Env == "dev",
		RedisAddr:      fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
	})

	// JWT验证器（直接使用，无需适配器）
	jwtValidator := auth.NewJWTManager(&cfg.JWT)

	// 注册 OnBootstrap 钩子 - 数据库迁移
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		db := database.Get()
		if err := db.AutoMigrate(&model.DictType{}, &model.DictData{}); err != nil {
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
			SkipPaths: []string{"/health", "/dicts/dict-data/dicts/"},
		})

		// 字典类型路由组（需要认证）
		dictTypeGroup := e.Router.Group("/dicts/dict-types")
		dictTypeGroup.Bind(jwtMiddleware)
		dictTypeGroup.POST("", dicttype.Create)
		dictTypeGroup.GET("", dicttype.List)
		dictTypeGroup.GET("/{id}", dicttype.Get)
		dictTypeGroup.PUT("/{id}", dicttype.Update)
		dictTypeGroup.DELETE("/{id}", dicttype.Delete)

		// 字典数据路由组（需要认证）
		dictDataGroup := e.Router.Group("/dicts/dict-data")
		dictDataGroup.Bind(jwtMiddleware)
		dictDataGroup.POST("", dictdata.Create)
		dictDataGroup.GET("/{id}", dictdata.Get)
		dictDataGroup.GET("/type/{typeId}", dictdata.ListByType)
		dictDataGroup.PUT("/{id}", dictdata.Update)
		dictDataGroup.DELETE("/{id}", dictdata.Delete)

		// 公开路由（无需认证）
		e.Router.GET("/dicts/dict-data/dicts/{code}", dictdata.GetByCode)

		return e.Next()
	})

	// 注册生命周期事件处理（使用硬编码钩子）
	app.OnServiceReady().BindFunc(func(e *core.LifecycleEvent) error {
		// 忽略自己的事件
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
