package main

import (
	api_middleware "blog-community/api-gateway/middleware"
	"blog-community/shared/database"
	share_middleware "blog-community/shared/middleware"
	"blog-community/shared/models"
	"blog-community/user-service/handler"
	"blog-community/user-service/repository"
	"blog-community/user-service/service"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.NewMySQL(database.LoadConfigFromEnv())
	db.AutoMigrate(&models.User{}, &models.Follow{})
	repo := repository.NewUserRepository(db)
	service := service.NewUserService(repo, []byte(api_middleware.JWTSecret))
	handler := handler.NewUserHandler(service)

	//启动路由
	router := gin.Default()
	router.Use(share_middleware.Logger())
	router.Use(share_middleware.CORS())

	router.POST("/api/users/register", handler.Register)
	router.POST("/api/users/login", handler.Login)
	router.GET("/api/users/:id", handler.GetProfile)
	router.PUT("/api/users/:id", handler.UpdateProfile)

	router.POST("/api/users/:id/follow", handler.Follow)
	router.DELETE("/api/users/:id/follow", handler.UnFollow)
	router.GET("/api/users/:id/followers", handler.GetFollowers)
	router.GET("/api/users/:id/followings", handler.GetFollowings)
	log.Println("用户服务启动在 :8001")
	router.Run(":8001")
}
