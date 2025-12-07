package golem

import (
	"strings"
	"testing"
	"time"
)

// TestEnhancedContextManagement tests the enhanced context management features
func TestEnhancedContextManagement(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Test context configuration initialization
	if session.ContextConfig == nil {
		t.Error("Expected context config to be initialized")
	}

	// Test default configuration values
	config := session.ContextConfig
	if config.MaxThatDepth != 20 {
		t.Errorf("Expected MaxThatDepth to be 20, got %d", config.MaxThatDepth)
	}
	if config.MaxRequestDepth != 20 {
		t.Errorf("Expected MaxRequestDepth to be 20, got %d", config.MaxRequestDepth)
	}
	if config.MaxResponseDepth != 20 {
		t.Errorf("Expected MaxResponseDepth to be 20, got %d", config.MaxResponseDepth)
	}
	if config.MaxTotalContext != 100 {
		t.Errorf("Expected MaxTotalContext to be 100, got %d", config.MaxTotalContext)
	}
	if config.WeightDecay != 0.9 {
		t.Errorf("Expected WeightDecay to be 0.9, got %f", config.WeightDecay)
	}
	if !config.EnableCompression {
		t.Error("Expected EnableCompression to be true")
	}
	if !config.EnableAnalytics {
		t.Error("Expected EnableAnalytics to be true")
	}
	if !config.EnablePruning {
		t.Error("Expected EnablePruning to be true")
	}
}

// TestEnhancedContextHistoryManagement tests enhanced history management
func TestEnhancedContextHistoryManagement(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Test enhanced that history management
	tags := []string{"test", "conversation"}
	metadata := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"length":    10,
	}

	session.AddToThatHistoryEnhanced("Hello there!", tags, metadata)
	session.AddToThatHistoryEnhanced("How are you?", tags, metadata)

	if len(session.ThatHistory) != 2 {
		t.Errorf("Expected 2 items in that history, got %d", len(session.ThatHistory))
	}

	// Test enhanced request history management
	requestTags := []string{"user_input", "question"}
	requestMetadata := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"length":    15,
	}

	session.AddToRequestHistoryEnhanced("What is your name?", requestTags, requestMetadata)
	session.AddToRequestHistoryEnhanced("Tell me about yourself", requestTags, requestMetadata)

	if len(session.RequestHistory) != 2 {
		t.Errorf("Expected 2 items in request history, got %d", len(session.RequestHistory))
	}

	// Test enhanced response history management
	responseTags := []string{"bot_response", "answer"}
	responseMetadata := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"length":    20,
	}

	session.AddToResponseHistoryEnhanced("I am Golem, your AI assistant", responseTags, responseMetadata)
	session.AddToResponseHistoryEnhanced("I can help you with various tasks", responseTags, responseMetadata)

	if len(session.ResponseHistory) != 2 {
		t.Errorf("Expected 2 items in response history, got %d", len(session.ResponseHistory))
	}
}

// TestContextDepthLimits tests context depth limits
func TestContextDepthLimits(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Set smaller limits for testing
	session.ContextConfig.MaxThatDepth = 3
	session.ContextConfig.MaxRequestDepth = 3
	session.ContextConfig.MaxResponseDepth = 3

	tags := []string{"test"}
	metadata := map[string]interface{}{"test": true}

	// Add more items than the limit
	for i := 0; i < 5; i++ {
		session.AddToThatHistoryEnhanced("That response", tags, metadata)
		session.AddToRequestHistoryEnhanced("User request", tags, metadata)
		session.AddToResponseHistoryEnhanced("Bot response", tags, metadata)
	}

	// Check that limits are enforced
	if len(session.ThatHistory) != 3 {
		t.Errorf("Expected that history to be limited to 3 items, got %d", len(session.ThatHistory))
	}
	if len(session.RequestHistory) != 3 {
		t.Errorf("Expected request history to be limited to 3 items, got %d", len(session.RequestHistory))
	}
	if len(session.ResponseHistory) != 3 {
		t.Errorf("Expected response history to be limited to 3 items, got %d", len(session.ResponseHistory))
	}

	// Check that the most recent items are kept
	expectedThat := "That response"
	if session.ThatHistory[len(session.ThatHistory)-1] != expectedThat {
		t.Errorf("Expected last that item to be '%s', got '%s'", expectedThat, session.ThatHistory[len(session.ThatHistory)-1])
	}
}

// TestContextWeighting tests context weighting system
func TestContextWeighting(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	tags := []string{"test"}
	metadata := map[string]interface{}{"test": true}

	// Add items with different usage patterns
	session.AddToThatHistoryEnhanced("Frequently used response", tags, metadata)
	session.AddToThatHistoryEnhanced("Rarely used response", tags, metadata)

	// Simulate usage by calling updateContextAnalytics directly
	session.ContextUsage["Frequently used response"] = 5
	session.ContextUsage["Rarely used response"] = 1

	// Update weights
	session.updateContextWeights()

	// Check that weights are calculated correctly
	frequentWeight := session.ContextWeights["that_0"]
	rareWeight := session.ContextWeights["that_1"]

	if frequentWeight <= rareWeight {
		t.Errorf("Expected frequently used item to have higher weight. Frequent: %f, Rare: %f", frequentWeight, rareWeight)
	}
}

