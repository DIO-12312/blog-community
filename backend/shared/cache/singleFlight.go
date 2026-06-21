package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu       sync.Mutex
	GroupMap map[string]*Call
}

func (g *Group) GetSingleFlight(key string, Do func(string) (interface{}, error)) (interface{}, error) {
	g.mu.Lock()

	if c, ok := g.GroupMap[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		fmt.Println("命中")
		return c.val, c.err
	}

	c := &Call{}
	c.wg.Add(1)
	g.GroupMap[key] = c
	g.mu.Unlock()
	fmt.Println("未命中，执行真实函数")
	c.val, c.err = Do(key)

	c.wg.Done()

	g.mu.Lock()
	delete(g.GroupMap, key)
	g.mu.Unlock()

	return c.val, c.err

}

func FetchOrCache[T any](ctx context.Context,
	redis *RedisClient,
	db *gorm.DB,
	group *Group,
	key string,
	ttl time.Duration,
	fetchDB func() (*T, error)) (*T, error) {
	// 用 singleflight 合并并发请求
	result, err := group.GetSingleFlight(key, func(_ string) (interface{}, error) {
		// 1. 查 Redis（仅当 redis 可用时）
		if redis != nil {
			val, redisErr := redis.Get(ctx, key)
			if redisErr == nil && val != "" {
				if val == NullValue {
					return nil, errors.New("文章不存在")
				}
				var data T
				if err := json.Unmarshal([]byte(val), &data); err == nil {
					return &data, nil
				}
			}
		}

		// 2. 查 DB（调用方传入的闭包）
		data, dbErr := fetchDB()
		if dbErr != nil {
			if errors.Is(dbErr, gorm.ErrRecordNotFound) && redis != nil {
				redis.Set(ctx, key, NullValue, EmptyValueExpiration*time.Second)
			}
			return nil, dbErr
		}

		// 3. 写回 Redis（仅当 redis 可用时）
		if redis != nil {
			if bytes, err := json.Marshal(data); err == nil {
				redis.Set(ctx, key, bytes, ttl)
			}
		}

		return data, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*T), nil // interface{} → *T
}
