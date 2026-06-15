package shared

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"blog-community/shared/cache"
	"blog-community/shared/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// =============================================================================
// 测试工具函数
// =============================================================================

func getTestDB(b *testing.B) *gorm.DB {
	b.Helper()
	dsn := "root:123456@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		b.Fatalf("连接数据库失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	// 自动迁移确保表存在
	db.AutoMigrate(&models.User{}, &models.Article{}, &models.Comment{})
	return db
}

func getTestRedis(b *testing.B) *cache.RedisClient {
	b.Helper()
	client, err := cache.NewRedisClient("localhost:6379", "")
	if err != nil {
		b.Fatalf("连接 Redis 失败: %v", err)
	}
	return client
}

// =============================================================================
// MySQL 连接池 Benchmarks
// =============================================================================

// BenchmarkDBConnectionPool 测试连接池获取与释放
func BenchmarkDBConnectionPool(b *testing.B) {
	db := getTestDB(b)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := sqlDB.Conn(context.Background())
			if err != nil {
				b.Fatalf("获取连接失败: %v", err)
			}
			conn.Close()
		}
	})
}

// =============================================================================
// 数据库 CRUD Benchmarks
// =============================================================================

// BenchmarkDBInsert 测试单条 INSERT 性能
func BenchmarkDBInsert(b *testing.B) {
	db := getTestDB(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := models.User{
			Username:     fmt.Sprintf("bench_user_%d_%d", time.Now().UnixNano(), i),
			Email:        fmt.Sprintf("bench_%d_%d@test.com", time.Now().UnixNano(), i),
			PasswordHash: "$2a$10$placeholderhashvalue",
			Role:         "user",
		}
		db.Create(&user)
	}
	// 清理
	b.StopTimer()
	db.Where("username LIKE ?", "bench_user_%").Delete(&models.User{})
}

// BenchmarkDBBatchInsert 测试批量 INSERT 性能
func BenchmarkDBBatchInsert(b *testing.B) {
	db := getTestDB(b)
	batchSize := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		users := make([]models.User, batchSize)
		ts := time.Now().UnixNano()
		for j := 0; j < batchSize; j++ {
			users[j] = models.User{
				Username:     fmt.Sprintf("bench_batch_%d_%d_%d", ts, i, j),
				Email:        fmt.Sprintf("bench_batch_%d_%d_%d@test.com", ts, i, j),
				PasswordHash: "$2a$10$hash",
				Role:         "user",
			}
		}
		b.StartTimer()

		db.CreateInBatches(users, batchSize)

		b.StopTimer()
		for _, u := range users {
			db.Delete(&u)
		}
		b.StartTimer()
	}
	b.StopTimer()
	db.Where("username LIKE ?", "bench_batch_%").Delete(&models.User{})
}

// BenchmarkDBSelectByPK 测试主键查询（最常用场景）
func BenchmarkDBSelectByPK(b *testing.B) {
	db := getTestDB(b)

	// 插入一条测试数据
	user := models.User{
		Username:     "bench_pk_user",
		Email:        "bench_pk@test.com",
		PasswordHash: "$2a$10$hash",
		Role:         "user",
	}
	db.Create(&user)
	defer db.Delete(&user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var u models.User
		db.First(&u, "id = ?", user.ID)
	}
}

// BenchmarkDBSelectByIndex 测试索引查询
func BenchmarkDBSelectByIndex(b *testing.B) {
	db := getTestDB(b)

	user := models.User{
		Username:     "bench_idx_user",
		Email:        "bench_idx@test.com",
		PasswordHash: "$2a$10$hash",
	}
	db.Create(&user)
	defer db.Delete(&user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var u models.User
		db.Where("username = ?", user.Username).First(&u)
	}
}

