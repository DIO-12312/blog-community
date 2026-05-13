package handler

import (
	"fmt"
	"testing"

	"blog-community/content-service/repository"
	"blog-community/content-service/service"
	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	db.AutoMigrate(&models.Article{})
	return db
}

func TestArticleLifecycle(t *testing.T) {
	db := setupTestDB()
	repo := repository.NewArticleRepository(db)
	svc := service.NewArticleService(repo)

	// 1. 创建文章
	article, err := svc.CreateArticle("user1", "测试文章", "内容", "摘要", "技术", []string{"Go", "微服务"})
	if err != nil {
		t.Fatalf("创建文章失败: %v", err)
	}
	if article.Status != models.StatusDraft {
		t.Error("新创建的文章应该是草稿状态")
	}

	// 2. 编辑文章
	updated, err := svc.EditArticle(article.ID, "user1", "更新标题", "更新内容", "更新摘要", "技术")
	if err != nil {
		t.Fatalf("编辑文章失败: %v", err)
	}
	if updated.Title != "更新标题" {
		t.Error("文章标题未更新")
	}

	// 3. 发布文章
	published, err := svc.PublishArticle(article.ID, "user1")
	if err != nil {
		t.Fatalf("发布文章失败: %v", err)
	}
	if published.Status != models.StatusPublished {
		t.Error("文章状态应该是已发布")
	}
	if published.PublishedAt.IsZero() {
		t.Error("发布时间应该被设置")
	}

	// 4. 不能重复发布
	_, err = svc.PublishArticle(article.ID, "user1")
	if err == nil {
		t.Error("不应该重复发布已发布的文章")
	}

	// 5. 权限检查：其他用户不能编辑
	_, err = svc.EditArticle(article.ID, "user2", "标题", "内容", "摘要", "技术")
	if err == nil {
		t.Error("其他用户不应该能编辑他人文章")
	}

	// 6. 删除文章
	err = svc.DeleteArticle(article.ID, "user1")
	if err != nil {
		t.Fatalf("删除文章失败: %v", err)
	}

	// 7. 软删除验证：查询不到已删除文章
	_, err = repo.GetByID(article.ID)
	if err == nil {
		t.Error("已删除的文章不应该被查询到")
	}

	// 8. Unscoped 可以查到已删除文章
	found, err := repo.GetByIDUnscoped(article.ID)
	if err != nil {
		t.Fatalf("Unscoped 应该能查到已删除的文章: %v", err)
	}
	if !found.DeletedAt.Valid {
		t.Error("已删除的文章 DeletedAt 应该不为空")
	}
}

func TestPagination(t *testing.T) {
	db := setupTestDB()
	repo := repository.NewArticleRepository(db)
	svc := service.NewArticleService(repo)

	// 创建 25 篇文章
	for i := 1; i <= 25; i++ {
		svc.CreateArticle("user1", fmt.Sprintf("文章%d", i), "内容", "摘要", "技术", nil)
	}

	// 第一页
	articles, total, err := svc.ListMyArticles("user1", 1, 10)
	if err != nil {
		t.Fatalf("获取列表失败: %v", err)
	}
	if len(articles) != 10 {
		t.Errorf("第一页应该有 10 条，实际 %d", len(articles))
	}
	if total != 25 {
		t.Errorf("总数应该是 25，实际 %d", total)
	}

	// 第二页
	articles, _, _ = svc.ListMyArticles("user1", 2, 10)
	if len(articles) != 10 {
		t.Errorf("第二页应该有 10 条，实际 %d", len(articles))
	}

	// 第三页
	articles, _, _ = svc.ListMyArticles("user1", 3, 10)
	if len(articles) != 5 {
		t.Errorf("第三页应该有 5 条，实际 %d", len(articles))
	}
}
