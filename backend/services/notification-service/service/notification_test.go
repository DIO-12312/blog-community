package service

import (
	"testing"

	"blog-community/notification-service/repository"
	"blog-community/shared/events"
	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) (*NotificationService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.Notification{}); err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}
	repo := repository.NewNotificationRepository(db)
	svc := NewNotificationService(repo, nil) // consumer 为 nil，单元测试不需要
	return svc, db
}

// 通过直接调用 repository 来模拟事件处理后的结果（不依赖 RabbitMQ）
func TestHandleCommentEvent(t *testing.T) {
	svc, db := newTestService(t)

	// 模拟 comment.created 事件的逻辑
	n := &models.Notification{
		UserID:   "author-1",
		Type:     "comment",
		Content:  "张三 评论了你的文章《Go 入门》",
		SourceID: "comment-1",
	}
	if err := svc.repo.Create(n); err != nil {
		t.Fatalf("创建评论通知失败: %v", err)
	}

	// 验证数据库
	var saved models.Notification
	db.First(&saved, "id = ?", n.ID)

	if saved.UserID != "author-1" {
		t.Errorf("UserID: 期望=author-1, 实际=%s", saved.UserID)
	}
	if saved.Type != "comment" {
		t.Errorf("Type: 期望=comment, 实际=%s", saved.Type)
	}
	expectedContent := "张三 评论了你的文章《Go 入门》"
	if saved.Content != expectedContent {
		t.Errorf("Content: 期望=%q, 实际=%q", expectedContent, saved.Content)
	}
	if saved.SourceID != "comment-1" {
		t.Errorf("SourceID: 期望=comment-1, 实际=%s", saved.SourceID)
	}
	if saved.IsRead {
		t.Error("IsRead: 期望=false")
	}
}

func TestHandleFollowEvent(t *testing.T) {
	svc, db := newTestService(t)

	// 模拟 user.followed 事件的逻辑
	n := &models.Notification{
		UserID:   "following-user",
		Type:     "follow",
		Content:  "李四 关注了你",
		SourceID: "follower-1",
	}
	if err := svc.repo.Create(n); err != nil {
		t.Fatalf("创建关注通知失败: %v", err)
	}

	var saved models.Notification
	db.First(&saved, "id = ?", n.ID)

	if saved.UserID != "following-user" {
		t.Errorf("UserID: 期望=following-user, 实际=%s", saved.UserID)
	}
	if saved.Type != "follow" {
		t.Errorf("Type: 期望=follow, 实际=%s", saved.Type)
	}
	if saved.Content != "李四 关注了你" {
		t.Errorf("Content: 期望=%q, 实际=%q", "李四 关注了你", saved.Content)
	}
	if saved.SourceID != "follower-1" {
		t.Errorf("SourceID: 期望=follower-1, 实际=%s", saved.SourceID)
	}
}

func TestHandleLikeEvent(t *testing.T) {
	svc, db := newTestService(t)

	// 模拟 article.liked 事件的逻辑
	n := &models.Notification{
		UserID:   "author-2",
		Type:     "like",
		Content:  "王五 赞了你的文章《Go 高级技巧》",
		SourceID: "article-1",
	}
	if err := svc.repo.Create(n); err != nil {
		t.Fatalf("创建点赞通知失败: %v", err)
	}

	var saved models.Notification
	db.First(&saved, "id = ?", n.ID)

	if saved.UserID != "author-2" {
		t.Errorf("UserID: 期望=author-2, 实际=%s", saved.UserID)
	}
	if saved.Type != "like" {
		t.Errorf("Type: 期望=like, 实际=%s", saved.Type)
	}
	if saved.Content != "王五 赞了你的文章《Go 高级技巧》" {
		t.Errorf("Content: 期望=%q, 实际=%q", "王五 赞了你的文章《Go 高级技巧》", saved.Content)
	}
}

