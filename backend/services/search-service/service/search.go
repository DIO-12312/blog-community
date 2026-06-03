package service

import (
	"log"

	"blog-community/search-service/repository"
	"blog-community/shared/events"
)

type SearchService struct {
	repo     *repository.SearchRepository
	consumer *events.Consumer
}

func NewSearchService(repo *repository.SearchRepository, consumer *events.Consumer) *SearchService {
	return &SearchService{repo: repo, consumer: consumer}
}

// StartListening 监听文章事件，同步到 ES
func (s *SearchService) StartListening() {
	// 监听 article.published: 索引文章
	s.consumer.Subscribe("search_article_published", events.EventArticlePublished, func(event events.Event) error {
		articleID := event.Data["article_id"].(string)
		if err := s.repo.IndexArticle(articleID, event.Data); err != nil {
			log.Printf("索引文章失败 [%s]: %v", articleID, err)
			return err
		}
		log.Printf("文章已索引到 ES: %s", articleID)
		return nil
	})

	// 监听 article.deleted: 删除索引
	s.consumer.Subscribe("search_article_deleted", events.EventArticleDeleted, func(event events.Event) error {
		articleID := event.Data["article_id"].(string)
		if err := s.repo.DeleteArticle(articleID); err != nil {
			log.Printf("删除文章索引失败 [%s]: %v", articleID, err)
			return err
		}
		log.Printf("文章索引已从 ES 删除: %s", articleID)
		return nil
	})
}

// Search 搜索文章
func (s *SearchService) Search(keyword string, page, size int) (*repository.SearchResult, error) {
	return s.repo.Search(keyword, page, size)
}
