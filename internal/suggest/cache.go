package suggest

import (
	"sync"
	"time"
)

type cacheEntry struct {
	resp    Response
	expires time.Time
}

type cache struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]cacheEntry
}

func newCache(ttl time.Duration) *cache {
	return &cache{ttl: ttl, m: map[string]cacheEntry{}}
}

func (c *cache) get(key string) (Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expires) {
		delete(c.m, key)
		return Response{}, false
	}
	return e.resp, true
}

func (c *cache) set(key string, r Response) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) > 500 { // crude bound
		for k := range c.m {
			delete(c.m, k)
		}
	}
	c.m[key] = cacheEntry{resp: r, expires: time.Now().Add(c.ttl)}
}
