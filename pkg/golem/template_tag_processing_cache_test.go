package golem

import (
	"testing"
	"time"
)

func TestTemplateTagProcessingCacheBasicOperations(t *testing.T) {
	// Create a new template tag processing cache
	cache := NewTemplateTagProcessingCache(10, 60)

	// Create a test context
	ctx := &VariableContext{
		LocalVars: map[string]string{"test_var": "local_value"},
		Session:   &ChatSession{ID: "test_session", Variables: map[string]string{"session_var": "session_value"}},
		Topic:     "",
		KnowledgeBase: &AIMLKnowledgeBase{
			Variables:  map[string]string{"global_var": "global_value"},
			Properties: map[string]string{"property_var": "property_value"},
		},
	}

	// Test getting a non-existent tag
	result, found := cache.GetProcessedTag("person", "hello world", ctx)
	if found {
		t.Error("Expected tag not to be found in cache")
	}
	if result != "" {
		t.Error("Expected empty result for non-existent tag")
	}

	// Test setting and getting a tag
	cache.SetProcessedTag("person", "hello world", "HELLO WORLD", ctx)
	result, found = cache.GetProcessedTag("person", "hello world", ctx)
	if !found {
		t.Error("Expected tag to be found in cache")
	}
	if result != "HELLO WORLD" {
		t.Errorf("Expected 'HELLO WORLD', got '%s'", result)
	}

	// Check cache stats
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 1 {
		t.Errorf("Expected 1 result in cache, got %v", stats["results"])
	}
	if stats["misses"].(int) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
}

func TestTemplateTagProcessingCacheContextInvalidation(t *testing.T) {
	cache := NewTemplateTagProcessingCache(10, 60)

	// Create initial context
	ctx1 := &VariableContext{
		LocalVars:     map[string]string{"test_var": "value1"},
		Session:       &ChatSession{ID: "session1", Variables: map[string]string{}},
		Topic:         "",
		KnowledgeBase: &AIMLKnowledgeBase{Variables: map[string]string{}},
	}

	// Set a tag in cache
	cache.SetProcessedTag("person", "hello", "HELLO", ctx1)
	result, found := cache.GetProcessedTag("person", "hello", ctx1)
	if !found || result != "HELLO" {
		t.Error("Expected tag to be found with correct value")
	}

	// Create context with different variables
	ctx2 := &VariableContext{
		LocalVars:     map[string]string{"test_var": "value2"}, // Different value
		Session:       &ChatSession{ID: "session1", Variables: map[string]string{}},
		Topic:         "",
		KnowledgeBase: &AIMLKnowledgeBase{Variables: map[string]string{}},
	}

	// Should not find cached value due to context change
	_, found = cache.GetProcessedTag("person", "hello", ctx2)
	if found {
		t.Error("Expected tag not to be found due to context change")
	}

	// Set new value for new context
	cache.SetProcessedTag("person", "hello", "HELLO2", ctx2)
	result, found = cache.GetProcessedTag("person", "hello", ctx2)
	if !found || result != "HELLO2" {
		t.Error("Expected tag to be found with new value")
	}
}

func TestTemplateTagProcessingCacheLRUEviction(t *testing.T) {
	// Create a small cache
	cache := NewTemplateTagProcessingCache(2, 60)

	ctx := &VariableContext{
		Session: &ChatSession{ID: "test_session"},
	}

	// Add tags to fill the cache
	cache.SetProcessedTag("person", "hello", "HELLO", ctx)
	cache.SetProcessedTag("gender", "world", "WORLD", ctx)

	// Add a third tag to trigger eviction
	cache.SetProcessedTag("uppercase", "test", "TEST", ctx)

	// Check that cache size is still 2
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 2 {
		t.Errorf("Expected 2 results in cache after eviction, got %v", stats["results"])
	}

	// person tag should be evicted (LRU)
	_, found := cache.GetProcessedTag("person", "hello", ctx)
	if found {
		t.Error("Expected person tag to be evicted")
	}

	// gender and uppercase tags should still be there
	_, found = cache.GetProcessedTag("gender", "world", ctx)
	if !found {
		t.Error("Expected gender tag to still be in cache")
	}
	_, found = cache.GetProcessedTag("uppercase", "test", ctx)
	if !found {
		t.Error("Expected uppercase tag to still be in cache")
	}
}

