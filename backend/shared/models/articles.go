package models

import "time"

// Article 文章模型
type Article struct {
	BaseModel
	PublishedAt  *time.Time `json:"published_at"`
	AuthorID     string     `gorm:"index:idx_articles_author_id;size:36" json:"author_id"`
	Title        string     `gorm:"size:200" json:"title"`
	Summary      string     `gorm:"type:text" json:"summary"`
	Content      string     `gorm:"type:text" json:"content"` // Markdown 内容
	Category     string     `gorm:"index:idx_articles_category;size:36" json:"category"`
	Status       string     `gorm:"index:idx_articles_status;size:20;default:draft" json:"status"` // draft/published/deleted
	ViewCount    int64      `gorm:"default:0" json:"view_count"`
	LikeCount    int64      `gorm:"default:0" json:"like_count"`
	CommentCount int64      `gorm:"default:0" json:"comment_count"`
	Tags         []byte     `gorm:"type:json" json:"tags"` // JSON 格式存储标签列表
}

// Category 分类
type Category struct {
	BaseModel
	Name string `gorm:"uniqueIndex;size:50" json:"name"`
	Slug string `gorm:"uniqueIndex;size:50" json:"slug"`
}

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusDelete    = "deleted"
)
