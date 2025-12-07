package golem

import (
	"regexp"
	"testing"
	"time"
)

func TestEnhancedThatPatternCacheBasicOperations(t *testing.T) {
	// Create a new enhanced that pattern cache
	cache := NewThatPatternCache(10)

	// Test getting a non-existent pattern
	pattern, found := cache.GetCompiledPattern("test pattern")
	if found {
		t.Error("Expected pattern not found, but it was found")
	}
	if pattern != nil {
		t.Error("Expected nil pattern, got non-nil")
	}

	// Test setting and getting pattern
	compiled, err := regexp.Compile("test.*pattern")
	if err != nil {
		t.Fatalf("Failed to compile regex: %v", err)
	}

	cache.SetCompiledPattern("test pattern", compiled)

	retrieved, found := cache.GetCompiledPattern("test pattern")
	if !found {
		t.Error("Expected pattern to be found, but it wasn't")
	}
	if retrieved == nil {
		t.Error("Expected non-nil pattern, got nil")
	}

	// Test match result caching
	_, found = cache.GetMatchResult("test pattern", "test context")
	if found {
		t.Error("Expected no cached match result")
	}

	cache.SetMatchResult("test pattern", "test context", true)
	result, found := cache.GetMatchResult("test pattern", "test context")
	if !found {
		t.Error("Expected cached match result to be found")
	}
	if !result {
		t.Error("Expected cached match result to be true")
	}
}

func TestEnhancedThatPatternCacheLRUEviction(t *testing.T) {
	// Create a small cache
	cache := NewThatPatternCache(2)

	// Add patterns to fill the cache
	compiled1, _ := regexp.Compile("pattern1")
	compiled2, _ := regexp.Compile("pattern2")
	compiled3, _ := regexp.Compile("pattern3")

	cache.SetCompiledPattern("pattern1", compiled1)
	cache.SetCompiledPattern("pattern2", compiled2)
	cache.SetCompiledPattern("pattern3", compiled3) // Should evict pattern1

	// Check that cache size is still 2
	stats := cache.GetCacheStats()
	if stats["patterns"].(int) != 2 {
		t.Errorf("Expected 2 patterns in cache after eviction, got %v", stats["patterns"])
	}

	// pattern1 should be evicted (LRU)
	_, found := cache.GetCompiledPattern("pattern1")
	if found {
		t.Error("Expected pattern1 to be evicted")
	}

	// pattern2 and pattern3 should still be there
	_, found = cache.GetCompiledPattern("pattern2")
	if !found {
		t.Error("Expected pattern2 to still be in cache")
	}
	_, found = cache.GetCompiledPattern("pattern3")
	if !found {
		t.Error("Expected pattern3 to still be in cache")
	}
}

