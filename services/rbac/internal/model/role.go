package model

import (
	"sync"

	"github.com/goback/pkg/dal"
)

// Role 角色（树形结构）
type Role struct {
	dal.Model
	*dal.Collection[Role] `gorm:"-" json:"-"`
	ParentID              int64   `gorm:"default:0;index" json:"parentId"`
	Name                  string  `gorm:"size:50;not null" json:"name"`
	Code                  string  `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Status                int8    `gorm:"default:1" json:"status"`
	Sort                  int     `gorm:"default:0" json:"sort"`
	Description           string  `gorm:"size:255" json:"description"`
	Children              []*Role `gorm:"-" json:"children,omitempty"`
}

func (Role) TableName() string { return "sys_role" }

// Roles 角色 Collection 实例
var Roles = &Role{
	Collection: &dal.Collection[Role]{
		DefaultSort: "sort,-id",
		MaxPerPage:  100,
		FieldAlias: map[string]string{
			"parentId":  "parent_id",
			"createdAt": "created_at",
			"updatedAt": "updated_at",
		},
	},
}

// Save 保存角色
func (c *Role) Save(data *Role) error {
	err := c.DB().Save(data).Error
	if err == nil {
		// 角色变更时刷新缓存
		RoleTreeCache.Refresh()
	}
	return err
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

// GetByParentID 根据父ID获取子角色列表
func (c *Role) GetByParentID(parentID int64) ([]Role, error) {
	var roles []Role
	err := c.DB().Where("parent_id = ?", parentID).Order("sort, id").Find(&roles).Error
	return roles, err
}

// ================== 角色树缓存 ==================

// RoleTreeCacheInstance 角色树缓存实例
type RoleTreeCacheInstance struct {
	mu           sync.RWMutex
	tree         []*Role           // 完整树结构
	roleMap      map[int64]*Role   // ID -> Role 映射
	childrenMap  map[int64][]int64 // 父ID -> 子ID列表
	descendantMap map[int64][]int64 // ID -> 所有后代ID（递归）
	initialized  bool
}

// RoleTreeCache 全局角色树缓存
var RoleTreeCache = &RoleTreeCacheInstance{
	roleMap:      make(map[int64]*Role),
	childrenMap:  make(map[int64][]int64),
	descendantMap: make(map[int64][]int64),
}

// Refresh 刷新缓存
func (c *RoleTreeCacheInstance) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 获取所有角色
	var roles []Role
	if err := Roles.DB().Order("sort, id").Find(&roles).Error; err != nil {
		return err
	}

	// 重建映射
	c.roleMap = make(map[int64]*Role, len(roles))
	c.childrenMap = make(map[int64][]int64)
	c.descendantMap = make(map[int64][]int64)

	for i := range roles {
		role := &roles[i]
		c.roleMap[role.ID] = role
		c.childrenMap[role.ParentID] = append(c.childrenMap[role.ParentID], role.ID)
	}

	// 构建树
	c.tree = c.buildTree(roles, 0)

	// 预计算所有后代
	for id := range c.roleMap {
		c.descendantMap[id] = c.collectDescendants(id)
	}

	c.initialized = true
	return nil
}

// buildTree 构建树结构
func (c *RoleTreeCacheInstance) buildTree(roles []Role, parentID int64) []*Role {
	var tree []*Role
	for i := range roles {
		if roles[i].ParentID == parentID {
			role := &roles[i]
			role.Children = c.buildTree(roles, role.ID)
			tree = append(tree, role)
		}
	}
	return tree
}

// collectDescendants 递归收集所有后代ID
func (c *RoleTreeCacheInstance) collectDescendants(roleID int64) []int64 {
	var descendants []int64
	children := c.childrenMap[roleID]
	for _, childID := range children {
		descendants = append(descendants, childID)
		descendants = append(descendants, c.collectDescendants(childID)...)
	}
	return descendants
}

// EnsureInitialized 确保缓存已初始化
func (c *RoleTreeCacheInstance) EnsureInitialized() error {
	c.mu.RLock()
	initialized := c.initialized
	c.mu.RUnlock()

	if !initialized {
		return c.Refresh()
	}
	return nil
}

// GetTree 获取完整角色树
func (c *RoleTreeCacheInstance) GetTree() ([]*Role, error) {
	if err := c.EnsureInitialized(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree, nil
}

// GetRole 根据ID获取角色
func (c *RoleTreeCacheInstance) GetRole(id int64) (*Role, error) {
	if err := c.EnsureInitialized(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roleMap[id], nil
}

// GetChildren 获取直接子角色ID列表
func (c *RoleTreeCacheInstance) GetChildren(roleID int64) ([]int64, error) {
	if err := c.EnsureInitialized(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.childrenMap[roleID], nil
}

// GetDescendants 获取所有后代角色ID列表（用于权限聚合）
func (c *RoleTreeCacheInstance) GetDescendants(roleID int64) ([]int64, error) {
	if err := c.EnsureInitialized(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.descendantMap[roleID], nil
}

// GetRoleAndDescendantIDs 获取角色自身及所有后代ID（用于权限查询）
func (c *RoleTreeCacheInstance) GetRoleAndDescendantIDs(roleID int64) ([]int64, error) {
	descendants, err := c.GetDescendants(roleID)
	if err != nil {
		return nil, err
	}
	return append([]int64{roleID}, descendants...), nil
}
