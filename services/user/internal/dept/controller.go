package dept

import (
	"context"

	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/user/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 部门控制器
type Controller struct {
	repo Repository
}

// NewController 创建部门控制器
func NewController(repo Repository) *Controller {
	return &Controller{repo: repo}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	depts := r.Group("/depts", jwtMiddleware)
	depts.Post("", c.Create)
	depts.Put("/:id", c.Update)
	depts.Delete("/:id", c.Delete)
	depts.Get("/:id", c.Get)
	depts.Get("", c.List)
	depts.Get("/tree", c.GetTree)
}

// Create 创建部门
// @Summary 创建部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param request body CreateRequest true "创建部门请求"
// @Success 200 {object} response.Response
// @Router /depts [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	dept, err := c.create(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, dept)
}

// create 创建部门业务逻辑
func (c *Controller) create(ctx context.Context, req *CreateRequest) (*model.Dept, error) {
	dept := &model.Dept{
		ParentID: req.ParentID,
		Name:     req.Name,
		Sort:     req.Sort,
		Leader:   req.Leader,
		Phone:    req.Phone,
		Email:    req.Email,
		Status:   req.Status,
	}

	if dept.Status == 0 {
		dept.Status = 1
	}

	if err := c.repo.Create(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

// Update 更新部门
// @Summary 更新部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param id path int true "部门ID"
// @Param request body UpdateRequest true "更新部门请求"
// @Success 200 {object} response.Response
// @Router /depts/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id := parseInt64(ctx.Params("id"))
	if id == 0 {
		return response.BadRequest(ctx, "invalid dept id")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	dept, err := c.update(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, dept)
}

// update 更新部门业务逻辑
func (c *Controller) update(ctx context.Context, id int64, req *UpdateRequest) (*model.Dept, error) {
	dept, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, errors.NotFound("dept")
	}

	if req.Name != "" {
		dept.Name = req.Name
	}
	if req.ParentID > 0 {
		dept.ParentID = req.ParentID
	}
	if req.Sort > 0 {
		dept.Sort = req.Sort
	}
	if req.Leader != "" {
		dept.Leader = req.Leader
	}
	if req.Phone != "" {
		dept.Phone = req.Phone
	}
	if req.Email != "" {
		dept.Email = req.Email
	}
	if req.Status > 0 {
		dept.Status = req.Status
	}

	if err := c.repo.Update(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

// Delete 删除部门
// @Summary 删除部门
// @Tags 部门管理
// @Param id path int true "部门ID"
// @Success 200 {object} response.Response
// @Router /depts/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id := parseInt64(ctx.Params("id"))
	if id == 0 {
		return response.BadRequest(ctx, "invalid dept id")
	}

	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Get 获取部门
// @Summary 获取部门详情
// @Tags 部门管理
// @Param id path int true "部门ID"
// @Success 200 {object} response.Response
// @Router /depts/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id := parseInt64(ctx.Params("id"))
	if id == 0 {
		return response.BadRequest(ctx, "invalid dept id")
	}

	dept, err := c.repo.FindByID(ctx.UserContext(), id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dept == nil {
		return response.NotFound(ctx, "dept not found")
	}

	return response.Success(ctx, dept)
}

// List 部门列表
// @Summary 部门列表
// @Tags 部门管理
// @Param name query string false "名称"
// @Param status query int false "状态"
// @Success 200 {object} response.Response
// @Router /depts [get]
func (c *Controller) List(ctx *fiber.Ctx) error {
	var req ListRequest
	if err := ctx.QueryParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	depts, err := c.list(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, depts)
}

// list 部门列表业务逻辑
func (c *Controller) list(ctx context.Context, req *ListRequest) ([]model.Dept, error) {
	conditions := make(map[string]interface{})
	if req.Status != nil {
		conditions["status"] = *req.Status
	}

	return c.repo.Find(ctx, conditions)
}

// GetTree 获取部门树
// @Summary 获取部门树
// @Tags 部门管理
// @Success 200 {object} response.Response
// @Router /depts/tree [get]
func (c *Controller) GetTree(ctx *fiber.Ctx) error {
	tree, err := c.getTree(ctx.UserContext())
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, tree)
}

// getTree 获取部门树业务逻辑
func (c *Controller) getTree(ctx context.Context) ([]*model.Dept, error) {
	depts, err := c.repo.FindAllEnabled(ctx)
	if err != nil {
		return nil, err
	}

	return buildTree(depts, 0), nil
}

// buildTree 构建树形结构
func buildTree(depts []model.Dept, parentID int64) []*model.Dept {
	var tree []*model.Dept
	for i := range depts {
		if depts[i].ParentID == parentID {
			dept := &depts[i]
			dept.Children = buildTree(depts, dept.ID)
			tree = append(tree, dept)
		}
	}
	return tree
}

// 辅助函数
func parseInt64(s string) int64 {
	var id int64
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			id = id*10 + int64(ch-'0')
		}
	}
	return id
}
