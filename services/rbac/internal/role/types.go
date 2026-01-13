package role

import "github.com/goback/pkg/dal"

// CreateRequest 创建角色请求
type CreateRequest struct {
	ParentID    int64  `json:"parentId"`
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Status      int8   `json:"status"`
	Sort        int    `json:"sort"`
	Description string `json:"description"`
}

// UpdateRequest 更新角色请求
type UpdateRequest struct {
	ParentID    int64  `json:"parentId"`
	Name        string `json:"name"`
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

// SetMenusRequest 设置角色菜单请求
type SetMenusRequest struct {
	MenuIDs []int64 `json:"menuIds"`
}

