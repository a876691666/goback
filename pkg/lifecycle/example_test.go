package lifecycle_test

import (
	"fmt"
	"time"

	"github.com/goback/pkg/lifecycle"
	"github.com/gofiber/fiber/v2"
	"go-micro.dev/v5/registry"
)

// ExampleService_basic 展示基本的服务启动流程
func ExampleService_basic() {
	// 创建mDNS注册中心
	reg := registry.NewMDNSRegistry()

	// 构建服务注册信息
	svcInfo := &registry.Service{
		Name:    "user-service",
		Version: "v1.0.0",
		Nodes: []*registry.Node{
			{Id: "user-1", Address: ":8081"},
		},
	}

	// 创建服务
	svc := lifecycle.NewService(&lifecycle.ServiceOptions{
		Name:     "user-service",
		NodeID:   "user-1",
		Address:  ":8081",
		Registry: reg,
		Service:  svcInfo,
	})

	// 创建Fiber应用
	app := fiber.New()
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	svc.SetApp(app)

	// 启动钩子
	svc.OnStart(func() error {
		fmt.Println("初始化数据库连接...")
		return nil
	})

	// 就绪钩子
	svc.OnReady(func() error {
		fmt.Println("服务已就绪，开始加载缓存...")
		return nil
	})

	// 停止钩子
	svc.OnStop(func() error {
		fmt.Println("清理资源...")
		return nil
	})

	// 运行服务（阻塞直到收到退出信号）
	// svc.Run()
}

// ExampleManager_lifecycle 展示如何监听其他服务的生命周期
func ExampleManager_lifecycle() {
	// 创建生命周期管理器
	mgr := lifecycle.NewManager("gateway-service", "gateway-1")

	// 监听服务就绪事件
	mgr.OnEvent(lifecycle.EventReady, func(msg *lifecycle.LifecycleMessage) {
		fmt.Printf("服务 %s (节点: %s) 已就绪\n", msg.Service, msg.NodeID)

		// 可以在这里更新路由表、重新加载配置等
		if msg.Service == "rbac-service" {
			fmt.Println("RBAC服务就绪，开始同步权限...")
		}
	})

	// 监听服务停止事件
	mgr.OnEvent(lifecycle.EventStopping, func(msg *lifecycle.LifecycleMessage) {
		fmt.Printf("服务 %s 正在停止\n", msg.Service)
	})

	// 监听所有事件
	mgr.OnAnyEvent(func(msg *lifecycle.LifecycleMessage) {
		fmt.Printf("[%s] 服务: %s, 事件: %s\n",
			msg.Timestamp.Format(time.RFC3339),
			msg.Service,
			msg.Event,
		)
	})

	// 启动监听
	// mgr.Start()
	// defer mgr.Stop()
}

// ExampleCacheBroadcaster_rbac 展示RBAC权限缓存广播
func ExampleCacheBroadcaster_rbac() {
	// === RBAC服务端 ===

	// 创建缓存广播器
	rbacCache := lifecycle.NewCacheBroadcaster("rbac-service")
	// rbacCache.Start()

	// 模拟权限数据
	permissions := []lifecycle.Permission{
		{ID: 1, Code: "user:read", Name: "读取用户", Resource: "/users", Action: "GET"},
		{ID: 2, Code: "user:write", Name: "编辑用户", Resource: "/users", Action: "POST"},
		{ID: 3, Code: "role:read", Name: "读取角色", Resource: "/roles", Action: "GET"},
	}

	// 广播权限数据到所有服务
	rbacCache.Broadcast(lifecycle.ModuleRBAC, lifecycle.KeyPermissions, permissions)

	// 角色权限映射
	rolePerms := lifecycle.RolePermissionMap{
		1: permissions[:2], // 管理员拥有user:read, user:write
		2: permissions[:1], // 普通用户只有user:read
	}
	rbacCache.Broadcast(lifecycle.ModuleRBAC, lifecycle.KeyRolePermissions, rolePerms)

	// === 其他服务订阅端 ===

	// 用户服务订阅RBAC权限更新
	userCache := lifecycle.NewCacheBroadcaster("user-service")
	// userCache.Start()

	// 订阅RBAC模块的所有缓存更新
	userCache.Subscribe("rbac-service", lifecycle.ModuleRBAC, func(msg *lifecycle.CacheMessage) {
		fmt.Printf("收到RBAC缓存更新: key=%s, action=%s\n", msg.Key, msg.Action)

		switch msg.Key {
		case lifecycle.KeyPermissions:
			// 获取缓存空间
			rbacSpace := userCache.GetSpace(lifecycle.ModuleRBAC)

			// 反序列化权限数据
			var perms []lifecycle.Permission
			if err := rbacSpace.Get(lifecycle.KeyPermissions, &perms); err == nil {
				fmt.Printf("加载了 %d 个权限\n", len(perms))
				// 在这里更新本地权限缓存、重载Casbin等
			}

		case lifecycle.KeyRolePermissions:
			rbacSpace := userCache.GetSpace(lifecycle.ModuleRBAC)
			var rolePerms lifecycle.RolePermissionMap
			if err := rbacSpace.Get(lifecycle.KeyRolePermissions, &rolePerms); err == nil {
				fmt.Printf("加载了 %d 个角色的权限映射\n", len(rolePerms))
			}
		}
	})
}

