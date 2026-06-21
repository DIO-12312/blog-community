# 审稿系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在博客社区平台上实现编辑审核制审稿系统——作者提交文章审稿，管理员审批，全链路通知。

**Architecture:** 扩展 content-service（不新建微服务），新增 ReviewRecord 模型、ReviewRepository/ReviewService/ReviewHandler 三层。审稿通过复用已有 `article.published` 事件，驳回触发新事件 `article.review_rejected`。

**Tech Stack:** Go + Gin + GORM (后端), Vue 3 + TypeScript + Pinia (前端), RabbitMQ (事件通知)

### Task 0: Git 分支管理 
- [ ] **Step 1: 创建新分支**

```bash
git add .
git commit -m "审核功能: 2026-06-21 审核系统开始"
git checkout -b 审核功能
```


---

### Task 1: 数据模型与常量

**Files:**
- Create: `backend/shared/models/review.go`
- Modify: `backend/shared/models/articles.go:24-32`
- Modify: `backend/shared/events/types.go:6-13`

- [ ] **Step 1: 创建 ReviewRecord 模型 + 常量**

```go
// backend/shared/models/review.go
package models

const (
	ReviewActionApproved = "approved"
	ReviewActionRejected = "rejected"
)

// ReviewRecord 审稿记录
type ReviewRecord struct {
	BaseModel
	ArticleID  string  `gorm:"index:idx_review_records_article;size:36" json:"article_id"`
	ReviewerID string  `gorm:"index:idx_review_records_reviewer;size:36" json:"reviewer_id"`
	Action     string  `gorm:"size:20" json:"action"`
	Comment    *string `gorm:"type:text" json:"comment"`
}
```

- [ ] **Step 2: 扩展 Article 状态枚举**

在 `backend/shared/models/articles.go` 中，将 `StatusDraft` 块改为：

在现有 const 块中追加一行：

```go
StatusPendingReview = "pending_review"
```

不修改已有的 `StatusDraft`、`StatusPublished`、`StatusDelete`。

- [ ] **Step 3: 新增事件类型常量**

在 `backend/shared/events/types.go` 的 const 块中追加：

```go
EventArticleSubmittedForReview = "article.submitted_for_review"
EventArticleReviewRejected     = "article.review_rejected"
```

- [ ] **Step 4: 运行现有测试确保无破坏**

Run: `cd backend/shared && go test ./...`
Expected: PASS（新增常量及模型不影响已有测试）

- [ ] **Step 5: Commit**

```bash
git add backend/shared/models/review.go backend/shared/models/articles.go backend/shared/events/types.go
git commit -m "feat: 新增 ReviewRecord 模型、审稿状态常量和事件类型"
```

---

### Task 2: ReviewRepository 数据访问层

**Files:**
- Create: `backend/services/content-service/repository/review.go`

- [ ] **Step 1: 创建 ReviewRepository**

```go
// backend/services/content-service/repository/review.go
package repository

import (
	"blog-community/shared/models"

	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// Create 创建审稿记录
func (r *ReviewRepository) Create(record *models.ReviewRecord) error {
	return r.db.Create(record).Error
}

// GetByArticleID 获取某文章的所有审稿记录（按时间倒序）
func (r *ReviewRepository) GetByArticleID(articleID string) ([]models.ReviewRecord, error) {
	var records []models.ReviewRecord
	err := r.db.Where("article_id = ?", articleID).
		Order("created_at DESC").
		Find(&records).Error
	return records, err
}

// ListPendingArticles 获取待审文章列表（分页），返回文章 + 作者信息
func (r *ReviewRepository) ListPendingArticles(page, size int) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	query := r.db.Model(&models.Article{}).Where("status = ?", models.StatusPendingReview)
	query.Count(&total)

	err := query.
		Order("updated_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&articles).Error

	return articles, total, err
}
```

- [ ] **Step 2: 创建 ReviewRepository 测试**

Create `backend/services/content-service/repository/review_test.go`:

```go
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
```

- [ ] **Step 3: 运行测试并验证覆盖率**

Run: `cd backend/services/content-service && go test ./repository/... -cover`
Expected: PASS，覆盖率 >= 70%

- [ ] **Step 4: Commit**

```bash
git add backend/services/content-service/repository/review.go backend/services/content-service/repository/review_test.go
git commit -m "feat: 新增 ReviewRepository + 测试"
```

---

### Task 3: ReviewService 业务逻辑层

**Files:**
- Create: `backend/services/content-service/service/review.go`

- [ ] **Step 1: 创建 ReviewService**

