package registry

import (
	"encoding/json"
	"strings"

	"go-micro.dev/v5/registry"
)

// RouteConfig 路由配置（存储在服务元数据中）
type RouteConfig struct {
	PathPrefix   string   `json:"path_prefix"`   // 网关路径前缀，如 /api/v1/logs
	StripPrefix  bool     `json:"strip_prefix"`  // 是否去除前缀
	Methods      []string `json:"methods"`       // 允许的HTTP方法
	AuthRequired bool     `json:"auth_required"` // 是否需要认证
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name      string        // 服务名称
	Version   string        // 服务版本
	NodeID    string        // 节点ID
	Address   string        // 服务地址
	BasePath  string        // 服务基础路径（如 logs），网关会代理 /api/v1/{BasePath}/* 到服务的 /*
	Routes    []RouteConfig // 路由配置（可选，用于细粒度控制）
}

// BuildService 构建服务注册信息
func BuildService(cfg *ServiceConfig) *registry.Service {
	// 将路由配置序列化为JSON存储在Metadata中
	routesJSON, _ := json.Marshal(cfg.Routes)

	return &registry.Service{
		Name:    cfg.Name,
		Version: cfg.Version,
		Nodes: []*registry.Node{
			{
				Id:      cfg.NodeID,
				Address: cfg.Address,
				Metadata: map[string]string{
					"routes":    string(routesJSON),
					"base_path": cfg.BasePath,
				},
			},
		},
	}
}

// ParseServiceMeta 从服务元数据中解析服务配置
func ParseServiceMeta(svc *registry.Service) (basePath string, routes []RouteConfig) {
	for _, node := range svc.Nodes {
		if bp, ok := node.Metadata["base_path"]; ok {
			basePath = bp
		}
		if routesJSON, ok := node.Metadata["routes"]; ok {
			var nodeRoutes []RouteConfig
			if err := json.Unmarshal([]byte(routesJSON), &nodeRoutes); err == nil {
				routes = append(routes, nodeRoutes...)
			}
		}
	}
	return
}

// ParseRoutes 从服务元数据中解析路由配置（兼容旧方法）
func ParseRoutes(svc *registry.Service) []RouteConfig {
	_, routes := ParseServiceMeta(svc)
	return routes
}

// DefaultMethods 默认HTTP方法
var DefaultMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

// NewRouteConfig 创建路由配置的快捷方法
func NewRouteConfig(pathPrefix string, authRequired bool, methods ...string) RouteConfig {
	if len(methods) == 0 {
		methods = DefaultMethods
	}
	return RouteConfig{
		PathPrefix:   pathPrefix,
		StripPrefix:  true, // 默认去除前缀
		Methods:      methods,
		AuthRequired: authRequired,
	}
}

// NewPublicRoute 创建公开路由（不需要认证）
func NewPublicRoute(pathPrefix string, methods ...string) RouteConfig {
	return NewRouteConfig(pathPrefix, false, methods...)
}

// NewProtectedRoute 创建受保护路由（需要认证）
func NewProtectedRoute(pathPrefix string, methods ...string) RouteConfig {
	return NewRouteConfig(pathPrefix, true, methods...)
}

// ServiceBuilder 服务构建器
type ServiceBuilder struct {
	config *ServiceConfig
}

// NewServiceBuilder 创建服务构建器
func NewServiceBuilder(name, version string) *ServiceBuilder {
	return &ServiceBuilder{
		config: &ServiceConfig{
			Name:    name,
			Version: version,
			Routes:  make([]RouteConfig, 0),
		},
	}
}

// WithNodeID 设置节点ID
func (b *ServiceBuilder) WithNodeID(nodeID string) *ServiceBuilder {
	b.config.NodeID = nodeID
	return b
}

// WithAddress 设置服务地址
func (b *ServiceBuilder) WithAddress(addr string) *ServiceBuilder {
	b.config.Address = addr
	return b
}

// WithBasePath 设置服务基础路径
// 例如: WithBasePath("logs") 则网关会将 /api/v1/logs/* 代理到服务的 /*
func (b *ServiceBuilder) WithBasePath(basePath string) *ServiceBuilder {
	b.config.BasePath = basePath
	return b
}

// AddRoute 添加路由
func (b *ServiceBuilder) AddRoute(route RouteConfig) *ServiceBuilder {
	b.config.Routes = append(b.config.Routes, route)
	return b
}

// AddPublicRoute 添加公开路由
func (b *ServiceBuilder) AddPublicRoute(pathPrefix string, methods ...string) *ServiceBuilder {
	return b.AddRoute(NewPublicRoute(pathPrefix, methods...))
}

// AddProtectedRoute 添加受保护路由
func (b *ServiceBuilder) AddProtectedRoute(pathPrefix string, methods ...string) *ServiceBuilder {
	return b.AddRoute(NewProtectedRoute(pathPrefix, methods...))
}

// Build 构建服务
func (b *ServiceBuilder) Build() *registry.Service {
	// 如果没有设置NodeID，使用服务名+"-1"
	if b.config.NodeID == "" {
		b.config.NodeID = b.config.Name + "-1"
	}
	return BuildService(b.config)
}

// MatchPath 检查路径是否匹配
func (r *RouteConfig) MatchPath(path string) bool {
	return strings.HasPrefix(path, r.PathPrefix)
}

// MatchMethod 检查方法是否允许
func (r *RouteConfig) MatchMethod(method string) bool {
	for _, m := range r.Methods {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}
