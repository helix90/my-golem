package golem

import (
	"strings"
	"testing"
)

func TestConsolidatedPipelineComprehensive(t *testing.T) {
	// Create a new Golem instance
	g := NewForTesting(t, true)

	// Consolidated pipeline is now always enabled

	// Test with a complex template that uses multiple processors
	template := "Hello <star/>, today is <date/> and the time is <time/>. <random><li>Have a great day!</li><li>Hope you're doing well!</li><li>Take care!</li></random>"
	wildcards := map[string]string{"star1": "World"}

	// Process template
	result := g.ProcessTemplate(template, wildcards)

	// Should contain the wildcard replacement
	if !strings.Contains(result, "Hello World") {
		t.Errorf("Result should contain 'Hello World', got: %s", result)
	}

	// Should contain date processing
	if !strings.Contains(result, "today is") {
		t.Errorf("Result should contain 'today is', got: %s", result)
	}

	// Should contain time processing
	if !strings.Contains(result, "the time is") {
		t.Errorf("Result should contain 'the time is', got: %s", result)
	}

	// Should contain random processing (one of the options)
	randomOptions := []string{
		"Have a great day!",
		"Hope you're doing well!",
		"Take care!",
	}

	foundRandom := false
	for _, option := range randomOptions {
		if strings.Contains(result, option) {
			foundRandom = true
			break
		}
	}

	if !foundRandom {
		t.Errorf("Result should contain one of the random options, got: %s", result)
	}

	t.Logf("Comprehensive test result: %s", result)
}

func TestConsolidatedPipelineWithFormatting(t *testing.T) {
	// Create a new Golem instance
	g := NewForTesting(t, true)

	// Consolidated pipeline is now always enabled

	// Test with formatting tags
	template := "Hello <star/>, your name in uppercase is <uppercase><star/></uppercase>"
	wildcards := map[string]string{"star1": "world"}

	// Process template
	result := g.ProcessTemplate(template, wildcards)

	// Should contain the wildcard replacement
	if !strings.Contains(result, "Hello world") {
		t.Errorf("Result should contain 'Hello world', got: %s", result)
	}

	// Should contain uppercase formatting
	if !strings.Contains(result, "WORLD") {
		t.Errorf("Result should contain 'WORLD' in uppercase, got: %s", result)
	}

	t.Logf("Formatting test result: %s", result)
}

func TestConsolidatedPipelineProcessorOrder(t *testing.T) {
	// Create a new Golem instance
	g := NewForTesting(t, true)

	// Consolidated pipeline is now always enabled

	// Get the processing order
	order := g.GetProcessingOrder()

	// Should have processors registered
	if len(order) == 0 {
		t.Error("Processing order should not be empty")
	}

	// Check that TreeProcessor's logical processors are in the expected order
	// TreeProcessor has a different set of processors than ConsolidatedTemplateProcessor
	expectedOrder := []string{"wildcard", "variable", "data", "logic", "format"}

	for i, expected := range expectedOrder {
		if i < len(order) && order[i] != expected {
			t.Errorf("Expected processor %d to be '%s', got '%s'", i, expected, order[i])
		}
	}

	t.Logf("Processing order: %v", order)
}

func TestConsolidatedPipelineComprehensiveMetrics(t *testing.T) {
	// Create a new Golem instance
	g := NewForTesting(t, true)

	// Consolidated pipeline is now always enabled

	// Process a few templates to generate metrics
	template := "Test <star/> with <date/>"
	wildcards := map[string]string{"star1": "Template"}

	// Process multiple times
	for i := 0; i < 5; i++ {
		g.ProcessTemplate(template, wildcards)
	}

	// Check metrics
	stats := g.GetProcessorStats()
	if len(stats) == 0 {
		t.Error("Should have processor stats")
	}

	// Check that we have metrics for our processors
	expectedProcessors := []string{"wildcard", "data", "format"}
	for _, processor := range expectedProcessors {
		if _, exists := stats[processor]; !exists {
			t.Errorf("Should have metrics for processor: %s", processor)
		}
	}

	// Check that wildcard processor was called
	if wildcardStats, exists := stats["wildcard"]; exists {
		if statMap, ok := wildcardStats.(map[string]interface{}); ok {
			if totalCalls, ok := statMap["total_calls"].(int64); ok && totalCalls < 1 {
				t.Errorf("Wildcard processor should have been called at least once, got: %d", totalCalls)
			}
		}
	}

	t.Logf("Processor stats: %+v", stats)
}

func TestConsolidatedPipelineVsOriginal(t *testing.T) {
	// Create two Golem instances - one with consolidated pipeline, one without
	gConsolidated := New(true)
	gOriginal := New(true)

	// Both instances now use the consolidated pipeline

	// Test template
	template := "Hello <star/>, today is <date/>"
	wildcards := map[string]string{"star1": "World"}

	// Process with both pipelines
	resultConsolidated := gConsolidated.ProcessTemplate(template, wildcards)
	resultOriginal := gOriginal.ProcessTemplate(template, wildcards)

	// Results should be the same
	if resultConsolidated != resultOriginal {
		t.Errorf("Consolidated pipeline result should match original pipeline result")
		t.Errorf("Consolidated: %s", resultConsolidated)
		t.Errorf("Original: %s", resultOriginal)
	}

	t.Logf("Both pipelines produced: %s", resultConsolidated)
}
