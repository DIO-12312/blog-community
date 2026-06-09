package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestSingleFlight_Coalescing 验证并发请求合并：100个并发请求只执行1次Do
func TestSingleFlight_Coalescing(t *testing.T) {
	g := &Group{GroupMap: make(map[string]*Call)}
	var callCount atomic.Int32

	Do := func(key string) (interface{}, error) {
		callCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return "result", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := g.GetSingleFlight("same-key", Do)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if val.(string) != "result" {
				t.Errorf("unexpected value: %v", val)
			}
		}()
	}
	wg.Wait()

	if callCount.Load() != 1 {
		t.Errorf("expected 1 Do call, got %d — 并发请求未合并，出现击穿！", callCount.Load())
	}
}

// TestSingleFlight_DifferentKeys 不同key独立执行，互不干扰
func TestSingleFlight_DifferentKeys(t *testing.T) {
	g := &Group{GroupMap: make(map[string]*Call)}
	var callCount atomic.Int32

	Do := func(key string) (interface{}, error) {
		callCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return key + "-result", nil
	}

	var wg sync.WaitGroup
	keys := []string{"a", "b", "c", "d", "e"}
	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			val, err := g.GetSingleFlight(k, Do)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			expected := k + "-result"
			if val.(string) != expected {
				t.Errorf("key=%s, expected %s, got %v", k, expected, val)
			}
		}(key)
	}
	wg.Wait()

	if callCount.Load() != 5 {
		t.Errorf("expected 5 Do calls, got %d", callCount.Load())
	}
}

// TestSingleFlight_ErrorPropagation 验证 Do 返回的错误能正确传播给所有等待者
func TestSingleFlight_ErrorPropagation(t *testing.T) {
	g := &Group{GroupMap: make(map[string]*Call)}
	var callCount atomic.Int32
	expectedErr := errors.New("db connection refused")

	Do := func(key string) (interface{}, error) {
		callCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return nil, expectedErr
	}

	var wg sync.WaitGroup
	errCount := atomic.Int32{}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := g.GetSingleFlight("same-key", Do)
			if err == nil {
				errCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if callCount.Load() != 1 {
		t.Errorf("expected 1 Do call, got %d", callCount.Load())
	}
	if errCount.Load() > 0 {
		t.Errorf("有 %d 个请求没收到错误，等待者丢失了 Do 返回的错误！", errCount.Load())
	}
}

// TestSingleFlight_SequentialReuse 验证执行完成后，新请求会重新执行Do（不复用过期结果）
func TestSingleFlight_SequentialReuse(t *testing.T) {
	g := &Group{GroupMap: make(map[string]*Call)}
	var callCount atomic.Int32

	Do := func(key string) (interface{}, error) {
		callCount.Add(1)
		return callCount.Load(), nil
	}

	// 第一批
	val1, _ := g.GetSingleFlight("key", Do)
	// 第二批（第一批已完成并删除了key）
	val2, _ := g.GetSingleFlight("key", Do)
	// 第三批
	val3, _ := g.GetSingleFlight("key", Do)

	if callCount.Load() != 3 {
		t.Errorf("expected 3 Do calls, got %d — 完成后未清理或后续请求复用了过期结果", callCount.Load())
	}
	if val1.(int32) != 1 || val2.(int32) != 2 || val3.(int32) != 3 {
		t.Errorf("顺序值错误: %v %v %v", val1, val2, val3)
	}
}

// TestSingleFlight_ConcurrentKeys 验证多key并发场景：每个key合并，但不同key独立
func TestSingleFlight_ConcurrentKeys(t *testing.T) {
	g := &Group{GroupMap: make(map[string]*Call)}
	var callCount atomic.Int32

	Do := func(key string) (interface{}, error) {
		callCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return key, nil
	}

	var wg sync.WaitGroup
	// 3个key，每个key 100个并发
	for _, key := range []string{"x", "y", "z"} {
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				val, _ := g.GetSingleFlight(k, Do)
				if val.(string) != k {
					t.Errorf("key=%s, got %v", k, val)
				}
			}(key)
		}
	}
	wg.Wait()

	if callCount.Load() != 3 {
		t.Errorf("expected 3 Do calls (3 keys), got %d", callCount.Load())
	}
}

// TestSingleFlight_RaceDetection 竞态安全测试（配合 -race 运行）
func TestSingleFlight_RaceDetection(t *testing.T) {
	g := &Group{GroupMap: make(map[string]*Call)}

	Do := func(key string) (interface{}, error) {
		time.Sleep(time.Millisecond)
		return "ok", nil
	}

	var wg sync.WaitGroup
	// 混合读写：有等着的，有新建的，有不同key的
	for i := 0; i < 200; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			g.GetSingleFlight("hot-key", Do)
		}()
		go func() {
			defer wg.Done()
			g.GetSingleFlight("cold-key", Do)
		}()
		go func(n int) {
			defer wg.Done()
			g.GetSingleFlight("hot-key", Do)
		}(i)
	}
	wg.Wait()
}
