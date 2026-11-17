package golem

import (
	"regexp"
	"testing"
	"time"
)

func TestEnhancedThatHistoryManagement(t *testing.T) {
	// Create a session with enhanced context management
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Initialize context config
	session.InitializeContextConfig()

	// Test basic that history management
	t.Run("BasicThatHistory", func(t *testing.T) {
		session.AddToThatHistory("Hello there!")
		session.AddToThatHistory("How are you?")
		session.AddToThatHistory("What's your name?")

		// Test getting that by index
		lastThat := session.GetThatByIndex(1)
		if lastThat != "What's your name?" {
			t.Errorf("Expected 'What's your name?', got '%s'", lastThat)
		}

		secondThat := session.GetThatByIndex(2)
		if secondThat != "How are you?" {
			t.Errorf("Expected 'How are you?', got '%s'", secondThat)
		}

		// Test history length
		history := session.GetThatHistory()
		if len(history) != 3 {
			t.Errorf("Expected history length 3, got %d", len(history))
		}
	})

	// Test that history statistics
	t.Run("ThatHistoryStats", func(t *testing.T) {
		stats := session.GetThatHistoryStats()

		if stats["total_items"] != 3 {
			t.Errorf("Expected total_items 3, got %v", stats["total_items"])
		}

		if stats["max_depth"] != 20 {
			t.Errorf("Expected max_depth 20, got %v", stats["max_depth"])
		}

		if stats["newest_item"] != "What's your name?" {
			t.Errorf("Expected newest_item 'What's your name?', got %v", stats["newest_item"])
		}

		if stats["oldest_item"] != "Hello there!" {
			t.Errorf("Expected oldest_item 'Hello there!', got %v", stats["oldest_item"])
		}
	})

	// Test that history validation
	t.Run("ThatHistoryValidation", func(t *testing.T) {
		// Add some problematic items
		session.AddToThatHistory("")          // Empty item
		session.AddToThatHistory("Duplicate") // Will add duplicate next
		session.AddToThatHistory("Duplicate") // Duplicate consecutive

		errors := session.ValidateThatHistory()

		// Should have validation errors
		if len(errors) == 0 {
			t.Error("Expected validation errors, got none")
		}

		// Check for specific errors
		foundEmpty := false
		foundDuplicate := false
		for _, err := range errors {
			if containsThatHistory(err, "Empty that history item") {
				foundEmpty = true
			}
			if containsThatHistory(err, "Duplicate consecutive") {
				foundDuplicate = true
			}
		}

		if !foundEmpty {
			t.Error("Expected empty item validation error")
		}

		if !foundDuplicate {
			t.Error("Expected duplicate validation error")
		}
	})

	// Test that history compression
	t.Run("ThatHistoryCompression", func(t *testing.T) {
		// Add many items to trigger compression
		for i := 0; i < 100; i++ {
			session.AddToThatHistory("Response " + string(rune(i+'A')))
		}

		// Compression should have been triggered
		history := session.GetThatHistory()
		if len(history) > session.ContextConfig.MaxThatDepth {
			t.Errorf("History length %d exceeds max depth %d", len(history), session.ContextConfig.MaxThatDepth)
		}
	})

	// Test that history debug info
	t.Run("ThatHistoryDebugInfo", func(t *testing.T) {
		debugInfo := session.GetThatHistoryDebugInfo()

		if debugInfo["length"] == nil {
			t.Error("Expected length in debug info")
		}

		if debugInfo["memory_usage"] == nil {
			t.Error("Expected memory_usage in debug info")
		}

		if debugInfo["validation_errors"] == nil {
			t.Error("Expected validation_errors in debug info")
		}

		if debugInfo["config"] == nil {
			t.Error("Expected config in debug info")
		}
	})

	// Test clearing that history
	t.Run("ClearThatHistory", func(t *testing.T) {
		session.ClearThatHistory()

		history := session.GetThatHistory()
		if len(history) != 0 {
			t.Errorf("Expected empty history after clear, got length %d", len(history))
		}

		lastThat := session.GetLastThat()
		if lastThat != "" {
			t.Errorf("Expected empty last that after clear, got '%s'", lastThat)
		}
	})
}

