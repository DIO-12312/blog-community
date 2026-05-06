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
		fmt.Printf("[%s] %s %s %d (%dms)\n",
			startTime.Format("2006-01-02 15:04:05"),
			c.Request.Method,
			c.Request.RequestURI,
			statusCode,
			duration.Milliseconds(),
		)

	}
}
