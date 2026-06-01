// consumer.go
package events

import (
	"encoding/json"
	"log"
)

type MessageHandler func(event Event) error

type Consumer struct {
	rmq *RabbitMQ
}

func NewConsumer(rmq *RabbitMQ) *Consumer {
	rmq.DeclareExchange(ExchangeName, "topic")
	return &Consumer{rmq: rmq}
}

// Subscribe 订阅事件
// queueName: 队列名（每个服务用不同的队列名）
// routingKey: 路由键（支持通配符 * 和 #）
// handler: 消息处理函数
func (c *Consumer) Subscribe(queueName, routingKey string, handler MessageHandler) error {
	// 声明队列
	queue, err := c.rmq.DeclareQueue(queueName, "topic")
	if err != nil {
		return err
	}

	// 绑定到交换机
	if err := c.rmq.BindQueue(queue.Name, routingKey, ExchangeName); err != nil {
		return err
	}

	// 开始消费
	messages, err := c.rmq.channel.Consume(
		queue.Name, // 队列名
		"",         // consumer tag（空 = 自动生成）
		false,      // auto-ack: false 表示手动确认
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return err
	}

	// 在新的 goroutine 中处理消息
	go func() {
		for msg := range messages {
			var event Event
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("消息解析失败: %v", err)
				msg.Nack(false, false) // 解析失败，丢弃消息
				continue
			}

			if err := handler(event); err != nil {
				log.Printf("消息处理失败: %v, 事件: %s", err, event.Type)
				msg.Nack(false, true) // 处理失败，重新入队
				continue
			}

			msg.Ack(false) // 处理成功，确认消息
		}
	}()

	log.Printf("开始消费队列: %s, 路由: %s", queueName, routingKey)
	return nil
}
