package main

import (
	"log"

	"blog-community/search-service/handler"
	"blog-community/search-service/repository"
	"blog-community/search-service/service"
	"blog-community/shared/events"
	"blog-community/shared/search"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 连接 Elasticsearch
	es := search.NewElasticsearch()

	// 2. 确保索引存在
	repo := repository.NewSearchRepository(es)
	if err := repo.EnsureIndex(); err != nil {
		log.Printf("警告：确保索引失败: %v", err)
	}

	// 3. 连接 RabbitMQ
	rmq := events.NewRabbitMQ()
	defer rmq.Close()

	// 4. 初始化各层
	consumer := events.NewConsumer(rmq)
	svc := service.NewSearchService(repo, consumer)
	h := handler.NewSearchHandler(svc)

	// 5. 启动事件监听（同步 ES）
	go svc.StartListening()

	// 6. 设置路由
	router := gin.Default()
	router.GET("/api/search", h.Search)

	// 7. 启动服务
	log.Println("搜索服务启动在 :8005")
	router.Run(":8005")
}
