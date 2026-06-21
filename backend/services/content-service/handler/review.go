package handler

import (
	"net/http"

	"blog-community/content-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	service *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: svc}
}

// SubmitForReview POST /api/articles/:id/submit-review
func (h *ReviewHandler) SubmitForReview(c *gin.Context) {
	articleID := c.Param("id")
	authorID := c.GetHeader("X-User-ID")
	if authorID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}

	article, err := h.service.SubmitForReview(c.Request.Context(), articleID, authorID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "已提交审稿", article)
}

// ReviewArticle POST /api/admin/articles/:id/review
func (h *ReviewHandler) ReviewArticle(c *gin.Context) {
	articleID := c.Param("id")
	reviewerID := c.GetHeader("X-User-ID")
	if reviewerID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}

	var req struct {
		Action  string  `json:"action" binding:"required"`
		Comment *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	record, err := h.service.ReviewArticle(c.Request.Context(), articleID, reviewerID, req.Action, req.Comment)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "审稿完成", record)
}

// GetReviewHistory GET /api/articles/:id/review-history
func (h *ReviewHandler) GetReviewHistory(c *gin.Context) {
	articleID := c.Param("id")
	authorID := c.GetHeader("X-User-ID")
	if authorID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}

	records, err := h.service.GetReviewHistory(c.Request.Context(), articleID, authorID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "获取成功", records)
}

// ListPendingReviews GET /api/admin/reviews/pending
func (h *ReviewHandler) ListPendingReviews(c *gin.Context) {
	page, size := parsePagination(c)

	articles, total, err := h.service.GetPendingArticles(page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取待审文章列表失败")
		return
	}
	utils.Paginated(c, articles, "获取成功", total, page, size)
}
