package dictdata

// CreateRequest 创建字典数据请求
type CreateRequest struct {
	TypeID    int64  `json:"typeId" binding:"required"`
	Label     string `json:"label" binding:"required"`
	Value     string `json:"value" binding:"required"`
	Sort      int    `json:"sort"`
	Status    int8   `json:"status"`
	CSSClass  string `json:"cssClass"`
	ListClass string `json:"listClass"`
	Remark    string `json:"remark"`
}

// UpdateRequest 更新字典数据请求
type UpdateRequest struct {
	Label     string `json:"label"`
	Value     string `json:"value"`
	Sort      *int   `json:"sort"`
	Status    *int8  `json:"status"`
	CSSClass  string `json:"cssClass"`
	ListClass string `json:"listClass"`
	Remark    string `json:"remark"`
}
