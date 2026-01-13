package model

import (
	"github.com/goback/pkg/dal"
)

// Menu 菜单模型
type Menu struct {
	dal.Model
	*dal.Collection[Menu] `gorm:"-" json:"-"`
	ParentID              int64   `gorm:"default:0;index" json:"parentId"`
	Name                  string  `gorm:"size:50;not null" json:"name"`
	Path                  string  `gorm:"size:255" json:"path"`
	Component             string  `gorm:"size:255" json:"component"`
	Icon                  string  `gorm:"size:50" json:"icon"`
	Type                  int8    `gorm:"default:1" json:"type"` // 1:目录 2:菜单 3:按钮
	Visible               int8    `gorm:"default:1" json:"visible"`
	Status                int8    `gorm:"default:1" json:"status"`
	Redirect              string  `gorm:"size:255" json:"redirect"`
	Sort                  int     `gorm:"default:0" json:"sort"`
	PermCode              string  `gorm:"size:100" json:"permCode"` // 权限标识
	Children              []*Menu `gorm:"-" json:"children,omitempty"`
}

func (Menu) TableName() string { return "sys_menu" }

var Menus = &Menu{
	Collection: &dal.Collection[Menu]{
		DefaultSort: "sort,-id",
		MaxPerPage:  500,
		FieldAlias: map[string]string{
			"parentId":  "parent_id",
			"permCode":  "perm_code",
			"createdAt": "created_at",
			"updatedAt": "updated_at",
		},
	},
}

// Save 保存菜单
func (c *Menu) Save(data *Menu) error {
	return c.DB().Save(data).Error
}

// PermissionMenu 权限菜单关联
type PermissionMenu struct {
	ID                            int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	*dal.Collection[PermissionMenu] `gorm:"-" json:"-"`
	PermissionID                  int64 `gorm:"index:idx_perm_menu;not null" json:"permissionId"`
	MenuID                        int64 `gorm:"index:idx_perm_menu;not null" json:"menuId"`
}

func (PermissionMenu) TableName() string { return "sys_permission_menu" }

var PermissionMenus = &PermissionMenu{
	Collection: &dal.Collection[PermissionMenu]{},
}

// DeleteByPermissionID 根据权限ID删除关联
func (c *PermissionMenu) DeleteByPermissionID(permissionID int64) error {
	return c.DB().Where("permission_id = ?", permissionID).Delete(&PermissionMenu{}).Error
}
