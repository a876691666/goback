package loginlog

import (
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/log/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 登录日志控制器
type Controller struct {
	router.BaseController
}

// Prefix 返回路由前缀
func (c *Controller) Prefix() string {
	return "/login-logs"
}

// Routes 返回路由配置
func (c *Controller) Routes(middlewares map[string]fiber.Handler) []router.Route {
	return []router.Route{
		{Method: "GET", Path: "", Handler: c.list, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "DELETE", Path: "/:ids", Handler: c.delete, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "DELETE", Path: "/clear", Handler: c.clear, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
	}
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	result, err := model.LoginLogs.GetList(params)
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
	if err := model.LoginLogs.DeleteByIDs(ids); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) clear(ctx *fiber.Ctx) error {
	if err := model.LoginLogs.Truncate(); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// CreateLog 创建登录日志
func (c *Controller) CreateLog(log *model.LoginLog) error {
	return model.LoginLogs.Create(log)
}