```go
// backend/services/content-service/service/review.go
package service

import (
	"context"
	"errors"
	"fmt"

	"blog-community/content-service/repository"
	"blog-community/shared/events"
	"blog-community/shared/models"
)

type ReviewService struct {
	articleRepo *repository.ArticleRepository
	reviewRepo  *repository.ReviewRepository
	publisher   *events.Publisher
}

func NewReviewService(
	articleRepo *repository.ArticleRepository,
	reviewRepo *repository.ReviewRepository,
	publisher *events.Publisher,
) *ReviewService {
	return &ReviewService{
		articleRepo: articleRepo,
		reviewRepo:  reviewRepo,
		publisher:   publisher,
	}
}

// SubmitForReview 作者提交审稿：draft → pending_review
func (s *ReviewService) SubmitForReview(ctx context.Context, articleID, authorID string) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("文章不存在: %w", err)
	}
	if article.AuthorID != authorID {
		return nil, errors.New("只有作者可以提交审稿")
	}
	if article.Status != models.StatusDraft {
		return nil, errors.New("只有草稿状态的文章可以提交审稿")
	}

	article.Status = models.StatusPendingReview
	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, fmt.Errorf("提交审稿失败: %w", err)
	}

	s.publisher.Publish(events.EventArticleSubmittedForReview, map[string]interface{}{
		"article_id": articleID,
		"author_id":  authorID,
		"title":      article.Title,
	})

	return article, nil
}

// ReviewArticle 管理员审稿：pending_review → published / draft
func (s *ReviewService) ReviewArticle(ctx context.Context, articleID, reviewerID, action string, comment *string) (*models.ReviewRecord, error) {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("文章不存在: %w", err)
	}
	if article.Status != models.StatusPendingReview {
		return nil, errors.New("只能审稿待审核状态的文章")
	}

	record := &models.ReviewRecord{
		ArticleID:  articleID,
		ReviewerID: reviewerID,
		Action:     action,
		Comment:    comment,
	}

	switch action {
	case models.ReviewActionApproved:
		article.Status = models.StatusPublished
		if err := s.articleRepo.Update(ctx, article); err != nil {
			return nil, fmt.Errorf("发布文章失败: %w", err)
		}
		if err := s.reviewRepo.Create(record); err != nil {
			return nil, fmt.Errorf("创建审稿记录失败: %w", err)
		}
		s.publisher.Publish(events.EventArticlePublished, map[string]interface{}{
			"article_id": articleID,
			"user_id":    article.AuthorID,
			"title":      article.Title,
		})

	case models.ReviewActionRejected:
		article.Status = models.StatusDraft
		if err := s.articleRepo.Update(ctx, article); err != nil {
			return nil, fmt.Errorf("驳回文章失败: %w", err)
		}
		if err := s.reviewRepo.Create(record); err != nil {
			return nil, fmt.Errorf("创建审稿记录失败: %w", err)
		}
		s.publisher.Publish(events.EventArticleReviewRejected, map[string]interface{}{
			"article_id": articleID,
			"author_id":  article.AuthorID,
			"title":      article.Title,
			"comment":    comment,
		})

	default:
		return nil, errors.New("无效的审稿操作，必须是 approved 或 rejected")
	}

	return record, nil
}

// GetReviewHistory 获取文章的审稿历史
func (s *ReviewService) GetReviewHistory(ctx context.Context, articleID, userID string) ([]models.ReviewRecord, error) {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("文章不存在: %w", err)
	}
	if article.AuthorID != userID {
		return nil, errors.New("只有作者可以查看审稿历史")
	}
	return s.reviewRepo.GetByArticleID(articleID)
}

// GetPendingArticles 管理员获取待审文章列表
func (s *ReviewService) GetPendingArticles(page, size int) ([]models.Article, int64, error) {
	return s.reviewRepo.ListPendingArticles(page, size)
}
```

- [ ] **Step 2: 创建 ReviewService 测试**

Create `backend/services/content-service/service/review_test.go`:

