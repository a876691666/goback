package permissionscope

import (
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/rbac/internal/common"
	"github.com/goback/services/rbac/internal/model"
)

// Create 创建数据过滤规则
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	// 验证权限存在
	perm, err := model.Permissions.GetOne(req.PermissionID)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if perm == nil {
		return apis.Error(e, 404, "权限不存在")
	}

	scope := &model.PermissionScope{
		PermissionID:   req.PermissionID,
		Name:           req.Name,
		ScopeTableName: req.TableName,
		SSQLRule:       req.SSQLRule,
		Description:    req.Description,
	}
	if err := model.PermissionScopes.Create(scope); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	e.App.GetBroadcaster().SendJSON(core.KeyRBACData, common.LoadRBACData(), "")
	return apis.Success(e, scope)
}

// Update 更新数据过滤规则
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的ID")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	scope, err := model.PermissionScopes.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if scope == nil {
		return apis.Error(e, 404, "数据过滤规则不存在")
	}

	if req.Name != "" {
		scope.Name = req.Name
	}
	if req.TableName != "" {
		scope.ScopeTableName = req.TableName
	}
	if req.SSQLRule != "" {
		scope.SSQLRule = req.SSQLRule
	}
	if req.Description != "" {
		scope.Description = req.Description
	}
	if err := model.PermissionScopes.Save(scope); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	e.App.GetBroadcaster().SendJSON(core.KeyRBACData, common.LoadRBACData(), "")
	return apis.Success(e, scope)
}

// Delete 删除数据过滤规则
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的ID")
	}
	if err := model.PermissionScopes.DeleteByID(id); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	e.App.GetBroadcaster().SendJSON(core.KeyRBACData, common.LoadRBACData(), "")
	return apis.Success(e, nil)
}

// Get 获取数据过滤规则详情
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的ID")
	}
	scope, err := model.PermissionScopes.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if scope == nil {
		return apis.Error(e, 404, "数据过滤规则不存在")
	}
	return apis.Success(e, scope)
}

// List 数据过滤规则列表
func List(e *core.RequestEvent) error {
	params, err := dal.BindQueryFromRequest(e.Request)
	if err != nil {
		return apis.Error(e, 400, err.Error())
	}
	result, err := model.PermissionScopes.GetList(params)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Paged(e, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// GetByPermission 根据权限获取数据过滤规则
func GetByPermission(e *core.RequestEvent) error {
	permissionID, err := strconv.ParseInt(e.Request.PathValue("permissionId"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的权限ID")
	}
	scopes, err := model.PermissionScopes.GetByPermissionID(permissionID)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, scopes)
}

// ================== 导出函数（供其他服务调用） ==================

// GetByPermissionID 根据权限ID获取数据过滤规则
func GetByPermissionID(permissionID int64) ([]model.PermissionScope, error) {
	return model.PermissionScopes.GetByPermissionID(permissionID)
}

// GetByPermissionIDs 根据多个权限ID获取数据过滤规则
func GetByPermissionIDs(permissionIDs []int64) ([]model.PermissionScope, error) {
	return model.PermissionScopes.GetByPermissionIDs(permissionIDs)
}

// GetByTableName 根据表名和权限ID列表获取数据过滤规则
func GetByTableName(tableName string, permissionIDs []int64) ([]model.PermissionScope, error) {
	return model.PermissionScopes.GetByScopeTableName(tableName, permissionIDs)
}

// DeleteByPermissionID 根据权限ID删除数据过滤规则
func DeleteByPermissionID(permissionID int64) error {
	return model.PermissionScopes.DeleteByPermissionID(permissionID)
}
