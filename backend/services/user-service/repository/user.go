package repository

import (
	"blog-community/shared/models"
	"fmt"

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

//Follow 关注用户

func (r *UserRepository) Follow(followerID, followingID string) error {
	follow := &models.Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}
	return r.db.Create(follow).Error
}

// UnFollow 取关用户
func (r *UserRepository) UnFollow(followerID, followingID string) error {
	return r.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&models.Follow{}).Error
}

// IsFollowing 查看是否关注
func (r *UserRepository) IsFollowing(followerID, followingID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Follow{}).Where("follower_id = ? AND following_id = ?", followerID, followingID).Count(&count).Error
	fmt.Println("当前计数为:%v", count)
	return count > 0, err
}

// GetFollowersCount 获取用户的粉丝数
func (r *UserRepository) GetFollowersCount(userID string) (int64, error) {
	var total int64
	err := r.db.Model(&models.Follow{}).Where("following_id = ?", userID).Count(&total).Error
	return total, err
}

// GetFollowingCount 获取用户的关注数
func (r *UserRepository) GetFollowingsCount(userID string) (int64, error) {
	var total int64
	err := r.db.Model(&models.Follow{}).Where("follower_id = ?", userID).Count(&total).Error
	return total, err
}

// GetFollowers 分页获取用户的粉丝列表
func (r *UserRepository) GetFollowers(userID string, page, size int) ([]models.User, int64, error) {
	total, err := r.GetFollowersCount(userID)

	if total == 0 || err != nil {
		return []models.User{}, total, err
	}

	var followers []models.Follow

	err = r.db.Where("following_id = ?", userID).Order("created_at DESC").Limit(size).Offset((page - 1) * size).Find(&followers).Error
	if err != nil {
		return nil, 0, err
	}

	//收集所有粉丝的id
	ids := make([]string, len(followers))
	for i := range followers {
		ids[i] = followers[i].FollowerID
	}
	var users []models.User
	err = r.db.Where("id IN ?", ids).Find(&users).Error
	return users, total, err
}

// GetFollowerings 获取用户的关注列表
func (r *UserRepository) GetFollowings(userID string, page, size int) ([]models.User, int64, error) {
	total, err := r.GetFollowingsCount(userID)

	if total == 0 || err != nil {
		return []models.User{}, total, err
	}

	var followings []models.Follow

	err = r.db.Where("follower_id = ?", userID).Order("created_at DESC").Limit(size).Offset((page - 1) * size).Find(&followings).Error
	if err != nil {
		return nil, 0, err
	}

	//收集所有关注的id
	ids := make([]string, len(followings))
	for i := range followings {
		ids[i] = followings[i].FollowingID
	}
	fmt.Printf("查找到的ID：%v", ids[0])
	var users []models.User
	err = r.db.Where("id IN ?", ids).Find(&users).Error
	return users, total, err
}
