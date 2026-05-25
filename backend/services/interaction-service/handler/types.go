package handler

// CommentResponse 评论响应（含用户信息和子评论）
type CommentResponse struct {
	ID        string            `json:"id"`
	ArticleID string            `json:"article_id"`
	UserID    string            `json:"user_id"`
	Username  string            `json:"username"`
	Avatar    string            `json:"avatar"`
	Content   string            `json:"content"`
	ParentID  *string           `json:"parent_id"`
	CreatedAt string            `json:"created_at"`
	Children  []CommentResponse `json:"children"`
}
