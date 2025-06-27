package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	config := Config{
		MaxSize:    100,
		DefaultTTL: 30 * time.Minute,
	}

	cache := NewCache(config)
	assert.NotNil(t, cache)
	assert.Equal(t, 100, cache.maxSize)
	assert.Equal(t, 30*time.Minute, cache.defaultTTL)
	assert.Equal(t, 0, cache.Size())
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, 1000, config.MaxSize)
	assert.Equal(t, 1*time.Hour, config.DefaultTTL)
	assert.Nil(t, config.OnEvict)
}

func TestCacheBasicOperations(t *testing.T) {
	cache := NewCache(DefaultConfig())

	// Test Set and Get
	cache.Set("key1", "value1")
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Test non-existent key
	value, exists = cache.Get("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, value)

	// Test Size
	assert.Equal(t, 1, cache.Size())
}

func TestCacheSetWithTTL(t *testing.T) {
	cache := NewCache(DefaultConfig())

	// Set with short TTL
	cache.SetWithTTL("key1", "value1", 50*time.Millisecond)
	
	// Should exist immediately
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)
	
	// Should be expired
	value, exists = cache.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(DefaultConfig())

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	
	assert.Equal(t, 2, cache.Size())

	// Delete existing key
	deleted := cache.Delete("key1")
	assert.True(t, deleted)
	assert.Equal(t, 1, cache.Size())

	// Delete non-existent key
	deleted = cache.Delete("nonexistent")
	assert.False(t, deleted)
	assert.Equal(t, 1, cache.Size())

	// Verify key1 is gone
	_, exists := cache.Get("key1")
	assert.False(t, exists)

	// Verify key2 still exists
	value, exists := cache.Get("key2")
	assert.True(t, exists)
	assert.Equal(t, "value2", value)
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(DefaultConfig())

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	
	assert.Equal(t, 3, cache.Size())

	cache.Clear()
	assert.Equal(t, 0, cache.Size())

	// Verify all keys are gone
	_, exists := cache.Get("key1")
	assert.False(t, exists)
	_, exists = cache.Get("key2")
	assert.False(t, exists)
	_, exists = cache.Get("key3")
	assert.False(t, exists)
}

func TestCacheLRUEviction(t *testing.T) {
	config := DefaultConfig()
	config.MaxSize = 3
	cache := NewCache(config)

	// Fill cache to capacity
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	assert.Equal(t, 3, cache.Size())

	// Add one more item - should evict oldest (key1)
	cache.Set("key4", "value4")
	assert.Equal(t, 3, cache.Size())

	// key1 should be evicted
	_, exists := cache.Get("key1")
	assert.False(t, exists)

	// Other keys should still exist
	_, exists = cache.Get("key2")
	assert.True(t, exists)
	_, exists = cache.Get("key3")
	assert.True(t, exists)
	_, exists = cache.Get("key4")
	assert.True(t, exists)
}

func TestCacheLRUUpdate(t *testing.T) {
	config := DefaultConfig()
	config.MaxSize = 3
	cache := NewCache(config)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Access key1 to move it to front
	cache.Get("key1")

	// Add another item - key2 should be evicted (oldest unused)
	cache.Set("key4", "value4")

	// key2 should be evicted, key1 should still exist
	_, exists := cache.Get("key1")
	assert.True(t, exists)
	_, exists = cache.Get("key2")
	assert.False(t, exists)
	_, exists = cache.Get("key3")
	assert.True(t, exists)
	_, exists = cache.Get("key4")
	assert.True(t, exists)
}

func TestCacheStats(t *testing.T) {
	cache := NewCache(DefaultConfig())

	// Initial stats
	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Sets)

	// Set some values
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	
	stats = cache.Stats()
	assert.Equal(t, int64(2), stats.Sets)
	assert.Equal(t, int64(2), stats.Size)

	// Get hits
	cache.Get("key1")
	cache.Get("key1")
	
	// Get misses
	cache.Get("nonexistent")
	
	stats = cache.Stats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, float64(2)/float64(3), stats.HitRatio)
}

