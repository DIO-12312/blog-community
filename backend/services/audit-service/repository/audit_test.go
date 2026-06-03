package repository

import (
	"testing"
	"time"

	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}
	return db
}

func TestCreateAuditLog(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	log := &models.AuditLog{
		UserID:     "user-1",
		Action:     "article.published",
		Resource:   "article",
		ResourceID: "article-1",
		Detail:     `{"title":"测试"}`,
	}

	err := repo.Create(log)
	if err != nil {
		t.Fatalf("Create() 失败: %v", err)
	}

	if log.ID == "" {
		t.Error("期望 ID 自动生成，但 ID 为空")
	}

	var saved models.AuditLog
	if err := db.First(&saved, "id = ?", log.ID).Error; err != nil {
		t.Fatalf("从数据库读取失败: %v", err)
	}
	if saved.Action != "article.published" {
		t.Errorf("Action: 期望=article.published, 实际=%s", saved.Action)
	}
}

func TestQuery_All(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	// 创建多条日志
	for i := 0; i < 5; i++ {
		db.Create(&models.AuditLog{
			UserID:   "user-1",
			Action:   "test.action",
			Resource: "test",
		})
	}

	logs, total, err := repo.Query("", "", "", 1, 10)
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}
	if total != 5 {
		t.Errorf("total: 期望=5, 实际=%d", total)
	}
	if len(logs) != 5 {
		t.Errorf("len(logs): 期望=5, 实际=%d", len(logs))
	}
}

func TestQuery_FilterByUserID(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	db.Create(&models.AuditLog{UserID: "user-1", Action: "login", Resource: "user"})
	db.Create(&models.AuditLog{UserID: "user-1", Action: "logout", Resource: "user"})
	db.Create(&models.AuditLog{UserID: "user-2", Action: "login", Resource: "user"})

	logs, total, err := repo.Query("user-1", "", "", 1, 10)
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("total: 期望=2, 实际=%d", total)
	}
	if len(logs) != 2 {
		t.Errorf("len(logs): 期望=2, 实际=%d", len(logs))
	}
}

func TestQuery_FilterByAction(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	db.Create(&models.AuditLog{UserID: "user-1", Action: "article.published", Resource: "article"})
	db.Create(&models.AuditLog{UserID: "user-1", Action: "article.deleted", Resource: "article"})
	db.Create(&models.AuditLog{UserID: "user-1", Action: "comment.created", Resource: "comment"})

	_, total, err := repo.Query("", "article.published", "", 1, 10)
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}
	if total != 1 {
		t.Errorf("total: 期望=1, 实际=%d", total)
	}
}

func TestQuery_FilterByResource(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	db.Create(&models.AuditLog{UserID: "user-1", Action: "create", Resource: "article"})
	db.Create(&models.AuditLog{UserID: "user-1", Action: "create", Resource: "comment"})

	logs, total, err := repo.Query("", "", "comment", 1, 10)
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}
	if total != 1 {
		t.Errorf("total: 期望=1, 实际=%d", total)
	}
	if logs[0].Resource != "comment" {
		t.Errorf("Resource: 期望=comment, 实际=%s", logs[0].Resource)
	}
}

func TestQuery_Pagination(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	for i := 0; i < 5; i++ {
		db.Create(&models.AuditLog{
			UserID:   "user-1",
			Action:   "action",
			Resource: "resource",
		})
	}

	// 第1页2条
	logs, total, err := repo.Query("", "", "", 1, 2)
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}
	if total != 5 {
		t.Errorf("total: 期望=5, 实际=%d", total)
	}
	if len(logs) != 2 {
		t.Errorf("len: 期望=2, 实际=%d", len(logs))
	}

	// 第3页1条
	logs2, total2, _ := repo.Query("", "", "", 3, 2)
	if total2 != 5 {
		t.Errorf("第3页 total: 期望=5, 实际=%d", total2)
	}
	if len(logs2) != 1 {
		t.Errorf("第3页 len: 期望=1, 实际=%d", len(logs2))
	}
}

func TestQuery_NoResults(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	logs, total, err := repo.Query("nonexistent", "", "", 1, 10)
	if err != nil {
		t.Fatalf("Query() 失败: %v", err)
	}
	if total != 0 {
		t.Errorf("total: 期望=0, 实际=%d", total)
	}
	if len(logs) != 0 {
		t.Errorf("len(logs): 期望=0, 实际=%d", len(logs))
	}
}

func TestQuery_OrderDesc(t *testing.T) {
	db := newTestDB(t)
	repo := NewAuditRepository(db)

	l1 := &models.AuditLog{UserID: "user-1", Action: "first", Resource: "test"}
	l2 := &models.AuditLog{UserID: "user-1", Action: "second", Resource: "test"}
	l3 := &models.AuditLog{UserID: "user-1", Action: "third", Resource: "test"}
	db.Create(l1)
	time.Sleep(10 * time.Millisecond)
	db.Create(l2)
	time.Sleep(10 * time.Millisecond)
	db.Create(l3)

	logs, _, _ := repo.Query("", "", "", 1, 10)
	if len(logs) != 3 {
		t.Fatalf("期望 3 条记录")
	}
	// 应该按创建时间倒序：l3 > l2 > l1
	if logs[0].Action != "third" {
		t.Errorf("第1条: 期望=third, 实际=%s", logs[0].Action)
	}
	if logs[1].Action != "second" {
		t.Errorf("第2条: 期望=second, 实际=%s", logs[1].Action)
	}
	if logs[2].Action != "first" {
		t.Errorf("第3条: 期望=first, 实际=%s", logs[2].Action)
	}
}
