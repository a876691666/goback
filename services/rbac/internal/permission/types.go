package permission

import "github.com/goback/pkg/dal"

// CreateRequest 创建权限请求
type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Type        int8   `json:"type"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// UpdateRequest 更新权限请求
type UpdateRequest struct {
	Name        string `json:"name"`
	Type        int8   `json:"type"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// ListRequest 权限列表请求（使用 PocketBase 风格参数）
type ListRequest = dal.ListParams
