package permissionscope

import "github.com/goback/pkg/dal"

// CreateRequest 创建数据过滤规则请求
type CreateRequest struct {
	PermissionID int64  `json:"permissionId" binding:"required"`
	Name         string `json:"name" binding:"required"`
	TableName    string `json:"tableName" binding:"required"`
	SSQLRule     string `json:"ssqlRule" binding:"required"`
	Description  string `json:"description"`
}

// UpdateRequest 更新数据过滤规则请求
type UpdateRequest struct {
	Name        string `json:"name"`
	TableName   string `json:"tableName"`
	SSQLRule    string `json:"ssqlRule"`
	Description string `json:"description"`
}

// ListRequest 数据过滤规则列表请求
type ListRequest = dal.ListParams
