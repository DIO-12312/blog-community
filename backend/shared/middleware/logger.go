package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		duration := time.Since(startTime)
		statusCode := c.Writer.Status()
		fmt.Println("[%s] %s %s %d (%dms)\n",
			startTime.Format("2006-01-02 15:04:05"),
			c.Request.Method,
			c.Request.RequestURI,
			statusCode,
			duration.Milliseconds(),
		)

	}
}

// 这一步封装有利于中间件格式的统一
func LoggerMiddleware() gin.HandlerFunc {
	return Logger()
}
