package cache

import (
	"context"
	"testing"
	"time"
)

// 获取测试 Redis 客户端
func getTestClient(t *testing.T) *RedisClient {
	t.Helper()
	client, err := NewRedisClient("localhost:6379", "")
	if err != nil {
		t.Fatalf("无法连接到 Redis (请确保 Redis 已启动): %v", err)
	}
	t.Cleanup(func() {
		client.Close()
	})
	return client
}

// 清理测试用的 key
func cleanKeys(t *testing.T, client *RedisClient, keys ...string) {
	t.Helper()
	ctx := context.Background()
	for _, key := range keys {
		client.Del(ctx, key)
	}
}

func TestNewRedisClient(t *testing.T) {
	t.Run("连接成功", func(t *testing.T) {
		client, err := NewRedisClient("localhost:6379", "")
		if err != nil {
			t.Fatalf("连接 Redis 失败: %v", err)
		}
		defer client.Close()

		if client.Client == nil {
			t.Fatal("RedisClient.Client 为 nil")
		}
	})

	t.Run("连接失败_无效地址", func(t *testing.T) {
		_, err := NewRedisClient("localhost:9999", "")
		if err == nil {
			t.Fatal("期望连接失败，但连接成功了")
		}
	})
}

func TestSetAndGet(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()
	key := "test:set_get:basic"
	cleanKeys(t, client, key)

	t.Run("设置并获取字符串", func(t *testing.T) {
		err := client.Set(ctx, key, "hello world", time.Minute)
		if err != nil {
			t.Fatalf("Set 失败: %v", err)
		}

		val, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get 失败: %v", err)
		}
		if val != "hello world" {
			t.Errorf("Get = %q, want %q", val, "hello world")
		}
	})

	t.Run("获取不存在的键", func(t *testing.T) {
		_, err := client.Get(ctx, "test:nonexistent:key_xyz_123")
		if err == nil {
			t.Fatal("期望返回错误(redis.Nil)，但成功了")
		}
	})

	t.Run("键过期", func(t *testing.T) {
		expireKey := "test:set_get:expire"
		cleanKeys(t, client, expireKey)

		err := client.Set(ctx, expireKey, "will_expire", time.Second)
		if err != nil {
			t.Fatalf("Set 失败: %v", err)
		}

		time.Sleep(2 * time.Second)

		_, err = client.Get(ctx, expireKey)
		if err == nil {
			t.Fatal("期望键已过期返回错误，但获取成功")
		}
	})

	t.Run("设置空值", func(t *testing.T) {
		emptyKey := "test:set_get:empty"
		cleanKeys(t, client, emptyKey)

		err := client.Set(ctx, emptyKey, "", time.Minute)
		if err != nil {
			t.Fatalf("Set 空值失败: %v", err)
		}

		val, err := client.Get(ctx, emptyKey)
		if err != nil {
			t.Fatalf("Get 空值失败: %v", err)
		}
		if val != "" {
			t.Errorf("Get = %q, want %q", val, "")
		}
	})

	t.Run("设置整数", func(t *testing.T) {
		intKey := "test:set_get:int"
		cleanKeys(t, client, intKey)

		err := client.Set(ctx, intKey, 42, time.Minute)
		if err != nil {
			t.Fatalf("Set 整数失败: %v", err)
		}

		val, err := client.Get(ctx, intKey)
		if err != nil {
			t.Fatalf("Get 整数失败: %v", err)
		}
		if val != "42" {
			t.Errorf("Get = %q, want %q", val, "42")
		}
	})
}

func TestDel(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	t.Run("删除存在的键", func(t *testing.T) {
		key := "test:del:exists"
		cleanKeys(t, client, key)

		client.Set(ctx, key, "value", time.Minute)

		err := client.Del(ctx, key)
		if err != nil {
			t.Fatalf("Del 失败: %v", err)
		}

		_, err = client.Get(ctx, key)
		if err == nil {
			t.Fatal("键应已被删除，但仍能获取到值")
		}
	})

	t.Run("删除不存在的键_不报错", func(t *testing.T) {
		err := client.Del(ctx, "test:del:nonexistent")
		if err != nil {
			t.Fatalf("Del 不存在键不应报错: %v", err)
		}
	})

	t.Run("批量删除多个键", func(t *testing.T) {
		keys := []string{"test:del:batch1", "test:del:batch2", "test:del:batch3"}
		cleanKeys(t, client, keys...)

		for _, k := range keys {
			client.Set(ctx, k, "v", time.Minute)
		}

		err := client.Del(ctx, keys...)
		if err != nil {
			t.Fatalf("批量 Del 失败: %v", err)
		}

		for _, k := range keys {
			_, err := client.Get(ctx, k)
			if err == nil {
				t.Errorf("键 %q 应已被删除", k)
			}
		}
	})
}

func TestExists(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	t.Run("存在的键", func(t *testing.T) {
		key := "test:exists:true"
		cleanKeys(t, client, key)

		client.Set(ctx, key, "v", time.Minute)

		ok, err := client.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists 失败: %v", err)
		}
		if !ok {
			t.Error("键应该存在")
		}
	})

	t.Run("不存在的键", func(t *testing.T) {
		ok, err := client.Exists(ctx, "test:exists:false_nonexistent")
		if err != nil {
			t.Fatalf("Exists 失败: %v", err)
		}
		if ok {
			t.Error("键不应该存在")
		}
	})
}

