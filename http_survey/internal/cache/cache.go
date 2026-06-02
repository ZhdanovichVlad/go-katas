package cache

import (
	"hash/crc32"
	"sync"

	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/surveyerr"
)

type Cache struct {
	nodes    []node
	nodesLen int
}

type node struct {
	mu      *sync.RWMutex
	storage map[string]int
}

func NewCache(nodeCount int) (*Cache, error) {

	if nodeCount < 1 {
		return nil, surveyerr.ErrNodeCountCannotLessThenOne
	}
	nodes := make([]node, nodeCount)

	for i := 0; i < nodeCount; i++ {
		nodes[i] = node{
			mu:      &sync.RWMutex{},
			storage: make(map[string]int),
		}
	}

	cache := Cache{
		nodes:    nodes,
		nodesLen: nodeCount,
	}

	return &cache, nil
}

func (c *Cache) getShard(key string) int {
	res := crc32.ChecksumIEEE([]byte(key))

	answer := res % uint32(c.nodesLen)
	return int(answer)
}

func (c *Cache) Set(url string, statusCode int) {
	shard := c.getShard(url)

	c.nodes[shard].mu.Lock()
	defer c.nodes[shard].mu.Unlock()

	c.nodes[shard].storage[url] = statusCode
}

func (c *Cache) Get(url string) (int, bool) {
	shard := c.getShard(url)
	c.nodes[shard].mu.RLock()
	defer c.nodes[shard].mu.RUnlock()
	v, ok := c.nodes[shard].storage[url]

	return v, ok
}
