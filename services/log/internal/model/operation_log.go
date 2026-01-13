package model

import "github.com/goback/pkg/dal"

// OperationLog 操作日志
type OperationLog struct {
	dal.Model
	*dal.Collection[OperationLog] `gorm:"-" json:"-"`
	UserID                        int64  `gorm:"index" json:"userId"`
	Username                      string `gorm:"size:50" json:"username"`
	Module                        string `gorm:"size:50" json:"module"`
	Action                        string `gorm:"size:50" json:"action"`
	Method                        string `gorm:"size:10" json:"method"`
	Path                          string `gorm:"size:255" json:"path"`
	Query                         string `gorm:"type:text" json:"query"`
	Body                          string `gorm:"type:text" json:"body"`
	IP                            string `gorm:"size:50" json:"ip"`
	UserAgent                     string `gorm:"size:500" json:"userAgent"`
	Status                        int    `gorm:"default:1" json:"status"` // 1:成功 0:失败
	ErrorMessage                  string `gorm:"type:text" json:"errorMessage"`
	Duration                      int64  `gorm:"default:0" json:"duration"` // 执行时长(ms)
}

func (OperationLog) TableName() string { return "sys_operation_log" }

// OperationLogs 操作日志 Collection 实例
var OperationLogs = &OperationLog{
	Collection: &dal.Collection[OperationLog]{
		DefaultSort: "-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
			"userId":    "user_id",
			"userAgent": "user_agent",
		},
	},
}

// DeleteByIDs 根据ID列表删除记录
func (c *OperationLog) DeleteByIDs(ids []int64) error {
	return c.DB().Where("id IN ?", ids).Delete(&OperationLog{}).Error
}
