package main

import (
	"blog-community/shared/events"
	"log"
)

func main() {
	rmq := events.NewRabbitMQ()
	defer rmq.Close()

	publisher := events.NewPublisher(rmq)

	// 测试1: 发布文章事件
	err := publisher.Publish(events.EventArticlePublished, map[string]interface{}{
		"article_id": "1",
		"user_id":    "zhangsan",
		"title":      "测试文章标题",
	})
	if err != nil {
		log.Fatalf("发布文章事件失败: %v", err)
	}
	log.Println("article.published 事件发布成功")

	// 测试2: 发布关注事件
	err = publisher.Publish(events.EventUserFollowed, map[string]interface{}{
		"follower_id":  "lisi",
		"following_id": "zhangsan",
	})
	if err != nil {
		log.Fatalf("发布关注事件失败: %v", err)
	}
	log.Println("user.followed 事件发布成功")

	log.Println("全部测试事件发布完成")
}