// BenchmarkDBUpdate 测试 UPDATE 性能
func BenchmarkDBUpdate(b *testing.B) {
	db := getTestDB(b)

	user := models.User{
		Username:     "bench_update_user",
		Email:        "bench_update@test.com",
		PasswordHash: "$2a$10$hash",
		Bio:          "original",
	}
	db.Create(&user)
	defer db.Delete(&user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Model(&user).Update("bio", fmt.Sprintf("updated_%d", i))
	}
}

// BenchmarkDBDelete 测试 DELETE 性能
func BenchmarkDBDelete(b *testing.B) {
	db := getTestDB(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		user := models.User{
			Username:     fmt.Sprintf("bench_del_%d_%d", time.Now().UnixNano(), i),
			Email:        fmt.Sprintf("bench_del_%d_%d@test.com", time.Now().UnixNano(), i),
			PasswordHash: "$2a$10$hash",
		}
		db.Create(&user)
		b.StartTimer()

		db.Delete(&user)
	}
}

// =============================================================================
// 复杂查询 Benchmarks
// =============================================================================

// BenchmarkDBJoinQuery 测试 JOIN 查询
func BenchmarkDBJoinQuery(b *testing.B) {
	db := getTestDB(b)

	// 准备数据：1个用户 + 10篇文章
	user := models.User{
		Username:     "bench_join_user",
		Email:        "bench_join@test.com",
		PasswordHash: "$2a$10$hash",
	}
	db.Create(&user)
	for i := 0; i < 10; i++ {
		article := models.Article{
			AuthorID: user.ID,
			Title:    fmt.Sprintf("Bench Article %d", i),
			Content:  "Content here",
			Status:   "published",
		}
		db.Create(&article)
	}
	defer func() {
		db.Where("author_id = ?", user.ID).Delete(&models.Article{})
		db.Delete(&user)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var articles []models.Article
		db.Joins("JOIN users ON users.id = articles.author_id").
			Where("users.username = ?", user.Username).
			Find(&articles)
	}
}

// BenchmarkDBPagination 测试分页查询（文章列表场景）
func BenchmarkDBPagination(b *testing.B) {
	db := getTestDB(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var articles []models.Article
		offset := (i % 10) * 20
		db.Model(&models.Article{}).
			Where("status = ?", "published").
			Order("created_at DESC").
			Offset(offset).Limit(20).
			Find(&articles)
	}
}

// BenchmarkDBCountQuery 测试 COUNT 查询（通知未读数场景）
func BenchmarkDBCountQuery(b *testing.B) {
	db := getTestDB(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var count int64
		db.Model(&models.Notification{}).
			Where("user_id = ? AND is_read = ?", "test-user-id", false).
			Count(&count)
	}
}

// =============================================================================
// 并发数据库操作 Benchmarks
// =============================================================================

// BenchmarkDBConcurrentRead 模拟并发读（文章详情场景）
func BenchmarkDBConcurrentRead(b *testing.B) {
	db := getTestDB(b)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(100)

	// 准备测试数据
	articles := make([]models.Article, 100)
	for i := 0; i < 100; i++ {
		articles[i] = models.Article{
			Title:   fmt.Sprintf("Bench Concurrent %d", i),
			Content: "Content",
			Status:  "published",
		}
		db.Create(&articles[i])
	}
	defer func() {
		for _, a := range articles {
			db.Delete(&a)
		}
	}()

	var counter atomic.Int32

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := int(counter.Add(1)) % 100
			var article models.Article
			db.First(&article, "id = ?", articles[idx].ID)
		}
	})
}

// BenchmarkDBConcurrentWrite 模拟并发写入（评论创建场景）
func BenchmarkDBConcurrentWrite(b *testing.B) {
	db := getTestDB(b)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(100)

	var counter atomic.Int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := counter.Add(1)
			comment := models.Comment{
				ArticleID: "bench-article-id",
				UserID:    "bench-user-id",
				Username:  "benchuser",
				Content:   fmt.Sprintf("Bench comment %d", n),
			}
			db.Create(&comment)
			db.Delete(&comment)
		}
	})
}

// =============================================================================
// 缓存+数据库联合 Benchmarks（模拟 FetchOrCache）
// =============================================================================

