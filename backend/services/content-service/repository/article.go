package repository

import (
	"errors"

	"blog-community/shared/models"

	"gorm.io/gorm"
)

// ArticleRepository 文章数据访问层
type ArticleRepository struct {
	db *gorm.DB
}

// NewArticleRepository 创建文章仓库
func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db}
}

// Create 创建文章
func (r *ArticleRepository) Create(article *models.Article) error {
	return r.db.Create(article).Error
}

// GetByID 根据 ID 获取文章
func (r *ArticleRepository) GetByID(id string) (*models.Article, error) {
	var article models.Article
	err := r.db.First(&article, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文章不存在")
		}
		return nil, err
	}
	return &article, nil
}

// GetByIDUnscoped 获取文章（包括已删除）
func (r *ArticleRepository) GetByIDUnscoped(id string) (*models.Article, error) {
	var article models.Article
	err := r.db.Unscoped().First(&article, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文章不存在")
		}
		return nil, err
	}
	return &article, nil
}

// ListByAuthor 获取某用户的文章列表
func (r *ArticleRepository) ListByAuthor(authorID string, page, size int) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	// 计数
	r.db.Model(&models.Article{}).Where("author_id = ?", authorID).Count(&total)

	// 分页查询
	err := r.db.
		Where("author_id = ?", authorID).
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&articles).Error

	return articles, total, err
}

// ListByCategory 按分类查询
func (r *ArticleRepository) ListByCategory(category string, page, size int) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	query := r.db.Model(&models.Article{}).Where("status = ?", models.StatusPublished)
	if category != "" {
		query = query.Where("category = ?", category)
	}

	query.Count(&total)

	err := query.
		Order("published_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&articles).Error

	return articles, total, err
}

// ListPublished 获取已发布的文章列表
func (r *ArticleRepository) ListPublished(page, size int) ([]models.Article, int64, error) {
	return r.ListByCategory("", page, size)
}

// Update 更新文章
func (r *ArticleRepository) Update(article *models.Article) error {
	// 只更新指定字段，避免覆盖其他字段
	return r.db.Model(article).
		Select("title", "content", "summary", "category", "tags", "status", "updated_at").
		Updates(article).Error
}

// Delete 软删除文章
func (r *ArticleRepository) Delete(id string) error {
	return r.db.Delete(&models.Article{}, "id = ?", id).Error
}

// HardDelete 硬删除文章（仅管理员）
func (r *ArticleRepository) HardDelete(id string) error {
	return r.db.Unscoped().Delete(&models.Article{}, "id = ?", id).Error
}

// IncrementViewCount 增加浏览次数
func (r *ArticleRepository) IncrementViewCount(id string) error {
	return r.db.Model(&models.Article{}).Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// UpdateStats 更新统计数据（点赞数、评论数）
func (r *ArticleRepository) UpdateStats(id string, likeCount, commentCount int64) error {
	return r.db.Model(&models.Article{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"like_count":    likeCount,
			"comment_count": commentCount,
		}).Error
}
