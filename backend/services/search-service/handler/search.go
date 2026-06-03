package handler

import (
	"net/http"
	"strconv"

	"blog-community/search-service/service"
	"blog-community/shared/utils"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	service *service.SearchService
}

func NewSearchHandler(service *service.SearchService) *SearchHandler {
	return &SearchHandler{service: service}
}

// Search GET /api/search?q=关键词&page=1&size=10
func (h *SearchHandler) Search(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		utils.Error(c, http.StatusBadRequest, "搜索关键词不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	result, err := h.service.Search(keyword, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "搜索失败")
		return
	}

	utils.Paginated(c, result.Articles, "ok", result.Total, page, size)
}
