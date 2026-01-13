package auth

import (
	"errors"
	"time"

	"github.com/goback/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrTokenNotValidYet = errors.New("token not valid yet")
	ErrTokenMalformed   = errors.New("token is malformed")
	ErrTokenInvalid     = errors.New("token is invalid")
)

// Claims JWT声明
type Claims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	RoleID   int64  `json:"roleId"`
	RoleCode string `json:"roleCode"`
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secret   []byte
	issuer   string
	expireIn time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(cfg *config.JWTConfig) *JWTManager {
	return &JWTManager{
		secret:   []byte(cfg.Secret),
		issuer:   cfg.Issuer,
		expireIn: time.Duration(cfg.Expire) * time.Second,
	}
}

// GenerateToken 生成Token
func (m *JWTManager) GenerateToken(userID int64, username string, roleID int64, roleCode string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RoleID:   roleID,
		RoleCode: roleCode,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expireIn)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseToken 解析Token
func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotValidYet
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenMalformed
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// RefreshToken 刷新Token
func (m *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil && !errors.Is(err, ErrTokenExpired) {
		return "", err
	}

	// 如果token过期不超过7天,可以刷新
	if claims != nil {
		return m.GenerateToken(claims.UserID, claims.Username, claims.RoleID, claims.RoleCode)
	}

	return "", ErrTokenInvalid
}

// GetExpireIn 获取过期时间
func (m *JWTManager) GetExpireIn() time.Duration {
	return m.expireIn
}

// TokenInfo Token信息
type TokenInfo struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int64  `json:"expiresIn"`
}

// CreateTokenInfo 创建Token信息
func (m *JWTManager) CreateTokenInfo(userID int64, username string, roleID int64, roleCode string) (*TokenInfo, error) {
	token, err := m.GenerateToken(userID, username, roleID, roleCode)
	if err != nil {
		return nil, err
	}

	return &TokenInfo{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(m.expireIn.Seconds()),
	}, nil
}
