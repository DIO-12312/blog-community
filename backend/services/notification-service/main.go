package main

import (
	"log"

	"blog-community/notification-service/handler"
	"blog-community/notification-service/repository"
	"blog-community/notification-service/service"
	"blog-community/shared/database"
	"blog-community/shared/events"
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 连接数据库
	db := database.NewMySQL(database.LoadConfigFromEnv())
	db.AutoMigrate(&models.Notification{})

	// 2. 连接 RabbitMQ
	rmq := events.NewRabbitMQ()
	defer rmq.Close()

	// 3. 初始化各层
	consumer := events.NewConsumer(rmq)
	repo := repository.NewNotificationRepository(db)
	svc := service.NewNotificationService(repo, consumer)
	h := handler.NewNotificationHandler(svc)

	// 4. 启动事件监听
	go svc.StartListening()

	// 5. 设置路由
	router := gin.Default()
	router.GET("/api/notifications", h.GetNotifications)
	router.PUT("/api/notifications/read-all", h.MarkAllAsRead)
	router.PUT("/api/notifications/:id/read", h.MarkAsRead)
	router.GET("/api/notifications/unread-count", h.GetUnreadCount)

	// 6. 启动服务
	log.Println("通知服务启动在 :8006")
	router.Run(":8006")
}
