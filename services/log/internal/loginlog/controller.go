package loginlog

import (
	"context"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/services/log/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 登录日志控制器
type Controller struct {
	repo       Repository
	collection *dal.Collection[model.LoginLog]
}

// NewController 创建登录日志控制器
func NewController(repo Repository) *Controller {
	return &Controller{
		repo: repo,
		collection: dal.NewCollection[model.LoginLog](repo.DB()).
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
	g := r.Group("/login-logs", jwtMiddleware)
	g.Get("", c.list)
	g.Delete("/:ids", c.delete)
	g.Delete("/clear", c.clear)
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

func (c *Controller) delete(ctx *fiber.Ctx) error {
	ids, err := dal.ParseInt64IDs(ctx.Params("ids"))
	if err != nil {
		return response.BadRequest(ctx, "无效的ID格式")
	}
	if err := c.repo.DeleteBatch(ctx.UserContext(), ids); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) clear(ctx *fiber.Ctx) error {
	if err := c.collection.TruncateHard(ctx.UserContext()); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// Create 创建登录日志
func (c *Controller) Create(ctx context.Context, log *model.LoginLog) error {
	return c.repo.Create(ctx, log)
}
