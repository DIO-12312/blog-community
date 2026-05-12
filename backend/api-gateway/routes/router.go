package routes

import (
	"net/http"
	"net/url"

	"blog-community/api-gateway/middleware"
	shared_middleware "blog-community/shared/middleware"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
)

// ServiceRegistry 服务注册表
var services = map[string]string{
	"user":        "http://localhost:8001",
	"article":     "http://localhost:8002",
	"interaction": "http://localhost:8003",
	"permission":  "http://localhost:8004",
	"search":      "http://localhost:8005",
}

// SetupRoutes 设置所有路由
func SetupRoutes(router *gin.Engine) {
	// 跨域中间件
	router.Use(shared_middleware.CORS())
	// 全局中间件
	router.Use(middleware.RateLimitMiddleware())

	// 不需要认证的路由
	setupPublicRoutes(router)

	// 需要认证的路由
	authenticated := router.Group("/")
	authenticated.Use(middleware.AuthMiddleware())
	setupPrivateRoutes(authenticated)

}

// setupPublicRoutes 设置公开路由
func setupPublicRoutes(router *gin.Engine) {
	// 用户注册、登录
	router.POST("/api/users/register", proxyTo("user"))
	router.POST("/api/users/login", proxyTo("user"))

	// 文章列表（可匿名查看）
	router.GET("/api/articles", middleware.OptionalAuthMiddleware(), proxyTo("article"))
	router.GET("/api/articles/:id", middleware.OptionalAuthMiddleware(), proxyTo("article"))
}

// setupPrivateRoutes 设置需要认证的路由
func setupPrivateRoutes(router *gin.RouterGroup) {
	// 用户相关
	router.GET("/api/users", proxyTo("user"))
	router.PUT("/api/users/:id", proxyTo("user"))
	router.POST("/api/users/:id/follow", proxyTo("user"))
	router.DELETE("/api/users/:id/follow", proxyTo("user"))
	router.GET("/api/users/:id/followers", proxyTo("user"))
	router.GET("/api/users/:id/following", proxyTo("user"))

	// 文章相关
	router.POST("/api/articles", proxyTo("article"))
	router.PUT("/api/articles/:id", proxyTo("article"))
	router.DELETE("/api/articles/:id", proxyTo("article"))

	// 互动相关
	router.POST("/api/articles/:id/comments", proxyTo("interaction"))
	router.GET("/api/articles/:id/comments", proxyTo("interaction"))
	router.POST("/api/comments/:id/like", proxyTo("interaction"))
}

// proxyTo 返回一个反向代理处理器
func proxyTo(serviceName string) gin.HandlerFunc {
	targetURL := services[serviceName]

	return func(c *gin.Context) {
		//将字符串形式的服务地址（如"http://localhost:8001"）解析为 url.URL 结构体
		target, _ := url.Parse(targetURL)

		// 创建反向代理
		proxy := httputil.NewSingleHostReverseProxy(target)

		// 修改请求，转发认证信息
		// 在每次请求转发前被调用，用来修改即将发往后端的*http.Request。
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)

			// 转发用户信息 Header
			if userID := c.GetHeader("X-User-ID"); userID != "" {
				req.Header.Set("X-User-ID", userID)
			}
			if username := c.GetHeader("X-Username"); username != "" {
				req.Header.Set("X-Username", username)
			}
		}

		// 代理请求
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
