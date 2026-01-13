package model

import (
	"github.com/goback/pkg/dal"
)

// OperationLog 操作日志模型
type OperationLog struct {
	dal.Model
	UserID       int64  `gorm:"index" json:"userId"`
	Username     string `gorm:"size:50" json:"username"`
	Module       string `gorm:"size:50" json:"module"`
	Action       string `gorm:"size:50" json:"action"`
	Method       string `gorm:"size:10" json:"method"`
	Path         string `gorm:"size:255" json:"path"`
	Query        string `gorm:"type:text" json:"query"`
	Body         string `gorm:"type:text" json:"body"`
	IP           string `gorm:"size:50" json:"ip"`
	UserAgent    string `gorm:"size:500" json:"userAgent"`
	Status       int    `gorm:"default:1" json:"status"` // 1:成功 0:失败
	ErrorMessage string `gorm:"type:text" json:"errorMessage"`
	Duration     int64  `gorm:"default:0" json:"duration"` // 执行时长(ms)
}

// TableName 表名
func (OperationLog) TableName() string {
	return "sys_operation_log"
}
