package handler

import (
	"net/http"
	"strconv"

	"blog-community/audit-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	service *service.AuditService
}

func NewAuditHandler(service *service.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// Query GET /api/audit-logs?user_id=xxx&action=article.published&resource=article&page=1&size=20
func (h *AuditHandler) Query(c *gin.Context) {
	userID := c.Query("user_id")
	action := c.Query("action")
	resource := c.Query("resource")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	logs, total, err := h.service.QueryLogs(userID, action, resource, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "查询失败")
		return
	}

	utils.Paginated(c, logs, "ok", total, page, size)
}
