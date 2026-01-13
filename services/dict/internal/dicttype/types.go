package dicttype

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

// ListRequest 字典类型列表请求
type ListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Name     string `form:"name"`
	Code     string `form:"code"`
	Status   *int8  `form:"status"`
}
