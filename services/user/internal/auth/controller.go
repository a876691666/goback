package auth

import (
	"context"
	"fmt"

	pkgAuth "github.com/goback/pkg/auth"
	"github.com/goback/pkg/config"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/services/user/internal/model"
	"github.com/goback/services/user/internal/user"
	"github.com/gofiber/fiber/v2"
)

// Controller 认证控制器
type Controller struct {
	userCtrl   *user.Controller
	jwtManager *pkgAuth.JWTManager
}

// NewController 创建认证控制器
func NewController(userCtrl *user.Controller, jwtCfg *config.JWTConfig) *Controller {
	return &Controller{
		userCtrl:   userCtrl,
		jwtManager: pkgAuth.NewJWTManager(jwtCfg),
	}
}

// RegisterRoutes 注册路由
func (c *Controller) RegisterRoutes(r fiber.Router, jwtMiddleware fiber.Handler) {
	auth := r.Group("/auth")
	auth.Post("/login", c.Login)
	auth.Post("/register", c.Register)
	auth.Post("/logout", jwtMiddleware, c.Logout)
	auth.Post("/refresh", c.RefreshToken)
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token    *pkgAuth.TokenInfo `json:"token"`
	UserInfo *UserInfo          `json:"userInfo"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	RoleID   int64  `json:"roleId"`
	RoleCode string `json:"roleCode"`
	DeptID   int64  `json:"deptId"`
}

// Login 用户登录
// @Summary 用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} response.Response
// @Router /auth/login [post]
func (c *Controller) Login(ctx *fiber.Ctx) error {
	var req LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	result, err := c.login(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 401, err.Error())
	}

	return response.Success(ctx, result)
}

// login 登录业务逻辑
func (c *Controller) login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user, err := c.userCtrl.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.ErrInvalidCredential
	}

	if !pkgAuth.CheckPassword(req.Password, user.Password) {
		return nil, errors.ErrInvalidCredential
	}

	if user.Status != 1 {
		return nil, errors.Forbidden("user is disabled")
	}

	// 获取角色信息
	var roleCode string
	if user.Role != nil {
		roleCode = user.Role.Code
	} else {
		roleCode = fmt.Sprintf("role_%d", user.RoleID)
	}

	// 生成Token
	token, err := c.jwtManager.CreateTokenInfo(user.ID, user.Username, user.RoleID, roleCode)
	if err != nil {
		return nil, errors.Wrap(err, 500, "failed to generate token")
	}

	return &LoginResponse{
		Token: token,
		UserInfo: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Email:    user.Email,
			Phone:    user.Phone,
			Avatar:   user.Avatar,
			RoleID:   user.RoleID,
			RoleCode: roleCode,
			DeptID:   user.DeptID,
		},
	}, nil
}

// Register 用户注册
// @Summary 用户注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body user.CreateRequest true "注册请求"
// @Success 200 {object} response.Response
// @Router /auth/register [post]
func (c *Controller) Register(ctx *fiber.Ctx) error {
	var req user.CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}

	u, err := c.register(ctx.UserContext(), &req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}

	return response.Success(ctx, u)
}

// register 注册业务逻辑（复用用户创建逻辑）
func (c *Controller) register(ctx context.Context, req *user.CreateRequest) (*model.User, error) {
	// 注册时设置默认角色和状态
	if req.Status == 0 {
		req.Status = 1
	}
	// 可以在这里设置默认角色ID
	// req.RoleID = defaultRoleID

	// 使用userCtrl内部的Create方法会更好，但这里直接调用HTTP handler不合适
	// 所以我们需要在user.Controller中暴露一个Create方法供内部调用
	return nil, errors.ErrNotImplemented
}

// Logout 用户登出
// @Summary 用户登出
// @Tags 认证
// @Success 200 {object} response.Response
// @Router /auth/logout [post]
func (c *Controller) Logout(ctx *fiber.Ctx) error {
	// TODO: 实现Token黑名单机制
	return response.Success(ctx, nil)
}

// RefreshToken 刷新Token
// @Summary 刷新Token
// @Tags 认证
// @Success 200 {object} response.Response
// @Router /auth/refresh [post]
func (c *Controller) RefreshToken(ctx *fiber.Ctx) error {
	token := ctx.Get("Authorization")
	if token == "" {
		return response.Unauthorized(ctx, "token required")
	}

	// 去掉Bearer前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	newToken, err := c.jwtManager.RefreshToken(token)
	if err != nil {
		return response.Unauthorized(ctx, err.Error())
	}

	return response.Success(ctx, &pkgAuth.TokenInfo{
		AccessToken: newToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(c.jwtManager.GetExpireIn().Seconds()),
	})
}
