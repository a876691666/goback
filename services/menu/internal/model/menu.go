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

// ================== RoleMenu 角色菜单关联 ==================

// RoleMenu 角色菜单关联
type RoleMenu struct {
	ID                        int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	*dal.Collection[RoleMenu] `gorm:"-" json:"-"`
	RoleID                    int64 `gorm:"index:idx_role_menu;not null" json:"roleId"`
	MenuID                    int64 `gorm:"index:idx_role_menu;not null" json:"menuId"`
}

func (RoleMenu) TableName() string { return "sys_role_menu" }

// RoleMenus 角色菜单关联 Collection 实例
var RoleMenus = &RoleMenu{
	Collection: &dal.Collection[RoleMenu]{
		FieldAlias: map[string]string{
			"roleId": "role_id",
			"menuId": "menu_id",
		},
	},
}

// DeleteByRoleID 根据角色ID删除菜单关联
func (c *RoleMenu) DeleteByRoleID(roleID int64) error {
	return c.DB().Where("role_id = ?", roleID).Delete(&RoleMenu{}).Error
}

// GetMenuIDsByRoleID 根据角色ID获取菜单ID列表
func (c *RoleMenu) GetMenuIDsByRoleID(roleID int64) ([]int64, error) {
	var menuIDs []int64
	err := c.DB().Model(&RoleMenu{}).Where("role_id = ?", roleID).Pluck("menu_id", &menuIDs).Error
	return menuIDs, err
}

// GetMenuIDsByRoleIDs 根据多个角色ID获取菜单ID列表（去重）
func (c *RoleMenu) GetMenuIDsByRoleIDs(roleIDs []int64) ([]int64, error) {
	if len(roleIDs) == 0 {
		return []int64{}, nil
	}
	var menuIDs []int64
	err := c.DB().Model(&RoleMenu{}).Distinct("menu_id").Where("role_id IN ?", roleIDs).Pluck("menu_id", &menuIDs).Error
	return menuIDs, err
}

// GetMenusByRoleID 根据角色ID获取菜单列表
func (c *Menu) GetByRoleID(roleID int64) ([]Menu, error) {
	var menus []Menu
	err := c.DB().
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Where("sys_role_menu.role_id = ?", roleID).
		Order("sort, id").
		Find(&menus).Error
	return menus, err
}

// GetMenusByRoleIDs 根据多个角色ID获取菜单列表（去重）
func (c *Menu) GetByRoleIDs(roleIDs []int64) ([]Menu, error) {
	if len(roleIDs) == 0 {
		return []Menu{}, nil
	}
	var menus []Menu
	err := c.DB().
		Distinct().
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Where("sys_role_menu.role_id IN ?", roleIDs).
		Order("sort, id").
		Find(&menus).Error
	return menus, err
}

// ================== PermissionMenu（保留兼容） ==================

// PermissionMenu 权限菜单关联
type PermissionMenu struct {
	ID                              int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	*dal.Collection[PermissionMenu] `gorm:"-" json:"-"`
	PermissionID                    int64 `gorm:"index:idx_perm_menu;not null" json:"permissionId"`
	MenuID                          int64 `gorm:"index:idx_perm_menu;not null" json:"menuId"`
}

func (PermissionMenu) TableName() string { return "sys_permission_menu" }

var PermissionMenus = &PermissionMenu{
	Collection: &dal.Collection[PermissionMenu]{},
}

// DeleteByPermissionID 根据权限ID删除关联
func (c *PermissionMenu) DeleteByPermissionID(permissionID int64) error {
	return c.DB().Where("permission_id = ?", permissionID).Delete(&PermissionMenu{}).Error
}
