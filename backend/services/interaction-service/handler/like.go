package handler

import (
	"net/http"

	"blog-community/interaction-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type LikeHandler struct {
	service *service.LikeService
}

func NewLikeHandler(service *service.LikeService) *LikeHandler {
	return &LikeHandler{service: service}
}

// Like POST /api/likes
func (h *LikeHandler) Like(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	var req struct {
		TargetID   string `json:"target_id" binding:"required"`
		TargetType string `json:"target_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	if err := h.service.Like(userID, req.TargetID, req.TargetType); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "点赞成功", nil)
}

// Unlike DELETE /api/likes
func (h *LikeHandler) Unlike(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	var req struct {
		TargetID   string `json:"target_id" binding:"required"`
		TargetType string `json:"target_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	if err := h.service.Unlike(userID, req.TargetID, req.TargetType); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "已取消点赞", nil)
}

// GetStatus GET /api/likes/status?target_id=xxx&target_type=article
func (h *LikeHandler) GetStatus(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	targetID := c.Query("target_id")
	targetType := c.Query("target_type")

	isLiked, count := h.service.GetLikeStatus(userID, targetID, targetType)
	utils.Success(c, http.StatusOK, "ok", gin.H{
		"is_liked": isLiked,
		"count":    count,
	})
}
