package router

import (
	"net/http"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
CORS 返回跨域处理中间件。
当前允许任意来源访问，并开放常用请求头和 GET/POST/PUT/DELETE/OPTIONS 方法；
浏览器预检 OPTIONS 请求会直接返回 204，不再进入后续业务 Handler。
*/
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

/*
Recovery 返回全局异常恢复中间件。
当后续 Handler 或中间件发生 panic 时，这里统一记录错误日志，
并向客户端返回标准 500 响应，避免进程因为单个请求异常而退出。
*/
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Log.Error("Panic recovered", zap.Any("error", err))
				response.FailWithStatus(c, http.StatusInternalServerError, response.CodeError, "Internal Server Error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

/*
Logger 返回请求日志中间件。
它会记录非健康检查请求的方法、路径、响应状态码和耗时，
用于排查接口访问情况和慢请求问题。
*/
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 健康检查请求频率较高且信息价值较低，跳过日志避免噪音。
		if c.Request.URL.Path == "/api/health" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		logger.Log.Info("Request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
		)
	}
}

/*
JWTAuth 返回 JWT 鉴权中间件。
它从 Authorization 请求头读取 Bearer Token，解析成功后把用户 ID、用户名和角色
写入 gin.Context，供后续 Handler 或权限中间件继续使用。
*/
func JWTAuth(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "Authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "Invalid authorization header format")
			c.Abort()
			return
		}

		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "Invalid or expired token")
			c.Abort()
			return
		}

		// 校验 token 归属用户是否仍然有效，确保禁用后已签发的 token 立即失效。
		user, err := userService.GetByID(claims.UserID)
		if err != nil || user == nil {
			response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "User not found")
			c.Abort()
			return
		}
		if user.Status != "" && user.Status != "enabled" {
			response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "User disabled")
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", user.Role) // 使用查库得到的最新角色覆盖 token 快照，让角色变更即时生效。
		c.Next()
	}
}

/*
isAdminRole 将 gin.Context 中取出的 role 值转换为字符串后判断是否为管理员角色。
单独拆出该函数可以隔离类型断言逻辑，避免 AdminOnly 中间件直接依赖具体存储类型。
*/
func isAdminRole(role any) bool {
	roleValue, ok := role.(string)
	if !ok {
		return false
	}
	return model.IsAdminRole(roleValue)
}

/*
AdminOnly 返回管理员权限校验中间件。
它依赖 JWTAuth 预先写入的 role 上下文值，只有管理员角色可以继续访问后续 Handler，
否则返回 403 并终止当前请求链路。
*/
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.FailWithStatus(c, http.StatusForbidden, response.CodeError, "Role not found")
			c.Abort()
			return
		}

		if !isAdminRole(role) {
			response.FailWithStatus(c, http.StatusForbidden, response.CodeError, "Admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