// BenchmarkCacheAsideRead 模拟 Cache-Aside 读模式
func BenchmarkCacheAsideRead(b *testing.B) {
	db := getTestDB(b)
	redis := getTestRedis(b)
	ctx := context.Background()

	// 准备数据：缓存预热
	user := models.User{
		Username:     "bench_cache_user",
		Email:        "bench_cache@test.com",
		PasswordHash: "$2a$10$hash",
	}
	db.Create(&user)
	redis.Set(ctx, cache.UserKey(user.ID), `{"id":"`+user.ID+`","username":"bench_cache_user"}`, time.Hour)
	defer func() {
		db.Delete(&user)
		redis.Del(ctx, cache.UserKey(user.ID))
	}()

	g := &cache.Group{GroupMap: make(map[string]*cache.Call)}
	userKey := cache.UserKey(user.ID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetSingleFlight(userKey, func(_ string) (interface{}, error) {
			val, err := redis.Get(ctx, userKey)
			if err == nil && val != "" {
				return val, nil
			}
			var u models.User
			db.First(&u, "id = ?", user.ID)
			return u.Username, nil
		})
	}
}

// =============================================================================
// 事务 Benchmarks
// =============================================================================

// BenchmarkDBTransaction 测试事务性能
func BenchmarkDBTransaction(b *testing.B) {
	db := getTestDB(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		uname := fmt.Sprintf("bench_tx_%d_%d", time.Now().UnixNano(), i)
		b.StartTimer()

		db.Transaction(func(tx *gorm.DB) error {
			user := models.User{
				Username:     uname,
				Email:        fmt.Sprintf("%s@test.com", uname),
				PasswordHash: "$2a$10$hash",
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			// 模拟第二条写入
			article := models.Article{
				AuthorID: user.ID,
				Title:    "Tx Article",
				Content:  "Content",
				Status:   "draft",
			}
			if err := tx.Create(&article).Error; err != nil {
				return err
			}
			return nil
		})

		b.StopTimer()
		db.Where("username LIKE ?", "bench_tx_%").Delete(&models.User{})
		db.Where("title = ?", "Tx Article").Delete(&models.Article{})
		b.StartTimer()
	}
}

// =============================================================================
// 综合压力测试：模拟真实业务负载
// =============================================================================

// BenchmarkRealisticWorkload 模拟真实混合负载
// 读文章详情(40%) + 读文章列表(25%) + 写评论(15%) + 浏览计数(10%) + 读用户(10%)
func BenchmarkRealisticWorkload(b *testing.B) {
	db := getTestDB(b)
	redis := getTestRedis(b)
	ctx := context.Background()
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(100)

	// 准备数据
	user := models.User{
		Username:     "bench_realistic_user",
		Email:        "bench_realistic@test.com",
		PasswordHash: "$2a$10$hash",
	}
	db.Create(&user)
	defer db.Delete(&user)

	articles := make([]models.Article, 50)
	for i := 0; i < 50; i++ {
		articles[i] = models.Article{
			AuthorID: user.ID,
			Title:    fmt.Sprintf("Realistic Article %d", i),
			Content:  "Article content for benchmarking realistic workload",
			Status:   "published",
		}
		db.Create(&articles[i])
	}
	defer func() {
		for _, a := range articles {
			db.Delete(&a)
		}
	}()

	// 预热缓存
	for _, a := range articles {
		redis.Set(ctx, cache.ArticleKey(a.ID),
			fmt.Sprintf(`{"id":"%s","title":"%s"}`, a.ID, a.Title),
			time.Hour)
	}
	defer func() {
		for _, a := range articles {
			redis.Del(ctx, cache.ArticleKey(a.ID))
		}
	}()

	g := &cache.Group{GroupMap: make(map[string]*cache.Call)}
	var counter atomic.Int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := counter.Add(1)
			switch n % 20 {
			case 0, 1, 2, 3, 4, 5, 6, 7: // 40% 读文章详情
				idx := int(n) % 50
				key := cache.ArticleKey(articles[idx].ID)
				g.GetSingleFlight(key, func(_ string) (interface{}, error) {
					val, err := redis.Get(ctx, key)
					if err == nil && val != "" {
						return val, nil
					}
					var a models.Article
					db.First(&a, "id = ?", articles[idx].ID)
					return &a, nil
				})

			case 8, 9, 10, 11, 12: // 25% 读文章列表
				var page []models.Article
				db.Where("status = ?", "published").
					Order("created_at DESC").
					Offset(0).Limit(20).
					Find(&page)

			case 13, 14, 15: // 15% 写评论
				comment := models.Comment{
					ArticleID: articles[n%50].ID,
					UserID:    user.ID,
					Username:  user.Username,
					Content:   fmt.Sprintf("Realistic comment %d", n),
				}
				db.Create(&comment)

			case 16, 17: // 10% 浏览计数
				redis.Incr(ctx, cache.ViewCountKey(articles[n%50].ID))

			case 18, 19: // 10% 读用户信息
				var u models.User
				db.First(&u, "id = ?", user.ID)
			}
		}
	})

	// 清理评论
	b.StopTimer()
	db.Where("username = ?", user.Username).Delete(&models.Comment{})
}

