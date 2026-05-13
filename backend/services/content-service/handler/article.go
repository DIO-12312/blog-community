package handler

import (
	"blog-community/content-service/service"
	"github.com/gin-gonic/gin"
)

// ArticleHandler HTTP 处理层
type ArticleHandler struct {
	service *service.ArticleService
}

// NewArticleHandler 创建处理器
func NewArticleHandler(svc *service.ArticleService) *ArticleHandler

// CreateArticle POST /api/articles
func (h *ArticleHandler) CreateArticle(c *gin.Context)

// GetArticle GET /api/articles/:id
func (h *ArticleHandler) GetArticle(c *gin.Context)

// ListArticles GET /api/articles
func (h *ArticleHandler) ListArticles(c *gin.Context)

// ListByCategory GET /api/articles?category=xxx
func (h *ArticleHandler) ListByCategory(c *gin.Context)

// EditArticle PUT /api/articles/:id
func (h *ArticleHandler) EditArticle(c *gin.Context)

// PublishArticle POST /api/articles/:id/publish
func (h *ArticleHandler) PublishArticle(c *gin.Context)

// DeleteArticle DELETE /api/articles/:id
func (h *ArticleHandler) DeleteArticle(c *gin.Context)

// parsePagination 解析分页参数
func parsePagination(c *gin.Context) (int, int)
