package model

import (
	"github.com/goback/pkg/dal"
)

// Role 角色模型
type Role struct {
	dal.Model
	Name        string `gorm:"size:50;not null" json:"name"`
	Code        string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	DataScope   int8   `gorm:"default:1" json:"dataScope"` // 数据权限范围
	Status      int8   `gorm:"default:1" json:"status"`
	Sort        int    `gorm:"default:0" json:"sort"`
	Description string `gorm:"size:255" json:"description"`
}

// TableName 表名
func (Role) TableName() string {
	return "sys_role"
}

// RoleDataScope 角色数据权限
type RoleDataScope struct {
	ID     int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID int64 `gorm:"index:idx_role_dept;not null" json:"roleId"`
	DeptID int64 `gorm:"index:idx_role_dept;not null" json:"deptId"`
}

// TableName 表名
func (RoleDataScope) TableName() string {
	return "sys_role_data_scope"
}
