package auth

import (
	"fmt"

	pkgAuth "github.com/goback/pkg/auth"
	"github.com/goback/pkg/errors"
	"github.com/goback/pkg/response"
	"github.com/goback/pkg/router"
	"github.com/goback/services/user/internal/model"
	"github.com/goback/services/user/internal/user"
	"github.com/gofiber/fiber/v2"
)

// Controller 认证控制器
type Controller struct {
	router.BaseController
	UserCtrl   *user.Controller
	JWTManager *pkgAuth.JWTManager
}

// Prefix 返回路由前缀
func (c *Controller) Prefix() string {
	return "/auth"
}

// Routes 返回路由配置
func (c *Controller) Routes(middlewares map[string]fiber.Handler) []router.Route {
	return []router.Route{
		{Method: "POST", Path: "/login", Handler: c.login},
		{Method: "POST", Path: "/register", Handler: c.register},
		{Method: "POST", Path: "/logout", Handler: c.logout, Middlewares: &[]fiber.Handler{middlewares["jwt"]}},
		{Method: "POST", Path: "/refresh", Handler: c.refreshToken},
	}
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
	user, err := c.UserCtrl.GetByUsername(req.Username)
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
	token, err := c.JWTManager.CreateTokenInfo(user.ID, user.Username, user.RoleID, roleCode)
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
	newToken, err := c.JWTManager.RefreshToken(token)
	if err != nil {
		return response.Unauthorized(ctx, err.Error())
	}
	return response.Success(ctx, &pkgAuth.TokenInfo{
		AccessToken: newToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(c.JWTManager.GetExpireIn().Seconds()),
	})
}