func TestTemplateTagProcessingCacheTTL(t *testing.T) {
	// Create a cache with very short TTL
	cache := NewTemplateTagProcessingCache(10, 1) // 1 second TTL

	ctx := &VariableContext{
		Session: &ChatSession{ID: "test_session"},
	}

	// Add a tag
	cache.SetProcessedTag("person", "hello", "HELLO", ctx)

	// Verify it's in cache
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 1 {
		t.Errorf("Expected 1 result in cache, got %v", stats["results"])
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Try to get the tag again - should not be found due to TTL expiry
	_, found := cache.GetProcessedTag("person", "hello", ctx)
	if found {
		t.Error("Expected tag not to be found after TTL expiry")
	}
}

func TestTemplateTagProcessingCacheTagTypeInvalidation(t *testing.T) {
	cache := NewTemplateTagProcessingCache(10, 60)

	ctx := &VariableContext{
		Session: &ChatSession{ID: "test_session"},
	}

	// Set up some tags with different types
	cache.SetProcessedTag("person", "hello", "HELLO", ctx)
	cache.SetProcessedTag("gender", "world", "WORLD", ctx)
	cache.SetProcessedTag("uppercase", "test", "TEST", ctx)

	// Invalidate person tags
	cache.InvalidateTagType("person")

	// person tag should be removed
	_, found := cache.GetProcessedTag("person", "hello", ctx)
	if found {
		t.Error("Expected person tag to be removed after tag type invalidation")
	}

	// gender and uppercase tags should still be there
	_, found = cache.GetProcessedTag("gender", "world", ctx)
	if !found {
		t.Error("Expected gender tag to still be in cache after tag type invalidation")
	}
	_, found = cache.GetProcessedTag("uppercase", "test", ctx)
	if !found {
		t.Error("Expected uppercase tag to still be in cache after tag type invalidation")
	}
}

func TestGolemTemplateTagProcessingCacheIntegration(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, true)

	// Test that template tag processing cache is initialized
	if g.templateTagProcessingCache == nil {
		t.Error("Expected templateTagProcessingCache to be initialized")
	}

	// Create a test session and knowledge base
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"user_name": "Alice"},
	}

	kb := &AIMLKnowledgeBase{
		Variables: map[string]string{
			"bot_name": "Golem",
		},
		Properties: map[string]string{
			"language": "en",
		},
	}
	g.SetKnowledgeBase(kb)

	_ = &VariableContext{
		LocalVars:     map[string]string{},
		Session:       session,
		Topic:         "",
		KnowledgeBase: kb,
	}

	// Test cache statistics
	stats := g.GetTemplateTagProcessingCacheStats()
	if stats["results"].(int) != 0 {
		t.Error("Expected empty cache initially")
	}
	if stats["max_size"].(int) != 1000 {
		t.Errorf("Expected max_size 1000, got %v", stats["max_size"])
	}

	// Test clearing cache
	g.ClearTemplateTagProcessingCache()
	stats = g.GetTemplateTagProcessingCacheStats()
	if stats["results"].(int) != 0 {
		t.Error("Expected cache to be empty after clear")
	}

	// Test tag type invalidation
	g.InvalidateTemplateTagType("person")
	// Should not cause any errors

	// Test context invalidation
	g.InvalidateTemplateTagContext("test_context")
	// Should not cause any errors
}

