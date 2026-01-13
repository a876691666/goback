package user

import "github.com/goback/pkg/dal"

// CreateRequest 创建用户请求
type CreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=50"`
	Nickname string `json:"nickname"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	RoleID   int64  `json:"roleId"`
	Status   int8   `json:"status"`
}

// UpdateRequest 更新用户请求
type UpdateRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	RoleID   int64  `json:"roleId"`
	Status   int8   `json:"status"`
}

// ListRequest 用户列表请求（使用 PocketBase 风格参数）
type ListRequest = dal.ListParams

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=50"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}
