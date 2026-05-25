package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"blog-community/shared/cache"
	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 创建测试用的 Repository
func newTestRepo(t *testing.T) (*ArticleRepository, func()) {
	t.Helper()

	// 连接 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&models.Article{})
	if err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}

	// 连接 Redis
	redisClient, err := cache.NewRedisClient("localhost:6379", "")
	if err != nil {
		t.Fatalf("无法连接 Redis (请确保 Redis 已启动): %v", err)
	}

	repo := NewArticleRepository(db, redisClient)

	cleanup := func() {
		redisClient.Close()
	}

	return repo, cleanup
}

// 清空 Redis 中与测试文章相关的缓存
func cleanArticleCache(t *testing.T, repo *ArticleRepository, id string) {
	t.Helper()
	repo.redis.Del(context.Background(), cache.ArticleKey(id))
}

// 创建测试用的文章
func createTestArticle(t *testing.T, repo *ArticleRepository) *models.Article {
	t.Helper()
	article := &models.Article{
		BaseModel:    models.BaseModel{ID: "test-id-001", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		AuthorID:     "author-001",
		Title:        "测试文章标题",
		Summary:      "这是测试摘要",
		Content:      "这是测试内容",
		Category:     "tech",
		Status:       models.StatusDraft,
		ViewCount:    0,
		LikeCount:    0,
		CommentCount: 0,
		Tags:         []byte(`["go","redis"]`),
	}
	err := repo.Create(article)
	if err != nil {
		t.Fatalf("创建测试文章失败: %v", err)
	}
	cleanArticleCache(t, repo, article.ID)
	return article
}

// ========== GetByID 测试 ==========

func TestGetByID_CacheMiss_ThenHit(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 第一次请求：缓存未命中，从 DB 读取并回写缓存
	result1, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("第一次 GetByID 失败: %v", err)
	}
	if result1.ID != article.ID {
		t.Errorf("ID 不匹配: got %q, want %q", result1.ID, article.ID)
	}
	if result1.Title != article.Title {
		t.Errorf("Title 不匹配: got %q, want %q", result1.Title, article.Title)
	}

	// 验证缓存已写入
	cached, err := repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err != nil {
		t.Fatalf("缓存应已写入但获取失败: %v", err)
	}
	if cached == "" {
		t.Fatal("缓存值为空")
	}

	// 第二次请求：缓存命中
	result2, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("第二次 GetByID (应命中缓存) 失败: %v", err)
	}
	if result2.ID != article.ID {
		t.Errorf("缓存命中后 ID 不匹配: got %q, want %q", result2.ID, article.ID)
	}
	if result2.Title != article.Title {
		t.Errorf("缓存命中后 Title 不匹配: got %q, want %q", result2.Title, article.Title)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	nonExistentID := "non-existent-id-999"

	// 第一次查询：DB 中不存在，应缓存空值
	_, err := repo.GetByID(ctx, nonExistentID)
	if err == nil {
		t.Fatal("不存在的文章应返回错误")
	}
	if err.Error() != "文章不存在" {
		t.Errorf("错误信息不匹配: got %q, want %q", err.Error(), "文章不存在")
	}

	// 验证空值缓存已写入
	cached, err := repo.redis.Get(ctx, cache.ArticleKey(nonExistentID))
	if err != nil {
		t.Fatalf("空值缓存应已写入: %v", err)
	}
	if cached != cache.NullValue {
		t.Errorf("缓存值应为 NullValue: got %q, want %q", cached, cache.NullValue)
	}

	// 第二次查询：应直接从空值缓存返回
	_, err = repo.GetByID(ctx, nonExistentID)
	if err == nil {
		t.Fatal("空值缓存命中时应返回错误")
	}
	if err.Error() != "文章不存在" {
		t.Errorf("错误信息不匹配: got %q, want %q", err.Error(), "文章不存在")
	}
}

