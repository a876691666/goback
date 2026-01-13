package operationlog

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/services/log/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 操作日志控制器
type Controller struct{}

func NewController() *Controller { return &Controller{} }

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	g := r.Group("/operation-logs", jwtMiddleware)
	g.Get("", c.list)
	g.Delete("/:ids", c.delete)
	g.Delete("/clear", c.clear)
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	result, err := model.OperationLogs.GetList(params)
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
	if err := model.OperationLogs.DeleteByIDs(ids); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) clear(ctx *fiber.Ctx) error {
	if err := model.OperationLogs.Truncate(); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// CreateLog 创建操作日志
func (c *Controller) CreateLog(log *model.OperationLog) error {
	return model.OperationLogs.Create(log)
}
