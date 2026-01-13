package dicttype

import "github.com/goback/pkg/dal"

// CreateRequest 创建字典类型请求
type CreateRequest struct {
	Name   string `json:"name" binding:"required"`
	Code   string `json:"code" binding:"required"`
	Status int8   `json:"status"`
	Remark string `json:"remark"`
}

// UpdateRequest 更新字典类型请求
type UpdateRequest struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Status *int8  `json:"status"`
	Remark string `json:"remark"`
}

// ListRequest 字典类型列表请求（使用 PocketBase 风格参数）
type ListRequest = dal.ListParams
