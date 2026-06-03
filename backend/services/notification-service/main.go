package main

import (
	"log"

	"blog-community/notification-service/handler"
	"blog-community/notification-service/repository"
	"blog-community/notification-service/service"
	"blog-community/shared/database"
	"blog-community/shared/events"
	share_middleware "blog-community/shared/middleware"
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

	// 4. 启动事件监听（MQ 消费）
	svc.StartListening()

	// 5. 启动 HTTP 服务（供前端查询通知）
	router := gin.Default()
	router.Use(share_middleware.CORS())
	router.GET("/api/notifications", h.GetNotifications)
	router.GET("/api/notifications/unread-count", h.GetUnreadCount)
	router.PUT("/api/notifications/:id/read", h.MarkAsRead)
	router.PUT("/api/notifications/read-all", h.MarkAllAsRead)

	go func() {
		log.Println("通知 HTTP 服务启动在 :8004")
		router.Run(":8004")
	}()

	log.Println("通知服务启动（MQ 消费 + HTTP API）")

	// 6. 阻塞主 goroutine
	select {}
}
