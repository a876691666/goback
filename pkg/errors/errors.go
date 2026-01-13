package errors

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	ErrNotFound          = New(404, "资源不存在")
	ErrUnauthorized      = New(401, "未授权")
	ErrForbidden         = New(403, "禁止访问")
	ErrBadRequest        = New(400, "请求错误")
	ErrInternalServer    = New(500, "服务器内部错误")
	ErrValidation        = New(422, "验证错误")
	ErrDuplicateEntry    = New(409, "数据已存在")
	ErrInvalidCredential = New(401, "用户名或密码错误")
	ErrTokenExpired      = New(401, "令牌已过期")
	ErrTokenInvalid      = New(401, "令牌无效")
	ErrRecordNotFound    = New(404, "记录不存在")
	ErrRecordExists      = New(409, "记录已存在")
	ErrNotImplemented    = New(501, "功能未实现")
)

// AppError 应用错误
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 解包错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(err error, code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapWithMsg 用消息包装错误
func WrapWithMsg(err error, message string) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:    appErr.Code,
			Message: message,
			Err:     err,
		}
	}
	return &AppError{
		Code:    500,
		Message: message,
		Err:     err,
	}
}

// Is 检查是否为指定错误
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As 类型转换错误
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// GetCode 获取错误码
func GetCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return 500
}

// GetMessage 获取错误消息
func GetMessage(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Message
	}
	return err.Error()
}

// NotFound 创建未找到错误
func NotFound(resource string) *AppError {
	return &AppError{
		Code:    404,
		Message: fmt.Sprintf("%s不存在", resource),
	}
}

// BadRequest 创建请求错误
func BadRequest(message string) *AppError {
	return &AppError{
		Code:    400,
		Message: message,
	}
}

// Unauthorized 创建未授权错误
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "未授权"
	}
	return &AppError{
		Code:    401,
		Message: message,
	}
}

// Forbidden 创建禁止访问错误
func Forbidden(message string) *AppError {
	if message == "" {
		message = "禁止访问"
	}
	return &AppError{
		Code:    403,
		Message: message,
	}
}

// Validation 创建验证错误
func Validation(message string) *AppError {
	return &AppError{
		Code:    422,
		Message: message,
	}
}

// Internal 创建内部错误
func Internal(message string) *AppError {
	if message == "" {
		message = "服务器内部错误"
	}
	return &AppError{
		Code:    500,
		Message: message,
	}
}

// Duplicate 创建重复错误
func Duplicate(field string) *AppError {
	return &AppError{
		Code:    409,
		Message: fmt.Sprintf("%s已存在", field),
	}
}
