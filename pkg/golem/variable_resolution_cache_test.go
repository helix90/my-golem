package golem

import (
	"testing"
	"time"
)

func TestVariableResolutionCacheBasicOperations(t *testing.T) {
	// Create a new variable resolution cache
	cache := NewVariableResolutionCache(10, 60) // 10 results, 1 minute TTL

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

	// Test getting a non-existent variable
	value, found := cache.GetResolvedVariable("nonexistent", ctx)
	if found {
		t.Error("Expected variable not to be found in cache")
	}
	if value != "" {
		t.Error("Expected empty value for non-existent variable")
	}

	// Test setting and getting a variable
	cache.SetResolvedVariable("test_var", "local_value", ctx)
	value, found = cache.GetResolvedVariable("test_var", ctx)
	if !found {
		t.Error("Expected variable to be found in cache")
	}
	if value != "local_value" {
		t.Errorf("Expected 'local_value', got '%s'", value)
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

func TestVariableResolutionCacheScopeInvalidation(t *testing.T) {
	cache := NewVariableResolutionCache(10, 60)

	// Create initial context
	ctx1 := &VariableContext{
		LocalVars:     map[string]string{"test_var": "value1"},
		Session:       &ChatSession{ID: "session1", Variables: map[string]string{}},
		Topic:         "",
		KnowledgeBase: &AIMLKnowledgeBase{Variables: map[string]string{}},
	}

	// Set a variable in cache
	cache.SetResolvedVariable("test_var", "value1", ctx1)
	value, found := cache.GetResolvedVariable("test_var", ctx1)
	if !found || value != "value1" {
		t.Error("Expected variable to be found with correct value")
	}

	// Create context with different scope (different local variable)
	ctx2 := &VariableContext{
		LocalVars:     map[string]string{"test_var": "value2"}, // Different value
		Session:       &ChatSession{ID: "session1", Variables: map[string]string{}},
		Topic:         "",
		KnowledgeBase: &AIMLKnowledgeBase{Variables: map[string]string{}},
	}

	// Should not find cached value due to scope change
	_, found = cache.GetResolvedVariable("test_var", ctx2)
	if found {
		t.Error("Expected variable not to be found due to scope change")
	}

	// Set new value for new scope
	cache.SetResolvedVariable("test_var", "value2", ctx2)
	value, found = cache.GetResolvedVariable("test_var", ctx2)
	if !found || value != "value2" {
		t.Error("Expected variable to be found with new value")
	}
}

func TestVariableResolutionCacheLRUEviction(t *testing.T) {
	// Create a small cache
	cache := NewVariableResolutionCache(2, 60)

	ctx := &VariableContext{
		Session: &ChatSession{ID: "test_session"},
	}

	// Add variables to fill the cache
	cache.SetResolvedVariable("var1", "value1", ctx)
	cache.SetResolvedVariable("var2", "value2", ctx)

	// Add a third variable to trigger eviction
	cache.SetResolvedVariable("var3", "value3", ctx)

	// Check that cache size is still 2
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 2 {
		t.Errorf("Expected 2 results in cache after eviction, got %v", stats["results"])
	}

	// var1 should be evicted (LRU)
	_, found := cache.GetResolvedVariable("var1", ctx)
	if found {
		t.Error("Expected var1 to be evicted")
	}

	// var2 and var3 should still be there
	_, found = cache.GetResolvedVariable("var2", ctx)
	if !found {
		t.Error("Expected var2 to still be in cache")
	}
	_, found = cache.GetResolvedVariable("var3", ctx)
	if !found {
		t.Error("Expected var3 to still be in cache")
	}
}

func TestVariableResolutionCacheTTL(t *testing.T) {
	// Create a cache with very short TTL
	cache := NewVariableResolutionCache(10, 1) // 1 second TTL

	ctx := &VariableContext{
		Session: &ChatSession{ID: "test_session"},
	}

	// Add a variable
	cache.SetResolvedVariable("test_var", "test_value", ctx)

	// Verify it's in cache
	stats := cache.GetCacheStats()
	if stats["results"].(int) != 1 {
		t.Errorf("Expected 1 result in cache, got %v", stats["results"])
	}

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Try to get the variable again - should not be found due to TTL expiry
	_, found := cache.GetResolvedVariable("test_var", ctx)
	if found {
		t.Error("Expected variable not to be found after TTL expiry")
	}
}

func TestGolemVariableResolutionCacheIntegration(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, true)

	// Test that variable resolution cache is initialized
	if g.variableResolutionCache == nil {
		t.Error("Expected variableResolutionCache to be initialized")
	}

	// Create a test session
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"session_var": "session_value"},
	}

	// Create a test knowledge base
	kb := &AIMLKnowledgeBase{
		Variables:  map[string]string{"global_var": "global_value"},
		Properties: map[string]string{"property_var": "property_value"},
	}
	g.SetKnowledgeBase(kb)

	// Test variable resolution with caching
	ctx := &VariableContext{
		LocalVars:     map[string]string{"local_var": "local_value"},
		Session:       session,
		Topic:         "",
		KnowledgeBase: kb,
	}

	// Test resolving different scopes
	testCases := []struct {
		varName  string
		expected string
	}{
		{"local_var", "local_value"},
		{"session_var", "session_value"},
		{"global_var", "global_value"},
		{"property_var", "property_value"},
		{"nonexistent", ""},
	}

	for _, tc := range testCases {
		result := g.resolveVariable(tc.varName, ctx)
		if result != tc.expected {
			t.Errorf("Expected '%s' for variable '%s', got '%s'", tc.expected, tc.varName, result)
		}
	}

	// Test cache stats
	stats := g.GetVariableResolutionCacheStats()
	if stats["results"].(int) == 0 {
		t.Error("Expected some results in cache")
	}

	// Test clearing cache
	g.ClearVariableResolutionCache()
	stats = g.GetVariableResolutionCacheStats()
	if stats["results"].(int) != 0 {
		t.Error("Expected cache to be empty after clear")
	}
}

