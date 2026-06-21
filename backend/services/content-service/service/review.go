package service

import (
	"context"
	"errors"
	"fmt"

	"blog-community/content-service/repository"
	"blog-community/shared/events"
	"blog-community/shared/models"
)

type ReviewService struct {
	articleRepo *repository.ArticleRepository
	reviewRepo  *repository.ReviewRepository
	publisher   *events.Publisher
}

func NewReviewService(
	articleRepo *repository.ArticleRepository,
	reviewRepo *repository.ReviewRepository,
	publisher *events.Publisher,
) *ReviewService {
	return &ReviewService{
		articleRepo: articleRepo,
		reviewRepo:  reviewRepo,
		publisher:   publisher,
	}
}

// SubmitForReview 作者提交审稿：draft → pending_review
func (s *ReviewService) SubmitForReview(ctx context.Context, articleID, authorID string) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("文章不存在: %w", err)
	}
	if article.AuthorID != authorID {
		return nil, errors.New("只有作者可以提交审稿")
	}
	if article.Status != models.StatusDraft {
		return nil, errors.New("只有草稿状态的文章可以提交审稿")
	}

	article.Status = models.StatusPendingReview
	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, fmt.Errorf("提交审稿失败: %w", err)
	}

	if s.publisher != nil {
		s.publisher.Publish(events.EventArticleSubmittedForReview, map[string]interface{}{
			"article_id": articleID,
			"author_id":  authorID,
			"title":      article.Title,
		})
	}

	return article, nil
}

// ReviewArticle 管理员审稿：pending_review → published / draft
func (s *ReviewService) ReviewArticle(ctx context.Context, articleID, reviewerID, action string, comment *string) (*models.ReviewRecord, error) {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("文章不存在: %w", err)
	}
	if article.Status != models.StatusPendingReview {
		return nil, errors.New("只能审稿待审核状态的文章")
	}

	record := &models.ReviewRecord{
		ArticleID:  articleID,
		ReviewerID: reviewerID,
		Action:     action,
		Comment:    comment,
	}

	switch action {
	case models.ReviewActionApproved:
		article.Status = models.StatusPublished
		if err := s.articleRepo.Update(ctx, article); err != nil {
			return nil, fmt.Errorf("发布文章失败: %w", err)
		}
		if err := s.reviewRepo.Create(record); err != nil {
			return nil, fmt.Errorf("创建审稿记录失败: %w", err)
		}
		if s.publisher != nil {
			s.publisher.Publish(events.EventArticlePublished, map[string]interface{}{
				"article_id": articleID,
				"user_id":    article.AuthorID,
				"title":      article.Title,
			})
		}

	case models.ReviewActionRejected:
		article.Status = models.StatusDraft
		if err := s.articleRepo.Update(ctx, article); err != nil {
			return nil, fmt.Errorf("驳回文章失败: %w", err)
		}
		if err := s.reviewRepo.Create(record); err != nil {
			return nil, fmt.Errorf("创建审稿记录失败: %w", err)
		}
		if s.publisher != nil {
			s.publisher.Publish(events.EventArticleReviewRejected, map[string]interface{}{
				"article_id": articleID,
				"author_id":  article.AuthorID,
				"title":      article.Title,
				"comment":    comment,
			})
		}

	default:
		return nil, errors.New("无效的审稿操作，必须是 approved 或 rejected")
	}

	return record, nil
}

// GetReviewHistory 获取文章的审稿历史
func (s *ReviewService) GetReviewHistory(ctx context.Context, articleID, userID string) ([]models.ReviewRecord, error) {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("文章不存在: %w", err)
	}
	if article.AuthorID != userID {
		return nil, errors.New("只有作者可以查看审稿历史")
	}
	return s.reviewRepo.GetByArticleID(articleID)
}

// GetPendingArticles 管理员获取待审文章列表
func (s *ReviewService) GetPendingArticles(page, size int) ([]models.Article, int64, error) {
	return s.reviewRepo.ListPendingArticles(page, size)
}
