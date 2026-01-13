package response

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data,omitempty"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// 响应码定义
const (
	CodeSuccess       = 0
	CodeError         = 1
	CodeUnauthorized  = 401
	CodeForbidden     = 403
	CodeNotFound      = 404
	CodeValidateError = 422
	CodeServerError   = 500
)

// 响应消息定义
const (
	MsgSuccess       = "success"
	MsgError         = "error"
	MsgUnauthorized  = "unauthorized"
	MsgForbidden     = "forbidden"
	MsgNotFound      = "not found"
	MsgValidateError = "validation error"
	MsgServerError   = "server error"
)

// Success 成功响应
func Success(c *fiber.Ctx, data interface{}) error {
	return c.Status(http.StatusOK).JSON(Response{
		Code:    CodeSuccess,
		Message: MsgSuccess,
		Data:    data,
	})
}

// SuccessWithMessage 成功响应(带消息)
func SuccessWithMessage(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(http.StatusOK).JSON(Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// SuccessPage 分页成功响应
func SuccessPage(c *fiber.Ctx, data interface{}, total int64, page, pageSize int) error {
	return c.Status(http.StatusOK).JSON(PageResponse{
		Code:     CodeSuccess,
		Message:  MsgSuccess,
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Error 错误响应
func Error(c *fiber.Ctx, code int, message string) error {
	return c.Status(http.StatusOK).JSON(Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithData 错误响应(带数据)
func ErrorWithData(c *fiber.Ctx, code int, message string, data interface{}) error {
	return c.Status(http.StatusOK).JSON(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// BadRequest 请求错误
func BadRequest(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(Response{
		Code:    CodeError,
		Message: message,
	})
}

// Unauthorized 未授权
func Unauthorized(c *fiber.Ctx, message string) error {
	if message == "" {
		message = MsgUnauthorized
	}
	return c.Status(http.StatusUnauthorized).JSON(Response{
		Code:    CodeUnauthorized,
		Message: message,
	})
}

// Forbidden 禁止访问
func Forbidden(c *fiber.Ctx, message string) error {
	if message == "" {
		message = MsgForbidden
	}
	return c.Status(http.StatusForbidden).JSON(Response{
		Code:    CodeForbidden,
		Message: message,
	})
}

// NotFound 未找到
func NotFound(c *fiber.Ctx, message string) error {
	if message == "" {
		message = MsgNotFound
	}
	return c.Status(http.StatusNotFound).JSON(Response{
		Code:    CodeNotFound,
		Message: message,
	})
}

// ValidateError 验证错误
func ValidateError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusUnprocessableEntity).JSON(Response{
		Code:    CodeValidateError,
		Message: message,
	})
}

// ParamError 请求参数错误
func ParamError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(Response{
		Code:    CodeError,
		Message: message,
	})
}

// PageSuccess 向后向兼容的分页响应
func PageSuccess(c *fiber.Ctx, data interface{}, total int64, page, pageSize int) error {
	return SuccessPage(c, data, total, page, pageSize)
}

// ServerError 服务器错误
func ServerError(c *fiber.Ctx, message string) error {
	if message == "" {
		message = MsgServerError
	}
	return c.Status(http.StatusInternalServerError).JSON(Response{
		Code:    CodeServerError,
		Message: message,
	})
}

// Abort 中止请求
func Abort(c *fiber.Ctx, httpCode int, code int, message string) error {
	return c.Status(httpCode).JSON(Response{
		Code:    code,
		Message: message,
	})
}
