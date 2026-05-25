package main

import (
	"blog-community/interaction-service/handler"
	"blog-community/interaction-service/repository"
	"blog-community/interaction-service/service"
	"blog-community/shared/database"
	"blog-community/shared/models"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.NewMySQL(database.LoadConfigFromEnv())
	db.AutoMigrate(&models.Comment{})

	repo := repository.NewCommentRepository(db)
	svc := service.NewCommentService(repo)
	h := handler.NewCommentHandler(svc)

	router := gin.Default()

	router.POST("/api/articles/:id/comments", h.Create)
	router.DELETE("/api/comments/:id", h.Delete)
	router.GET("/api/articles/:id/comments", h.GetByArticle)

	log.Println("评论服务启动在 :8003")
	router.Run(":8003")
}
