package role

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/lifecycle"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/rbac/internal/common"
	"github.com/goback/services/rbac/internal/model"
	"github.com/goback/services/rbac/internal/permission"
	"github.com/gofiber/fiber/v2"
)

// Controller 角色控制器
type Controller struct {
	router.BaseController
	PermCtrl *permission.Controller
}

// Prefix 返回路由前缀
func (c *Controller) Prefix() string {
	return "/roles"
}

// Routes 返回路由配置
func (c *Controller) Routes(middlewares map[string]fiber.Handler) []router.Route {
	return []router.Route{
		{Method: "POST", Path: "", Handler: c.create, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "PUT", Path: "/:id", Handler: c.update, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "DELETE", Path: "/:id", Handler: c.delete, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/:id", Handler: c.get, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "", Handler: c.list, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/all", Handler: c.getAll, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/tree", Handler: c.getTree, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/:id/permissions", Handler: c.getPermissions, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "PUT", Path: "/:id/permissions", Handler: c.setPermissions, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/:id/all-permissions", Handler: c.getAllPermissions, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "POST", Path: "/cache/refresh", Handler: c.refreshCache, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
	}
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	role, err := c.doCreate(&req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	c.Service().Broadcaster().SendJSON(lifecycle.KeyRBACData, common.LoadRBACData(), "")
	return response.Success(ctx, role)
}

func (c *Controller) doCreate(req *CreateRequest) (*model.Role, error) {
	exists, err := model.Roles.ExistsByCode(req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.Duplicate("角色编码")
	}
	// 验证父角色存在
	if req.ParentID > 0 {
		parent, err := model.Roles.GetOne(req.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, errors.NotFound("父角色")
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
		return nil, err
	}
	// 刷新缓存
	model.RoleTreeCache.Refresh()
	return role, nil
}

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	role, err := c.doUpdate(id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	c.Service().Broadcaster().SendJSON(lifecycle.KeyRBACData, common.LoadRBACData(), "")
	return response.Success(ctx, role)
}

func (c *Controller) doUpdate(id int64, req *UpdateRequest) (*model.Role, error) {
	role, err := model.Roles.GetOne(id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.NotFound("角色")
	}
	// 验证父角色（不能设置为自己或自己的后代）
	if req.ParentID > 0 && req.ParentID != role.ParentID {
		if req.ParentID == id {
			return nil, errors.BadRequest("不能将自己设为父角色")
		}
		descendants, _ := model.RoleTreeCache.GetDescendants(id)
		for _, descID := range descendants {
			if descID == req.ParentID {
				return nil, errors.BadRequest("不能将后代角色设为父角色")
			}
		}
		parent, err := model.Roles.GetOne(req.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, errors.NotFound("父角色")
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
		return nil, err
	}
	return role, nil
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	// 检查是否有子角色
	children, err := model.RoleTreeCache.GetChildren(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if len(children) > 0 {
		return response.BadRequest(ctx, "存在子角色，无法删除")
	}
	c.PermCtrl.DeleteRolePermissions(id)
	if err := model.Roles.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	// 刷新缓存
	model.RoleTreeCache.Refresh()
	c.Service().Broadcaster().SendJSON(lifecycle.KeyRBACData, common.LoadRBACData(), "")
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	role, err := model.Roles.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if role == nil {
		return response.NotFound(ctx, "角色不存在")
	}
	return response.Success(ctx, role)
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	result, err := model.Roles.GetList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

func (c *Controller) getAll(ctx *fiber.Ctx) error {
	roles, err := model.Roles.GetFullList(&dal.ListParams{
		Filter: "status=1",
		Sort:   "sort",
	})
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, roles)
}

func (c *Controller) getTree(ctx *fiber.Ctx) error {
	tree, err := model.RoleTreeCache.GetTree()
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, tree)
}

func (c *Controller) getPermissions(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	permissions, err := c.PermCtrl.GetByRoleID(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, permissions)
}

// getAllPermissions 获取角色及其所有子角色的权限（聚合）
func (c *Controller) getAllPermissions(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	// 获取角色及所有后代ID
	roleIDs, err := model.RoleTreeCache.GetRoleAndDescendantIDs(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	// 获取所有权限
	permissions, err := model.Permissions.GetByRoleIDs(roleIDs)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, permissions)
}

func (c *Controller) setPermissions(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	var req SetPermissionsRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	if err := c.doSetPermissions(id, req.PermissionIDs); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	c.Service().Broadcaster().SendJSON(lifecycle.KeyRBACData, common.LoadRBACData(), "")
	return response.Success(ctx, nil)
}

func (c *Controller) doSetPermissions(roleID int64, permissionIDs []int64) error {
	role, err := model.Roles.GetOne(roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.NotFound("角色")
	}
	return c.PermCtrl.SetRolePermissions(roleID, permissionIDs)
}

func (c *Controller) refreshCache(ctx *fiber.Ctx) error {
	if err := model.RoleTreeCache.Refresh(); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// GetByID 根据ID获取角色
func (c *Controller) GetByID(id int64) (*model.Role, error) {
	return model.Roles.GetOne(id)
}

// GetRoleAndDescendantIDs 获取角色及其所有后代ID（供其他服务调用）
func (c *Controller) GetRoleAndDescendantIDs(roleID int64) ([]int64, error) {
	return model.RoleTreeCache.GetRoleAndDescendantIDs(roleID)
}