func TestCacheKeys(t *testing.T) {
	cache := NewCache(DefaultConfig())

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.SetWithTTL("key3", "value3", 1*time.Millisecond)
	
	// Wait for key3 to expire
	time.Sleep(5 * time.Millisecond)

	keys := cache.Keys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
	assert.NotContains(t, keys, "key3") // Should be expired
}

func TestCacheExpireExpiredEntries(t *testing.T) {
	cache := NewCache(DefaultConfig())

	cache.Set("key1", "value1")
	cache.SetWithTTL("key2", "value2", 1*time.Millisecond)
	cache.SetWithTTL("key3", "value3", 1*time.Millisecond)
	
	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	expired := cache.ExpireExpiredEntries()
	assert.Equal(t, 2, expired)
	assert.Equal(t, 1, cache.Size())

	// Only key1 should remain
	_, exists := cache.Get("key1")
	assert.True(t, exists)
}

func TestCacheGetOrSet(t *testing.T) {
	cache := NewCache(DefaultConfig())

	// First call should compute value
	called := false
	value, err := cache.GetOrSet("key1", func() (interface{}, error) {
		called = true
		return "computed1", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "computed1", value)
	assert.True(t, called)

	// Second call should return cached value
	called = false
	value, err = cache.GetOrSet("key1", func() (interface{}, error) {
		called = true
		return "computed2", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "computed1", value) // Original cached value
	assert.False(t, called) // Provider should not be called
}

func TestCacheGetOrSetWithError(t *testing.T) {
	cache := NewCache(DefaultConfig())

	expectedErr := errors.New("computation error")
	value, err := cache.GetOrSet("key1", func() (interface{}, error) {
		return nil, expectedErr
	})
	
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, value)

	// Cache should not contain the key
	_, exists := cache.Get("key1")
	assert.False(t, exists)
}

func TestCacheGetOrSetWithTTL(t *testing.T) {
	cache := NewCache(DefaultConfig())

	value, err := cache.GetOrSetWithTTL("key1", 50*time.Millisecond, func() (interface{}, error) {
		return "value1", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "value1", value)

	// Should be cached
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)
	
	// Should be expired
	_, exists = cache.Get("key1")
	assert.False(t, exists)
}

func TestCachePeek(t *testing.T) {
	cache := NewCache(DefaultConfig())

	cache.Set("key1", "value1")
	
	// Get initial stats
	initialStats := cache.Stats()

	// Peek should not affect stats
	value, exists := cache.Peek("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Stats should be unchanged
	stats := cache.Stats()
	assert.Equal(t, initialStats.Hits, stats.Hits)
	assert.Equal(t, initialStats.Misses, stats.Misses)

	// Peek non-existent key
	_, exists = cache.Peek("nonexistent")
	assert.False(t, exists)
}

func TestCacheContains(t *testing.T) {
	cache := NewCache(DefaultConfig())

	cache.Set("key1", "value1")
	
	assert.True(t, cache.Contains("key1"))
	assert.False(t, cache.Contains("nonexistent"))
}

func TestCacheGetEntry(t *testing.T) {
	cache := NewCache(DefaultConfig())

	cache.Set("key1", "value1")
	
	entry, exists := cache.GetEntry("key1")
	require.True(t, exists)
	require.NotNil(t, entry)
	
	assert.Equal(t, "key1", entry.Key)
	assert.Equal(t, "value1", entry.Value)
	assert.Equal(t, int64(1), entry.AccessCount)
	assert.False(t, entry.CreatedAt.IsZero())
	assert.False(t, entry.AccessedAt.IsZero())
}

func TestCacheOnEvict(t *testing.T) {
	evicted := make(map[string]interface{})
	
	config := DefaultConfig()
	config.MaxSize = 2
	config.OnEvict = func(key string, value interface{}) {
		evicted[key] = value
	}
	
	cache := NewCache(config)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3") // Should evict key1

	assert.Len(t, evicted, 1)
	assert.Equal(t, "value1", evicted["key1"])
}

func TestCacheStartCleanupTimer(t *testing.T) {
	cache := NewCache(DefaultConfig())
	
	cache.SetWithTTL("key1", "value1", 10*time.Millisecond)
	cache.SetWithTTL("key2", "value2", 10*time.Millisecond)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start cleanup timer with short interval
	cache.StartCleanupTimer(ctx, 20*time.Millisecond)
	
	// Wait for cleanup to run
	time.Sleep(50 * time.Millisecond)
	
	// Cache should be empty after cleanup
	assert.Equal(t, 0, cache.Size())
}

func TestMultiCache(t *testing.T) {
	config := DefaultConfig()
	config.MaxSize = 10
	
	mc := NewMultiCache(config)

	// Get caches
	cache1 := mc.GetCache("cache1")
	cache2 := mc.GetCache("cache2")
	
	assert.NotNil(t, cache1)
	assert.NotNil(t, cache2)
	assert.True(t, cache1 != cache2) // Different cache instances

	// Same name should return same cache
	cache1Again := mc.GetCache("cache1")
	assert.True(t, cache1 == cache1Again) // Same cache instance

	// Add data to caches
	cache1.Set("key1", "value1")
	cache2.Set("key2", "value2")

	// Verify data isolation
	_, exists := cache1.Get("key2")
	assert.False(t, exists)
	_, exists = cache2.Get("key1")
	assert.False(t, exists)

	// Test stats
	stats := mc.GetStats()
	assert.Len(t, stats, 2)
	assert.Contains(t, stats, "cache1")
	assert.Contains(t, stats, "cache2")

	// Test clear all
	mc.ClearAll()
	assert.Equal(t, 0, cache1.Size())
	assert.Equal(t, 0, cache2.Size())
}

func TestCacheMiddleware(t *testing.T) {
	cache := NewCache(DefaultConfig())
	
	callCount := 0
	fn := func(args ...interface{}) (interface{}, error) {
		callCount++
		return "result", nil
	}

	keyFunc := func(args ...interface{}) string {
		return "testkey"
	}

	cachedFn := CacheMiddleware(cache, keyFunc)(fn)

	// First call should execute function
	result, err := cachedFn("arg1", "arg2")
	require.NoError(t, err)
	assert.Equal(t, "result", result)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	result, err = cachedFn("arg1", "arg2")
	require.NoError(t, err)
	assert.Equal(t, "result", result)
	assert.Equal(t, 1, callCount) // Function not called again
}

func TestHash(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{"single"}, "single"},
		{[]string{"part1", "part2"}, "part1:part2"},
		{[]string{"a", "b", "c"}, "a:b:c"},
	}

	for _, test := range tests {
		result := Hash(test.input...)
		assert.Equal(t, test.expected, result)
	}
}

func TestCacheEntryExpiration(t *testing.T) {
	now := time.Now()
	
	// Non-expiring entry
	entry1 := &CacheEntry{
		ExpiresAt: time.Time{}, // Zero time means no expiration
	}
	assert.False(t, entry1.IsExpired())

	// Future expiration
	entry2 := &CacheEntry{
		ExpiresAt: now.Add(1 * time.Hour),
	}
	assert.False(t, entry2.IsExpired())

	// Past expiration
	entry3 := &CacheEntry{
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	assert.True(t, entry3.IsExpired())
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache := NewCache(DefaultConfig())
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(i int) {
			cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(i int) {
			cache.Get(fmt.Sprintf("key%d", i))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Cache should have some entries (exact number depends on timing)
	assert.GreaterOrEqual(t, cache.Size(), 0)
	assert.LessOrEqual(t, cache.Size(), 10)
}