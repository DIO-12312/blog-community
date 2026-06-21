package repository

import (
	"fmt"
	"testing"

	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReviewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.Article{}, &models.ReviewRecord{}); err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}
	return db
}

func createTestArticle(t *testing.T, db *gorm.DB, authorID, title, status string) *models.Article {
	t.Helper()
	a := &models.Article{
		AuthorID: authorID,
		Title:    title,
		Content:  "test content",
		Status:   status,
	}
	if err := db.Create(a).Error; err != nil {
		t.Fatalf("创建测试文章失败: %v", err)
	}
	return a
}

func TestReviewRepo_Create(t *testing.T) {
	db := newReviewTestDB(t)
	repo := NewReviewRepository(db)

	comment := "审稿意见"
	record := &models.ReviewRecord{
		ArticleID:  "article-1",
		ReviewerID: "admin-1",
		Action:     models.ReviewActionApproved,
		Comment:    &comment,
	}
	if err := repo.Create(record); err != nil {
		t.Fatalf("Create() 失败: %v", err)
	}
	if record.ID == "" {
		t.Error("期望 ID 自动生成，但 ID 为空")
	}

	// 验证 Comment 为 NULL 时也能创建
	record2 := &models.ReviewRecord{
		ArticleID:  "article-2",
		ReviewerID: "admin-1",
		Action:     models.ReviewActionRejected,
		Comment:    nil,
	}
	if err := repo.Create(record2); err != nil {
		t.Fatalf("Create(nil comment) 失败: %v", err)
	}
}

func TestReviewRepo_GetByArticleID(t *testing.T) {
	db := newReviewTestDB(t)
	repo := NewReviewRepository(db)

	// 为 article-1 创建 3 条审稿记录
	for i := 0; i < 3; i++ {
		comment := fmt.Sprintf("意见 %d", i)
		repo.Create(&models.ReviewRecord{
			ArticleID:  "article-1",
			ReviewerID: "admin-1",
			Action:     models.ReviewActionApproved,
			Comment:    &comment,
		})
	}
	// 为 article-2 创建 2 条（不应被查出）
	for i := 0; i < 2; i++ {
		repo.Create(&models.ReviewRecord{
			ArticleID:  "article-2",
			ReviewerID: "admin-2",
			Action:     models.ReviewActionRejected,
		})
	}

	records, err := repo.GetByArticleID("article-1")
	if err != nil {
		t.Fatalf("GetByArticleID() 失败: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("期望 3 条记录, 实际=%d", len(records))
	}

	// 验证倒序
	if len(records) >= 2 && records[0].CreatedAt.Before(records[1].CreatedAt) {
		t.Error("期望按 created_at DESC 排序")
	}

	// 查询无记录的文章
	empty, err := repo.GetByArticleID("article-99")
	if err != nil {
		t.Fatalf("GetByArticleID() 失败: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("期望 0 条, 实际=%d", len(empty))
	}
}

func TestReviewRepo_ListPendingArticles(t *testing.T) {
	db := newReviewTestDB(t)
	repo := NewReviewRepository(db)

	// 创建 5 篇 pending_review + 3 篇其他状态
	for i := 0; i < 5; i++ {
		createTestArticle(t, db, fmt.Sprintf("user-%d", i), fmt.Sprintf("待审 %d", i), models.StatusPendingReview)
	}
	for i := 0; i < 3; i++ {
		createTestArticle(t, db, fmt.Sprintf("user-%d", i+10), fmt.Sprintf("已发布 %d", i), models.StatusPublished)
	}

	tests := []struct {
		name        string
		page, size  int
		expectCount int
		expectTotal int64
	}{
		{"第1页2条", 1, 2, 2, 5},
		{"第2页2条", 2, 2, 2, 5},
		{"第3页2条", 3, 2, 1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			articles, total, err := repo.ListPendingArticles(tt.page, tt.size)
			if err != nil {
				t.Fatalf("ListPendingArticles() 失败: %v", err)
			}
			if total != tt.expectTotal {
				t.Errorf("total: 期望=%d, 实际=%d", tt.expectTotal, total)
			}
			if len(articles) != tt.expectCount {
				t.Errorf("count: 期望=%d, 实际=%d", tt.expectCount, len(articles))
			}
			for _, a := range articles {
				if a.Status != models.StatusPendingReview {
					t.Errorf("期望所有文章 status=pending_review, 实际=%s", a.Status)
				}
			}
		})
	}
}
