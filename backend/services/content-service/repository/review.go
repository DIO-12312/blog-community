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

// ListPendingArticles 获取待审文章列表（分页）
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
