package repository

import (
	"blog-community/shared/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

// Like 点赞（幂等：重复点赞不报错）
func (r *LikeRepository) Like(userID, targetID, targetType string) error {
	like := &models.Like{
		UserID:     userID,
		TargetID:   targetID,
		TargetType: targetType,
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

// Unlike 取消点赞
func (r *LikeRepository) Unlike(userID, targetID, targetType string) error {
	return r.db.Where("user_id = ? AND target_id = ? AND target_type = ?",
		userID, targetID, targetType).Delete(&models.Like{}).Error
}

// IsLiked 是否已点赞
func (r *LikeRepository) IsLiked(userID, targetID, targetType string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Like{}).
		Where("user_id = ? AND target_id = ? AND target_type = ?",
			userID, targetID, targetType).
		Count(&count).Error
	return count > 0, err
}

// GetLikeCount 获取点赞数
func (r *LikeRepository) GetLikeCount(targetID, targetType string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Like{}).
		Where("target_id = ? AND target_type = ?", targetID, targetType).
		Count(&count).Error
	return count, err
}

// GetUserLikedIDs 获取用户点赞的目标ID列表（分页）
func (r *LikeRepository) GetUserLikedIDs(userID, targetType string, page, size int) ([]string, int64, error) {
	var total int64
	r.db.Model(&models.Like{}).
		Where("user_id = ? AND target_type = ?", userID, targetType).
		Count(&total)

	var likes []models.Like
	err := r.db.Where("user_id = ? AND target_type = ?", userID, targetType).
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&likes).Error
	if err != nil {
		return nil, 0, err
	}

	ids := make([]string, len(likes))
	for i, l := range likes {
		ids[i] = l.TargetID
	}
	return ids, total, nil
}
