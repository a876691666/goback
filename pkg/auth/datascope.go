package auth

import (
	"context"

	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/logger"
	"github.com/goback/pkg/ssql"
	"go.uber.org/zap"
)

// ========================= RBAC 数据权限函数 =========================
// 使用 core.RBACCache 作为统一的数据源

// BuildDataScopeSSQL 基于user->role->permission->permissionScope架构，从RBAC缓存生成SSQL对象
// 使用 RBACCache 作为数据源，自动订阅 RBAC 数据更新
func BuildDataScopeSSQL(
	ctx context.Context,
	rbacCache *core.RBACCache,
	userID int64,
	roleID int64,
	tableName string,
	resource string,
) (*ssql.Builder, error) {
	// 获取角色及其所有启用的后代角色ID
	roleIDs, err := rbacCache.GetRoleAndDescendantIDs(roleID)
	if err != nil {
		return nil, err
	}
	logger.Debug("收集角色ID", zap.Int64("roleID", roleID), zap.Int64s("allRoleIDs", roleIDs))

	// 聚合所有角色的权限
	permissionMap := rbacCache.GetAggregatedPermissions(roleIDs)

	// 筛选匹配资源的权限
	permissionIDs := make([]int64, 0)
	for _, perm := range permissionMap {
		if matchResource(perm.Resource, resource) {
			permissionIDs = append(permissionIDs, perm.ID)
		}
	}

	if len(permissionIDs) == 0 {
		logger.Warn("未找到匹配的权限", zap.String("resource", resource), zap.Int64("roleID", roleID))
		return ssql.NewBuilder().Eq("1", 0), nil
	}
	logger.Debug("匹配权限", zap.Int64s("permissionIDs", permissionIDs), zap.String("tableName", tableName))

	// 获取匹配的数据范围规则
	permissionScopes := rbacCache.GetPermissionScopes(permissionIDs, tableName)

	// 构建SSQL
	builder := ssql.NewBuilder().Or()
	hasRule := false

	for _, scope := range permissionScopes {
		if scope.SSQLRule == "" {
			continue
		}
		expr, err := ssql.Parse(scope.SSQLRule)
		if err != nil {
			logger.Error("解析SSQL规则失败", zap.Error(err), zap.String("rule", scope.SSQLRule), zap.Int64("scopeID", scope.ID))
			continue
		}
		builder.Expr(expr)
		hasRule = true
		logger.Debug("添加数据范围规则", zap.String("scopeName", scope.Name), zap.String("rule", scope.SSQLRule))
	}

	if !hasRule {
		logger.Debug("未找到数据范围规则，允许访问所有数据", zap.String("tableName", tableName), zap.Int64s("permissionIDs", permissionIDs))
		return ssql.NewBuilder(), nil
	}

	return builder, nil
}

// matchResource 匹配资源路径（支持通配符*）
func matchResource(pattern, resource string) bool {
	if pattern == resource {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(resource) >= len(prefix) && resource[:len(prefix)] == prefix
	}
	return false
}

// GetUserPermissions 获取用户的所有权限（聚合角色树，过滤禁用角色）
func GetUserPermissions(rbacCache *core.RBACCache, roleID int64) ([]core.Permission, error) {
	roleIDs, err := rbacCache.GetRoleAndDescendantIDs(roleID)
	if err != nil {
		return nil, err
	}

	permissionMap := rbacCache.GetAggregatedPermissions(roleIDs)

	permissions := make([]core.Permission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		permissions = append(permissions, perm)
	}
	return permissions, nil
}
