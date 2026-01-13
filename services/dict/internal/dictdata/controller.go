package dictdata

import (
	"context"
	"strconv"

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
	dictData := r.Group("/dict-data", jwtMiddleware)
	dictData.Post("", c.Create)
	dictData.Put("/:id", c.Update)
	dictData.Delete("/:id", c.Delete)
	dictData.Get("/:id", c.Get)
	dictData.Get("/type/:typeId", c.ListByType)

	// 公开接口（根据编码获取字典）
	r.Get("/dicts/:code", c.GetByCode)
}

// Create 创建字典数据
// @Summary 创建字典数据
// @Tags 字典数据
// @Accept json
// @Produce json
// @Param request body CreateRequest true "创建字典数据请求"
// @Success 200 {object} response.Response
// @Router /dict-data [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	dictData, err := c.create(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, dictData)
}

// create 创建字典数据业务逻辑
func (c *Controller) create(ctx context.Context, req *CreateRequest) (*model.DictData, error) {
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

// Update 更新字典数据
// @Summary 更新字典数据
// @Tags 字典数据
// @Accept json
// @Produce json
// @Param id path int true "字典数据ID"
// @Param request body UpdateRequest true "更新字典数据请求"
// @Success 200 {object} response.Response
// @Router /dict-data/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid dict data id")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	if err := c.update(ctx.UserContext(), id, &req); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// update 更新字典数据业务逻辑
func (c *Controller) update(ctx context.Context, id int64, req *UpdateRequest) error {
	dictData, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if dictData == nil {
		return errors.NotFound("dict data")
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

// Delete 删除字典数据
// @Summary 删除字典数据
// @Tags 字典数据
// @Param id path int true "字典数据ID"
// @Success 200 {object} response.Response
// @Router /dict-data/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid dict data id")
	}

	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Get 获取字典数据
// @Summary 获取字典数据详情
// @Tags 字典数据
// @Param id path int true "字典数据ID"
// @Success 200 {object} response.Response
// @Router /dict-data/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id, _ := strconv.ParseInt(ctx.Params("id"), 10, 64)
	if id == 0 {
		return response.BadRequest(ctx, "invalid dict data id")
	}

	dictData, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dictData == nil {
		return response.NotFound(ctx, "dict data not found")
	}

	return response.Success(ctx, dictData)
}

// ListByType 根据类型ID获取字典数据列表
// @Summary 根据类型ID获取字典数据列表
// @Tags 字典数据
// @Param typeId path int true "字典类型ID"
// @Success 200 {object} response.Response
// @Router /dict-data/type/{typeId} [get]
func (c *Controller) ListByType(ctx *fiber.Ctx) error {
	typeID, _ := strconv.ParseInt(ctx.Params("typeId"), 10, 64)
	if typeID == 0 {
		return response.BadRequest(ctx, "invalid type id")
	}

	list, err := c.repo.FindByTypeID(ctx.UserContext(), typeID)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, list)
}

// GetByCode 根据类型编码获取字典数据列表
// @Summary 根据类型编码获取字典数据列表
// @Tags 字典数据
// @Param code path string true "字典类型编码"
// @Success 200 {object} response.Response
// @Router /dicts/{code} [get]
func (c *Controller) GetByCode(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	if code == "" {
		return response.BadRequest(ctx, "invalid dict code")
	}

	list, err := c.repo.FindByTypeCode(ctx.UserContext(), code)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, list)
}

// DeleteByTypeID 根据类型ID删除所有字典数据（供内部调用）
func (c *Controller) DeleteByTypeID(ctx context.Context, typeID int64) error {
	return c.repo.DeleteByTypeID(ctx, typeID)
}

// GetByTypeCode 根据类型编码获取字典数据（供内部调用）
func (c *Controller) GetByTypeCode(ctx context.Context, code string) ([]model.DictData, error) {
	return c.repo.FindByTypeCode(ctx, code)
}
