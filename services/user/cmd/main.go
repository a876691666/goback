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
	authpkg "github.com/goback/services/user/internal/auth"
	"github.com/goback/services/user/internal/dept"
	"github.com/goback/services/user/internal/model"
	"github.com/goback/services/user/internal/user"
	"go.uber.org/zap"
)

const (
	serviceName = "user-service"
	servicePort = 8081
	basePath    = "users"
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

	// 创建 Redis 注册中心
	reg := pkgRegistry.NewRedisRegistry()

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

	// 创建并运行服务
	err := lifecycle.New(serviceName).
		Node(serviceName + "-1").
		Addr(addr).
		Registry(reg).
		RegInfo(svcInfo).
		App(app).
		OnStart(func(s *lifecycle.Service) error {
			// 数据库迁移
			db := database.Get()
			if err := db.AutoMigrate(&model.User{}, &model.Role{}); err != nil {
				return fmt.Errorf("数据库迁移失败: %w", err)
			}
			logger.Info("数据库迁移完成")

			// JWT中间件
			jwtManager := auth.NewJWTManager(&cfg.JWT)
			jwtMiddleware := middleware.JWTAuth(jwtManager)

			// 创建控制器
			baseCtrl := router.NewBaseController(s)
			userCtrl := &user.Controller{BaseController: baseCtrl, JWTManager: jwtManager}
			authCtrl := &authpkg.Controller{BaseController: baseCtrl, UserCtrl: userCtrl, JWTManager: jwtManager}
			deptCtrl := &dept.Controller{BaseController: baseCtrl}

			middlewares := map[string]fiber.Handler{
				"jwt": jwtMiddleware,
			}
			router.Register(app, middlewares, authCtrl, userCtrl, deptCtrl)
			return nil
		}).
		OnReady(func(s *lifecycle.Service) error {
			logger.Info("用户服务就绪", zap.String("addr", addr))
			return nil
		}).
		OnStop(func(s *lifecycle.Service) error {
			logger.Info("用户服务正在清理资源...")
			return nil
		}).
		On(lifecycle.EventReady, func(msg *lifecycle.EventMessage, s *lifecycle.Service) {
			if msg.Service == serviceName {
				return
			}
			logger.Info("检测到服务就绪", zap.String("service", msg.Service))
		}).
		Run()

	if err != nil {
		logger.Fatal("服务运行失败", zap.Error(err))
	}
}
