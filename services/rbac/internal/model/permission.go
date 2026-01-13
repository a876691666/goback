package model

import "github.com/goback/pkg/dal"

// Permission 权限
type Permission struct {
	dal.Model
	*dal.Collection[Permission] `gorm:"-" json:"-"`
	Name                        string `gorm:"size:50;not null" json:"name"`
	Code                        string `gorm:"size:100;uniqueIndex;not null" json:"code"`
	Type                        int8   `gorm:"default:1" json:"type"` // 1:菜单 2:按钮 3:API 4:数据
	Resource                    string `gorm:"size:255" json:"resource"`
	Action                      string `gorm:"size:50" json:"action"`
	Description                 string `gorm:"size:255" json:"description"`
}

func (Permission) TableName() string { return "sys_permission" }

// Permissions 权限 Collection 实例
var Permissions = &Permission{
	Collection: &dal.Collection[Permission]{
		DefaultSort: "-id",
		MaxPerPage:  500,
		FieldAlias: map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
		},
	},
}

// Save 保存权限
func (c *Permission) Save(data *Permission) error {
	return c.DB().Save(data).Error
}

// ExistsByCode 检查编码是否存在
func (c *Permission) ExistsByCode(code string, excludeID ...int64) (bool, error) {
	var count int64
	db := c.DB().Model(&Permission{}).Where("code = ?", code)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}
	err := db.Count(&count).Error
	return count > 0, err
}

// GetByRoleID 根据角色ID获取权限列表
func (c *Permission) GetByRoleID(roleID int64) ([]Permission, error) {
	var permissions []Permission
	err := c.DB().
		Joins("JOIN sys_role_permission ON sys_role_permission.permission_id = sys_permission.id").
		Where("sys_role_permission.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}

// RolePermission 角色权限关联
type RolePermission struct {
	ID                             int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	*dal.Collection[RolePermission] `gorm:"-" json:"-"`
	RoleID                         int64 `gorm:"index:idx_role_perm;not null" json:"roleId"`
	PermissionID                   int64 `gorm:"index:idx_role_perm;not null" json:"permissionId"`
}

func (RolePermission) TableName() string { return "sys_role_permission" }

// RolePermissions 角色权限 Collection 实例
var RolePermissions = &RolePermission{
	Collection: &dal.Collection[RolePermission]{},
}

// DeleteByRoleID 根据角色ID删除权限关联
func (c *RolePermission) DeleteByRoleID(roleID int64) error {
	return c.DB().Where("role_id = ?", roleID).Delete(&RolePermission{}).Error
}
