package menu

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/menu/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 菜单控制器
type Controller struct{}

// Prefix 返回路由前缀
func (c *Controller) Prefix() string {
	return "/menus"
}

// Routes 返回路由配置
func (c *Controller) Routes(middlewares map[string]fiber.Handler) []router.Route {
	return []router.Route{
		{Method: "POST", Path: "", Handler: c.create, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "PUT", Path: "/:id", Handler: c.update, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "DELETE", Path: "/:id", Handler: c.delete, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/:id", Handler: c.get, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "", Handler: c.list, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/tree", Handler: c.getTree, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "GET", Path: "/user/tree", Handler: c.getUserMenuTree, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
	}
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
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
	if err := model.Menus.Save(menu); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, menu)
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
	menu, err := model.Menus.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if menu == nil {
		return response.NotFound(ctx, "菜单不存在")
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
	if err := model.Menus.Save(menu); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, menu)
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的菜单ID")
	}
	children, err := model.Menus.GetFullList(&dal.ListParams{
		Filter: "parent_id=" + ctx.Params("id"),
	})

	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if len(children) > 0 {
		return response.BadRequest(ctx, "存在子菜单，无法删除")
	}
	if err := model.Menus.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的菜单ID")
	}
	menu, err := model.Menus.GetOne(id)
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
	menus, err := model.Menus.GetFullList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, menus)
}

func (c *Controller) getTree(ctx *fiber.Ctx) error {
	menus, err := model.Menus.GetFullList(&dal.ListParams{
		Filter: "status=1",
		Sort:   "sort,id",
	})
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, buildMenuTree(menus, 0))
}

func (c *Controller) getUserMenuTree(ctx *fiber.Ctx) error {
	permCodes := ctx.Locals("permCodes")
	var codes []string
	if permCodes != nil {
		if pc, ok := permCodes.([]string); ok {
			codes = pc
		}
	}
	menus, err := model.Menus.GetFullList(&dal.ListParams{
		Filter: "status=1 && visible=1",
		Sort:   "sort,id",
	})
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	permCodeSet := make(map[string]struct{})
	for _, code := range codes {
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
	return response.Success(ctx, buildMenuTree(filteredMenus, 0))
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
