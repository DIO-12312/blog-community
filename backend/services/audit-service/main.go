package main

import (
	"log"

	"blog-community/audit-service/handler"
	"blog-community/audit-service/repository"
	"blog-community/audit-service/service"
	"blog-community/shared/database"
	"blog-community/shared/events"
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 连接数据库
	db := database.NewMySQL(database.LoadConfigFromEnv())
	db.AutoMigrate(&models.AuditLog{})

	// 2. 连接 RabbitMQ
	rmq := events.NewRabbitMQ()
	defer rmq.Close()

	// 3. 初始化各层
	consumer := events.NewConsumer(rmq)
	repo := repository.NewAuditRepository(db)
	svc := service.NewAuditService(repo, consumer)
	h := handler.NewAuditHandler(svc)

	// 4. 启动事件监听（消费所有事件）
	go svc.StartListening()

	// 5. 设置 HTTP 路由
	router := gin.Default()
	router.GET("/api/audit-logs", h.Query)

	// 6. 启动 HTTP 服务
	log.Println("审计服务启动在 :8006")
	router.Run(":8006")
}
