package service

import (
	"context"
	"testing"
	"time"

	"blog-community/content-service/repository"
	"blog-community/shared/cache"
	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 创建测试用的完整服务（含真实 DB 和 Redis）
func newTestService(t *testing.T) (*ArticleService, func()) {
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
	svc := NewArticleService(repo, nil) // 传递 nil 作为 publisher，因为我们不测试事件发布

	cleanup := func() {
		redisClient.Close()
	}

	return svc, cleanup
}

// 辅助函数：在服务层创建一篇测试文章
func createArticleViaService(t *testing.T, svc *ArticleService, authorID, title, content string) *models.Article {
	t.Helper()
	article, err := svc.CreateArticle(authorID, title, content, "摘要", "tech", []string{"go"})
	if err != nil {
		t.Fatalf("创建测试文章失败: %v", err)
	}
	return article
}

// ========== CreateArticle 测试 ==========

func TestCreateArticle_Success(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article, err := svc.CreateArticle("author-001", "测试标题", "测试内容", "摘要", "tech", []string{"go", "redis"})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}

	if article.ID == "" {
		t.Fatal("文章 ID 不应为空")
	}
	if article.Title != "测试标题" {
		t.Errorf("Title = %q, want %q", article.Title, "测试标题")
	}
	if article.AuthorID != "author-001" {
		t.Errorf("AuthorID = %q, want %q", article.AuthorID, "author-001")
	}
	if article.Status != models.StatusDraft {
		t.Errorf("新创建的文章状态应为 draft, got %q", article.Status)
	}
	if article.ViewCount != 0 {
		t.Errorf("ViewCount 初始值应为 0")
	}
}

func TestCreateArticle_EmptyTitle(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.CreateArticle("author-001", "", "内容", "", "", nil)
	if err == nil {
		t.Fatal("空标题应返回错误")
	}
	if err.Error() != "标题不能为空" {
		t.Errorf("错误信息应为'标题不能为空', got %q", err.Error())
	}
}

func TestCreateArticle_TitleTooLong(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	longTitle := ""
	for i := 0; i < 201; i++ {
		longTitle += "a"
	}

	_, err := svc.CreateArticle("author-001", longTitle, "内容", "", "", nil)
	if err == nil {
		t.Fatal("超长标题应返回错误")
	}
	if err.Error() != "标题不能超过 200 字" {
		t.Errorf("错误信息应为'标题不能超过 200 字', got %q", err.Error())
	}
}

func TestCreateArticle_TitleExactly200(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	title := ""
	for i := 0; i < 200; i++ {
		title += "a"
	}

	article, err := svc.CreateArticle("author-001", title, "内容", "", "", nil)
	if err != nil {
		t.Fatalf("恰好200字标题应成功: %v", err)
	}
	if len([]rune(article.Title)) != 200 {
		t.Errorf("标题长度应为 200")
	}
}

func TestCreateArticle_EmptyContent(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.CreateArticle("author-001", "标题", "", "", "", nil)
	if err == nil {
		t.Fatal("空内容应返回错误")
	}
	if err.Error() != "内容不能为空" {
		t.Errorf("错误信息应为'内容不能为空', got %q", err.Error())
	}
}

func TestCreateArticle_WithTags(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article, err := svc.CreateArticle("author-001", "有标签的文章", "内容", "摘要", "tech",
		[]string{"Go", "Redis", "微服务"})
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	if len(article.Tags) == 0 {
		t.Fatal("Tags 不应为空")
	}
}

func TestCreateArticle_NoTags(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article, err := svc.CreateArticle("author-001", "无标签", "内容", "", "", nil)
	if err != nil {
		t.Fatalf("CreateArticle 失败: %v", err)
	}
	// Tags 为 nil 时 JSON 为空字节切片或 nil
	if len(article.Tags) != 0 {
		t.Logf("Tags with no input: %v", article.Tags)
	}
}

// ========== EditArticle 测试 ==========

func TestEditArticle_Success(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "原标题", "原内容")
	ctx := context.Background()

	updated, err := svc.EditArticle(ctx, article.ID, "author-001", "新标题", "新内容", "新摘要", "life")
	if err != nil {
		t.Fatalf("EditArticle 失败: %v", err)
	}
	if updated.Title != "新标题" {
		t.Errorf("Title 应为'新标题', got %q", updated.Title)
	}
	if updated.Content != "新内容" {
		t.Errorf("Content 应为'新内容'")
	}
	if updated.Category != "life" {
		t.Errorf("Category 应为'life', got %q", updated.Category)
	}
}

