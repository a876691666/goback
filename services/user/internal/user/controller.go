package user

import (
	"github.com/goback/pkg/auth"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/dal"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/user/internal/model"
	"github.com/gofiber/fiber/v2"
)

// Controller 用户控制器
type Controller struct {
	jwtManager *auth.JWTManager
}

// NewController 创建用户控制器
func NewController(jwtCfg *config.JWTConfig) *Controller {
	return &Controller{
		jwtManager: auth.NewJWTManager(jwtCfg),
	}
}

// GetJWTManager 获取JWT管理器
func (c *Controller) GetJWTManager() *auth.JWTManager {
	return c.jwtManager
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	users := r.Group("/users", jwtMiddleware)
	users.Post("", c.create)
	users.Put("/:id", c.update)
	users.Delete("/:id", c.delete)
	users.Get("/:id", c.get)
	users.Get("", c.list)
	users.Put("/:id/password/reset", c.resetPassword)

	profile := r.Group("/profile", jwtMiddleware)
	profile.Get("", c.getProfile)
	profile.Put("", c.updateProfile)
	profile.Put("/password", c.changePassword)
}

func (c *Controller) create(ctx *fiber.Ctx) error {
	var req CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	user, err := c.doCreate(&req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, user)
}

func (c *Controller) doCreate(req *CreateRequest) (*model.User, error) {
	exists, err := model.Users.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.Duplicate("用户名")
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, errors.Wrap(err, 500, "密码加密失败")
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

	if err := model.Users.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (c *Controller) update(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的用户ID")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	user, err := c.doUpdate(id, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, user)
}

func (c *Controller) doUpdate(id int64, req *UpdateRequest) (*model.User, error) {
	user, err := model.Users.GetOne(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.NotFound("用户")
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

	if err := model.Users.Save(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (c *Controller) delete(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的用户ID")
	}
	if err := model.Users.DeleteByID(id); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) get(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的用户ID")
	}
	user, err := model.Users.GetByIDWithPreload(id, "Role")
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if user == nil {
		return response.NotFound(ctx, "用户不存在")
	}
	return response.Success(ctx, user)
}

func (c *Controller) list(ctx *fiber.Ctx) error {
	params, err := dal.BindQuery(ctx)
	if err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	if params.Expand == "" {
		params.Expand = "Role"
	}
	result, err := model.Users.GetList(params)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.SuccessPage(ctx, result.Items, result.TotalItems, result.Page, result.PerPage)
}

func (c *Controller) getProfile(ctx *fiber.Ctx) error {
	userID := getUserID(ctx)
	if userID == 0 {
		return response.Unauthorized(ctx, "")
	}
	user, err := model.Users.GetByIDWithPreload(userID, "Role")
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, user)
}

func (c *Controller) updateProfile(ctx *fiber.Ctx) error {
	userID := getUserID(ctx)
	if userID == 0 {
		return response.Unauthorized(ctx, "")
	}
	var req UpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	req.RoleID = 0
	req.Status = 0
	user, err := c.doUpdate(userID, &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, user)
}

func (c *Controller) changePassword(ctx *fiber.Ctx) error {
	userID := getUserID(ctx)
	if userID == 0 {
		return response.Unauthorized(ctx, "")
	}
	var req ChangePasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	if err := c.doChangePassword(userID, &req); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

func (c *Controller) doChangePassword(userID int64, req *ChangePasswordRequest) error {
	user, err := model.Users.GetOne(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.NotFound("用户")
	}
	if !auth.CheckPassword(req.OldPassword, user.Password) {
		return errors.BadRequest("旧密码错误")
	}
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return errors.Wrap(err, 500, "密码加密失败")
	}
	return model.Users.UpdateByID(userID, map[string]any{"password": hashedPassword})
}

func (c *Controller) resetPassword(ctx *fiber.Ctx) error {
	id, err := dal.ParseInt64ID(ctx.Params("id"))
	if err != nil {
		return response.BadRequest(ctx, "无效的用户ID")
	}
	hashedPassword, err := auth.HashPassword("123456")
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	if err := model.Users.UpdateByID(id, map[string]any{"password": hashedPassword}); err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, nil)
}

// GetByUsername 根据用户名获取用户
func (c *Controller) GetByUsername(username string) (*model.User, error) {
	return model.Users.GetByUsername(username)
}

// GetByID 根据ID获取用户
func (c *Controller) GetByID(id int64) (*model.User, error) {
	return model.Users.GetByIDWithPreload(id, "Role")
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
