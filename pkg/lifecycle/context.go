package lifecycle

// ServiceContext 服务上下文 - 用于在服务内部共享生命周期组件
// 通过构造函数创建，显式传入，不使用全局单例
type ServiceContext struct {
	service *Service
}

// newServiceContext 创建服务上下文（内部使用）
func newServiceContext(svc *Service) *ServiceContext {
	return &ServiceContext{service: svc}
}

// GetService 获取服务实例
func (sc *ServiceContext) GetService() *Service {
	return sc.service
}

// Cache 获取缓存广播器
func (sc *ServiceContext) Cache() *CacheBroadcaster {
	if sc.service == nil {
		return nil
	}
	return sc.service.Cache()
}

// Lifecycle 获取生命周期管理器
func (sc *ServiceContext) Lifecycle() *Manager {
	if sc.service == nil {
		return nil
	}
	return sc.service.Lifecycle()
}

// Broadcast 广播缓存更新
func (sc *ServiceContext) Broadcast(module, key string, value any) error {
	cache := sc.Cache()
	if cache == nil {
		return nil
	}
	return cache.Broadcast(module, key, value)
}

// BroadcastDelete 广播删除缓存
func (sc *ServiceContext) BroadcastDelete(module, key string) error {
	cache := sc.Cache()
	if cache == nil {
		return nil
	}
	return cache.BroadcastDelete(module, key)
}

// GetCacheSpace 获取缓存空间
func (sc *ServiceContext) GetCacheSpace(module string) *CacheSpace {
	cache := sc.Cache()
	if cache == nil {
		return nil
	}
	return cache.GetSpace(module)
}

// EmitEvent 发送生命周期事件
func (sc *ServiceContext) EmitEvent(event Event, metadata any) error {
	lc := sc.Lifecycle()
	if lc == nil {
		return nil
	}
	return lc.Emit(event, metadata)
}
