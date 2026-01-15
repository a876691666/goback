package sysconfig

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/config/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 系统参数配置控制器
type Controller struct {
	router.BaseController
}

// Prefix 返回路由前缀
func (c *Controller) Prefix() string {
	return "/config"
}

// Routes 返回路由配置
func (c *Controller) Routes(middlewares map[string]fiber.Handler) []router.Route {
	return []router.Route{
		// US6: 获取配置详情
		{Method: "GET", Path: "info", Handler: c.info, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		// US3: 按键名获取配置
		{Method: "GET", Path: "get-by-key", Handler: c.getByKey},
		// US1: 分页查询列表
		{Method: "GET", Path: "page", Handler: c.page, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		// US2: 新增配置
		{Method: "POST", Path: "add", Handler: c.add, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		// US4: 更新配置
		{Method: "PUT", Path: "update", Handler: c.update, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		// US5: 批量删除配置
		{Method: "DELETE", Path: "remove", Handler: c.remove, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
	}
}

// info 获取配置详情 (US6)
func (c *Controller) info(ctx *fiber.Ctx) error {
	idStr := ctx.Query("id")
	if idStr == "" {
		return response.ValidateError(ctx, "配置ID不能为空")
	}

	id, err := dal.ParseInt64ID(idStr)
	if err != nil {
		return response.BadRequest(ctx, "无效的配置ID")
	}

	sysConfig, err := model.SysConfigs.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if sysConfig == nil {
		return response.NotFound(ctx, "参数配置不存在")
	}

	return response.Success(ctx, sysConfig)
}

// getByKey 按键名获取配置 (US3)
func (c *Controller) getByKey(ctx *fiber.Ctx) error {
	configKey := ctx.Query("configKey")
	if configKey == "" {
		return response.ValidateError(ctx, "参数键名不能为空")
	}

	sysConfig, err := model.SysConfigs.GetByKey(configKey)
	if err != nil {
		return response.NotFound(ctx, "参数配置不存在")
	}

	return response.Success(ctx, sysConfig)
}

// page 分页查询列表 (US1)
func (c *Controller) page(ctx *fiber.Ctx) error {
	// 绑定查询参数
	var req PageRequest
	if err := ctx.QueryParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 构建 SSQL 过滤条件
	filter := ""
	if req.ConfigName != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "config_name ~ '" + req.ConfigName + "'"
	}
	if req.ConfigKey != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "config_key ~ '" + req.ConfigKey + "'"
	}
	if req.ConfigType != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "config_type = '" + req.ConfigType + "'"
	}

	// 构建列表查询参数
	params := &dal.ListParams{
		Page:    req.Page,
		PerPage: req.PageSize,
		Filter:  filter,
		Sort:    "-id",
	}

	// 查询列表
	result, err := model.SysConfigs.GetList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// add 新增配置 (US2)
func (c *Controller) add(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	// 必填字段验证
	if req.ConfigName == "" {
		return response.ValidateError(ctx, "参数名称不能为空")
	}
	if req.ConfigKey == "" {
		return response.ValidateError(ctx, "参数键名不能为空")
	}

	// 键名唯一性校验
	exists, err := model.SysConfigs.ExistsByKey(req.ConfigKey)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if exists {
		return response.Error(ctx, 409, "参数键名已存在")
	}

	// 创建配置
	sysConfig := &model.SysConfig{
		ConfigName:  req.ConfigName,
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigType:  req.ConfigType,
		Remark:      req.Remark,
	}

	// 设置默认值
	if sysConfig.ConfigType == "" {
		sysConfig.ConfigType = "N"
	}

	// 获取当前用户ID（从JWT中解析）
	if userID, ok := ctx.Locals("userID").(int64); ok {
		sysConfig.CreateBy = userID
	}

	if err := model.SysConfigs.Create(sysConfig); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, sysConfig)
}

// update 更新配置 (US4)
func (c *Controller) update(ctx *fiber.Ctx) error {
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	// ID 必填验证
	if req.ID <= 0 {
		return response.ValidateError(ctx, "配置ID不能为空")
	}

	// 获取现有配置
	sysConfig, err := model.SysConfigs.GetOne(req.ID)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if sysConfig == nil {
		return response.NotFound(ctx, "参数配置不存在")
	}

	// 如果要更新键名，检查唯一性
	if req.ConfigKey != "" && req.ConfigKey != sysConfig.ConfigKey {
		exists, err := model.SysConfigs.ExistsByKey(req.ConfigKey, req.ID)
		if err != nil {
			return response.Error(ctx, 500, err.Error())
		}
		if exists {
			return response.Error(ctx, 409, "参数键名已存在")
		}
		sysConfig.ConfigKey = req.ConfigKey
	}

	// 更新字段
	if req.ConfigName != "" {
		sysConfig.ConfigName = req.ConfigName
	}
	if req.ConfigValue != "" {
		sysConfig.ConfigValue = req.ConfigValue
	}
	if req.ConfigType != "" {
		sysConfig.ConfigType = req.ConfigType
	}
	if req.Remark != "" {
		sysConfig.Remark = req.Remark
	}

	if err := model.SysConfigs.Save(sysConfig); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, sysConfig)
}

// remove 批量删除配置 (US5)
func (c *Controller) remove(ctx *fiber.Ctx) error {
	var req RemoveRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	// 验证 ID 列表
	if len(req.IDs) == 0 {
		return response.ValidateError(ctx, "配置ID列表不能为空")
	}

	// 批量删除
	affected, err := model.SysConfigs.DeleteByIds(req.IDs)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, fiber.Map{
		"deleted": affected,
	})
}
