package repository

import (
	"fmt"
	"testing"
	"time"

	"blog-community/shared/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法连接 SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.Notification{}); err != nil {
		t.Fatalf("无法迁移表: %v", err)
	}
	return db
}

func createTestNotification(t *testing.T, repo *NotificationRepository, userID, nType, content, sourceID string) *models.Notification {
	t.Helper()
	n := &models.Notification{
		UserID:   userID,
		Type:     nType,
		Content:  content,
		SourceID: sourceID,
	}
	if err := repo.Create(n); err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}
	return n
}

func TestCreate(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationRepository(db)

	n := &models.Notification{
		UserID:   "user-1",
		Type:     "comment",
		Content:  "张三 评论了你的文章《测试》",
		SourceID: "comment-1",
	}

	err := repo.Create(n)
	if err != nil {
		t.Fatalf("Create() 失败: %v", err)
	}
	// 验证 ID 自动生成
	if n.ID == "" {
		t.Error("期望 ID 自动生成，但 ID 为空")
	}
	// 验证 IsRead 默认为 false
	if n.IsRead {
		t.Error("期望 IsRead 默认为 false")
	}
	// 验证从数据库可读取
	var saved models.Notification
	if err := db.First(&saved, "id = ?", n.ID).Error; err != nil {
		t.Fatalf("从数据库读取失败: %v", err)
	}
	if saved.Content != n.Content {
		t.Errorf("期望 Content=%q, 实际=%q", n.Content, saved.Content)
	}
}

func TestGetByUserID_Pagination(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationRepository(db)

	// 为 user-1 创建 5 条通知
	for i := 0; i < 5; i++ {
		createTestNotification(t, repo, "user-1", "comment", fmt.Sprintf("通知 %d", i), fmt.Sprintf("src-%d", i))
	}
	// 为 user-2 创建 3 条通知（确保不被查出）
	for i := 0; i < 3; i++ {
		createTestNotification(t, repo, "user-2", "like", fmt.Sprintf("其他 %d", i), fmt.Sprintf("other-%d", i))
	}

	tests := []struct {
		name          string
		userID        string
		page, size    int
		expectCount   int
		expectTotal   int64
	}{
		{"第1页2条", "user-1", 1, 2, 2, 5},
		{"第2页2条", "user-1", 2, 2, 2, 5},
		{"第3页2条", "user-1", 3, 2, 1, 5},
		{"其他用户", "user-2", 1, 10, 3, 3},
		{"无通知用户", "user-99", 1, 10, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifications, total, err := repo.GetByUserID(tt.userID, tt.page, tt.size)
			if err != nil {
				t.Fatalf("GetByUserID() 出错: %v", err)
			}
			if total != tt.expectTotal {
				t.Errorf("total: 期望=%d, 实际=%d", tt.expectTotal, total)
			}
			if len(notifications) != tt.expectCount {
				t.Errorf("count: 期望=%d, 实际=%d", tt.expectCount, len(notifications))
			}
		})
	}
}

func TestGetByUserID_OrderDesc(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationRepository(db)

	n1 := createTestNotification(t, repo, "user-1", "comment", "第一条", "src-1")
	time.Sleep(10 * time.Millisecond)
	n2 := createTestNotification(t, repo, "user-1", "like", "第二条", "src-2")
	time.Sleep(10 * time.Millisecond)
	n3 := createTestNotification(t, repo, "user-1", "follow", "第三条", "src-3")

	notifications, _, err := repo.GetByUserID("user-1", 1, 10)
	if err != nil {
		t.Fatalf("GetByUserID() 出错: %v", err)
	}

	// 应该按创建时间倒序：n3 > n2 > n1
	if len(notifications) != 3 {
		t.Fatalf("期望 3 条记录，实际=%d", len(notifications))
	}
	if notifications[0].ID != n3.ID {
		t.Errorf("第1条: 期望=%s, 实际=%s", n3.ID, notifications[0].ID)
	}
	if notifications[1].ID != n2.ID {
		t.Errorf("第2条: 期望=%s, 实际=%s", n2.ID, notifications[1].ID)
	}
	if notifications[2].ID != n1.ID {
		t.Errorf("第3条: 期望=%s, 实际=%s", n1.ID, notifications[2].ID)
	}
}

