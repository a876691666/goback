package lifecycle

import (
	"fmt"
	"sync"
)

// RBACCache RBAC缓存管理器 - 自动订阅并解析RBAC数据
// 提供线程安全的数据访问接口
type RBACCache struct {
	data RBACData
	mu   sync.RWMutex

	// 预计算的索引
	roleMap    map[int64]*Role       // roleID -> Role
	permMap    map[int64]*Permission // permID -> Permission
	childRoles map[int64][]int64     // parentID -> childIDs
}

// NewRBACCache 创建RBAC缓存管理器
func NewRBACCache() *RBACCache {
	return &RBACCache{
		roleMap:    make(map[int64]*Role),
		permMap:    make(map[int64]*Permission),
		childRoles: make(map[int64][]int64),
	}
}

// Update 更新RBAC数据
func (rc *RBACCache) Update(data RBACData) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.data = data

	// 重建索引
	rc.roleMap = make(map[int64]*Role, len(data.Roles))
	for i := range data.Roles {
		rc.roleMap[data.Roles[i].ID] = &data.Roles[i]
	}

	rc.permMap = make(map[int64]*Permission, len(data.Permissions))
	for i := range data.Permissions {
		rc.permMap[data.Permissions[i].ID] = &data.Permissions[i]
	}

	rc.childRoles = make(map[int64][]int64)
	for _, role := range data.Roles {
		rc.childRoles[role.ParentID] = append(rc.childRoles[role.ParentID], role.ID)
	}
}

// GetRole 获取角色
func (rc *RBACCache) GetRole(roleID int64) (*Role, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	role, ok := rc.roleMap[roleID]
	return role, ok
}

// GetPermission 获取权限
func (rc *RBACCache) GetPermission(permID int64) (*Permission, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	perm, ok := rc.permMap[permID]
	return perm, ok
}

// GetRoleAndDescendantIDs 获取角色及其所有启用的后代角色ID
// 只返回Status=1的角色
func (rc *RBACCache) GetRoleAndDescendantIDs(roleID int64) ([]int64, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	role, ok := rc.roleMap[roleID]
	if !ok {
		return nil, fmt.Errorf("角色不存在: %d", roleID)
	}
	if role.Status != 1 {
		return nil, fmt.Errorf("角色已被禁用: %d", roleID)
	}

	result := []int64{roleID}
	collected := make(map[int64]bool)
	collected[roleID] = true

	// BFS遍历所有后代
	queue := []int64{roleID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		// 查找所有子角色
		for _, childID := range rc.childRoles[currentID] {
			if collected[childID] {
				continue
			}
			collected[childID] = true

			child, ok := rc.roleMap[childID]
			if ok && child.Status == 1 {
				result = append(result, childID)
				queue = append(queue, childID)
			}
		}
	}

	return result, nil
}

// GetAggregatedPermissions 聚合指定角色列表的所有权限（去重）
func (rc *RBACCache) GetAggregatedPermissions(roleIDs []int64) map[int64]Permission {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	permissionMap := make(map[int64]Permission)
	for _, rid := range roleIDs {
		if perms, ok := rc.data.RolePermissions[rid]; ok {
			for _, perm := range perms {
				permissionMap[perm.ID] = perm
			}
		}
	}
	return permissionMap
}

// GetPermissionScopes 获取指定权限ID列表和表名的数据范围规则
func (rc *RBACCache) GetPermissionScopes(permissionIDs []int64, tableName string) []PermissionScope {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	permIDSet := make(map[int64]bool, len(permissionIDs))
	for _, id := range permissionIDs {
		permIDSet[id] = true
	}

	var result []PermissionScope
	for _, scope := range rc.data.PermissionScopes {
		if permIDSet[scope.PermissionID] && scope.ScopeTableName == tableName {
			result = append(result, scope)
		}
	}
	return result
}

// GetAllPermissionScopes 获取所有权限数据范围
func (rc *RBACCache) GetAllPermissionScopes() []PermissionScope {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.PermissionScopes
}

// GetAllRoles 获取所有角色
func (rc *RBACCache) GetAllRoles() []Role {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.Roles
}

// GetAllPermissions 获取所有权限
func (rc *RBACCache) GetAllPermissions() []Permission {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.Permissions
}

// GetRolePermissions 获取角色权限映射
func (rc *RBACCache) GetRolePermissions() RolePermissionMap {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.data.RolePermissions
}

// IsReady 检查缓存是否已加载数据
func (rc *RBACCache) IsReady() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.data.Roles) > 0 || len(rc.data.Permissions) > 0
}
