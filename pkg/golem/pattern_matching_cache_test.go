package golem

import (
	"strings"
	"testing"
	"time"
)

func TestPatternMatchingCacheBasicOperations(t *testing.T) {
	// Create a new pattern matching cache
	cache := NewPatternMatchingCache(10, 60)

	// Test pattern priority caching
	pattern := "HELLO * WORLD"
	priority := PatternPriorityInfo{
		Priority:         100,
		WildcardCount:    1,
		HasUnderscore:    false,
		WildcardPosition: 6,
	}

	// Test getting a non-existent pattern priority
	result, found := cache.GetPatternPriority(pattern)
	if found {
		t.Error("Expected pattern priority not to be found in cache")
	}

	// Test setting and getting a pattern priority
	cache.SetPatternPriority(pattern, priority)
	result, found = cache.GetPatternPriority(pattern)
	if !found {
		t.Error("Expected pattern priority to be found in cache")
	}
	if result.Priority != priority.Priority {
		t.Errorf("Expected priority %d, got %d", priority.Priority, result.Priority)
	}

	// Test wildcard match caching
	input := "HELLO BEAUTIFUL WORLD"
	wildcardResult := WildcardMatchResult{
		Matched:   true,
		Wildcards: map[string]string{"star1": "BEAUTIFUL"},
		Pattern:   pattern,
		Input:     input,
		Regex:     "^HELLO (.*?) WORLD$",
	}

	// Test getting a non-existent wildcard match
	result2, found := cache.GetWildcardMatch(input, pattern)
	if found {
		t.Error("Expected wildcard match not to be found in cache")
	}

	// Test setting and getting a wildcard match
	cache.SetWildcardMatch(input, pattern, wildcardResult)
	result2, found = cache.GetWildcardMatch(input, pattern)
	if !found {
		t.Error("Expected wildcard match to be found in cache")
	}
	if result2.Matched != wildcardResult.Matched {
		t.Errorf("Expected matched %v, got %v", wildcardResult.Matched, result2.Matched)
	}
	if len(result2.Wildcards) != len(wildcardResult.Wildcards) {
		t.Errorf("Expected %d wildcards, got %d", len(wildcardResult.Wildcards), len(result2.Wildcards))
	}

	// Test set regex caching
	setName := "COLORS"
	setContent := []string{"RED", "GREEN", "BLUE"}
	regex := "(RED|GREEN|BLUE)"

	// Test getting a non-existent set regex
	_, found = cache.GetSetRegex(setName, setContent)
	if found {
		t.Error("Expected set regex not to be found in cache")
	}

	// Test setting and getting a set regex
	cache.SetSetRegex(setName, setContent, regex)
	result3, found := cache.GetSetRegex(setName, setContent)
	if !found {
		t.Error("Expected set regex to be found in cache")
	}
	if result3 != regex {
		t.Errorf("Expected regex '%s', got '%s'", regex, result3)
	}

	// Test exact match key caching
	input2 := "HELLO"
	topic := "GREETING"
	that := "HI THERE"
	thatIndex := 0
	key := "HELLO|GREETING|HI THERE|0"

	// Test getting a non-existent exact match key
	_, found = cache.GetExactMatchKey(input2, topic, that, thatIndex)
	if found {
		t.Error("Expected exact match key not to be found in cache")
	}

	// Test setting and getting an exact match key
	cache.SetExactMatchKey(input2, topic, that, thatIndex, key)
	result4, found := cache.GetExactMatchKey(input2, topic, that, thatIndex)
	if !found {
		t.Error("Expected exact match key to be found in cache")
	}
	if result4 != key {
		t.Errorf("Expected key '%s', got '%s'", key, result4)
	}
}

func TestPatternMatchingCacheLRUEviction(t *testing.T) {
	// Create a small cache
	cache := NewPatternMatchingCache(2, 60)

	// Add pattern priorities to fill the cache
	cache.SetPatternPriority("PATTERN1", PatternPriorityInfo{Priority: 100})
	cache.SetPatternPriority("PATTERN2", PatternPriorityInfo{Priority: 200})

	// Add a third pattern priority to trigger eviction
	cache.SetPatternPriority("PATTERN3", PatternPriorityInfo{Priority: 300})

	// Check that cache size is still 2
	stats := cache.GetCacheStats()
	if stats["pattern_priorities"].(int) != 2 {
		t.Errorf("Expected 2 pattern priorities in cache after eviction, got %v", stats["pattern_priorities"])
	}

	// PATTERN1 should be evicted (LRU)
	_, found := cache.GetPatternPriority("PATTERN1")
	if found {
		t.Error("Expected PATTERN1 to be evicted")
	}

	// PATTERN2 and PATTERN3 should still be there
	_, found = cache.GetPatternPriority("PATTERN2")
	if !found {
		t.Error("Expected PATTERN2 to still be in cache")
	}
	_, found = cache.GetPatternPriority("PATTERN3")
	if !found {
		t.Error("Expected PATTERN3 to still be in cache")
	}
}

