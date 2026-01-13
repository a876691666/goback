package model

import "github.com/goback/pkg/dal"

// LoginLog 登录日志
type LoginLog struct {
	dal.Model
	*dal.Collection[LoginLog] `gorm:"-" json:"-"`
	UserID                    int64  `gorm:"index" json:"userId"`
	Username                  string `gorm:"size:50" json:"username"`
	IP                        string `gorm:"size:50" json:"ip"`
	Location                  string `gorm:"size:100" json:"location"`
	Browser                   string `gorm:"size:100" json:"browser"`
	OS                        string `gorm:"size:100" json:"os"`
	Status                    int8   `gorm:"default:1" json:"status"` // 1:成功 0:失败
	Message                   string `gorm:"size:255" json:"message"`
	LoginTime                 int64  `gorm:"autoCreateTime" json:"loginTime"`
}

func (LoginLog) TableName() string { return "sys_login_log" }

// LoginLogs 登录日志 Collection 实例
var LoginLogs = &LoginLog{
	Collection: &dal.Collection[LoginLog]{
		DefaultSort: "-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
			"userId":    "user_id",
			"loginTime": "login_time",
		},
	},
}

// DeleteByIDs 根据ID列表删除记录
func (c *LoginLog) DeleteByIDs(ids []int64) error {
	return c.DB().Where("id IN ?", ids).Delete(&LoginLog{}).Error
}
