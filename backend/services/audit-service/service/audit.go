package service

import (
	"encoding/json"
	"log"

	"blog-community/audit-service/repository"
	"blog-community/shared/events"
	"blog-community/shared/models"
)

type AuditService struct {
	repo     *repository.AuditRepository
	consumer *events.Consumer
}

func NewAuditService(repo *repository.AuditRepository, consumer *events.Consumer) *AuditService {
	return &AuditService{repo: repo, consumer: consumer}
}

// StartListening 监听所有事件（# 通配符）
func (s *AuditService) StartListening() {
	s.consumer.Subscribe("audit_queue", "#", func(event events.Event) error {
		detailBytes, _ := json.Marshal(event.Data)
		detailStr := string(detailBytes)

		auditLog := &models.AuditLog{
			UserID:     getStringFromMap(event.Data, "user_id", "follower_id", "userID"),
			Action:     event.Type,
			Resource:   getResourceFromEvent(event.Type),
			ResourceID: getStringFromMap(event.Data, "article_id", "comment_id", "following_id", "target_id"),
			Detail:     detailStr,
		}

		if err := s.repo.Create(auditLog); err != nil {
			log.Printf("审计日志写入失败: %v", err)
			return err
		}

		log.Printf("审计: [%s] %s", event.Type, detailStr)
		return nil
	})
}

// QueryLogs 查询审计日志
func (s *AuditService) QueryLogs(userID, action, resource string, page, size int) ([]models.AuditLog, int64, error) {
	return s.repo.Query(userID, action, resource, page, size)
}

// getStringFromMap 从 map 中获取第一个存在的 key 的值
func getStringFromMap(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if s, ok := val.(string); ok {
				return s
			}
		}
	}
	return ""
}

// getResourceFromEvent 从事件类型推断资源类型
func getResourceFromEvent(eventType string) string {
	for i, c := range eventType {
		if c == '.' {
			return eventType[:i]
		}
	}
	return eventType
}

// 确保 Consumer 可用以编译（审计服务同时作为消费者和 HTTP 服务）
var _ = (*events.Consumer)(nil)
