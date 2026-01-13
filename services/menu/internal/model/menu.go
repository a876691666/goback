package model

import (
	"github.com/goback/pkg/dal"
)

// Menu 菜单模型
type Menu struct {
	dal.Model
	ParentID  int64   `gorm:"default:0;index" json:"parentId"`
	Name      string  `gorm:"size:50;not null" json:"name"`
	Path      string  `gorm:"size:255" json:"path"`
	Component string  `gorm:"size:255" json:"component"`
	Icon      string  `gorm:"size:50" json:"icon"`
	Type      int8    `gorm:"default:1" json:"type"` // 1:目录 2:菜单 3:按钮
	Visible   int8    `gorm:"default:1" json:"visible"`
	Status    int8    `gorm:"default:1" json:"status"`
	Redirect  string  `gorm:"size:255" json:"redirect"`
	Sort      int     `gorm:"default:0" json:"sort"`
	PermCode  string  `gorm:"size:100" json:"permCode"` // 权限标识
	Children  []*Menu `gorm:"-" json:"children,omitempty"`
}

// TableName 表名
func (Menu) TableName() string {
	return "sys_menu"
}

// PermissionMenu 权限菜单关联
type PermissionMenu struct {
	ID           int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	PermissionID int64 `gorm:"index:idx_perm_menu;not null" json:"permissionId"`
	MenuID       int64 `gorm:"index:idx_perm_menu;not null" json:"menuId"`
}

// TableName 表名
func (PermissionMenu) TableName() string {
	return "sys_permission_menu"
}
