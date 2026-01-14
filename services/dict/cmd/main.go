package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/database"
	"github.com/goback/pkg/lifecycle"
	"github.com/goback/pkg/logger"
	"github.com/goback/pkg/middleware"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/goback/pkg/router"
	"github.com/goback/services/dict/internal/dictdata"
	"github.com/goback/services/dict/internal/dicttype"
	"github.com/goback/services/dict/internal/model"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
)

const (
	serviceName = "dict-service"
	servicePort = 8084
	basePath    = "dicts"
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

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}

	// 服务地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, servicePort)

	// 创建mDNS注册中心
	reg := registry.NewMDNSRegistry()

	// 构建服务注册信息
	svcInfo := pkgRegistry.NewServiceBuilder(serviceName, "v1.0.0").
		WithAddress(addr).
		WithBasePath(basePath).
		Build()

	// 创建Fiber应用
	app := fiber.New()

	// 全局中间件
	app.Use(middleware.Recovery())
	app.Use(middleware.Cors())
	app.Use(middleware.RequestID())

	// 健康检查
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"status":  "healthy",
			"service": serviceName,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 声明服务变量（用于闭包）
	var svc *lifecycle.Service

	// 创建服务
	svc = lifecycle.NewBuilder(serviceName).
		WithNodeID(serviceName + "-1").
		WithAddress(addr).
		WithRegistry(reg).
		WithService(svcInfo).
		WithApp(app).
		OnStart(func(ctx *lifecycle.ServiceContext) error {
			// 数据库迁移
			db := database.Get()
			if err := db.AutoMigrate(&model.DictType{}, &model.DictData{}); err != nil {
				return fmt.Errorf("数据库迁移失败: %w", err)
			}
			logger.Info("数据库迁移完成")

			// JWT中间件
			jwtManager := auth.NewJWTManager(&cfg.JWT)
			jwtMiddleware := middleware.JWTAuth(jwtManager)

			// 创建控制器（传入ServiceContext）
			baseCtrl := router.NewBaseController(ctx)
			dictTypeCtrl := &dicttype.Controller{BaseController: baseCtrl}
			dictDataCtrl := &dictdata.Controller{BaseController: baseCtrl}

			middlewares := map[string]fiber.Handler{
				"jwt": jwtMiddleware,
			}
			// 注册路由
			router.Register(app, middlewares, dictTypeCtrl, dictDataCtrl)

			return nil
		}).
		OnReady(func(ctx *lifecycle.ServiceContext) error {
			// 订阅RBAC权限更新
			cache := ctx.Cache()
			cache.Subscribe("rbac-service", lifecycle.ModuleRBAC, func(msg *lifecycle.CacheMessage) {
				logger.Info("收到RBAC缓存更新",
					zap.String("key", msg.Key),
					zap.String("action", msg.Action),
				)
			})

			logger.Info("字典服务就绪", zap.String("addr", addr))
			return nil
		}).
		OnStop(func(ctx *lifecycle.ServiceContext) error {
			logger.Info("字典服务正在清理资源...")
			return nil
		}).
		Build()

	// 监听其他服务的生命周期
	svc.Lifecycle().OnEvent(lifecycle.EventReady, func(msg *lifecycle.LifecycleMessage) {
		if msg.Service == serviceName {
			return
		}
		logger.Info("检测到服务就绪", zap.String("service", msg.Service))
	})

	// 运行服务
	if err := svc.Run(); err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
