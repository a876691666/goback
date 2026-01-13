package model

import "github.com/goback/pkg/dal"

// Role 角色
type Role struct {
	dal.Model
	*dal.Collection[Role] `gorm:"-" json:"-"`
	Name                  string `gorm:"size:50;not null" json:"name"`
	Code                  string `gorm:"size:50;uniqueIndex;not null" json:"code"`
	DataScope             int8   `gorm:"default:1" json:"dataScope"` // 数据权限范围
	Status                int8   `gorm:"default:1" json:"status"`
	Sort                  int    `gorm:"default:0" json:"sort"`
	Description           string `gorm:"size:255" json:"description"`
}

func (Role) TableName() string { return "sys_role" }

// Roles 角色 Collection 实例
var Roles = &Role{
	Collection: &dal.Collection[Role]{
		DefaultSort: "sort,-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
			"dataScope": "data_scope",
		},
	},
}

// Save 保存角色
func (c *Role) Save(data *Role) error {
	return c.DB().Save(data).Error
}

// ExistsByCode 检查编码是否存在
func (c *Role) ExistsByCode(code string, excludeID ...int64) (bool, error) {
	var count int64
	db := c.DB().Model(&Role{}).Where("code = ?", code)
	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}
	err := db.Count(&count).Error
	return count > 0, err
}

// RoleDataScope 角色数据权限
type RoleDataScope struct {
	ID                            int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	*dal.Collection[RoleDataScope] `gorm:"-" json:"-"`
	RoleID                        int64 `gorm:"index:idx_role_dept;not null" json:"roleId"`
	DeptID                        int64 `gorm:"index:idx_role_dept;not null" json:"deptId"`
}

func (RoleDataScope) TableName() string { return "sys_role_data_scope" }

// RoleDataScopes 角色数据权限 Collection 实例
var RoleDataScopes = &RoleDataScope{
	Collection: &dal.Collection[RoleDataScope]{},
}

// DeleteByRoleID 根据角色ID删除数据权限
func (c *RoleDataScope) DeleteByRoleID(roleID int64) error {
	return c.DB().Where("role_id = ?", roleID).Delete(&RoleDataScope{}).Error
}
