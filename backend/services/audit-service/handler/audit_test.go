package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog-community/audit-service/repository"
	"blog-community/audit-service/service"
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}

	repo := repository.NewAuditRepository(db)
	svc := service.NewAuditService(repo, nil)
	h := NewAuditHandler(svc)

	r := gin.New()
	r.GET("/api/audit-logs", h.Query)

	return r, db
}

func TestAuditLog_Empty(t *testing.T) {
	r, _ := setupAuditRouter(t)

	req, _ := http.NewRequest("GET", "/api/audit-logs?page=1&size=20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Total != 0 {
		t.Errorf("total: 期望=0, 实际=%d", resp.Pagination.Total)
	}
	if resp.Message != "ok" {
		t.Errorf("message: 期望=ok, 实际=%s", resp.Message)
	}
}

func TestAuditLog_WithData(t *testing.T) {
	r, db := setupAuditRouter(t)

	for i := 0; i < 3; i++ {
		db.Create(&models.AuditLog{
			UserID:   "user-1",
			Action:   "test.action",
			Resource: "test",
		})
	}

	req, _ := http.NewRequest("GET", "/api/audit-logs?page=1&size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Total != 3 {
		t.Errorf("total: 期望=3, 实际=%d", resp.Pagination.Total)
	}
}

func TestAuditLog_FilterByUserID(t *testing.T) {
	r, db := setupAuditRouter(t)

	db.Create(&models.AuditLog{UserID: "user-1", Action: "login", Resource: "user"})
	db.Create(&models.AuditLog{UserID: "user-2", Action: "login", Resource: "user"})

	req, _ := http.NewRequest("GET", "/api/audit-logs?user_id=user-1&page=1&size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Total != 1 {
		t.Errorf("total: 期望=1, 实际=%d", resp.Pagination.Total)
	}
}

func TestAuditLog_FilterByAction(t *testing.T) {
	r, db := setupAuditRouter(t)

	db.Create(&models.AuditLog{UserID: "user-1", Action: "article.published", Resource: "article"})
	db.Create(&models.AuditLog{UserID: "user-1", Action: "article.deleted", Resource: "article"})

	req, _ := http.NewRequest("GET", "/api/audit-logs?action=article.published&page=1&size=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Total != 1 {
		t.Errorf("total: 期望=1, 实际=%d", resp.Pagination.Total)
	}
}

func TestAuditLog_DefaultPageSize(t *testing.T) {
	r, _ := setupAuditRouter(t)

	req, _ := http.NewRequest("GET", "/api/audit-logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Page != 1 {
		t.Errorf("page: 期望=1, 实际=%d", resp.Pagination.Page)
	}
	if resp.Pagination.PageSize != 20 {
		t.Errorf("pageSize: 期望=20, 实际=%d", resp.Pagination.PageSize)
	}
}
