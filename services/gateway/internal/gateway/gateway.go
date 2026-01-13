package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/goback/pkg/config"
	"github.com/goback/pkg/logger"
	pkgRegistry "github.com/goback/pkg/registry"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go-micro.dev/v5/registry"
	"go.uber.org/zap"
)

const (
	// APIVersion API版本前缀
	APIVersion = "/api/v1"
)

// Gateway API网关
type Gateway struct {
	registry registry.Registry
	config   *config.Config
	routes   map[string]*ServiceRoute // key: 网关路径前缀
	mu       sync.RWMutex             // 保护routes的并发访问
	watcher  registry.Watcher
	stopChan chan struct{}
}

// ServiceRoute 服务路由配置
type ServiceRoute struct {
	ServiceName  string   // 微服务名称
	PathPrefix   string   // 网关路径前缀，如 /api/v1/logs
	TargetPrefix string   // 目标服务路径前缀，如 / 或 /api
	StripPrefix  bool     // 是否去除前缀转发
	Methods      []string // 允许的HTTP方法
	AuthRequired bool     // 是否需要认证
}

// NewGateway 创建网关
func NewGateway(reg registry.Registry, cfg *config.Config) *Gateway {
	return &Gateway{
		registry: reg,
		config:   cfg,
		routes:   make(map[string]*ServiceRoute),
		stopChan: make(chan struct{}),
	}
}

// RegisterRoute 注册服务路由
func (g *Gateway) RegisterRoute(route *ServiceRoute) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.routes[route.PathPrefix] = route
	logger.Info("注册路由",
		zap.String("service", route.ServiceName),
		zap.String("gateway_path", route.PathPrefix),
		zap.String("target_prefix", route.TargetPrefix),
		zap.Bool("strip_prefix", route.StripPrefix),
		zap.Strings("methods", route.Methods),
		zap.Bool("auth", route.AuthRequired),
	)
}

// UnregisterRoute 注销服务路由
func (g *Gateway) UnregisterRoute(pathPrefix string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if route, exists := g.routes[pathPrefix]; exists {
		delete(g.routes, pathPrefix)
		logger.Info("注销路由",
			zap.String("service", route.ServiceName),
			zap.String("path", pathPrefix),
		)
	}
}

// SyncRoutes 从注册中心同步所有服务路由
func (g *Gateway) SyncRoutes() error {
	services, err := g.registry.ListServices()
	if err != nil {
		return err
	}

	for _, svc := range services {
		// 获取服务详情
		svcDetails, err := g.registry.GetService(svc.Name)
		if err != nil {
			logger.Warn("获取服务详情失败",
				zap.String("service", svc.Name),
				zap.Error(err),
			)
			continue
		}

		for _, s := range svcDetails {
			g.registerServiceRoutes(s)
		}
	}

	return nil
}

// registerServiceRoutes 从服务元数据注册路由
func (g *Gateway) registerServiceRoutes(svc *registry.Service) {
	basePath, routes := pkgRegistry.ParseServiceMeta(svc)

	// 如果设置了 basePath，注册基于 basePath 的统一路由
	// 网关 /api/v1/{basePath}/* -> 服务 /*
	if basePath != "" {
		gatewayPrefix := fmt.Sprintf("%s/%s", APIVersion, basePath)
		g.RegisterRoute(&ServiceRoute{
			ServiceName:  svc.Name,
			PathPrefix:   gatewayPrefix,
			TargetPrefix: "/",
			StripPrefix:  true,
			Methods:      pkgRegistry.DefaultMethods,
			AuthRequired: true, // 默认需要认证
		})
	}

	// 如果有细粒度的路由配置，也注册它们（覆盖默认配置）
	for _, route := range routes {
		g.RegisterRoute(&ServiceRoute{
			ServiceName:  svc.Name,
			PathPrefix:   route.PathPrefix,
			TargetPrefix: "/",
			StripPrefix:  route.StripPrefix,
			Methods:      route.Methods,
			AuthRequired: route.AuthRequired,
		})
	}
}

