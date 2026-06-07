package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient 创建 Redis 客户端
// addr 格式：localhost:6379
func NewRedisClient(addr, pwd string) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           0,
		Password:     pwd,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})

	//测试连接
	/*
			创建上下文（5秒倒计时开始）
		    ↓
			将 ctx 传给 Redis 操作
		    ↓
			情况1：操作在5秒内完成 → 正常返回 → defer cancel() 执行
		    ↓
			情况2：操作超过5秒 → ctx 自动取消 → Redis 操作立即返回错误（DeadlineExceeded）
	*/

	//context.Background()空的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保在函数结束时取消上下文

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{Client: client}, nil

}

// Set 设置键值，带过期时间
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Del 删除键
func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	val, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

// Incr 原子递增
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

// IncrBy 加上指定数值
func (r *RedisClient) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	return r.Client.IncrBy(ctx, key, increment).Result()
}

// Expire 设置过期时间
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// HSet Hash 表设置字段
func (r *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.Client.HSet(ctx, key, values...).Err()
}

// HGet Hash 表获取字段
func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return r.Client.HGet(ctx, key, field).Result()
}

// HGetAll Hash 表获取所有字段
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Client.HGetAll(ctx, key).Result()
}

// HDel Hash 表删除字段
func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	return r.Client.HDel(ctx, key, fields...).Err()
}

// GetInt64 获取 int64 值（不存在时返回 0）
func (r *RedisClient) GetInt64(ctx context.Context, key string) (int64, error) {
	val, err := r.Client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// ScanKeys 使用 SCAN 命令遍历匹配的键，返回所有匹配的键名
func (r *RedisClient) ScanKeys(ctx context.Context, pattern string, count int64) ([]string, error) {
	var keys []string
	iter := r.Client.Scan(ctx, 0, pattern, count).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

// Close 关闭连接
func (r *RedisClient) Close() error {
	return r.Client.Close()
}
