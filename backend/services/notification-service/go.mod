module blog-community/notification-service

require blog-community/shared v0.0.0

require github.com/rabbitmq/amqp091-go v1.11.0 // indirect

go 1.26.2

replace blog-community/shared => ../../shared