// unregisterServiceRoutes 注销服务的所有路由
func (g *Gateway) unregisterServiceRoutes(serviceName string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for prefix, route := range g.routes {
		if route.ServiceName == serviceName {
			delete(g.routes, prefix)
			logger.Info("注销路由",
				zap.String("service", serviceName),
				zap.String("path", prefix),
			)
		}
	}
}

// WatchServices 监听服务变化，自动更新路由
func (g *Gateway) WatchServices() error {
	watcher, err := g.registry.Watch()
	if err != nil {
		return err
	}
	g.watcher = watcher

	go func() {
		for {
			select {
			case <-g.stopChan:
				return
			default:
				result, err := watcher.Next()
				if err != nil {
					logger.Error("监听服务变化失败", zap.Error(err))
					time.Sleep(time.Second)
					continue
				}

				g.handleServiceEvent(result)
			}
		}
	}()

	logger.Info("开始监听服务变化")
	return nil
}

// handleServiceEvent 处理服务事件
func (g *Gateway) handleServiceEvent(result *registry.Result) {
	if result.Service == nil {
		return
	}

	switch result.Action {
	case "create", "update":
		logger.Info("服务注册/更新",
			zap.String("service", result.Service.Name),
			zap.String("action", result.Action),
		)
		g.registerServiceRoutes(result.Service)
	case "delete":
		logger.Info("服务注销",
			zap.String("service", result.Service.Name),
		)
		g.unregisterServiceRoutes(result.Service.Name)
	}
}

// StopWatch 停止监听
func (g *Gateway) StopWatch() {
	close(g.stopChan)
	if g.watcher != nil {
		g.watcher.Stop()
	}
}

// GetHandler 获取Fiber处理器
func (g *Gateway) GetHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// 查找匹配的路由
		g.mu.RLock()
		var matchedRoute *ServiceRoute
		for prefix, route := range g.routes {
			if strings.HasPrefix(path, prefix) {
				matchedRoute = route
				break
			}
		}
		g.mu.RUnlock()

		if matchedRoute == nil {
			return c.Status(404).JSON(fiber.Map{"code": 404, "message": "服务未找到"})
		}

		// 检查方法是否允许
		methodAllowed := false
		for _, m := range matchedRoute.Methods {
			if m == c.Method() {
				methodAllowed = true
				break
			}
		}
		if !methodAllowed {
			return c.Status(405).JSON(fiber.Map{"code": 405, "message": "方法不允许"})
		}

		// 服务发现
		services, err := g.registry.GetService(matchedRoute.ServiceName)
		if err != nil || len(services) == 0 {
			logger.Error("服务发现失败",
				zap.String("service", matchedRoute.ServiceName),
				zap.Error(err),
			)
			return c.Status(503).JSON(fiber.Map{"code": 503, "message": "服务不可用"})
		}

		// 简单轮询负载均衡
		service := services[0]
		if len(service.Nodes) == 0 {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
				"code":    503,
				"message": "服务节点不可用",
			})
		}

		node := service.Nodes[time.Now().UnixNano()%int64(len(service.Nodes))]

		// 代理请求
		return g.proxyRequest(c, node.Address, matchedRoute)
	}
}

