package models

// AuditLog 审计日志
type AuditLog struct {
	BaseModel
	UserID     string `gorm:"index;size:36" json:"user_id"`
	Action     string `gorm:"index;size:50" json:"action"`
	Resource   string `gorm:"size:50" json:"resource"`
	ResourceID string `gorm:"size:36" json:"resource_id"`
	Detail     string `gorm:"type:text" json:"detail"`
	IP         string `gorm:"size:50" json:"ip"`
}
