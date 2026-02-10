package cache

import (
	"sync"
	"time"
)

// Item represents a cached item with expiration
type Item[T any] struct {
	Data      T
	ExpiresAt time.Time
}

// Cache is a generic thread-safe in-memory cache with TTL support
type Cache[T any] struct {
	mu       sync.RWMutex
	items    map[string]Item[T]
	stopChan chan struct{}
}

// New creates a new cache instance
func New[T any]() *Cache[T] {
	return &Cache[T]{
		items:    make(map[string]Item[T]),
		stopChan: make(chan struct{}),
	}
}

// NewWithCleanup creates a cache that automatically cleans up expired items
func NewWithCleanup[T any](cleanupInterval time.Duration) *Cache[T] {
	c := New[T]()
	go c.startCleanup(cleanupInterval)
	return c
}

// Set stores a value with the given TTL
func (c *Cache[T]) Set(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = Item[T]{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves an item if it exists and hasn't expired
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		var zero T
		return zero, false
	}

	if time.Now().After(item.ExpiresAt) {
		var zero T
		return zero, false
	}

	return item.Data, true
}

// GetOrSet retrieves an existing item or sets a new one using the provided function
func (c *Cache[T]) GetOrSet(key string, ttl time.Duration, fn func() (T, error)) (T, error) {
	// Try to get existing
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	// Generate new value
	val, err := fn()
	if err != nil {
		var zero T
		return zero, err
	}

	c.Set(key, val, ttl)
	return val, nil
}

// Delete removes an item from the cache
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]Item[T])
}

// Size returns the number of items in the cache
func (c *Cache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Keys returns all keys in the cache
func (c *Cache[T]) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

// startCleanup periodically removes expired items
func (c *Cache[T]) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopChan:
			return
		}
	}
}

// cleanupExpired removes all expired items
func (c *Cache[T]) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
}

// Stop stops the cleanup goroutine
func (c *Cache[T]) Stop() {
	select {
	case <-c.stopChan:
		// Already closed
	default:
		close(c.stopChan)
	}
}

// ==========================================
// Legacy MemoryCache for backward compatibility
// ==========================================

// CacheItem represents a cached item (legacy)
type CacheItem struct {
	Data      interface{}
	ExpiresAt time.Time
}

// MemoryCache is a thread-safe in-memory cache (legacy, use Cache[T] instead)
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]CacheItem
	stopChan chan struct{}
}

// NewMemoryCache creates a new memory cache (legacy)
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items:    make(map[string]CacheItem),
		stopChan: make(chan struct{}),
	}
}

// Set stores a value with the given TTL (legacy)
func (c *MemoryCache) Set(key string, value interface{}, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = CacheItem{
		Data:      value,
		ExpiresAt: time.Now().Add(duration),
	}
}

// Get retrieves an item only if it hasn't expired (legacy)
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	return item.Data, true
}

// Delete starts periodic cleanup of expired items (legacy - renamed for clarity)
func (c *MemoryCache) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.mu.Lock()
				now := time.Now()
				for key, item := range c.items {
					if now.After(item.ExpiresAt) {
						delete(c.items, key)
					}
				}
				c.mu.Unlock()
			case <-c.stopChan:
				return
			}
		}
	}()
}

// Stop stops the cleanup goroutine (legacy)
func (c *MemoryCache) Stop() {
	select {
	case <-c.stopChan:
	default:
		close(c.stopChan)
	}
}
