package repository

import (
	"blog-community/shared/models"

	"gorm.io/gorm"
)

// 封装gorm.DB对象，提供用户相关的数据库操作方法
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造函数，接受gorm.DB对象并返回UserRepository实例
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

//在UserRepository中定义用户相关的数据库操作方法，例如创建用户、查询用户等

// CreateUser 创建用户
func (r *UserRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByID 根据ID查询用户
func (r *UserRepository) GetUserByID(id string) (*models.User, bool) {
	var user models.User

	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, false
	}
	return &user, true
}

// GetByUsername 根据用户名查询用户
func (r *UserRepository) GetUserByUsername(username string) (*models.User, bool) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, false
	}
	return &user, true
}

// GetByEmail 根据邮箱查询用户
func (r *UserRepository) GetUserByEmail(email string) (*models.User, bool) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, false
	}
	return &user, true
}

// UpdateUsers 更新用户信息
func (r *UserRepository) UpdateUsers(id string, updates map[string]interface{}) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}
