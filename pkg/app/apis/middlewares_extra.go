package apis

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/app/tools/hook"
	"github.com/goback/pkg/app/tools/router"
	"github.com/goback/pkg/app/tools/security"
)

// --- Recovery Middleware ---

const DefaultRecoveryMiddlewareId = "pbRecovery"

// Recovery returns a panic recovery middleware.
func Recovery() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       DefaultRecoveryMiddlewareId,
		Priority: -10000, // 最高优先级，最先执行
		Func: func(e *core.RequestEvent) (err error) {
			defer func() {
				if r := recover(); r != nil {
					recoverErr, ok := r.(error)
					if !ok {
						recoverErr = fmt.Errorf("%v", r)
					}

					stack := make([]byte, 4<<10) // 4 KB
					length := runtime.Stack(stack, false)
					err = e.InternalServerError("服务器内部错误", fmt.Errorf("[PANIC RECOVER] %w\n%s", recoverErr, stack[:length]))
				}
			}()

			return e.Next()
		},
	}
}

// --- RequestID Middleware ---

const DefaultRequestIDMiddlewareId = "pbRequestID"

// RequestID returns a middleware that generates and sets a unique request ID.
func RequestID() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       DefaultRequestIDMiddlewareId,
		Priority: -9000, // 高优先级
		Func: func(e *core.RequestEvent) error {
			requestID := e.Request.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = security.RandomString(32)
			}

			e.Set("requestId", requestID)
			e.Response.Header().Set("X-Request-ID", requestID)

			return e.Next()
		},
	}
}

// GetRequestID returns the request ID from the event context.
func GetRequestID(e *core.RequestEvent) string {
	if id := e.Get("requestId"); id != nil {
		return id.(string)
	}
	return ""
}

// --- Rate Limiter Middleware ---

const DefaultRateLimitMiddlewareId = "pbRateLimit"

// RateLimiterConfig defines the rate limiter configuration.
type RateLimiterConfig struct {
	Rate   int // 每秒请求数
	Burst  int // 突发请求数
	KeyGen func(*core.RequestEvent) string
}

// RateLimiter is a simple token bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	tokens   map[string]*tokenBucket
	rate     int
	burst    int
	keyGen   func(*core.RequestEvent) string
	interval time.Duration
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.Rate <= 0 {
		config.Rate = 100
	}
	if config.Burst <= 0 {
		config.Burst = config.Rate
	}
	if config.KeyGen == nil {
		config.KeyGen = func(e *core.RequestEvent) string {
			return e.RemoteIP()
		}
	}

	return &RateLimiter{
		tokens:   make(map[string]*tokenBucket),
		rate:     config.Rate,
		burst:    config.Burst,
		keyGen:   config.KeyGen,
		interval: time.Second / time.Duration(config.Rate),
	}
}

// Middleware returns the rate limiter middleware handler.
func (rl *RateLimiter) Middleware() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       DefaultRateLimitMiddlewareId,
		Priority: -8000,
		Func: func(e *core.RequestEvent) error {
			key := rl.keyGen(e)

			if !rl.allow(key) {
				return router.NewTooManyRequestsError("请求过于频繁，请稍后重试", nil)
			}

			return e.Next()
		},
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, ok := rl.tokens[key]

	if !ok {
		rl.tokens[key] = &tokenBucket{
			tokens:     rl.burst - 1,
			lastRefill: now,
		}
		return true
	}

	// 补充令牌
	elapsed := now.Sub(bucket.lastRefill)
	refillTokens := int(elapsed / rl.interval)
	if refillTokens > 0 {
		bucket.tokens = min(bucket.tokens+refillTokens, rl.burst)
		bucket.lastRefill = now
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// --- JWT Auth Middleware ---

const DefaultJWTAuthMiddlewareId = "pbJWTAuth"

// JWTConfig JWT认证配置
type JWTConfig struct {
	Validator core.JWTValidator
	// SkipPaths 跳过认证的路径（支持前缀匹配）
	SkipPaths []string
	// ErrorHandler 自定义错误处理
	ErrorHandler func(*core.RequestEvent, error) error
}

// JWTAuth returns a JWT authentication middleware.
func JWTAuth(config JWTConfig) *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       DefaultJWTAuthMiddlewareId,
		Priority: -5000,
		Func: func(e *core.RequestEvent) error {
			// 检查是否跳过
			path := e.Request.URL.Path
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(path, skipPath) {
					return e.Next()
				}
			}

			// 获取token
			token := e.Request.Header.Get("Authorization")
			if token == "" {
				token = e.Request.URL.Query().Get("token")
			}

			if token == "" {
				err := router.NewUnauthorizedError("未提供认证令牌", nil)
				if config.ErrorHandler != nil {
					return config.ErrorHandler(e, err)
				}
				return err
			}

			// 去除Bearer前缀
			token = strings.TrimPrefix(token, "Bearer ")

			// 验证token
			claims, err := config.Validator.ParseToken(token)
			if err != nil {
				authErr := router.NewUnauthorizedError("无效的认证令牌", nil)
				if config.ErrorHandler != nil {
					return config.ErrorHandler(e, authErr)
				}
				return authErr
			}

			// 将用户信息存入上下文
			e.Set("userId", claims.UserID)
			e.Set("username", claims.Username)
			e.Set("roleId", claims.RoleID)
			e.Set("roleCode", claims.RoleCode)
			e.Set("claims", claims)

			// 设置Auth字段
			authMap := map[string]any{
				"userId":   claims.UserID,
				"username": claims.Username,
				"roleId":   claims.RoleID,
				"roleCode": claims.RoleCode,
			}
			e.Auth = &authMap

			return e.Next()
		},
	}
}

