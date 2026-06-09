package main

import (
	"log"

	"blog-community/interaction-service/handler"
	"blog-community/interaction-service/repository"
	"blog-community/interaction-service/service"
	"blog-community/shared/cache"
	"blog-community/shared/database"
	"blog-community/shared/events"
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.NewMySQL(database.LoadConfigFromEnv())
	db.AutoMigrate(&models.Comment{}, &models.Like{}, &models.Collection{})

	// RabbitMQ 事件发布
	rmq := events.NewRabbitMQ()
	defer rmq.Close()
	publisher := events.NewPublisher(rmq)

	// 评论
	commentRepo := repository.NewCommentRepository(db)
	commentSvc := service.NewCommentService(commentRepo, db, publisher)
	commentH := handler.NewCommentHandler(commentSvc)

	// Redis 缓存
	redisClient, err := cache.NewRedisClient(
		database.GetEnv("REDIS_ADDR", "127.0.0.1:6379"),
		database.GetEnv("REDIS_PASSWORD", ""),
	)
	if err != nil {
		log.Printf("Redis 连接失败（服务将降级为纯 DB 查询）: %v", err)
	} else {
		log.Println("Redis 连接成功")
	}

	// 点赞
	likeRepo := repository.NewLikeRepository(db)
	likeSvc := service.NewLikeService(likeRepo, redisClient, db, publisher)
	likeH := handler.NewLikeHandler(likeSvc)

	// 收藏
	collectionRepo := repository.NewCollectionRepository(db)
	collectionSvc := service.NewCollectionService(collectionRepo)
	collectionH := handler.NewCollectionHandler(collectionSvc)

	router := gin.Default()

	// 评论路由
	router.POST("/api/articles/:id/comments", commentH.Create)
	router.DELETE("/api/comments/:id", commentH.Delete)
	router.GET("/api/articles/:id/comments", commentH.GetByArticle)

	// 管理员评论路由
	router.GET("/api/admin/comments", commentH.ListAll)
	router.DELETE("/api/admin/comments/:id", commentH.AdminDelete)

	// 点赞路由
	router.POST("/api/likes", likeH.Like)
	router.DELETE("/api/likes", likeH.Unlike)
	router.GET("/api/likes/status", likeH.GetStatus)

	// 收藏路由
	router.POST("/api/collections", collectionH.Collect)
	router.DELETE("/api/collections/:article_id", collectionH.Uncollect)
	router.GET("/api/collections/status", collectionH.GetStatus)
	router.GET("/api/collections", collectionH.GetMyCollections)

	log.Println("互动服务启动在 :8003")
	router.Run(":8003")
}
