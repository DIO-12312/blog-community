package repository

import (
	"blog-community/shared/models"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 创建评论
func (r *CommentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

// GetByID 根据ID获取评论
func (r *CommentRepository) GetByID(id string) (*models.Comment, error) {
	var comment models.Comment
	err := r.db.Where("id = ?", id).First(&comment).Error
	return &comment, err
}

// Delete 软删除评论
func (r *CommentRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Comment{}).Error
}

// GetTopLevelByArticle 获取文章的顶层评论（分页）
func (r *CommentRepository) GetTopLevelByArticle(articleID string, page, size int) ([]models.Comment, int64, error) {
	var total int64
	r.db.Model(&models.Comment{}).
		Where("article_id = ? AND parent_id IS NULL", articleID).
		Count(&total)

	var comments []models.Comment
	err := r.db.Where("article_id = ? AND parent_id IS NULL", articleID).
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&comments).Error

	return comments, total, err
}

// GetChildrenByParentIDs 批量获取子评论
func (r *CommentRepository) GetChildrenByParentIDs(parentIDs []string) ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("parent_id IN ?", parentIDs).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

// ListAll 管理员获取所有评论（含已删除）
func (r *CommentRepository) ListAll(page, size int) ([]models.Comment, int64, error) {
	var total int64
	var comments []models.Comment
	r.db.Unscoped().Model(&models.Comment{}).Count(&total)
	err := r.db.Unscoped().Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&comments).Error
	return comments, total, err
}

// CountByArticle 获取文章评论总数
func (r *CommentRepository) CountByArticle(articleID string) int64 {
	var count int64
	r.db.Model(&models.Comment{}).Where("article_id = ?", articleID).Count(&count)
	return count
}
