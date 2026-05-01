package cache

import "sync"

type Cache struct {
	data map[string]int
	mu   sync.RWMutex
}

func NewCache() *Cache {
	cache := Cache{
		data: make(map[string]int),
		mu:   sync.RWMutex{},
	}

	return &cache
}

func (c *Cache) Set(url string, statusCode int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[url] = statusCode
}

func (c *Cache) Get(url string) (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.data[url]

	return v, ok
}