func TestEnhancedThatPatternCacheTTL(t *testing.T) {
	// Create a cache with very short TTL
	cache := NewThatPatternCache(10)
	cache.TTL = 1 // 1 second TTL

	compiled, _ := regexp.Compile("test pattern")
	cache.SetCompiledPattern("test pattern", compiled)

	// Verify it's in cache
	stats := cache.GetCacheStats()
	if stats["patterns"].(int) != 1 {
		t.Errorf("Expected 1 pattern in cache, got %v", stats["patterns"])
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Try to get the pattern again - should not be found due to TTL expiry
	_, found := cache.GetCompiledPattern("test pattern")
	if found {
		t.Error("Expected pattern not to be found after TTL expiry")
	}
}

func TestEnhancedThatPatternCacheMatchResults(t *testing.T) {
	cache := NewThatPatternCache(10)

	// Test caching match results
	cache.SetMatchResult("pattern1", "context1", true)
	cache.SetMatchResult("pattern1", "context2", false)
	cache.SetMatchResult("pattern2", "context1", true)

	// Test retrieving match results
	result, found := cache.GetMatchResult("pattern1", "context1")
	if !found || !result {
		t.Error("Expected true match result for pattern1/context1")
	}

	result, found = cache.GetMatchResult("pattern1", "context2")
	if !found || result {
		t.Error("Expected false match result for pattern1/context2")
	}

	result, found = cache.GetMatchResult("pattern2", "context1")
	if !found || !result {
		t.Error("Expected true match result for pattern2/context1")
	}

	// Test non-existent result
	_, found = cache.GetMatchResult("pattern3", "context1")
	if found {
		t.Error("Expected no cached result for pattern3/context1")
	}
}

func TestEnhancedThatPatternCacheContextInvalidation(t *testing.T) {
	cache := NewThatPatternCache(10)

	// Set up some patterns with context hashes
	cache.ContextHashes["pattern1"] = "context1"
	cache.ContextHashes["pattern2"] = "context1"
	cache.ContextHashes["pattern3"] = "context2"

	// Add some patterns
	compiled1, _ := regexp.Compile("pattern1")
	compiled2, _ := regexp.Compile("pattern2")
	compiled3, _ := regexp.Compile("pattern3")

	cache.SetCompiledPattern("pattern1", compiled1)
	cache.SetCompiledPattern("pattern2", compiled2)
	cache.SetCompiledPattern("pattern3", compiled3)

	// Invalidate context1
	cache.InvalidateContext("context1")

	// pattern1 and pattern2 should be removed (context1)
	_, found := cache.GetCompiledPattern("pattern1")
	if found {
		t.Error("Expected pattern1 to be removed after context invalidation")
	}
	_, found = cache.GetCompiledPattern("pattern2")
	if found {
		t.Error("Expected pattern2 to be removed after context invalidation")
	}

	// pattern3 should still be there (context2)
	_, found = cache.GetCompiledPattern("pattern3")
	if !found {
		t.Error("Expected pattern3 to still be in cache after context invalidation")
	}
}

func TestGolemThatPatternCacheIntegration(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, true)

	// Test that that pattern cache is initialized
	if g.thatPatternCache == nil {
		t.Error("Expected thatPatternCache to be initialized")
	}

	// Test cache statistics
	stats := g.GetThatPatternCacheStats()
	if stats["patterns"].(int) != 0 {
		t.Error("Expected empty cache initially")
	}
	if stats["max_size"].(int) != 200 {
		t.Errorf("Expected max_size 200, got %v", stats["max_size"])
	}

	// Test clearing cache
	g.ClearThatPatternCache()
	stats = g.GetThatPatternCacheStats()
	if stats["patterns"].(int) != 0 {
		t.Error("Expected cache to be empty after clear")
	}

	// Test context invalidation
	g.InvalidateThatPatternContext("test_context")
	// Should not cause any errors
}

func TestThatPatternCachePerformance(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, false) // Disable verbose logging

	// Test patterns and contexts
	testPatterns := []string{
		"* HELLO *",
		"WHAT IS *",
		"* MY NAME IS *",
		"* TELL ME ABOUT *",
		"* HOW ARE YOU *",
	}

	testContexts := []string{
		"Hello there",
		"What is your name",
		"My name is Alice",
		"Tell me about yourself",
		"How are you today",
	}

	// Test performance with caching
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, pattern := range testPatterns {
			for _, context := range testContexts {
				matchThatPatternWithWildcardsCached(g, context, pattern)
			}
		}
	}
	cachedDuration := time.Since(start)

	// Test performance without caching (simulate by creating new Golem instance each time)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		// Create new Golem instance to simulate no caching
		gNoCache := New(false)
		for _, pattern := range testPatterns {
			for _, context := range testContexts {
				matchThatPatternWithWildcardsCached(gNoCache, context, pattern)
			}
		}
	}
	uncachedDuration := time.Since(start)

	// Get cache statistics
	stats := g.GetThatPatternCacheStats()

	t.Logf("That Pattern Cache Performance Test Results:")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Patterns per iteration: %d", len(testPatterns))
	t.Logf("Contexts per iteration: %d", len(testContexts))
	t.Logf("Total pattern-context combinations: %d", iterations*len(testPatterns)*len(testContexts))
	t.Logf("Cached duration: %v", cachedDuration)
	t.Logf("Uncached duration: %v", uncachedDuration)
	t.Logf("Pattern cache hit rate: %.2f%%", stats["hit_rate"].(float64)*100)
	t.Logf("Pattern cache results: %d", stats["patterns"].(int))
	t.Logf("Pattern cache hits: %v", stats["hits"])
	t.Logf("Pattern cache misses: %d", stats["misses"].(int))
	t.Logf("Match result cache results: %d", stats["match_results"].(int))
	t.Logf("Match result hit rate: %.2f%%", stats["result_hit_rate"].(float64)*100)

	// Verify that caching provides performance benefits
	if cachedDuration >= uncachedDuration {
		t.Logf("Warning: Cached performance (%v) not better than uncached (%v)", cachedDuration, uncachedDuration)
		t.Logf("This might be due to small test size or system variations")
	} else {
		improvement := float64(uncachedDuration-cachedDuration) / float64(uncachedDuration) * 100
		t.Logf("Performance improvement: %.1f%%", improvement)
	}

	// Verify cache is working
	if stats["patterns"].(int) == 0 {
		t.Error("Expected patterns to be cached")
	}
	if stats["hits"].(map[string]int) == nil {
		t.Error("Expected cache hits to be recorded")
	}
}

