package main

import (
	"log"

	api_middleware "blog-community/api-gateway/middleware"
	"blog-community/shared/database"
	"blog-community/shared/events"
	share_middleware "blog-community/shared/middleware"
	"blog-community/shared/models"
	"blog-community/user-service/handler"
	"blog-community/user-service/repository"
	"blog-community/user-service/service"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.NewMySQL(database.LoadConfigFromEnv())
	db.AutoMigrate(&models.User{}, &models.Follow{})

	// RabbitMQ 事件发布
	rmq := events.NewRabbitMQ()
	defer rmq.Close()
	publisher := events.NewPublisher(rmq)

	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo, []byte(api_middleware.JWTSecret), publisher)
	h := handler.NewUserHandler(svc)

	//启动路由
	router := gin.Default()
	router.Use(share_middleware.Logger())
	router.Use(share_middleware.CORS())

	router.POST("/api/users/register", h.Register)
	router.POST("/api/users/login", h.Login)
	router.GET("/api/users", h.GetProfile)
	router.PUT("/api/users/:id", h.UpdateProfile)

	router.POST("/api/users/:id/follow", h.Follow)
	router.DELETE("/api/users/:id/follow", h.UnFollow)
	router.GET("/api/users/:id/followers", h.GetFollowers)
	router.GET("/api/users/:id/followings", h.GetFollowings)
	log.Println("用户服务启动在 :8001")
	router.Run(":8001")
}
