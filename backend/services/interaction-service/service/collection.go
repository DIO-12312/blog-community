package service

import (
	"blog-community/interaction-service/repository"
)

type CollectionService struct {
	repo *repository.CollectionRepository
}

func NewCollectionService(repo *repository.CollectionRepository) *CollectionService {
	return &CollectionService{repo: repo}
}

// Collect 收藏文章
func (s *CollectionService) Collect(userID, articleID string) error {
	return s.repo.Collect(userID, articleID)
}

// Uncollect 取消收藏
func (s *CollectionService) Uncollect(userID, articleID string) error {
	return s.repo.Uncollect(userID, articleID)
}

// GetCollectionStatus 获取收藏状态和数量
func (s *CollectionService) GetCollectionStatus(userID, articleID string) (bool, int64) {
	isCollected := s.repo.IsCollected(userID, articleID)
	count := s.repo.GetCollectionCount(articleID)
	return isCollected, count
}

// GetUserCollections 获取收藏的文章
func (s *CollectionService) GetUserCollections(userID string, page, size int) ([]string, int64, error) {
	return s.repo.GetUserCollections(userID, page, size)
}
