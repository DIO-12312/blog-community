package repository

import (
	"blog-community/shared/models"

	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

// Query 条件查询审计日志
func (r *AuditRepository) Query(userID, action, resource string, page, size int) ([]models.AuditLog, int64, error) {
	query := r.db.Model(&models.AuditLog{})

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}

	var total int64
	query.Count(&total)

	var logs []models.AuditLog
	err := query.Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&logs).Error

	return logs, total, err
}
