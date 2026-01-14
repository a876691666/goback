package permission

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/lifecycle"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/rbac/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 权限控制器
type Controller struct {
	router.BaseController
}

// Prefix 返回路由前缀
func (c *Controller) Prefix() string {
	return "/permissions"
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
	}
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	perm, err := c.doCreate(&req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	// 广播权限变更
	c.broadcastPermissions()
	return response.Success(ctx, perm)
}

func (c *Controller) doCreate(req *CreateRequest) (*model.Permission, error) {
	exists, err := model.Permissions.ExistsByCode(req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.Duplicate("权限编码")
	}
	perm := &model.Permission{
		Name:        req.Name,
		Code:        req.Code,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: req.Description,
	}
	if err := model.Permissions.Create(perm); err != nil {
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
	perm, err := c.doUpdate(id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	// 广播权限变更
	c.broadcastPermissions()
	return response.Success(ctx, perm)
}

func (c *Controller) doUpdate(id int64, req *UpdateRequest) (*model.Permission, error) {
	perm, err := model.Permissions.GetOne(id)
	if err != nil {
		return nil, err
	}
	if perm == nil {
		return nil, errors.NotFound("权限")
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
		return nil, err
	}
	return perm, nil
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的权限ID")
	}
	if err := model.Permissions.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	// 广播权限变更
	c.broadcastPermissions()
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的权限ID")
	}
	perm, err := model.Permissions.GetOne(id)
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
	result, err := model.Permissions.GetList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

func (c *Controller) getAll(ctx *fiber.Ctx) error {
	permissions, err := model.Permissions.GetFullList(&dal.ListParams{
		Sort: "id",
	})
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, permissions)
}

// GetByRoleID 根据角色ID获取权限列表
func (c *Controller) GetByRoleID(roleID int64) ([]model.Permission, error) {
	return model.Permissions.GetByRoleID(roleID)
}

// SetRolePermissions 设置角色权限
func (c *Controller) SetRolePermissions(roleID int64, permissionIDs []int64) error {
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
	if err := model.RolePermissions.CreateBatch(rps); err != nil {
		return err
	}
	// 广播角色权限变更
	c.broadcastRolePermissions()
	return nil
}

// DeleteRolePermissions 删除角色权限
func (c *Controller) DeleteRolePermissions(roleID int64) error {
	if err := model.RolePermissions.DeleteByRoleID(roleID); err != nil {
		return err
	}
	// 广播角色权限变更
	c.broadcastRolePermissions()
	return nil
}

// broadcastPermissions 广播权限数据
func (c *Controller) broadcastPermissions() {
	go func() {
		permissions := LoadPermissions()
		c.Broadcast(lifecycle.ModuleRBAC, lifecycle.KeyPermissions, permissions)
	}()
}

// broadcastRolePermissions 广播角色权限映射
func (c *Controller) broadcastRolePermissions() {
	go func() {
		rolePerms := LoadRolePermissions()
		c.Broadcast(lifecycle.ModuleRBAC, lifecycle.KeyRolePermissions, rolePerms)
	}()
}

// LoadPermissions 加载所有权限（导出供其他模块使用）
func LoadPermissions() []lifecycle.Permission {
	var perms []model.Permission
	if err := model.Permissions.DB().Find(&perms).Error; err != nil {
		return nil
	}

	result := make([]lifecycle.Permission, len(perms))
	for i, p := range perms {
		result[i] = lifecycle.Permission{
			ID:       p.ID,
			Code:     p.Code,
			Name:     p.Name,
			Resource: p.Resource,
			Action:   p.Action,
		}
	}
	return result
}

// LoadRolePermissions 加载角色权限映射
func LoadRolePermissions() lifecycle.RolePermissionMap {
	result := make(lifecycle.RolePermissionMap)
	
	var rolePerms []model.RolePermission
	model.RolePermissions.DB().Find(&rolePerms)
	
	// 按角色分组
	rolePermMap := make(map[int64][]int64)
	for _, rp := range rolePerms {
		rolePermMap[rp.RoleID] = append(rolePermMap[rp.RoleID], rp.PermissionID)
	}
	
	// 获取所有权限
	permMap := make(map[int64]lifecycle.Permission)
	for _, p := range LoadPermissions() {
		permMap[p.ID] = p
	}
	
	// 构建角色权限映射
	for roleID, permIDs := range rolePermMap {
		perms := make([]lifecycle.Permission, 0, len(permIDs))
		for _, permID := range permIDs {
			if p, ok := permMap[permID]; ok {
				perms = append(perms, p)
			}
		}
		result[roleID] = perms
	}
	
	return result
}