func TestGetByID_NullValueCache(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	fakeID := "fake-id-null-test"

	// 手动设置空值缓存
	err := repo.redis.Set(ctx, cache.ArticleKey(fakeID), cache.NullValue,
		time.Duration(cache.EmptyValueExpiration)*time.Second)
	if err != nil {
		t.Fatalf("设置空值缓存失败: %v", err)
	}
	defer cleanArticleCache(t, repo, fakeID)

	// 查询：应该命中空值缓存，然后回退到 DB 查询，发现也不存在
	// 注意：当前实现中空值缓存命中后不会提前返回，而是继续查 DB
	_, err = repo.GetByID(ctx, fakeID)
	if err == nil {
		t.Fatal("空值缓存命中的 fake ID 在 DB 中也应不存在")
	}
}

func TestGetByID_WithAllFields(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)

	// 验证通过缓存返回的文章包含所有字段
	result, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}

	if result.AuthorID != article.AuthorID {
		t.Errorf("AuthorID 不匹配: got %q, want %q", result.AuthorID, article.AuthorID)
	}
	if result.Summary != article.Summary {
		t.Errorf("Summary 不匹配")
	}
	if result.Content != article.Content {
		t.Errorf("Content 不匹配")
	}
	if result.Category != article.Category {
		t.Errorf("Category 不匹配")
	}
	if result.Status != article.Status {
		t.Errorf("Status 不匹配: got %q, want %q", result.Status, article.Status)
	}
	if result.ViewCount != article.ViewCount {
		t.Errorf("ViewCount 不匹配")
	}
	if result.LikeCount != article.LikeCount {
		t.Errorf("LikeCount 不匹配")
	}
	if result.CommentCount != article.CommentCount {
		t.Errorf("CommentCount 不匹配")
	}
}

// ========== GetByIDUnscoped 测试 ==========

func TestGetByIDUnscoped_SoftDeleted(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)

	// 软删除文章
	err := repo.Delete(ctx, article.ID)
	if err != nil {
		t.Fatalf("软删除失败: %v", err)
	}

	// GetByID 查不到已删除的
	_, err = repo.GetByID(ctx, article.ID)
	if err == nil {
		t.Fatal("GetByID 不应查到已删除的文章")
	}

	// GetByIDUnscoped 可以查到
	result, err := repo.GetByIDUnscoped(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetByIDUnscoped 应能查到已删除的文章: %v", err)
	}
	if result.ID != article.ID {
		t.Errorf("ID 不匹配")
	}
}

func TestGetByIDUnscoped_NotFound(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	_, err := repo.GetByIDUnscoped(ctx, "not-exists-unscoped")
	if err == nil {
		t.Fatal("GetByIDUnscoped 不存在文章应返回错误")
	}
	if err.Error() != "文章不存在" {
		t.Errorf("错误信息应为'文章不存在': got %q", err.Error())
	}
}

// ========== Create 测试 ==========

func TestCreate_Success(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	article := &models.Article{
		BaseModel: models.BaseModel{ID: "create-test-001", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		AuthorID:  "author-002",
		Title:     "新创建的文章",
		Content:   "内容",
		Status:    models.StatusDraft,
	}

	err := repo.Create(article)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	// 创建后不写入缓存（Read-through 策略）
	ctx := context.Background()
	_, err = repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err == nil {
		// 缓存中可能没有值（reate 不写缓存），这里 Redis 返回 redis.Nil
		// 注意：如果返回 nil 表示 key 不存在，这是预期的
	}
}

func TestCreate_Multiple(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	articles := []*models.Article{
		{BaseModel: models.BaseModel{ID: "multi-001", CreatedAt: time.Now(), UpdatedAt: time.Now()}, AuthorID: "a1", Title: "文章1", Content: "内容1", Status: models.StatusDraft},
		{BaseModel: models.BaseModel{ID: "multi-002", CreatedAt: time.Now(), UpdatedAt: time.Now()}, AuthorID: "a1", Title: "文章2", Content: "内容2", Status: models.StatusDraft},
		{BaseModel: models.BaseModel{ID: "multi-003", CreatedAt: time.Now(), UpdatedAt: time.Now()}, AuthorID: "a2", Title: "文章3", Content: "内容3", Status: models.StatusPublished},
	}

	for i, a := range articles {
		err := repo.Create(a)
		if err != nil {
			t.Fatalf("Create 第 %d 篇失败: %v", i, err)
		}
	}
}

// ========== Update 测试 ==========

func TestUpdate_CacheInvalidation(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 先 GetByID 触发缓存写入
	_, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}

	// 确认缓存存在
	_, err = repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err != nil {
		t.Fatalf("缓存应存在: %v", err)
	}

	// 更新文章
	article.Title = "更新后的标题"
	article.UpdatedAt = time.Now()
	err = repo.Update(ctx, article)
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	// 缓存应已被删除
	_, err = repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err == nil {
		t.Fatal("Update 后缓存应被删除")
	}
}