```go
package service

import (
	"testing"

	"blog-community/content-service/repository"
	"blog-community/shared/cache"
	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReviewTestSvc(t *testing.T) (*ReviewService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.Article{}, &models.ReviewRecord{}); err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}
	g := &cache.Group{GroupMap: make(map[string]*cache.Call)}
	articleRepo := repository.NewArticleRepository(db, nil, g)
	reviewRepo := repository.NewReviewRepository(db)
	svc := NewReviewService(articleRepo, reviewRepo, nil) // publisher 为 nil
	return svc, db
}

func createDraftArticle(t *testing.T, svc *ReviewService, authorID, title string) *models.Article {
	t.Helper()
	a := &models.Article{AuthorID: authorID, Title: title, Content: "test", Status: models.StatusDraft}
	if err := svc.articleRepo.Create(a); err != nil {
		t.Fatalf("创建测试文章失败: %v", err)
	}
	return a
}

func TestSubmitForReview_Success(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试文章")

	article, err := svc.SubmitForReview(ctx, a.ID, "user-1")
	if err != nil {
		t.Fatalf("SubmitForReview() 失败: %v", err)
	}
	if article.Status != models.StatusPendingReview {
		t.Errorf("期望 status=pending_review, 实际=%s", article.Status)
	}
}

func TestSubmitForReview_NotAuthor(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试文章")

	_, err := svc.SubmitForReview(ctx, a.ID, "user-2")
	if err == nil {
		t.Fatal("期望非作者提交审稿返回错误，实际=nil")
	}
}

func TestSubmitForReview_NotDraft(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := &models.Article{AuthorID: "user-1", Title: "已发布", Content: "test", Status: models.StatusPublished}
	svc.articleRepo.Create(a)

	_, err := svc.SubmitForReview(ctx, a.ID, "user-1")
	if err == nil {
		t.Fatal("期望非草稿状态提交返回错误，实际=nil")
	}
}

func TestSubmitForReview_AlreadyPending(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试")
	svc.SubmitForReview(ctx, a.ID, "user-1") // 第一次成功

	_, err := svc.SubmitForReview(ctx, a.ID, "user-1") // 第二次应为 pending_review
	if err == nil {
		t.Fatal("期望 pending_review 状态再次提交返回错误")
	}
}

func TestReviewArticle_Approve(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试文章")
	svc.SubmitForReview(ctx, a.ID, "user-1")

	comment := "写得好"
	record, err := svc.ReviewArticle(ctx, a.ID, "admin-1", models.ReviewActionApproved, &comment)
	if err != nil {
		t.Fatalf("ReviewArticle(approved) 失败: %v", err)
	}
	if record.Action != models.ReviewActionApproved {
		t.Errorf("期望 action=approved, 实际=%s", record.Action)
	}
	if *record.Comment != "写得好" {
		t.Errorf("期望 comment=写得好, 实际=%s", *record.Comment)
	}

	// 验证文章已成为 published
	article, _ := svc.articleRepo.GetByID(ctx, a.ID)
	if article.Status != models.StatusPublished {
		t.Errorf("期望 status=published, 实际=%s", article.Status)
	}
}

func TestReviewArticle_Reject(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试文章")
	svc.SubmitForReview(ctx, a.ID, "user-1")

	record, err := svc.ReviewArticle(ctx, a.ID, "admin-1", models.ReviewActionRejected, nil)
	if err != nil {
		t.Fatalf("ReviewArticle(rejected) 失败: %v", err)
	}
	if record.Action != models.ReviewActionRejected {
		t.Errorf("期望 action=rejected, 实际=%s", record.Action)
	}

	// 验证文章已回到 draft
	article, _ := svc.articleRepo.GetByID(ctx, a.ID)
	if article.Status != models.StatusDraft {
		t.Errorf("期望 status=draft, 实际=%s", article.Status)
	}
}

func TestReviewArticle_NotPending(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试") // 未提交审稿

	_, err := svc.ReviewArticle(ctx, a.ID, "admin-1", models.ReviewActionApproved, nil)
	if err == nil {
		t.Fatal("期望未提交审稿的文章返回错误，实际=nil")
	}
}

func TestReviewArticle_InvalidAction(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试")
	svc.SubmitForReview(ctx, a.ID, "user-1")

	_, err := svc.ReviewArticle(ctx, a.ID, "admin-1", "invalid", nil)
	if err == nil {
		t.Fatal("期望无效 action 返回错误，实际=nil")
	}
}

func TestGetReviewHistory(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试")
	svc.SubmitForReview(ctx, a.ID, "user-1")
	svc.ReviewArticle(ctx, a.ID, "admin-1", models.ReviewActionRejected, nil)
	svc.SubmitForReview(ctx, a.ID, "user-1")
	svc.ReviewArticle(ctx, a.ID, "admin-1", models.ReviewActionApproved, nil)

	records, err := svc.GetReviewHistory(ctx, a.ID, "user-1")
	if err != nil {
		t.Fatalf("GetReviewHistory() 失败: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("期望 2 条审稿记录, 实际=%d", len(records))
	}
}

func TestGetReviewHistory_NotAuthor(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()
	a := createDraftArticle(t, svc, "user-1", "测试")

	_, err := svc.GetReviewHistory(ctx, a.ID, "user-2")
	if err == nil {
		t.Fatal("期望非作者查看审稿历史返回错误，实际=nil")
	}
}

func TestGetPendingArticles(t *testing.T) {
	svc, _ := newReviewTestSvc(t)
	ctx := t.Context()

	for i := 0; i < 5; i++ {
		a := createDraftArticle(t, svc, "user-1", "测试")
		svc.SubmitForReview(ctx, a.ID, "user-1")
	}

	articles, total, err := svc.GetPendingArticles(1, 3)
	if err != nil {
		t.Fatalf("GetPendingArticles() 失败: %v", err)
	}
	if total != 5 {
		t.Errorf("total: 期望=5, 实际=%d", total)
	}
	if len(articles) != 3 {
		t.Errorf("count: 期望=3, 实际=%d", len(articles))
	}
}
```

- [ ] **Step 3: 运行测试并验证覆盖率**

Run: `cd backend/services/content-service && go test ./service/... -cover`
Expected: PASS，覆盖率 >= 70%

- [ ] **Step 4: Commit**

```bash
git add backend/services/content-service/service/review.go backend/services/content-service/service/review_test.go
git commit -m "feat: 新增 ReviewService + 测试"
```

---

### Task 4: ReviewHandler HTTP 处理层

**Files:**
- Create: `backend/services/content-service/handler/review.go`

- [ ] **Step 1: 创建 ReviewHandler**

