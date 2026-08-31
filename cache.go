package main

import (
	"sync"
	"time"
)

type ForecastCache struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry
}

func NewForecastCache() *ForecastCache {
	return &ForecastCache{entries: make(map[string]CacheEntry)}
}

func (c *ForecastCache) Get(key string, maxAge time.Duration) (ApiResponseJSON, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return ApiResponseJSON{}, false
	}
	if time.Since(entry.Created) > maxAge {
		return ApiResponseJSON{}, false
	}
	return entry.Data, true
}

func (c *ForecastCache) Set(key string, value ApiResponseJSON) {
	c.mu.Lock()
	c.entries[key] = CacheEntry{Created: time.Now(), Data: value}
	c.mu.Unlock()
}
