// internal/middleware/auth.go
package middleware

import (
	"strings"

	"sentinel-agent-go/internal/pkg"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 登录鉴权中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header, 格式通常为: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			pkg.Error(c, 401, "未登录，请先登录")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			pkg.Error(c, 401, "认证格式错误")
			c.Abort()
			return
		}

		// 2. 解析并校验 Token
		claims, err := pkg.ParseToken(parts[1])
		if err != nil {
			pkg.Error(c, 401, "Token 已过期或无效")
			c.Abort()
			return
		}

		// 3. 将用户信息存入上下文，方便后续 Controller 使用
		c.Set("currentUserID", claims.UserID)
		c.Set("currentUserAccount", claims.UserAccount)
		c.Set("currentUserRole", claims.UserRole)

		c.Next()
	}
}
