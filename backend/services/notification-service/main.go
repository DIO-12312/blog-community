package main

import (
	"log"

	"blog-community/notification-service/repository"
	"blog-community/notification-service/service"
	"blog-community/shared/database"
	"blog-community/shared/events"
	"blog-community/shared/models"
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

	// 4. 启动事件监听
	svc.StartListening()

	log.Println("通知服务启动（纯消息队列消费模式）")

	// 5. 阻塞主 goroutine
	select {}
}