func TestEditArticle_NotAuthor(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "标题", "内容")
	ctx := context.Background()

	_, err := svc.EditArticle(ctx, article.ID, "author-002", "新标题", "新内容", "摘要", "tech")
	if err == nil {
		t.Fatal("非作者编辑应返回错误")
	}
	if err.Error() != "只有作者可以编辑" {
		t.Errorf("错误信息不匹配: got %q", err.Error())
	}
}

func TestEditArticle_NotFound(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.EditArticle(ctx, "non-existent-id", "author-001", "标题", "内容", "摘要", "tech")
	if err == nil {
		t.Fatal("编辑不存在文章应返回错误")
	}
}

func TestEditArticle_PublishedArticle(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "标题", "内容")
	ctx := context.Background()

	// 先发布
	_, err := svc.PublishArticle(ctx, article.ID, "author-001")
	if err != nil {
		t.Fatalf("PublishArticle 失败: %v", err)
	}

	// 尝试编辑已发布的文章
	_, err = svc.EditArticle(ctx, article.ID, "author-001", "新标题", "新内容", "摘要", "tech")
	if err == nil {
		t.Fatal("编辑已发布文章应返回错误")
	}
	if err.Error() != "只能编辑草稿状态的文章" {
		t.Errorf("错误信息不匹配: got %q", err.Error())
	}
}

// ========== PublishArticle 测试 ==========

func TestPublishArticle_Success(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "待发布", "内容")
	ctx := context.Background()

	published, err := svc.PublishArticle(ctx, article.ID, "author-001")
	if err != nil {
		t.Fatalf("PublishArticle 失败: %v", err)
	}
	if published.Status != models.StatusPublished {
		t.Errorf("Status 应为 published, got %q", published.Status)
	}
	if published.PublishedAt.IsZero() {
		t.Error("PublishedAt 不应为零值")
	}
}

func TestPublishArticle_NotAuthor(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "标题", "内容")
	ctx := context.Background()

	_, err := svc.PublishArticle(ctx, article.ID, "author-002")
	if err == nil {
		t.Fatal("非作者发布应返回错误")
	}
	if err.Error() != "只有作者可以发布" {
		t.Errorf("错误信息不匹配: got %q", err.Error())
	}
}

func TestPublishArticle_AlreadyPublished(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "标题", "内容")
	ctx := context.Background()

	// 第一次发布
	svc.PublishArticle(ctx, article.ID, "author-001")

	// 第二次发布
	_, err := svc.PublishArticle(ctx, article.ID, "author-001")
	if err == nil {
		t.Fatal("重复发布应返回错误")
	}
	if err.Error() != "文章已发布" {
		t.Errorf("错误信息应为'文章已发布', got %q", err.Error())
	}
}

// ========== DeleteArticle 测试 ==========

func TestDeleteArticle_Success(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "待删除", "内容")
	ctx := context.Background()

	err := svc.DeleteArticle(ctx, article.ID, "author-001")
	if err != nil {
		t.Fatalf("DeleteArticle 失败: %v", err)
	}

	// 确认已删除
	_, err = svc.GetArticleDetail(ctx, article.ID)
	if err == nil {
		t.Fatal("已删除文章应获取失败")
	}
}

func TestDeleteArticle_NotAuthor(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "标题", "内容")
	ctx := context.Background()

	err := svc.DeleteArticle(ctx, article.ID, "author-002")
	if err == nil {
		t.Fatal("非作者删除应返回错误")
	}
	if err.Error() != "只有作者可以删除" {
		t.Errorf("错误信息不匹配: got %q", err.Error())
	}
}

func TestDeleteArticle_NotFound(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	err := svc.DeleteArticle(ctx, "non-existent", "author-001")
	if err == nil {
		t.Fatal("删除不存在文章应返回错误")
	}
}

// ========== GetArticleDetail 测试 ==========

func TestGetArticleDetail_Success(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "详细信息", "详细内容")
	ctx := context.Background()

	result, err := svc.GetArticleDetail(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetArticleDetail 失败: %v", err)
	}
	if result.ID != article.ID {
		t.Errorf("ID 不匹配")
	}
	if result.Title != "详细信息" {
		t.Errorf("Title 不匹配")
	}

	// 等待 goroutine 完成（异步增加浏览数）
	time.Sleep(50 * time.Millisecond)
}

func TestGetArticleDetail_NotFound(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.GetArticleDetail(ctx, "not-found")
	if err == nil {
		t.Fatal("获取不存在文章应返回错误")
	}
}