func TestTemplateTagProcessingCachePerformance(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, false) // Disable verbose logging

	// Create test session and knowledge base
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"user_name": "Alice"},
	}

	kb := &AIMLKnowledgeBase{
		Variables: map[string]string{
			"bot_name": "Golem",
		},
		Properties: map[string]string{
			"language": "en",
		},
	}
	g.SetKnowledgeBase(kb)

	ctx := &VariableContext{
		LocalVars:     map[string]string{},
		Session:       session,
		Topic:         "",
		KnowledgeBase: kb,
	}

	// Test patterns and contents
	testTags := []struct {
		tagType string
		content string
	}{
		{"person", "hello world"},
		{"gender", "how are you"},
		{"uppercase", "test message"},
		{"lowercase", "HELLO WORLD"},
		{"sentence", "this is a test"},
	}

	// Test performance with caching
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, tag := range testTags {
			// Simulate tag processing with cache
			if g.templateTagProcessingCache != nil {
				if cached, found := g.templateTagProcessingCache.GetProcessedTag(tag.tagType, tag.content, ctx); found {
					_ = cached // Use the cached result
				} else {
					// Simulate processing
					var result string
					switch tag.tagType {
					case "person":
						result = "HELLO WORLD"
					case "gender":
						result = "HOW ARE YOU"
					case "uppercase":
						result = "TEST MESSAGE"
					case "lowercase":
						result = "hello world"
					case "sentence":
						result = "This is a test"
					}
					g.templateTagProcessingCache.SetProcessedTag(tag.tagType, tag.content, result, ctx)
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
		_ = &VariableContext{
			LocalVars:     map[string]string{},
			Session:       session,
			Topic:         "",
			KnowledgeBase: kb,
		}
		for _, tag := range testTags {
			// Simulate processing without cache
			var result string
			switch tag.tagType {
			case "person":
				result = "HELLO WORLD"
			case "gender":
				result = "HOW ARE YOU"
			case "uppercase":
				result = "TEST MESSAGE"
			case "lowercase":
				result = "hello world"
			case "sentence":
				result = "This is a test"
			}
			_ = result // Use the result
		}
	}
	uncachedDuration := time.Since(start)

	// Get cache statistics
	stats := g.GetTemplateTagProcessingCacheStats()

	t.Logf("Template Tag Processing Cache Performance Test Results:")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Tags per iteration: %d", len(testTags))
	t.Logf("Total tag processing operations: %d", iterations*len(testTags))
	t.Logf("Cached duration: %v", cachedDuration)
	t.Logf("Uncached duration: %v", uncachedDuration)
	t.Logf("Cache hit rate: %.2f%%", stats["hit_rate"].(float64)*100)
	t.Logf("Cache results: %d", stats["results"].(int))
	t.Logf("Cache hits: %v", stats["hits"])
	t.Logf("Cache misses: %d", stats["misses"].(int))
	t.Logf("Tag types: %d", stats["tag_types"].(int))
	t.Logf("Context hashes: %d", stats["context_hashes"].(int))

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

func TestTemplateTagProcessingCacheAdvancedFeatures(t *testing.T) {
	cache := NewTemplateTagProcessingCache(3, 2) // Small cache, 2 second TTL

	ctx := &VariableContext{
		Session: &ChatSession{ID: "test_session"},
	}

	// Test TTL functionality
	cache.SetProcessedTag("person", "hello", "HELLO", ctx)

	// Should be found immediately
	_, found := cache.GetProcessedTag("person", "hello", ctx)
	if !found {
		t.Error("Expected tag to be found immediately")
	}

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)

	// Should not be found after TTL expiry
	_, found = cache.GetProcessedTag("person", "hello", ctx)
	if found {
		t.Error("Expected tag not to be found after TTL expiry")
	}

	// Test access order tracking
	cache.SetProcessedTag("person", "hello", "HELLO", ctx)
	cache.SetProcessedTag("gender", "world", "WORLD", ctx)
	cache.SetProcessedTag("uppercase", "test", "TEST", ctx)

	// Access person tag to make it most recent
	cache.GetProcessedTag("person", "hello", ctx)

	// Add lowercase tag, should evict gender tag (least recently used)
	cache.SetProcessedTag("lowercase", "message", "MESSAGE", ctx)

	// gender tag should be evicted
	_, found = cache.GetProcessedTag("gender", "world", ctx)
	if found {
		t.Error("Expected gender tag to be evicted")
	}

	// person, uppercase, lowercase tags should still be there
	_, found = cache.GetProcessedTag("person", "hello", ctx)
	if !found {
		t.Error("Expected person tag to still be in cache")
	}
	_, found = cache.GetProcessedTag("uppercase", "test", ctx)
	if !found {
		t.Error("Expected uppercase tag to still be in cache")
	}
	_, found = cache.GetProcessedTag("lowercase", "message", ctx)
	if !found {
		t.Error("Expected lowercase tag to still be in cache")
	}
}

func TestTemplateTagProcessingCacheStatistics(t *testing.T) {
	cache := NewTemplateTagProcessingCache(10, 60)

	ctx := &VariableContext{
		Session: &ChatSession{ID: "test_session"},
	}

	// Add some tags
	cache.SetProcessedTag("person", "hello", "HELLO", ctx)
	cache.SetProcessedTag("gender", "world", "WORLD", ctx)

	// Access tags to generate hits
	cache.GetProcessedTag("person", "hello", ctx)
	cache.GetProcessedTag("person", "hello", ctx)
	cache.GetProcessedTag("gender", "world", ctx)

	stats := cache.GetCacheStats()

	// Verify statistics
	if stats["results"].(int) != 2 {
		t.Errorf("Expected 2 results, got %v", stats["results"])
	}
	if stats["max_size"].(int) != 10 {
		t.Errorf("Expected max_size 10, got %v", stats["max_size"])
	}
	if stats["ttl_seconds"].(int64) != 60 {
		t.Errorf("Expected ttl_seconds 60, got %v", stats["ttl_seconds"])
	}
	if stats["misses"].(int) != 0 {
		t.Errorf("Expected 0 misses, got %v", stats["misses"])
	}
	if stats["tag_types"].(int) != 2 {
		t.Errorf("Expected 2 tag types, got %v", stats["tag_types"])
	}

	// Check hit rate
	hitRate := stats["hit_rate"].(float64)
	if hitRate <= 0 {
		t.Error("Expected positive hit rate")
	}
}
