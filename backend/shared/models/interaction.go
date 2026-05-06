package models

import "time"

// Comment 评论（支持嵌套）
type Comment struct {
	BaseModel
	ArticleID string  `gorm:"index;size:36" json:"article_id"`
	UserID    string  `gorm:"index;size:36" json:"user_id"`
	Content   string  `gorm:"type:text" json:"content"`
	ParentID  *string `gorm:"size:36" json:"parent_id"` // 指针类型，允许为 nil
}

// Like 点赞（多态：可点赞文章或评论）
type Like struct {
	UserID     string    `gorm:"primaryKey;size:36" json:"user_id"`
	TargetID   string    `gorm:"primaryKey;size:36" json:"target_id"`
	TargetType string    `gorm:"primaryKey;size:20" json:"target_type"` // article / comment
	CreatedAt  time.Time `json:"created_at"`
}

// Collection 收藏
type Collection struct {
	UserID    string    `gorm:"primaryKey;size:36" json:"user_id"`
	ArticleID string    `gorm:"primaryKey;size:36" json:"article_id"`
	CreatedAt time.Time `json:"created_at"`
}
