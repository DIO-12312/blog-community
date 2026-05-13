package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"blog-community/content-service/repository"
	"blog-community/shared/models"
)

// ArticleService 文章业务逻辑层
type ArticleService struct {
	repo *repository.ArticleRepository
}

// NewArticleService 创建文章服务
func NewArticleService(repo *repository.ArticleRepository) *ArticleService {
	return &ArticleService{repo}
}

// CreateArticle 创建文章
func (s *ArticleService) CreateArticle(authorID, title, content, summary, category string, tags []string) (*models.Article, error) {
	// 1. 校验输入
	if title == "" {
		return nil, errors.New("标题不能为空")
	}
	if len(title) > 200 {
		return nil, errors.New("标题不能超过 200 字")
	}
	if content == "" {
		return nil, errors.New("内容不能为空")
	}

	// 2. 创建文章实例
	article := &models.Article{
		BaseModel:    models.BaseModel{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		AuthorID:     authorID,
		Title:        title,
		Content:      content,
		Summary:      summary,
		Category:     category,
		Status:       models.StatusDraft, // 默认草稿
		ViewCount:    0,
		LikeCount:    0,
		CommentCount: 0,
	}

	// 处理标签
	if len(tags) > 0 {
		jsonTags, _ := json.Marshal(tags)
		article.Tags = jsonTags
	}

	// 3. 保存到数据库
	if err := s.repo.Create(article); err != nil {
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}

	return article, nil
}

// EditArticle 编辑文章（仅草稿可编辑）
func (s *ArticleService) EditArticle(articleID, authorID, title, content, summary, category string) (*models.Article, error) {
	// 1. 获取文章
	article, err := s.repo.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	// 2. 权限检查
	if article.AuthorID != authorID {
		return nil, errors.New("只有作者可以编辑")
	}

	// 3. 状态检查
	if article.Status != models.StatusDraft {
		return nil, errors.New("只能编辑草稿状态的文章")
	}

	// 4. 更新字段
	article.Title = title
	article.Content = content
	article.Summary = summary
	article.Category = category
	article.UpdatedAt = time.Now()

	// 5. 保存
	if err := s.repo.Update(article); err != nil {
		return nil, fmt.Errorf("更新文章失败: %w", err)
	}

	return article, nil
}

// PublishArticle 发布文章（只能发布自己的草稿）
func (s *ArticleService) PublishArticle(articleID, authorID string) (*models.Article, error) {
	article, err := s.repo.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	if article.AuthorID != authorID {
		return nil, errors.New("只有作者可以发布")
	}

	if article.Status == models.StatusPublished {
		return nil, errors.New("文章已发布")
	}

	article.Status = models.StatusPublished
	now := time.Now()
	article.PublishedAt = now
	article.UpdatedAt = now

	if err := s.repo.Update(article); err != nil {
		return nil, fmt.Errorf("发布文章失败: %w", err)
	}

	// TODO: 发布事件到 RabbitMQ，通知其他服务

	return article, nil
}

// DeleteArticle 删除文章（软删除）
func (s *ArticleService) DeleteArticle(articleID, authorID string) error {
	article, err := s.repo.GetByID(articleID)
	if err != nil {
		return err
	}

	if article.AuthorID != authorID {
		return errors.New("只有作者可以删除")
	}

	return s.repo.Delete(articleID)
}

// GetArticleDetail 获取文章详情（增加浏览次数）
func (s *ArticleService) GetArticleDetail(articleID string) (*models.Article, error) {
	article, err := s.repo.GetByID(articleID)
	if err != nil {
		return nil, err
	}

	// 异步增加浏览次数（可以用 goroutine）
	go s.repo.IncrementViewCount(articleID)

	return article, nil
}

// ListArticles 列出文章
func (s *ArticleService) ListArticles(page, size int) ([]models.Article, int64, error) {
	return s.repo.ListPublished(page, size)
}

// ListArticlesByCategory 按分类列出文章
func (s *ArticleService) ListArticlesByCategory(category string, page, size int) ([]models.Article, int64, error) {
	return s.repo.ListByCategory(category, page, size)
}

// ListMyArticles 列出当前用户的文章
func (s *ArticleService) ListMyArticles(authorID string, page, size int) ([]models.Article, int64, error) {
	return s.repo.ListByAuthor(authorID, page, size)
}
