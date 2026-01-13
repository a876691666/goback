package role

import (
	"context"
	"strconv"

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
}

// NewController 创建角色控制器
func NewController(repo Repository, dataScopeRepo DataScopeRepository, permCtrl *permission.Controller) *Controller {
	return &Controller{
		repo:          repo,
		dataScopeRepo: dataScopeRepo,
		permCtrl:      permCtrl,
		casbinService: auth.NewCasbinService(),
	}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	roles := r.Group("/roles", jwtMiddleware)
	roles.Post("", c.Create)
	roles.Put("/:id", c.Update)
	roles.Delete("/:id", c.Delete)
	roles.Get("/:id", c.Get)
	roles.Get("", c.List)
	roles.Get("/all", c.GetAll)
	roles.Get("/:id/permissions", c.GetPermissions)
	roles.Put("/:id/permissions", c.SetPermissions)
	roles.Get("/:id/datascope", c.GetDataScope)
	roles.Put("/:id/datascope", c.SetDataScope)
}

// Create 创建角色
// @Summary 创建角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param request body CreateRequest true "创建角色请求"
// @Success 200 {object} response.Response
// @Router /roles [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	role, err := c.create(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, role)
}

// create 创建角色业务逻辑
func (c *Controller) create(ctx context.Context, req *CreateRequest) (*model.Role, error) {
	existing, err := c.repo.FindByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Duplicate("role code")
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

// Update 更新角色
// @Summary 更新角色
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Param request body UpdateRequest true "更新角色请求"
// @Success 200 {object} response.Response
// @Router /roles/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid role id")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	role, err := c.update(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, role)
}

// update 更新角色业务逻辑
func (c *Controller) update(ctx context.Context, id int64, req *UpdateRequest) (*model.Role, error) {
	role, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.NotFound("role")
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

// Delete 删除角色
// @Summary 删除角色
// @Tags 角色管理
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response
// @Router /roles/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid role id")
	}

	// 删除角色数据权限
	c.dataScopeRepo.DeleteByRoleID(ctx.UserContext(), id)
	// 删除角色权限关联
	c.permCtrl.DeleteRolePermissions(ctx.UserContext(), id)

	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Get 获取角色
// @Summary 获取角色详情
// @Tags 角色管理
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response
// @Router /roles/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid role id")
	}

	role, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if role == nil {
		return response.NotFound(ctx, "role not found")
	}

	return response.Success(ctx, role)
}

// List 角色列表
// @Summary 角色列表
// @Tags 角色管理
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param name query string false "名称"
// @Param code query string false "编码"
// @Param status query int false "状态"
// @Success 200 {object} response.Response
// @Router /roles [get]
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

// list 角色列表业务逻辑
func (c *Controller) list(ctx context.Context, req *ListRequest) (*dal.PagedResult[model.Role], error) {
	pagination := dal.NewPagination(req.Page, req.PageSize)
	qb := dal.NewQueryBuilder[model.Role](c.repo.DB())

	if req.Name != "" {
		qb.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Code != "" {
		qb.Where("code LIKE ?", "%"+req.Code+"%")
	}
	if req.Status != nil {
		qb.Where("status = ?", *req.Status)
	}

	qb.Order("sort ASC, id DESC")

	return qb.Paged(ctx, pagination)
}

// GetAll 获取所有角色
// @Summary 获取所有角色
// @Tags 角色管理
// @Success 200 {object} response.Response
// @Router /roles/all [get]
func (c *Controller) GetAll(ctx *fiber.Ctx) error {
	roles, err := c.repo.FindAll(ctx.UserContext(), map[string]interface{}{"status": 1}, dal.WithOrder("sort ASC"))
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, roles)
}

// GetPermissions 获取角色权限
// @Summary 获取角色权限
// @Tags 角色管理
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response
// @Router /roles/{id}/permissions [get]
func (c *Controller) GetPermissions(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid role id")
	}

	permissions, err := c.permCtrl.GetByRoleID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, permissions)
}

// SetPermissions 设置角色权限
// @Summary 设置角色权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Param request body SetPermissionsRequest true "设置权限请求"
// @Success 200 {object} response.Response
// @Router /roles/{id}/permissions [put]
func (c *Controller) SetPermissions(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid role id")
	}

	var req SetPermissionsRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	if err := c.setPermissions(ctx.UserContext(), id, req.PermissionIDs); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// setPermissions 设置角色权限业务逻辑
func (c *Controller) setPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	role, err := c.repo.FindByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.NotFound("role")
	}

	// 设置角色权限
	if err := c.permCtrl.SetRolePermissions(ctx, roleID, permissionIDs); err != nil {
		return err
	}

	// 同步到Casbin
	permissions, err := c.permCtrl.GetByRoleID(ctx, roleID)
	if err != nil {
		return err
	}

	casbinPerms := make([]auth.Permission, len(permissions))
	for i, p := range permissions {
		casbinPerms[i] = auth.Permission{
			Resource: p.Resource,
			Action:   p.Action,
		}
	}

	return c.casbinService.SetRolePermissions(role.Code, casbinPerms)
}

// GetDataScope 获取角色数据权限
// @Summary 获取角色数据权限
// @Tags 角色管理
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response
// @Router /roles/{id}/datascope [get]
func (c *Controller) GetDataScope(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid role id")
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

// SetDataScope 设置角色数据权限
// @Summary 设置角色数据权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Param request body SetDataScopeRequest true "设置数据权限请求"
// @Success 200 {object} response.Response
// @Router /roles/{id}/datascope [put]
func (c *Controller) SetDataScope(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid role id")
	}

	var req SetDataScopeRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	// 删除现有数据权限
	if err := c.dataScopeRepo.DeleteByRoleID(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	// 批量创建新数据权限
	if err := c.dataScopeRepo.BatchCreate(ctx.UserContext(), id, req.DeptIDs); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// GetByID 根据ID获取角色（供内部调用）
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
		return nil, errors.NotFound("role")
	}

	info := auth.NewDataScopeInfo(auth.DataScopeType(role.DataScope), userID, deptID)

	// 如果是自定义数据权限,获取部门ID列表
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
