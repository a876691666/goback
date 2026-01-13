package dicttype

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/dict/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 字典类型控制器
type Controller struct {
	repo       Repository
	collection *dal.Collection[model.DictType]
}

// NewController 创建字典类型控制器
func NewController(repo Repository) *Controller {
	return &Controller{
		repo: repo,
		collection: dal.NewCollection[model.DictType](repo.DB()).
			WithDefaultSort("-id").
			WithMaxPerPage(100).
			WithFieldAlias(map[string]string{
				"createdAt": "created_at",
				"updatedAt": "updated_at",
			}),
	}
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
	dictType, err := c.doCreate(ctx.UserContext(), &req)
	if err != nil {
		if err == errors.ErrRecordExists {
			return response.Error(ctx, 400, "字典编码已存在")
		}
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dictType)
}

func (c *Controller) doCreate(ctx context.Context, req *CreateRequest) (*model.DictType, error) {
	exists, err := c.repo.ExistsByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrRecordExists
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
	if err := c.repo.Create(ctx, dictType); err != nil {
		return nil, err
	}
	return dictType, nil
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
	if err := c.doUpdate(ctx.UserContext(), id, &req); err != nil {
		if err == errors.ErrRecordExists {
			return response.Error(ctx, 400, "字典编码已存在")
		}
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) doUpdate(ctx context.Context, id int64, req *UpdateRequest) error {
	dictType, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if dictType == nil {
		return errors.NotFound("字典类型")
	}
	if req.Code != "" && req.Code != dictType.Code {
		exists, err := c.repo.ExistsByCode(ctx, req.Code, id)
		if err != nil {
			return err
		}
		if exists {
			return errors.ErrRecordExists
		}
	}
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Code != "" {
		updates["code"] = req.Code
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Remark != "" {
		updates["description"] = req.Remark
	}
	return c.repo.UpdateFields(ctx, id, updates)
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典类型ID")
	}
	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的字典类型ID")
	}
	dictType, err := c.repo.FindByID(ctx.UserContext(), id)
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
	result, err := c.collection.GetList(ctx.UserContext(), params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// GetByCode 根据编码获取字典类型
func (c *Controller) GetByCode(ctx context.Context, code string) (*model.DictType, error) {
	return c.repo.FindByCode(ctx, code)
}

// GetByID 根据ID获取字典类型
func (c *Controller) GetByID(ctx context.Context, id int64) (*model.DictType, error) {
	return c.repo.FindByID(ctx, id)
}
