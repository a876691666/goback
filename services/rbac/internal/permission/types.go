package permission

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

// ListRequest 权限列表请求
type ListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Name     string `form:"name"`
	Code     string `form:"code"`
	Type     *int8  `form:"type"`
}
