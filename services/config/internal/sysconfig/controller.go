package sysconfig

import (
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/config/internal/model"
)

// Info 获取配置详情 (US6)
func Info(e *core.RequestEvent) error {
	idStr := e.Request.URL.Query().Get("id")
	if idStr == "" {
		return apis.Error(e, 400, "配置ID不能为空")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的配置ID")
	}

	sysConfig, err := model.SysConfigs.GetOne(id)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if sysConfig == nil {
		return apis.Error(e, 404, "参数配置不存在")
	}

	return apis.Success(e, sysConfig)
}

// GetByKey 按键名获取配置 (US3)
func GetByKey(e *core.RequestEvent) error {
	configKey := e.Request.URL.Query().Get("configKey")
	if configKey == "" {
		return apis.Error(e, 400, "参数键名不能为空")
	}

	sysConfig, err := model.SysConfigs.GetByKey(configKey)
	if err != nil {
		return apis.Error(e, 404, "参数配置不存在")
	}

	return apis.Success(e, sysConfig)
}

// Page 分页查询列表 (US1)
func Page(e *core.RequestEvent) error {
	query := e.Request.URL.Query()
	
	// 解析分页参数
	page := 1
	pageSize := 10
	if p := query.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := query.Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	configName := query.Get("configName")
	configKey := query.Get("configKey")
	configType := query.Get("configType")

	// 构建 SSQL 过滤条件
	filter := ""
	if configName != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "config_name ~ '" + configName + "'"
	}
	if configKey != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "config_key ~ '" + configKey + "'"
	}
	if configType != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "config_type = '" + configType + "'"
	}

	// 构建列表查询参数
	params := &dal.ListParams{
		Page:    page,
		PerPage: pageSize,
		Filter:  filter,
		Sort:    "-id",
	}

	// 查询列表
	result, err := model.SysConfigs.GetList(params)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}

	return apis.Paged(e, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// Add 新增配置 (US2)
func Add(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	// 必填字段验证
	if req.ConfigName == "" {
		return apis.Error(e, 400, "参数名称不能为空")
	}
	if req.ConfigKey == "" {
		return apis.Error(e, 400, "参数键名不能为空")
	}

	// 键名唯一性校验
	exists, err := model.SysConfigs.ExistsByKey(req.ConfigKey)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if exists {
		return apis.Error(e, 409, "参数键名已存在")
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

	// 获取当前用户ID
	if userID := apis.GetUserID(e); userID > 0 {
		sysConfig.CreateBy = userID
	}

	if err := model.SysConfigs.Create(sysConfig); err != nil {
		return apis.ErrorFromErr(e, err)
	}

	return apis.Success(e, sysConfig)
}

// Update 更新配置 (US4)
func Update(e *core.RequestEvent) error {
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	// ID 必填验证
	if req.ID <= 0 {
		return apis.Error(e, 400, "配置ID不能为空")
	}

	// 获取现有配置
	sysConfig, err := model.SysConfigs.GetOne(req.ID)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if sysConfig == nil {
		return apis.Error(e, 404, "参数配置不存在")
	}

	// 如果要更新键名，检查唯一性
	if req.ConfigKey != "" && req.ConfigKey != sysConfig.ConfigKey {
		exists, err := model.SysConfigs.ExistsByKey(req.ConfigKey, req.ID)
		if err != nil {
			return apis.ErrorFromErr(e, err)
		}
		if exists {
			return apis.Error(e, 409, "参数键名已存在")
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
		return apis.ErrorFromErr(e, err)
	}

	return apis.Success(e, sysConfig)
}

// Remove 批量删除配置 (US5)
func Remove(e *core.RequestEvent) error {
	var req RemoveRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}

	// 验证 ID 列表
	if len(req.IDs) == 0 {
		return apis.Error(e, 400, "配置ID列表不能为空")
	}

	// 批量删除
	affected, err := model.SysConfigs.DeleteByIds(req.IDs)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}

	return apis.Success(e, map[string]any{
		"deleted": affected,
	})
}
