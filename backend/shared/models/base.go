package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string         `gorm:"primaryKey;size:36" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

//GORM的钩子函数:
//在执行某个函数前，触发对应的钩子函数

func (b *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	//生成UUID
	//使用 UUID 而非自增 ID，在分布式环境下不会冲突
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}
