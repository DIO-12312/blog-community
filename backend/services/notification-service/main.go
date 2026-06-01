package main

import (
	"blog-community/shared/events"
	"log"
)

func main() {
	rmq := events.NewRabbitMQ()
	defer rmq.Close()

	consumer := events.NewConsumer(rmq)

	// 订阅文章发布事件
	consumer.Subscribe("notification_queue", "article.published", func(event events.Event) error {
		userID := event.Data["user_id"].(string)
		title := event.Data["title"].(string)

		// 获取用户的粉丝列表
		// 给每个粉丝发送通知："你关注的 xxx 发表了新文章《title》"
		log.Printf("发送通知: 用户 %s 发表了文章 %s", userID, title)
		return nil
	})

	// 订阅关注事件
	consumer.Subscribe("notification_queue", "user.followed", func(event events.Event) error {
		followerID := event.Data["follower_id"].(string)
		followingID := event.Data["following_id"].(string)
		log.Printf("发送通知: %s 关注了 %s", followerID, followingID)
		return nil
	})

	// 阻塞主 goroutine
	select {}
}
