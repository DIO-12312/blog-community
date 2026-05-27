package repository

import (
	"blog-community/shared/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CollectionRepository struct {
	db *gorm.DB
}

func NewCollectionRepository(db *gorm.DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

// Collect 收藏文章（幂等）
func (r *CollectionRepository) Collect(userID, articleID string) error {
	collection := &models.Collection{
		UserID:    userID,
		ArticleID: articleID,
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(collection).Error
}

// Uncollect 取消收藏
func (r *CollectionRepository) Uncollect(userID, articleID string) error {
	return r.db.Where("user_id = ? AND article_id = ?",
		userID, articleID).Delete(&models.Collection{}).Error
}

// IsCollected 是否已收藏
func (r *CollectionRepository) IsCollected(userID, articleID string) bool {
	var count int64
	r.db.Model(&models.Collection{}).
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Count(&count)
	return count > 0
}

// GetCollectionCount 获取收藏数
func (r *CollectionRepository) GetCollectionCount(articleID string) int64 {
	var count int64
	r.db.Model(&models.Collection{}).
		Where("article_id = ?", articleID).
		Count(&count)
	return count
}

// GetUserCollections 获取用户收藏列表（分页）
func (r *CollectionRepository) GetUserCollections(userID string, page, size int) ([]string, int64, error) {
	var total int64
	r.db.Model(&models.Collection{}).
		Where("user_id = ?", userID).
		Count(&total)

	var collections []models.Collection
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&collections).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]string, len(collections))
	for i, c := range collections {
		ids[i] = c.ArticleID
	}
	return ids, total, nil
}
