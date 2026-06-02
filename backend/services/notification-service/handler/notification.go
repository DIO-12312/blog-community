package handler

import (
	"net/http"
	"strconv"

	"blog-community/notification-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// GetNotifications GET /api/notifications?page=1&size=10
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	notifications, total, err := h.service.GetUserNotifications(userID, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取通知失败")
		return
	}
	utils.Paginated(c, notifications, "ok", total, page, size)
}

// MarkAsRead PUT /api/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	notificationID := c.Param("id")

	if err := h.service.MarkAsRead(notificationID, userID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "已标记已读", nil)
}

// MarkAllAsRead PUT /api/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if err := h.service.MarkAllAsRead(userID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "全部已读", nil)
}

// GetUnreadCount GET /api/notifications/unread-count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	count := h.service.GetUnreadCount(userID)
	utils.Success(c, http.StatusOK, "ok", gin.H{"count": count})
}
