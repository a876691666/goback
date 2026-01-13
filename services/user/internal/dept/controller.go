package dept

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/user/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 部门控制器
type Controller struct{}

func NewController() *Controller { return &Controller{} }

// FindAllEnabled 查找所有启用的部门
func (c *Controller) FindAllEnabled() ([]model.Dept, error) {
	return model.Depts.GetFullList(&dal.ListParams{
		Filter: "status=1",
	})
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	g := r.Group("/depts", jwtMiddleware)
	g.Post("", c.create)
	g.Put("/:id", c.update)
	g.Delete("/:id", c.delete)
	g.Get("/:id", c.get)
	g.Get("", c.list)
	g.Get("/tree", c.getTree)
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	dept, err := c.doCreate(&req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dept)
}

func (c *Controller) doCreate(req *CreateRequest) (*model.Dept, error) {
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
	if err := model.Depts.Create(dept); err != nil {
		return nil, err
	}
	return dept, nil
}

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的部门ID")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	dept, err := c.doUpdate(id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, dept)
}

func (c *Controller) doUpdate(id int64, req *UpdateRequest) (*model.Dept, error) {
	dept, err := model.Depts.GetOne(id)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, errors.NotFound("部门")
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

	if err := model.Depts.Save(dept); err != nil {
		return nil, err
	}
	return dept, nil
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的部门ID")
	}
	if err := model.Depts.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的部门ID")
	}
	dept, err := model.Depts.GetOne(id)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if dept == nil {
		return response.NotFound(ctx, "部门不存在")
	}
	return response.Success(ctx, dept)
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	depts, err := model.Depts.GetFullList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, depts)
}

func (c *Controller) getTree(ctx *fiber.Ctx) error {
	tree, err := c.doGetTree()
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, tree)
}

func (c *Controller) doGetTree() ([]*model.Dept, error) {
	depts, err := c.FindAllEnabled()
	if err != nil {
		return nil, err
	}
	return buildTree(depts, 0), nil
}

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

