// publisher.go
package events

import (
	"context"
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	rmq *RabbitMQ
}

func NewPublisher(rmq *RabbitMQ) *Publisher {
	// 声明交换机
	rmq.DeclareExchange(ExchangeName, "topic")
	return &Publisher{rmq: rmq}
}

// Publish 发布事件
func (p *Publisher) Publish(eventType string, data map[string]interface{}) error {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return p.rmq.channel.PublishWithContext(ctx,
		ExchangeName, // exchange
		eventType,    // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // 消息持久化
			Body:         body,
		},
	)
}
