package auth

import (
	"fmt"

	"github.com/goback/pkg/app/apis"
	"github.com/goback/pkg/app/core"
	pkgAuth "github.com/goback/pkg/auth"
	"github.com/goback/services/user/internal/user"
)

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

// Login 登录（需要JWTManager的闭包）
func Login(jwtManager *pkgAuth.JWTManager) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req LoginRequest
		if err := e.BindBody(&req); err != nil {
			return apis.Error(e, 400, err.Error())
		}

		u, err := user.GetByUsername(req.Username)
		if err != nil {
			return apis.Error(e, 500, err.Error())
		}
		if u == nil {
			return apis.Error(e, 401, "用户名或密码错误")
		}
		if !pkgAuth.CheckPassword(req.Password, u.Password) {
			return apis.Error(e, 401, "用户名或密码错误")
		}
		if u.Status != 1 {
			return apis.Error(e, 403, "用户已被禁用")
		}

		var roleCode string
		if u.Role != nil {
			roleCode = u.Role.Code
		} else {
			roleCode = fmt.Sprintf("role_%d", u.RoleID)
		}

		token, err := jwtManager.CreateTokenInfo(u.ID, u.Username, u.RoleID, roleCode)
		if err != nil {
			return apis.Error(e, 500, "生成令牌失败: "+err.Error())
		}

		return apis.Success(e, &LoginResponse{
			Token: token,
			UserInfo: &UserInfo{
				ID:       u.ID,
				Username: u.Username,
				Nickname: u.Nickname,
				Email:    u.Email,
				Phone:    u.Phone,
				Avatar:   u.Avatar,
				RoleID:   u.RoleID,
				RoleCode: roleCode,
			},
		})
	}
}

// Register 注册
func Register(e *core.RequestEvent) error {
	var req user.CreateRequest
	if err := e.BindBody(&req); err != nil {
		return apis.Error(e, 400, err.Error())
	}
	// TODO: 实现注册逻辑
	return apis.Error(e, 501, "功能未实现")
}

// Logout 登出
func Logout(e *core.RequestEvent) error {
	return apis.Success(e, nil)
}

// RefreshToken 刷新令牌（需要JWTManager的闭包）
func RefreshToken(jwtManager *pkgAuth.JWTManager) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := e.Request.Header.Get("Authorization")
		if token == "" {
			return apis.Error(e, 401, "需要令牌")
		}
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		newToken, err := jwtManager.RefreshToken(token)
		if err != nil {
			return apis.Error(e, 401, err.Error())
		}
		return apis.Success(e, &pkgAuth.TokenInfo{
			AccessToken: newToken,
			TokenType:   "Bearer",
			ExpiresIn:   int64(jwtManager.GetExpireIn().Seconds()),
		})
	}
}