func TestMarkAsRead(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationRepository(db)

	n := createTestNotification(t, repo, "user-1", "comment", "内容", "src-1")

	// 标记已读
	err := repo.MarkAsRead(n.ID, "user-1")
	if err != nil {
		t.Fatalf("MarkAsRead() 失败: %v", err)
	}

	// 验证已读状态
	var updated models.Notification
	db.First(&updated, "id = ?", n.ID)
	if !updated.IsRead {
		t.Error("期望 IsRead=true, 实际=false")
	}

	// 用错误的 userID 标记已读，应返回错误
	err = repo.MarkAsRead(n.ID, "wrong-user")
	if err == nil {
		t.Error("期望 MarkAsRead(wrong user) 返回错误，实际返回 nil")
	}
	// 通知状态不变
	var unchanged models.Notification
	db.First(&unchanged, "id = ?", n.ID)
	if !unchanged.IsRead {
		t.Error("期望 IsRead=true (状态不变), 实际=false")
	}
}

func TestMarkAllAsRead(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, repo, "user-1", "comment", "未读1", "src-1")
	createTestNotification(t, repo, "user-1", "like", "未读2", "src-2")
	createTestNotification(t, repo, "user-2", "follow", "其他人的", "src-3")

	// 标记 user-1 全部已读
	err := repo.MarkAllAsRead("user-1")
	if err != nil {
		t.Fatalf("MarkAllAsRead() 失败: %v", err)
	}

	// user-1 的未读数应为 0
	count := repo.GetUnreadCount("user-1")
	if count != 0 {
		t.Errorf("user-1 未读数: 期望=0, 实际=%d", count)
	}

	// user-2 的未读数应为 1（不受影响）
	count = repo.GetUnreadCount("user-2")
	if count != 1 {
		t.Errorf("user-2 未读数: 期望=1, 实际=%d", count)
	}
}

func TestMarkAllAsRead_AlreadyRead(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationRepository(db)

	n := createTestNotification(t, repo, "user-1", "comment", "已读通知", "src-1")
	repo.MarkAsRead(n.ID, "user-1")

	// 再次 MarkAllAsRead 应该只影响 is_read=false 的记录
	err := repo.MarkAllAsRead("user-1")
	if err != nil {
		t.Fatalf("MarkAllAsRead() 失败: %v", err)
	}
	// 仍为已读
	var updated models.Notification
	db.First(&updated, "id = ?", n.ID)
	if !updated.IsRead {
		t.Error("期望 IsRead=true, 实际=false")
	}
}

func TestGetUnreadCount(t *testing.T) {
	db := newTestDB(t)
	repo := NewNotificationRepository(db)

	// 初始为 0
	if count := repo.GetUnreadCount("user-1"); count != 0 {
		t.Errorf("初始未读数: 期望=0, 实际=%d", count)
	}

	createTestNotification(t, repo, "user-1", "comment", "未读1", "src-1")
	createTestNotification(t, repo, "user-1", "like", "未读2", "src-2")
	n3 := createTestNotification(t, repo, "user-1", "follow", "未读3", "src-3")

	if count := repo.GetUnreadCount("user-1"); count != 3 {
		t.Errorf("3条未读: 期望=3, 实际=%d", count)
	}

	repo.MarkAsRead(n3.ID, "user-1")
	if count := repo.GetUnreadCount("user-1"); count != 2 {
		t.Errorf("2条未读: 期望=2, 实际=%d", count)
	}

	repo.MarkAllAsRead("user-1")
	if count := repo.GetUnreadCount("user-1"); count != 0 {
		t.Errorf("全部已读: 期望=0, 实际=%d", count)
	}
}

func TestGetAdminUserIDs(t *testing.T) {
	db := newTestDB(t)
	// 使用 AutoMigrate 创建 users 表（确保与 GORM 模型一致，包括 deleted_at）
	db.AutoMigrate(&models.User{})
	db.Exec("INSERT INTO users (id, username, role, created_at, updated_at) VALUES ('admin-1', '管理员1', 'admin', datetime('now'), datetime('now'))")
	db.Exec("INSERT INTO users (id, username, role, created_at, updated_at) VALUES ('admin-2', '管理员2', 'admin', datetime('now'), datetime('now'))")
	db.Exec("INSERT INTO users (id, username, role, created_at, updated_at) VALUES ('user-1', '普通用户', 'user', datetime('now'), datetime('now'))")

	repo := NewNotificationRepository(db)
	ids, err := repo.GetAdminUserIDs()
	if err != nil {
		t.Fatalf("GetAdminUserIDs() 失败: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("期望 2 个管理员, 实际=%d", len(ids))
	}
}

func TestGetAdminUserIDs_Empty(t *testing.T) {
	db := newTestDB(t)
	db.AutoMigrate(&models.User{})
	// 不插入任何 admin

	repo := NewNotificationRepository(db)
	ids, err := repo.GetAdminUserIDs()
	if err != nil {
		t.Fatalf("GetAdminUserIDs() 失败: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("期望 0 个管理员, 实际=%d", len(ids))
	}
}