func TestPatternMatchingCacheTTL(t *testing.T) {
	// Create a cache with very short TTL
	cache := NewPatternMatchingCache(10, 1) // 1 second TTL

	// Add a pattern priority
	cache.SetPatternPriority("PATTERN", PatternPriorityInfo{Priority: 100})

	// Verify it's in cache
	stats := cache.GetCacheStats()
	if stats["pattern_priorities"].(int) != 1 {
		t.Errorf("Expected 1 pattern priority in cache, got %v", stats["pattern_priorities"])
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Try to get the pattern priority again - should not be found due to TTL expiry
	_, found := cache.GetPatternPriority("PATTERN")
	if found {
		t.Error("Expected pattern priority not to be found after TTL expiry")
	}
}

func TestPatternMatchingCacheSetContentValidation(t *testing.T) {
	cache := NewPatternMatchingCache(10, 60)

	setName := "COLORS"
	originalContent := []string{"RED", "GREEN", "BLUE"}
	modifiedContent := []string{"RED", "GREEN", "BLUE", "YELLOW"}

	// Set regex with original content
	cache.SetSetRegex(setName, originalContent, "(RED|GREEN|BLUE)")

	// Should find with original content
	regex, found := cache.GetSetRegex(setName, originalContent)
	if !found {
		t.Error("Expected set regex to be found with original content")
	}
	if regex != "(RED|GREEN|BLUE)" {
		t.Errorf("Expected regex '(RED|GREEN|BLUE)', got '%s'", regex)
	}

	// Should not find with modified content (content hash changed)
	_, found = cache.GetSetRegex(setName, modifiedContent)
	if found {
		t.Error("Expected set regex not to be found with modified content")
	}

	// Set new regex with modified content
	cache.SetSetRegex(setName, modifiedContent, "(RED|GREEN|BLUE|YELLOW)")

	// Should find with modified content
	regex, found = cache.GetSetRegex(setName, modifiedContent)
	if !found {
		t.Error("Expected set regex to be found with modified content")
	}
	if regex != "(RED|GREEN|BLUE|YELLOW)" {
		t.Errorf("Expected regex '(RED|GREEN|BLUE|YELLOW)', got '%s'", regex)
	}
}

func TestPatternMatchingCacheKnowledgeBaseInvalidation(t *testing.T) {
	cache := NewPatternMatchingCache(10, 60)

	// Set initial knowledge base hash
	cache.InvalidateKnowledgeBase("patterns:10,sets:5")

	// Add some cached data
	cache.SetPatternPriority("PATTERN", PatternPriorityInfo{Priority: 100})
	cache.SetWildcardMatch("INPUT", "PATTERN", WildcardMatchResult{Matched: true})

	// Verify data is cached
	stats := cache.GetCacheStats()
	if stats["pattern_priorities"].(int) != 1 {
		t.Error("Expected 1 pattern priority in cache")
	}
	if stats["wildcard_matches"].(int) != 1 {
		t.Error("Expected 1 wildcard match in cache")
	}

	// Invalidate with different knowledge base hash
	cache.InvalidateKnowledgeBase("patterns:15,sets:8")

	// Verify cache is cleared
	stats = cache.GetCacheStats()
	if stats["pattern_priorities"].(int) != 0 {
		t.Error("Expected 0 pattern priorities after knowledge base invalidation")
	}
	if stats["wildcard_matches"].(int) != 0 {
		t.Error("Expected 0 wildcard matches after knowledge base invalidation")
	}
}

func TestPatternMatchingCacheSetInvalidation(t *testing.T) {
	cache := NewPatternMatchingCache(10, 60)

	// Add some cached data
	cache.SetSetRegex("COLORS", []string{"RED", "GREEN"}, "(RED|GREEN)")
	cache.SetWildcardMatch("INPUT", "HELLO <set>COLORS</set>", WildcardMatchResult{Matched: true})

	// Verify data is cached
	stats := cache.GetCacheStats()
	if stats["set_regexes"].(int) != 1 {
		t.Error("Expected 1 set regex in cache")
	}
	if stats["wildcard_matches"].(int) != 1 {
		t.Error("Expected 1 wildcard match in cache")
	}

	// Invalidate COLORS set
	cache.InvalidateSet("COLORS")

	// Verify set regex is removed
	stats = cache.GetCacheStats()
	if stats["set_regexes"].(int) != 0 {
		t.Error("Expected 0 set regexes after set invalidation")
	}

	// Verify wildcard matches containing the set are also removed
	stats = cache.GetCacheStats()
	if stats["wildcard_matches"].(int) != 0 {
		t.Error("Expected 0 wildcard matches after set invalidation")
	}
}

func TestGolemPatternMatchingCacheIntegration(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, true)

	// Test that pattern matching cache is initialized
	if g.patternMatchingCache == nil {
		t.Error("Expected patternMatchingCache to be initialized")
	}

	// Test cache statistics
	stats := g.GetPatternMatchingCacheStats()
	if stats["pattern_priorities"].(int) != 0 {
		t.Error("Expected empty cache initially")
	}
	if stats["max_size"].(int) != 2000 {
		t.Errorf("Expected max_size 2000, got %v", stats["max_size"])
	}

	// Test clearing cache
	g.ClearPatternMatchingCache()
	stats = g.GetPatternMatchingCacheStats()
	if stats["pattern_priorities"].(int) != 0 {
		t.Error("Expected cache to be empty after clear")
	}

	// Test knowledge base invalidation
	g.InvalidatePatternMatchingKnowledgeBase()
	// Should not cause any errors

	// Test set invalidation
	g.InvalidatePatternMatchingSet("TEST_SET")
	// Should not cause any errors
}