```go
// backend/services/content-service/handler/review.go
package handler

import (
	"net/http"
	"strconv"

	"blog-community/content-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	service *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: svc}
}

// SubmitForReview POST /api/articles/:id/submit-review
func (h *ReviewHandler) SubmitForReview(c *gin.Context) {
	articleID := c.Param("id")
	authorID := c.GetHeader("X-User-ID")
	if authorID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}

	article, err := h.service.SubmitForReview(c.Request.Context(), articleID, authorID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "已提交审稿", article)
}

// ReviewArticle POST /api/admin/articles/:id/review
func (h *ReviewHandler) ReviewArticle(c *gin.Context) {
	articleID := c.Param("id")
	reviewerID := c.GetHeader("X-User-ID")
	if reviewerID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}

	var req struct {
		Action  string  `json:"action" binding:"required"`
		Comment *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	record, err := h.service.ReviewArticle(c.Request.Context(), articleID, reviewerID, req.Action, req.Comment)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "审稿完成", record)
}

// GetReviewHistory GET /api/articles/:id/review-history
func (h *ReviewHandler) GetReviewHistory(c *gin.Context) {
	articleID := c.Param("id")
	authorID := c.GetHeader("X-User-ID")
	if authorID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}

	records, err := h.service.GetReviewHistory(c.Request.Context(), articleID, authorID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "获取成功", records)
}

// ListPendingReviews GET /api/admin/reviews/pending
func (h *ReviewHandler) ListPendingReviews(c *gin.Context) {
	page, size := parsePagination(c)

	articles, total, err := h.service.GetPendingArticles(page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取待审文章列表失败")
		return
	}
	utils.Paginated(c, articles, "获取成功", total, page, size)
}

// parsePagination 已在 article.go 中定义，此处复用
```

- [ ] **Step 2: 创建 ReviewHandler 测试**

Create `backend/services/content-service/handler/review_test.go`:

```go
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
```

- [ ] **Step 3: 运行测试并验证覆盖率**

Run: `cd backend/services/content-service && go test ./handler/... -cover`
Expected: PASS，覆盖率 >= 70%

- [ ] **Step 4: Commit**

```bash
git add backend/services/content-service/handler/review.go backend/services/content-service/handler/review_test.go
git commit -m "feat: 新增 ReviewHandler + 测试"
```

---

### Task 5: 注册 content-service 路由与依赖注入

**Files:**
- Modify: `backend/services/content-service/main.go`

- [ ] **Step 1: 在 main.go 中初始化审稿层并注册路由**

修改 `backend/services/content-service/main.go`：

在 `db.AutoMigrate(&models.Article{}, &models.Category{})` 后添加 `ReviewRecord` 迁移：

```go
db.AutoMigrate(&models.Article{}, &models.Category{}, &models.ReviewRecord{})
```

在初始化各层区域（`h := handler.NewArticleHandler(svc)` 之后）追加：

```go
// 审稿模块
reviewRepo := repository.NewReviewRepository(db)
reviewSvc := service.NewReviewService(repo, reviewRepo, publisher)
reviewH := handler.NewReviewHandler(reviewSvc)
```

在路由设置区域追加审稿路由（放在管理员路由块内）：

```go
// 审稿路由（需认证）
router.POST("/api/articles/:id/submit-review", reviewH.SubmitForReview)
router.GET("/api/articles/:id/review-history", reviewH.GetReviewHistory)

// 审稿管理员路由
router.GET("/api/admin/reviews/pending", reviewH.ListPendingReviews)
router.POST("/api/admin/articles/:id/review", reviewH.ReviewArticle)
```

- [ ] **Step 2: 运行全部测试确保集成正确**

Run: `cd backend/services/content-service && go test ./... -cover`
Expected: 所有包 PASS，每个包覆盖率 >= 70%

- [ ] **Step 3: Commit**

```bash
git add backend/services/content-service/main.go
git commit -m "feat: content-service 注册审稿路由与依赖注入"
```

---

### Task 6: notification-service 新增审稿事件消费者

**Files:**
- Modify: `backend/services/notification-service/repository/notification.go` — 新增 `GetAdminUserIDs`
- Modify: `backend/services/notification-service/service/notification.go` — 新增两个事件消费者

- [ ] **Step 1: Repository 新增查询管理员 ID 方法**

在 `repository/notification.go` 中追加：

```go
// GetAdminUserIDs 查询所有管理员的用户 ID
func (r *NotificationRepository) GetAdminUserIDs() ([]string, error) {
	var ids []string
	err := r.db.Model(&models.User{}).Where("role = ?", "admin").Pluck("id", &ids).Error
	return ids, err
}
```

- [ ] **Step 2: 添加审稿事件消费者**

在 `notification.go` 的 `StartListening()` 方法末尾追加两个订阅：

