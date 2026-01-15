package dictdata

import (
	"fmt"
	"strconv"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	"github.com/goback/pkg/dal"
	"github.com/goback/services/dict/internal/model"
)

// Create 创建字典数据
func Create(e *core.RequestEvent) error {
	var req CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}
	dictData := &model.DictData{
		DictTypeID: req.TypeID,
		Label:      req.Label,
		Value:      req.Value,
		Sort:       req.Sort,
		Status:     req.Status,
		CSSClass:   req.CSSClass,
		ListClass:  req.ListClass,
		Remark:     req.Remark,
	}
	if dictData.Status == 0 {
		dictData.Status = 1
	}
	if err := model.DictDatas.Create(dictData); err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, dictData)
}

// Update 更新字典数据
func Update(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的字典数据ID")
	}
	var req UpdateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}
	dictData, err := model.DictDatas.GetOne(id)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if dictData == nil {
		return apis.Error(e, 404, "字典数据不存在")
	}
	if req.Label != "" {
		dictData.Label = req.Label
	}
	if req.Value != "" {
		dictData.Value = req.Value
	}
	if req.Sort != nil {
		dictData.Sort = *req.Sort
	}
	if req.Status != nil {
		dictData.Status = *req.Status
	}
	if req.CSSClass != "" {
		dictData.CSSClass = req.CSSClass
	}
	if req.ListClass != "" {
		dictData.ListClass = req.ListClass
	}
	if req.Remark != "" {
		dictData.Remark = req.Remark
	}
	if err := model.DictDatas.Save(dictData); err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, dictData)
}

// Delete 删除字典数据
func Delete(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的字典数据ID")
	}
	if err := model.DictDatas.DeleteByID(id); err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, nil)
}

// Get 获取字典数据
func Get(e *core.RequestEvent) error {
	id, err := strconv.ParseInt(e.Request.PathValue("id"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的字典数据ID")
	}
	dictData, err := model.DictDatas.GetOne(id)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	if dictData == nil {
		return apis.Error(e, 404, "字典数据不存在")
	}
	return apis.Success(e, dictData)
}

// ListByType 根据类型ID获取字典数据列表
func ListByType(e *core.RequestEvent) error {
	typeID, err := strconv.ParseInt(e.Request.PathValue("typeId"), 10, 64)
	if err != nil {
		return apis.Error(e, 400, "无效的类型ID")
	}
	list, err := model.DictDatas.GetFullList(&dal.ListParams{
		Filter: fmt.Sprintf("dict_type_id=%d", typeID),
		Sort:   "sort",
	})
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, list)
}

// GetByCode 根据字典编码获取字典数据（公开路由）
func GetByCode(e *core.RequestEvent) error {
	code := e.Request.PathValue("code")
	if code == "" {
		return apis.Error(e, 400, "无效的字典编码")
	}
	list, err := GetByTypeCode(code)
	if err != nil {
		return apis.ErrorFromErr(e, err)
	}
	return apis.Success(e, list)
}

// GetByTypeCode 根据类型编码获取字典数据
func GetByTypeCode(code string) ([]model.DictData, error) {
	return model.DictDatas.GetByTypeCode(code)
}
