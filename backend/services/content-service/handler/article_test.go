package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-community/content-service/repository"
	"blog-community/content-service/service"
	"blog-community/shared/cache"
	"blog-community/shared/events"
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 创建测试用的完整 handler 链
func newTestHandler(t *testing.T) (*ArticleHandler, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	err = db.AutoMigrate(&models.Article{})
	if err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}

	redisClient, err := cache.NewRedisClient("localhost:6379", "")
	if err != nil {
		t.Fatalf("无法连接 Redis: %v", err)
	}

	repo := repository.NewArticleRepository(db, redisClient)
	rmq := events.NewRabbitMQ()
	publisher := events.NewPublisher(rmq)
	svc := service.NewArticleService(repo, publisher)
	handler := NewArticleHandler(svc)

	cleanup := func() {
		redisClient.Close()
	}

	return handler, cleanup
}

// 通用请求辅助函数（无路径参数）
func performRequest(handler gin.HandlerFunc, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	return performRequestWithParams(handler, method, path, body, headers, nil)
}

// 通用请求辅助函数（带路径参数）
func performRequestWithParams(handler gin.HandlerFunc, method, path string, body string, headers map[string]string, params map[string]string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	c.Request = req

	if params != nil {
		var ginParams []gin.Param
		for k, v := range params {
			ginParams = append(ginParams, gin.Param{Key: k, Value: v})
		}
		c.Params = ginParams
	}

	handler(c)
	return w
}

// 创建文章并返回 ID 的辅助函数
func createAndGetID(t *testing.T, h *ArticleHandler, body string, userID string) string {
	t.Helper()
	w := performRequest(h.CreateArticle, "POST", "/api/articles", body, map[string]string{
		"X-User-ID": userID,
	})
	if w.Code != 201 {
		t.Fatalf("创建文章失败: %d, body=%s", w.Code, w.Body.String())
	}
	var resp models.ApiResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	articleMap := resp.Data.(map[string]any)
	return articleMap["id"].(string)
}

// ========== CreateArticle Handler 测试 ==========

