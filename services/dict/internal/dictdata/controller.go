package dictdata

import (
	"fmt"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/services/dict/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 字典数据控制器
type Controller struct{}

// NewController 创建字典数据控制器
func NewController() *Controller {
	return &Controller{}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	g := r.Group("/dict-data", jwtMiddleware)
	g.Post("", c.create)
	g.Put("/:id", c.update)
	g.Delete("/:id", c.delete)
	g.Get("/:id", c.get)
	g.Get("/type/:typeId", c.listByType)
	r.Get("/dicts/:code", c.getByCode)
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
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
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dictData)
}

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典数据ID")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	dictData, err := model.DictDatas.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dictData == nil {
		return response.NotFound(ctx, "字典数据不存在")
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
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dictData)
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典数据ID")
	}
	if err := model.DictDatas.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典数据ID")
	}
	dictData, err := model.DictDatas.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dictData == nil {
		return response.NotFound(ctx, "字典数据不存在")
	}
	return response.Success(ctx, dictData)
}

func (c *Controller) listByType(ctx *fiber.Ctx) error {
	typeID, err := dal.ParseInt64ID(ctx.Params("typeId"))
	if err != nil {
		return response.BadRequest(ctx, "无效的类型ID")
	}
	list, err := model.DictDatas.GetFullList(&dal.ListParams{
		Filter: fmt.Sprintf("dict_type_id=%d", typeID),
		Sort:   "sort",
	})
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, list)
}

func (c *Controller) getByCode(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	if code == "" {
		return response.BadRequest(ctx, "无效的字典编码")
	}
	list, err := c.GetByTypeCode(code)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, list)
}

// GetByTypeCode 根据类型编码获取字典数据
func (c *Controller) GetByTypeCode(code string) ([]model.DictData, error) {
	return model.DictDatas.GetByTypeCode(code)
}
