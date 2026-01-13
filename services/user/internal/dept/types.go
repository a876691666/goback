package dept

import "github.com/goback/pkg/dal"

// CreateRequest 创建部门请求
type CreateRequest struct {
	ParentID int64  `json:"parentId"`
	Name     string `json:"name" binding:"required"`
	Sort     int    `json:"sort"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Status   int8   `json:"status"`
}

// UpdateRequest 更新部门请求
type UpdateRequest struct {
	ParentID int64  `json:"parentId"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Status   int8   `json:"status"`
}

// ListRequest 部门列表请求（使用 PocketBase 风格参数）
type ListRequest = dal.ListParams
