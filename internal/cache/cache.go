package cache

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a single cache entry
type CacheEntry struct {
	Key        string
	Value      interface{}
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AccessedAt time.Time
	AccessCount int64
	element    *list.Element // For LRU implementation
}

// IsExpired returns true if the entry has expired
func (e *CacheEntry) IsExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

// Cache provides an in-memory cache with LRU eviction and TTL support
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	lruList    *list.List
	maxSize    int
	defaultTTL time.Duration
	stats      Stats
	onEvict    func(key string, value interface{})
}

// Stats tracks cache performance metrics
type Stats struct {
	Hits          int64
	Misses        int64
	Evictions     int64
	Expirations   int64
	Sets          int64
	Deletes       int64
	Size          int64
	MaxSize       int64
	HitRatio      float64
	MemoryUsage   int64
}

// Config holds cache configuration options
type Config struct {
	MaxSize    int
	DefaultTTL time.Duration
	OnEvict    func(key string, value interface{})
}

// DefaultConfig returns sensible defaults for cache configuration
func DefaultConfig() Config {
	return Config{
		MaxSize:    1000,
		DefaultTTL: 1 * time.Hour,
		OnEvict:    nil,
	}
}

// NewCache creates a new cache with the given configuration
func NewCache(config Config) *Cache {
	if config.MaxSize <= 0 {
		config.MaxSize = 1000
	}
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = 1 * time.Hour
	}

	return &Cache{
		entries:    make(map[string]*CacheEntry),
		lruList:    list.New(),
		maxSize:    config.MaxSize,
		defaultTTL: config.DefaultTTL,
		onEvict:    config.OnEvict,
		stats:      Stats{MaxSize: int64(config.MaxSize)},
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		c.updateHitRatio()
		return nil, false
	}

	// Check if expired
	if entry.IsExpired() {
		c.removeEntryLocked(key, entry)
		c.stats.Misses++
		c.stats.Expirations++
		c.updateHitRatio()
		return nil, false
	}

	// Update access information
	entry.AccessedAt = time.Now()
	entry.AccessCount++

	// Move to front of LRU list
	c.lruList.MoveToFront(entry.element)

	c.stats.Hits++
	c.updateHitRatio()
	return entry.Value, true
}

// Set stores a value in the cache with default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = now.Add(ttl)
	}

	// Check if key already exists
	if existingEntry, exists := c.entries[key]; exists {
		// Update existing entry
		existingEntry.Value = value
		existingEntry.ExpiresAt = expiresAt
		existingEntry.AccessedAt = now
		existingEntry.AccessCount++
		
		// Move to front
		c.lruList.MoveToFront(existingEntry.element)
		return
	}

	// Create new entry
	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		AccessedAt:  now,
		AccessCount: 1,
	}

	// Add to LRU list
	entry.element = c.lruList.PushFront(entry)
	c.entries[key] = entry

	c.stats.Sets++
	c.stats.Size = int64(len(c.entries))

	// Check if we need to evict
	c.evictIfNeeded()
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return false
	}

	c.removeEntryLocked(key, entry)
	c.stats.Deletes++
	return true
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		if c.onEvict != nil {
			c.onEvict(key, entry.Value)
		}
	}

	c.entries = make(map[string]*CacheEntry)
	c.lruList = list.New()
	c.stats.Size = 0
}

// Size returns the current number of entries in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns current cache statistics
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.stats
	stats.Size = int64(len(c.entries))
	return stats
}

// Keys returns all keys currently in the cache
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		if !c.entries[key].IsExpired() {
			keys = append(keys, key)
		}
	}
	return keys
}

// ExpireExpiredEntries removes all expired entries from the cache
func (c *Cache) ExpireExpiredEntries() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	expired := make([]string, 0)
	for key, entry := range c.entries {
		if entry.IsExpired() {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		entry := c.entries[key]
		c.removeEntryLocked(key, entry)
		c.stats.Expirations++
	}

	return len(expired)
}

