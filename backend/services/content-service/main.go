package main

import (
	"blog-community/content-service/handler"
	"blog-community/content-service/repository"
	"blog-community/content-service/service"
	"blog-community/shared/cache"
	"blog-community/shared/events"
	"blog-community/shared/models"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func main() {
	// 1. 连接数据库
	mysqlHost := getEnv("MYSQL_HOST", "mysql")
	mysqlPort := getEnv("MYSQL_PORT", "3306")
	mysqlUser := getEnv("MYSQL_USER", "root")
	mysqlPass := getEnv("MYSQL_PASSWORD", "123456")
	mysqlDB := getEnv("MYSQL_DATABASE", "blog")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlUser, mysqlPass, mysqlHost, mysqlPort, mysqlDB)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 2. 执行数据库迁移
	db.AutoMigrate(&models.Article{}, &models.Category{}, &models.ReviewRecord{})

	// 3. 连接 Redis
	redisAddr := getEnv("REDIS_ADDR", "redis:6379")
	redisClient, err := cache.NewRedisClient(redisAddr, "")
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer redisClient.Close()

	// 4. 初始化各层
	g := &cache.Group{
		GroupMap: make(map[string]*cache.Call),
	}
	repo := repository.NewArticleRepository(db, redisClient, g)

	rmq := events.NewRabbitMQ()
	defer rmq.Close()

	publisher := events.NewPublisher(rmq)
	svc := service.NewArticleService(repo, publisher)
	h := handler.NewArticleHandler(svc)

	// 审稿模块
	reviewRepo := repository.NewReviewRepository(db)
	reviewSvc := service.NewReviewService(repo, reviewRepo, publisher)
	reviewH := handler.NewReviewHandler(reviewSvc)

	// 5. 设置路由
	router := gin.Default()

	// 公开路由
	router.GET("/api/articles", h.ListArticles)
	router.GET("/api/articles/:id", h.GetArticle)
	router.GET("/api/articles/category/:category", h.ListByCategory)

	// 需要认证的路由
	router.POST("/api/articles", h.CreateArticle)
	router.PUT("/api/articles/:id", h.EditArticle)
	router.POST("/api/articles/:id/publish", h.PublishArticle)
	router.DELETE("/api/articles/:id", h.DeleteArticle)
	router.GET("/api/articles/my", h.ListMyArticles)

	// 管理员路由
	router.GET("/api/admin/articles", h.ListAllArticles)
	router.DELETE("/api/admin/articles/:id", h.AdminDeleteArticle)

	// 审稿路由（需认证）
	router.POST("/api/articles/:id/submit-review", reviewH.SubmitForReview)
	router.GET("/api/articles/:id/review-history", reviewH.GetReviewHistory)

	// 审稿管理员路由
	router.GET("/api/admin/reviews/pending", reviewH.ListPendingReviews)
	router.POST("/api/admin/articles/:id/review", reviewH.ReviewArticle)

	// 6. 启动浏览计数定期同步任务（每 5 分钟将 Redis 计数写入 MySQL）
	svc.StartViewCountSyncWorker(context.Background(), 5*time.Minute)

	// 7. 启动服务
	log.Println("Article service listening on :8002")
	router.Run(":8002")
}
