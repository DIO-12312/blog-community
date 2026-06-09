package service

import (
	"errors"
	"log"

	"blog-community/interaction-service/repository"
	"blog-community/shared/events"
	"blog-community/shared/models"

	"gorm.io/gorm"
)

type CommentService struct {
	repo      *repository.CommentRepository
	db        *gorm.DB
	publisher *events.Publisher
}

func NewCommentService(repo *repository.CommentRepository, db *gorm.DB, publisher *events.Publisher) *CommentService {
	return &CommentService{repo: repo, db: db, publisher: publisher}
}

// Create 创建评论
func (s *CommentService) Create(articleID, userID, username, content string, parentID *string) (*models.Comment, error) {
	// 如果是回复，检查父评论是否存在
	if parentID != nil {
		parent, err := s.repo.GetByID(*parentID)
		if err != nil {
			return nil, errors.New("父评论不存在")
		}
		// 确保父评论属于同一篇文章
		if parent.ArticleID != articleID {
			return nil, errors.New("父评论不属于该文章")
		}
	}

	comment := &models.Comment{
		ArticleID: articleID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		ParentID:  parentID,
	}

	if err := s.repo.Create(comment); err != nil {
		return nil, errors.New("创建评论失败")
	}

	// 异步发布评论创建事件，通知文章作者
	if s.publisher != nil {
		var article struct {
			AuthorID string
			Title    string
		}
		if err := s.db.Table("articles").Select("author_id, title").Where("id = ?", articleID).First(&article).Error; err == nil {
			var user struct {
				Username string
			}
			if err := s.db.Table("users").Select("username").Where("id = ?", userID).First(&user).Error; err == nil {
				go func() {
					if err := s.publisher.Publish(events.EventCommentCreated, map[string]interface{}{
						"comment_id":        comment.ID,
						"article_author_id": article.AuthorID,
						"article_title":     article.Title,
						"commenter_name":    user.Username,
					}); err != nil {
						log.Printf("发布评论创建事件失败: %v", err)
					}
				}()
			}
		}
	}

	return comment, nil
}

// Delete 删除评论（仅允许评论作者）
func (s *CommentService) Delete(commentID, userID string) error {
	comment, err := s.repo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("评论不存在")
		}
		return err
	}

	if comment.UserID != userID {
		return errors.New("只能删除自己的评论")
	}

	return s.repo.Delete(commentID)
}

// AdminDelete 管理员强制删除评论（绕过权限检查）
func (s *CommentService) AdminDelete(commentID string) error {
	if _, err := s.repo.GetByID(commentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("评论不存在")
		}
		return err
	}
	return s.repo.Delete(commentID)
}

// ListAll 管理员获取所有评论
func (s *CommentService) ListAll(page, size int) ([]models.Comment, int64, error) {
	return s.repo.ListAll(page, size)
}

// GetByArticle 获取文章评论（树形结构）
func (s *CommentService) GetByArticle(articleID string, page, size int) ([]models.Comment, []models.Comment, int64, error) {
	// 第一步：获取顶层评论（分页）
	topComments, total, err := s.repo.GetTopLevelByArticle(articleID, page, size)
	if err != nil {
		return nil, nil, 0, err
	}

	if len(topComments) == 0 {
		return topComments, nil, total, nil
	}

	// 第二步：批量获取所有子评论
	parentIDs := make([]string, len(topComments))
	for i, c := range topComments {
		parentIDs[i] = c.ID
	}

	children, err := s.repo.GetChildrenByParentIDs(parentIDs)
	if err != nil {
		return topComments, nil, total, nil // 子评论获取失败不影响顶层
	}

	return topComments, children, total, nil
}
