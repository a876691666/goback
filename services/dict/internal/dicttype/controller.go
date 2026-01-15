package dicttype

import (
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/dict/internal/model"
)

// Create 创建字典类型
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}
	exists, err := model.DictTypes.ExistsByCode(req.Code)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if exists {
		return apis.Error(e, 400, "字典编码已存在")
	}
	dictType := &model.DictType{
		Name:        req.Name,
		Code:        req.Code,
		Status:      req.Status,
		Description: req.Remark,
	}
	if dictType.Status == 0 {
		dictType.Status = 1
	}
	if err := model.DictTypes.Create(dictType); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, dictType)
}

// Update 更新字典类型
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的字典类型ID")
	}

	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}
	dictType, err := model.DictTypes.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if dictType == nil {
		return apis.Error(e, 404, "字典类型不存在")
	}

	if req.Code != "" && req.Code != dictType.Code {
		exists, err := model.DictTypes.ExistsByCode(req.Code, id)
		if err != nil {
			return apis.Error(e, 500, err.Error())
		}
		if exists {
			return apis.Error(e, 400, "字典编码已存在")
		}
		dictType.Code = req.Code
	}

	if req.Name != "" {
		dictType.Name = req.Name
	}
	if req.Status != nil {
		dictType.Status = *req.Status
	}
	if req.Remark != "" {
		dictType.Description = req.Remark
	}
	if err := model.DictTypes.Save(dictType); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, dictType)
}

// Delete 删除字典类型
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的字典类型ID")
	}
	if err := model.DictTypes.DeleteByID(id); err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Success(e, nil)
}

// Get 获取字典类型
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的字典类型ID")
	}
	dictType, err := model.DictTypes.GetOne(id)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	if dictType == nil {
		return apis.Error(e, 404, "字典类型不存在")
	}
	return apis.Success(e, dictType)
}

// List 获取字典类型列表
func List(e *core.RequestEvent) error {
	params := &dal.ListParams{
		Page:    apis.GetQueryParamInt(e, "page", 1),
		PerPage: apis.GetQueryParamInt(e, "size", 10),
		Filter:  e.Request.URL.Query().Get("filter"),
		Sort:    e.Request.URL.Query().Get("sort"),
	}
	result, err := model.DictTypes.GetList(params)
	if err != nil {
		return apis.Error(e, 500, err.Error())
	}
	return apis.Paged(e, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// GetByCode 根据编码获取字典类型
func GetByCode(code string) (*model.DictType, error) {
	return model.DictTypes.GetByCode(code)
}

// GetByID 根据ID获取字典类型
func GetByID(id int64) (*model.DictType, error) {
	return model.DictTypes.GetOne(id)
}
