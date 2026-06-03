package search

import (
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
)

func NewElasticsearch() *elasticsearch.Client {
	cfg := elasticsearch.Config{
		Addresses: []string{
			getEnv("ES_URL", "http://localhost:9200"),
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Elasticsearch 连接失败: %v", err)
	}

	// 测试连接
	res, err := client.Info()
	if err != nil {
		log.Fatalf("Elasticsearch 不可用: %v", err)
	}
	defer res.Body.Close()

	log.Println("Elasticsearch 连接成功")
	return client
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
