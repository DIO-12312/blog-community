package migration

import (
	"blog-community/shared/models"

	"gorm.io/gorm"
)

// RunMigrations 执行数据库迁移
func RunMigrations(db *gorm.DB) error {
	// 创建表
	if err := db.AutoMigrate(&models.Article{}); err != nil {
		return err
	}

	// 创建索引
	return db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_articles_author_id ON articles(author_id);
		CREATE INDEX IF NOT EXISTS idx_articles_category ON articles(category);
		CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
		CREATE INDEX IF NOT EXISTS idx_articles_deleted_at ON articles(deleted_at);
	`).Error
}
