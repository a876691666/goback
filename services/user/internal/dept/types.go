package dept

// CreateRequest 创建部门请求
type CreateRequest struct {
	ParentID int64  `json:"parentId"`
	Name     string `json:"name" binding:"required,min=1,max=50"`
	Sort     int    `json:"sort"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email" binding:"omitempty,email"`
	Status   int8   `json:"status"`
}

// UpdateRequest 更新部门请求
type UpdateRequest struct {
	ParentID int64  `json:"parentId"`
	Name     string `json:"name"`
	Sort     int    `json:"sort"`
	Leader   string `json:"leader"`
	Phone    string `json:"phone"`
	Email    string `json:"email" binding:"omitempty,email"`
	Status   int8   `json:"status"`
}
