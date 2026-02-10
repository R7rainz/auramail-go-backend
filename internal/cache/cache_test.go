package cache

import (
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New[string]()

	c.Set("key1", "value1", time.Hour)

	val, ok := c.Get("key1")
	if !ok {
		t.Error("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got '%s'", val)
	}
}

func TestCache_GetNonExistent(t *testing.T) {
	c := New[string]()

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected to not find nonexistent key")
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New[string]()

	c.Set("key1", "value1", 10*time.Millisecond)

	// Should exist immediately
	_, ok := c.Get("key1")
	if !ok {
		t.Error("expected to find key1 before expiration")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	_, ok = c.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New[string]()

	c.Set("key1", "value1", time.Hour)
	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	c := New[string]()

	c.Set("key1", "value1", time.Hour)
	c.Set("key2", "value2", time.Hour)
	c.Clear()

	if c.Size() != 0 {
		t.Errorf("expected size 0, got %d", c.Size())
	}
}

func TestCache_Size(t *testing.T) {
	c := New[string]()

	if c.Size() != 0 {
		t.Errorf("expected size 0, got %d", c.Size())
	}

	c.Set("key1", "value1", time.Hour)
	c.Set("key2", "value2", time.Hour)

	if c.Size() != 2 {
		t.Errorf("expected size 2, got %d", c.Size())
	}
}

func TestCache_Keys(t *testing.T) {
	c := New[string]()

	c.Set("key1", "value1", time.Hour)
	c.Set("key2", "value2", time.Hour)

	keys := c.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestCache_GetOrSet(t *testing.T) {
	c := New[string]()

	// First call should set the value
	val, err := c.GetOrSet("key1", time.Hour, func() (string, error) {
		return "computed_value", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "computed_value" {
		t.Errorf("expected 'computed_value', got '%s'", val)
	}

	// Second call should return cached value
	callCount := 0
	val, err = c.GetOrSet("key1", time.Hour, func() (string, error) {
		callCount++
		return "new_value", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "computed_value" {
		t.Errorf("expected cached 'computed_value', got '%s'", val)
	}
	if callCount != 0 {
		t.Error("function should not have been called for cached key")
	}
}

func TestCache_Concurrent(t *testing.T) {
	c := New[int]()
	done := make(chan bool)

	// Start multiple goroutines doing reads and writes
	for i := 0; i < 100; i++ {
		go func(n int) {
			c.Set("key", n, time.Hour)
			c.Get("key")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// Legacy MemoryCache tests
func TestMemoryCache_SetAndGet(t *testing.T) {
	c := NewMemoryCache()

	c.Set("key1", "value1", time.Hour)

	val, ok := c.Get("key1")
	if !ok {
		t.Error("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected 'value1', got '%v'", val)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	c := NewMemoryCache()

	c.Set("key1", "value1", 10*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}
