package golem

import (
	"testing"
	"time"
)

func TestTextNormalizationCacheBasicOperations(t *testing.T) {
	// Create a new text normalization cache
	cache := NewTextNormalizationCache(10, 60) // 10 results, 1 minute TTL

	// Test getting a non-existent result
	result, err := cache.GetNormalizedText(nil, "hello world", "NormalizePattern")
	if err != nil {
		t.Fatalf("Failed to normalize text: %v", err)
	}
	if result == "" {
		t.Error("Expected normalized result, got empty string")
	}

	// Test getting the same input again (should hit cache)
	result2, err := cache.GetNormalizedText(nil, "hello world", "NormalizePattern")
	if err != nil {
		t.Fatalf("Failed to get cached result: %v", err)
	}
	if result2 != result {
		t.Error("Expected same result from cache")
	}

	// Check cache stats
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 1 {
		t.Errorf("Expected 1 result in cache, got %v", stats["results"])
	}
	if stats["misses"].(int) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
	if stats["hits"].(map[string]int)["NormalizePattern:hello world"] != 1 {
		t.Errorf("Expected 1 hit for pattern, got %v", stats["hits"])
	}
}

func TestTextNormalizationCacheDifferentTypes(t *testing.T) {
	cache := NewTextNormalizationCache(10, 60)

	testCases := []struct {
		input    string
		funcType string
	}{
		{"hello world", "NormalizePattern"},
		{"hello world", "NormalizeForMatchingCasePreserving"},
		{"hello world", "NormalizeThatPattern"},
		{"hello world", "normalizeForMatching"},
		{"don't worry", "expandContractions"},
	}

	for _, tc := range testCases {
		result, err := cache.GetNormalizedText(nil, tc.input, tc.funcType)
		if err != nil {
			t.Fatalf("Failed to normalize %s with %s: %v", tc.input, tc.funcType, err)
		}
		if result == "" {
			t.Errorf("Expected normalized result for %s, got empty string", tc.funcType)
		}
	}

	// Check that all results are cached separately
	stats := cache.GetCacheStats()
	if stats["results"].(int) != len(testCases) {
		t.Errorf("Expected %d results in cache, got %v", len(testCases), stats["results"])
	}
}

func TestTextNormalizationCacheLRUEviction(t *testing.T) {
	// Create a small cache
	cache := NewTextNormalizationCache(2, 60)

	// Add results to fill the cache
	_, err := cache.GetNormalizedText(nil, "pattern1", "NormalizePattern")
	if err != nil {
		t.Fatalf("Failed to normalize pattern1: %v", err)
	}
	_, err = cache.GetNormalizedText(nil, "pattern2", "NormalizePattern")
	if err != nil {
		t.Fatalf("Failed to normalize pattern2: %v", err)
	}

	// Add a third result to trigger eviction
	_, err = cache.GetNormalizedText(nil, "pattern3", "NormalizePattern")
	if err != nil {
		t.Fatalf("Failed to normalize pattern3: %v", err)
	}

	// Check that cache size is still 2
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 2 {
		t.Errorf("Expected 2 results in cache after eviction, got %v", stats["results"])
	}

	// pattern1 should be evicted (LRU)
	_, found := cache.Results["NormalizePattern:pattern1"]
	if found {
		t.Error("Expected pattern1 to be evicted")
	}

	// pattern2 and pattern3 should still be there
	_, found = cache.Results["NormalizePattern:pattern2"]
	if !found {
		t.Error("Expected pattern2 to still be in cache")
	}
	_, found = cache.Results["NormalizePattern:pattern3"]
	if !found {
		t.Error("Expected pattern3 to still be in cache")
	}
}

