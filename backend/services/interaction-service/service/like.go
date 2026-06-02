package service

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"blog-community/interaction-service/repository"
	"blog-community/shared/cache"
	"blog-community/shared/events"

	"gorm.io/gorm"
)

type LikeService struct {
	repo      *repository.LikeRepository
	counter   *cache.RedisClient
	db        *gorm.DB
	publisher *events.Publisher
}

func NewLikeService(repo *repository.LikeRepository, counter *cache.RedisClient, db *gorm.DB, publisher *events.Publisher) *LikeService {
	return &LikeService{repo: repo, counter: counter, db: db, publisher: publisher}
}

// Like 点赞
func (s *LikeService) Like(userID, targetID, targetType string) error {
	if targetType != "article" && targetType != "comment" {
		return errors.New("不支持的点赞类型")
	}
	if err := s.repo.Like(userID, targetID, targetType); err != nil {
		return err
	}
	s.delCache(targetType, targetID)

	// 异步发布点赞事件（仅文章点赞时通知文章作者）
	if s.publisher != nil && targetType == "article" {
		var article struct {
			AuthorID string
			Title    string
		}
		if err := s.db.Table("articles").Select("author_id, title").Where("id = ?", targetID).First(&article).Error; err == nil {
			var user struct {
				Username string
			}
			if err := s.db.Table("users").Select("username").Where("id = ?", userID).First(&user).Error; err == nil {
				go func() {
					if err := s.publisher.Publish(events.EventArticleLiked, map[string]interface{}{
						"article_id":        targetID,
						"article_author_id": article.AuthorID,
						"article_title":     article.Title,
						"liker_name":        user.Username,
					}); err != nil {
						log.Printf("发布点赞事件失败: %v", err)
					}
				}()
			}
		}
	}

	return nil
}

// Unlike 取消点赞
func (s *LikeService) Unlike(userID, targetID, targetType string) error {
	if err := s.repo.Unlike(userID, targetID, targetType); err != nil {
		return err
	}
	s.delCache(targetType, targetID)
	return nil
}

// GetLikeStatus 获取点赞状态和数量（Cache-Aside）
func (s *LikeService) GetLikeStatus(userID, targetID, targetType string) (bool, int64) {
	ctx := context.Background()
	likeKey := cache.LikeCountKey(targetType, targetID)

	// 1. 先从 Redis 获取点赞数
	count, hit := s.getFromCache(ctx, likeKey)
	if hit {
		isLiked, err := s.repo.IsLiked(userID, targetID, targetType)
		if err != nil {
			return false, count
		}
		return isLiked, count
	}

	// 2. Redis 未命中，查 DB
	isLiked, err := s.repo.IsLiked(userID, targetID, targetType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.setNullCache(ctx, likeKey)
		}
		return false, 0
	}
	count, err = s.repo.GetLikeCount(targetID, targetType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.setNullCache(ctx, likeKey)
		}
		return isLiked, 0
	}

	// 3. 写回 Redis 缓存
	s.setToCache(ctx, likeKey, count)

	return isLiked, count
}

// getFromCache 从 Redis 获取点赞数，hit=false 表示未命中
func (s *LikeService) getFromCache(ctx context.Context, key string) (count int64, hit bool) {
	if s.counter == nil {
		return 0, false
	}
	countStr, err := s.counter.Get(ctx, key)
	if err != nil {
		return 0, false
	}
	if countStr == cache.NullValue {
		return 0, true // 目标不存在，计数值视为 0
	}
	count, parseErr := strconv.ParseInt(countStr, 10, 64)
	if parseErr != nil {
		return 0, false
	}
	return count, true
}

// setToCache 将点赞数写入 Redis 缓存（包括 0，count=0 是有效值）
func (s *LikeService) setToCache(ctx context.Context, key string, count int64) {
	if s.counter == nil {
		return
	}
	s.counter.Set(ctx, key, count, cache.LikeExpiration*time.Second)
}

// setNullCache 目标不存在时缓存空值，防止缓存穿透
func (s *LikeService) setNullCache(ctx context.Context, key string) {
	if s.counter == nil {
		return
	}
	s.counter.Set(ctx, key, cache.NullValue, cache.EmptyValueExpiration*time.Second)
}

// delCache 写完 DB 后删除缓存，下次读取时重新加载
func (s *LikeService) delCache(targetType, targetID string) {
	if s.counter == nil {
		return
	}
	key := cache.LikeCountKey(targetType, targetID)
	s.counter.Del(context.Background(), key)
}
