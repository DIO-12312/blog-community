package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog-community/notification-service/repository"
	"blog-community/notification-service/service"
	"blog-community/shared/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.Notification{}); err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}

	repo := repository.NewNotificationRepository(db)
	svc := service.NewNotificationService(repo, nil)
	handler := NewNotificationHandler(svc)

	r := gin.New()
	r.GET("/api/notifications", handler.GetNotifications)
	r.PUT("/api/notifications/read-all", handler.MarkAllAsRead)
	r.PUT("/api/notifications/:id/read", handler.MarkAsRead)
	r.GET("/api/notifications/unread-count", handler.GetUnreadCount)

	return r, db
}

func TestGetNotifications_Empty(t *testing.T) {
	r, _ := setupRouter(t)

	req, _ := http.NewRequest("GET", "/api/notifications?page=1&size=10", nil)
	req.Header.Set("X-User-ID", "user-1")
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
	if resp.Pagination.Page != 1 {
		t.Errorf("page: 期望=1, 实际=%d", resp.Pagination.Page)
	}
	if resp.Pagination.PageSize != 10 {
		t.Errorf("pageSize: 期望=10, 实际=%d", resp.Pagination.PageSize)
	}
}

func TestGetNotifications_WithData(t *testing.T) {
	r, db := setupRouter(t)

	// 创建3条 user-1 的通知
	for i := 0; i < 3; i++ {
		db.Create(&models.Notification{
			UserID:  "user-1",
			Type:    "comment",
			Content: "通知内容",
		})
	}

	req, _ := http.NewRequest("GET", "/api/notifications?page=1&size=10", nil)
	req.Header.Set("X-User-ID", "user-1")
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
	if resp.Message != "ok" {
		t.Errorf("message: 期望=ok, 实际=%s", resp.Message)
	}
}

func TestGetNotifications_Pagination(t *testing.T) {
	r, db := setupRouter(t)

	for i := 0; i < 5; i++ {
		db.Create(&models.Notification{
			UserID:  "user-1",
			Type:    "like",
			Content: "分页测试",
		})
	}

	// 第1页 2条
	req, _ := http.NewRequest("GET", "/api/notifications?page=1&size=2", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp models.PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Pagination.Total != 5 {
		t.Errorf("total: 期望=5, 实际=%d", resp.Pagination.Total)
	}
	if resp.Pagination.Page != 1 {
		t.Errorf("page: 期望=1, 实际=%d", resp.Pagination.Page)
	}
	if resp.Pagination.PageSize != 2 {
		t.Errorf("pageSize: 期望=2, 实际=%d", resp.Pagination.PageSize)
	}

	// 第2页 2条
	req2, _ := http.NewRequest("GET", "/api/notifications?page=2&size=2", nil)
	req2.Header.Set("X-User-ID", "user-1")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 models.PaginatedResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Pagination.Total != 5 {
		t.Errorf("第2页 total: 期望=5, 实际=%d", resp2.Pagination.Total)
	}

	// 第3页（只有1条剩余）
	req3, _ := http.NewRequest("GET", "/api/notifications?page=3&size=2", nil)
	req3.Header.Set("X-User-ID", "user-1")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	var resp3 models.PaginatedResponse
	json.Unmarshal(w3.Body.Bytes(), &resp3)
	if resp3.Pagination.Total != 5 {
		t.Errorf("第3页 total: 期望=5, 实际=%d", resp3.Pagination.Total)
	}
}

func TestMarkAsRead(t *testing.T) {
	r, db := setupRouter(t)

	n := &models.Notification{
		UserID:  "user-1",
		Type:    "comment",
		Content: "待已读",
	}
	db.Create(n)

	// 标记已读
	req, _ := http.NewRequest("PUT", "/api/notifications/"+n.ID+"/read", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.ApiResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Message != "已标记已读" {
		t.Errorf("message: 期望=已标记已读, 实际=%s", resp.Message)
	}

	// 验证数据库
	var updated models.Notification
	db.First(&updated, "id = ?", n.ID)
	if !updated.IsRead {
		t.Error("IsRead: 期望=true, 实际=false")
	}
}

func TestMarkAsRead_WrongUser(t *testing.T) {
	r, db := setupRouter(t)

	n := &models.Notification{
		UserID:  "user-1",
		Type:    "comment",
		Content: "待已读",
	}
	db.Create(n)

	// 用错误的用户标记 → 应返回 404
	req, _ := http.NewRequest("PUT", "/api/notifications/"+n.ID+"/read", nil)
	req.Header.Set("X-User-ID", "user-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("wrong user status: 期望=%d, 实际=%d", http.StatusNotFound, w.Code)
	}

	// 验证通知仍为未读
	var updated models.Notification
	db.First(&updated, "id = ?", n.ID)
	if updated.IsRead {
		t.Error("IsRead: 期望=false (未读状态不变), 实际=true")
	}
}

func TestMarkAllAsRead(t *testing.T) {
	r, db := setupRouter(t)

	db.Create(&models.Notification{UserID: "user-1", Type: "comment", Content: "未读1"})
	db.Create(&models.Notification{UserID: "user-1", Type: "like", Content: "未读2"})

	req, _ := http.NewRequest("PUT", "/api/notifications/read-all", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.ApiResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Message != "全部已读" {
		t.Errorf("message: 期望=全部已读, 实际=%s", resp.Message)
	}

	// 验证未读数
	var unreadCount int64
	db.Model(&models.Notification{}).Where("user_id = ? AND is_read = false", "user-1").Count(&unreadCount)
	if unreadCount != 0 {
		t.Errorf("未读数: 期望=0, 实际=%d", unreadCount)
	}
}

func TestGetUnreadCount(t *testing.T) {
	r, db := setupRouter(t)

	db.Create(&models.Notification{UserID: "user-1", Type: "comment", Content: "未读1"})
	db.Create(&models.Notification{UserID: "user-1", Type: "like", Content: "未读2"})
	n3 := &models.Notification{UserID: "user-1", Type: "follow", Content: "已读"}
	n3.IsRead = true
	db.Create(n3)

	req, _ := http.NewRequest("GET", "/api/notifications/unread-count", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.ApiResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Data 类型错误")
	}
	count, ok := dataMap["count"].(float64)
	if !ok {
		t.Fatalf("count 字段类型错误")
	}
	if int64(count) != 2 {
		t.Errorf("count: 期望=2, 实际=%v", dataMap["count"])
	}
}

func TestGetUnreadCount_AllRead(t *testing.T) {
	r, _ := setupRouter(t)

	// 无通知的用户
	req, _ := http.NewRequest("GET", "/api/notifications/unread-count", nil)
	req.Header.Set("X-User-ID", "new-user")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("HTTP状态码: 期望=200, 实际=%d", w.Code)
	}

	var resp models.ApiResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	dataMap := resp.Data.(map[string]interface{})
	if dataMap["count"].(float64) != 0 {
		t.Errorf("count: 期望=0, 实际=%v", dataMap["count"])
	}
}