func TestVariableResolutionCachePerformance(t *testing.T) {
	// Create a Golem instance
	g := NewForTesting(t, false) // Disable verbose logging

	// Create test session and knowledge base
	session := &ChatSession{
		ID: "test_session",
		Variables: map[string]string{
			"user_name": "Alice",
			"user_age":  "25",
			"user_city": "New York",
		},
	}

	kb := &AIMLKnowledgeBase{
		Variables: map[string]string{
			"bot_name": "Golem",
			"version":  "1.0",
		},
		Properties: map[string]string{
			"language": "en",
			"country":  "US",
		},
	}
	g.SetKnowledgeBase(kb)

	ctx := &VariableContext{
		LocalVars:     map[string]string{},
		Session:       session,
		Topic:         "",
		KnowledgeBase: kb,
	}

	// Test variables
	testVars := []string{"user_name", "user_age", "user_city", "bot_name", "version", "language", "country"}

	// Warm up the cache
	for _, varName := range testVars {
		g.resolveVariable(varName, ctx)
	}

	// Clear cache stats
	g.ClearVariableResolutionCache()

	// Test performance with caching
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		for _, varName := range testVars {
			g.resolveVariable(varName, ctx)
		}
	}
	cachedDuration := time.Since(start)

	// Test performance without caching (simulate by creating new Golem instance each time)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		// Create new Golem instance to simulate no caching
		gNoCache := New(false)
		gNoCache.SetKnowledgeBase(kb)
		ctxNoCache := &VariableContext{
			LocalVars:     map[string]string{},
			Session:       session,
			Topic:         "",
			KnowledgeBase: kb,
		}
		for _, varName := range testVars {
			gNoCache.resolveVariable(varName, ctxNoCache)
		}
	}
	uncachedDuration := time.Since(start)

	// Get cache statistics
	stats := g.GetVariableResolutionCacheStats()

	t.Logf("Variable Resolution Cache Performance Test Results:")
	t.Logf("Iterations: %d", iterations)
	t.Logf("Variables per iteration: %d", len(testVars))
	t.Logf("Total variable resolutions: %d", iterations*len(testVars))
	t.Logf("Cached duration: %v", cachedDuration)
	t.Logf("Uncached duration: %v", uncachedDuration)
	t.Logf("Cache hit rate: %.2f%%", stats["hit_rate"].(float64)*100)
	t.Logf("Cache results: %d", stats["results"].(int))
	t.Logf("Cache hits: %v", stats["hits"])
	t.Logf("Cache misses: %d", stats["misses"].(int))
	t.Logf("Scope hashes: %d", stats["scope_hashes"].(int))

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

func TestVariableResolutionCacheScopeHashConsistency(t *testing.T) {
	cache := NewVariableResolutionCache(10, 60)

	// Test that same scope produces same hash
	ctx1 := &VariableContext{
		LocalVars:     map[string]string{"a": "1", "b": "2"},
		Session:       &ChatSession{ID: "session1", Variables: map[string]string{"c": "3"}},
		KnowledgeBase: &AIMLKnowledgeBase{Variables: map[string]string{"d": "4"}},
	}

	ctx2 := &VariableContext{
		LocalVars:     map[string]string{"b": "2", "a": "1"}, // Different order
		Session:       &ChatSession{ID: "session1", Variables: map[string]string{"c": "3"}},
		KnowledgeBase: &AIMLKnowledgeBase{Variables: map[string]string{"d": "4"}},
	}

	hash1 := cache.generateScopeHash(ctx1)
	hash2 := cache.generateScopeHash(ctx2)

	if hash1 != hash2 {
		t.Errorf("Expected same scope hash for different variable order, got %s vs %s", hash1, hash2)
	}

	// Test that different scope produces different hash
	ctx3 := &VariableContext{
		LocalVars:     map[string]string{"a": "1", "b": "3"}, // Different value
		Session:       &ChatSession{ID: "session1", Variables: map[string]string{"c": "3"}},
		KnowledgeBase: &AIMLKnowledgeBase{Variables: map[string]string{"d": "4"}},
	}

	hash3 := cache.generateScopeHash(ctx3)

	if hash1 == hash3 {
		t.Error("Expected different scope hash for different variable values")
	}
}