// --- Auth Helper Functions ---

// GetUserID 从上下文获取用户ID
func GetUserID(e *core.RequestEvent) int64 {
	if id := e.Get("userId"); id != nil {
		return id.(int64)
	}
	return 0
}

// GetUsername 从上下文获取用户名
func GetUsername(e *core.RequestEvent) string {
	if username := e.Get("username"); username != nil {
		return username.(string)
	}
	return ""
}

// GetRoleID 从上下文获取角色ID
func GetRoleID(e *core.RequestEvent) int64 {
	if id := e.Get("roleId"); id != nil {
		return id.(int64)
	}
	return 0
}

// GetRoleCode 从上下文获取角色编码
func GetRoleCode(e *core.RequestEvent) string {
	if code := e.Get("roleCode"); code != nil {
		return code.(string)
	}
	return ""
}

// GetClaims 从上下文获取JWT Claims
func GetClaims(e *core.RequestEvent) *core.JWTClaims {
	if claims := e.Get("claims"); claims != nil {
		return claims.(*core.JWTClaims)
	}
	return nil
}

// --- Logger Middleware ---

const DefaultLoggerMiddlewareId = "pbLogger"

// LoggerConfig 日志中间件配置
type LoggerConfig struct {
	// SkipPaths 跳过日志的路径
	SkipPaths []string
	// LogFunc 自定义日志函数
	LogFunc func(method, path, ip string, status int, latency time.Duration)
}

// Logger returns a request logging middleware.
func Logger(config LoggerConfig) *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       DefaultLoggerMiddlewareId,
		Priority: -7000,
		Func: func(e *core.RequestEvent) error {
			// 检查是否跳过
			path := e.Request.URL.Path
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(path, skipPath) {
					return e.Next()
				}
			}

			start := time.Now()

			err := e.Next()

			latency := time.Since(start)
			status := e.Status()
			if status == 0 {
				status = http.StatusOK
			}

			if config.LogFunc != nil {
				config.LogFunc(e.Request.Method, path, e.RemoteIP(), status, latency)
			} else {
				fmt.Printf("[%s] %s %s %d %v\n",
					time.Now().Format("2006-01-02 15:04:05"),
					e.Request.Method,
					path,
					status,
					latency,
				)
			}

			return err
		},
	}
}

// --- Timeout Middleware ---

const DefaultTimeoutMiddlewareId = "pbTimeout"

// Timeout returns a request timeout middleware.
func Timeout(timeout time.Duration) *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       DefaultTimeoutMiddlewareId,
		Priority: -6000,
		Func: func(e *core.RequestEvent) error {
			done := make(chan error, 1)

			go func() {
				done <- e.Next()
			}()

			select {
			case err := <-done:
				return err
			case <-time.After(timeout):
				return router.NewApiError(http.StatusGatewayTimeout, "请求超时", nil)
			}
		},
	}
}

// --- Response Helpers ---

// SuccessResponse 成功响应
type SuccessResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// PagedResponse 分页响应
type PagedResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Total   int64  `json:"total"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

// Success 返回成功响应
func Success(e *core.RequestEvent, data any) error {
	return e.JSON(200, SuccessResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 返回带消息的成功响应
func SuccessWithMessage(e *core.RequestEvent, message string, data any) error {
	return e.JSON(200, SuccessResponse{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 返回错误响应
func Error(e *core.RequestEvent, code int, message string) error {
	return e.JSON(code, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// ErrorWithDetails 返回带详情的错误响应
func ErrorWithDetails(e *core.RequestEvent, code int, message string, details any) error {
	return e.JSON(code, ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// Paged 返回分页响应
func Paged(e *core.RequestEvent, data any, total int64, page, size int) error {
	return e.JSON(200, PagedResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// --- Request Helpers ---

// PageParams 分页参数
type PageParams struct {
	Page int `json:"page" form:"page"`
	Size int `json:"size" form:"size"`
}

// GetPageParams 获取分页参数
func GetPageParams(e *core.RequestEvent) PageParams {
	params := PageParams{
		Page: 1,
		Size: 10,
	}

	if pageStr := e.Request.URL.Query().Get("page"); pageStr != "" {
		if p := parseInt(pageStr); p > 0 {
			params.Page = p
		}
	}

	if sizeStr := e.Request.URL.Query().Get("size"); sizeStr != "" {
		if s := parseInt(sizeStr); s > 0 && s <= 100 {
			params.Size = s
		}
	}

	return params
}

// GetOffset 获取偏移量
func (p PageParams) GetOffset() int {
	return (p.Page - 1) * p.Size
}

func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			break
		}
	}
	return result
}

// GetPathParam 获取路径参数
func GetPathParam(e *core.RequestEvent, name string) string {
	return e.Request.PathValue(name)
}

// GetQueryParam 获取查询参数
func GetQueryParam(e *core.RequestEvent, name string) string {
	return e.Request.URL.Query().Get(name)
}

// GetQueryParamInt 获取整数查询参数
func GetQueryParamInt(e *core.RequestEvent, name string, defaultValue int) int {
	val := e.Request.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	result := parseInt(val)
	if result == 0 {
		return defaultValue
	}
	return result
}
