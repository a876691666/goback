package lifecycle

import (
	"github.com/gofiber/fiber/v2"
	"go-micro.dev/v5/registry"
)

// Builder 服务构建器 - 链式调用创建服务
type Builder struct {
	opts    *ServiceOptions
	app     *fiber.App
	onStart []Hook
	onReady []Hook
	onStop  []Hook
}

// NewBuilder 创建服务构建器
func NewBuilder(name string) *Builder {
	return &Builder{
		opts: &ServiceOptions{
			Name:   name,
			NodeID: name + "-1",
		},
		onStart: make([]Hook, 0),
		onReady: make([]Hook, 0),
		onStop:  make([]Hook, 0),
	}
}

// WithNodeID 设置节点ID
func (b *Builder) WithNodeID(nodeID string) *Builder {
	b.opts.NodeID = nodeID
	return b
}

// WithAddress 设置服务地址
func (b *Builder) WithAddress(addr string) *Builder {
	b.opts.Address = addr
	return b
}

// WithRegistry 设置服务注册中心
func (b *Builder) WithRegistry(reg registry.Registry) *Builder {
	b.opts.Registry = reg
	return b
}

// WithService 设置服务注册信息
func (b *Builder) WithService(svc *registry.Service) *Builder {
	b.opts.Service = svc
	return b
}

// WithApp 设置Fiber应用
func (b *Builder) WithApp(app *fiber.App) *Builder {
	b.app = app
	return b
}

// OnStart 添加启动钩子
func (b *Builder) OnStart(fn Hook) *Builder {
	b.onStart = append(b.onStart, fn)
	return b
}

// OnReady 添加就绪钩子
func (b *Builder) OnReady(fn Hook) *Builder {
	b.onReady = append(b.onReady, fn)
	return b
}

// OnStop 添加停止钩子
func (b *Builder) OnStop(fn Hook) *Builder {
	b.onStop = append(b.onStop, fn)
	return b
}

// Build 构建服务
func (b *Builder) Build() *Service {
	// 如果没有设置注册中心，创建默认的mDNS注册中心
	if b.opts.Registry == nil {
		b.opts.Registry = registry.NewMDNSRegistry()
	}

	// 如果没有设置服务注册信息，自动创建
	if b.opts.Service == nil && b.opts.Name != "" && b.opts.Address != "" {
		b.opts.Service = &registry.Service{
			Name:    b.opts.Name,
			Version: "1.0.0",
			Nodes: []*registry.Node{
				{
					Id:      b.opts.NodeID,
					Address: b.opts.Address,
				},
			},
		}
	}

	svc := NewService(b.opts)

	if b.app != nil {
		svc.SetApp(b.app)
	}

	for _, fn := range b.onStart {
		svc.OnStart(fn)
	}
	for _, fn := range b.onReady {
		svc.OnReady(fn)
	}
	for _, fn := range b.onStop {
		svc.OnStop(fn)
	}

	return svc
}

// Run 构建并运行服务
func (b *Builder) Run() error {
	return b.Build().Run()
}

// QuickService 快速创建一个简单的服务
// 适用于不需要复杂配置的场景
func QuickService(name, address string, app *fiber.App) *Service {
	return NewBuilder(name).
		WithAddress(address).
		WithApp(app).
		Build()
}
