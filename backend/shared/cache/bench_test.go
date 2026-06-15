package cache

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Redis 基础操作 Benchmarks
// =============================================================================

func getBenchClient(b *testing.B) *RedisClient {
	b.Helper()
	client, err := NewRedisClient("localhost:6379", "")
	if err != nil {
		b.Fatalf("连接 Redis 失败: %v", err)
	}
	b.Cleanup(func() { client.Close() })
	return client
}

// BenchmarkRedisSet 测试 Redis Set 操作吞吐量
func BenchmarkRedisSet(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()
	keys := make([]string, b.N)
	for i := range keys {
		keys[i] = fmt.Sprintf("bench:set:%d:%d", time.Now().UnixNano(), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Set(ctx, keys[i], "benchmark_value_hello_world", time.Minute)
	}
	b.StopTimer()

	// Cleanup
	for _, k := range keys {
		client.Del(ctx, k)
	}
}

// BenchmarkRedisGet 测试 Redis Get 操作吞吐量（命中场景）
func BenchmarkRedisGet(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()
	key := "bench:get:static"
	client.Set(ctx, key, "cached_value", time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Get(ctx, key)
	}
	b.StopTimer()
	client.Del(ctx, key)
}

// BenchmarkRedisGetMiss 测试 Redis Get 操作（未命中场景）
func BenchmarkRedisGetMiss(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Get(ctx, fmt.Sprintf("bench:miss:%d", i))
	}
}

// BenchmarkRedisIncr 测试 Redis 原子递增吞吐量（模拟浏览计数场景）
func BenchmarkRedisIncr(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()
	key := "bench:incr:single"
	client.Set(ctx, key, 0, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Incr(ctx, key)
	}
	b.StopTimer()
	client.Del(ctx, key)
}

// BenchmarkRedisPipelineSet 测试 Redis Pipeline 批量写入
func BenchmarkRedisPipelineSet(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pipe := client.Client.Pipeline()
		keys := make([]string, 100)
		for j := 0; j < 100; j++ {
			keys[j] = fmt.Sprintf("bench:pipe:%d:%d", i, j)
			pipe.Set(ctx, keys[j], "v", time.Minute)
		}
		b.StartTimer()

		_, err := pipe.Exec(ctx)
		if err != nil {
			b.Fatalf("Pipeline exec failed: %v", err)
		}

		b.StopTimer()
		for _, k := range keys {
			client.Del(ctx, k)
		}
		b.StartTimer()
	}
}

// =============================================================================
// Redis 并发操作 Benchmarks（模拟真实并发场景）
// =============================================================================

// BenchmarkRedisConcurrentGet 模拟高并发读缓存
func BenchmarkRedisConcurrentGet(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()
	key := "bench:concurrent:get"
	client.Set(ctx, key, "shared_value", time.Hour)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client.Get(ctx, key)
		}
	})
	b.StopTimer()
	client.Del(ctx, key)
}

// BenchmarkRedisConcurrentIncr 模拟高并发浏览计数（核心场景）
func BenchmarkRedisConcurrentIncr(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()
	key := "bench:concurrent:incr"
	client.Set(ctx, key, 0, time.Hour)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client.Incr(ctx, key)
		}
	})
	b.StopTimer()
	client.Del(ctx, key)
}

// =============================================================================
// SingleFlight Benchmarks
// =============================================================================

// BenchmarkSingleFlightCoalescing 测试 singleflight 请求合并性能
func BenchmarkSingleFlightCoalescing(b *testing.B) {
	g := &Group{GroupMap: make(map[string]*Call)}

	Do := func(key string) (interface{}, error) {
		time.Sleep(time.Microsecond) // 模拟耗时操作
		return key + "-result", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var wg sync.WaitGroup
		b.StartTimer()

		// 每次迭代模拟 50 个并发请求合并为 1 次 Do
		for j := 0; j < 50; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				g.GetSingleFlight("bench-sf-key", Do)
			}()
		}
		wg.Wait()
	}
}

// BenchmarkSingleFlightDifferentKeys 测试不同 key 的并发执行
func BenchmarkSingleFlightDifferentKeys(b *testing.B) {
	g := &Group{GroupMap: make(map[string]*Call)}

	Do := func(key string) (interface{}, error) {
		return key + "-result", nil
	}

	keys := make([]string, 100)
	for i := range keys {
		keys[i] = fmt.Sprintf("bench-sf-key-%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, k := range keys {
			wg.Add(1)
			go func(key string) {
				defer wg.Done()
				g.GetSingleFlight(key, Do)
			}(k)
		}
		wg.Wait()
	}
}

// =============================================================================
// Cache Key Generation Benchmarks
// =============================================================================

func BenchmarkArticleKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ArticleKey("550e8400-e29b-41d4-a716-446655440000")
	}
}

func BenchmarkArticleListKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ArticleListKey("tech", 1, 20)
	}
}

func BenchmarkViewCountKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ViewCountKey("550e8400-e29b-41d4-a716-446655440000")
	}
}

// =============================================================================
// 综合场景 Benchmark：模拟 FetchOrCache 全流程
// =============================================================================

// BenchmarkFetchOrCacheHit Redis 命中场景
func BenchmarkFetchOrCacheHit(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()
	key := "bench:fetch:hit"
	// 预填充缓存
	client.Set(ctx, key, `{"id":"1","title":"Test"}`, time.Hour)

	g := &Group{GroupMap: make(map[string]*Call)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.GetSingleFlight(key, func(_ string) (interface{}, error) {
			val, err := client.Get(ctx, key)
			if err == nil && val != "" {
				return val, nil
			}
			return "db-fallback", nil
		})
	}
	b.StopTimer()
	client.Del(ctx, key)
}

// BenchmarkFetchOrCacheMiss 缓存未命中 → 回源场景
func BenchmarkFetchOrCacheMiss(b *testing.B) {
	g := &Group{GroupMap: make(map[string]*Call)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uniqueKey := fmt.Sprintf("bench:fetch:miss:%d", i)
		g.GetSingleFlight(uniqueKey, func(_ string) (interface{}, error) {
			// 模拟 DB 查询
			return fmt.Sprintf(`{"id":"%d"}`, i), nil
		})
	}
}

// =============================================================================
// 并发混合读写场景
// =============================================================================

// BenchmarkMixedReadWrite 模拟真实场景：80%读 + 20%写
func BenchmarkMixedReadWrite(b *testing.B) {
	client := getBenchClient(b)
	ctx := context.Background()

	// 预填充读缓存
	for i := 0; i < 1000; i++ {
		client.Set(ctx, fmt.Sprintf("bench:mix:read:%d", i), "cached", time.Hour)
	}
	defer func() {
		for i := 0; i < 1000; i++ {
			client.Del(ctx, fmt.Sprintf("bench:mix:read:%d", i))
		}
	}()

	readKeys := make([]string, 1000)
	for i := range readKeys {
		readKeys[i] = fmt.Sprintf("bench:mix:read:%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			if counter%5 == 0 {
				// 20% 写
				_ = client.Set(ctx, fmt.Sprintf("bench:mix:write:%d", counter), "v", time.Minute)
			} else {
				// 80% 读
				_, _ = client.Get(ctx, readKeys[rand.Intn(1000)])
			}
			counter++
		}
	})
}