```go
// 监听审稿提交事件 → 通知所有管理员（每人一条）
s.consumer.Subscribe("notification_review_submitted", "article.submitted_for_review", func(event events.Event) error {
	articleID := event.Data["article_id"].(string)
	title := event.Data["title"].(string)

	adminIDs, err := s.repo.GetAdminUserIDs()
	if err != nil {
		return err
	}

	for _, adminID := range adminIDs {
		notification := &models.Notification{
			UserID:   adminID,
			Type:     "new_submission",
			Content:  fmt.Sprintf("《%s》已提交审核，请处理", title),
			SourceID: articleID,
		}
		if err := s.repo.Create(notification); err != nil {
			log.Printf("创建管理员通知失败 (admin: %s): %v", adminID, err)
		}
	}
	return nil
})

// 监听审稿驳回事件 → 通知作者
s.consumer.Subscribe("notification_review_rejected", "article.review_rejected", func(event events.Event) error {
	authorID := event.Data["author_id"].(string)
	title := event.Data["title"].(string)
	articleID := event.Data["article_id"].(string)

	content := fmt.Sprintf("你的文章《%s》已被退回", title)
	if comment, ok := event.Data["comment"].(string); ok && comment != "" {
		content += "，原因：" + comment
	}

	notification := &models.Notification{
		UserID:   authorID,
		Type:     "review_rejected",
		Content:  content,
		SourceID: articleID,
	}
	return s.repo.Create(notification)
})
```

注意：NewSubmission 事件的 `UserID` 是真实管理员 ID，不是伪造的 `"admin"`。管理员用自己已有的 `GET /api/notifications` API 即可拉到——无需新增路由。

- [ ] **Step 3: 新增 GetAdminUserIDs 测试**

在 `repository/notification_test.go` 末尾追加：

```go
func TestGetAdminUserIDs(t *testing.T) {
	db := newTestDB(t)
	// 手动创建 users 表用于测试
	db.Exec("DROP TABLE IF EXISTS users")
	db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT, role TEXT)")
	db.Exec("INSERT INTO users (id, username, role) VALUES ('admin-1', '管理员1', 'admin')")
	db.Exec("INSERT INTO users (id, username, role) VALUES ('admin-2', '管理员2', 'admin')")
	db.Exec("INSERT INTO users (id, username, role) VALUES ('user-1', '普通用户', 'user')")

	repo := NewNotificationRepository(db)
	ids, err := repo.GetAdminUserIDs()
	if err != nil {
		t.Fatalf("GetAdminUserIDs() 失败: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("期望 2 个管理员, 实际=%d", len(ids))
	}
}

func TestGetAdminUserIDs_Empty(t *testing.T) {
	db := newTestDB(t)
	db.Exec("DROP TABLE IF EXISTS users")
	db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT, role TEXT)")
	// 不插入任何 admin

	repo := NewNotificationRepository(db)
	ids, err := repo.GetAdminUserIDs()
	if err != nil {
		t.Fatalf("GetAdminUserIDs() 失败: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("期望 0 个管理员, 实际=%d", len(ids))
	}
}
```

- [ ] **Step 4: 运行测试并验证覆盖率**

Run: `cd backend/services/notification-service && go test ./... -cover`
Expected: 全部 PASS，每个包覆盖率 >= 70%

- [ ] **Step 5: Commit**

```bash
git add backend/services/notification-service/
git commit -m "feat: notification-service 新增审稿事件消费者 + 测试"
```
```

---

### Task 7: API Gateway 注册审稿路由

**Files:**
- Modify: `backend/api-gateway/routes/router.go`

- [ ] **Step 1: 添加审稿路由**

在 `setupPrivateRoutes` 函数中添加：

```go
// 审稿相关
router.POST("/api/articles/:id/submit-review", proxyTo("article"))
router.GET("/api/articles/:id/review-history", proxyTo("article"))
```

在 `setupAdminRoutes` 函数中添加：

```go
// 审稿管理
router.GET("/reviews/pending", proxyTo("article"))
router.POST("/articles/:id/review", proxyTo("article"))
```

注意：`setupAdminRoutes` 的 group 前缀是 `/api/admin`，所以完整路径为 `/api/admin/reviews/pending` 和 `/api/admin/articles/:id/review`。

- [ ] **Step 2: 验证无破坏**

Run: `cd backend/api-gateway && go build ./... && go test ./...`
Expected: 编译通过，测试 PASS

- [ ] **Step 3: Commit**

```bash
git add backend/api-gateway/routes/router.go
git commit -m "feat: api-gateway 注册审稿路由"
```

---

### Task 8: 前端 API 层 + 路由

**Files:**
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: 添加审稿 API 方法**

在 `frontend/src/api/index.ts` 的 `articleApi` 对象中追加：

```typescript
  submitReview(id: string) {
    return api.post(`/articles/${id}/submit-review`)
  },
  getReviewHistory(id: string) {
    return api.get(`/articles/${id}/review-history`)
  },
```

在 `adminApi` 对象中追加：

```typescript
  getPendingReviews(page = 1, size = 20) {
    return api.get('/admin/reviews/pending', { params: { page, size } })
  },
  reviewArticle(id: string, data: { action: string; comment?: string }) {
    return api.post(`/admin/articles/${id}/review`, data)
  },
