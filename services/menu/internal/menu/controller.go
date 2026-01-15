package menu

import (
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/menu/internal/model"
)

// Create 创建菜单
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
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
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, menu)
}

// Update 更新菜单
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的菜单ID")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}
	menu, err := model.Menus.GetOne(id)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if menu == nil {
		return apis.Error(e, 404, "菜单不存在")
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
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, menu)
}

// Delete 删除菜单
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的菜单ID")
	}
	children, err := model.Menus.GetFullList(&dal.ListParams{
		Filter: "parent_id=" + e.Request.PathValue("id"),
	})

	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if len(children) > 0 {
		return apis.Error(e, 400, "存在子菜单，无法删除")
	}
	if err := model.Menus.DeleteByID(id); err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, nil)
}

// Get 获取菜单详情
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的菜单ID")
	}
	menu, err := model.Menus.GetOne(id)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if menu == nil {
		return apis.Error(e, 404, "菜单不存在")
	}
	return apis.Success(e, menu)
}

// List 菜单列表
func List(e *core.RequestEvent) error {
	params, err := dal.BindQueryFromRequest(e.Request)
	if err != nil {
		return apis.Error(e, 400, err.Error())
	}
	menus, err := model.Menus.GetFullList(params)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, menus)
}

// GetTree 获取菜单树
func GetTree(e *core.RequestEvent) error {
	menus, err := model.Menus.GetFullList(&dal.ListParams{
		Filter: "status=1",
		Sort:   "sort,id",
	})
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, buildMenuTree(menus, 0))
}

// GetUserMenuTree 获取用户菜单树
func GetUserMenuTree(e *core.RequestEvent) error {
	permCodes := e.Get("permCodes")
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
		return apis.ErrorFromErr(e, err)
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
	return apis.Success(e, buildMenuTree(filteredMenus, 0))
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

// ================== 角色菜单关联 ==================

// GetRoleMenus 获取角色菜单
func GetRoleMenus(e *core.RequestEvent) error {
	roleID, err := strconv.ParseInt(e.Request.PathValue("roleId"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	menus, err := model.Menus.GetByRoleID(roleID)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, menus)
}

// SetRoleMenus 设置角色菜单
func SetRoleMenus(e *core.RequestEvent) error {
	roleID, err := strconv.ParseInt(e.Request.PathValue("roleId"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	var req SetRoleMenusRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}
	if err := doSetRoleMenus(roleID, req.MenuIDs); err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, nil)
}

func doSetRoleMenus(roleID int64, menuIDs []int64) error {
	// 删除原有关联
	if err := model.RoleMenus.DeleteByRoleID(roleID); err != nil {
		return err
	}
	if len(menuIDs) == 0 {
		return nil
	}
	// 创建新关联
	rms := make([]model.RoleMenu, len(menuIDs))
	for i, menuID := range menuIDs {
		rms[i] = model.RoleMenu{
			RoleID: roleID,
			MenuID: menuID,
		}
	}
	return model.RoleMenus.CreateBatch(rms)
}

// GetRoleMenuTree 获取角色菜单树
func GetRoleMenuTree(e *core.RequestEvent) error {
	roleID, err := strconv.ParseInt(e.Request.PathValue("roleId"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的角色ID")
	}
	menus, err := model.Menus.GetByRoleID(roleID)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, buildMenuTree(menus, 0))
}

// ================== 导出函数（供其他服务调用） ==================

// GetByRoleID 根据角色ID获取菜单列表
func GetByRoleID(roleID int64) ([]model.Menu, error) {
	return model.Menus.GetByRoleID(roleID)
}

// GetByRoleIDs 根据多个角色ID获取菜单列表
func GetByRoleIDs(roleIDs []int64) ([]model.Menu, error) {
	return model.Menus.GetByRoleIDs(roleIDs)
}

// DoSetRoleMenus 设置角色菜单（供其他服务调用）
func DoSetRoleMenus(roleID int64, menuIDs []int64) error {
	return doSetRoleMenus(roleID, menuIDs)
}

// DeleteRoleMenus 删除角色菜单（供其他服务调用）
func DeleteRoleMenus(roleID int64) error {
	return model.RoleMenus.DeleteByRoleID(roleID)
}
