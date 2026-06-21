package service

import (
	"fmt"
	"log"

	"blog-community/notification-service/repository"
	"blog-community/shared/events"
	"blog-community/shared/models"
)

type NotificationService struct {
	repo     *repository.NotificationRepository
	consumer *events.Consumer
}

func NewNotificationService(repo *repository.NotificationRepository, consumer *events.Consumer) *NotificationService {
	return &NotificationService{repo: repo, consumer: consumer}
}

// StartListening 开始监听事件
func (s *NotificationService) StartListening() {
	// 监听文章发布事件
	s.consumer.Subscribe("notification_article", "article.published", func(event events.Event) error {
		userID := event.Data["user_id"].(string)
		title := event.Data["title"].(string)

		// 这里应该查询粉丝列表，简化处理只记录日志
		log.Printf("文章发布通知: 用户 %s 发表了《%s》", userID, title)

		// TODO: 查询粉丝列表，批量创建通知
		return nil
	})

	// 监听评论事件
	s.consumer.Subscribe("notification_comment", "comment.created", func(event events.Event) error {
		articleAuthorID := event.Data["article_author_id"].(string)
		commenterName := event.Data["commenter_name"].(string)
		articleTitle := event.Data["article_title"].(string)

		notification := &models.Notification{
			UserID:   articleAuthorID,
			Type:     "comment",
			Content:  fmt.Sprintf("%s 评论了你的文章《%s》", commenterName, articleTitle),
			SourceID: event.Data["comment_id"].(string),
		}
		return s.repo.Create(notification)
	})

	// 监听关注事件
	s.consumer.Subscribe("notification_follow", "user.followed", func(event events.Event) error {
		followingID := event.Data["following_id"].(string)
		followerName := event.Data["follower_name"].(string)

		notification := &models.Notification{
			UserID:   followingID,
			Type:     "follow",
			Content:  fmt.Sprintf("%s 关注了你", followerName),
			SourceID: event.Data["follower_id"].(string),
		}
		return s.repo.Create(notification)
	})

	// 监听点赞事件
	s.consumer.Subscribe("notification_like", "article.liked", func(event events.Event) error {
		articleAuthorID := event.Data["article_author_id"].(string)
		likerName := event.Data["liker_name"].(string)
		articleTitle := event.Data["article_title"].(string)

		notification := &models.Notification{
			UserID:   articleAuthorID,
			Type:     "like",
			Content:  fmt.Sprintf("%s 赞了你的文章《%s》", likerName, articleTitle),
			SourceID: event.Data["article_id"].(string),
		}
		return s.repo.Create(notification)
	})

	// 监听审稿提交事件 → 通知所有管理员（每人一条）
	s.consumer.Subscribe("notification_review_submitted", "article.submitted_for_review", func(event events.Event) error {
		articleID := event.Data["article_id"].(string)
		title := event.Data["title"].(string)

		adminIDs, err := s.repo.GetAdminUserIDs()
		if err != nil {
			return err
		}

		for _, adminID := range adminIDs {
			notification := &models.Notification{
				UserID:   adminID,
				Type:     "new_submission",
				Content:  fmt.Sprintf("《%s》已提交审核，请处理", title),
				SourceID: articleID,
			}
			if err := s.repo.Create(notification); err != nil {
				log.Printf("创建管理员通知失败 (admin: %s): %v", adminID, err)
			}
		}
		return nil
	})

	// 监听审稿驳回事件 → 通知作者
	s.consumer.Subscribe("notification_review_rejected", "article.review_rejected", func(event events.Event) error {
		authorID := event.Data["author_id"].(string)
		title := event.Data["title"].(string)
		articleID := event.Data["article_id"].(string)

		content := fmt.Sprintf("你的文章《%s》已被退回", title)
		if comment, ok := event.Data["comment"].(string); ok && comment != "" {
			content += "，原因：" + comment
		}

		notification := &models.Notification{
			UserID:   authorID,
			Type:     "review_rejected",
			Content:  content,
			SourceID: articleID,
		}
		return s.repo.Create(notification)
	})
}

// GetUserNotifications 获取用户通知（分页）
func (s *NotificationService) GetUserNotifications(userID string, page, size int) ([]models.Notification, int64, error) {
	return s.repo.GetByUserID(userID, page, size)
}

// MarkAsRead 标记已读
func (s *NotificationService) MarkAsRead(notificationID, userID string) error {
	return s.repo.MarkAsRead(notificationID, userID)
}

// MarkAllAsRead 全部标记已读
func (s *NotificationService) MarkAllAsRead(userID string) error {
	return s.repo.MarkAllAsRead(userID)
}

// GetUnreadCount 获取未读数量
func (s *NotificationService) GetUnreadCount(userID string) int64 {
	return s.repo.GetUnreadCount(userID)
}
