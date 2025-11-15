package twitchgo

import (
	"container/list"
	"sync"
	"time"
)

type dedupeCache struct {
	mu        sync.Mutex
	ttl       time.Duration
	entries   map[string]time.Time
	evictList *list.List
}

func newDedupeCache(ttl time.Duration) *dedupeCache {
	return &dedupeCache{
		ttl:       ttl,
		entries:   make(map[string]time.Time),
		evictList: list.New(),
	}
}

func (c *dedupeCache) Exists(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	t, ok := c.entries[id]
	if !ok {
		return false
	}
	if time.Since(t) > c.ttl {
		delete(c.entries, id)
		return false
	}
	return true
}

func (c *dedupeCache) Add(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.entries[id] = now
	c.evictList.PushBack(id)

	for c.evictList.Len() > 2000 {
		front := c.evictList.Front()
		key := front.Value.(string)
		delete(c.entries, key)
		c.evictList.Remove(front)
	}
}
