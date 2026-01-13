package auth

import (
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
	g := r.Group("/auth")
	g.Post("/login", c.login)
	g.Post("/register", c.register)
	g.Post("/logout", jwtMiddleware, c.logout)
	g.Post("/refresh", c.refreshToken)
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

func (c *Controller) login(ctx *fiber.Ctx) error {
	var req LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	result, err := c.doLogin(&req)
	if err != nil {
		return response.Error(ctx, 401, err.Error())
	}
	return response.Success(ctx, result)
}

func (c *Controller) doLogin(req *LoginRequest) (*LoginResponse, error) {
	user, err := c.userCtrl.GetByUsername(req.Username)
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
		return nil, errors.Forbidden("用户已被禁用")
	}
	var roleCode string
	if user.Role != nil {
		roleCode = user.Role.Code
	} else {
		roleCode = fmt.Sprintf("role_%d", user.RoleID)
	}
	token, err := c.jwtManager.CreateTokenInfo(user.ID, user.Username, user.RoleID, roleCode)
	if err != nil {
		return nil, errors.Wrap(err, 500, "生成令牌失败")
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

func (c *Controller) register(ctx *fiber.Ctx) error {
	var req user.CreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return response.ValidateError(ctx, err.Error())
	}
	u, err := c.doRegister(&req)
	if err != nil {
		return response.Error(ctx, 500, err.Error())
	}
	return response.Success(ctx, u)
}

func (c *Controller) doRegister(req *user.CreateRequest) (*model.User, error) {
	if req.Status == 0 {
		req.Status = 1
	}
	return nil, errors.ErrNotImplemented
}

func (c *Controller) logout(ctx *fiber.Ctx) error {
	return response.Success(ctx, nil)
}

func (c *Controller) refreshToken(ctx *fiber.Ctx) error {
	token := ctx.Get("Authorization")
	if token == "" {
		return response.Unauthorized(ctx, "需要令牌")
	}
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
