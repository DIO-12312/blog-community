package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"blog-community/shared/cache"
	"blog-community/shared/models"

	"gorm.io/gorm"
)

// ArticleRepository 文章数据访问层
type ArticleRepository struct {
	db    *gorm.DB
	redis *cache.RedisClient
}

// NewArticleRepository 创建文章仓库
func NewArticleRepository(db *gorm.DB, redis *cache.RedisClient) *ArticleRepository {
	return &ArticleRepository{
		db:    db,
		redis: redis,
	}
}

// Create 创建文章
func (r *ArticleRepository) Create(article *models.Article) error {
	return r.db.Create(article).Error
}

// GetByID 根据 ID 获取文章 (带缓存)
func (r *ArticleRepository) GetByID(ctx context.Context, id string) (*models.Article, error) {

	//1.查询缓存
	ArticleCacheKey := cache.ArticleKey(id)
	ArticleValue, err := r.redis.Get(ctx, ArticleCacheKey)
	//1.1 查询到了缓存
	if err == nil && ArticleValue != "" {
		//防止缓存穿透，预设空值
		if ArticleValue == cache.NullValue {
			return nil, errors.New("文章不存在")
		}
		var article models.Article
		if err := json.Unmarshal(([]byte)(ArticleValue), &article); err != nil {
			return &article, err
		}
	}

	// 2.没有查询到缓存，查询数据库
	var article models.Article
	err = r.db.First(&article, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 缓存空值，防止缓存穿透
			r.redis.Set(ctx, ArticleCacheKey, cache.NullValue, cache.EmptyValueExpiration*time.Second)
			return nil, errors.New("文章不存在")
		}
		return nil, err
	}

	// 3.查询到数据，写入缓存
	if articleSQLValue, err := json.Marshal(article); err == nil {
		r.redis.Set(ctx, ArticleCacheKey, articleSQLValue, cache.ArticleExpiration*time.Second)
	}

	return &article, nil
}

// GetByIDUnscoped 获取文章（包括已删除）(带缓存)
func (r *ArticleRepository) GetByIDUnscoped(ctx context.Context, id string) (*models.Article, error) {

	//1.查询缓存
	ArticleCacheKey := cache.ArticleKey(id)
	ArticleValue, err := r.redis.Get(ctx, ArticleCacheKey)
	//1.1 查询到了缓存
	if err == nil && ArticleValue != "" {
		//防止缓存穿透，预设空值
		if ArticleValue != cache.NullValue {
			var article models.Article
			if err := json.Unmarshal(([]byte)(ArticleValue), &article); err != nil {
				return &article, err
			}
		}
	}

	// 2.没有查询到缓存，查询数据库
	var article models.Article
	err = r.db.Unscoped().First(&article, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 缓存空值，防止缓存穿透
			r.redis.Set(ctx, ArticleCacheKey, cache.NullValue, cache.EmptyValueExpiration*time.Second)
			return nil, errors.New("文章不存在")
		}
		return nil, err
	}

	// 3.查询到数据，写入缓存
	if articleSQLValue, err := json.Marshal(article); err == nil {
		r.redis.Set(ctx, ArticleCacheKey, articleSQLValue, cache.ArticleExpiration*time.Second)
	}

	return &article, nil
}

// ListByAuthor 获取某用户的文章列表
func (r *ArticleRepository) ListByAuthor(authorID string, page, size int) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	// 计数
	r.db.Model(&models.Article{}).Where("author_id = ?", authorID).Count(&total)

	// 分页查询
	err := r.db.
		Where("author_id = ?", authorID).
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&articles).Error

	return articles, total, err
}

// ListByCategory 按分类查询
func (r *ArticleRepository) ListByCategory(category string, page, size int) ([]models.Article, int64, error) {
	var articles []models.Article
	var total int64

	query := r.db.Model(&models.Article{}).Where("status = ?", models.StatusPublished)
	if category != "" {
		query = query.Where("category = ?", category)
	}

	query.Count(&total)

	err := query.
		Order("published_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&articles).Error

	return articles, total, err
}

// ListPublished 获取已发布的文章列表
func (r *ArticleRepository) ListPublished(page, size int) ([]models.Article, int64, error) {
	return r.ListByCategory("", page, size)
}

// Update 更新文章
func (r *ArticleRepository) Update(ctx context.Context, article *models.Article) error {
	// 只更新指定字段，避免覆盖其他字段
	err := r.db.Model(article).
		Select("title", "content", "summary", "category", "tags", "status", "updated_at").
		Updates(article).Error

	//删除原有缓存
	if err == nil {
		articleKey := cache.ArticleKey(article.ID)
		r.redis.Del(ctx, articleKey)
	}
	return err
}

// Delete 软删除文章 （带缓存）
func (r *ArticleRepository) Delete(ctx context.Context, id string) error {
	err := r.db.Delete(&models.Article{}, "id = ?", id).Error
	if err == nil {
		articleKey := cache.ArticleKey(id)
		r.redis.Del(ctx, articleKey)
	}
	return err
}

// HardDelete 硬删除文章（仅管理员）（带缓存）
func (r *ArticleRepository) HardDelete(ctx context.Context, id string) error {
	err := r.db.Unscoped().Delete(&models.Article{}, "id = ?", id).Error
	if err == nil {
		articleKey := cache.ArticleKey(id)
		r.redis.Del(ctx, articleKey)
	}
	return err
}

// IncrementViewCount 增加浏览次数（Redis 原子计数，消除 DB 行级锁）
func (r *ArticleRepository) IncrementViewCount(ctx context.Context, id string) error {
	viewKey := cache.ViewCountKey(id)
	if _, err := r.redis.Incr(ctx, viewKey); err != nil {
		return err
	}
	// 设置过期时间防止内存泄漏（定期同步任务会清除）
	r.redis.Expire(ctx, viewKey, cache.ViewCountExpiration*time.Second)
	return nil
}

// GetViewCount 获取文章在 Redis 中的浏览次数增量
func (r *ArticleRepository) GetViewCount(ctx context.Context, id string) (int64, error) {
	return r.redis.GetInt64(ctx, cache.ViewCountKey(id))
}

// SyncViewCounts 将 Redis 中的浏览次数批量同步到 MySQL
// 返回同步的文章数
func (r *ArticleRepository) SyncViewCounts(ctx context.Context) (int, error) {
	keys, err := r.redis.ScanKeys(ctx, cache.ViewCountPattern, 100)
	if err != nil {
		return 0, err
	}

	synced := 0
	for _, key := range keys {
		// key 格式: "view_count:<article_id>"
		articleID := key[len("view_count:"):]
		if articleID == "" {
			continue
		}

		count, err := r.redis.GetInt64(ctx, key)
		if err != nil || count <= 0 {
			continue
		}

		// 批量 +count 到 DB
		err = r.db.Model(&models.Article{}).Where("id = ?", articleID).
			Update("view_count", gorm.Expr("view_count + ?", count)).Error
		if err != nil {
			continue // 跳过失败的文章，下次重试
		}

		// 清除已同步的 Redis 计数
		r.redis.Del(ctx, key)
		// 清除文章缓存（view_count 变了）
		r.redis.Del(ctx, cache.ArticleKey(articleID))
		synced++
	}

	return synced, nil
}

// UpdateStats 更新统计数据（点赞数、评论数）
func (r *ArticleRepository) UpdateStats(ctx context.Context, id string, likeCount, commentCount int64) error {
	err := r.db.Model(&models.Article{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"like_count":    likeCount,
			"comment_count": commentCount,
		}).Error
	if err != nil {
		r.redis.Del(ctx, cache.ArticleKey(id))
	}
	return err
}
