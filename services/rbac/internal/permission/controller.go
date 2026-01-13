package permission

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/rbac/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 权限控制器
type Controller struct {
	repo         Repository
	rolePermRepo RolePermissionRepository
	collection   *dal.Collection[model.Permission]
}

// NewController 创建权限控制器
func NewController(repo Repository, rolePermRepo RolePermissionRepository) *Controller {
	return &Controller{
		repo:         repo,
		rolePermRepo: rolePermRepo,
		collection: dal.NewCollection[model.Permission](repo.DB()).
			WithDefaultSort("-id").
			WithMaxPerPage(500).
			WithFieldAlias(map[string]string{
				"createdAt": "created_at",
				"updatedAt": "updated_at",
			}),
	}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	g := r.Group("/permissions", jwtMiddleware)
	g.Post("", c.create)
	g.Put("/:id", c.update)
	g.Delete("/:id", c.delete)
	g.Get("/:id", c.get)
	g.Get("", c.list)
	g.Get("/all", c.getAll)
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	perm, err := c.doCreate(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, perm)
}

func (c *Controller) doCreate(ctx context.Context, req *CreateRequest) (*model.Permission, error) {
	existing, err := c.repo.FindByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Duplicate("权限编码")
	}
	perm := &model.Permission{
		Name:        req.Name,
		Code:        req.Code,
		Type:        req.Type,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}
	if perm.Type == 0 {
		perm.Type = 1
	}
	if err := c.repo.Create(ctx, perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的权限ID")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	perm, err := c.doUpdate(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, perm)
}

func (c *Controller) doUpdate(ctx context.Context, id int64, req *UpdateRequest) (*model.Permission, error) {
	perm, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if perm == nil {
		return nil, errors.NotFound("权限")
	}
	if req.Name != "" {
		perm.Name = req.Name
	}
	if req.Type > 0 {
		perm.Type = req.Type
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
	if err := c.repo.Update(ctx, perm); err != nil {
		return nil, err
	}
	return perm, nil
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的权限ID")
	}
	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的权限ID")
	}
	perm, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if perm == nil {
		return response.NotFound(ctx, "权限不存在")
	}
	return response.Success(ctx, perm)
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
	permissions, err := c.repo.FindAll(ctx.UserContext(), nil, dal.WithOrder("id ASC"))
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, permissions)
}

// GetByRoleID 根据角色ID获取权限列表
func (c *Controller) GetByRoleID(ctx context.Context, roleID int64) ([]model.Permission, error) {
	return c.repo.FindByRoleID(ctx, roleID)
}

// SetRolePermissions 设置角色权限
func (c *Controller) SetRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	if err := c.rolePermRepo.DeleteByRoleID(ctx, roleID); err != nil {
		return err
	}
	return c.rolePermRepo.BatchCreate(ctx, roleID, permissionIDs)
}

// DeleteRolePermissions 删除角色权限
func (c *Controller) DeleteRolePermissions(ctx context.Context, roleID int64) error {
	return c.rolePermRepo.DeleteByRoleID(ctx, roleID)
}
