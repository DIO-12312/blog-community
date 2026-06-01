package events

import (
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ 连接封装

/*
Connection:TCP连接，应该复用
Channel:虚拟连接，建立于TCP之上，类似于协程和线程的关系
*/
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQ 创建 RabbitMQ 连接
func NewRabbitMQ() *RabbitMQ {
	url := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	//接收MQURL格式的字符串，生成TCP连接
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("RabbitMQ 连接失败: %v", err)
	}

	//并发的处理连接，应该每个协程使用一个
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("创建 Channel 失败: %v", err)
	}

	log.Println("RabbitMQ 连接成功")
	return &RabbitMQ{conn: conn, channel: ch}
}

// Close 关闭连接
func (r *RabbitMQ) Close() {
	r.channel.Close()
	r.conn.Close()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// DeclareExchange 声明交换机
func (r *RabbitMQ) DeclareExchange(name, kind string) error {
	return r.channel.ExchangeDeclare(
		name,  // 交换机名称
		kind,  // 类型：direct, fanout, topic, headers
		true,  // durable: 持久化，RabbitMQ 重启后不丢失
		false, // auto-delete: 没有绑定时不自动删除
		false, // internal: 不是内部交换机
		false, // no-wait: 等待服务器确认
		nil,   // arguments: 额外参数
	)
}

func (r *RabbitMQ) DeclareQueue(name, kind string) (amqp.Queue, error) {
	return r.channel.QueueDeclare(
		name,  // 队列名称
		true,  // durable: 持久化，RabbitMQ 重启后不丢失
		false, // auto-delete: 没有消费者时不自动删除
		false, // internal: 不是内部队列
		false, // no-wait: 等待服务器确认
		nil,   // arguments: 额外参数

	)
}

// BindQueue 绑定队列到交换机
func (r *RabbitMQ) BindQueue(queueName, routingKey, exchangeName string) error {
	return r.channel.QueueBind(
		queueName,    // 队列名
		routingKey,   // 路由键（支持通配符）
		exchangeName, // 交换机名
		false,        // no-wait
		nil,          // arguments
	)
}
