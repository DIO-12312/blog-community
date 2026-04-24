package utils

import (
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
)

// Success返回响应
func Success(c *gin.Context, code int, message string, data interface{}) {
	//为什么要用两个code?
	//前者作用于HTTP协议层，浏览器、网络工具、中间件根据这个状态码做出反应，决定是否缓存、重试、路由
	//后者作用于前端应用层，决定是否显示页面
	c.JSON(code, models.ApiResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Error返回错误响应
func Error(c *gin.Context, code int, message string) {
	//为什么要用两个code?
	//前者作用于HTTP协议层，浏览器、网络工具、中间件根据这个状态码做出反应，决定是否缓存、重试、路由
	//后者作用于前端应用层，决定是否显示页面
	c.JSON(code, models.ApiResponse{
		Code:    code,
		Message: message,
	})
}

func Paginated(c *gin.Context, data interface{}, message string, total int64, page int, pageSize int) {
	c.JSON(200, models.PaginatedResponse{
		Code:    200,
		Message: message,
		Pagination: models.PaginationMeta{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}
