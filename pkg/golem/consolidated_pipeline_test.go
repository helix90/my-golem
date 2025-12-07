package golem

import (
	"strings"
	"testing"
)

func TestConsolidatedPipeline(t *testing.T) {
	// Create a new Golem instance
	g := NewForTesting(t, true)

	// Consolidated pipeline is now always enabled

	// Test basic template processing
	template := "Hello <star/>"
	wildcards := map[string]string{"star1": "World"}

	// Process template
	result := g.ProcessTemplate(template, wildcards)
	expected := "Hello World"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test consolidated processor methods
	processor := g.GetConsolidatedProcessor()
	if processor == nil {
		t.Error("Consolidated processor should not be nil")
	}

	// Test metrics
	metrics := g.GetProcessorMetrics()
	if metrics == nil {
		t.Error("Processor metrics should not be nil")
	}

	// Test stats
	stats := g.GetProcessorStats()
	if stats == nil {
		t.Error("Processor stats should not be nil")
	}

	// Test processing order
	order := g.GetProcessingOrder()
	if len(order) == 0 {
		t.Error("Processing order should not be empty")
	}

	// Note: Consolidated pipeline cannot be disabled anymore
}

func TestConsolidatedPipelineWithComplexTemplate(t *testing.T) {
	// Create a new Golem instance
	g := NewForTesting(t, true)

	// Consolidated pipeline is now always enabled

	// Test with a more complex template
	template := "Hello <star/>, today is <date/>"
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
}

func TestConsolidatedPipelineMetrics(t *testing.T) {
	// Create a new Golem instance
	g := NewForTesting(t, true)

	// Consolidated pipeline is now always enabled

	// Process a few templates to generate metrics
	template := "Test <star/>"
	wildcards := map[string]string{"star1": "Template"}

	// Process multiple times
	for i := 0; i < 3; i++ {
		g.ProcessTemplate(template, wildcards)
	}

	// Check metrics
	stats := g.GetProcessorStats()
	if len(stats) == 0 {
		t.Error("Should have processor stats")
	}

	// Reset metrics
	g.ResetProcessorMetrics()

	// Check that metrics are reset
	statsAfterReset := g.GetProcessorStats()
	for _, stat := range statsAfterReset {
		if statMap, ok := stat.(map[string]interface{}); ok {
			if totalCalls, ok := statMap["total_calls"].(int64); ok && totalCalls > 0 {
				t.Error("Metrics should be reset")
			}
		}
	}
}
