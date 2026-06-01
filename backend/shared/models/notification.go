package models

// Notification 通知
type Notification struct {
	BaseModel
	UserID   string `gorm:"index;size:36" json:"user_id"` // 接收人
	Type     string `gorm:"size:50" json:"type"`          // 通知类型
	Content  string `gorm:"size:500" json:"content"`      // 通知内容
	SourceID string `gorm:"size:36" json:"source_id"`     // 来源ID（文章/评论/用户）
	IsRead   bool   `gorm:"default:false" json:"is_read"` // 是否已读
}
