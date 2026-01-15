package common

import (
	"github.com/goback/pkg/app/core"
	"github.com/goback/services/rbac/internal/model"
)

// LoadPermissions 加载所有权限
func LoadPermissions() []core.Permission {
	var perms []model.Permission
	if err := model.Permissions.DB().Find(&perms).Error; err != nil {
		return nil
	}

	result := make([]core.Permission, len(perms))
	for i, p := range perms {
		result[i] = core.Permission{
			ID:       p.ID,
			Code:     p.Code,
			Name:     p.Name,
			Resource: p.Resource,
			Action:   p.Action,
		}
	}
	return result
}

// LoadRoles 加载所有角色
func LoadRoles() []core.Role {
	var roles []model.Role
	if err := model.Roles.DB().Find(&roles).Error; err != nil {
		return nil
	}

	result := make([]core.Role, len(roles))
	for i, r := range roles {
		result[i] = core.Role{
			ID:       r.ID,
			ParentID: r.ParentID,
			Code:     r.Code,
			Name:     r.Name,
			Status:   r.Status,
		}
	}
	return result
}

// LoadRolePermissions 加载角色权限映射
func LoadRolePermissions() core.RolePermissionMap {
	result := make(core.RolePermissionMap)

	var rolePerms []model.RolePermission
	model.RolePermissions.DB().Find(&rolePerms)

	// 按角色分组
	rolePermMap := make(map[int64][]int64)
	for _, rp := range rolePerms {
		rolePermMap[rp.RoleID] = append(rolePermMap[rp.RoleID], rp.PermissionID)
	}

	// 获取所有权限
	permMap := make(map[int64]core.Permission)
	for _, p := range LoadPermissions() {
		permMap[p.ID] = p
	}

	// 构建角色权限映射
	for roleID, permIDs := range rolePermMap {
		perms := make([]core.Permission, 0, len(permIDs))
		for _, permID := range permIDs {
			if p, ok := permMap[permID]; ok {
				perms = append(perms, p)
			}
		}
		result[roleID] = perms
	}

	return result
}

// LoadPermissionScopes 加载所有权限数据范围
func LoadPermissionScopes() []core.PermissionScope {
	var scopes []model.PermissionScope
	if err := model.PermissionScopes.DB().Find(&scopes).Error; err != nil {
		return nil
	}

	result := make([]core.PermissionScope, len(scopes))
	for i, s := range scopes {
		result[i] = core.PermissionScope{
			ID:             s.ID,
			PermissionID:   s.PermissionID,
			Name:           s.Name,
			ScopeTableName: s.ScopeTableName,
			SSQLRule:       s.SSQLRule,
			Description:    s.Description,
		}
	}
	return result
}

// LoadRBACData 加载完整的RBAC数据
func LoadRBACData() core.RBACData {
	return core.RBACData{
		Permissions:      LoadPermissions(),
		Roles:            LoadRoles(),
		RolePermissions:  LoadRolePermissions(),
		PermissionScopes: LoadPermissionScopes(),
	}
}
