package model

import (
	"github.com/goback/pkg/dal"
)

// User 用户模型
type User struct {
	dal.Model
	Username string `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password string `gorm:"size:255;not null" json:"-"`
	Nickname string `gorm:"size:50" json:"nickname"`
	Email    string `gorm:"size:100" json:"email"`
	Phone    string `gorm:"size:20" json:"phone"`
	Avatar   string `gorm:"size:255" json:"avatar"`
	Status   int8   `gorm:"default:1" json:"status"` // 1:正常 0:禁用
	RoleID   int64  `gorm:"index" json:"roleId"`
	DeptID   int64  `gorm:"index" json:"deptId"`
	Role     *Role  `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// TableName 表名
func (User) TableName() string {
	return "sys_user"
}

// Role 角色模型（用于关联查询）
type Role struct {
	dal.Model
	Name        string `gorm:"size:50;not null" json:"name"`
	Code        string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	DataScope   int8   `gorm:"default:1" json:"dataScope"`
	Status      int8   `gorm:"default:1" json:"status"`
	Sort        int    `gorm:"default:0" json:"sort"`
	Description string `gorm:"size:255" json:"description"`
}

// TableName 表名
func (Role) TableName() string {
	return "sys_role"
}
