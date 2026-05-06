package models

import "time"

// User 用户模型
type User struct {
	BaseModel
	Username     string `gorm:"uniqueIndex;size:50" json:"username"`
	Email        string `gorm:"uniqueIndex;size:100" json:"email"`
	PasswordHash string `gorm:"size:255" json:"-"` // json:"-" 不返回给前端
	Avatar       string `gorm:"size:255" json:"avatar"`
	Bio          string `gorm:"size:500" json:"bio"`
}

// Follow 关注关系（多对多）
type Follow struct {
	FollowerID  string    `gorm:"primaryKey;size:36" json:"follower_id"`
	FollowingID string    `gorm:"primaryKey;size:36" json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}
