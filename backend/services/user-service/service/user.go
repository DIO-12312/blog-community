package service

import (
	"errors"
	"log"
	"time"

	"blog-community/shared/events"
	"blog-community/shared/models"
	"blog-community/user-service/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo      *repository.UserRepository
	jwtSecret []byte
	publisher *events.Publisher
}

func NewUserService(repo *repository.UserRepository, jwtSecret []byte, publisher *events.Publisher) *UserService {
	return &UserService{
		repo:      repo,
		jwtSecret: jwtSecret,
		publisher: publisher,
	}
}

// 用户注册
func (s *UserService) Register(username, email, password string) (*models.User, error) {
	// 1. 检查用户名和邮箱是否已存在
	if _, ok := s.repo.GetUserByUsername(username); ok {
		return nil, errors.New("用户已存在")
	}

	if _, ok := s.repo.GetUserByEmail(email); ok {
		return nil, errors.New("邮箱已被使用")
	}

	// bcrypt生成密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码生成失败")
	}

	// 创建用户
	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	err = s.repo.CreateUser(user)
	if err != nil {
		return nil, errors.New("用户创建失败")
	}

	return user, nil
}

// 用户登录，返回JWT token
func (s *UserService) Login(username, password string) (string, error) {

	user, ok := s.repo.GetUserByUsername(username)
	if !ok {
		return "", errors.New("用户名或密码错误")
	}
	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("用户名或密码错误")
	}

	//生成JWT
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"roles":   []string{"user"},
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}).SignedString(s.jwtSecret)
	if err != nil {
		return "", errors.New("生成token失败")
	}
	return token, nil
}

// 获取用户信息
func (s *UserService) GetProfile(id, username, email string) (*models.User, error) {

	if id != "" {
		user, ok := s.repo.GetUserByID(id)
		if !ok {
			return nil, errors.New("用户不存在")
		}
		return user, nil
	}
	if username != "" {
		user, ok := s.repo.GetUserByUsername(username)
		if !ok {
			return nil, errors.New("用户不存在")
		}
		return user, nil
	}
	if email != "" {
		user, ok := s.repo.GetUserByEmail(email)
		if !ok {
			return nil, errors.New("用户不存在")
		}
		return user, nil
	}
	return nil, errors.New("查询参数为空")
}

// 更新用户信息
func (s *UserService) UpdateProfile(id string, updates map[string]interface{}) error {
	if err := s.repo.UpdateUsers(id, updates); err != nil {
		return errors.New("更新用户信息失败")
	}
	return nil
}

// Follow 关注用户
func (s *UserService) Follow(followerID, followingID string) error {
	// 不能关注自己
	if followerID == followingID {
		return errors.New("不能关注自己")
	}
	// 被关注用户是否存在
	if _, ok := s.repo.GetUserByID(followingID); !ok {
		return errors.New("被关注用户不存在")
	}
	// 是否已经关注
	if ok, _ := s.repo.IsFollowing(followerID, followingID); ok {
		return errors.New("请勿重复关注")
	}
	if err := s.repo.Follow(followerID, followingID); err != nil {
		return err
	}

	// 异步发布关注事件，通知被关注用户
	if s.publisher != nil {
		follower, ok := s.repo.GetUserByID(followerID)
		if ok {
			go func() {
				if err := s.publisher.Publish(events.EventUserFollowed, map[string]interface{}{
					"follower_id":   followerID,
					"following_id":  followingID,
					"follower_name": follower.Username,
				}); err != nil {
					log.Printf("发布关注事件失败: %v", err)
				}
			}()
		}
	}

	return nil
}

func (s *UserService) UnFollow(followerID, followingID string) error {
	// 被关注用户是否存在
	if _, ok := s.repo.GetUserByID(followingID); !ok {
		return errors.New("该用户不存在")
	}

	//未关注
	if ok, _ := s.repo.IsFollowing(followerID, followingID); !ok {
		return errors.New("未关注该用户")
	}
	return s.repo.UnFollow(followerID, followingID)
}

// GetFollowers 获取粉丝列表
func (s *UserService) GetFollowers(userID string, page, size int) ([]models.User, int64, error) {
	return s.repo.GetFollowers(userID, page, size)
}

// GetFollowing 获取关注列表
func (s *UserService) GetFollowings(userID string, page, size int) ([]models.User, int64, error) {
	return s.repo.GetFollowings(userID, page, size)
}