// StartCleanupTimer starts a background goroutine that periodically cleans expired entries
func (c *Cache) StartCleanupTimer(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.ExpireExpiredEntries()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// GetOrSet retrieves a value or sets it if not found (atomic operation)
func (c *Cache) GetOrSet(key string, provider func() (interface{}, error)) (interface{}, error) {
	// First try to get
	if value, exists := c.Get(key); exists {
		return value, nil
	}

	// Generate value
	value, err := provider()
	if err != nil {
		return nil, err
	}

	// Set and return
	c.Set(key, value)
	return value, nil
}

// GetOrSetWithTTL retrieves a value or sets it with custom TTL if not found
func (c *Cache) GetOrSetWithTTL(key string, ttl time.Duration, provider func() (interface{}, error)) (interface{}, error) {
	// First try to get
	if value, exists := c.Get(key); exists {
		return value, nil
	}

	// Generate value
	value, err := provider()
	if err != nil {
		return nil, err
	}

	// Set and return
	c.SetWithTTL(key, value, ttl)
	return value, nil
}

// Peek retrieves a value without updating access time or LRU position
func (c *Cache) Peek(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	return entry.Value, true
}

// Contains checks if a key exists in the cache without affecting access stats
func (c *Cache) Contains(key string) bool {
	_, exists := c.Peek(key)
	return exists
}

// GetEntry returns the full cache entry for a key
func (c *Cache) GetEntry(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	// Return a copy to avoid race conditions
	entryCopy := *entry
	return &entryCopy, true
}

// removeEntryLocked removes an entry (must be called with lock held)
func (c *Cache) removeEntryLocked(key string, entry *CacheEntry) {
	delete(c.entries, key)
	c.lruList.Remove(entry.element)
	c.stats.Size = int64(len(c.entries))

	if c.onEvict != nil {
		c.onEvict(key, entry.Value)
	}
}

// evictIfNeeded evicts least recently used entries if cache is over capacity
func (c *Cache) evictIfNeeded() {
	for len(c.entries) > c.maxSize {
		// Remove least recently used (back of list)
		oldest := c.lruList.Back()
		if oldest == nil {
			break
		}

		entry := oldest.Value.(*CacheEntry)
		c.removeEntryLocked(entry.Key, entry)
		c.stats.Evictions++
	}
}

// updateHitRatio calculates the current hit ratio
func (c *Cache) updateHitRatio() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRatio = float64(c.stats.Hits) / float64(total)
	}
}

// MultiCache manages multiple named caches
type MultiCache struct {
	caches map[string]*Cache
	mu     sync.RWMutex
	config Config
}

// NewMultiCache creates a new multi-cache manager
func NewMultiCache(config Config) *MultiCache {
	return &MultiCache{
		caches: make(map[string]*Cache),
		config: config,
	}
}

// GetCache returns a named cache, creating it if it doesn't exist
func (mc *MultiCache) GetCache(name string) *Cache {
	mc.mu.RLock()
	cache, exists := mc.caches[name]
	mc.mu.RUnlock()

	if exists {
		return cache
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Double-check after acquiring write lock
	if cache, exists := mc.caches[name]; exists {
		return cache
	}

	// Create new cache
	cache = NewCache(mc.config)
	mc.caches[name] = cache
	return cache
}

// ClearAll clears all caches
func (mc *MultiCache) ClearAll() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for _, cache := range mc.caches {
		cache.Clear()
	}
}

// GetStats returns statistics for all caches
func (mc *MultiCache) GetStats() map[string]Stats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := make(map[string]Stats)
	for name, cache := range mc.caches {
		stats[name] = cache.Stats()
	}
	return stats
}

// CacheMiddleware provides a caching function wrapper
func CacheMiddleware(cache *Cache, keyFunc func(args ...interface{}) string) func(func(args ...interface{}) (interface{}, error)) func(args ...interface{}) (interface{}, error) {
	return func(fn func(args ...interface{}) (interface{}, error)) func(args ...interface{}) (interface{}, error) {
		return func(args ...interface{}) (interface{}, error) {
			key := keyFunc(args...)
			
			return cache.GetOrSet(key, func() (interface{}, error) {
				return fn(args...)
			})
		}
	}
}

// Hash provides a simple string hash function for cache keys
func Hash(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	
	result := parts[0]
	for _, part := range parts[1:] {
		result = fmt.Sprintf("%s:%s", result, part)
	}
	return result
}