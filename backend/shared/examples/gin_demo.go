package main

import (
	"blog-community/shared/middleware"

	"github.com/gin-gonic/gin"
)

// 模拟数据
var articles = []map[string]string{
	{"id": "1", "title": "Go 并发编程", "author": "Tom"},
	{"id": "2", "title": "微服务设计模式", "author": "Jack"},
	{"id": "3", "title": "Redis 实战", "author": "Sam"},
}

func main() {
	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

}
