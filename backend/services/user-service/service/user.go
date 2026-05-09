package service

import (
	"blog-community/shared/models"
	"errors"

	"github.com/blog-community/user-service/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo      *repository.UserRepository
	jwtSecret []byte
}

func NewUserService(repo *repository.UserRepository, jwtSecret []byte) *UserService {
	return &UserService{
		repo:      repo,
		jwtSecret: jwtSecret,
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
	token := jwt
	return user, nil
}

//
