package repository

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
)

// mockESServer 模拟 Elasticsearch HTTP 服务
type mockESServer struct {
	indices    map[string]bool            // 索引名 -> 是否存在
	docs       map[string]map[string]interface{} // docID -> document
	failPrefix string                     // 对该路径前缀返回 500
	nextCode   int                        // 下次响应的 HTTP 状态码（0 = 使用默认行为，仅用于已被路由的请求）
}

func newMockES() (*elasticsearch.Client, *mockESServer, func()) {
	m := &mockESServer{
		indices: make(map[string]bool),
		docs:    make(map[string]map[string]interface{}),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ES v8 客户端要求所有响应必须包含此 header
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.Header().Set("Content-Type", "application/json")

		// 允许覆盖状态码用于测试错误场景
		if m.nextCode != 0 {
			code := m.nextCode
			m.nextCode = 0
			w.WriteHeader(code)
			if code >= 400 {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "mock error",
				})
			}
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")

		// ES v8 客户端产品检查: GET / 或 HEAD /
		if path == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version": map[string]interface{}{
					"number": "8.0.0-mock",
				},
			})
			return
		}

		// failPrefix: 对该路径返回 500
		if m.failPrefix != "" && strings.HasPrefix(path, m.failPrefix) {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "mock error",
			})
			return
		}

		switch {
		// HEAD /articles — 检查索引是否存在
		case r.Method == "HEAD" && path == "articles":
			if m.indices["articles"] {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(404)
			}

		// PUT /articles — 创建索引
		case r.Method == "PUT" && path == "articles":
			m.indices["articles"] = true
			json.NewEncoder(w).Encode(map[string]interface{}{
				"acknowledged": true,
			})

		// PUT /articles/_doc/{id} — 索引文档
		case r.Method == "PUT" && strings.HasPrefix(path, "articles/_doc/"):
			docID := strings.TrimPrefix(path, "articles/_doc/")
			docID = strings.TrimSuffix(docID, "?refresh=true")
			// 读取请求体
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			m.docs[docID] = body
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": "created",
				"_id":    docID,
			})

		// DELETE /articles/_doc/{id} — 删除文档
		case r.Method == "DELETE" && strings.HasPrefix(path, "articles/_doc/"):
			docID := strings.TrimPrefix(path, "articles/_doc/")
			result := "not_found"
			if _, ok := m.docs[docID]; ok {
				delete(m.docs, docID)
				result = "deleted"
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": result,
			})

		// POST /articles/_search — 搜索
		case r.Method == "POST" && strings.HasSuffix(path, "/_search"):
			var query map[string]interface{}
			json.NewDecoder(r.Body).Decode(&query)

			// 简单模拟搜索：在所有文档中做字符串包含匹配
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
						"_id":     id,
						"_source": doc,
					}
					// 模拟高亮
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
					"total": map[string]interface{}{
						"value": len(hits),
					},
					"hits": hits,
				},
			})

		default:
			w.WriteHeader(404)
		}
	})

	ts := httptest.NewServer(handler)

	cfg := elasticsearch.Config{
		Addresses: []string{ts.URL},
	}
	client, _ := elasticsearch.NewClient(cfg)

	return client, m, ts.Close
}

func TestEnsureIndex_AlreadyExists(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	m.indices["articles"] = true
	repo := NewSearchRepository(client)

	err := repo.EnsureIndex()
	if err != nil {
		t.Fatalf("EnsureIndex() 失败（索引已存在）: %v", err)
	}
}

func TestEnsureIndex_Create(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	repo := NewSearchRepository(client)

	err := repo.EnsureIndex()
	if err != nil {
		t.Fatalf("EnsureIndex() 失败: %v", err)
	}

	if !m.indices["articles"] {
		t.Error("期望索引 articles 被创建，但未被创建")
	}
}

func TestEnsureIndex_ExistsCheckError(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	m.failPrefix = "articles" // HEAD /articles 返回 500（路径匹配前缀）
	repo := NewSearchRepository(client)

	err := repo.EnsureIndex()
	if err == nil {
		t.Error("期望 EnsureIndex() 返回错误，实际返回 nil")
	}
}

