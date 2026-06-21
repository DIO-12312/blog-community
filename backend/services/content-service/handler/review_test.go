package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-community/content-service/repository"
	"blog-community/content-service/service"
	"blog-community/shared/cache"
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReviewHandlerTest(t *testing.T) (*ReviewHandler, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	db.AutoMigrate(&models.Article{}, &models.ReviewRecord{})

	g := &cache.Group{GroupMap: make(map[string]*cache.Call)}
	articleRepo := repository.NewArticleRepository(db, nil, g)
	reviewRepo := repository.NewReviewRepository(db)
	svc := service.NewReviewService(articleRepo, reviewRepo, nil)
	h := NewReviewHandler(svc)
	return h, db
}

func setupGinCtx(method, path, body string, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c, w
}

func TestSubmitForReview_Handler(t *testing.T) {
	h, db := newReviewHandlerTest(t)

	// 创建一篇草稿
	a := &models.Article{AuthorID: "user-1", Title: "测试", Content: "test", Status: models.StatusDraft}
	db.Create(a)

	c, w := setupGinCtx("POST", "/api/articles/"+a.ID+"/submit-review", "", map[string]string{
		"X-User-ID": "user-1",
	})
	c.Params = gin.Params{{Key: "id", Value: a.ID}}

	h.SubmitForReview(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200, 实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestSubmitForReview_NoAuth(t *testing.T) {
	h, _ := newReviewHandlerTest(t)

	c, w := setupGinCtx("POST", "/api/articles/xxx/submit-review", "", nil)
	c.Params = gin.Params{{Key: "id", Value: "xxx"}}

	h.SubmitForReview(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望 401, 实际=%d", w.Code)
	}
}

func TestReviewArticle_Handler(t *testing.T) {
	h, db := newReviewHandlerTest(t)

	a := &models.Article{AuthorID: "user-1", Title: "测试", Content: "test", Status: models.StatusPendingReview}
	db.Create(a)

	body := `{"action":"approved","comment":"可以发布"}`
	c, w := setupGinCtx("POST", "/api/admin/articles/"+a.ID+"/review", body, map[string]string{
		"X-User-ID": "admin-1",
	})
	c.Params = gin.Params{{Key: "id", Value: a.ID}}

	h.ReviewArticle(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200, 实际=%d, body=%s", w.Code, w.Body.String())
	}
	// 验证返回数据含 record
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["data"] == nil {
		t.Error("期望返回 data 字段")
	}
}

func TestGetReviewHistory_Handler(t *testing.T) {
	h, db := newReviewHandlerTest(t)

	a := &models.Article{AuthorID: "user-1", Title: "测试", Content: "test", Status: models.StatusDraft}
	db.Create(a)
	db.Create(&models.ReviewRecord{ArticleID: a.ID, ReviewerID: "admin-1", Action: models.ReviewActionApproved})

	c, w := setupGinCtx("GET", "/api/articles/"+a.ID+"/review-history", "", map[string]string{
		"X-User-ID": "user-1",
	})
	c.Params = gin.Params{{Key: "id", Value: a.ID}}

	h.GetReviewHistory(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200, 实际=%d", w.Code)
	}
}

func TestListPendingReviews_Handler(t *testing.T) {
	h, db := newReviewHandlerTest(t)

	for i := 0; i < 5; i++ {
		db.Create(&models.Article{AuthorID: "user-1", Title: "待审", Content: "test", Status: models.StatusPendingReview})
	}

	c, w := setupGinCtx("GET", "/api/admin/reviews/pending?page=1&size=3", "", nil)
	h.ListPendingReviews(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200, 实际=%d", w.Code)
	}
}
