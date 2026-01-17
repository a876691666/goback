package role

import (
	"fmt"
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/rbac/internal/common"
	"github.com/goback/services/rbac/internal/model"
	"github.com/goback/services/rbac/internal/permission"
)

// Create 创建角色
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	exists, err := model.Roles.ExistsByCode(req.Code)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if exists {
		return apis.Error(e, 409, "角色编码已存在")
	}

	// 验证父角色存在
	if req.ParentID > 0 {
		parent, err := model.Roles.GetOne(req.ParentID)
		if err != nil {
			return apis.Error(e, 500, err.Error())
		}
		if parent == nil {
			return apis.Error(e, 404, "父角色不存在")
		}
	}

	role := &model.Role{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Code:        req.Code,
		Status:      req.Status,
		Sort:        req.Sort,
		Description: req.Description,
	}
	if role.Status == 0 {
		role.Status = 1
	}
	if err := model.Roles.Create(role); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	// 刷新缓存
	model.RoleTreeCache.Refresh()
	e.App.PublishTopicJSON(core.KeyRBACData, common.LoadRBACData())
	return apis.Success(e, role)
}

// Update 更新角色
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	role, err := model.Roles.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if role == nil {
		return apis.Error(e, 404, "角色不存在")
	}

	// 验证父角色（不能设置为自己或自己的后代）
	if req.ParentID > 0 && req.ParentID != role.ParentID {
		if req.ParentID == id {
			return apis.Error(e, 400, "不能将自己设为父角色")
		}
		descendants, _ := model.RoleTreeCache.GetDescendants(id)
		for _, descID := range descendants {
			if descID == req.ParentID {
				return apis.Error(e, 400, "不能将后代角色设为父角色")
			}
		}
		parent, err := model.Roles.GetOne(req.ParentID)
		if err != nil {
			return apis.Error(e, 500, err.Error())
		}
		if parent == nil {
			return apis.Error(e, 404, "父角色不存在")
		}
		role.ParentID = req.ParentID
	} else if req.ParentID == 0 {
		role.ParentID = 0
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Status > 0 {
		role.Status = req.Status
	}
	if req.Sort > 0 {
		role.Sort = req.Sort
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if err := model.Roles.Save(role); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	e.App.PublishTopicJSON(core.KeyRBACData, common.LoadRBACData())
	return apis.Success(e, role)
}

// Delete 删除角色
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}

	// 检查是否有子角色
	children, err := model.RoleTreeCache.GetChildren(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if len(children) > 0 {
		return apis.Error(e, 400, "存在子角色，无法删除")
	}

	permission.DeleteRolePermissions(id)
	if err := model.Roles.DeleteByID(id); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	// 刷新缓存
	model.RoleTreeCache.Refresh()
	e.App.PublishTopicJSON(core.KeyRBACData, common.LoadRBACData())
	return apis.Success(e, nil)
}

// Get 获取角色详情
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	role, err := model.Roles.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if role == nil {
		return apis.Error(e, 404, "角色不存在")
	}
	return apis.Success(e, role)
}

// List 角色列表
func List(e *core.RequestEvent) error {
	params, err := dal.BindQueryFromRequest(e.Request)
	if err != nil {
		return apis.Error(e, 400, err.Error())
	}
	result, err := model.Roles.GetList(params)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Paged(e, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// GetAll 获取所有角色
func GetAll(e *core.RequestEvent) error {
	roles, err := model.Roles.GetFullList(&dal.ListParams{
		Filter: "status=1",
		Sort:   "sort",
	})
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, roles)
}

// GetTree 获取角色树
func GetTree(e *core.RequestEvent) error {
	tree, err := model.RoleTreeCache.GetTree()
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, tree)
}

// GetPermissions 获取角色权限
func GetPermissions(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	permissions, err := permission.GetByRoleID(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, permissions)
}

// GetAllPermissions 获取角色及其所有子角色的权限（聚合）
func GetAllPermissions(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	// 获取角色及所有后代ID
	roleIDs, err := model.RoleTreeCache.GetRoleAndDescendantIDs(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	// 获取所有权限
	permissions, err := model.Permissions.GetByRoleIDs(roleIDs)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, permissions)
}

// SetPermissions 设置角色权限
func SetPermissions(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	var req SetPermissionsRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	role, err := model.Roles.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if role == nil {
		return apis.Error(e, 404, "角色不存在")
	}

	if err := permission.SetRolePermissions(id, req.PermissionIDs); err != nil {
		return apis.Error(e, 500, err.Error())
	}

	e.App.PublishTopicJSON(core.KeyRBACData, common.LoadRBACData())
	return apis.Success(e, nil)
}

// RefreshCache 刷新缓存
func RefreshCache(e *core.RequestEvent) error {
	if err := model.RoleTreeCache.Refresh(); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, nil)
}

// ================== 导出函数（供其他服务调用） ==================

// GetByID 根据ID获取角色
func GetByID(id int64) (*model.Role, error) {
	return model.Roles.GetOne(id)
}

// GetRoleAndDescendantIDs 获取角色及其所有后代ID
func GetRoleAndDescendantIDs(roleID int64) ([]int64, error) {
	return model.RoleTreeCache.GetRoleAndDescendantIDs(roleID)
}

// ErrNotFound 角色不存在错误
var ErrNotFound = fmt.Errorf("角色不存在")
