package cache

import (
	"sync"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/config"
	gocache "github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// Cache provides a thread-safe in-memory key-value store with TTL expiration.
// It wraps go-cache and adds structured logging and convenience methods.
type Cache struct {
	store      *gocache.Cache
	defaultTTL time.Duration
	mu         sync.RWMutex
}

// New creates a new Cache instance. Items expire after cfg.TTL and the cleanup
// interval runs at half that duration to keep memory usage under control.
func New(cfg config.CacheConfig) *Cache {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	cleanupInterval := ttl / 2
	if cleanupInterval < 1*time.Minute {
		cleanupInterval = 1 * time.Minute
	}

	store := gocache.New(ttl, cleanupInterval)

	log.Info().
		Dur("default_ttl", ttl).
		Dur("cleanup_interval", cleanupInterval).
		Int("max_items_hint", cfg.MaxItems).
		Msg("cache: initialized in-memory cache")

	return &Cache{
		store:      store,
		defaultTTL: ttl,
	}
}

// Get retrieves a value by key. Returns the value and true if found and not expired,
// or nil and false otherwise.
func (c *Cache) Get(key string) (interface{}, bool) {
	return c.store.Get(key)
}

// GetString is a convenience wrapper that returns the value as a string.
// Returns empty string and false if the key is missing or the value is not a string.
func (c *Cache) GetString(key string) (string, bool) {
	val, found := c.store.Get(key)
	if !found {
		return "", false
	}
	s, ok := val.(string)
	if !ok {
		return "", false
	}
	return s, true
}

// Set stores a value with the default TTL.
func (c *Cache) Set(key string, value interface{}) {
	c.store.Set(key, value, c.defaultTTL)
}

// SetWithTTL stores a value with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.store.Set(key, value, ttl)
}

// Delete removes a single key from the cache.
func (c *Cache) Delete(key string) {
	c.store.Delete(key)
}

// DeleteByPrefix removes all keys that start with the given prefix.
// This is useful for cache invalidation of related entries (e.g. all project cache keys).
func (c *Cache) DeleteByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := c.store.Items()
	count := 0
	for k := range items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			c.store.Delete(k)
			count++
		}
	}

	if count > 0 {
		log.Debug().
			Str("prefix", prefix).
			Int("evicted", count).
			Msg("cache: evicted keys by prefix")
	}
}

// Flush removes all items from the cache.
func (c *Cache) Flush() {
	c.store.Flush()
	log.Debug().Msg("cache: flushed all entries")
}

// ItemCount returns the number of items currently in the cache, including expired
// items that have not yet been cleaned up.
func (c *Cache) ItemCount() int {
	return c.store.ItemCount()
}

// Keys returns a slice of all current cache keys. This is primarily useful for
// debugging and monitoring.
func (c *Cache) Keys() []string {
	items := c.store.Items()
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	return keys
}
