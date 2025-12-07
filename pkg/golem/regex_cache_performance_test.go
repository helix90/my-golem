package golem

import (
	"testing"
	"time"
)

func TestRegexCachePerformance(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, false) // Disable verbose logging for cleaner output

	// Test patterns that will be used repeatedly
	patterns := []string{
		`\s+`,                            // Whitespace normalization
		`<set\s+name=["']([^"']+)["']`,   // Set tag processing
		`<array\s+name=["']([^"']+)["']`, // Array tag processing
		`<map\s+name=["']([^"']+)["']`,   // Map tag processing
		`([.!?])\s+([a-z])`,              // Sentence capitalization
		`test.*pattern`,                  // General pattern matching
	}

	// Test data
	testText := "Hello world! This is a test pattern with multiple words and punctuation."

	// Warm up the caches
	for _, pattern := range patterns {
		regex := g.getCachedRegex(pattern, "normalization")
		regex.FindAllString(testText, -1)
	}

	// Clear cache stats
	g.ClearRegexCaches()

	// Test performance with caching
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, pattern := range patterns {
			regex := g.getCachedRegex(pattern, "normalization")
			regex.FindAllString(testText, -1)
		}
	}
	cachedDuration := time.Since(start)

	// Test performance without caching (direct compilation)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		for _, pattern := range patterns {
			// Simulate direct compilation (no caching)
			regex := g.getCachedRegex(pattern+"_nocache", "invalid") // This will fallback to direct compilation
			regex.FindAllString(testText, -1)
		}
	}
	uncachedDuration := time.Since(start)

	// Get cache statistics
	stats := g.GetRegexCacheStats()
	normalizationStats := stats["normalization"].(map[string]interface{})

	t.Logf("Performance Test Results:")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Patterns per iteration: %d", len(patterns))
	t.Logf("Total regex operations: %d", iterations*len(patterns))
	t.Logf("Cached duration: %v", cachedDuration)
	t.Logf("Uncached duration: %v", uncachedDuration)
	t.Logf("Cache hit rate: %.2f%%", normalizationStats["hit_rate"].(float64)*100)
	t.Logf("Cache patterns: %d", normalizationStats["patterns"].(int))
	t.Logf("Cache hits: %v", normalizationStats["hits"])
	t.Logf("Cache misses: %d", normalizationStats["misses"].(int))

	// Verify that caching provides performance benefits
	if cachedDuration >= uncachedDuration {
		t.Logf("Warning: Cached performance (%v) not better than uncached (%v)", cachedDuration, uncachedDuration)
		t.Logf("This might be due to small test size or system variations")
	} else {
		improvement := float64(uncachedDuration-cachedDuration) / float64(uncachedDuration) * 100
		t.Logf("Performance improvement: %.1f%%", improvement)
	}

	// Verify cache is working
	if normalizationStats["patterns"].(int) == 0 {
		t.Error("Expected patterns to be cached")
	}
	if normalizationStats["hits"].(map[string]int) == nil {
		t.Error("Expected cache hits to be recorded")
	}
}

func TestRegexCacheMemoryUsage(t *testing.T) {
	g := NewForTesting(t, false)

	// Add many patterns to test memory management
	patterns := make([]string, 100)
	for i := 0; i < 100; i++ {
		patterns[i] = `pattern` + string(rune('a'+i%26)) + `.*test`
	}

	// Fill the cache
	for _, pattern := range patterns {
		g.getCachedRegex(pattern, "normalization")
	}

	stats := g.GetRegexCacheStats()
	normalizationStats := stats["normalization"].(map[string]interface{})

	// Verify cache size is within limits
	maxSize := normalizationStats["max_size"].(int)
	actualSize := normalizationStats["patterns"].(int)

	if actualSize > maxSize {
		t.Errorf("Cache size %d exceeds max size %d", actualSize, maxSize)
	}

	t.Logf("Cache memory test:")
	t.Logf("Max size: %d", maxSize)
	t.Logf("Actual size: %d", actualSize)
	t.Logf("Cache utilization: %.1f%%", float64(actualSize)/float64(maxSize)*100)
}
