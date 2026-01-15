package permission

import (
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/rbac/internal/common"
	"github.com/goback/services/rbac/internal/model"
)

// Create 创建权限
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	exists, err := model.Permissions.ExistsByCode(req.Code)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if exists {
		return apis.Error(e, 409, "权限编码已存在")
	}

	perm := &model.Permission{
		Name:        req.Name,
		Code:        req.Code,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}
	if err := model.Permissions.Create(perm); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	e.App.GetBroadcaster().SendJSON(core.KeyRBACData, common.LoadRBACData(), "")
	return apis.Success(e, perm)
}

// Update 更新权限
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的权限ID")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	perm, err := model.Permissions.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if perm == nil {
		return apis.Error(e, 404, "权限不存在")
	}

	if req.Name != "" {
		perm.Name = req.Name
	}
	if req.Resource != "" {
		perm.Resource = req.Resource
	}
	if req.Action != "" {
		perm.Action = req.Action
	}
	if req.Description != "" {
		perm.Description = req.Description
	}
	if err := model.Permissions.Save(perm); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	e.App.GetBroadcaster().SendJSON(core.KeyRBACData, common.LoadRBACData(), "")
	return apis.Success(e, perm)
}

// Delete 删除权限
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的权限ID")
	}
	if err := model.Permissions.DeleteByID(id); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	e.App.GetBroadcaster().SendJSON(core.KeyRBACData, common.LoadRBACData(), "")
	return apis.Success(e, nil)
}

// Get 获取权限详情
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的权限ID")
	}
	perm, err := model.Permissions.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if perm == nil {
		return apis.Error(e, 404, "权限不存在")
	}
	return apis.Success(e, perm)
}

// List 权限列表
func List(e *core.RequestEvent) error {
	params, err := dal.BindQueryFromRequest(e.Request)
	if err != nil {
		return apis.Error(e, 400, err.Error())
	}
	result, err := model.Permissions.GetList(params)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Paged(e, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// GetAll 获取所有权限
func GetAll(e *core.RequestEvent) error {
	permissions, err := model.Permissions.GetFullList(&dal.ListParams{
		Sort: "id",
	})
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, permissions)
}

// ================== 导出函数（供其他服务调用） ==================

// GetByRoleID 根据角色ID获取权限列表
func GetByRoleID(roleID int64) ([]model.Permission, error) {
	return model.Permissions.GetByRoleID(roleID)
}

// SetRolePermissions 设置角色权限
func SetRolePermissions(roleID int64, permissionIDs []int64) error {
	if err := model.RolePermissions.DeleteByRoleID(roleID); err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}
	rps := make([]model.RolePermission, len(permissionIDs))
	for i, permID := range permissionIDs {
		rps[i] = model.RolePermission{
			RoleID:       roleID,
			PermissionID: permID,
		}
	}
	return model.RolePermissions.CreateBatch(rps)
}

// DeleteRolePermissions 删除角色权限
func DeleteRolePermissions(roleID int64) error {
	return model.RolePermissions.DeleteByRoleID(roleID)
}
