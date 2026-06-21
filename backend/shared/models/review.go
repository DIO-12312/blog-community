package models

const (
	ReviewActionApproved = "approved"
	ReviewActionRejected = "rejected"
)

// ReviewRecord 审稿记录
type ReviewRecord struct {
	BaseModel
	ArticleID  string  `gorm:"index:idx_review_records_article;size:36" json:"article_id"`
	ReviewerID string  `gorm:"index:idx_review_records_reviewer;size:36" json:"reviewer_id"`
	Action     string  `gorm:"size:20" json:"action"`
	Comment    *string `gorm:"type:text" json:"comment"`
}
