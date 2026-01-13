package permission

import (
	"context"
	"strconv"

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
}

// NewController 创建权限控制器
func NewController(repo Repository, rolePermRepo RolePermissionRepository) *Controller {
	return &Controller{
		repo:         repo,
		rolePermRepo: rolePermRepo,
	}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	permissions := r.Group("/permissions", jwtMiddleware)
	permissions.Post("", c.Create)
	permissions.Put("/:id", c.Update)
	permissions.Delete("/:id", c.Delete)
	permissions.Get("/:id", c.Get)
	permissions.Get("", c.List)
	permissions.Get("/all", c.GetAll)
}

// Create 创建权限
// @Summary 创建权限
// @Tags 权限管理
// @Accept json
// @Produce json
// @Param request body CreateRequest true "创建权限请求"
// @Success 200 {object} response.Response
// @Router /permissions [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	perm, err := c.create(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, perm)
}

// create 创建权限业务逻辑
func (c *Controller) create(ctx context.Context, req *CreateRequest) (*model.Permission, error) {
	existing, err := c.repo.FindByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Duplicate("permission code")
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

// Update 更新权限
// @Summary 更新权限
// @Tags 权限管理
// @Accept json
// @Produce json
// @Param id path int true "权限ID"
// @Param request body UpdateRequest true "更新权限请求"
// @Success 200 {object} response.Response
// @Router /permissions/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid permission id")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	perm, err := c.update(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, perm)
}

// update 更新权限业务逻辑
func (c *Controller) update(ctx context.Context, id int64, req *UpdateRequest) (*model.Permission, error) {
	perm, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if perm == nil {
		return nil, errors.NotFound("permission")
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

// Delete 删除权限
// @Summary 删除权限
// @Tags 权限管理
// @Param id path int true "权限ID"
// @Success 200 {object} response.Response
// @Router /permissions/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid permission id")
	}

	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Get 获取权限
// @Summary 获取权限详情
// @Tags 权限管理
// @Param id path int true "权限ID"
// @Success 200 {object} response.Response
// @Router /permissions/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid permission id")
	}

	perm, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if perm == nil {
		return response.NotFound(ctx, "permission not found")
	}

	return response.Success(ctx, perm)
}

// List 权限列表
// @Summary 权限列表
// @Tags 权限管理
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param name query string false "名称"
// @Param code query string false "编码"
// @Param type query int false "类型"
// @Success 200 {object} response.Response
// @Router /permissions [get]
func (c *Controller) List(ctx *fiber.Ctx) error {
	var req ListRequest
	if err := ctx.QueryParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	result, err := c.list(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.SuccessPage(ctx, result.List, result.Total, result.Page, result.PageSize)
}

// list 权限列表业务逻辑
func (c *Controller) list(ctx context.Context, req *ListRequest) (*dal.PagedResult[model.Permission], error) {
	pagination := dal.NewPagination(req.Page, req.PageSize)
	qb := dal.NewQueryBuilder[model.Permission](c.repo.DB())

	if req.Name != "" {
		qb.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Code != "" {
		qb.Where("code LIKE ?", "%"+req.Code+"%")
	}
	if req.Type != nil {
		qb.Where("type = ?", *req.Type)
	}

	qb.Order("id DESC")

	return qb.Paged(ctx, pagination)
}

// GetAll 获取所有权限
// @Summary 获取所有权限
// @Tags 权限管理
// @Success 200 {object} response.Response
// @Router /permissions/all [get]
func (c *Controller) GetAll(ctx *fiber.Ctx) error {
	permissions, err := c.repo.FindAll(ctx.UserContext(), nil, dal.WithOrder("id ASC"))
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, permissions)
}

// GetByRoleID 根据角色ID获取权限（供内部调用）
func (c *Controller) GetByRoleID(ctx context.Context, roleID int64) ([]model.Permission, error) {
	return c.repo.FindByRoleID(ctx, roleID)
}

// SetRolePermissions 设置角色权限（供内部调用）
func (c *Controller) SetRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	// 删除现有权限关联
	if err := c.rolePermRepo.DeleteByRoleID(ctx, roleID); err != nil {
		return err
	}

	// 批量创建新关联
	return c.rolePermRepo.BatchCreate(ctx, roleID, permissionIDs)
}

// DeleteRolePermissions 删除角色权限（供内部调用）
func (c *Controller) DeleteRolePermissions(ctx context.Context, roleID int64) error {
	return c.rolePermRepo.DeleteByRoleID(ctx, roleID)
}