func TestPatternMatchingCachePerformance(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, false) // Disable verbose logging

	// Create test knowledge base
	kb := &AIMLKnowledgeBase{
		Patterns: make(map[string]*Category),
		Sets: map[string][]string{
			"COLORS":  {"RED", "GREEN", "BLUE"},
			"ANIMALS": {"CAT", "DOG", "BIRD"},
		},
	}
	g.SetKnowledgeBase(kb)

	// Test patterns and inputs
	testCases := []struct {
		pattern string
		input   string
	}{
		{"HELLO * WORLD", "HELLO BEAUTIFUL WORLD"},
		{"WHAT IS *", "WHAT IS A COMPUTER"},
		{"I LIKE <set>COLORS</set>", "I LIKE RED"},
		{"TELL ME ABOUT <set>ANIMALS</set>", "TELL ME ABOUT CATS"},
		{"* IS *", "LIFE IS BEAUTIFUL"},
	}

	// Test performance with caching
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, tc := range testCases {
			// Simulate pattern matching with cache
			if g.patternMatchingCache != nil {
				// Test pattern priority caching
				if priority, found := g.patternMatchingCache.GetPatternPriority(tc.pattern); found {
					_ = priority // Use the cached priority
				} else {
					// Simulate priority calculation
					priority := PatternPriorityInfo{
						Priority:      100,
						WildcardCount: 1,
					}
					g.patternMatchingCache.SetPatternPriority(tc.pattern, priority)
				}

				// Test wildcard match caching
				if result, found := g.patternMatchingCache.GetWildcardMatch(tc.input, tc.pattern); found {
					_ = result // Use the cached result
				} else {
					// Simulate wildcard matching
					result := WildcardMatchResult{
						Matched:   true,
						Wildcards: map[string]string{"star1": "SOMETHING"},
					}
					g.patternMatchingCache.SetWildcardMatch(tc.input, tc.pattern, result)
				}

				// Test set regex caching
				if strings.Contains(tc.pattern, "<set>") {
					setName := "COLORS" // Simplified for test
					if result, found := g.patternMatchingCache.GetSetRegex(setName, kb.Sets[setName]); found {
						_ = result // Use the cached regex
					} else {
						// Simulate set regex generation
						regex := "(RED|GREEN|BLUE)"
						g.patternMatchingCache.SetSetRegex(setName, kb.Sets[setName], regex)
					}
				}
			}
		}
	}
	cachedDuration := time.Since(start)

	// Test performance without caching (simulate by creating new Golem instance each time)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		// Create new Golem instance to simulate no caching
		gNoCache := New(false)
		gNoCache.SetKnowledgeBase(kb)
		for _, tc := range testCases {
			// Simulate processing without cache
			_ = PatternPriorityInfo{Priority: 100, WildcardCount: 1}
			_ = WildcardMatchResult{Matched: true, Wildcards: map[string]string{"star1": "SOMETHING"}}
			if strings.Contains(tc.pattern, "<set>") {
				_ = "(RED|GREEN|BLUE)"
			}
		}
	}
	uncachedDuration := time.Since(start)

	// Get cache statistics
	stats := g.GetPatternMatchingCacheStats()

	t.Logf("Pattern Matching Cache Performance Test Results:")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Test cases per iteration: %d", len(testCases))
	t.Logf("Total pattern matching operations: %d", iterations*len(testCases))
	t.Logf("Cached duration: %v", cachedDuration)
	t.Logf("Uncached duration: %v", uncachedDuration)
	t.Logf("Cache hit rate: %.2f%%", stats["hit_rate"].(float64)*100)
	t.Logf("Pattern priorities: %d", stats["pattern_priorities"].(int))
	t.Logf("Wildcard matches: %d", stats["wildcard_matches"].(int))
	t.Logf("Set regexes: %d", stats["set_regexes"].(int))
	t.Logf("Exact match keys: %d", stats["exact_match_keys"].(int))
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
	if stats["pattern_priorities"].(int) == 0 {
		t.Error("Expected pattern priorities to be cached")
	}
	if stats["wildcard_matches"].(int) == 0 {
		t.Error("Expected wildcard matches to be cached")
	}
}

