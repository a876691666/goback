package dicttype

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/services/dict/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 字典类型控制器
type Controller struct{}

// NewController 创建字典类型控制器
func NewController() *Controller {
	return &Controller{}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	g := r.Group("/dict-types", jwtMiddleware)
	g.Post("", c.create)
	g.Put("/:id", c.update)
	g.Delete("/:id", c.delete)
	g.Get("/:id", c.get)
	g.Get("", c.list)
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	exists, err := model.DictTypes.ExistsByCode(req.Code)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if exists {
		return response.Error(ctx, 400, "字典编码已存在")
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
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dictType)
}

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典类型ID")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	dictType, err := model.DictTypes.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dictType == nil {
		return response.NotFound(ctx, "字典类型不存在")
	}

	if req.Code != "" && req.Code != dictType.Code {
		exists, err := model.DictTypes.ExistsByCode(req.Code, id)
		if err != nil {
			return response.Error(ctx, 500, err.Error())
		}
		if exists {
			return response.Error(ctx, 400, "字典编码已存在")
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
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dictType)
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典类型ID")
	}
	if err := model.DictTypes.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典类型ID")
	}
	dictType, err := model.DictTypes.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dictType == nil {
		return response.NotFound(ctx, "字典类型不存在")
	}
	return response.Success(ctx, dictType)
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	result, err := model.DictTypes.GetList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// GetByCode 根据编码获取字典类型
func (c *Controller) GetByCode(code string) (*model.DictType, error) {
	return model.DictTypes.GetByCode(code)
}

// GetByID 根据ID获取字典类型
func (c *Controller) GetByID(id int64) (*model.DictType, error) {
	return model.DictTypes.GetOne(id)
}