func TestThatPatternCache(t *testing.T) {
	// Create a new pattern cache
	cache := NewThatPatternCache(10)

	// Test cache operations
	t.Run("CacheOperations", func(t *testing.T) {
		// Test getting non-existent pattern
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
	})

	// Test cache statistics
	t.Run("CacheStats", func(t *testing.T) {
		stats := cache.GetCacheStats()

		if stats["patterns"] != 1 {
			t.Errorf("Expected 1 pattern in cache, got %v", stats["patterns"])
		}

		if stats["max_size"] != 10 {
			t.Errorf("Expected max_size 10, got %v", stats["max_size"])
		}

		if stats["misses"] == nil {
			t.Error("Expected misses in stats")
		}

		if stats["hit_rate"] == nil {
			t.Error("Expected hit_rate in stats")
		}
	})

	// Test cache eviction
	t.Run("CacheEviction", func(t *testing.T) {
		// Fill cache beyond capacity
		for i := 0; i < 15; i++ {
			pattern := "pattern" + string(rune(i+'A'))
			compiled, _ := regexp.Compile("test.*" + pattern)
			cache.SetCompiledPattern(pattern, compiled)
		}

		// Cache should not exceed max size
		stats := cache.GetCacheStats()
		if stats["patterns"].(int) > cache.MaxSize {
			t.Errorf("Cache size %d exceeds max size %d", stats["patterns"], cache.MaxSize)
		}
	})

	// Test cache clearing
	t.Run("CacheClear", func(t *testing.T) {
		cache.ClearCache()

		stats := cache.GetCacheStats()
		if stats["patterns"] != 0 {
			t.Errorf("Expected 0 patterns after clear, got %v", stats["patterns"])
		}

		if stats["hits"] == nil {
			t.Error("Expected hits in stats")
		}
	})
}

func TestThatHistoryMemoryManagement(t *testing.T) {
	session := &ChatSession{
		ID:              "memory-test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	session.InitializeContextConfig()

	// Test memory usage calculation
	t.Run("MemoryUsageCalculation", func(t *testing.T) {
		// Add some items
		session.AddToThatHistory("Short response")
		session.AddToThatHistory("This is a much longer response that should use more memory")
		session.AddToThatHistory("Another response")

		memoryUsage := session.calculateThatHistoryMemoryUsage()
		if memoryUsage <= 0 {
			t.Error("Expected positive memory usage")
		}

		// Memory usage should be reasonable
		if memoryUsage > 10000 { // 10KB
			t.Errorf("Memory usage %d seems too high", memoryUsage)
		}
	})

	// Test memory limit validation
	t.Run("MemoryLimitValidation", func(t *testing.T) {
		// Add a very long response to test memory limit
		longResponse := ""
		for i := 0; i < 2000; i++ { // Increased to ensure we exceed 100KB
			longResponse += "This is a very long response that will use a lot of memory. "
		}

		session.AddToThatHistory(longResponse)

		// Check actual memory usage
		memoryUsage := session.calculateThatHistoryMemoryUsage()
		t.Logf("Memory usage: %d bytes", memoryUsage)

		errors := session.ValidateThatHistory()

		// Should have memory usage warning
		foundMemoryWarning := false
		for _, err := range errors {
			if containsThatHistory(err, "memory usage too high") {
				foundMemoryWarning = true
				break
			}
		}

		if !foundMemoryWarning {
			t.Errorf("Expected memory usage warning, got errors: %v", errors)
		}
	})
}

func TestThatHistoryPerformance(t *testing.T) {
	session := &ChatSession{
		ID:              "performance-test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	session.InitializeContextConfig()

	// Test performance with many operations
	t.Run("PerformanceTest", func(t *testing.T) {
		start := time.Now()

		// Add many items
		for i := 0; i < 1000; i++ {
			session.AddToThatHistory("Response " + string(rune(i%26+'A')))
		}

		// Get items by index
		for i := 1; i <= 100; i++ {
			session.GetThatByIndex(i)
		}

		// Get statistics
		session.GetThatHistoryStats()
		session.GetThatHistoryDebugInfo()

		elapsed := time.Since(start)

		// Should complete within reasonable time (1 second)
		if timingChecksEnabled() && elapsed > time.Second {
			t.Errorf("That history operations took too long: %v", elapsed)
		}
	})
}

// Helper function to check if a string contains a substring
func containsThatHistory(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstringHelper(s, substr))))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
