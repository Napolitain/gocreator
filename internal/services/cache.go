package services

import (
	"time"

	"gocreator/internal/interfaces"

	"github.com/patrickmn/go-cache"
)

// CacheService manages in-memory caching with TTL for runtime data
// This is NOT used for filesystem caches (translations, audio, video segments)
// which persist indefinitely with hash-based invalidation.
type CacheService struct {
	cache *cache.Cache
}

// NewCacheService creates a new cache service for in-memory TTL-based caching
// defaultExpiration: default expiration time for cache entries (TTL)
// cleanupInterval: interval for cleaning up expired entries
// Note: This cache is for runtime data only. Filesystem caches never expire.
func NewCacheService(defaultExpiration, cleanupInterval time.Duration) interfaces.CacheService {
	return &CacheService{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// Get retrieves a value from the cache
func (c *CacheService) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// Set stores a value in the cache with default expiration
func (c *CacheService) Set(key string, value interface{}) {
	c.cache.Set(key, value, cache.DefaultExpiration)
}

// Delete removes a value from the cache
func (c *CacheService) Delete(key string) {
	c.cache.Delete(key)
}

// Clear removes all items from the cache
func (c *CacheService) Clear() {
	c.cache.Flush()
}
