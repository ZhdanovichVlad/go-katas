package cache

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ZhdanovichVlad/go-katas/http_survey/internal/surveyerr"
)

func TestNewCacheRejectsInvalidNodeCount(t *testing.T) {
	c, err := NewCache(0)

	if c != nil {
		t.Fatalf("NewCache(0) cache = %v, want nil", c)
	}
	if !errors.Is(err, surveyerr.ErrNodeCountCannotLessThenOne) {
		t.Fatalf("NewCache(0) error = %v, want %v", err, surveyerr.ErrNodeCountCannotLessThenOne)
	}
}

func TestNewCacheCreatesRequestedShards(t *testing.T) {
	const shardCount = 4

	c, err := NewCache(shardCount)
	if err != nil {
		t.Fatalf("NewCache(%d) error = %v", shardCount, err)
	}

	if c.nodesLen != shardCount {
		t.Fatalf("nodesLen = %d, want %d", c.nodesLen, shardCount)
	}
	if len(c.nodes) != shardCount {
		t.Fatalf("len(nodes) = %d, want %d", len(c.nodes), shardCount)
	}

	for i, node := range c.nodes {
		if node.mu == nil {
			t.Fatalf("nodes[%d].mu is nil", i)
		}
		if node.storage == nil {
			t.Fatalf("nodes[%d].storage is nil", i)
		}
	}
}

func TestCacheSetGetAndOverwrite(t *testing.T) {
	c, err := NewCache(3)
	if err != nil {
		t.Fatalf("NewCache error = %v", err)
	}

	if got, ok := c.Get("https://example.test/miss"); ok {
		t.Fatalf("Get(missing) = (%d, true), want (_, false)", got)
	}

	c.Set("https://example.test/a", 200)
	got, ok := c.Get("https://example.test/a")
	if !ok {
		t.Fatal("Get(existing) ok = false, want true")
	}
	if got != 200 {
		t.Fatalf("Get(existing) = %d, want 200", got)
	}

	c.Set("https://example.test/a", 503)
	got, ok = c.Get("https://example.test/a")
	if !ok {
		t.Fatal("Get(overwritten) ok = false, want true")
	}
	if got != 503 {
		t.Fatalf("Get(overwritten) = %d, want 503", got)
	}
}

func TestCacheUsesAllShards(t *testing.T) {
	const shardCount = 4

	c, err := NewCache(shardCount)
	if err != nil {
		t.Fatalf("NewCache error = %v", err)
	}

	seen := make(map[int]struct{})
	for i := 0; len(seen) < shardCount && i < 1000; i++ {
		shard := c.getShard(fmt.Sprintf("https://example.test/%d", i))
		if shard < 0 || shard >= shardCount {
			t.Fatalf("getShard returned %d, want [0, %d)", shard, shardCount)
		}
		seen[shard] = struct{}{}
	}

	if len(seen) != shardCount {
		t.Fatalf("used %d shards, want %d", len(seen), shardCount)
	}
}

func TestCacheConcurrentSetGet(t *testing.T) {
	c, err := NewCache(8)
	if err != nil {
		t.Fatalf("NewCache error = %v", err)
	}

	const entries = 200

	wg := sync.WaitGroup{}
	for i := 0; i < entries; i++ {
		i := i
		wg.Go(func() {
			url := fmt.Sprintf("https://example.test/%d", i)
			c.Set(url, i)

			got, ok := c.Get(url)
			if !ok {
				t.Errorf("Get(%q) ok = false, want true", url)
				return
			}
			if got != i {
				t.Errorf("Get(%q) = %d, want %d", url, got, i)
			}
		})
	}
	wg.Wait()

	for i := 0; i < entries; i++ {
		url := fmt.Sprintf("https://example.test/%d", i)
		got, ok := c.Get(url)
		if !ok {
			t.Fatalf("Get(%q) ok = false after concurrent writes", url)
		}
		if got != i {
			t.Fatalf("Get(%q) = %d, want %d", url, got, i)
		}
	}
}
