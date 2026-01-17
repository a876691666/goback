package registry

import (
	"sync"

	"go-micro.dev/v5/registry"
)

// MemoryRegistry 内存注册中心（用于 Redis 服务等基础设施服务）
type MemoryRegistry struct {
	services map[string]*registry.Service
	mu       sync.RWMutex
}

// NewMemoryRegistry 创建内存注册中心
func NewMemoryRegistry() registry.Registry {
	return &MemoryRegistry{
		services: make(map[string]*registry.Service),
	}
}

// Init 初始化
func (r *MemoryRegistry) Init(opts ...registry.Option) error {
	return nil
}

// Options 获取选项
func (r *MemoryRegistry) Options() registry.Options {
	return registry.Options{}
}

// Register 注册服务
func (r *MemoryRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if s == nil {
		return nil
	}
	r.mu.Lock()
	r.services[s.Name] = s
	r.mu.Unlock()
	return nil
}

// Deregister 注销服务
func (r *MemoryRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if s == nil {
		return nil
	}
	r.mu.Lock()
	delete(r.services, s.Name)
	r.mu.Unlock()
	return nil
}

// GetService 获取服务
func (r *MemoryRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if s, ok := r.services[name]; ok {
		return []*registry.Service{s}, nil
	}
	return nil, registry.ErrNotFound
}

// ListServices 列出所有服务
func (r *MemoryRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*registry.Service, 0, len(r.services))
	for _, s := range r.services {
		services = append(services, s)
	}
	return services, nil
}

// Watch 监听服务变化
func (r *MemoryRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return &memoryWatcher{
		exit: make(chan bool),
	}, nil
}

// String 返回注册中心名称
func (r *MemoryRegistry) String() string {
	return "memory"
}

// memoryWatcher 内存监听器
type memoryWatcher struct {
	exit chan bool
}

func (w *memoryWatcher) Next() (*registry.Result, error) {
	<-w.exit
	return nil, registry.ErrWatcherStopped
}

func (w *memoryWatcher) Stop() {
	select {
	case <-w.exit:
		return
	default:
		close(w.exit)
	}
}
