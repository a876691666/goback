package middleware

import (
	"strings"
	"time"

	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/lifecycle"
	"github.com/goback/pkg/logger"
	"github.com/goback/pkg/response"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// JWTAuth JWT认证中间件
func JWTAuth(jwtManager *auth.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 从Header获取token
		token := c.Get("Authorization")
		if token == "" {
			// 尝试从query参数获取
			token = c.Query("token")
		}

		if token == "" {
			return response.Error(c, 401, "未提供认证令牌")
		}

		// 去除Bearer前缀
		token = strings.TrimPrefix(token, "Bearer ")

		// 验证token
		claims, err := jwtManager.ParseToken(token)
		if err != nil {
			return response.Error(c, 401, "无效的认证令牌")
		}

		// 将用户信息存入上下文
		c.Locals("userId", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("roleId", claims.RoleID)
		c.Locals("roleCode", claims.RoleCode)
		c.Locals("claims", claims)

		return c.Next()
	}
}

// OperationLogFunc 操作日志记录回调函数类型
type OperationLogFunc func(userID int64, username, module, action, method, path, ip, userAgent, reqBody string, status int, respBody string, latency time.Duration)

// OperationLog 操作日志记录中间件
func OperationLog(logFunc OperationLogFunc, moduleName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 记录开始时间
		startTime := time.Now()

		// 读取请求体
		var reqBody string
		if c.Method() != fiber.MethodGet {
			reqBody = string(c.Body())
		}

		// 处理请求
		if err := c.Next(); err != nil {
			return err
		}

		// 计算耗时
		latency := time.Since(startTime)

		// 获取用户信息
		userID := c.Locals("userId")
		username := c.Locals("username")

		userIDInt := int64(0)
		if userID != nil {
			userIDInt = userID.(int64)
		}
		usernameStr := ""
		if username != nil {
			usernameStr = username.(string)
		}

		// 获取操作类型
		action := getActionByMethod(c.Method())

		// 记录日志
		if logFunc != nil {
			logFunc(
				userIDInt,
				usernameStr,
				moduleName,
				action,
				c.Method(),
				c.Path(),
				c.IP(),
				c.Get("User-Agent"),
				reqBody,
				c.Response().StatusCode(),
				string(c.Response().Body()),
				latency,
			)
		}

		return nil
	}
}

// bodyLogWriter 自定义ResponseWriter用于记录响应体
// Fiber 使用 c.Response()，无需自定义 ResponseWriter

// getActionByMethod 根据HTTP方法获取操作类型
func getActionByMethod(method string) string {
	switch method {
	case "POST":
		return "新增"
	case "PUT", "PATCH":
		return "修改"
	case "DELETE":
		return "删除"
	case "GET":
		return "查询"
	default:
		return "其他"
	}
}

// Recovery 恢复中间件
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Path()),
					zap.String("method", c.Method()),
				)
				_ = response.Error(c, 500, "服务器内部错误")
			}
		}()
		return c.Next()
	}
}

// Cors 跨域中间件
func Cors() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		origin := c.Get("Origin")

		if origin != "" {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
			c.Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			c.Set("Access-Control-Allow-Credentials", "true")
		}

		if method == "OPTIONS" {
			return c.SendStatus(204)
		}

		return c.Next()
	}
}

// RateLimiter 限流中间件（简单实现）
type RateLimiter struct {
	rate     int           // 每秒请求数
	burst    int           // 突发请求数
	tokens   chan struct{} // 令牌桶
	interval time.Duration
}

// NewRateLimiter 创建限流器
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:     rate,
		burst:    burst,
		tokens:   make(chan struct{}, burst),
		interval: time.Second / time.Duration(rate),
	}

	// 初始化令牌桶
	for i := 0; i < burst; i++ {
		rl.tokens <- struct{}{}
	}

	// 启动令牌补充协程
	go rl.refillTokens()

	return rl
}

func (rl *RateLimiter) refillTokens() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}
}

// Middleware 限流中间件
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		select {
		case <-rl.tokens:
			return c.Next()
		default:
			return response.Error(c, 429, "请求过于频繁，请稍后重试")
		}
	}
}