// ExampleCacheSpace_usage 展示缓存空间的基本操作
func ExampleCacheSpace_usage() {
	cache := lifecycle.NewCacheBroadcaster("my-service")

	// 获取RBAC模块的缓存空间
	rbacSpace := cache.GetSpace(lifecycle.ModuleRBAC)

	// 设置本地缓存（不广播）
	rbacSpace.Set("local_config", map[string]any{
		"cache_ttl": 3600,
		"enabled":   true,
	})

	// 读取缓存
	var config map[string]any
	if err := rbacSpace.Get("local_config", &config); err != nil {
		fmt.Println("缓存未找到")
	} else {
		fmt.Printf("配置: %+v\n", config)
	}

	// 获取原始JSON
	raw, ok := rbacSpace.GetRaw("local_config")
	if ok {
		fmt.Printf("原始JSON: %s\n", raw)
	}

	// 监听特定key变更
	rbacSpace.OnKeyChange(lifecycle.KeyPermissions, func(msg *lifecycle.CacheMessage) {
		fmt.Println("权限数据已更新")
	})

	// 监听所有变更
	rbacSpace.OnAnyChange(func(msg *lifecycle.CacheMessage) {
		fmt.Printf("缓存变更: key=%s, action=%s\n", msg.Key, msg.Action)
	})

	// 获取所有缓存键
	keys := rbacSpace.Keys()
	fmt.Printf("缓存键: %v\n", keys)

	// 删除缓存
	rbacSpace.Delete("local_config")

	// 清空所有缓存
	rbacSpace.Clear()
}

// ExampleService_complete 展示完整的服务配置
func ExampleService_complete() {
	reg := registry.NewMDNSRegistry()

	svcInfo := &registry.Service{
		Name:    "rbac-service",
		Version: "v1.0.0",
		Nodes: []*registry.Node{
			{
				Id:      "rbac-1",
				Address: ":8082",
				Metadata: map[string]string{
					"base_path": "rbac",
				},
			},
		},
	}

	svc := lifecycle.NewService(&lifecycle.ServiceOptions{
		Name:     "rbac-service",
		NodeID:   "rbac-1",
		Address:  ":8082",
		Registry: reg,
		Service:  svcInfo,
	})

	app := fiber.New()
	svc.SetApp(app)

	// 获取生命周期管理器
	lc := svc.Lifecycle()

	// 监听其他服务上线
	lc.OnEvent(lifecycle.EventReady, func(msg *lifecycle.LifecycleMessage) {
		if msg.Service == "rbac-service" {
			return // 忽略自己
		}
		fmt.Printf("服务 %s 已就绪，推送权限数据...\n", msg.Service)
		// 可以主动推送权限给新上线的服务
	})

	// 服务启动时加载并广播权限
	svc.OnReady(func() error {
		cache := svc.Cache()

		// 从数据库加载权限
		permissions := loadPermissionsFromDB()

		// 广播给所有服务
		cache.Broadcast(lifecycle.ModuleRBAC, lifecycle.KeyPermissions, permissions)

		return nil
	})

	// svc.Run()
}

// 模拟从数据库加载权限
func loadPermissionsFromDB() []lifecycle.Permission {
	return []lifecycle.Permission{
		{ID: 1, Code: "user:read", Name: "读取用户"},
		{ID: 2, Code: "user:write", Name: "编辑用户"},
	}
}
