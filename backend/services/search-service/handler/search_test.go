package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-community/search-service/repository"
	"blog-community/search-service/service"
	"blog-community/shared/models"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
)

// mockESServer 模拟 Elasticsearch HTTP 服务（与 repository 测试中的 mock 相同）
type mockESServer struct {
	indices    map[string]bool
	docs       map[string]map[string]interface{}
	failPrefix string
	nextCode   int
}

func newMockES() (*elasticsearch.Client, *mockESServer, func()) {
	m := &mockESServer{
		indices: make(map[string]bool),
		docs:    make(map[string]map[string]interface{}),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.Header().Set("Content-Type", "application/json")

		if m.nextCode != 0 {
			code := m.nextCode
			m.nextCode = 0
			w.WriteHeader(code)
			if code >= 400 {
				json.NewEncoder(w).Encode(map[string]interface{}{"error": "mock error"})
			}
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")

		if path == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version": map[string]interface{}{"number": "8.0.0-mock"},
			})
			return
		}

		if m.failPrefix != "" && strings.HasPrefix(path, m.failPrefix) {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "mock error"})
			return
		}

		switch {
		case r.Method == "HEAD" && path == "articles":
			if m.indices["articles"] {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(404)
			}

		case r.Method == "PUT" && path == "articles":
			m.indices["articles"] = true
			json.NewEncoder(w).Encode(map[string]interface{}{"acknowledged": true})

		case r.Method == "PUT" && strings.HasPrefix(path, "articles/_doc/"):
			docID := strings.TrimPrefix(path, "articles/_doc/")
			docID = strings.TrimSuffix(docID, "?refresh=true")
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			m.docs[docID] = body
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": "created", "_id": docID,
			})

		case r.Method == "DELETE" && strings.HasPrefix(path, "articles/_doc/"):
			docID := strings.TrimPrefix(path, "articles/_doc/")
			result := "not_found"
			if _, ok := m.docs[docID]; ok {
				delete(m.docs, docID)
				result = "deleted"
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"result": result})

		case r.Method == "POST" && strings.HasSuffix(path, "/_search"):
			var query map[string]interface{}
			json.NewDecoder(r.Body).Decode(&query)

			keyword := ""
			if q, ok := query["query"].(map[string]interface{}); ok {
				if mm, ok := q["multi_match"].(map[string]interface{}); ok {
					if k, ok := mm["query"].(string); ok {
						keyword = k
					}
				}
			}

			var hits []map[string]interface{}
			for id, doc := range m.docs {
				title, _ := doc["title"].(string)
				content, _ := doc["content"].(string)
				if strings.Contains(title, keyword) || strings.Contains(content, keyword) {
					hit := map[string]interface{}{
						"_id": id, "_source": doc,
					}
					highlight := map[string]interface{}{}
					if strings.Contains(title, keyword) {
						highlight["title"] = []string{
							strings.ReplaceAll(title, keyword, "<em>"+keyword+"</em>"),
						}
					}
					if strings.Contains(content, keyword) {
						highlight["content"] = []string{
							strings.ReplaceAll(content, keyword, "<em>"+keyword+"</em>"),
						}
					}
					if len(highlight) > 0 {
						hit["highlight"] = highlight
					}
					hits = append(hits, hit)
				}
			}
			if hits == nil {
				hits = []map[string]interface{}{}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": len(hits)},
					"hits":  hits,
				},
			})

		default:
			w.WriteHeader(404)
		}
	})

	ts := httptest.NewServer(handler)

	cfg := elasticsearch.Config{Addresses: []string{ts.URL}}
	client, _ := elasticsearch.NewClient(cfg)

	return client, m, ts.Close
}

// setupSearchRouter 创建测试路由，使用 mock ES
func setupSearchRouter(t *testing.T) (*gin.Engine, *mockSearchEnv) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client, m, close := newMockES()
	t.Cleanup(close)

	repo := repository.NewSearchRepository(client)
	svc := service.NewSearchService(repo, nil)
	h := NewSearchHandler(svc)

	r := gin.New()
	r.GET("/api/search", h.Search)

	return r, &mockSearchEnv{mock: m, svc: svc}
}