// RequestID 请求ID中间件
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Locals("requestId", requestID)
		c.Set("X-Request-ID", requestID)
		return c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *fiber.Ctx) int64 {
	userID := c.Locals("userId")
	if userID == nil {
		return 0
	}
	return userID.(int64)
}

// GetUsername 从上下文获取用户名
func GetUsername(c *fiber.Ctx) string {
	username := c.Locals("username")
	if username == nil {
		return ""
	}
	return username.(string)
}

// GetRoleID 从上下文获取角色ID
func GetRoleID(c *fiber.Ctx) int64 {
	roleID := c.Locals("roleId")
	if roleID == nil {
		return 0
	}
	return roleID.(int64)
}

// GetRoleCode 从上下文获取角色编码
func GetRoleCode(c *fiber.Ctx) string {
	roleCode := c.Locals("roleCode")
	if roleCode == nil {
		return ""
	}
	return roleCode.(string)
}

// ErrorHandler 统一错误处理中间件
func ErrorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil {
			// 根据错误类型返回不同的响应
			switch e := err.(type) {
			case *errors.AppError:
				_ = response.Error(c, e.Code, e.Message)
			default:
				_ = response.Error(c, 500, "服务器内部错误")
			}
			return nil
		}
		return nil
	}
}

// RBACAuth RBAC权限认证中间件
// 使用 RBACCache 作为数据源，自动订阅 RBAC 数据更新
func RBACAuth(rbacCache *lifecycle.RBACCache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. 获取用户ID和角色ID
		userID := c.Locals("userId")
		roleID := c.Locals("roleId")
		if userID == nil || roleID == nil {
			return response.Error(c, 401, "未获取到用户信息")
		}

		roleIDInt := roleID.(int64)

		// 2. 检查RBAC缓存是否就绪
		if !rbacCache.IsReady() {
			logger.Warn("RBAC缓存未就绪")
			return response.Error(c, 500, "权限系统初始化中，请稍后重试")
		}

		// 3. 获取角色及其所有启用的子角色ID
		roleIDs, err := rbacCache.GetRoleAndDescendantIDs(roleIDInt)
		if err != nil {
			logger.Error("获取角色信息失败", zap.Error(err), zap.Int64("roleId", roleIDInt))
			return response.Error(c, 403, err.Error())
		}

		// 4. 聚合所有角色的权限
		permissionMap := rbacCache.GetAggregatedPermissions(roleIDs)

		// 5. 验证当前请求是否有权限
		path := c.Path()
		method := c.Method()

		hasPermission := false
		for _, perm := range permissionMap {
			if matchPermission(perm, path, method) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return response.Error(c, 403, "没有访问权限")
		}

		// 6. 将权限信息存入上下文，供后续使用
		permissions := make([]lifecycle.Permission, 0, len(permissionMap))
		for _, perm := range permissionMap {
			permissions = append(permissions, perm)
		}
		c.Locals("permissions", permissions)
		c.Locals("roleIds", roleIDs)

		return c.Next()
	}
}

// matchPermission 匹配权限
func matchPermission(perm lifecycle.Permission, path, method string) bool {
	// Resource 支持通配符匹配
	// 例如: /api/users/* 可以匹配 /api/users/1
	if !matchPath(perm.Resource, path) {
		return false
	}

	// Action 匹配 HTTP 方法
	// 支持: GET, POST, PUT, DELETE, PATCH, * (所有方法)
	if perm.Action != "*" && !strings.EqualFold(perm.Action, method) {
		return false
	}

	return true
}

// matchPath 路径匹配（支持通配符）
func matchPath(pattern, path string) bool {
	// 精确匹配
	if pattern == path {
		return true
	}

	// 通配符匹配
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}

	// ** 匹配所有路径
	if pattern == "**" || pattern == "/*" {
		return true
	}

	return false
}

// GetPermissions 从上下文获取用户权限列表
func GetPermissions(c *fiber.Ctx) []lifecycle.Permission {
	permissions := c.Locals("permissions")
	if permissions == nil {
		return []lifecycle.Permission{}
	}
	return permissions.([]lifecycle.Permission)
}

// GetRoleIDs 从上下文获取用户角色ID列表（包含子角色）
func GetRoleIDs(c *fiber.Ctx) []int64 {
	roleIDs := c.Locals("roleIds")
	if roleIDs == nil {
		return []int64{}
	}
	return roleIDs.([]int64)
}
