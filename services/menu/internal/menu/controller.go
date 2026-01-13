package menu

import (
	"context"
	"strconv"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/menu/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 菜单控制器
type Controller struct {
	repo Repository
}

// NewController 创建菜单控制器
func NewController(repo Repository) *Controller {
	return &Controller{repo: repo}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	menus := r.Group("/menus", jwtMiddleware)
	menus.Post("", c.Create)
	menus.Put("/:id", c.Update)
	menus.Delete("/:id", c.Delete)
	menus.Get("/:id", c.Get)
	menus.Get("", c.List)
	menus.Get("/tree", c.GetTree)
	menus.Get("/user/tree", c.GetUserMenuTree)
}

// Create 创建菜单
// @Summary 创建菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param request body CreateRequest true "创建菜单请求"
// @Success 200 {object} response.Response
// @Router /menus [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	menu, err := c.create(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, menu)
}

// create 创建菜单业务逻辑
func (c *Controller) create(ctx context.Context, req *CreateRequest) (*model.Menu, error) {
	menu := &model.Menu{
		ParentID:  req.ParentID,
		Name:      req.Name,
		Path:      req.Path,
		Component: req.Component,
		Icon:      req.Icon,
		Type:      req.Type,
		Visible:   req.Visible,
		Status:    req.Status,
		Redirect:  req.Redirect,
		Sort:      req.Sort,
		PermCode:  req.PermCode,
	}

	if menu.Type == 0 {
		menu.Type = 1
	}
	if menu.Visible == 0 {
		menu.Visible = 1
	}
	if menu.Status == 0 {
		menu.Status = 1
	}

	if err := c.repo.Create(ctx, menu); err != nil {
		return nil, err
	}

	return menu, nil
}

// Update 更新菜单
// @Summary 更新菜单
// @Tags 菜单管理
// @Accept json
// @Produce json
// @Param id path int true "菜单ID"
// @Param request body UpdateRequest true "更新菜单请求"
// @Success 200 {object} response.Response
// @Router /menus/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid menu id")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	menu, err := c.update(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, menu)
}

// update 更新菜单业务逻辑
func (c *Controller) update(ctx context.Context, id int64, req *UpdateRequest) (*model.Menu, error) {
	menu, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if menu == nil {
		return nil, errors.NotFound("menu")
	}

	if req.Name != "" {
		menu.Name = req.Name
	}
	menu.ParentID = req.ParentID
	if req.Path != "" {
		menu.Path = req.Path
	}
	if req.Component != "" {
		menu.Component = req.Component
	}
	if req.Icon != "" {
		menu.Icon = req.Icon
	}
	if req.Type > 0 {
		menu.Type = req.Type
	}
	if req.Visible > 0 {
		menu.Visible = req.Visible
	}
	if req.Status > 0 {
		menu.Status = req.Status
	}
	if req.Redirect != "" {
		menu.Redirect = req.Redirect
	}
	menu.Sort = req.Sort
	if req.PermCode != "" {
		menu.PermCode = req.PermCode
	}

	if err := c.repo.Update(ctx, menu); err != nil {
		return nil, err
	}

	return menu, nil
}

// Delete 删除菜单
// @Summary 删除菜单
// @Tags 菜单管理
// @Param id path int true "菜单ID"
// @Success 200 {object} response.Response
// @Router /menus/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid menu id")
	}

	// 检查是否有子菜单
	children, err := c.repo.FindByParentID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if len(children) > 0 {
		return response.BadRequest(ctx, "cannot delete menu with children")
	}

	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Get 获取菜单
// @Summary 获取菜单详情
// @Tags 菜单管理
// @Param id path int true "菜单ID"
// @Success 200 {object} response.Response
// @Router /menus/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid menu id")
	}

	menu, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if menu == nil {
		return response.NotFound(ctx, "menu not found")
	}

	return response.Success(ctx, menu)
}

// List 菜单列表
// @Summary 菜单列表
// @Tags 菜单管理
// @Param name query string false "名称"
// @Param status query int false "状态"
// @Param parentId query int false "父ID"
// @Success 200 {object} response.Response
// @Router /menus [get]
func (c *Controller) List(ctx *fiber.Ctx) error {
	var req ListRequest
	if err := ctx.QueryParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	menus, err := c.list(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, menus)
}

// list 菜单列表业务逻辑
func (c *Controller) list(ctx context.Context, req *ListRequest) ([]model.Menu, error) {
	qb := dal.NewQueryBuilder[model.Menu](c.repo.DB())

	if req.Name != "" {
		qb.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Status != nil {
		qb.Where("status = ?", *req.Status)
	}
	if req.ParentID != nil {
		qb.Where("parent_id = ?", *req.ParentID)
	}

	qb.Order("sort ASC, id ASC")

	return qb.Find(ctx)
}

// GetTree 获取菜单树
// @Summary 获取菜单树
// @Tags 菜单管理
// @Success 200 {object} response.Response
// @Router /menus/tree [get]
func (c *Controller) GetTree(ctx *fiber.Ctx) error {
	tree, err := c.getTree(ctx.UserContext())
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, tree)
}

// getTree 获取菜单树业务逻辑
func (c *Controller) getTree(ctx context.Context) ([]*model.Menu, error) {
	menus, err := c.repo.FindAll(ctx, map[string]interface{}{"status": 1}, dal.WithOrder("sort ASC, id ASC"))
	if err != nil {
		return nil, err
	}

	return buildMenuTree(menus, 0), nil
}

// GetUserMenuTree 获取用户菜单树
// @Summary 获取用户菜单树
// @Tags 菜单管理
// @Success 200 {object} response.Response
// @Router /menus/user/tree [get]
func (c *Controller) GetUserMenuTree(ctx *fiber.Ctx) error {
	// 从JWT中获取权限编码（实际应该从context中获取）
	permCodes := ctx.Locals("permCodes")
	var codes []string
	if permCodes != nil {
		if pc, ok := permCodes.([]string); ok {
			codes = pc
		}
	}

	tree, err := c.getUserMenuTree(ctx.UserContext(), codes)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, tree)
}

// getUserMenuTree 获取用户菜单树业务逻辑
func (c *Controller) getUserMenuTree(ctx context.Context, permCodes []string) ([]*model.Menu, error) {
	menus, err := c.repo.FindAll(ctx, map[string]interface{}{
		"status":  1,
		"visible": 1,
	}, dal.WithOrder("sort ASC, id ASC"))
	if err != nil {
		return nil, err
	}

	// 过滤有权限的菜单
	permCodeSet := make(map[string]struct{})
	for _, code := range permCodes {
		permCodeSet[code] = struct{}{}
	}

	var filteredMenus []model.Menu
	for _, menu := range menus {
		// 目录类型或无权限编码的菜单默认显示
		if menu.Type == 1 || menu.PermCode == "" {
			filteredMenus = append(filteredMenus, menu)
			continue
		}
		// 检查权限
		if _, ok := permCodeSet[menu.PermCode]; ok {
			filteredMenus = append(filteredMenus, menu)
		}
	}

	return buildMenuTree(filteredMenus, 0), nil
}

// buildMenuTree 构建菜单树
func buildMenuTree(menus []model.Menu, parentID int64) []*model.Menu {
	var tree []*model.Menu

	for i := range menus {
		if menus[i].ParentID == parentID {
			menu := &menus[i]
			menu.Children = buildMenuTree(menus, menu.ID)
			tree = append(tree, menu)
		}
	}

	return tree
}
