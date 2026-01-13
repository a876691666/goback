package role

import (
	"context"

	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/rbac/internal/model"
	"github.com/goback/services/rbac/internal/permission"
	"github.com/gofiber/fiber/v2"
)

// Controller 角色控制器
type Controller struct {
	repo          Repository
	dataScopeRepo DataScopeRepository
	permCtrl      *permission.Controller
	casbinService *auth.CasbinService
	collection    *dal.Collection[model.Role]
}

// NewController 创建角色控制器
func NewController(repo Repository, dataScopeRepo DataScopeRepository, permCtrl *permission.Controller) *Controller {
	return &Controller{
		repo:          repo,
		dataScopeRepo: dataScopeRepo,
		permCtrl:      permCtrl,
		casbinService: auth.NewCasbinService(),
		collection: dal.NewCollection[model.Role](repo.DB()).
			WithDefaultSort("sort,-id").
			WithMaxPerPage(100).
			WithFieldAlias(map[string]string{
				"createdAt": "created_at",
				"updatedAt": "updated_at",
				"dataScope": "data_scope",
			}),
	}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	g := r.Group("/roles", jwtMiddleware)
	g.Post("", c.create)
	g.Put("/:id", c.update)
	g.Delete("/:id", c.delete)
	g.Get("/:id", c.get)
	g.Get("", c.list)
	g.Get("/all", c.getAll)
	g.Get("/:id/permissions", c.getPermissions)
	g.Put("/:id/permissions", c.setPermissions)
	g.Get("/:id/datascope", c.getDataScope)
	g.Put("/:id/datascope", c.setDataScope)
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	role, err := c.doCreate(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, role)
}

func (c *Controller) doCreate(ctx context.Context, req *CreateRequest) (*model.Role, error) {
	existing, err := c.repo.FindByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
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
	if err := c.repo.Create(ctx, role); err != nil {
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
	role, err := c.doUpdate(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, role)
}

func (c *Controller) doUpdate(ctx context.Context, id int64, req *UpdateRequest) (*model.Role, error) {
	role, err := c.repo.FindByID(ctx, id)
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
	if err := c.repo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	c.dataScopeRepo.DeleteByRoleID(ctx.UserContext(), id)
	c.permCtrl.DeleteRolePermissions(ctx.UserContext(), id)
	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	role, err := c.repo.FindByID(ctx.UserContext(), id)
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
	result, err := c.collection.GetList(ctx.UserContext(), params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

func (c *Controller) getAll(ctx *fiber.Ctx) error {
	roles, err := c.repo.FindAll(ctx.UserContext(), map[string]interface{}{"status": 1}, dal.WithOrder("sort ASC"))
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
	permissions, err := c.permCtrl.GetByRoleID(ctx.UserContext(), id)
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
	if err := c.doSetPermissions(ctx.UserContext(), id, req.PermissionIDs); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) doSetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	role, err := c.repo.FindByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.NotFound("角色")
	}
	if err := c.permCtrl.SetRolePermissions(ctx, roleID, permissionIDs); err != nil {
		return err
	}
	permissions, err := c.permCtrl.GetByRoleID(ctx, roleID)
	if err != nil {
		return err
	}
	casbinPerms := make([]auth.Permission, len(permissions))
	for i, p := range permissions {
		casbinPerms[i] = auth.Permission{Resource: p.Resource, Action: p.Action}
	}
	return c.casbinService.SetRolePermissions(role.Code, casbinPerms)
}

func (c *Controller) getDataScope(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的角色ID")
	}
	dataScopes, err := c.dataScopeRepo.FindByRoleID(ctx.UserContext(), id)
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
	if err := c.dataScopeRepo.DeleteByRoleID(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if err := c.dataScopeRepo.BatchCreate(ctx.UserContext(), id, req.DeptIDs); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// GetByID 根据ID获取角色
func (c *Controller) GetByID(ctx context.Context, id int64) (*model.Role, error) {
	return c.repo.FindByID(ctx, id)
}

// GetUserDataScope 获取用户数据权限范围
func (c *Controller) GetUserDataScope(ctx context.Context, userID, roleID, deptID int64) (*auth.DataScopeInfo, error) {
	role, err := c.repo.FindByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.NotFound("角色")
	}
	info := auth.NewDataScopeInfo(auth.DataScopeType(role.DataScope), userID, deptID)
	if role.DataScope == int8(auth.DataScopeCustom) {
		dataScopes, err := c.dataScopeRepo.FindByRoleID(ctx, roleID)
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
