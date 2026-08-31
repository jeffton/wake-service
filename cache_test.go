package main

import (
	"testing"
	"time"
)

func TestForecastCacheExpiresAtHourBoundary(t *testing.T) {
	created := time.Date(2026, 8, 31, 16, 59, 30, 0, time.UTC)
	cache := NewForecastCache()
	cache.entries["malmo"] = CacheEntry{
		Created: created,
		Data:    ApiResponseJSON{RequestTime: created.Unix()},
	}

	if _, ok := cache.getAt("malmo", 2*time.Minute, created.Add(29*time.Second)); !ok {
		t.Fatal("cache entry should be valid within its forecast hour")
	}
	if _, ok := cache.getAt("malmo", 2*time.Minute, created.Add(30*time.Second)); ok {
		t.Fatal("cache entry should expire when the forecast hour changes")
	}
}
