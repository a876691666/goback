package dictdata

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/dict/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 字典数据控制器
type Controller struct {
	repo Repository
}

// NewController 创建字典数据控制器
func NewController(repo Repository) *Controller {
	return &Controller{repo: repo}
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
	dictData, err := c.doCreate(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dictData)
}

func (c *Controller) doCreate(ctx context.Context, req *CreateRequest) (*model.DictData, error) {
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
	if err := c.repo.Create(ctx, dictData); err != nil {
		return nil, err
	}
	return dictData, nil
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
	if err := c.doUpdate(ctx.UserContext(), id, &req); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) doUpdate(ctx context.Context, id int64, req *UpdateRequest) error {
	dictData, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if dictData == nil {
		return errors.NotFound("字典数据")
	}
	updates := make(map[string]interface{})
	if req.Label != "" {
		updates["label"] = req.Label
	}
	if req.Value != "" {
		updates["value"] = req.Value
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.CSSClass != "" {
		updates["css_class"] = req.CSSClass
	}
	if req.ListClass != "" {
		updates["list_class"] = req.ListClass
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	return c.repo.UpdateFields(ctx, id, updates)
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典数据ID")
	}
	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典数据ID")
	}
	dictData, err := c.repo.FindByID(ctx.UserContext(), id)
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
	list, err := c.repo.FindByTypeID(ctx.UserContext(), typeID)
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
	list, err := c.repo.FindByTypeCode(ctx.UserContext(), code)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, list)
}

// DeleteByTypeID 根据类型ID删除字典数据
func (c *Controller) DeleteByTypeID(ctx context.Context, typeID int64) error {
	return c.repo.DeleteByTypeID(ctx, typeID)
}

// GetByTypeCode 根据类型编码获取字典数据
func (c *Controller) GetByTypeCode(ctx context.Context, code string) ([]model.DictData, error) {
	return c.repo.FindByTypeCode(ctx, code)
}
