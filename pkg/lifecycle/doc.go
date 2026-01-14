// Package lifecycle 提供去中心化的微服务生命周期管理和缓存广播功能
//
// # 核心功能
//
// 1. 生命周期管理 (Manager)
//   - 监听其他服务的生命周期事件
//   - 发布自身服务的生命周期状态
//   - 支持的事件: starting, started, ready, stopping, stopped, healthy, unhealthy
//
// 2. 缓存广播 (CacheBroadcaster)
//   - 每个服务/模块独立的缓存空间
//   - 订阅和触发缓存更新
//   - JSON序列化传输，调用时反序列化
//   - 缓存更新自动通知业务层
//
// 3. 服务包装器 (Service)
//   - 简化服务启动流程
//   - 统一的生命周期钩子
//   - 自动的优雅关闭
//
// # 使用示例
//
// ## 基本用法
//
//	// 创建服务
//	svc := lifecycle.NewService(&lifecycle.ServiceOptions{
//		Name:    "user-service",
//		NodeID:  "user-1",
//		Address: ":8081",
//	})
//
//	// 设置Fiber应用
//	app := fiber.New()
//	svc.SetApp(app)
//
//	// 注册生命周期钩子
//	svc.OnStart(func() error {
//		// 初始化数据库等
//		return nil
//	})
//
//	svc.OnReady(func() error {
//		// 服务就绪后的操作，如加载缓存
//		return nil
//	})
//
//	// 运行服务
//	svc.Run()
//
// ## 监听其他服务的生命周期
//
//	lifecycle := svc.Lifecycle()
//
//	// 监听特定事件
//	lifecycle.OnEvent(lifecycle.EventReady, func(msg *lifecycle.LifecycleMessage) {
//		log.Printf("服务 %s 已就绪", msg.Service)
//	})
//
//	// 监听所有事件
//	lifecycle.OnAnyEvent(func(msg *lifecycle.LifecycleMessage) {
//		log.Printf("服务 %s 事件: %s", msg.Service, msg.Event)
//	})
//
// ## 缓存广播使用
//
//	cache := svc.Cache()
//
//	// 获取缓存空间
//	rbacSpace := cache.GetSpace(lifecycle.ModuleRBAC)
//
//	// 广播权限数据（会通知所有订阅者）
//	permissions := []Permission{{ID: 1, Code: "user:read"}}
//	cache.Broadcast(lifecycle.ModuleRBAC, lifecycle.KeyPermissions, permissions)
//
//	// 订阅RBAC模块的缓存更新
//	cache.Subscribe("rbac-service", lifecycle.ModuleRBAC, func(msg *CacheMessage) {
//		if msg.Key == lifecycle.KeyPermissions {
//			var perms []Permission
//			rbacSpace.Get(lifecycle.KeyPermissions, &perms)
//			// 更新本地权限缓存
//		}
//	})
//
//	// 从缓存读取数据
//	var perms []Permission
//	rbacSpace.Get(lifecycle.KeyPermissions, &perms)
//
// ## RBAC权限缓存示例
//
//	// RBAC服务端 - 广播权限更新
//	func (c *RBACController) UpdatePermission(ctx *fiber.Ctx) error {
//		// ... 更新数据库
//
//		// 广播权限更新
//		perms, _ := c.loadAllPermissions()
//		svc.Cache().Broadcast(lifecycle.ModuleRBAC, lifecycle.KeyPermissions, perms)
//
//		return ctx.JSON(response.Success(nil))
//	}
//
//	// 其他服务 - 订阅权限更新
//	svc.OnReady(func() error {
//		cache := svc.Cache()
//
//		// 订阅RBAC权限更新
//		cache.Subscribe("rbac-service", lifecycle.ModuleRBAC, func(msg *CacheMessage) {
//			switch msg.Key {
//			case lifecycle.KeyPermissions:
//				var perms []Permission
//				cache.GetSpace(lifecycle.ModuleRBAC).Get(msg.Key, &perms)
//				// 重新加载权限到Casbin
//				casbinService.ReloadPermissions(perms)
//			case lifecycle.KeyRolePermissions:
//				// 处理角色权限更新
//			}
//		})
//
//		return nil
//	})
package lifecycle
