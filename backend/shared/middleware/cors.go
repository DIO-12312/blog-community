package middleware

import "github.com/gin-gonic/gin"

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		//"*"代表接受任意的前端页面访问
		//可以换成该项目的前端源"http://localhost:5173"
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		/*
			┌───────────────┬───────────────────────────────────────────────┐
			│     字段      │                     用途                      │
			├───────────────┼───────────────────────────────────────────────┤
			│ Content-Type  │ 告诉服务器请求体的格式（如 application/json）   │
			├───────────────┼───────────────────────────────────────────────┤
			│ Authorization │ 用来发送身份验证信息（如 JWT Token）            │
			└───────────────┴───────────────────────────────────────────────┘
		*/
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		//缓存预检请求
		//浏览器会先发送OPTIONS作为预检请求
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
