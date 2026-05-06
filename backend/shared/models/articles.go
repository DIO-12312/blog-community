package models

// Article 文章模型
type Article struct {
	BaseModel
	UserID     string `gorm:"index;size:36" json:"user_id"`
	Title      string `gorm:"size:200" json:"title"`
	Content    string `gorm:"type:text" json:"content"` // Markdown 内容
	CategoryID string `gorm:"index;size:36" json:"category_id"`
	Status     string `gorm:"size:20;default:draft" json:"status"` // draft/published/deleted
	ViewCount  int64  `gorm:"default:0" json:"view_count"`
}

// Category 分类
type Category struct {
	BaseModel
	Name string `gorm:"uniqueIndex;size:50" json:"name"`
	Slug string `gorm:"uniqueIndex;size:50" json:"slug"`
}
