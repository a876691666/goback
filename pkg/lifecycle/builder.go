package lifecycle

import (
	"github.com/gofiber/fiber/v2"
	"go-micro.dev/v5/registry"
)

// Builder 服务构建器 - 链式调用创建服务
type Builder struct {
	opts    *ServiceOptions
	app     *fiber.App
	onStart []func(*ServiceContext) error
	onReady []func(*ServiceContext) error
	onStop  []func(*ServiceContext) error
}

// NewBuilder 创建服务构建器
func NewBuilder(name string) *Builder {
	return &Builder{
		opts: &ServiceOptions{
			Name:   name,
			NodeID: name + "-1",
		},
		onStart: make([]func(*ServiceContext) error, 0),
		onReady: make([]func(*ServiceContext) error, 0),
		onStop:  make([]func(*ServiceContext) error, 0),
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
func (b *Builder) OnStart(fn func(*ServiceContext) error) *Builder {
	b.onStart = append(b.onStart, fn)
	return b
}

// OnReady 添加就绪钩子
func (b *Builder) OnReady(fn func(*ServiceContext) error) *Builder {
	b.onReady = append(b.onReady, fn)
	return b
}

// OnStop 添加停止钩子
func (b *Builder) OnStop(fn func(*ServiceContext) error) *Builder {
	b.onStop = append(b.onStop, fn)
	return b
}

// Build 构建服务
func (b *Builder) Build() *Service {
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