func TestIncr(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	t.Run("递增不存在键_从0开始", func(t *testing.T) {
		key := "test:incr:new"
		cleanKeys(t, client, key)

		val, err := client.Incr(ctx, key)
		if err != nil {
			t.Fatalf("Incr 失败: %v", err)
		}
		if val != 1 {
			t.Errorf("Incr = %d, want 1", val)
		}
	})

	t.Run("多次递增", func(t *testing.T) {
		key := "test:incr:multi"
		cleanKeys(t, client, key)

		for i := 1; i <= 5; i++ {
			val, err := client.Incr(ctx, key)
			if err != nil {
				t.Fatalf("Incr 第 %d 次失败: %v", i, err)
			}
			if val != int64(i) {
				t.Errorf("第 %d 次 Incr = %d, want %d", i, val, i)
			}
		}
	})
}

func TestIncrBy(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	t.Run("递增指定值", func(t *testing.T) {
		key := "test:incrby:basic"
		cleanKeys(t, client, key)

		val, err := client.IncrBy(ctx, key, 10)
		if err != nil {
			t.Fatalf("IncrBy 失败: %v", err)
		}
		if val != 10 {
			t.Errorf("IncrBy = %d, want 10", val)
		}

		val, err = client.IncrBy(ctx, key, -3)
		if err != nil {
			t.Fatalf("IncrBy 负数失败: %v", err)
		}
		if val != 7 {
			t.Errorf("IncrBy -3 = %d, want 7", val)
		}
	})
}

func TestExpire(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()

	t.Run("设置过期时间", func(t *testing.T) {
		key := "test:expire:basic"
		cleanKeys(t, client, key)

		client.Set(ctx, key, "v", 0) // 先设置永不过期
		err := client.Expire(ctx, key, time.Second)
		if err != nil {
			t.Fatalf("Expire 失败: %v", err)
		}

		time.Sleep(2 * time.Second)

		_, err = client.Get(ctx, key)
		if err == nil {
			t.Fatal("键应已过期")
		}
	})
}

func TestHashOperations(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()
	key := "test:hash:full"
	cleanKeys(t, client, key)

	t.Run("HSet和HGet", func(t *testing.T) {
		err := client.HSet(ctx, key, "name", "Alice", "age", "30")
		if err != nil {
			t.Fatalf("HSet 失败: %v", err)
		}

		name, err := client.HGet(ctx, key, "name")
		if err != nil {
			t.Fatalf("HGet name 失败: %v", err)
		}
		if name != "Alice" {
			t.Errorf("HGet name = %q, want %q", name, "Alice")
		}

		age, err := client.HGet(ctx, key, "age")
		if err != nil {
			t.Fatalf("HGet age 失败: %v", err)
		}
		if age != "30" {
			t.Errorf("HGet age = %q, want %q", age, "30")
		}
	})

	t.Run("HGetAll", func(t *testing.T) {
		all, err := client.HGetAll(ctx, key)
		if err != nil {
			t.Fatalf("HGetAll 失败: %v", err)
		}
		if len(all) != 2 {
			t.Errorf("HGetAll 字段数 = %d, want 2", len(all))
		}
		if all["name"] != "Alice" || all["age"] != "30" {
			t.Errorf("HGetAll 内容不正确: %v", all)
		}
	})

	t.Run("HDel", func(t *testing.T) {
		err := client.HDel(ctx, key, "age")
		if err != nil {
			t.Fatalf("HDel 失败: %v", err)
		}

		_, err = client.HGet(ctx, key, "age")
		if err == nil {
			t.Fatal("age 字段应已被删除")
		}

		name, err := client.HGet(ctx, key, "name")
		if err != nil {
			t.Fatalf("HGet name 在 HDel 后失败: %v", err)
		}
		if name != "Alice" {
			t.Errorf("name 不应被删除，应该为 %q", name)
		}
	})

	t.Run("HGet不存在的字段", func(t *testing.T) {
		_, err := client.HGet(ctx, key, "nonexistent_field")
		if err == nil {
			t.Fatal("HGet 不存在字段应返回错误")
		}
	})
}

func TestRedisClient_Comprehensive(t *testing.T) {
	client := getTestClient(t)
	ctx := context.Background()
	prefix := "test:comprehensive:"

	t.Run("完整读写删除流程", func(t *testing.T) {
		keys := []string{prefix + "k1", prefix + "k2"}
		cleanKeys(t, client, keys...)

		// SET
		for i, k := range keys {
			if err := client.Set(ctx, k, i, 10*time.Minute); err != nil {
				t.Fatalf("Set %q 失败: %v", k, err)
			}
		}

		// EXISTS
		for _, k := range keys {
			ok, err := client.Exists(ctx, k)
			if err != nil || !ok {
				t.Errorf("Exists %q 应返回 true", k)
			}
		}

		// GET + DEL
		for _, k := range keys {
			_, err := client.Get(ctx, k)
			if err != nil {
				t.Errorf("Get %q 失败: %v", k, err)
			}
			client.Del(ctx, k)
			ok, _ := client.Exists(ctx, k)
			if ok {
				t.Errorf("Del %q 后应不存在", k)
			}
		}
	})

	t.Run("并发安全_简单验证", func(t *testing.T) {
		key := prefix + "concurrent"
		cleanKeys(t, client, key)

		client.Set(ctx, key, 0, time.Minute)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				client.Incr(ctx, key)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		val, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("并发测试 Get 失败: %v", err)
		}
		if val != "10" {
			t.Errorf("10 次并发 Incr 后值应为 10，实际: %s", val)
		}
	})
}

func TestClose(t *testing.T) {
	client := getTestClient(t)
	err := client.Close()
	if err != nil {
		t.Errorf("Close 失败: %v", err)
	}
}
