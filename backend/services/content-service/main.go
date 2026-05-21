package main

import (
	"blog-community/content-service/handler"
	"blog-community/content-service/migration"
	"blog-community/content-service/repository"
	"blog-community/content-service/service"
	"blog-community/shared/cache"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. 连接数据库
	dsn := "user:password@tcp(localhost:3306)/blog_community?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 2. 执行数据库迁移
	if err := migration.RunMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// 3. 连接 Redis
	redisClient, err := cache.NewRedisClient("localhost:6379", "")
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer redisClient.Close()

	// 4. 初始化各层
	repo := repository.NewArticleRepository(db, redisClient)
	svc := service.NewArticleService(repo)
	h := handler.NewArticleHandler(svc)

	// 5. 设置路由
	router := gin.Default()

	// 公开路由
	router.GET("/api/articles", h.ListArticles)
	router.GET("/api/articles/:id", h.GetArticle)
	router.GET("/api/articles", h.ListByCategory)

	// 需要认证的路由
	router.POST("/api/articles", h.CreateArticle)
	router.PUT("/api/articles/:id", h.EditArticle)
	router.POST("/api/articles/:id/publish", h.PublishArticle)
	router.DELETE("/api/articles/:id", h.DeleteArticle)

	// 6. 启动服务
	log.Println("Article service listening on :8002")
	router.Run(":8002")
}