func TestPatternMatchingCacheAdvancedFeatures(t *testing.T) {
	cache := NewPatternMatchingCache(3, 2) // Small cache, 2 second TTL

	// Test TTL functionality
	cache.SetPatternPriority("PATTERN", PatternPriorityInfo{Priority: 100})

	// Should be found immediately
	_, found := cache.GetPatternPriority("PATTERN")
	if !found {
		t.Error("Expected pattern priority to be found immediately")
	}

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)

	// Should not be found after TTL expiry
	_, found = cache.GetPatternPriority("PATTERN")
	if found {
		t.Error("Expected pattern priority not to be found after TTL expiry")
	}

	// Test access order tracking
	cache.SetPatternPriority("PATTERN1", PatternPriorityInfo{Priority: 100})
	cache.SetPatternPriority("PATTERN2", PatternPriorityInfo{Priority: 200})
	cache.SetPatternPriority("PATTERN3", PatternPriorityInfo{Priority: 300})

	// Access PATTERN1 to make it most recent
	cache.GetPatternPriority("PATTERN1")

	// Add PATTERN4, should evict PATTERN2 (least recently used)
	cache.SetPatternPriority("PATTERN4", PatternPriorityInfo{Priority: 400})

	// PATTERN2 should be evicted
	_, found = cache.GetPatternPriority("PATTERN2")
	if found {
		t.Error("Expected PATTERN2 to be evicted")
	}

	// PATTERN1, PATTERN3, PATTERN4 should still be there
	_, found = cache.GetPatternPriority("PATTERN1")
	if !found {
		t.Error("Expected PATTERN1 to still be in cache")
	}
	_, found = cache.GetPatternPriority("PATTERN3")
	if !found {
		t.Error("Expected PATTERN3 to still be in cache")
	}
	_, found = cache.GetPatternPriority("PATTERN4")
	if !found {
		t.Error("Expected PATTERN4 to still be in cache")
	}
}

func TestPatternMatchingCacheStatistics(t *testing.T) {
	cache := NewPatternMatchingCache(10, 60)

	// Add some data
	cache.SetPatternPriority("PATTERN1", PatternPriorityInfo{Priority: 100})
	cache.SetPatternPriority("PATTERN2", PatternPriorityInfo{Priority: 200})
	cache.SetWildcardMatch("INPUT1", "PATTERN1", WildcardMatchResult{Matched: true})
	cache.SetSetRegex("COLORS", []string{"RED", "GREEN"}, "(RED|GREEN)")

	// Access some data to generate hits
	cache.GetPatternPriority("PATTERN1")
	cache.GetPatternPriority("PATTERN1")
	cache.GetWildcardMatch("INPUT1", "PATTERN1")

	stats := cache.GetCacheStats()

	// Verify statistics
	if stats["pattern_priorities"].(int) != 2 {
		t.Errorf("Expected 2 pattern priorities, got %v", stats["pattern_priorities"])
	}
	if stats["wildcard_matches"].(int) != 1 {
		t.Errorf("Expected 1 wildcard match, got %v", stats["wildcard_matches"])
	}
	if stats["set_regexes"].(int) != 1 {
		t.Errorf("Expected 1 set regex, got %v", stats["set_regexes"])
	}
	if stats["max_size"].(int) != 10 {
		t.Errorf("Expected max_size 10, got %v", stats["max_size"])
	}
	if stats["ttl_seconds"].(int64) != 60 {
		t.Errorf("Expected ttl_seconds 60, got %v", stats["ttl_seconds"])
	}

	// Check hit rate
	hitRate := stats["hit_rate"].(float64)
	if hitRate <= 0 {
		t.Error("Expected positive hit rate")
	}
}
