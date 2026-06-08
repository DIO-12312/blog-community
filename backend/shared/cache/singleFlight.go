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

	if c, ok := g.GroupMap[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		fmt.Println("命中")
		return c.val, nil
	}

	c := &call{}
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