// =============================================================================
// 连接池满负荷测试
// =============================================================================

// BenchmarkDBConnectionSaturation 测试连接池满负荷并发
func BenchmarkDBConnectionSaturation(b *testing.B) {
	db := getTestDB(b)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25) // 限制连接池大小
	sqlDB.SetMaxIdleConns(5)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var count int64
			db.Model(&models.User{}).Count(&count)
		}
	})
}

// =============================================================================
// 批量读取 Benchmarks（N+1 场景）
// =============================================================================

// BenchmarkNPlusOneDetection 测试 N+1 查询模式延迟
func BenchmarkNPlusOneDetection(b *testing.B) {
	db := getTestDB(b)

	// 准备：1个作者 + 20篇文章
	user := models.User{
		Username:     "bench_n1_user",
		Email:        "bench_n1@test.com",
		PasswordHash: "$2a$10$hash",
	}
	db.Create(&user)
	articles := make([]models.Article, 20)
	for i := 0; i < 20; i++ {
		articles[i] = models.Article{
			AuthorID: user.ID,
			Title:    fmt.Sprintf("N+1 Article %d", i),
			Content:  "Content",
			Status:   "published",
		}
		db.Create(&articles[i])
	}
	defer func() {
		for _, a := range articles {
			db.Delete(&a)
		}
		db.Delete(&user)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟 N+1：先查文章列表，再逐条查作者
		var list []models.Article
		db.Limit(20).Find(&list)
		for _, a := range list {
			var author models.User
			db.First(&author, "id = ?", a.AuthorID)
		}
	}
}

// =============================================================================
// 内存分配 Benchmarks
// =============================================================================

// BenchmarkUUIDGeneration 测试 UUID 生成性能（每次插入都会触发）
func BenchmarkUUIDGeneration(b *testing.B) {
	m := &models.BaseModel{}
	for i := 0; i < b.N; i++ {
		m.ID = ""
		m.BeforeCreate(nil)
	}
}

// BenchmarkModelSerialization 测试 GORM 模型 JSON 序列化
func BenchmarkModelSerialization(b *testing.B) {
	article := models.Article{
		Title:   "Benchmark Serialization",
		Content: "This is a long article content for serialization benchmark testing...",
		Status:  "published",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 模拟 json.Marshal(article) 的分配开销
		_ = article.Title + article.Content + article.Status
	}
}

// =============================================================================
// 网络往返延迟模拟（用于对比）
// =============================================================================

// BenchmarkRedisPingLatency 测试 Redis Ping 往返延迟
func BenchmarkRedisPingLatency(b *testing.B) {
	redis := getTestRedis(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		redis.Client.Ping(ctx)
	}
}
