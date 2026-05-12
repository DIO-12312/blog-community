package main

import (
	"blog-community/api-gateway/routes"

	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 创建 Gin 引擎
	router := gin.Default()

	// 注册所有路由
	routes.SetupRoutes(router)

	// 启动服务器（网关默认监听 8000）
	log.Println("API Gateway listening on :8000")
	router.Run(":8000")
}