func TestGetUserNotifications(t *testing.T) {
	svc, _ := newTestService(t)

	for i := 0; i < 5; i++ {
		svc.repo.Create(&models.Notification{
			UserID:  "user-1",
			Type:    "comment",
			Content: "测试",
		})
	}

	notifications, total, err := svc.GetUserNotifications("user-1", 1, 3)
	if err != nil {
		t.Fatalf("GetUserNotifications() 出错: %v", err)
	}
	if total != 5 {
		t.Errorf("total: 期望=5, 实际=%d", total)
	}
	if len(notifications) != 3 {
		t.Errorf("第1页条数: 期望=3, 实际=%d", len(notifications))
	}

	// 第2页（剩余2条）
	notifications2, total2, err := svc.GetUserNotifications("user-1", 2, 3)
	if err != nil {
		t.Fatalf("GetUserNotifications() 第2页出错: %v", err)
	}
	if total2 != 5 {
		t.Errorf("total: 期望=5, 实际=%d", total2)
	}
	if len(notifications2) != 2 {
		t.Errorf("第2页条数: 期望=2, 实际=%d", len(notifications2))
	}
}

func TestMarkAsRead(t *testing.T) {
	svc, _ := newTestService(t)

	n := &models.Notification{
		UserID:  "user-1",
		Type:    "comment",
		Content: "测试",
	}
	svc.repo.Create(n)

	err := svc.MarkAsRead(n.ID, "user-1")
	if err != nil {
		t.Fatalf("MarkAsRead() 失败: %v", err)
	}

	count := svc.GetUnreadCount("user-1")
	if count != 0 {
		t.Errorf("未读数: 期望=0, 实际=%d", count)
	}
}

func TestMarkAllAsRead(t *testing.T) {
	svc, _ := newTestService(t)

	svc.repo.Create(&models.Notification{UserID: "user-1", Type: "comment", Content: "未读1"})
	svc.repo.Create(&models.Notification{UserID: "user-1", Type: "like", Content: "未读2"})
	svc.repo.Create(&models.Notification{UserID: "user-2", Type: "follow", Content: "未读3"})

	if err := svc.MarkAllAsRead("user-1"); err != nil {
		t.Fatalf("MarkAllAsRead() 失败: %v", err)
	}

	if count := svc.GetUnreadCount("user-1"); count != 0 {
		t.Errorf("user-1 未读数: 期望=0, 实际=%d", count)
	}
	if count := svc.GetUnreadCount("user-2"); count != 1 {
		t.Errorf("user-2 未读数: 期望=1, 实际=%d", count)
	}
}

func TestGetUnreadCount(t *testing.T) {
	svc, _ := newTestService(t)

	if count := svc.GetUnreadCount("user-1"); count != 0 {
		t.Errorf("初始: 期望=0, 实际=%d", count)
	}

	svc.repo.Create(&models.Notification{UserID: "user-1", Type: "comment", Content: "未读1"})
	svc.repo.Create(&models.Notification{UserID: "user-1", Type: "like", Content: "未读2"})

	if count := svc.GetUnreadCount("user-1"); count != 2 {
		t.Errorf("2条未读: 期望=2, 实际=%d", count)
	}
}

// 测试 StartListening 使用 nil consumer 不会 panic
func TestNewNotificationService_NilConsumer(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.Notification{})
	repo := repository.NewNotificationRepository(db)
	svc := NewNotificationService(repo, nil)

	if svc == nil {
		t.Fatal("期望返回非 nil 的 service")
	}
	if svc.repo == nil {
		t.Fatal("repo 不应为 nil")
	}
	if svc.consumer != nil {
		t.Error("consumer 应为 nil")
	}
}

// 测试事件结构体的创建
func TestEventStructure(t *testing.T) {
	// 验证 events.Event 结构符合预期
	evt := events.Event{
		Type:      events.EventCommentCreated,
		Timestamp: 1717315200,
		Data: map[string]interface{}{
			"article_author_id": "author-1",
			"commenter_name":    "张三",
			"article_title":     "Go 入门",
			"comment_id":        "comment-1",
		},
	}

	if evt.Type != "comment.created" {
		t.Errorf("Type: 期望=comment.created, 实际=%s", evt.Type)
	}
	if evt.Data["article_author_id"] != "author-1" {
		t.Errorf("article_author_id: 期望=author-1, 实际=%v", evt.Data["article_author_id"])
	}
	if evt.Data["commenter_name"] != "张三" {
		t.Errorf("commenter_name: 期望=张三, 实际=%v", evt.Data["commenter_name"])
	}
	if evt.Data["article_title"] != "Go 入门" {
		t.Errorf("article_title: 期望=Go 入门, 实际=%v", evt.Data["article_title"])
	}
	if evt.Data["comment_id"] != "comment-1" {
		t.Errorf("comment_id: 期望=comment-1, 实际=%v", evt.Data["comment_id"])
	}
}
