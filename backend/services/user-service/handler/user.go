package handler

import (
	"blog-community/shared/utils"
	"blog-community/user-service/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Register API格式:POST /api/users/register
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=50"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误"+err.Error())
		return
	}
	user, err := h.service.Register(req.Username, req.Email, req.Password)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.Success(c, http.StatusCreated, "注册成功", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// Login POST /api/users/login
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误"+err.Error())
		return
	}
	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "登录成功", gin.H{
		"token": token,
	})

}

// GetProfile GET /api/users/:id
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.Param("id")
	user, err := h.service.GetProfile(userID)

	if err != nil {
		utils.Error(c, http.StatusNotFound, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "获取用户信息成功", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"avatar":   user.Avatar,
		"bio":      user.Bio,
	})
}

// UpdateProfile PUT /api/users/:id
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.Param("id")
	currentID := c.GetHeader("X-User-ID")

	if userID != currentID {
		utils.Error(c, http.StatusForbidden, "只能更新自己的信息")
		return
	}

	var req struct {
		Bio    string `json:"bio"`
		Avatar string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "参数错误"+err.Error())
		return
	}

	if err := h.service.UpdateProfile(userID, map[string]interface{}{
		"bio":    req.Bio,
		"avatar": req.Avatar,
	}); err != nil {
		utils.Error(c, http.StatusInternalServerError, "更新用户信息失败"+err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "更新用户信息成功", nil)

}

// Follow POST /api/users/:id/follow
func (h *UserHandler) Follow(c *gin.Context) {
	followerID := c.GetHeader("X-User-ID")
	followingID := c.Param("id")
	err := h.service.Follow(followerID, followingID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "关注成功", nil)
}

// UnFollow DELETE /api/users/:id/follow
func (h *UserHandler) UnFollow(c *gin.Context) {
	followerID := c.GetHeader("X-User-ID")
	followingID := c.Param("id")
	err := h.service.UnFollow(followerID, followingID)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.Success(c, http.StatusOK, "取消关注成功", nil)
}

// GetFollowers GET /api/users/:id/followers
func (h *UserHandler) GetFollowers(c *gin.Context) {
	userID := c.Param("id")
	page, size := parsePagination(c)
	users, total, err := h.service.GetFollowers(userID, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取粉丝列表失败："+err.Error())
		return
	}
	utils.Paginated(c, users, "获取粉丝", total, page, size)
}

// GetFollowings GET /api/users/:id/followings
func (h *UserHandler) GetFollowings(c *gin.Context) {
	userID := c.Param("id")
	page, size := parsePagination(c)
	users, total, err := h.service.GetFollowings(userID, page, size)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "获取关注列表失败："+err.Error())
		return
	}
	utils.Paginated(c, users, "获取关注", total, page, size)
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
