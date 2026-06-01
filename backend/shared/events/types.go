package events

// 交换机名称
const ExchangeName = "blog_events"

// 事件类型常量
const (
	EventArticlePublished = "article.published"
	EventArticleDeleted   = "article.deleted"
	EventCommentCreated   = "comment.created"
	EventUserFollowed     = "user.followed"
	EventArticleLiked     = "article.liked"
)

// Event 事件结构
type Event struct {
	Type      string                 `json:"type"`      // 事件类型
	Timestamp int64                  `json:"timestamp"` // 时间戳
	Data      map[string]interface{} `json:"data"`      // 事件数据
}
