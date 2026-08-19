package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	SecretKey      []byte
	accessExpires  = time.Hour
	refreshExpires = 7 * 24 * time.Hour
)

/*
MyCustomClaims 是系统 JWT 中携带的自定义声明。
除标准过期时间和签发方外，还包含用户 ID、用户名和角色。
*/
type MyCustomClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

/*
TokenPair 是登录和刷新 token 接口返回的令牌对。
AccessToken 用于接口鉴权，RefreshToken 用于续期。
*/
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

/*
Configure 设置 JWT 签名密钥、access token 和 refresh token 的有效期。
服务启动时会根据 config.yaml 中的 JWT 配置调用该函数。
*/
func Configure(secretKey string, expires, refresh time.Duration) {
	SecretKey = []byte(secretKey)
	accessExpires = expires
	refreshExpires = refresh
}

/*
GenerateTokenPair 生成 access token 和 refresh token。
两个 token 都使用 HS256 签名，并携带相同的用户身份信息。
*/
func GenerateTokenPair(userID uint, username, role string) (*TokenPair, error) {
	// access token 生命周期较短，用于日常 API 鉴权。
	accessClaims := MyCustomClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessExpires)),
			Issuer:    "sql-audit-platform",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(SecretKey)
	if err != nil {
		return nil, err
	}

	// refresh token 生命周期较长，仅用于换取新的 token pair。
	refreshClaims := MyCustomClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshExpires)),
			Issuer:    "sql-audit-platform",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(SecretKey)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
	}, nil
}

/*
RefreshToken 解析 refresh token 并重新生成 token pair。
refresh token 无效或过期时返回解析错误。
*/
func RefreshToken(refreshToken string) (*TokenPair, error) {
	claims, err := ParseToken(refreshToken)
	if err != nil {
		return nil, err
	}
	return GenerateTokenPair(claims.UserID, claims.Username, claims.Role)
}

/*
ParseToken 解析并校验 JWT。
校验通过时返回自定义 claims，否则返回 invalid token 或底层解析错误。
*/
func ParseToken(tokenString string) (*MyCustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*MyCustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

/*
GetUserIDFromContext 从 gin.Context 兼容对象中获取用户 ID。
JWTAuth 中间件会把 userID 写入上下文，缺失时返回 0。
*/
func GetUserIDFromContext(c interface{}) uint {
	type ContextWithValue interface {
		Value(key interface{}) interface{}
	}

	if ctx, ok := c.(ContextWithValue); ok {
		if userID, exists := ctx.Value("userID").(uint); exists {
			return userID
		}
	}
	return 0
}

/*
GetUsernameFromContext 从 gin.Context 兼容对象中获取用户名。
缺失时返回空字符串。
*/
func GetUsernameFromContext(c interface{}) string {
	type ContextWithValue interface {
		Value(key interface{}) interface{}
	}

	if ctx, ok := c.(ContextWithValue); ok {
		if username, exists := ctx.Value("username").(string); exists {
			return username
		}
	}
	return ""
}

/*
GetRoleFromContext 从 gin.Context 兼容对象中获取用户角色。
缺失时返回空字符串。
*/
func GetRoleFromContext(c interface{}) string {
	type ContextWithValue interface {
		Value(key interface{}) interface{}
	}

	if ctx, ok := c.(ContextWithValue); ok {
		if role, exists := ctx.Value("role").(string); exists {
			return role
		}
	}
	return ""
}