func TestTextNormalizationCacheTTL(t *testing.T) {
	// Create a cache with very short TTL
	cache := NewTextNormalizationCache(10, 1) // 1 second TTL

	// Add a result
	_, err := cache.GetNormalizedText(nil, "test pattern", "NormalizePattern")
	if err != nil {
		t.Fatalf("Failed to normalize text: %v", err)
	}

	// Verify it's in cache
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 1 {
		t.Errorf("Expected 1 result in cache, got %v", stats["results"])
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Try to get the result again - should renormalize due to TTL expiry
	_, err = cache.GetNormalizedText(nil, "test pattern", "NormalizePattern")
	if err != nil {
		t.Fatalf("Failed to renormalize text after TTL: %v", err)
	}

	// Should still have 1 result (renormalized)
	stats = cache.GetCacheStats()
	if stats["results"].(int) != 1 {
		t.Errorf("Expected 1 result in cache after TTL expiry, got %v", stats["results"])
	}
}

func TestTextNormalizationCacheInvalidType(t *testing.T) {
	cache := NewTextNormalizationCache(10, 60)

	// Try to normalize with invalid type
	_, err := cache.GetNormalizedText(nil, "test", "InvalidType")
	if err == nil {
		t.Error("Expected error for invalid normalization type")
	}

	// Check that it was counted as a miss (error case doesn't increment misses)
	stats := cache.GetCacheStats()
	if stats["misses"].(int) != 0 {
		t.Errorf("Expected 0 misses for invalid type (error case), got %v", stats["misses"])
	}
}

func TestGolemTextNormalizationCacheIntegration(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, true)

	// Test that text normalization cache is initialized
	if g.textNormalizationCache == nil {
		t.Error("Expected textNormalizationCache to be initialized")
	}

	// Test cached normalization methods
	testInput := "Hello World! This is a test."

	// Test CachedNormalizePattern
	result1 := g.CachedNormalizePattern(testInput)
	result2 := g.CachedNormalizePattern(testInput)
	if result1 != result2 {
		t.Error("Expected same result from cached normalization")
	}

	// Test CachedNormalizeForMatchingCasePreserving
	result3 := g.CachedNormalizeForMatchingCasePreserving(testInput)
	result4 := g.CachedNormalizeForMatchingCasePreserving(testInput)
	if result3 != result4 {
		t.Error("Expected same result from cached case-preserving normalization")
	}

	// Test CachedNormalizeThatPattern
	result5 := g.CachedNormalizeThatPattern(testInput)
	result6 := g.CachedNormalizeThatPattern(testInput)
	if result5 != result6 {
		t.Error("Expected same result from cached that pattern normalization")
	}

	// Test CachedExpandContractions
	result7 := g.CachedExpandContractions("don't worry")
	result8 := g.CachedExpandContractions("don't worry")
	if result7 != result8 {
		t.Error("Expected same result from cached contraction expansion")
	}

	// Test cache stats
	stats := g.GetTextNormalizationCacheStats()
	if stats["results"].(int) == 0 {
		t.Error("Expected some results in cache")
	}

	// Test clearing cache
	g.ClearTextNormalizationCache()
	stats = g.GetTextNormalizationCacheStats()
	if stats["results"].(int) != 0 {
		t.Error("Expected cache to be empty after clear")
	}
}

func TestTextNormalizationCachePerformance(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, false) // Disable verbose logging

	// Test data
	testInputs := []string{
		"Hello world! This is a test.",
		"Don't worry, it'll work.",
		"I'm happy you're here.",
		"Can't you see the problem?",
		"What's going on here?",
	}

	// Warm up the cache
	for _, input := range testInputs {
		g.CachedNormalizePattern(input)
		g.CachedNormalizeForMatchingCasePreserving(input)
		g.CachedExpandContractions(input)
	}

	// Clear cache stats
	g.ClearTextNormalizationCache()

	// Test performance with caching
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, input := range testInputs {
			g.CachedNormalizePattern(input)
			g.CachedNormalizeForMatchingCasePreserving(input)
			g.CachedExpandContractions(input)
		}
	}
	cachedDuration := time.Since(start)

	// Test performance without caching (direct calls)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		for _, input := range testInputs {
			NormalizePattern(input)
			NormalizeForMatchingCasePreserving(input)
			expandContractions(input)
		}
	}
	uncachedDuration := time.Since(start)

	// Get cache statistics
	stats := g.GetTextNormalizationCacheStats()

	t.Logf("Normalization Cache Performance Test Results:")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Inputs per iteration: %d", len(testInputs))
	t.Logf("Total normalization operations: %d", iterations*len(testInputs)*3)
	t.Logf("Cached duration: %v", cachedDuration)
	t.Logf("Uncached duration: %v", uncachedDuration)
	t.Logf("Cache hit rate: %.2f%%", stats["hit_rate"].(float64)*100)
	t.Logf("Cache results: %d", stats["results"].(int))
	t.Logf("Cache hits: %v", stats["hits"])
	t.Logf("Cache misses: %d", stats["misses"].(int))

	// Verify that caching provides performance benefits
	if cachedDuration >= uncachedDuration {
		t.Logf("Warning: Cached performance (%v) not better than uncached (%v)", cachedDuration, uncachedDuration)
		t.Logf("This might be due to small test size or system variations")
	} else {
		improvement := float64(uncachedDuration-cachedDuration) / float64(uncachedDuration) * 100
		t.Logf("Performance improvement: %.1f%%", improvement)
	}

	// Verify cache is working
	if stats["results"].(int) == 0 {
		t.Error("Expected results to be cached")
	}
	if stats["hits"].(map[string]int) == nil {
		t.Error("Expected cache hits to be recorded")
	}
}
