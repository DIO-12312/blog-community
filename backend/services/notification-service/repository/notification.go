package repository

import (
	"fmt"

	"blog-community/shared/models"

	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(n *models.Notification) error {
	return r.db.Create(n).Error
}

func (r *NotificationRepository) GetByUserID(userID string, page, size int) ([]models.Notification, int64, error) {
	var total int64
	r.db.Model(&models.Notification{}).Where("user_id = ?", userID).Count(&total)

	var notifications []models.Notification
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&notifications).Error
	return notifications, total, err
}

func (r *NotificationRepository) MarkAsRead(id, userID string) error {
	result := r.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)
	if result.Error != nil {
		return fmt.Errorf("标记已读失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("通知不存在或无权操作")
	}
	return nil
}

func (r *NotificationRepository) MarkAllAsRead(userID string) error {
	return r.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

func (r *NotificationRepository) GetUnreadCount(userID string) int64 {
	var count int64
	r.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count)
	return count
}

// GetAdminUserIDs 查询所有管理员的用户 ID
func (r *NotificationRepository) GetAdminUserIDs() ([]string, error) {
	var ids []string
	err := r.db.Model(&models.User{}).Where("role = ?", "admin").Pluck("id", &ids).Error
	return ids, err
}
