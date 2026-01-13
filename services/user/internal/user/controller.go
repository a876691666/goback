package user

import (
	"context"
	"fmt"

	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/user/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 用户控制器（融合了Service层）
type Controller struct {
	repo       Repository
	jwtManager *auth.JWTManager
}

// NewController 创建用户控制器
func NewController(repo Repository, jwtCfg *config.JWTConfig) *Controller {
	return &Controller{
		repo:       repo,
		jwtManager: auth.NewJWTManager(jwtCfg),
	}
}

// GetJWTManager 获取JWT管理器
func (c *Controller) GetJWTManager() *auth.JWTManager {
	return c.jwtManager
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	// 需要认证的接口
	users := r.Group("/users", jwtMiddleware)
	users.Post("", c.Create)
	users.Put("/:id", c.Update)
	users.Delete("/:id", c.Delete)
	users.Get("/:id", c.Get)
	users.Get("", c.List)
	users.Put("/:id/password/reset", c.ResetPassword)

	// 个人中心
	profile := r.Group("/profile", jwtMiddleware)
	profile.Get("", c.GetProfile)
	profile.Put("", c.UpdateProfile)
	profile.Put("/password", c.ChangePassword)
}

// Create 创建用户
// @Summary 创建用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body CreateRequest true "创建用户请求"
// @Success 200 {object} response.Response
// @Router /users [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	user, err := c.create(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, user)
}

// create 创建用户业务逻辑
func (c *Controller) create(ctx context.Context, req *CreateRequest) (*model.User, error) {
	// 检查用户名是否存在
	existing, err := c.repo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Duplicate("username")
	}

	// 加密密码
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, errors.Wrap(err, 500, "failed to hash password")
	}

	user := &model.User{
		Username: req.Username,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Avatar:   req.Avatar,
		RoleID:   req.RoleID,
		DeptID:   req.DeptID,
		Status:   req.Status,
	}

	if user.Status == 0 {
		user.Status = 1
	}

	if err := c.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Update 更新用户
// @Summary 更新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body UpdateRequest true "更新用户请求"
// @Success 200 {object} response.Response
// @Router /users/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id := parseInt64(ctx.Params("id"))
	if id == 0 {
		return response.BadRequest(ctx, "invalid user id")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	user, err := c.update(ctx.UserContext(), id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, user)
}

