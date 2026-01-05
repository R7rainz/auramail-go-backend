package cache

import (
	"sync"
	"time"
)

type CacheItem struct {
	Data interface{}
	ExpiresAt time.Time
}

type MemoryCache struct {
	mu sync.RWMutex
	items map[string]CacheItem
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]CacheItem),
	}
}

func (c *MemoryCache) Set(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheItem{
		Data: value, 
		ExpiresAt: time.Now().Add(duration),
	}
}

//Get retrieves an item only if it hasn't expired
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.RUnlock()

	item,found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	return item.Data,true 

}

func (c *MemoryCache) Delete(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.mu.Lock()
			for key, item := range c.items {
				if time.Now().After(item.ExpiresAt) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		}
	}()
}