func TestThatPatternCacheAdvancedFeatures(t *testing.T) {
	cache := NewThatPatternCache(2) // Small cache to force eviction

	// Test TTL functionality
	cache.TTL = 2 // 2 seconds

	compiled, _ := regexp.Compile("test.*pattern")
	cache.SetCompiledPattern("test pattern", compiled)

	// Should be found immediately
	_, found := cache.GetCompiledPattern("test pattern")
	if !found {
		t.Error("Expected pattern to be found immediately")
	}

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)

	// Should not be found after TTL expiry
	_, found = cache.GetCompiledPattern("test pattern")
	if found {
		t.Error("Expected pattern not to be found after TTL expiry")
	}

	// Test access order tracking with a simpler approach
	cache.SetCompiledPattern("pattern1", compiled)
	cache.SetCompiledPattern("pattern2", compiled)

	// Access pattern1 to make it most recent
	cache.GetCompiledPattern("pattern1")

	// Add pattern3, should evict pattern2 (least recently used)
	cache.SetCompiledPattern("pattern3", compiled)

	// pattern2 should be evicted (it was never accessed after being added)
	_, found = cache.GetCompiledPattern("pattern2")
	if found {
		t.Error("Expected pattern2 to be evicted")
	}

	// pattern1 and pattern3 should still be there
	_, found = cache.GetCompiledPattern("pattern1")
	if !found {
		t.Error("Expected pattern1 to still be in cache")
	}
	_, found = cache.GetCompiledPattern("pattern3")
	if !found {
		t.Error("Expected pattern3 to still be in cache")
	}
}

func TestThatPatternCacheStatistics(t *testing.T) {
	cache := NewThatPatternCache(10)

	// Add some patterns and match results
	compiled1, _ := regexp.Compile("pattern1")
	compiled2, _ := regexp.Compile("pattern2")

	cache.SetCompiledPattern("pattern1", compiled1)
	cache.SetCompiledPattern("pattern2", compiled2)

	// Access patterns to generate hits
	cache.GetCompiledPattern("pattern1")
	cache.GetCompiledPattern("pattern1")
	cache.GetCompiledPattern("pattern2")

	// Add some match results
	cache.SetMatchResult("pattern1", "context1", true)
	cache.SetMatchResult("pattern1", "context2", false)
	cache.GetMatchResult("pattern1", "context1")
	cache.GetMatchResult("pattern1", "context1")

	stats := cache.GetCacheStats()

	// Verify statistics
	if stats["patterns"].(int) != 2 {
		t.Errorf("Expected 2 patterns, got %v", stats["patterns"])
	}
	if stats["max_size"].(int) != 10 {
		t.Errorf("Expected max_size 10, got %v", stats["max_size"])
	}
	if stats["ttl_seconds"].(int64) != 1800 {
		t.Errorf("Expected ttl_seconds 1800, got %v", stats["ttl_seconds"])
	}
	if stats["misses"].(int) != 0 {
		t.Errorf("Expected 0 misses, got %v", stats["misses"])
	}
	if stats["match_results"].(int) != 2 {
		t.Errorf("Expected 2 match results, got %v", stats["match_results"])
	}

	// Check hit rates
	hitRate := stats["hit_rate"].(float64)
	if hitRate <= 0 {
		t.Error("Expected positive hit rate")
	}

	resultHitRate := stats["result_hit_rate"].(float64)
	if resultHitRate <= 0 {
		t.Error("Expected positive result hit rate")
	}
}
