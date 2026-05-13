package handler

import (
	"net/http"
	"strconv"

	"blog-community/content-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

// ArticleHandler HTTP 处理层
type ArticleHandler struct {
	service *service.ArticleService
}

// NewArticleHandler 创建处理器
func NewArticleHandler(svc *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{service: svc}
}

// CreateArticle POST /api/articles
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	var req struct {
		Title    string   `json:"title" binding:"required"`
		Content  string   `json:"content" binding:"required"`
		Summary  string   `json:"summary"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	authorID := c.GetHeader("X-User-ID")
	if authorID == "" {
		utils.Error(c, http.StatusUnauthorized, "缺少用户身份")
		return
	}

	article, err := h.service.CreateArticle(authorID, req.Title, req.Content, req.Summary, req.Category, req.Tags)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusCreated, "文章创建成功", article)
}

// GetArticle GET /api/articles/:id
func (h *ArticleHandler) GetArticle(c *gin.Context) {
	articleID := c.Param("id")

	article, err := h.service.GetArticleDetail(articleID)
	if err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "获取成功", article)
}

// ListArticles GET /api/articles
func (h *ArticleHandler) ListArticles(c *gin.Context) {
	page, size := parsePagination(c)

	articles, total, err := h.service.ListArticles(page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取文章列表失败")
		return
	}

	utils.Paginated(c, articles, "获取成功", total, page, size)
}

// ListByCategory GET /api/articles?category=xxx
func (h *ArticleHandler) ListByCategory(c *gin.Context) {
	category := c.Query("category")
	page, size := parsePagination(c)

	articles, total, err := h.service.ListArticlesByCategory(category, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取分类文章失败")
		return
	}

	utils.Paginated(c, articles, "获取成功", total, page, size)
}

// EditArticle PUT /api/articles/:id
func (h *ArticleHandler) EditArticle(c *gin.Context) {
	articleID := c.Param("id")
	authorID := c.GetHeader("X-User-ID")

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Summary  string `json:"summary"`
		Category string `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	article, err := h.service.EditArticle(articleID, authorID, req.Title, req.Content, req.Summary, req.Category)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "编辑成功", article)
}

// PublishArticle POST /api/articles/:id/publish
func (h *ArticleHandler) PublishArticle(c *gin.Context) {
	articleID := c.Param("id")
	authorID := c.GetHeader("X-User-ID")

	article, err := h.service.PublishArticle(articleID, authorID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "发布成功", article)
}

// DeleteArticle DELETE /api/articles/:id
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	articleID := c.Param("id")
	authorID := c.GetHeader("X-User-ID")

	if err := h.service.DeleteArticle(articleID, authorID); err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusOK, "删除成功", nil)
}

// parsePagination 解析分页参数
func parsePagination(c *gin.Context) (int, int) {
	page := 1
	size := 10
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if s := c.Query("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 50 {
			size = v
		}
	}
	return page, size
}
