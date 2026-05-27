package service

import (
	"blog-community/interaction-service/repository"
	"errors"
)

type LikeService struct {
	repo *repository.LikeRepository
}

func NewLikeService(repo *repository.LikeRepository) *LikeService {
	return &LikeService{repo: repo}
}

// Like 点赞
func (s *LikeService) Like(userID, targetID, targetType string) error {
	if targetType != "article" && targetType != "comment" {
		return errors.New("不支持的点赞类型")
	}
	return s.repo.Like(userID, targetID, targetType)
}

// Unlike 取消点赞
func (s *LikeService) Unlike(userID, targetID, targetType string) error {
	return s.repo.Unlike(userID, targetID, targetType)
}

// GetLikeStatus 获取点赞状态和数量
func (s *LikeService) GetLikeStatus(userID, targetID, targetType string) (bool, int64) {
	isLiked := s.repo.IsLiked(userID, targetID, targetType)
	count := s.repo.GetLikeCount(targetID, targetType)
	return isLiked, count
}
