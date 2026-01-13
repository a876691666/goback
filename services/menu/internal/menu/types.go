package menu

import "github.com/goback/pkg/dal"

// CreateRequest 创建菜单请求
type CreateRequest struct {
	ParentID  int64  `json:"parentId"`
	Name      string `json:"name" binding:"required"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Type      int8   `json:"type"`
	Visible   int8   `json:"visible"`
	Status    int8   `json:"status"`
	Redirect  string `json:"redirect"`
	Sort      int    `json:"sort"`
	PermCode  string `json:"permCode"`
}

// UpdateRequest 更新菜单请求
type UpdateRequest struct {
	ParentID  int64  `json:"parentId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	Type      int8   `json:"type"`
	Visible   int8   `json:"visible"`
	Status    int8   `json:"status"`
	Redirect  string `json:"redirect"`
	Sort      int    `json:"sort"`
	PermCode  string `json:"permCode"`
}

// ListRequest 菜单列表请求（使用 PocketBase 风格参数）
type ListRequest = dal.ListParams