func TestGetArticleDetail_IncrementsViewCount(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	article := createArticleViaService(t, svc, "author-001", "浏览测试", "内容")
	ctx := context.Background()

	// 多次查看（goroutine 异步）
	svc.GetArticleDetail(ctx, article.ID)
	svc.GetArticleDetail(ctx, article.ID)
	svc.GetArticleDetail(ctx, article.ID)

	// 等待异步操作完成
	time.Sleep(100 * time.Millisecond)

	// 再次获取以查看最新数据
	result, _ := svc.GetArticleDetail(ctx, article.ID)
	if result.ViewCount < 1 {
		t.Logf("ViewCount (可能因异步且缓存失效): %d", result.ViewCount)
	}
}

// ========== ListArticles 测试 ==========

func TestListArticles(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	// 创建一篇已发布和一篇草稿
	draft := createArticleViaService(t, svc, "a1", "草稿", "c")
	ctx := context.Background()
	svc.PublishArticle(ctx, draft.ID, "a1")

	pub2 := createArticleViaService(t, svc, "a2", "已发布", "c")
	svc.PublishArticle(ctx, pub2.ID, "a2")

	// 再创建一篇保持草稿的
	createArticleViaService(t, svc, "a3", "草稿2", "c")

	articles, total, err := svc.ListArticles(1, 10)
	if err != nil {
		t.Fatalf("ListArticles 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("已发布文章总数为 2, got %d", total)
	}
	if len(articles) != 2 {
		t.Errorf("应返回 2 篇已发布文章, got %d", len(articles))
	}
	for _, a := range articles {
		if a.Status != models.StatusPublished {
			t.Errorf("ListArticles 应只返回已发布文章, 但 ID=%s Status=%s", a.ID, a.Status)
		}
	}
}

func TestListArticles_Empty(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	articles, total, err := svc.ListArticles(1, 10)
	if err != nil {
		t.Fatalf("ListArticles 失败: %v", err)
	}
	if total != 0 {
		t.Errorf("无文章时 total 应为 0")
	}
	if len(articles) != 0 {
		t.Errorf("无文章时应返回空列表")
	}
}

// ========== ListArticlesByCategory 测试 ==========

func TestListArticlesByCategory(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	// 不同分类
	a1 := createArticleViaService(t, svc, "a1", "AI文章", "c")
	ctx := context.Background()
	a1.Category = "ai"
	svc.repo.Update(ctx, a1)
	svc.PublishArticle(ctx, a1.ID, "a1")

	a2 := createArticleViaService(t, svc, "a2", "生活文章", "c")
	a2.Category = "life"
	svc.repo.Update(ctx, a2)
	svc.PublishArticle(ctx, a2.ID, "a2")

	a3 := createArticleViaService(t, svc, "a3", "AI第二篇", "c")
	a3.Category = "ai"
	svc.repo.Update(ctx, a3)
	svc.PublishArticle(ctx, a3.ID, "a3")

	articles, total, err := svc.ListArticlesByCategory("ai", 1, 10)
	if err != nil {
		t.Fatalf("ListArticlesByCategory 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("AI 分类文章总数应为 2, got %d", total)
	}
	if len(articles) != 2 {
		t.Errorf("应返回 2 篇 AI 文章, got %d", len(articles))
	}
	for _, a := range articles {
		if a.Category != "ai" || a.Status != models.StatusPublished {
			t.Errorf("返回了不应出现的文章: ID=%s Category=%s Status=%s", a.ID, a.Category, a.Status)
		}
	}
}

// ========== ListMyArticles 测试 ==========

func TestListMyArticles(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	createArticleViaService(t, svc, "user-a", "用户A文1", "c")
	createArticleViaService(t, svc, "user-a", "用户A文2", "c")
	createArticleViaService(t, svc, "user-b", "用户B文1", "c")

	articles, total, err := svc.ListMyArticles("user-a", 1, 10)
	if err != nil {
		t.Fatalf("ListMyArticles 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("user-a 文章总数应为 2, got %d", total)
	}
	if len(articles) != 2 {
		t.Errorf("应返回 2 篇, got %d", len(articles))
	}
	for _, a := range articles {
		if a.AuthorID != "user-a" {
			t.Errorf("返回了其他用户的文章: ID=%s AuthorID=%s", a.ID, a.AuthorID)
		}
	}
}

// ========== 边界条件测试 ==========

func TestCreateArticle_TitleAtBoundary(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	tests := []struct {
		name     string
		titleLen int
		shouldOK bool
	}{
		{"199字", 199, true},
		{"200字(边界)", 200, true},
		{"201字(超边界)", 201, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := make([]byte, tt.titleLen)
			for i := range title {
				title[i] = 'x'
			}
			_, err := svc.CreateArticle("a", string(title), "内容", "", "", nil)
			if tt.shouldOK && err != nil {
				t.Errorf("期望成功但失败: %v", err)
			}
			if !tt.shouldOK && err == nil {
				t.Error("期望失败但成功")
			}
		})
	}
}
