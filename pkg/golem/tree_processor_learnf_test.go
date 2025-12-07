package golem

import (
	"strings"
	"testing"
	"time"
)

// TestTreeProcessorLearnfTagBasic tests basic <learnf> tag processing with AST
func TestTreeProcessorLearnfTagBasic(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Use AST processor

	// Initialize knowledge base
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create session
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name         string
		learnContent string
		testPattern  string
		expected     string
	}{
		{
			name: "Basic learnf",
			learnContent: `<learnf>
				<category>
					<pattern>TEST PATTERN</pattern>
					<template>Test response</template>
				</category>
			</learnf>`,
			testPattern: "test pattern",
			expected:    "Test response",
		},
		{
			name: "Learnf with wildcard",
			learnContent: `<learnf>
				<category>
					<pattern>HELLO *</pattern>
					<template>Hi <star/></template>
				</category>
			</learnf>`,
			testPattern: "hello world",
			expected:    "Hi world",
		},
		{
			name: "Learnf with variable",
			learnContent: `<learnf>
				<category>
					<pattern>WHAT IS MY NAME</pattern>
					<template>Your name is <get name="name"/></template>
				</category>
			</learnf>`,
			testPattern: "what is my name",
			expected:    "Your name is Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up variables if needed BEFORE learning (AST evaluates at learn time)
			if strings.Contains(tt.expected, "Alice") {
				session.Variables["name"] = "Alice"
			}

			// Process the learnf content
			result := g.ProcessTemplateWithContext(tt.learnContent, map[string]string{}, session)

			// learnf should return empty string after processing
			if result != "" {
				t.Errorf("Expected empty result after learnf, got '%s'", result)
			}

			// Test that the learned pattern works
			response, err := g.ProcessInput(tt.testPattern, session)
			if err != nil {
				t.Fatalf("Failed to process learned pattern: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorLearnfTagMultiple tests multiple learnf tags
func TestTreeProcessorLearnfTagMultiple(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Learn multiple patterns
	learnContent := `
		<learnf>
			<category>
				<pattern>PATTERN ONE</pattern>
				<template>Response one</template>
			</category>
		</learnf>
		<learnf>
			<category>
				<pattern>PATTERN TWO</pattern>
				<template>Response two</template>
			</category>
		</learnf>
	`

	// Process the learn content
	result := g.ProcessTemplateWithContext(learnContent, map[string]string{}, session)

	// Should return empty after processing
	if strings.TrimSpace(result) != "" {
		t.Errorf("Expected empty result after learnf, got '%s'", result)
	}

	// Test first pattern
	response1, err := g.ProcessInput("pattern one", session)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if response1 != "Response one" {
		t.Errorf("Expected 'Response one', got '%s'", response1)
	}

	// Test second pattern
	response2, err := g.ProcessInput("pattern two", session)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if response2 != "Response two" {
		t.Errorf("Expected 'Response two', got '%s'", response2)
	}
}

// TestTreeProcessorLearnfTagWithDynamicContent tests learnf with dynamic evaluation
func TestTreeProcessorLearnfTagWithDynamicContent(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Set variables for dynamic content
	session.Variables["mypattern"] = "DYNAMIC PATTERN"
	session.Variables["myresponse"] = "Dynamic response"

	// Learn with dynamic content
	learnContent := `
		<learnf>
			<category>
				<pattern><get name="mypattern"/></pattern>
				<template><get name="myresponse"/></template>
			</category>
		</learnf>
	`

	// Process the learn content
	result := g.ProcessTemplateWithContext(learnContent, map[string]string{}, session)

	// Should return empty
	if strings.TrimSpace(result) != "" {
		t.Errorf("Expected empty result, got '%s'", result)
	}

	// Test the dynamically learned pattern
	response, err := g.ProcessInput("dynamic pattern", session)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if response != "Dynamic response" {
		t.Errorf("Expected 'Dynamic response', got '%s'", response)
	}
}

// TestTreeProcessorLearnfTagPersistence tests that learnf persists across sessions
func TestTreeProcessorLearnfTagPersistence(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// First session - learn a pattern
	session1 := &ChatSession{
		ID:              "session-1",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	learnContent := `
		<learnf>
			<category>
				<pattern>PERSISTENT PATTERN</pattern>
				<template>Persistent response</template>
			</category>
		</learnf>
	`

	// Process in first session
	g.ProcessTemplateWithContext(learnContent, map[string]string{}, session1)

	// Second session - pattern should still work
	session2 := &ChatSession{
		ID:              "session-2",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Test pattern in second session
	response, err := g.ProcessInput("persistent pattern", session2)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if response != "Persistent response" {
		t.Errorf("Expected 'Persistent response', got '%s'", response)
	}
}

// TestTreeProcessorLearnfTagVsLearn tests difference between learn and learnf
func TestTreeProcessorLearnfTagVsLearn(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// First session
	session1 := &ChatSession{
		ID:              "session-1",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Learn both session-specific and persistent patterns
	learnContent := `
		<learn>
			<category>
				<pattern>SESSION PATTERN</pattern>
				<template>Session response</template>
			</category>
		</learn>
		<learnf>
			<category>
				<pattern>PERSISTENT PATTERN</pattern>
				<template>Persistent response</template>
			</category>
		</learnf>
	`

	g.ProcessTemplateWithContext(learnContent, map[string]string{}, session1)

	// Test both patterns in first session
	resp1, err := g.ProcessInput("session pattern", session1)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if resp1 != "Session response" {
		t.Errorf("Expected 'Session response', got '%s'", resp1)
	}

	resp2, err := g.ProcessInput("persistent pattern", session1)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if resp2 != "Persistent response" {
		t.Errorf("Expected 'Persistent response', got '%s'", resp2)
	}

	// Second session - only persistent pattern should work
	session2 := &ChatSession{
		ID:              "session-2",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Persistent pattern should still work
	resp3, err := g.ProcessInput("persistent pattern", session2)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if resp3 != "Persistent response" {
		t.Errorf("Expected 'Persistent response', got '%s'", resp3)
	}

	// Session-specific pattern should NOT work (would error or return default)
	// This test would depend on how the system handles unknown patterns
}

// TestTreeProcessorLearnfTagWithFormatting tests learnf with formatted content
func TestTreeProcessorLearnfTagWithFormatting(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Learn pattern with formatting in template
	learnContent := `
		<learnf>
			<category>
				<pattern>SHOUT *</pattern>
				<template><uppercase><star/></uppercase></template>
			</category>
		</learnf>
	`

	g.ProcessTemplateWithContext(learnContent, map[string]string{}, session)

	// Test the learned pattern
	response, err := g.ProcessInput("shout hello world", session)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if response != "HELLO WORLD" {
		t.Errorf("Expected 'HELLO WORLD', got '%s'", response)
	}
}

// TestTreeProcessorLearnfTagEdgeCases tests edge cases
func TestTreeProcessorLearnfTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name         string
		learnContent string
		shouldError  bool
	}{
		{
			name:         "Empty learnf",
			learnContent: "<learnf></learnf>",
			shouldError:  false,
		},
		{
			name:         "Learnf with whitespace only",
			learnContent: "<learnf>   </learnf>",
			shouldError:  false,
		},
		{
			name: "Learnf with invalid AIML",
			learnContent: `<learnf>
				<category>
					<pattern>INVALID</pattern>
				</category>
			</learnf>`,
			shouldError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tt.learnContent, map[string]string{}, session)

			// Should return empty string
			if strings.TrimSpace(result) != "" {
				t.Errorf("Expected empty result, got '%s'", result)
			}
		})
	}
}

// TestTreeProcessorLearnfTagNoContext tests learnf without context
func TestTreeProcessorLearnfTagNoContext(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	learnContent := `
		<learnf>
			<category>
				<pattern>NO CONTEXT</pattern>
				<template>Response</template>
			</category>
		</learnf>
	`

	// Process without session context
	result := g.ProcessTemplate(learnContent, map[string]string{})

	// Should still process
	if strings.TrimSpace(result) != "" {
		t.Errorf("Expected empty result, got '%s'", result)
	}
}
