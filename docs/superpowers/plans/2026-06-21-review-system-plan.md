# 审稿系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在博客社区平台上实现编辑审核制审稿系统——作者提交文章审稿，管理员审批，全链路通知。

**Architecture:** 扩展 content-service（不新建微服务），新增 ReviewRecord 模型、ReviewRepository/ReviewService/ReviewHandler 三层。审稿通过复用已有 `article.published` 事件，驳回触发新事件 `article.review_rejected`。

**Tech Stack:** Go + Gin + GORM (后端), Vue 3 + TypeScript + Pinia (前端), RabbitMQ (事件通知)

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

- [ ] **Step 4: 验证编译**

Run: `cd backend/content-service && go build ./...`
Expected: 编译通过（新增模型未引用，无破坏）

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

- [ ] **Step 2: 验证编译**

Run: `cd backend/services/content-service && go build ./...`
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add backend/services/content-service/repository/review.go
git commit -m "feat: 新增 ReviewRepository 数据访问层"
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
func (s *ReviewService) SubmitForReview(articleID, authorID string) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(context.Background(), articleID)
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
	if err := s.articleRepo.Update(context.Background(), article); err != nil {
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
func (s *ReviewService) ReviewArticle(articleID, reviewerID, action string, comment *string) (*models.ReviewRecord, error) {
	article, err := s.articleRepo.GetByID(context.Background(), articleID)
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
		now := article.UpdatedAt // 使用 GORM 自动时间
		_ = now
		if err := s.articleRepo.Update(context.Background(), article); err != nil {
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
		if err := s.articleRepo.Update(context.Background(), article); err != nil {
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
func (s *ReviewService) GetReviewHistory(articleID, userID string) ([]models.ReviewRecord, error) {
	article, err := s.articleRepo.GetByID(context.Background(), articleID)
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

- [ ] **Step 2: 验证编译**

Run: `cd backend/services/content-service && go build ./...`
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add backend/services/content-service/service/review.go
git commit -m "feat: 新增 ReviewService 业务逻辑层"
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

	article, err := h.service.SubmitForReview(articleID, authorID)
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

	record, err := h.service.ReviewArticle(articleID, reviewerID, req.Action, req.Comment)
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

	records, err := h.service.GetReviewHistory(articleID, authorID)
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

- [ ] **Step 2: 验证编译**

Run: `cd backend/services/content-service && go build ./...`
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add backend/services/content-service/handler/review.go
git commit -m "feat: 新增 ReviewHandler HTTP 处理层"
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

- [ ] **Step 2: 验证编译**

Run: `cd backend/services/content-service && go build ./...`
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add backend/services/content-service/main.go
git commit -m "feat: content-service 注册审稿路由与依赖注入"
```

---

### Task 6: notification-service 新增审稿事件消费者

**Files:**
- Modify: `backend/services/notification-service/service/notification.go`
- Modify: `backend/services/notification-service/main.go`

- [ ] **Step 1: 添加审稿事件消费者**

在 `notification.go` 的 `StartListening()` 方法末尾（`s.consumer.Subscribe("notification_like"...` 之后，`}` 之前）追加两个新的订阅：

```go
// 监听审稿提交事件 → 通知所有管理员
s.consumer.Subscribe("notification_review_submitted", "article.submitted_for_review", func(event events.Event) error {
	articleID := event.Data["article_id"].(string)
	title := event.Data["title"].(string)

	// 创建通知给所有管理员（user_id 用特殊标记 "admin"，
	// 前端管理员拉取时查询 user_id = "admin" 的通知）
	notification := &models.Notification{
		UserID:   "admin",
		Type:     "new_submission",
		Content:  fmt.Sprintf("《%s》已提交审核，请处理", title),
		SourceID: articleID,
	}
	return s.repo.Create(notification)
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

- [ ] **Step 2: 修改 main.go 拉取管理员通知**

在 `notification-service/main.go` 中，现有的 `router.GET("/api/notifications", h.GetNotifications)` 已经支持按 userID 查询。前端管理员调用时传 `X-User-ID` header，后端通过 `c.GetHeader("X-User-ID")` 获取。但管理员也需要看到 `user_id = "admin"` 的公共通知。需修改 `GetNotifications` 的查询逻辑：

在 `repository/notification.go` 中，检查现有 `GetByUserID` 方法。如果只查询单用户，需要改为：当 userID 是管理员时，同时查询 `user_id = 'admin'` 或 `user_id = <当前管理员ID>` 的通知。

简化处理：新增 `GetAdminNotifications` 方法（或者修改 handler 层，管理员调用时单独处理）。这里采用简单方案——修改 `GetByUserID` 查询，增加 OR 条件。

在 `repository/notification.go` 的 `GetByUserID` 方法中（需查看现有实现），将查询条件改为：

```go
func (r *NotificationRepository) GetByUserID(userID string, page, size int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	query := r.db.Model(&models.Notification{}).Where("user_id = ?", userID)
	query.Count(&total)

	err := query.
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&notifications).Error

	return notifications, total, err
}
```

注意：管理员通知目前存 `user_id = "admin"`，所以管理员调用时传自己的 userID 只能拿到个人通知。前台在 Navbar 里获取 unread count 和通知列表时需要额外处理。简化处理：在 service 层加一个方法合并管理员的两类通知。在 `notification.go` service 中新增：

```go
// GetAdminNotifications 管理员获取通知（个人 + 公共管理通知）
func (s *NotificationService) GetAdminNotifications(adminID string, page, size int) ([]models.Notification, int64, error) {
	return s.repo.GetAdminNotifications(adminID, page, size)
}
```

在 repository 中新增：

```go
// GetAdminNotifications 管理员通知（个人 + 公共）
func (r *NotificationRepository) GetAdminNotifications(adminID string, page, size int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	query := r.db.Model(&models.Notification{}).
		Where("user_id = ? OR user_id = ?", adminID, "admin")
	query.Count(&total)

	err := query.
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&notifications).Error

	return notifications, total, err
}
```

- [ ] **Step 3: 注册管理员通知路由**

在 `main.go` 中添加管理员通知路由：

```go
router.GET("/api/admin/notifications", h.GetAdminNotifications)
```

在 handler 中新增：

```go
// GetAdminNotifications GET /api/admin/notifications
func (h *NotificationHandler) GetAdminNotifications(c *gin.Context) {
	adminID := c.GetHeader("X-User-ID")
	if adminID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}
	page, size := parsePagination(c)
	notifications, total, err := h.service.GetAdminNotifications(adminID, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取通知失败")
		return
	}
	utils.Paginated(c, notifications, "获取成功", total, page, size)
}
```

- [ ] **Step 4: 验证编译**

Run: `cd backend/services/notification-service && go build ./...`
Expected: 编译通过

- [ ] **Step 5: Commit**

```bash
git add backend/services/notification-service/
git commit -m "feat: notification-service 新增审稿事件消费者和管理员通知查询"
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

- [ ] **Step 2: 验证编译**

Run: `cd backend/api-gateway && go build ./...`
Expected: 编译通过

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

修改 `<template>`，在表单下方（`<form>` 结束后）添加：

```html
    <!-- 审稿状态（仅编辑已有文章时显示） -->
    <div v-if="isEdit && reviewStatus" class="review-section">
      <div class="review-status" :class="'status-' + reviewStatus">
        <span v-if="reviewStatus === 'pending_review'">⏳ 审核中，暂不可编辑</span>
        <span v-else-if="reviewStatus === 'published'">✓ 已通过审核并发布</span>
        <span v-else-if="reviewStatus === 'draft' && reviewHistory.length > 0">✗ 已被退回，可修改后重新提交</span>
      </div>

      <!-- 提交审稿按钮 -->
      <button
        v-if="reviewStatus === 'draft'"
        type="button"
        class="btn-submit-review"
        :disabled="submittingReview"
        @click="handleSubmitReview"
      >
        {{ submittingReview ? '提交中...' : '提交审稿' }}
      </button>

      <!-- 审稿历史 -->
      <div v-if="reviewHistory.length > 0" class="review-history">
        <h3>审稿记录</h3>
        <div v-for="record in reviewHistory" :key="record.id" class="review-record">
          <span :class="record.action === 'approved' ? 'tag-approved' : 'tag-rejected'">
            {{ record.action === 'approved' ? '✓ 通过' : '✗ 驳回' }}
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

// 加载审稿信息（在 onMounted 中已有 fetchArticle 后追加）
async function fetchReviewInfo() {
  try {
    const res: any = await articleApi.getReviewHistory(editId)
    reviewHistory.value = res.data || []
  } catch { /* 忽略 */ }
}

// 提交审稿
async function handleSubmitReview() {
  if (!confirm('确认提交审稿？提交后将无法编辑。')) return
  submittingReview.value = true
  try {
    await articleApi.submitReview(editId)
    reviewStatus.value = 'pending_review'
  } catch (e: any) {
    error.value = e?.message || '提交审稿失败'
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

修改 `handleSubmit`，创建新文章时默认不再自动发布（移除 `await articleApi.publish(res.data.id)`）：

```typescript
async function handleSubmit() {
  submitting.value = true
  error.value = ''
  try {
    if (isEdit.value) {
      await articleApi.update(editId, {
        title: title.value,
        content: content.value,
      })
    } else {
      await articleApi.create({
        title: title.value,
        content: content.value,
        category_id: categoryId.value,
      })
    }
    router.push('/')
  } catch (e: any) {
    error.value = e.message || '保存失败'
  } finally {
    submitting.value = false
  }
}
```

注意：新文章创建后应跳转到编辑页让用户提交审稿，或者直接留在编辑器。这里保持跳转到首页，用户可从文章列表或通知中找到自己的文章继续操作。

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
git commit -m "test: 审稿系统端到端验证通过"
```