```

- [ ] **Step 2: 添加管理员审稿路由**

在 `frontend/src/router/index.ts` 的 `routes` 数组中，admin 路由后追加：

```typescript
    {
      path: '/admin/reviews',
      name: 'adminReviews',
      component: () => import('@/views/AdminReviewView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
```

- [ ] **Step 3: 验证编译**

Run: `cd frontend && npm run type-check`
Expected: TypeScript 类型检查通过

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/index.ts frontend/src/router/index.ts
git commit -m "feat: 前端新增审稿 API 方法和路由"
```

---

### Task 9: 前端 EditorView 审稿功能

**Files:**
- Modify: `frontend/src/views/EditorView.vue`

- [ ] **Step 1: 修改 EditorView，添加审稿状态展示和提交按钮**

修改 `<template>`：

1. 将表单提交按钮的文字改为动态：

```html
<button type="submit" :disabled="submitting">
  {{ submitting ? '保存中...' : submitButtonText }}
</button>
```

2. 在表单下方（`<form>` 结束后）添加审稿状态区域：

```html
    <!-- 审稿状态（仅编辑已有文章时显示） -->
    <div v-if="isEdit && reviewStatus" class="review-section">
      <div class="review-status" :class="'status-' + reviewStatus">
        <span v-if="reviewStatus === 'pending_review'">审核中，暂不可编辑</span>
        <span v-else-if="reviewStatus === 'published'">已通过审核并发布</span>
        <span v-else-if="reviewStatus === 'draft' && reviewHistory.length > 0">已被退回，可修改后重新提交</span>
      </div>

      <!-- 重新审核按钮（驳回后重投） -->
      <button
        v-if="reviewStatus === 'draft' && reviewHistory.length > 0"
        type="button"
        class="btn-submit-review"
        :disabled="submittingReview"
        @click="handleResubmitReview"
      >
        {{ submittingReview ? '提交中...' : '重新审核' }}
      </button>

      <!-- 审稿历史 -->
      <div v-if="reviewHistory.length > 0" class="review-history">
        <h3>审稿记录</h3>
        <div v-for="record in reviewHistory" :key="record.id" class="review-record">
          <span :class="record.action === 'approved' ? 'tag-approved' : 'tag-rejected'">
            {{ record.action === 'approved' ? '通过' : '驳回' }}
          </span>
          <span class="review-time">{{ formatDate(record.created_at) }}</span>
          <p v-if="record.comment" class="review-comment">{{ record.comment }}</p>
        </div>
      </div>
    </div>
```

修改 `<script setup>`，新增审稿相关状态和方法：

```typescript
const reviewStatus = ref('')
const reviewHistory = ref<any[]>([])
const submittingReview = ref(false)

// 提交按钮文字：新建文章 = "发布文章"，编辑已有 = "保存草稿"
const submitButtonText = computed(() => isEdit.value ? '保存草稿' : '发布文章')

// 加载审稿信息
async function fetchReviewInfo() {
  try {
    const res: any = await articleApi.getReviewHistory(editId)
    reviewHistory.value = res.data || []
  } catch { /* 忽略 */ }
}

// 重新审核（驳回后重投，仅调用审稿 API）
async function handleResubmitReview() {
  if (!confirm('确认重新提交审核？提交后将无法编辑。')) return
  submittingReview.value = true
  try {
    await articleApi.submitReview(editId)
    reviewStatus.value = 'pending_review'
  } catch (e: any) {
    error.value = e?.message || '重新审核失败'
  } finally {
    submittingReview.value = false
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}
```

修改 `onMounted`，在加载文章数据后读取审稿状态：

```typescript
onMounted(async () => {
  if (editId) {
    isEdit.value = true
    try {
      const res: any = await articleApi.getDetail(editId)
      title.value = res.data.title
      content.value = res.data.content
      reviewStatus.value = res.data.status
      await fetchReviewInfo()
    } catch {
      error.value = '加载文章失败'
    }
  }
})
```

修改 `handleSubmit`。新建文章时：创建（draft）→ 立即提交审稿 → 跳转到编辑页展示"审核中"；编辑已有文章时：仅保存草稿：

```typescript
async function handleSubmit() {
  submitting.value = true
  error.value = ''
  try {
    if (isEdit.value) {
      // 编辑已有文章 → 保存草稿
      await articleApi.update(editId, {
        title: title.value,
        content: content.value,
      })
      router.push('/')
    } else {
      // 新建文章 → 创建草稿 → 立即提交审稿 → 跳转到编辑页
      const res: any = await articleApi.create({
        title: title.value,
        content: content.value,
        category_id: categoryId.value,
      })
      const newId = res.data.id
      await articleApi.submitReview(newId)
      router.push(`/editor/${newId}`)
    }
  } catch (e: any) {
    error.value = e.message || '保存失败'
  } finally {
    submitting.value = false
  }
}
```

关键流程：
- **新建文章**：按钮"发布文章" → `create`（status=draft）→ `submitReview`（draft→pending_review）→ 跳转 `/editor/:id`（页面显示"审核中"）
- **驳回后重投**：按钮"重新审核" → `submitReview`（draft→pending_review）→ 页面变"审核中"
- **仅保存草稿**：按钮"保存草稿" → `update` → 回到首页

- [ ] **Step 2: 添加审稿区域样式**

在 `<style scoped>` 末尾追加：

```css
.review-section {
  margin-top: 24px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
}

.review-status {
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 14px;
  margin-bottom: 16px;
}

.status-pending_review {
  background: #fff3cd;
  color: #856404;
}

.status-published {
  background: #d4edda;
  color: #155724;
}

.status-draft {
  background: #f8d7da;
  color: #721c24;
}

.btn-submit-review {
  padding: 10px 28px;
  background: #27ae60;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
  margin-bottom: 16px;
}

.btn-submit-review:hover {
  background: #219a52;
}

.btn-submit-review:disabled {
  opacity: 0.6;
}

.review-history {
  margin-top: 20px;
}

.review-history h3 {
  font-size: 16px;
  margin-bottom: 12px;
  color: #333;
}

.review-record {
  padding: 12px;
  border-bottom: 1px solid #eee;
}

.tag-approved {
  color: #27ae60;
  font-weight: 600;
}

.tag-rejected {
  color: #e74c3c;
  font-weight: 600;
}

.review-time {
  margin-left: 12px;
  font-size: 13px;
  color: #999;
}

.review-comment {
  margin-top: 6px;
  font-size: 14px;
  color: #555;
  line-height: 1.5;
}
```

- [ ] **Step 3: 验证编译**

Run: `cd frontend && npm run type-check`
Expected: 通过

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/EditorView.vue
git commit -m "feat: EditorView 新增审稿状态、历史展示和提交按钮"
```

---

### Task 10: AdminReviewView 管理员审稿页面

**Files:**
- Create: `frontend/src/views/AdminReviewView.vue`

- [ ] **Step 1: 创建管理员审稿页面**

```vue
<!-- frontend/src/views/AdminReviewView.vue -->
<template>
  <div class="admin-page">
    <h1>审稿管理</h1>

    <div class="tabs">
      <button :class="{ active: activeTab === 'pending' }" @click="activeTab = 'pending'">待审文章</button>
      <router-link to="/admin" class="nav-back">← 返回管理面板</router-link>
    </div>

    <!-- 待审文章列表 -->
    <div class="tab-content">
      <h2>待审文章列表</h2>
      <table class="data-table">
        <thead>
          <tr>
            <th>标题</th>
            <th>作者</th>
            <th>提交时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="article in articles" :key="article.id">
            <td class="col-title">{{ article.title }}</td>
            <td>{{ article.author_id }}</td>
            <td>{{ formatDate(article.updated_at) }}</td>
            <td>
              <button class="btn-review" @click="openReview(article)">审稿</button>
            </td>
          </tr>
          <tr v-if="articles.length === 0">
            <td colspan="4" class="placeholder">暂无待审文章</td>
          </tr>
        </tbody>
      </table>
      <div class="pagination" v-if="total > size">
        <button :disabled="page <= 1" @click="changePage(page - 1)">上一页</button>
        <span>{{ page }} / {{ Math.ceil(total / size) }}</span>
        <button :disabled="page >= Math.ceil(total / size)" @click="changePage(page + 1)">下一页</button>
      </div>
    </div>

    <!-- 审稿对话框 -->
    <div v-if="reviewing" class="modal-overlay" @click.self="closeReview">
      <div class="modal">
        <h2>审稿：{{ reviewingArticle?.title }}</h2>
        <div class="article-preview">
          <pre class="raw-markdown">{{ reviewingArticle?.content }}</pre>
        </div>
        <div class="review-actions">
          <label>
            <input type="radio" v-model="action" value="approved" /> 通过
          </label>
          <label>
            <input type="radio" v-model="action" value="rejected" /> 驳回
          </label>
        </div>
        <div class="review-comment-input">
          <label>意见（可选）</label>
          <textarea v-model="comment" rows="3" placeholder="审稿意见..."></textarea>
        </div>
        <div class="modal-buttons">
          <button class="btn-cancel" @click="closeReview">取消</button>
          <button class="btn-confirm" :disabled="submitting" @click="handleReview">
            {{ submitting ? '提交中...' : '提交审稿结果' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { adminApi } from '@/api'

const activeTab = ref('pending')
const articles = ref<any[]>([])
const page = ref(1)
const size = 20
const total = ref(0)

// 审稿对话框
const reviewing = ref(false)
const reviewingArticle = ref<any>(null)
const action = ref('approved')
const comment = ref('')
const submitting = ref(false)

async function fetchPending() {
  try {
    const res: any = await adminApi.getPendingReviews(page.value, size)
    articles.value = res.data || []
    total.value = res.pagination?.total || 0
  } catch {
    alert('获取待审文章失败')
  }
}

function changePage(p: number) {
  page.value = p
  fetchPending()
}

function openReview(article: any) {
  reviewingArticle.value = article
  action.value = 'approved'
  comment.value = ''
  reviewing.value = true
}

function closeReview() {
  reviewing.value = false
  reviewingArticle.value = null
}

async function handleReview() {
  submitting.value = true
  try {
    await adminApi.reviewArticle(reviewingArticle.value.id, {
      action: action.value,
      comment: comment.value || undefined,
    })
    closeReview()
    fetchPending()
  } catch (e: any) {
    alert(e?.message || '审稿失败')
  } finally {
    submitting.value = false
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  fetchPending()
})
</script>

<style scoped>
.admin-page {
  max-width: 1100px;
  margin: 0 auto;
  padding: 24px;
}

h1 { font-size: 24px; margin-bottom: 20px; }

.tabs {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 2px solid #e8e8e8;
  margin-bottom: 24px;
}

.tabs button {
  padding: 10px 24px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: #666;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
}

.tabs button.active {
  color: #3498db;
  border-bottom-color: #3498db;
}

.nav-back {
  font-size: 14px;
  color: #3498db;
  text-decoration: none;
}

.tab-content { min-height: 300px; }

h2 { font-size: 18px; margin-bottom: 16px; }

.data-table { width: 100%; border-collapse: collapse; }

.data-table th, .data-table td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid #eee;
  font-size: 14px;
}

.data-table th { background: #f9f9f9; font-weight: 600; color: #333; }

.col-title { max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.btn-review {
  padding: 4px 14px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.btn-review:hover { background: #2980b9; }

.placeholder { color: #999; font-size: 15px; padding: 40px 0; text-align: center; }

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  margin-top: 20px;
}

.pagination button {
  padding: 6px 14px;
  border: 1px solid #ddd;
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
}

.pagination button:disabled { opacity: 0.4; cursor: not-allowed; }

.pagination span { font-size: 14px; color: #666; }

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 200;
}

.modal {
  background: #fff;
  border-radius: 12px;
  padding: 32px;
  width: 720px;
  max-height: 85vh;
  overflow-y: auto;
}

.modal h2 { font-size: 20px; margin-bottom: 16px; }

.article-preview {
  max-height: 400px;
  overflow-y: auto;
  background: #f5f5f5;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 20px;
}

.raw-markdown {
  font-family: 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}

.review-actions {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
}

.review-actions label {
  font-size: 15px;
  cursor: pointer;
}

.review-actions input { margin-right: 6px; }

.review-comment-input { margin-bottom: 20px; }

.review-comment-input label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 6px;
}

.review-comment-input textarea {
  width: 100%;
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
}

.modal-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn-cancel {
  padding: 8px 20px;
  background: #fff;
  border: 1px solid #ddd;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-confirm {
  padding: 8px 20px;
  background: #3498db;
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-confirm:hover { background: #2980b9; }
.btn-confirm:disabled { opacity: 0.6; }
</style>
```

- [ ] **Step 2: 验证编译**

Run: `cd frontend && npm run type-check`
Expected: 通过

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/AdminReviewView.vue
git commit -m "feat: 新增管理员审稿管理页面"
```

---

### Task 11: Navbar 添加管理员审稿入口

**Files:**
- Modify: `frontend/src/components/Navbar.vue`

- [ ] **Step 1: 添加审稿管理导航链接**

在 `Navbar.vue` 的管理链接后添加：

```html
<router-link v-if="userStore.isAdmin" to="/admin/reviews" class="nav-link">审稿</router-link>
```

放在 `<router-link v-if="userStore.isAdmin" to="/admin" class="nav-link">管理</router-link>` 之后。

- [ ] **Step 2: 验证编译**

Run: `cd frontend && npm run type-check`
Expected: 通过

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Navbar.vue
git commit -m "feat: Navbar 添加管理员审稿入口"
```

---

### Task 12: 集成验证与全文测试

- [ ] **Step 1: 启动所有服务**

```bash
docker compose up -d
```

Wait for all services healthy.

- [ ] **Step 2: 端到端验证流程**

1. 作者登录 → 创建文章 → 编辑器中点击"提交审稿"
2. 管理员登录 → 导航栏出现"审稿"链接 → 看到待审文章列表
3. 管理员点击"审稿" → 弹出对话框 → 选择"通过" → 提交
4. 作者收到通知"文章已通过审核" → 文章出现在首页列表
5. 作者创建另一篇文章 → 提交审稿 → 管理员"驳回" + 填写意见
6. 作者收到通知"文章已被退回" → 编辑器中看到驳回记录 → 修改后重新提交审稿

- [ ] **Step 3: 验证通知系统**

确认：
- 管理员收到"新待审文章"通知
- 作者收到"通过/驳回"通知
- 通知列表正确显示

- [ ] **Step 4: 检查状态机边界**

手动测试（通过 API 调用）：
- `draft` 文章才能 `submit-review` → 其他状态应返回错误
- `pending_review` 文章才能 `review` → 其他状态应返回错误
- 非作者不能提交审稿 → 应返回权限错误
- 非管理员不能审稿 → 网关层拦截

- [ ] **Step 5: Commit 最终确认**

```bash
git add -A
git commit -m "审核系统: 审稿系统端到端验证通过"
```
