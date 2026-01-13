package operationlog

import (
	"context"
	"strconv"
	"strings"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/services/log/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 操作日志控制器
type Controller struct {
	repo       Repository
	collection *dal.Collection[model.OperationLog]
}

// NewController 创建操作日志控制器
func NewController(repo Repository) *Controller {
	// 创建 Collection 查询器，配置字段映射和默认值
	collection := dal.NewCollection[model.OperationLog](repo.DB()).
		WithDefaultSort("-id").
		WithMaxPerPage(100).
		WithFieldAlias(map[string]string{
			"createdAt": "created_at",
			"updatedAt": "updated_at",
		})

	return &Controller{
		repo:       repo,
		collection: collection,
	}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	opLog := r.Group("/operation-logs", jwtMiddleware)
	opLog.Get("", c.List)
	opLog.Delete("/:ids", c.Delete)
	opLog.Delete("/clear", c.Clear)
}

// List 操作日志列表
// @Summary 操作日志列表
// @Tags 操作日志
// @Param filter query string false "SSQL过滤条件 例如: username~\"admin\"&&status=1"
// @Param fields query string false "返回字段 例如: id,username,module,action"
// @Param sort query string false "排序 例如: -created_at,id (- 表示降序)"
// @Param page query int false "页码"
// @Param perPage query int false "每页数量"
// @Param skipTotal query bool false "是否跳过总数统计"
// @Success 200 {object} response.Response
// @Router /operation-logs [get]
func (c *Controller) List(ctx *fiber.Ctx) error {
	// 绑定查询参数
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	// 使用 Collection API 查询
	result, err := c.collection.GetList(ctx.UserContext(), params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

// Delete 删除操作日志
// @Summary 删除操作日志
// @Tags 操作日志
// @Param ids path string true "ID列表,逗号分隔"
// @Success 200 {object} response.Response
// @Router /operation-logs/{ids} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	idsStr := ctx.Params("ids")
	ids, err := parseIDs(idsStr)
	if err != nil {
		return response.BadRequest(ctx, "无效的ID格式")
	}

	if err := c.repo.DeleteBatch(ctx.UserContext(), ids); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Clear 清空操作日志
// @Summary 清空操作日志
// @Tags 操作日志
// @Success 200 {object} response.Response
// @Router /operation-logs/clear [delete]
func (c *Controller) Clear(ctx *fiber.Ctx) error {
	if err := c.repo.DB().WithContext(ctx.UserContext()).Exec("TRUNCATE TABLE sys_operation_log").Error; err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Create 创建操作日志（供内部调用）
func (c *Controller) Create(ctx context.Context, log *model.OperationLog) error {
	return c.repo.Create(ctx, log)
}

// parseIDs 解析ID列表
func parseIDs(idsStr string) ([]int64, error) {
	parts := strings.Split(idsStr, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