// TestContextSearch tests context search functionality
func TestContextSearch(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	tags := []string{"test"}
	metadata := map[string]interface{}{"test": true}

	// Add various context items
	session.AddToThatHistoryEnhanced("Hello there!", tags, metadata)
	session.AddToThatHistoryEnhanced("How are you doing?", tags, metadata)
	session.AddToRequestHistoryEnhanced("What is your name?", tags, metadata)
	session.AddToRequestHistoryEnhanced("Tell me about yourself", tags, metadata)
	session.AddToResponseHistoryEnhanced("I am Golem", tags, metadata)
	session.AddToResponseHistoryEnhanced("I can help you", tags, metadata)

	// Test search across all context types
	results := session.SearchContext("hello", []string{})
	if len(results) == 0 {
		t.Error("Expected to find 'hello' in context search")
	}

	// Test search in specific context type
	thatResults := session.SearchContext("hello", []string{"that"})
	if len(thatResults) == 0 {
		t.Error("Expected to find 'hello' in that context search")
	}

	// Test search with no results
	noResults := session.SearchContext("nonexistent", []string{})
	if len(noResults) != 0 {
		t.Error("Expected no results for nonexistent search term")
	}
}

// TestContextAnalytics tests context analytics
func TestContextAnalytics(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	tags := []string{"test", "conversation"}
	metadata := map[string]interface{}{"test": true}

	// Add context items
	session.AddToThatHistoryEnhanced("Response 1", tags, metadata)
	session.AddToThatHistoryEnhanced("Response 2", tags, metadata)
	session.AddToRequestHistoryEnhanced("Request 1", tags, metadata)
	session.AddToResponseHistoryEnhanced("Bot response", tags, metadata)

	// Get analytics
	analytics := session.GetContextAnalytics()

	// Check basic counts
	if analytics.TotalItems != 4 {
		t.Errorf("Expected total items to be 4, got %d", analytics.TotalItems)
	}
	if analytics.ThatItems != 2 {
		t.Errorf("Expected that items to be 2, got %d", analytics.ThatItems)
	}
	if analytics.RequestItems != 1 {
		t.Errorf("Expected request items to be 1, got %d", analytics.RequestItems)
	}
	if analytics.ResponseItems != 1 {
		t.Errorf("Expected response items to be 1, got %d", analytics.ResponseItems)
	}

	// Check tag distribution
	if analytics.TagDistribution["test"] != 4 {
		t.Errorf("Expected 'test' tag to appear 4 times, got %d", analytics.TagDistribution["test"])
	}
	if analytics.TagDistribution["conversation"] != 4 {
		t.Errorf("Expected 'conversation' tag to appear 4 times, got %d", analytics.TagDistribution["conversation"])
	}
}

// TestContextPruning tests smart context pruning
func TestContextPruning(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Set small limits for testing
	session.ContextConfig.MaxTotalContext = 5
	session.ContextConfig.EnablePruning = true

	tags := []string{"test"}
	metadata := map[string]interface{}{"test": true}

	// Add more items than the total limit
	for i := 0; i < 10; i++ {
		session.AddToThatHistoryEnhanced("That response", tags, metadata)
		session.AddToRequestHistoryEnhanced("User request", tags, metadata)
		session.AddToResponseHistoryEnhanced("Bot response", tags, metadata)
	}

	// Check that pruning occurred
	totalContext := len(session.ThatHistory) + len(session.RequestHistory) + len(session.ResponseHistory)
	if totalContext > session.ContextConfig.MaxTotalContext {
		t.Errorf("Expected total context to be pruned to %d, got %d", session.ContextConfig.MaxTotalContext, totalContext)
	}

	// Check pruning count in metadata
	if session.ContextMetadata["pruning_count"] == nil {
		t.Error("Expected pruning count to be tracked")
	}
}

// TestContextCompression tests context compression
func TestContextCompression(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Set compression threshold
	session.ContextConfig.CompressionThreshold = 2
	session.ContextConfig.EnableCompression = true

	tags := []string{"test"}
	metadata := map[string]interface{}{"test": true}

	// Add items that exceed compression threshold
	for i := 0; i < 4; i++ {
		longContent := "This is a very long response that should be compressed when the threshold is exceeded"
		session.AddToThatHistoryEnhanced(longContent, tags, metadata)
	}

	// Trigger compression
	session.CompressContext()

	// Check that compression occurred (at least one item should be compressed)
	compressed := false
	for _, content := range session.ThatHistory {
		if len(content) <= 50 && strings.Contains(content, "...") {
			compressed = true
			break
		}
	}

	if !compressed {
		t.Error("Expected context compression to occur")
	}

	// Check compression ratio in metadata
	if session.ContextMetadata["compression_ratio"] == nil {
		t.Error("Expected compression ratio to be tracked")
	}
}

