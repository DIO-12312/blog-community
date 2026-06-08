package cache

import (
	"sync"
	"testing"
)

func TestGetSingleFlight(t *testing.T) {
	g := &Group{
		GroupMap: make(map[string]*call),
	}
	Ans := map[string]string{"alice": "sum"}
	Do := func(key string) (interface{}, error) {
		return Ans[key], nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			ans, _ := g.GetSingleFlight(key, Do)
			if ans.(string) != Ans[key] {
				t.Error("答案错误")
			}
		}("alice")
	}
	wg.Wait()
}
