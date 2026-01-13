package loginlog

import (
	"context"
	"strconv"
	"strings"

	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/response"
	"github.com/goback/services/log/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 登录日志控制器
type Controller struct {
	repo Repository
}

// NewController 创建登录日志控制器
func NewController(repo Repository) *Controller {
	return &Controller{repo: repo}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	loginLog := r.Group("/login-logs", jwtMiddleware)
	loginLog.Get("", c.List)
	loginLog.Delete("/:ids", c.Delete)
	loginLog.Delete("/clear", c.Clear)
}

// List 登录日志列表
// @Summary 登录日志列表
// @Tags 登录日志
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param username query string false "用户名"
// @Param ip query string false "IP地址"
// @Param status query int false "状态"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} response.Response
// @Router /login-logs [get]
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

// list 登录日志列表业务逻辑
func (c *Controller) list(ctx context.Context, req *ListRequest) (*dal.PagedResult[model.LoginLog], error) {
	pagination := dal.NewPagination(req.Page, req.PageSize)
	qb := dal.NewQueryBuilder[model.LoginLog](c.repo.DB())

	if req.Username != "" {
		qb.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.IP != "" {
		qb.Where("ip LIKE ?", "%"+req.IP+"%")
	}
	if req.Status != nil {
		qb.Where("status = ?", *req.Status)
	}
	if req.StartTime != "" {
		qb.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		qb.Where("created_at <= ?", req.EndTime)
	}

	qb.Order("id DESC")

	return qb.Paged(ctx, pagination)
}

// Delete 删除登录日志
// @Summary 删除登录日志
// @Tags 登录日志
// @Param ids path string true "ID列表,逗号分隔"
// @Success 200 {object} response.Response
// @Router /login-logs/{ids} [delete]
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

// Clear 清空登录日志
// @Summary 清空登录日志
// @Tags 登录日志
// @Success 200 {object} response.Response
// @Router /login-logs/clear [delete]
func (c *Controller) Clear(ctx *fiber.Ctx) error {
	if err := c.repo.DB().WithContext(ctx.UserContext()).Exec("TRUNCATE TABLE sys_login_log").Error; err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Create 创建登录日志（供内部调用）
func (c *Controller) Create(ctx context.Context, log *model.LoginLog) error {
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
