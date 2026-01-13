package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/database"
	"github.com/goback/pkg/logger"
	"github.com/goback/pkg/middleware"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/goback/pkg/router"
	"github.com/goback/services/rbac/internal/model"
	"github.com/goback/services/rbac/internal/permission"
	"github.com/goback/services/rbac/internal/permissionscope"
	"github.com/goback/services/rbac/internal/role"
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

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	db := database.Get()

	// 自动迁移
	if err := db.AutoMigrate(
		&model.Role{},
		&model.Permission{},
		&model.RolePermission{},
		&model.PermissionScope{},
	); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	// 初始化角色树缓存
	if err := model.RoleTreeCache.Refresh(); err != nil {
		logger.Warn("初始化角色树缓存失败", zap.Error(err))
	}

	// 创建控制器
	permCtrl := &permission.Controller{}
	permScopeCtrl := &permissionscope.Controller{}
	roleCtrl := &role.Controller{
		PermCtrl:      permCtrl,
		CasbinService: auth.NewCasbinService(),
	}

	// JWT中间件
	jwtManager := auth.NewJWTManager(&cfg.JWT)
	jwtMiddleware := middleware.JWTAuth(jwtManager)

	// 创建mDNS注册中心
	reg := registry.NewMDNSRegistry()

	// 创建Fiber应用
	app := fiber.New()

	// 全局中间件
	app.Use(middleware.Recovery())
	app.Use(middleware.Cors())
	app.Use(middleware.RequestID())

	middlewares := map[string]fiber.Handler{
		"jwt": jwtMiddleware,
	}
	// 注册路由
	router.Register(app, middlewares,
		roleCtrl,
		permCtrl,
		permScopeCtrl,
	)

	// 健康检查
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"status":  "healthy",
			"service": "rbac-service",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 服务地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, 8082)

	// 注册服务
	svc := pkgRegistry.NewServiceBuilder("rbac-service", "v1.0.0").
		WithAddress(addr).
		WithBasePath("rbac").
		Build()

	if err := reg.Register(svc); err != nil {
		logger.Fatal("注册服务失败", zap.Error(err))
	}

	// 启动服务
	go func() {
		logger.Info("RBAC服务启动",
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
	_ = ctx

	// 注销服务
	if err := reg.Deregister(svc); err != nil {
		logger.Error("注销服务失败", zap.Error(err))
	}

	if err := app.Shutdown(); err != nil {
		logger.Error("服务关闭异常", zap.Error(err))
	}

	logger.Info("服务已关闭")
}