// proxyRequest 代理请求到后端服务
func (g *Gateway) proxyRequest(c *fiber.Ctx, targetAddr string, route *ServiceRoute) error {
	targetURL, err := url.Parse("http://" + targetAddr)
	if err != nil {
		logger.Error("解析目标URL失败", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"code": 500, "message": "内部错误"})
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 修改请求，需要在 handler 中设置
	originalDirector := proxy.Director

	clientIP := c.IP()
	reqHost := c.Hostname()
	scheme := c.Protocol()

	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// 路径转换：去除网关前缀，保留服务路径
		// 例如: /api/v1/logs/health -> /health
		if route.StripPrefix && route.PathPrefix != "" {
			newPath := strings.TrimPrefix(req.URL.Path, route.PathPrefix)
			if newPath == "" || newPath[0] != '/' {
				newPath = "/" + newPath
			}
			req.URL.Path = route.TargetPrefix + strings.TrimPrefix(newPath, "/")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			// 规范化路径
			if !strings.HasPrefix(req.URL.Path, "/") {
				req.URL.Path = "/" + req.URL.Path
			}
		}

		// 传递原始请求信息
		req.Header.Set("X-Forwarded-For", clientIP)
		req.Header.Set("X-Real-IP", clientIP)
		req.Header.Set("X-Forwarded-Proto", scheme)
		req.Header.Set("X-Forwarded-Host", reqHost)
	}

	// 错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("代理请求失败",
			zap.String("target", targetAddr),
			zap.Error(err),
		)
	}

	// 将 net/http handler 转为 fasthttp handler 并直接调用
	fastHandler := fasthttpadaptor.NewFastHTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	fastHandler(c.Context())
	return nil
}

// HealthCheck 健康检查
func (g *Gateway) HealthCheck(c *fiber.Ctx) error {
	return c.Status(200).JSON(fiber.Map{"status": "healthy", "service": "gateway", "time": time.Now().Format(time.RFC3339)})
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Nodes     int      `json:"nodes"`
	Addresses []string `json:"addresses,omitempty"`
}

// GetServicesStatus 获取所有服务状态
func (g *Gateway) GetServicesStatus(c *fiber.Ctx) error {
	var statuses []ServiceStatus

	// 获取所有注册的服务
	services, err := g.registry.ListServices()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"code": 500, "message": "获取服务列表失败"})
	}

	for _, svc := range services {
		// 获取服务详情
		svcDetails, err := g.registry.GetService(svc.Name)
		if err != nil {
			statuses = append(statuses, ServiceStatus{
				Name:   svc.Name,
				Status: "unknown",
				Nodes:  0,
			})
			continue
		}

		var addresses []string
		nodeCount := 0
		for _, s := range svcDetails {
			for _, node := range s.Nodes {
				addresses = append(addresses, node.Address)
				nodeCount++
			}
		}

		status := "unhealthy"
		if nodeCount > 0 {
			status = "healthy"
		}

		statuses = append(statuses, ServiceStatus{
			Name:      svc.Name,
			Status:    status,
			Nodes:     nodeCount,
			Addresses: addresses,
		})
	}

	return c.Status(200).JSON(fiber.Map{"code": 0, "message": "success", "data": statuses})
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	failures    int
	threshold   int
	timeout     time.Duration
	lastFailure time.Time
	state       string // closed, open, half-open
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     "closed",
	}
}

// Allow 是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	switch cb.state {
	case "closed":
		return true
	case "open":
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = "half-open"
			return true
		}
		return false
	case "half-open":
		return true
	}
	return false
}

// Success 记录成功
func (cb *CircuitBreaker) Success() {
	cb.failures = 0
	cb.state = "closed"
}

// Failure 记录失败
func (cb *CircuitBreaker) Failure() {
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

// State 获取状态
func (cb *CircuitBreaker) State() string {
	return cb.state
}

// Shutdown 关闭网关
func (g *Gateway) Shutdown(ctx context.Context) error {
	logger.Info("正在关闭网关...")
	// 停止监听服务变化
	g.StopWatch()
	// 清理资源
	return nil
}

// GetRoutes 获取所有已注册的路由（用于调试）
func (g *Gateway) GetRoutes() map[string]*ServiceRoute {
	g.mu.RLock()
	defer g.mu.RUnlock()

	routes := make(map[string]*ServiceRoute)
	for k, v := range g.routes {
		routes[k] = v
	}
	return routes
}
