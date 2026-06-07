package cache

import (
	"fmt"
	"sync"
)

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu       sync.Mutex
	GroupMap map[string]*call
}

func (g *Group) GetSingleFlight(key string, Do func(string) (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	res, ok := g.GroupMap[key]
	if ok {
		g.mu.Unlock()
		res.wg.Wait()
		fmt.Println("命中")
		return res.val, nil
	}
	g.mu.Unlock()
	val, err := Do(key)
	if err != nil {
		return nil, err
	}

	res.wg.Done()

	g.mu.Lock()
	delete(g.GroupMap, key)
	g.mu.Unlock()

	return val, nil

}
