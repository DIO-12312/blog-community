package handler

import (
	"net/http"
	"strconv"

	"blog-community/interaction-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	service *service.CommentService
}

func NewCommentHandler(service *service.CommentService) *CommentHandler {
	return &CommentHandler{service: service}
}

// Create POST /api/articles/:id/comments
func (h *CommentHandler) Create(c *gin.Context) {
	articleID := c.Param("id")
	userID := c.GetHeader("X-User-ID")
	username := c.GetHeader("X-Username")

	var req struct {
		Content  string  `json:"content" binding:"required,min=1"`
		ParentID *string `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "评论内容不能为空")
		return
	}

	comment, err := h.service.Create(articleID, userID, username, req.Content, req.ParentID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusCreated, "评论成功", comment)
}

// Delete DELETE /api/comments/:id
func (h *CommentHandler) Delete(c *gin.Context) {
	commentID := c.Param("id")
	userID := c.GetHeader("X-User-ID")

	if err := h.service.Delete(commentID, userID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "删除成功", nil)
}

// GetByArticle GET /api/articles/:id/comments
func (h *CommentHandler) GetByArticle(c *gin.Context) {
	articleID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	topComments, children, total, err := h.service.GetByArticle(articleID, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取评论失败")
		return
	}

	// 在内存中组装树形结构
	childrenMap := make(map[string][]CommentResponse)
	for _, child := range children {
		if child.ParentID != nil {
			childrenMap[*child.ParentID] = append(childrenMap[*child.ParentID], CommentResponse{
				ID:        child.ID,
				ArticleID: child.ArticleID,
				UserID:    child.UserID,
				Username:  child.Username,
				Content:   child.Content,
				ParentID:  child.ParentID,
				CreatedAt: child.CreatedAt.Format("2006-01-02 15:04:05"),
				Children:  []CommentResponse{},
			})
		}
	}

	result := make([]CommentResponse, len(topComments))
	for i, top := range topComments {
		result[i] = CommentResponse{
			ID:        top.ID,
			ArticleID: top.ArticleID,
			UserID:    top.UserID,
				Username:  top.Username,
			Content:   top.Content,
			ParentID:  top.ParentID,
			CreatedAt: top.CreatedAt.Format("2006-01-02 15:04:05"),
			Children:  childrenMap[top.ID],
		}
		if result[i].Children == nil {
			result[i].Children = []CommentResponse{}
		}
	}

	utils.Paginated(c, result, "ok", total, page, size)
}
