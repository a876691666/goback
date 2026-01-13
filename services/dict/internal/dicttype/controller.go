package dicttype

import (
	"context"
	"strconv"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/dict/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 字典类型控制器
type Controller struct {
	repo Repository
}

// NewController 创建字典类型控制器
func NewController(repo Repository) *Controller {
	return &Controller{repo: repo}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	dictTypes := r.Group("/dict-types", jwtMiddleware)
	dictTypes.Post("", c.Create)
	dictTypes.Put("/:id", c.Update)
	dictTypes.Delete("/:id", c.Delete)
	dictTypes.Get("/:id", c.Get)
	dictTypes.Get("", c.List)
}

// Create 创建字典类型
// @Summary 创建字典类型
// @Tags 字典类型
// @Accept json
// @Produce json
// @Param request body CreateRequest true "创建字典类型请求"
// @Success 200 {object} response.Response
// @Router /dict-types [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	dictType, err := c.create(ctx.UserContext(), &req)
	if err != nil {
		if err == errors.ErrRecordExists {
			return response.Error(ctx, 400, "字典编码已存在")
		}
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, dictType)
}

// create 创建字典类型业务逻辑
func (c *Controller) create(ctx context.Context, req *CreateRequest) (*model.DictType, error) {
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

// Update 更新字典类型
// @Summary 更新字典类型
// @Tags 字典类型
// @Accept json
// @Produce json
// @Param id path int true "字典类型ID"
// @Param request body UpdateRequest true "更新字典类型请求"
// @Success 200 {object} response.Response
// @Router /dict-types/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid dict type id")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	if err := c.update(ctx.UserContext(), id, &req); err != nil {
		if err == errors.ErrRecordExists {
			return response.Error(ctx, 400, "字典编码已存在")
		}
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// update 更新字典类型业务逻辑
func (c *Controller) update(ctx context.Context, id int64, req *UpdateRequest) error {
	dictType, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if dictType == nil {
		return errors.NotFound("dict type")
	}

	// 检查编码是否已存在
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

// Delete 删除字典类型
// @Summary 删除字典类型
// @Tags 字典类型
// @Param id path int true "字典类型ID"
// @Success 200 {object} response.Response
// @Router /dict-types/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid dict type id")
	}

	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Get 获取字典类型
// @Summary 获取字典类型详情
// @Tags 字典类型
// @Param id path int true "字典类型ID"
// @Success 200 {object} response.Response
// @Router /dict-types/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid dict type id")
	}

	dictType, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dictType == nil {
		return response.NotFound(ctx, "dict type not found")
	}

	return response.Success(ctx, dictType)
}

// List 字典类型列表
// @Summary 字典类型列表
// @Tags 字典类型
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param name query string false "名称"
// @Param code query string false "编码"
// @Param status query int false "状态"
// @Success 200 {object} response.Response
// @Router /dict-types [get]
func (c *Controller) List(ctx *fiber.Ctx) error {
	var req ListRequest
	if err := ctx.QueryParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	result, err := c.list(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.SuccessPage(ctx, result.List, result.Total, result.Page, result.PageSize)
}

// list 字典类型列表业务逻辑
func (c *Controller) list(ctx context.Context, req *ListRequest) (*dal.PagedResult[model.DictType], error) {
	pagination := dal.NewPagination(req.Page, req.PageSize)
	qb := dal.NewQueryBuilder[model.DictType](c.repo.DB())

	if req.Name != "" {
		qb.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Code != "" {
		qb.Where("code LIKE ?", "%"+req.Code+"%")
	}
	if req.Status != nil {
		qb.Where("status = ?", *req.Status)
	}

	qb.Order("id DESC")

	return qb.Paged(ctx, pagination)
}

// GetByCode 根据编码获取字典类型（供内部调用）
func (c *Controller) GetByCode(ctx context.Context, code string) (*model.DictType, error) {
	return c.repo.FindByCode(ctx, code)
}

// GetByID 根据ID获取字典类型（供内部调用）
func (c *Controller) GetByID(ctx context.Context, id int64) (*model.DictType, error) {
	return c.repo.FindByID(ctx, id)
}