func TestCreateArticle_Handler_Success(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	body := `{"title":"测试文章","content":"文章内容","summary":"摘要","category":"tech","tags":["go","redis"]}`
	w := performRequest(h.CreateArticle, "POST", "/api/articles", body, map[string]string{
		"X-User-ID": "user-001",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("状态码应为 201, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp models.ApiResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != http.StatusCreated {
		t.Errorf("响应 Code 应为 201, got %d", resp.Code)
	}
	if resp.Message != "文章创建成功" {
		t.Errorf("Message = %q", resp.Message)
	}
}

func TestCreateArticle_Handler_MissingAuth(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	body := `{"title":"测试","content":"内容"}`
	w := performRequest(h.CreateArticle, "POST", "/api/articles", body, nil)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("缺少认证应返回 401, got %d", w.Code)
	}
}

func TestCreateArticle_Handler_InvalidJSON(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequest(h.CreateArticle, "POST", "/api/articles", `{invalid json}`, map[string]string{
		"X-User-ID": "user-001",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效JSON应返回 400, got %d", w.Code)
	}
}

func TestCreateArticle_Handler_MissingRequired(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	body := `{"summary":"only summary"}`
	w := performRequest(h.CreateArticle, "POST", "/api/articles", body, map[string]string{
		"X-User-ID": "user-001",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("缺少必填字段应返回 400, got %d", w.Code)
	}
}

func TestCreateArticle_Handler_EmptyTitle(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	body := `{"title":"","content":"内容"}`
	w := performRequest(h.CreateArticle, "POST", "/api/articles", body, map[string]string{
		"X-User-ID": "user-001",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("空标题应返回 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

// ========== GetArticle Handler 测试 ==========

func TestGetArticle_Handler_Success(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"获取测试","content":"内容"}`, "user-001")

	w := performRequestWithParams(h.GetArticle, "GET", "/api/articles/"+articleID, "", nil, map[string]string{
		"id": articleID,
	})

	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestGetArticle_Handler_NotFound(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequestWithParams(h.GetArticle, "GET", "/api/articles/non-existent-uuid", "", nil, map[string]string{
		"id": "non-existent-uuid",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("不存在文章应返回 404, got %d", w.Code)
	}
}

// ========== ListArticles Handler 测试 ==========

func TestListArticles_Handler_Success(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequest(h.ListArticles, "GET", "/api/articles?page=1&size=10", "", nil)

	if w.Code != http.StatusOK {
		t.Errorf("ListArticles 应返回 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestListArticles_Handler_DefaultPagination(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequest(h.ListArticles, "GET", "/api/articles", "", nil)
	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200, got %d", w.Code)
	}
}

func TestListArticles_Handler_InvalidParams(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequest(h.ListArticles, "GET", "/api/articles?page=abc&size=xyz", "", nil)
	if w.Code != http.StatusOK {
		t.Errorf("无效分页参数应使用默认值, got %d", w.Code)
	}
}

// ========== ListByCategory Handler 测试 ==========

func TestListByCategory_Handler_Success(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequestWithParams(h.ListByCategory, "GET", "/api/articles/category/tech", "", nil, map[string]string{
		"category": "tech",
	})

	if w.Code != http.StatusOK {
		t.Errorf("状态码应为 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestListByCategory_Handler_ChineseCategory(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequestWithParams(h.ListByCategory, "GET", "/api/articles/category/科技", "", nil, map[string]string{
		"category": "科技",
	})

	if w.Code != http.StatusOK {
		t.Errorf("中文分类应正常: got %d", w.Code)
	}
}

// ========== EditArticle Handler 测试 ==========

func TestEditArticle_Handler_Success(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"原标题","content":"原内容"}`, "user-001")

	editBody := `{"title":"新标题","content":"新内容","summary":"新摘要","category":"life"}`
	w := performRequestWithParams(h.EditArticle, "PUT", "/api/articles/"+articleID, editBody, map[string]string{
		"X-User-ID": "user-001",
	}, map[string]string{
		"id": articleID,
	})

	if w.Code != http.StatusOK {
		t.Errorf("编辑应返回 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestEditArticle_Handler_NotAuthor(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"原标题","content":"原内容"}`, "user-001")

	editBody := `{"title":"新标题","content":"新内容","summary":"摘要","category":"tech"}`
	w := performRequestWithParams(h.EditArticle, "PUT", "/api/articles/"+articleID, editBody, map[string]string{
		"X-User-ID": "user-002",
	}, map[string]string{
		"id": articleID,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("非作者编辑应返回 400, got %d", w.Code)
	}
}

func TestEditArticle_Handler_MissingAuth(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	editBody := `{"title":"标题","content":"内容"}`
	w := performRequestWithParams(h.EditArticle, "PUT", "/api/articles/some-id", editBody, nil, map[string]string{
		"id": "some-id",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("缺少认证应返回错误, got %d", w.Code)
	}
}

// ========== PublishArticle Handler 测试 ==========

func TestPublishArticle_Handler_Success(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"待发布","content":"内容"}`, "user-001")

	w := performRequestWithParams(h.PublishArticle, "POST", "/api/articles/"+articleID+"/publish", "", map[string]string{
		"X-User-ID": "user-001",
	}, map[string]string{
		"id": articleID,
	})

	if w.Code != http.StatusOK {
		t.Errorf("发布应返回 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestPublishArticle_Handler_NotAuthor(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"待发布","content":"内容"}`, "user-001")

	w := performRequestWithParams(h.PublishArticle, "POST", "/api/articles/"+articleID+"/publish", "", map[string]string{
		"X-User-ID": "user-002",
	}, map[string]string{
		"id": articleID,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("非作者发布应返回 400, got %d", w.Code)
	}
}

// ========== DeleteArticle Handler 测试 ==========

func TestDeleteArticle_Handler_Success(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"待删除","content":"内容"}`, "user-001")

	w := performRequestWithParams(h.DeleteArticle, "DELETE", "/api/articles/"+articleID, "", map[string]string{
		"X-User-ID": "user-001",
	}, map[string]string{
		"id": articleID,
	})

	if w.Code != http.StatusOK {
		t.Errorf("删除应返回 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteArticle_Handler_NotAuthor(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"待删除","content":"内容"}`, "user-001")

	w := performRequestWithParams(h.DeleteArticle, "DELETE", "/api/articles/"+articleID, "", map[string]string{
		"X-User-ID": "user-002",
	}, map[string]string{
		"id": articleID,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("非作者删除应返回 400, got %d", w.Code)
	}
}

func TestDeleteArticle_Handler_MissingAuth(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	w := performRequestWithParams(h.DeleteArticle, "DELETE", "/api/articles/some-id", "", nil, map[string]string{
		"id": "some-id",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("缺少认证应返回错误, got %d", w.Code)
	}
}

// ========== parsePagination 测试 ==========

func TestParsePagination_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	page, size := parsePagination(c)

	if page != 1 {
		t.Errorf("默认 page 应为 1, got %d", page)
	}
	if size != 10 {
		t.Errorf("默认 size 应为 10, got %d", size)
	}
}

func TestParsePagination_ValidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test?page=3&size=20", nil)

	page, size := parsePagination(c)

	if page != 3 {
		t.Errorf("page 应为 3, got %d", page)
	}
	if size != 20 {
		t.Errorf("size 应为 20, got %d", size)
	}
}

func TestParsePagination_NegativeValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test?page=-1&size=-5", nil)

	page, size := parsePagination(c)

	if page != 1 {
		t.Errorf("负数 page 应使用默认值 1, got %d", page)
	}
	if size != 10 {
		t.Errorf("负数 size 应使用默认值 10, got %d", size)
	}
}

func TestParsePagination_SizeOverflow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test?page=1&size=100", nil)

	_, size := parsePagination(c)

	if size != 10 {
		t.Errorf("size 超过最大值 50 应使用默认值 10, got %d", size)
	}
}

func TestParsePagination_SizeAtBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		size     string
		expected int
	}{
		{"size=50 (边界值)", "50", 50},
		{"size=51 (超过边界)", "51", 10},
		{"size=1 (最小值)", "1", 1},
		{"size=0 (零)", "0", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test?size="+tt.size, nil)

			_, size := parsePagination(c)
			if size != tt.expected {
				t.Errorf("size = %d, want %d", size, tt.expected)
			}
		})
	}
}

func TestParsePagination_NonNumeric(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test?page=abc&size=xyz", nil)

	page, size := parsePagination(c)

	if page != 1 {
		t.Errorf("非数字 page 应使用默认值, got %d", page)
	}
	if size != 10 {
		t.Errorf("非数字 size 应使用默认值, got %d", size)
	}
}

// ========== 端到端流程测试 ==========

func TestFullArticleLifecycle(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	user := "user-lifecycle"

	// 1. 创建文章
	articleID := createAndGetID(t, h, `{"title":"生命周期测试","content":"E2E内容","summary":"E2E摘要","category":"tech"}`, user)

	// 2. 获取文章
	w := performRequestWithParams(h.GetArticle, "GET", "/api/articles/"+articleID, "", nil, map[string]string{
		"id": articleID,
	})
	if w.Code != 200 {
		t.Fatalf("步骤2 获取失败: %d", w.Code)
	}

	// 3. 编辑文章
	editBody := `{"title":"编辑后的标题","content":"编辑后的内容","summary":"编辑摘要","category":"life"}`
	w = performRequestWithParams(h.EditArticle, "PUT", "/api/articles/"+articleID, editBody, map[string]string{
		"X-User-ID": user,
	}, map[string]string{
		"id": articleID,
	})
	if w.Code != 200 {
		t.Fatalf("步骤3 编辑失败: %d, %s", w.Code, w.Body.String())
	}

	// 4. 发布文章
	w = performRequestWithParams(h.PublishArticle, "POST", "/api/articles/"+articleID+"/publish", "", map[string]string{
		"X-User-ID": user,
	}, map[string]string{
		"id": articleID,
	})
	if w.Code != 200 {
		t.Fatalf("步骤4 发布失败: %d, %s", w.Code, w.Body.String())
	}

	// 5. 删除文章
	w = performRequestWithParams(h.DeleteArticle, "DELETE", "/api/articles/"+articleID, "", map[string]string{
		"X-User-ID": user,
	}, map[string]string{
		"id": articleID,
	})
	if w.Code != 200 {
		t.Fatalf("步骤5 删除失败: %d", w.Code)
	}

	// 6. 确认已删除
	w = performRequestWithParams(h.GetArticle, "GET", "/api/articles/"+articleID, "", nil, map[string]string{
		"id": articleID,
	})
	if w.Code != 404 {
		t.Errorf("步骤6 已删除文章应返回 404, got %d", w.Code)
	}
}

// TestContextPropagation 验证 context 传递
func TestContextPropagation(t *testing.T) {
	h, cleanup := newTestHandler(t)
	defer cleanup()

	articleID := createAndGetID(t, h, `{"title":"Context测试","content":"内容"}`, "user-ctx")

	// 验证 context 正确传递（GetArticle 使用 c.Request.Context()）
	w := performRequestWithParams(h.GetArticle, "GET", "/api/articles/"+articleID, "", nil, map[string]string{
		"id": articleID,
	})
	if w.Code != 200 {
		t.Errorf("context 传递验证失败: %d, %s", w.Code, w.Body.String())
	}

	// 验证 context 正确传递（EditArticle 使用 c.Request.Context()）
	editBody := `{"title":"Context编辑","content":"内容","summary":"","category":"tech"}`
	w = performRequestWithParams(h.EditArticle, "PUT", "/api/articles/"+articleID, editBody, map[string]string{
		"X-User-ID": "user-ctx",
	}, map[string]string{
		"id": articleID,
	})
	if w.Code != 200 {
		t.Errorf("Edit context 传递失败: %d", w.Code)
	}

	// 验证 context 在 PublishArticle 中传递
	ctx := context.Background()
	_ = ctx
	w = performRequestWithParams(h.PublishArticle, "POST", "/api/articles/"+articleID+"/publish", "", map[string]string{
		"X-User-ID": "user-ctx",
	}, map[string]string{
		"id": articleID,
	})
	if w.Code != 200 {
		t.Errorf("Publish context 传递失败: %d", w.Code)
	}
}
