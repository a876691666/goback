package middleware

import (
	"strings"
	"time"

	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/errors"
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

// Permission 权限验证中间件
func Permission(casbinSvc *auth.CasbinService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleCode := c.Locals("roleCode")
		if roleCode == nil {
			return response.Error(c, 403, "未获取到角色信息")
		}

		// 获取请求路径和方法
		obj := c.Path()
		act := c.Method()

		// 检查权限
		if !casbinSvc.HasPermission(roleCode.(string), obj, act) {
			return response.Error(c, 403, "没有访问权限")
		}

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

// DataScope 数据权限中间件
func DataScope(dataScopeFunc func(userID int64, roleID int64) (*auth.DataScopeInfo, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("userId")
		if userID == nil {
			return c.Next()
		}

		roleID := c.Locals("roleId")
		if roleID == nil {
			return c.Next()
		}

		scopeInfo, err := dataScopeFunc(userID.(int64), roleID.(int64))
		if err != nil {
			logger.Error("获取数据权限失败", zap.Error(err))
			return c.Next()
		}

		c.Locals("dataScope", scopeInfo)
		return c.Next()
	}
}

// GetDataScope 从上下文获取数据权限
func GetDataScope(c *fiber.Ctx) *auth.DataScopeInfo {
	scopeInfo := c.Locals("dataScope")
	if scopeInfo == nil {
		return nil
	}
	return scopeInfo.(*auth.DataScopeInfo)
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
