package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const JWTSecret = "your-secret-key-change-in-production"

type Claims struct {
	UserID   string `json:"user_id"`
	UserName string `json:"username"`
	jwt.RegisteredClaims
}

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1.获取认证请求头
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少令牌",
			})
		}
		// 2. 解析 "Bearer <token>" 格式
		parts := strings.SplitN(auth, ".", -1)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "令牌格式错误",
			})
		}
		tokenString := parts[1]

		// 3. 验证令牌有效
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			//断言验证
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return ([]byte)(JWTSecret), nil
		})
		if err != nil || token.Valid != true {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "令牌无效或已过期",
			})
		}

		// 4. 自定义字段头部注入
		c.Header("X-User-ID", claims.UserID)
		c.Header("X-Username", claims.UserName)

		// 5. 继续处理请求
		c.Next()
	}
}

// OptionalAuthMiddleware 可选的认证中间件（某些路由不需要认证）
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {}
}
