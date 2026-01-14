package permissionscope

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/rbac/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 数据过滤规则控制器
type Controller struct {
	router.BaseController
}

// Prefix 返回路由前缀
func (c *Controller) Prefix() string {
	return "/permission-scopes"
}

// Routes 返回路由配置
func (c *Controller) Routes(middlewares map[string]fiber.Handler) []router.Route {
	return []router.Route{
		{Method: "POST", Path: "", Handler: c.create, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "PUT", Path: "/:id", Handler: c.update, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "DELETE", Path: "/:id", Handler: c.delete, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/:id", Handler: c.get, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "", Handler: c.list, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/by-permission/:permissionId", Handler: c.getByPermission, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
	}
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	scope, err := c.doCreate(&req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, scope)
}

func (c *Controller) doCreate(req *CreateRequest) (*model.PermissionScope, error) {
	// 验证权限存在
	perm, err := model.Permissions.GetOne(req.PermissionID)
	if err != nil {
		return nil, err
	}
	if perm == nil {
		return nil, errors.NotFound("权限")
	}
	scope := &model.PermissionScope{
		PermissionID:   req.PermissionID,
		Name:           req.Name,
		ScopeTableName: req.TableName,
		SSQLRule:       req.SSQLRule,
		Description:    req.Description,
	}
	if err := model.PermissionScopes.Create(scope); err != nil {
		return nil, err
	}
	return scope, nil
}

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的ID")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	scope, err := c.doUpdate(id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, scope)
}

func (c *Controller) doUpdate(id int64, req *UpdateRequest) (*model.PermissionScope, error) {
	scope, err := model.PermissionScopes.GetOne(id)
	if err != nil {
		return nil, err
	}
	if scope == nil {
		return nil, errors.NotFound("数据过滤规则")
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
		return nil, err
	}
	return scope, nil
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的ID")
	}
	if err := model.PermissionScopes.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的ID")
	}
	scope, err := model.PermissionScopes.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if scope == nil {
		return response.NotFound(ctx, "数据过滤规则不存在")
	}
	return response.Success(ctx, scope)
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	result, err := model.PermissionScopes.GetList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

func (c *Controller) getByPermission(ctx *fiber.Ctx) error {
	permissionID, err := dal.ParseInt64ID(ctx.Params("permissionId"))
	if err != nil {
		return response.BadRequest(ctx, "无效的权限ID")
	}
	scopes, err := model.PermissionScopes.GetByPermissionID(permissionID)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, scopes)
}

// GetByPermissionID 根据权限ID获取数据过滤规则
func (c *Controller) GetByPermissionID(permissionID int64) ([]model.PermissionScope, error) {
	return model.PermissionScopes.GetByPermissionID(permissionID)
}

// GetByPermissionIDs 根据多个权限ID获取数据过滤规则
func (c *Controller) GetByPermissionIDs(permissionIDs []int64) ([]model.PermissionScope, error) {
	return model.PermissionScopes.GetByPermissionIDs(permissionIDs)
}

// GetByTableName 根据表名和权限ID列表获取数据过滤规则
func (c *Controller) GetByTableName(tableName string, permissionIDs []int64) ([]model.PermissionScope, error) {
	return model.PermissionScopes.GetByScopeTableName(tableName, permissionIDs)
}

// DeleteByPermissionID 根据权限ID删除数据过滤规则
func (c *Controller) DeleteByPermissionID(permissionID int64) error {
	return model.PermissionScopes.DeleteByPermissionID(permissionID)
}
