package routes

import (
	"net/url"

	"blog-community/api-gateway/middleware"
	shared_middleware "blog-community/shared/middleware"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
)

// ServiceRegistry 服务注册表
var services = map[string]string{
	"user":         "http://user-service:8001",
	"article":      "http://content-service:8002",
	"interaction":  "http://interaction-service:8003",
	"notification": "http://notification-service:8004",
	"search":       "http://search-service:8005",
	"audit":        "http://audit-service:8006",
}

// proxyCache 预创建每个服务的反向代理（单例，复用 HTTP 连接池）
var proxyCache = initProxyCache()

func initProxyCache() map[string]*httputil.ReverseProxy {
	cache := make(map[string]*httputil.ReverseProxy, len(services))
	for name, urlStr := range services {
		target, _ := url.Parse(urlStr)
		cache[name] = httputil.NewSingleHostReverseProxy(target)
	}
	return cache
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

	// 管理员路由（需要管理员权限）
	admin := router.Group("/api/admin")
	admin.Use(middleware.AdminMiddleware())
	setupAdminRoutes(admin)
}

// setupPublicRoutes 设置公开路由
func setupPublicRoutes(router *gin.Engine) {
	// 用户注册、登录
	router.POST("/api/users/register", proxyTo("user"))
	router.POST("/api/users/login", proxyTo("user"))

	// 搜索（可匿名查看）
	router.GET("/api/search", middleware.OptionalAuthMiddleware(), proxyTo("search"))

	// 用户信息查询（无需认证）
	router.GET("/api/users", proxyTo("user"))

	// 文章列表（可匿名查看）
	router.GET("/api/articles", middleware.OptionalAuthMiddleware(), proxyTo("article"))
	router.GET("/api/articles/:id", middleware.OptionalAuthMiddleware(), proxyTo("article"))
	router.GET("/api/articles/category/:category", middleware.OptionalAuthMiddleware(), proxyTo("article"))
}

// setupPrivateRoutes 设置需要认证的路由
func setupPrivateRoutes(router *gin.RouterGroup) {
	// 用户相关
	router.PUT("/api/users/:id", proxyTo("user"))
	router.POST("/api/users/:id/follow", proxyTo("user"))
	router.DELETE("/api/users/:id/follow", proxyTo("user"))
	router.GET("/api/users/:id/followers", proxyTo("user"))
	router.GET("/api/users/:id/followings", proxyTo("user"))

	// 文章相关
	router.POST("/api/articles", proxyTo("article"))
	router.PUT("/api/articles/:id", proxyTo("article"))
	router.DELETE("/api/articles/:id", proxyTo("article"))
	router.POST("/api/articles/:id/publish", proxyTo("article"))

	// 评论相关
	router.POST("/api/articles/:id/comments", proxyTo("interaction"))
	router.GET("/api/articles/:id/comments", proxyTo("interaction"))
	router.DELETE("/api/comments/:id", proxyTo("interaction"))

	// 点赞相关
	router.POST("/api/likes", proxyTo("interaction"))
	router.DELETE("/api/likes", proxyTo("interaction"))
	router.GET("/api/likes/status", proxyTo("interaction"))

	// 收藏相关
	router.POST("/api/collections", proxyTo("interaction"))
	router.DELETE("/api/collections/:article_id", proxyTo("interaction"))
	router.GET("/api/collections/status", proxyTo("interaction"))
	router.GET("/api/collections", proxyTo("interaction"))

	// 通知
	router.GET("/api/notifications", proxyTo("notification"))
	router.GET("/api/notifications/unread-count", proxyTo("notification"))
	router.PUT("/api/notifications/:id/read", proxyTo("notification"))
	router.PUT("/api/notifications/read-all", proxyTo("notification"))

	// 审计日志
	router.GET("/api/audit-logs", proxyTo("audit"))
}

func setupAdminRoutes(router *gin.RouterGroup) {
	// 用户管理
	router.GET("/users", proxyTo("user"))
	router.PUT("/users/:id/ban", proxyTo("user"))
	router.PUT("/users/:id/unban", proxyTo("user"))
}

// proxyTo 返回一个反向代理处理器（复用全局单例 proxy，共享 HTTP 连接池）
func proxyTo(serviceName string) gin.HandlerFunc {
	targetURL := services[serviceName]
	target, _ := url.Parse(targetURL)
	proxy := proxyCache[serviceName]

	return func(c *gin.Context) {
		// 修改当前请求的目标地址（c.Request 是每次请求独立的）
		c.Request.URL.Scheme = target.Scheme
		c.Request.URL.Host = target.Host
		c.Request.Host = target.Host

		// 转发用户认证信息到下游
		if userID, exists := c.Get("userID"); exists {
			c.Request.Header.Set("X-User-ID", userID.(string))
		}
		if username, exists := c.Get("username"); exists {
			c.Request.Header.Set("X-Username", username.(string))
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