func TestIndexArticle(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	repo := NewSearchRepository(client)

	data := map[string]interface{}{
		"title":   "测试文章",
		"content": "这是内容",
	}
	err := repo.IndexArticle("article-1", data)
	if err != nil {
		t.Fatalf("IndexArticle() 失败: %v", err)
	}

	// 验证文档被存储
	if doc, ok := m.docs["article-1"]; !ok {
		t.Error("期望文档被存储，但未找到")
	} else if doc["title"] != "测试文章" {
		t.Errorf("title: 期望=测试文章, 实际=%v", doc["title"])
	}
}

func TestIndexArticle_Error(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	m.nextCode = 500
	repo := NewSearchRepository(client)

	err := repo.IndexArticle("article-1", map[string]interface{}{})
	if err == nil {
		t.Error("期望 IndexArticle() 返回错误，实际返回 nil")
	}
}

func TestDeleteArticle(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	// 先索引一个文档
	m.docs["article-1"] = map[string]interface{}{
		"title": "待删除",
	}

	repo := NewSearchRepository(client)

	err := repo.DeleteArticle("article-1")
	if err != nil {
		t.Fatalf("DeleteArticle() 失败: %v", err)
	}

	if _, ok := m.docs["article-1"]; ok {
		t.Error("期望文档被删除，但仍存在")
	}
}

func TestDeleteArticle_NotFound(t *testing.T) {
	client, _, close := newMockES()
	defer close()

	repo := NewSearchRepository(client)

	// 删除不存在的文档，只要能成功执行即可（ES 不报错）
	err := repo.DeleteArticle("non-existent")
	if err != nil {
		t.Errorf("删除不存在的文档不应报错: %v", err)
	}
}

func TestSearch_WithResults(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	// 预置文档
	m.docs["article-1"] = map[string]interface{}{
		"title":   "Go 语言入门",
		"content": "Go 是一门简洁高效的编程语言",
	}
	m.docs["article-2"] = map[string]interface{}{
		"title":   "Python 教程",
		"content": "Python 适合初学者",
	}
	m.docs["article-3"] = map[string]interface{}{
		"title":   "高级 Go 编程",
		"content": "深入学习 Go 语言特性",
	}

	repo := NewSearchRepository(client)

	result, err := repo.Search("Go", 1, 10)
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("total: 期望=2, 实际=%d", result.Total)
	}
	if len(result.Articles) != 2 {
		t.Errorf("len(articles): 期望=2, 实际=%d", len(result.Articles))
	}
	// 验证高亮
	for _, article := range result.Articles {
		if _, ok := article["_id"]; !ok {
			t.Error("期望文章包含 _id 字段")
		}
		if _, ok := article["highlight"]; !ok {
			t.Error("期望文章包含 highlight 字段")
		}
	}
}

func TestSearch_NoResults(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	m.docs["article-1"] = map[string]interface{}{
		"title":   "Go 语言入门",
		"content": "内容",
	}

	repo := NewSearchRepository(client)

	result, err := repo.Search("Python", 1, 10)
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}

	if result.Total != 0 {
		t.Errorf("total: 期望=0, 实际=%d", result.Total)
	}
	if len(result.Articles) != 0 {
		t.Errorf("len(articles): 期望=0, 实际=%d", len(result.Articles))
	}
}

func TestSearch_Pagination(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	// 预置 5 篇 Go 文章
	for i := 0; i < 5; i++ {
		m.docs["article-"+string(rune('0'+i))] = map[string]interface{}{
			"title":   "Go 教程",
			"content": "Go 编程",
		}
	}

	repo := NewSearchRepository(client)

	// 模拟 search body 中的 from/size — mock 暂不处理分页，
	// 但验证结果可正常返回
	result, err := repo.Search("Go", 1, 2)
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}
	if result.Total == 0 {
		t.Error("期望 total > 0")
	}
}

func TestSearch_Error(t *testing.T) {
	client, m, close := newMockES()
	defer close()

	m.nextCode = 500
	repo := NewSearchRepository(client)

	_, err := repo.Search("test", 1, 10)
	if err == nil {
		t.Error("期望 Search() 返回错误，实际返回 nil")
	}
}

func TestSearchResult_Types(t *testing.T) {
	// 验证 SearchResult 结构体
	articles := []map[string]interface{}{
		{"title": "test"},
	}
	sr := &SearchResult{
		Articles: articles,
		Total:    1,
	}
	if sr.Total != 1 {
		t.Errorf("Total: 期望=1, 实际=%d", sr.Total)
	}
	if len(sr.Articles) != 1 {
		t.Errorf("len(Articles): 期望=1, 实际=%d", len(sr.Articles))
	}
}
