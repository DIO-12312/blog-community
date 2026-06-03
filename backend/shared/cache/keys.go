package cache

import "fmt"

// 缓存键前缀和格式定义

// ArticleKey 文章详情缓存键
// key: article:article_id
func ArticleKey(articleID string) string {
	return fmt.Sprintf("article:%s", articleID)
}

// ArticleListKey 文章列表缓存键
// key: articles:category:category_name:page:size
func ArticleListKey(category string, page, size int) string {
	return fmt.Sprintf("articles:%s:%d:%d", category, page, size)
}

// ViewCountKey 文章浏览次数计数器
// key: view_count:article_id
func ViewCountKey(articleID string) string {
	return fmt.Sprintf("view_count:%s", articleID)
}

// ViewCountPattern 所有浏览次数计数器的 SCAN 模式
const ViewCountPattern = "view_count:*"

// UserKey 用户信息缓存键
func UserKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

// CommentListKey 评论列表缓存键
func CommentListKey(articleID string, page, size int) string {
	return fmt.Sprintf("comments:%s:%d:%d", articleID, page, size)
}

// LikeCountKey 生成点赞计数的 Redis Key
func LikeCountKey(targetType, targetID string) string {
	return fmt.Sprintf("like_count:%s:%s", targetType, targetID)
}

// CollectionCountKey 生成收藏计数的 Redis Key
func CollectionCountKey(articleID string) string {
	return fmt.Sprintf("collection_count:%s", articleID)
}

// NullValue 缓存空值，用于防止缓存穿透
const NullValue = "__NULL__"

// 缓存过期时间常量
const (
	ArticleExpiration     = 24 * 60 * 60 // 文章详情：1 天
	ArticleListExpiration = 60 * 60      // 文章列表：1 小时
	UserExpiration        = 12 * 60 * 60 // 用户信息：12 小时
	CommentListExpiration = 30 * 60      // 评论列表：30 分钟
	EmptyValueExpiration  = 5 * 60       // 空值缓存：5 分钟
	LikeExpiration        = 1 * 60       // 点赞缓存：1 分钟
	ViewCountExpiration   = 2 * 60 * 60  // 浏览次数计数器：2 小时（定期同步到 DB 后清除）
)
