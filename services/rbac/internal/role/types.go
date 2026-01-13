package role

import "github.com/goback/pkg/dal"

// CreateRequest 创建角色请求
type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	DataScope   int8   `json:"dataScope"`
	Status      int8   `json:"status"`
	Sort        int    `json:"sort"`
	Description string `json:"description"`
}

// UpdateRequest 更新角色请求
type UpdateRequest struct {
	Name        string `json:"name"`
	DataScope   int8   `json:"dataScope"`
	Status      int8   `json:"status"`
	Sort        int    `json:"sort"`
	Description string `json:"description"`
}

// ListRequest 角色列表请求（使用 PocketBase 风格参数）
type ListRequest = dal.ListParams

// SetPermissionsRequest 设置角色权限请求
type SetPermissionsRequest struct {
	PermissionIDs []int64 `json:"permissionIds"`
}

// SetDataScopeRequest 设置角色数据权限请求
type SetDataScopeRequest struct {
	DeptIDs []int64 `json:"deptIds"`
}