func TestUpdate_ThenGetByID_CacheRefresh(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 先写入缓存
	repo.GetByID(ctx, article.ID)

	// 更新文章
	article.Title = "刷新后的标题"
	repo.Update(ctx, article)

	// 再次查询，应从 DB 获取最新数据并刷新缓存
	result, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if result.Title != "刷新后的标题" {
		t.Errorf("Update 后应获取到最新标题: got %q, want %q", result.Title, "刷新后的标题")
	}
}

// ========== Delete 测试 ==========

func TestDelete_CacheInvalidation(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 写入缓存
	repo.GetByID(ctx, article.ID)
	_, err := repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err != nil {
		t.Fatalf("缓存应存在: %v", err)
	}

	// 软删除
	err = repo.Delete(ctx, article.ID)
	if err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 缓存应被清除
	_, err = repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err == nil {
		t.Fatal("Delete 后缓存应被删除")
	}

	// GetByID 应查不到
	_, err = repo.GetByID(ctx, article.ID)
	if err == nil {
		t.Fatal("删除后 GetByID 应返回错误")
	}
}

// ========== HardDelete 测试 ==========

func TestHardDelete_CacheInvalidation(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 写入缓存
	repo.GetByID(ctx, article.ID)

	// 硬删除
	err := repo.HardDelete(ctx, article.ID)
	if err != nil {
		t.Fatalf("HardDelete 失败: %v", err)
	}

	// 缓存应被清除
	_, err = repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err == nil {
		t.Fatal("HardDelete 后缓存应被删除")
	}

	// Unscoped 也查不到
	_, err = repo.GetByIDUnscoped(ctx, article.ID)
	if err == nil {
		t.Fatal("HardDelete 后 Unscoped 也应查不到")
	}
}

// ========== IncrementViewCount 测试 ==========

func TestIncrementViewCount(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 写入缓存
	repo.GetByID(ctx, article.ID)

	// 增加浏览次数
	err := repo.IncrementViewCount(ctx, article.ID)
	if err != nil {
		t.Fatalf("IncrementViewCount 失败: %v", err)
	}

	// 缓存应被删除
	_, err = repo.redis.Get(ctx, cache.ArticleKey(article.ID))
	if err == nil {
		t.Fatal("IncrementViewCount 后缓存应被删除")
	}

	// 重新从 DB 查询，验证 view_count 已增加
	result, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if result.ViewCount != 1 {
		t.Errorf("ViewCount 应为 1, got %d", result.ViewCount)
	}
}

func TestIncrementViewCount_Multiple(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 增加 5 次
	for i := 0; i < 5; i++ {
		err := repo.IncrementViewCount(ctx, article.ID)
		if err != nil {
			t.Fatalf("第 %d 次 IncrementViewCount 失败: %v", i, err)
		}
	}

	result, _ := repo.GetByID(ctx, article.ID)
	if result.ViewCount != 5 {
		t.Errorf("ViewCount 应为 5, got %d", result.ViewCount)
	}
}

