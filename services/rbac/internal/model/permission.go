package model

import (
	"github.com/goback/pkg/dal"
)

// Permission 权限模型
type Permission struct {
	dal.Model
	Name        string `gorm:"size:50;not null" json:"name"`
	Code        string `gorm:"size:100;uniqueIndex;not null" json:"code"`
	Type        int8   `gorm:"default:1" json:"type"` // 1:菜单 2:按钮 3:API 4:数据
	Resource    string `gorm:"size:255" json:"resource"`
	Action      string `gorm:"size:50" json:"action"`
	Description string `gorm:"size:255" json:"description"`
}

// TableName 表名
func (Permission) TableName() string {
	return "sys_permission"
}

// RolePermission 角色权限关联
type RolePermission struct {
	ID           int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID       int64 `gorm:"index:idx_role_perm;not null" json:"roleId"`
	PermissionID int64 `gorm:"index:idx_role_perm;not null" json:"permissionId"`
}

// TableName 表名
func (RolePermission) TableName() string {
	return "sys_role_permission"
}