type mockSearchEnv struct {
	mock *mockESServer
	svc  *service.SearchService
}

func TestSearch_MissingKeyword(t *testing.T) {
	r, _ := setupSearchRouter(t)

	req, _ := http.NewRequest("GET", "/api/search", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HTTP状态码: 期望=%d, 实际=%d", http.StatusBadRequest, w.Code)
	}

	var resp models.ApiResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Message == "" {
		t.Error("期望返回错误消息")
	}
}

func TestSearch_EmptyKeyword(t *testing.T) {
	r, _ := setupSearchRouter(t)

	req, _ := http.NewRequest("GET", "/api/search?q=", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HTTP状态码: 期望=%d, 实际=%d", http.StatusBadRequest, w.Code)
	}
}

func TestSearch_WithResults(t *testing.T) {
	r, env := setupSearchRouter(t)

	// 预置文档
	env.mock.docs["article-1"] = map[string]interface{}{
		"title":   "Go 语言入门教程",
		"content": "这是一篇关于 Go 语言的入门教程",
	}
	env.mock.docs["article-2"] = map[string]interface{}{
		"title":   "Python 入门",
		"content": "Python 入门指南",
	}

	req, _ := http.NewRequest("GET", "/api/search?q=Go&page=1&size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Message != "ok" {
		t.Errorf("message: 期望=ok, 实际=%s", resp.Message)
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("total: 期望=1, 实际=%d", resp.Pagination.Total)
	}
	if resp.Pagination.Page != 1 {
		t.Errorf("page: 期望=1, 实际=%d", resp.Pagination.Page)
	}
	if resp.Pagination.PageSize != 10 {
		t.Errorf("pageSize: 期望=10, 实际=%d", resp.Pagination.PageSize)
	}
}

func TestSearch_NoResults(t *testing.T) {
	r, env := setupSearchRouter(t)

	env.mock.docs["article-1"] = map[string]interface{}{
		"title":   "Go 教程",
		"content": "Go 语言",
	}

	req, _ := http.NewRequest("GET", "/api/search?q=Python&page=1&size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Message != "ok" {
		t.Errorf("message: 期望=ok, 实际=%s", resp.Message)
	}
	if resp.Pagination.Total != 0 {
		t.Errorf("total: 期望=0, 实际=%d", resp.Pagination.Total)
	}
}

func TestSearch_DefaultPageSize(t *testing.T) {
	r, env := setupSearchRouter(t)

	env.mock.docs["article-1"] = map[string]interface{}{
		"title":   "Go 教程",
		"content": "Go 语言",
	}

	// 不传 page 和 size，使用默认值
	req, _ := http.NewRequest("GET", "/api/search?q=Go", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Page != 1 {
		t.Errorf("page: 期望=1 (默认), 实际=%d", resp.Pagination.Page)
	}
	if resp.Pagination.PageSize != 10 {
		t.Errorf("pageSize: 期望=10 (默认), 实际=%d", resp.Pagination.PageSize)
	}
}

func TestSearch_ResultContainsHighlight(t *testing.T) {
	r, env := setupSearchRouter(t)

	env.mock.docs["article-1"] = map[string]interface{}{
		"title":   "Go 语言入门",
		"content": "这是一篇 Go 语言教程",
	}

	req, _ := http.NewRequest("GET", "/api/search?q=Go", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	articles, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("Data 类型错误，期望 []interface{}")
	}
	if len(articles) == 0 {
		t.Fatal("期望至少一篇搜索结果")
	}
	article := articles[0].(map[string]interface{})
	if _, ok := article["_id"]; !ok {
		t.Error("期望结果包含 _id 字段")
	}
	if _, ok := article["highlight"]; !ok {
		t.Error("期望结果包含 highlight 高亮字段")
	}
}

func TestSearch_ServiceError(t *testing.T) {
	r, env := setupSearchRouter(t)

	// 让 ES 返回错误
	env.mock.failPrefix = "articles/_search"

	req, _ := http.NewRequest("GET", "/api/search?q=Go", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("HTTP状态码: 期望=%d, 实际=%d", http.StatusInternalServerError, w.Code)
	}
}
