package role

import (
	"fmt"

	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/rbac/internal/model"
	"github.com/goback/services/rbac/internal/permission"
	"github.com/gofiber/fiber/v2"
)

// Controller 角色控制器
type Controller struct {
	PermCtrl      *permission.Controller
	CasbinService *auth.CasbinService
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
		{Method: "GET", Path: "/:id/permissions", Handler: c.getPermissions, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "PUT", Path: "/:id/permissions", Handler: c.setPermissions, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/:id/datascope", Handler: c.getDataScope, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "PUT", Path: "/:id/datascope", Handler: c.setDataScope, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
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
	role := &model.Role{
		Name:        req.Name,
		Code:        req.Code,
		DataScope:   req.DataScope,
		Status:      req.Status,
		Sort:        req.Sort,
		Description: req.Description,
	}
	if role.DataScope == 0 {
		role.DataScope = 1
	}
	if role.Status == 0 {
		role.Status = 1
	}
	if err := model.Roles.Create(role); err != nil {
		return nil, err
	}
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
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.DataScope > 0 {
		role.DataScope = req.DataScope
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
	model.RoleDataScopes.DeleteByRoleID(id)
	c.PermCtrl.DeleteRolePermissions(id)
	if err := model.Roles.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
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
	if err := c.PermCtrl.SetRolePermissions(roleID, permissionIDs); err != nil {
		return err
	}
	permissions, err := c.PermCtrl.GetByRoleID(roleID)
	if err != nil {
		return err
	}
	casbinPerms := make([]auth.Permission, len(permissions))
	for i, p := range permissions {
		casbinPerms[i] = auth.Permission{Resource: p.Resource, Action: p.Action}
	}
	return c.CasbinService.SetRolePermissions(role.Code, casbinPerms)
}

func (c *Controller) getDataScope(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	dataScopes, err := model.RoleDataScopes.GetFullList(&dal.ListParams{
		Filter: fmt.Sprintf("role_id=%d", id),
	})
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	deptIDs := make([]int64, len(dataScopes))
	for i, ds := range dataScopes {
		deptIDs[i] = ds.DeptID
	}
	return response.Success(ctx, deptIDs)
}

func (c *Controller) setDataScope(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	var req SetDataScopeRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	if err := model.RoleDataScopes.DeleteByRoleID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if len(req.DeptIDs) > 0 {
		rdss := make([]model.RoleDataScope, len(req.DeptIDs))
		for i, deptID := range req.DeptIDs {
			rdss[i] = model.RoleDataScope{RoleID: id, DeptID: deptID}
		}
		if err := model.RoleDataScopes.CreateBatch(rdss); err != nil {
			return response.Error(ctx, 500, err.Error())
		}
	}
	return response.Success(ctx, nil)
}

// GetByID 根据ID获取角色
func (c *Controller) GetByID(id int64) (*model.Role, error) {
	return model.Roles.GetOne(id)
}

// GetUserDataScope 获取用户数据权限范围
func (c *Controller) GetUserDataScope(userID, roleID, deptID int64) (*auth.DataScopeInfo, error) {
	role, err := model.Roles.GetOne(roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.NotFound("角色")
	}
	info := auth.NewDataScopeInfo(auth.DataScopeType(role.DataScope), userID, deptID)
	if role.DataScope == int8(auth.DataScopeCustom) {
		dataScopes, err := model.RoleDataScopes.GetFullList(&dal.ListParams{
			Filter: fmt.Sprintf("role_id=%d", roleID),
		})
		if err != nil {
			return nil, err
		}
		deptIDs := make([]int64, len(dataScopes))
		for i, ds := range dataScopes {
			deptIDs[i] = ds.DeptID
		}
		info.DeptIDs = deptIDs
	}
	return info, nil
}
