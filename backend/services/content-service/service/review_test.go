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