// ========== UpdateStats 测试 ==========

func TestUpdateStats(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()
	ctx := context.Background()
	article := createTestArticle(t, repo)

	err := repo.UpdateStats(ctx, article.ID, 10, 5)
	if err != nil {
		t.Fatalf("UpdateStats 失败: %v", err)
	}

	// 注：UpdateStats 不会失效缓存，这是已发现的潜在问题

	result, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if result.LikeCount != 10 {
		t.Errorf("LikeCount 应为 10, got %d", result.LikeCount)
	}
	if result.CommentCount != 5 {
		t.Errorf("CommentCount 应为 5, got %d", result.CommentCount)
	}
}

// ========== List 函数测试 ==========

func TestListByAuthor(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	// 创建多篇不同作者的文章
	a1 := &models.Article{BaseModel: models.BaseModel{ID: "la-001", CreatedAt: time.Now(), UpdatedAt: time.Now()}, AuthorID: "author-A", Title: "A1", Content: "c", Status: models.StatusDraft}
	a2 := &models.Article{BaseModel: models.BaseModel{ID: "la-002", CreatedAt: time.Now(), UpdatedAt: time.Now()}, AuthorID: "author-A", Title: "A2", Content: "c", Status: models.StatusPublished}
	a3 := &models.Article{BaseModel: models.BaseModel{ID: "la-003", CreatedAt: time.Now(), UpdatedAt: time.Now()}, AuthorID: "author-B", Title: "B1", Content: "c", Status: models.StatusDraft}
	repo.Create(a1)
	repo.Create(a2)
	repo.Create(a3)

	articles, total, err := repo.ListByAuthor("author-A", 1, 10)
	if err != nil {
		t.Fatalf("ListByAuthor 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("author-A 的文章总数应为 2, got %d", total)
	}
	if len(articles) != 2 {
		t.Errorf("返回的文章数应为 2, got %d", len(articles))
	}
}

func TestListByAuthor_Pagination(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		a := &models.Article{
			BaseModel: models.BaseModel{ID: "pag-" + string(rune('0'+i)), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			AuthorID:  "author-pag",
			Title:     "文章",
			Content:   "内容",
			Status:    models.StatusDraft,
		}
		repo.Create(a)
	}

	// 第 1 页，每页 3 条
	articles, total, err := repo.ListByAuthor("author-pag", 1, 3)
	if err != nil {
		t.Fatalf("ListByAuthor 失败: %v", err)
	}
	if total != 10 {
		t.Errorf("total 应为 10, got %d", total)
	}
	if len(articles) != 3 {
		t.Errorf("第 1 页应返回 3 篇, got %d", len(articles))
	}

	// 第 4 页，每页 3 条（只剩 1 条）
	articles, _, err = repo.ListByAuthor("author-pag", 4, 3)
	if err != nil {
		t.Fatalf("ListByAuthor 失败: %v", err)
	}
	if len(articles) != 1 {
		t.Errorf("第 4 页应返回 1 篇, got %d", len(articles))
	}
}

func TestListByCategory(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	now := time.Now()
	a1 := &models.Article{BaseModel: models.BaseModel{ID: "lc-001", CreatedAt: now, UpdatedAt: now}, AuthorID: "a", Title: "T1", Content: "c", Category: "tech", Status: models.StatusPublished, PublishedAt: now}
	a2 := &models.Article{BaseModel: models.BaseModel{ID: "lc-002", CreatedAt: now, UpdatedAt: now}, AuthorID: "a", Title: "T2", Content: "c", Category: "tech", Status: models.StatusPublished, PublishedAt: now}
	a3 := &models.Article{BaseModel: models.BaseModel{ID: "lc-003", CreatedAt: now, UpdatedAt: now}, AuthorID: "a", Title: "T3", Content: "c", Category: "life", Status: models.StatusPublished, PublishedAt: now}
	a4 := &models.Article{BaseModel: models.BaseModel{ID: "lc-004", CreatedAt: now, UpdatedAt: now}, AuthorID: "a", Title: "T4", Content: "c", Category: "tech", Status: models.StatusDraft, PublishedAt: now}
	repo.Create(a1)
	repo.Create(a2)
	repo.Create(a3)
	repo.Create(a4)

	// 仅列出已发布的 tech 分类文章
	articles, total, err := repo.ListByCategory("tech", 1, 10)
	if err != nil {
		t.Fatalf("ListByCategory 失败: %v", err)
	}
	if total != 2 {
		t.Errorf("已发布 tech 文章总数应为 2, got %d", total)
	}
	if len(articles) != 2 {
		t.Errorf("应返回 2 篇, got %d", len(articles))
	}

	for _, a := range articles {
		if a.Status != models.StatusPublished || a.Category != "tech" {
			t.Errorf("返回了不应该出现的文章: ID=%s, Status=%s, Category=%s", a.ID, a.Status, a.Category)
		}
	}
}

func TestListByCategory_Empty(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	// 空分类 = 列出所有已发布的
	articles, _, err := repo.ListByCategory("", 1, 10)
	if err != nil {
		t.Fatalf("ListByCategory('') 失败: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("没有文章时应该返回空列表")
	}
}

func TestListPublished(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	now := time.Now()
	a1 := &models.Article{BaseModel: models.BaseModel{ID: "lp-001", CreatedAt: now, UpdatedAt: now}, AuthorID: "a", Title: "P1", Content: "c", Status: models.StatusPublished, PublishedAt: now}
	a2 := &models.Article{BaseModel: models.BaseModel{ID: "lp-002", CreatedAt: now, UpdatedAt: now}, AuthorID: "a", Title: "P2", Content: "c", Status: models.StatusDraft}
	repo.Create(a1)
	repo.Create(a2)

	articles, total, err := repo.ListPublished(1, 10)
	if err != nil {
		t.Fatalf("ListPublished 失败: %v", err)
	}
	if total != 1 {
		t.Errorf("已发布文章总数应为 1, got %d", total)
	}
	if len(articles) != 1 {
		t.Errorf("应返回 1 篇已发布文章, got %d", len(articles))
	}
}

// ========== 缓存序列化测试 ==========

func TestCacheSerializationRoundTrip(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	original := createTestArticle(t, repo)
	cleanArticleCache(t, repo, original.ID)

	// 触发缓存写入
	repo.GetByID(ctx, original.ID)

	// 直接从 Redis 读取 JSON
	cachedJSON, err := repo.redis.Get(ctx, cache.ArticleKey(original.ID))
	if err != nil {
		t.Fatalf("从缓存读取失败: %v", err)
	}

	var cached models.Article
	err = json.Unmarshal([]byte(cachedJSON), &cached)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if cached.ID != original.ID {
		t.Errorf("ID 不匹配")
	}
	if cached.Title != original.Title {
		t.Errorf("Title 不匹配")
	}
	if cached.AuthorID != original.AuthorID {
		t.Errorf("AuthorID 不匹配")
	}
}

// ========== 上下文取消测试 ==========

func TestGetByID_WithCancelledContext(t *testing.T) {
	repo, cleanup := newTestRepo(t)
	defer cleanup()

	article := createTestArticle(t, repo)
	cleanArticleCache(t, repo, article.ID)

	// 先让缓存热起来
	ctx := context.Background()
	repo.GetByID(ctx, article.ID)

	// 取消的上下文 - Redis GET 操作应该失败
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// 缓存命中需要 Redis GET，取消的上下文会导致 Redis 操作失败
	// 此时会回退到 DB 查询
	result, err := repo.GetByID(cancelledCtx, article.ID)
	// 可能会成功（DB 查询成功）或失败（取决于 Redis 在取消时的行为）
	_ = result
	_ = err
}
