package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
)

type SearchRepository struct {
	es *elasticsearch.Client
}

func NewSearchRepository(es *elasticsearch.Client) *SearchRepository {
	return &SearchRepository{es: es}
}

// EnsureIndex 确保 articles 索引存在，不存在则创建
func (r *SearchRepository) EnsureIndex() error {
	res, err := r.es.Indices.Exists([]string{"articles"})
	if err != nil {
		return fmt.Errorf("检查索引失败: %w", err)
	}
	res.Body.Close()

	if res.StatusCode == 200 {
		return nil // 索引已存在
	}

	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":           "text",
					"analyzer":       "ik_max_word",
					"search_analyzer": "ik_smart",
				},
				"content": map[string]interface{}{
					"type":            "text",
					"analyzer":        "ik_max_word",
					"search_analyzer": "ik_smart",
				},
				"user_id": map[string]interface{}{
					"type": "keyword",
				},
				"category_id": map[string]interface{}{
					"type": "keyword",
				},
				"status": map[string]interface{}{
					"type": "keyword",
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}

	body, _ := json.Marshal(mapping)
	res, err = r.es.Indices.Create("articles", r.es.Indices.Create.WithBody(bytes.NewReader(body)))
	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		errBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("创建索引返回错误: %s", string(errBody))
	}

	return nil
}

// IndexArticle 索引文章到 ES
func (r *SearchRepository) IndexArticle(articleID string, data map[string]interface{}) error {
	body, _ := json.Marshal(data)
	res, err := r.es.Index(
		"articles",
		bytes.NewReader(body),
		r.es.Index.WithDocumentID(articleID),
		r.es.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("索引文章失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		errBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES 返回错误: %s: %s", res.Status(), string(errBody))
	}

	return nil
}

// DeleteArticle 从 ES 删除文章
func (r *SearchRepository) DeleteArticle(articleID string) error {
	res, err := r.es.Delete("articles", articleID)
	if err != nil {
		return fmt.Errorf("删除文章索引失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		errBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES 返回错误: %s: %s", res.Status(), string(errBody))
	}

	return nil
}

// SearchResult 搜索结果
type SearchResult struct {
	Articles []map[string]interface{}
	Total    int64
}

// Search 搜索文章，返回高亮结果
func (r *SearchRepository) Search(keyword string, page, size int) (*SearchResult, error) {
	from := (page - 1) * size

	query := map[string]interface{}{
		"from": from,
		"size": size,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  keyword,
				"fields": []string{"title^3", "content"},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"title":   map[string]interface{}{},
				"content": map[string]interface{}{"fragment_size": 150},
			},
			"pre_tags":  []string{"<em>"},
			"post_tags": []string{"</em>"},
		},
		"sort": []interface{}{
			"_score",
			map[string]interface{}{"created_at": "desc"},
		},
	}

	body, _ := json.Marshal(query)
	res, err := r.es.Search(
		r.es.Search.WithIndex("articles"),
		r.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		errBody, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("ES 返回错误: %s: %s", res.Status(), string(errBody))
	}

	// 解析响应
	var result map[string]interface{}
	responseBody, _ := io.ReadAll(res.Body)
	json.Unmarshal(responseBody, &result)

	hits, ok := result["hits"].(map[string]interface{})
	if !ok {
		return &SearchResult{Articles: []map[string]interface{}{}, Total: 0}, nil
	}

	total := int64(hits["total"].(map[string]interface{})["value"].(float64))

	var articles []map[string]interface{}
	hitsList, ok := hits["hits"].([]interface{})
	if !ok {
		return &SearchResult{Articles: []map[string]interface{}{}, Total: total}, nil
	}

	for _, hit := range hitsList {
		h := hit.(map[string]interface{})
		article := h["_source"].(map[string]interface{})
		article["_id"] = h["_id"]

		if highlight, ok := h["highlight"]; ok {
			article["highlight"] = highlight
		}

		articles = append(articles, article)
	}

	return &SearchResult{Articles: articles, Total: total}, nil
}
