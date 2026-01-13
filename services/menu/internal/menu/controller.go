package menu

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/menu/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 菜单控制器
type Controller struct {
	repo       Repository
	collection *dal.Collection[model.Menu]
}

// NewController 创建菜单控制器
func NewController(repo Repository) *Controller {
	return &Controller{
		repo: repo,
		collection: dal.NewCollection[model.Menu](repo.DB()).
			WithDefaultSort("sort,id").
			WithMaxPerPage(500).
			WithFieldAlias(map[string]string{
				"createdAt": "created_at",
				"updatedAt": "updated_at",
				"parentId":  "parent_id",
				"permCode":  "perm_code",
			}),
	}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	g := r.Group("/menus", jwtMiddleware)
	g.Post("", c.create)
	g.Put("/:id", c.update)
	g.Delete("/:id", c.delete)
	g.Get("/:id", c.get)
	g.Get("", c.list)
	g.Get("/tree", c.getTree)
	g.Get("/user/tree", c.getUserMenuTree)
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	menu, err := c.doCreate(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, menu)
}

func (c *Controller) doCreate(ctx context.Context, req *CreateRequest) (*model.Menu, error) {
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

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的菜单ID")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	menu, err := c.doUpdate(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, menu)
}

func (c *Controller) doUpdate(ctx context.Context, id int64, req *UpdateRequest) (*model.Menu, error) {
	menu, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if menu == nil {
		return nil, errors.NotFound("菜单")
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

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的菜单ID")
	}
	children, err := c.repo.FindByParentID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if len(children) > 0 {
		return response.BadRequest(ctx, "存在子菜单，无法删除")
	}
	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的菜单ID")
	}
	menu, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if menu == nil {
		return response.NotFound(ctx, "菜单不存在")
	}
	return response.Success(ctx, menu)
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	menus, err := c.collection.GetFullList(ctx.UserContext(), params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, menus)
}

func (c *Controller) getTree(ctx *fiber.Ctx) error {
	tree, err := c.doGetTree(ctx.UserContext())
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, tree)
}

func (c *Controller) doGetTree(ctx context.Context) ([]*model.Menu, error) {
	menus, err := c.repo.FindAll(ctx, map[string]interface{}{"status": 1}, dal.WithOrder("sort ASC, id ASC"))
	if err != nil {
		return nil, err
	}
	return buildMenuTree(menus, 0), nil
}

func (c *Controller) getUserMenuTree(ctx *fiber.Ctx) error {
	permCodes := ctx.Locals("permCodes")
	var codes []string
	if permCodes != nil {
		if pc, ok := permCodes.([]string); ok {
			codes = pc
		}
	}
	tree, err := c.doGetUserMenuTree(ctx.UserContext(), codes)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, tree)
}

func (c *Controller) doGetUserMenuTree(ctx context.Context, permCodes []string) ([]*model.Menu, error) {
	menus, err := c.repo.FindAll(ctx, map[string]interface{}{"status": 1, "visible": 1}, dal.WithOrder("sort ASC, id ASC"))
	if err != nil {
		return nil, err
	}
	permCodeSet := make(map[string]struct{})
	for _, code := range permCodes {
		permCodeSet[code] = struct{}{}
	}
	var filteredMenus []model.Menu
	for _, menu := range menus {
		if menu.Type == 1 || menu.PermCode == "" {
			filteredMenus = append(filteredMenus, menu)
			continue
		}
		if _, ok := permCodeSet[menu.PermCode]; ok {
			filteredMenus = append(filteredMenus, menu)
		}
	}
	return buildMenuTree(filteredMenus, 0), nil
}

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
