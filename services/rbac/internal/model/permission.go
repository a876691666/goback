package model

import "github.com/goback/pkg/dal"

// Permission 权限
type Permission struct {
	dal.Model
	*dal.Collection[Permission] `gorm:"-" json:"-"`
	Name                        string `gorm:"size:50;not null" json:"name"`
	Code                        string `gorm:"size:100;uniqueIndex;not null" json:"code"`
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

// GetByRoleIDs 根据多个角色ID获取权限列表（用于获取角色及其子角色的全部权限）
func (c *Permission) GetByRoleIDs(roleIDs []int64) ([]Permission, error) {
	if len(roleIDs) == 0 {
		return []Permission{}, nil
	}
	var permissions []Permission
	err := c.DB().
		Distinct().
		Joins("JOIN sys_role_permission ON sys_role_permission.permission_id = sys_permission.id").
		Where("sys_role_permission.role_id IN ?", roleIDs).
		Find(&permissions).Error
	return permissions, err
}

// RolePermission 角色权限关联
type RolePermission struct {
	ID                              int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	*dal.Collection[RolePermission] `gorm:"-" json:"-"`
	RoleID                          int64 `gorm:"index:idx_role_perm;not null" json:"roleId"`
	PermissionID                    int64 `gorm:"index:idx_role_perm;not null" json:"permissionId"`
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

// ================== PermissionScope 数据过滤规则 ==================

// PermissionScope 数据过滤规则（承载SSQL）
type PermissionScope struct {
	dal.Model
	*dal.Collection[PermissionScope] `gorm:"-" json:"-"`
	PermissionID                     int64  `gorm:"index;not null" json:"permissionId"`
	Name                             string `gorm:"size:50;not null" json:"name"`
	ScopeTableName                   string `gorm:"column:table_name;size:100;not null" json:"tableName"`
	SSQLRule                         string `gorm:"size:1000;not null" json:"ssqlRule"`
	Description                      string `gorm:"size:255" json:"description"`
}

func (PermissionScope) TableName() string { return "sys_permission_scope" }

// PermissionScopes 数据过滤规则 Collection 实例
var PermissionScopes = &PermissionScope{
	Collection: &dal.Collection[PermissionScope]{
		DefaultSort: "-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"permissionId": "permission_id",
			"tableName":    "table_name",
			"ssqlRule":     "ssql_rule",
			"createdAt":    "created_at",
			"updatedAt":    "updated_at",
		},
	},
}

// Save 保存数据过滤规则
func (c *PermissionScope) Save(data *PermissionScope) error {
	return c.DB().Save(data).Error
}

// GetByPermissionID 根据权限ID获取数据过滤规则列表
func (c *PermissionScope) GetByPermissionID(permissionID int64) ([]PermissionScope, error) {
	var scopes []PermissionScope
	err := c.DB().Where("permission_id = ?", permissionID).Find(&scopes).Error
	return scopes, err
}

// GetByPermissionIDs 根据多个权限ID获取数据过滤规则列表
func (c *PermissionScope) GetByPermissionIDs(permissionIDs []int64) ([]PermissionScope, error) {
	if len(permissionIDs) == 0 {
		return []PermissionScope{}, nil
	}
	var scopes []PermissionScope
	err := c.DB().Where("permission_id IN ?", permissionIDs).Find(&scopes).Error
	return scopes, err
}

// GetByScopeTableName 根据表名获取数据过滤规则列表
func (c *PermissionScope) GetByScopeTableName(tableName string, permissionIDs []int64) ([]PermissionScope, error) {
	if len(permissionIDs) == 0 {
		return []PermissionScope{}, nil
	}
	var scopes []PermissionScope
	err := c.DB().Where("table_name = ? AND permission_id IN ?", tableName, permissionIDs).Find(&scopes).Error
	return scopes, err
}

// DeleteByPermissionID 根据权限ID删除数据过滤规则
func (c *PermissionScope) DeleteByPermissionID(permissionID int64) error {
	return c.DB().Where("permission_id = ?", permissionID).Delete(&PermissionScope{}).Error
}
