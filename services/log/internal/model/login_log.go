package model

import (
	"github.com/goback/pkg/dal"
)

// LoginLog 登录日志模型
type LoginLog struct {
	dal.Model
	UserID    int64  `gorm:"index" json:"userId"`
	Username  string `gorm:"size:50" json:"username"`
	IP        string `gorm:"size:50" json:"ip"`
	Location  string `gorm:"size:100" json:"location"`
	Browser   string `gorm:"size:100" json:"browser"`
	OS        string `gorm:"size:100" json:"os"`
	Status    int8   `gorm:"default:1" json:"status"` // 1:成功 0:失败
	Message   string `gorm:"size:255" json:"message"`
	LoginTime int64  `gorm:"autoCreateTime" json:"loginTime"`
}

// TableName 表名
func (LoginLog) TableName() string {
	return "sys_login_log"
}