// update 更新用户业务逻辑
func (c *Controller) update(ctx context.Context, id int64, req *UpdateRequest) (*model.User, error) {
	user, err := c.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.NotFound("user")
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.RoleID > 0 {
		user.RoleID = req.RoleID
	}
	if req.DeptID > 0 {
		user.DeptID = req.DeptID
	}
	if req.Status > 0 {
		user.Status = req.Status
	}

	if err := c.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Delete 删除用户
// @Summary 删除用户
// @Tags 用户管理
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /users/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id := parseInt64(ctx.Params("id"))
	if id == 0 {
		return response.BadRequest(ctx, "invalid user id")
	}

	if err := c.repo.Delete(ctx.UserContext(), id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// Get 获取用户
// @Summary 获取用户详情
// @Tags 用户管理
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /users/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id := parseInt64(ctx.Params("id"))
	if id == 0 {
		return response.BadRequest(ctx, "invalid user id")
	}

	user, err := c.repo.FindByID(ctx.UserContext(), id, dal.WithPreload("Role"))
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if user == nil {
		return response.NotFound(ctx, "user not found")
	}

	return response.Success(ctx, user)
}

// List 用户列表
// @Summary 用户列表
// @Tags 用户管理
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param username query string false "用户名"
// @Param nickname query string false "昵称"
// @Param status query int false "状态"
// @Success 200 {object} response.Response
// @Router /users [get]
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

// list 用户列表业务逻辑
func (c *Controller) list(ctx context.Context, req *ListRequest) (*dal.PagedResult[model.User], error) {
	pagination := dal.NewPagination(req.Page, req.PageSize)

	qb := dal.NewQueryBuilder[model.User](c.repo.DB())

	if req.Username != "" {
		qb.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Nickname != "" {
		qb.Where("nickname LIKE ?", "%"+req.Nickname+"%")
	}
	if req.Phone != "" {
		qb.Where("phone LIKE ?", "%"+req.Phone+"%")
	}
	if req.Status != nil {
		qb.Where("status = ?", *req.Status)
	}
	if req.RoleID != nil {
		qb.Where("role_id = ?", *req.RoleID)
	}
	if req.DeptID != nil {
		qb.Where("dept_id = ?", *req.DeptID)
	}

	if req.SSql != "" {
		qb.WhereSSql(req.SSql)
	}

	qb.Order("id DESC")
	qb.Preload("Role")

	return qb.Paged(ctx, pagination)
}

// GetProfile 获取个人信息
// @Summary 获取个人信息
// @Tags 个人中心
// @Success 200 {object} response.Response
// @Router /profile [get]
func (c *Controller) GetProfile(ctx *fiber.Ctx) error {
	userID := getUserID(ctx)
	if userID == 0 {
		return response.Unauthorized(ctx, "")
	}

	user, err := c.repo.FindByID(ctx.UserContext(), userID, dal.WithPreload("Role"))
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, user)
}

// UpdateProfile 更新个人信息
// @Summary 更新个人信息
// @Tags 个人中心
// @Accept json
// @Produce json
// @Param request body UpdateRequest true "更新信息请求"
// @Success 200 {object} response.Response
// @Router /profile [put]
func (c *Controller) UpdateProfile(ctx *fiber.Ctx) error {
	userID := getUserID(ctx)
	if userID == 0 {
		return response.Unauthorized(ctx, "")
	}

	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	// 不允许修改角色和状态
	req.RoleID = 0
	req.Status = 0

	user, err := c.update(ctx.UserContext(), userID, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, user)
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Tags 个人中心
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} response.Response
// @Router /profile/password [put]
func (c *Controller) ChangePassword(ctx *fiber.Ctx) error {
	userID := getUserID(ctx)
	if userID == 0 {
		return response.Unauthorized(ctx, "")
	}

	var req ChangePasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	if err := c.changePassword(ctx.UserContext(), userID, &req); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// changePassword 修改密码业务逻辑
func (c *Controller) changePassword(ctx context.Context, userID int64, req *ChangePasswordRequest) error {
	user, err := c.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.NotFound("user")
	}

	if !auth.CheckPassword(req.OldPassword, user.Password) {
		return errors.BadRequest("old password is incorrect")
	}

	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return errors.Wrap(err, 500, "failed to hash password")
	}

	return c.repo.UpdatePassword(ctx, userID, hashedPassword)
}

// ResetPassword 重置密码
// @Summary 重置用户密码
// @Tags 用户管理
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /users/{id}/password/reset [put]
func (c *Controller) ResetPassword(ctx *fiber.Ctx) error {
	id := parseInt64(ctx.Params("id"))
	if id == 0 {
		return response.BadRequest(ctx, "invalid user id")
	}

	// 默认重置密码为123456
	hashedPassword, err := auth.HashPassword("123456")
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	if err := c.repo.UpdatePassword(ctx.UserContext(), id, hashedPassword); err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, nil)
}

// GetByUsername 根据用户名获取用户（供内部调用）
func (c *Controller) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return c.repo.FindByUsername(ctx, username)
}

// GetByID 根据ID获取用户（供内部调用）
func (c *Controller) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return c.repo.FindByID(ctx, id, dal.WithPreload("Role"))
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

func getUserID(ctx *fiber.Ctx) int64 {
	userID := ctx.Locals("userId")
	if userID == nil {
		return 0
	}
	if id, ok := userID.(int64); ok {
		return id
	}
	return 0
}

// Ensure Controller doesn't have unused variable
var _ = fmt.Sprintf
