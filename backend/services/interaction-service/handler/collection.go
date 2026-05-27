package handler

import (
	"net/http"
	"strconv"

	"blog-community/interaction-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type CollectionHandler struct {
	service *service.CollectionService
}

func NewCollectionHandler(service *service.CollectionService) *CollectionHandler {
	return &CollectionHandler{service: service}
}

// Collect POST /api/collections
func (h *CollectionHandler) Collect(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	var req struct {
		ArticleID string `json:"article_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	if err := h.service.Collect(userID, req.ArticleID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "收藏成功", nil)
}

// Uncollect DELETE /api/collections/:article_id
func (h *CollectionHandler) Uncollect(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	articleID := c.Param("article_id")

	if err := h.service.Uncollect(userID, articleID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "已取消收藏", nil)
}

// GetStatus GET /api/collections/status?article_id=xxx
func (h *CollectionHandler) GetStatus(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	articleID := c.Query("article_id")

	isCollected, count := h.service.GetCollectionStatus(userID, articleID)
	utils.Success(c, http.StatusOK, "ok", gin.H{
		"is_collected": isCollected,
		"count":        count,
	})
}

// GetMyCollections GET /api/collections?page=1&size=10
func (h *CollectionHandler) GetMyCollections(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	ids, total, err := h.service.GetUserCollections(userID, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取收藏列表失败")
		return
	}

	utils.Paginated(c, ids, "ok", total, page, size)
}