// TestContextWeightingWithAge tests context weighting with age decay
func TestContextWeightingWithAge(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	tags := []string{"test"}
	metadata := map[string]interface{}{"test": true}

	// Add items with same usage count
	session.AddToThatHistoryEnhanced("Older item", tags, metadata)
	session.AddToThatHistoryEnhanced("Newer item", tags, metadata)

	// Set same usage count for both
	session.ContextUsage["Older item"] = 2
	session.ContextUsage["Newer item"] = 2

	// Update weights
	session.updateContextWeights()

	// Check that newer item has higher weight due to age decay
	olderWeight := session.ContextWeights["that_0"]
	newerWeight := session.ContextWeights["that_1"]

	if newerWeight <= olderWeight {
		t.Errorf("Expected newer item to have higher weight due to age decay. Older: %f, Newer: %f", olderWeight, newerWeight)
	}
}

// TestContextTagManagement tests context tag management
func TestContextTagManagement(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Add items with different tags
	tags1 := []string{"greeting", "conversation"}
	tags2 := []string{"question", "conversation"}
	tags3 := []string{"response", "helpful"}

	metadata := map[string]interface{}{"test": true}

	session.AddToThatHistoryEnhanced("Hello there!", tags1, metadata)
	session.AddToRequestHistoryEnhanced("What can you do?", tags2, metadata)
	session.AddToResponseHistoryEnhanced("I can help you", tags3, metadata)

	// Check that tags are stored correctly
	if len(session.ContextTags["Hello there!"]) != 2 {
		t.Errorf("Expected 2 tags for 'Hello there!', got %d", len(session.ContextTags["Hello there!"]))
	}

	// Check tag distribution in analytics
	analytics := session.GetContextAnalytics()
	if analytics.TagDistribution["conversation"] != 2 {
		t.Errorf("Expected 'conversation' tag to appear 2 times, got %d", analytics.TagDistribution["conversation"])
	}
	if analytics.TagDistribution["greeting"] != 1 {
		t.Errorf("Expected 'greeting' tag to appear 1 time, got %d", analytics.TagDistribution["greeting"])
	}
}

// TestContextMetadataManagement tests context metadata management
func TestContextMetadataManagement(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Add items with different metadata
	metadata1 := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"length":    10,
		"type":      "greeting",
	}
	metadata2 := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"length":    15,
		"type":      "question",
	}

	tags := []string{"test"}

	session.AddToThatHistoryEnhanced("Hello!", tags, metadata1)
	session.AddToRequestHistoryEnhanced("How are you?", tags, metadata2)

	// Check that metadata is stored correctly
	if session.ContextMetadata["Hello!"] == nil {
		t.Error("Expected metadata to be stored for 'Hello!'")
	}

	// Check specific metadata values
	if storedMeta, ok := session.ContextMetadata["Hello!"].(map[string]interface{}); ok {
		if storedMeta["type"] != "greeting" {
			t.Errorf("Expected type to be 'greeting', got %v", storedMeta["type"])
		}
		if storedMeta["length"] != 10 {
			t.Errorf("Expected length to be 10, got %v", storedMeta["length"])
		}
	} else {
		t.Error("Expected metadata to be stored as map[string]interface{}")
	}
}

// TestContextSearchWithWeights tests context search with weight sorting
func TestContextSearchWithWeights(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	tags := []string{"test"}
	metadata := map[string]interface{}{"test": true}

	// Add items with different usage patterns
	session.AddToThatHistoryEnhanced("Low weight item", tags, metadata)
	session.AddToThatHistoryEnhanced("High weight item", tags, metadata)

	// Set different usage counts
	session.ContextUsage["Low weight item"] = 1
	session.ContextUsage["High weight item"] = 5

	// Update weights
	session.updateContextWeights()

	// Search for items
	results := session.SearchContext("weight", []string{"that"})

	// Check that results are sorted by weight (descending)
	if len(results) < 2 {
		t.Error("Expected at least 2 search results")
	}

	if results[0].Weight < results[1].Weight {
		t.Error("Expected results to be sorted by weight (descending)")
	}

	// Check that high weight item comes first
	if results[0].Content != "High weight item" {
		t.Errorf("Expected 'High weight item' to be first, got '%s'", results[0].Content)
	}
}

// TestContextEdgeCases tests various edge cases
func TestContextEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Test empty search
	results := session.SearchContext("", []string{})
	if len(results) != 0 {
		t.Error("Expected empty search to return no results")
	}

	// Test search with invalid context types
	results = session.SearchContext("test", []string{"invalid"})
	if len(results) != 0 {
		t.Error("Expected search with invalid context types to return no results")
	}

	// Test analytics with empty context
	analytics := session.GetContextAnalytics()
	if analytics.TotalItems != 0 {
		t.Errorf("Expected total items to be 0 for empty context, got %d", analytics.TotalItems)
	}

	// Test compression with empty context
	session.CompressContext()
	// Should not panic or cause errors

	// Test pruning with empty context
	session.pruneContextIfNeeded()
	// Should not panic or cause errors
}
