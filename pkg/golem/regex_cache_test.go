package golem

import (
	"testing"
	"time"
)

func TestRegexCacheBasicOperations(t *testing.T) {
	// Create a new regex cache
	cache := NewRegexCache(10, 60) // 10 patterns, 1 minute TTL

	// Test getting a non-existent pattern
	pattern, err := cache.GetCompiledRegex("test.*pattern")
	if err != nil {
		t.Fatalf("Failed to compile regex: %v", err)
	}
	if pattern == nil {
		t.Error("Expected compiled regex, got nil")
	}

	// Test getting the same pattern again (should hit cache)
	pattern2, err := cache.GetCompiledRegex("test.*pattern")
	if err != nil {
		t.Fatalf("Failed to get cached regex: %v", err)
	}
	if pattern2 == nil {
		t.Error("Expected compiled regex, got nil")
	}

	// Verify it's the same instance (cached)
	if pattern != pattern2 {
		t.Error("Expected same regex instance from cache")
	}

	// Check cache stats
	stats := cache.GetCacheStats()
	if stats["patterns"].(int) != 1 {
		t.Errorf("Expected 1 pattern in cache, got %v", stats["patterns"])
	}
	if stats["misses"].(int) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
	if stats["hits"].(map[string]int)["test.*pattern"] != 1 {
		t.Errorf("Expected 1 hit for pattern, got %v", stats["hits"])
	}
}

func TestRegexCacheLRUEviction(t *testing.T) {
	// Create a small cache
	cache := NewRegexCache(2, 60)

	// Add patterns to fill the cache
	_, err := cache.GetCompiledRegex("pattern1")
	if err != nil {
		t.Fatalf("Failed to compile pattern1: %v", err)
	}
	_, err = cache.GetCompiledRegex("pattern2")
	if err != nil {
		t.Fatalf("Failed to compile pattern2: %v", err)
	}

	// Add a third pattern to trigger eviction
	_, err = cache.GetCompiledRegex("pattern3")
	if err != nil {
		t.Fatalf("Failed to compile pattern3: %v", err)
	}

	// Check that cache size is still 2
	stats := cache.GetCacheStats()
	if stats["patterns"].(int) != 2 {
		t.Errorf("Expected 2 patterns in cache after eviction, got %v", stats["patterns"])
	}

	// pattern1 should be evicted (LRU)
	_, found := cache.Patterns["pattern1"]
	if found {
		t.Error("Expected pattern1 to be evicted")
	}

	// pattern2 and pattern3 should still be there
	_, found = cache.Patterns["pattern2"]
	if !found {
		t.Error("Expected pattern2 to still be in cache")
	}
	_, found = cache.Patterns["pattern3"]
	if !found {
		t.Error("Expected pattern3 to still be in cache")
	}
}

func TestRegexCacheTTL(t *testing.T) {
	// Create a cache with very short TTL
	cache := NewRegexCache(10, 1) // 1 second TTL

	// Add a pattern
	_, err := cache.GetCompiledRegex("test.*pattern")
	if err != nil {
		t.Fatalf("Failed to compile regex: %v", err)
	}

	// Verify it's in cache
	stats := cache.GetCacheStats()
	if stats["patterns"].(int) != 1 {
		t.Errorf("Expected 1 pattern in cache, got %v", stats["patterns"])
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Try to get the pattern again - should recompile due to TTL expiry
	_, err = cache.GetCompiledRegex("test.*pattern")
	if err != nil {
		t.Fatalf("Failed to recompile regex after TTL: %v", err)
	}

	// Should still have 1 pattern (recompiled)
	stats = cache.GetCacheStats()
	if stats["patterns"].(int) != 1 {
		t.Errorf("Expected 1 pattern in cache after TTL expiry, got %v", stats["patterns"])
	}
}

func TestRegexCacheInvalidPattern(t *testing.T) {
	cache := NewRegexCache(10, 60)

	// Try to compile an invalid regex
	_, err := cache.GetCompiledRegex("[invalid")
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}

	// Check that it was counted as a miss
	stats := cache.GetCacheStats()
	if stats["misses"].(int) != 1 {
		t.Errorf("Expected 1 miss for invalid pattern, got %v", stats["misses"])
	}
}

func TestGolemRegexCacheIntegration(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, true)

	// Test that regex caches are initialized
	if g.patternRegexCache == nil {
		t.Error("Expected patternRegexCache to be initialized")
	}
	if g.tagProcessingCache == nil {
		t.Error("Expected tagProcessingCache to be initialized")
	}
	if g.normalizationCache == nil {
		t.Error("Expected normalizationCache to be initialized")
	}

	// Test cache stats
	stats := g.GetRegexCacheStats()
	if len(stats) != 3 {
		t.Errorf("Expected 3 cache types in stats, got %d", len(stats))
	}

	// Test clearing caches
	g.ClearRegexCaches()

	// All caches should be empty
	stats = g.GetRegexCacheStats()
	for cacheType, cacheStats := range stats {
		if cacheStats.(map[string]interface{})["patterns"].(int) != 0 {
			t.Errorf("Expected 0 patterns in %s cache after clear, got %v", cacheType, cacheStats.(map[string]interface{})["patterns"])
		}
	}
}

func TestGetCachedRegexHelper(t *testing.T) {
	g := NewForTesting(t, true)

	// Test with different cache types
	patterns := []string{
		`\s+`,
		`<set\s+name=["']([^"']+)["']`,
		`test.*pattern`,
	}

	cacheTypes := []string{"normalization", "tag_processing", "pattern"}

	for i, pattern := range patterns {
		regex := g.getCachedRegex(pattern, cacheTypes[i])
		if regex == nil {
			t.Errorf("Expected compiled regex for pattern %s, got nil", pattern)
		}
	}

	// Test with invalid cache type (should fallback to direct compilation)
	regex := g.getCachedRegex("test.*pattern", "invalid")
	if regex == nil {
		t.Error("Expected fallback compilation to work")
	}
}
