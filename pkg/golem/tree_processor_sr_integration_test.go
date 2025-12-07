package golem

import (
	"strings"
	"testing"
	"time"
)

// TestTreeProcessorSRTagIntegration tests SR tag in full AIML conversation flow
func TestTreeProcessorSRTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML with SR tag patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<!-- Base patterns -->
	<category>
		<pattern>HELLO</pattern>
		<template>Hi! How can I help you?</template>
	</category>
	
	<category>
		<pattern>GOODBYE</pattern>
		<template>See you later!</template>
	</category>
	
	<category>
		<pattern>HOW ARE YOU</pattern>
		<template>I'm doing great, thanks for asking!</template>
	</category>
	
	<!-- Pattern with wildcard and SR -->
	<category>
		<pattern>GREETING *</pattern>
		<template>Nice to meet you! <sr/></template>
	</category>
	
	<category>
		<pattern>SAY *</pattern>
		<template>You asked me to say: <sr/></template>
	</category>
	
	<category>
		<pattern>PROCESS *</pattern>
		<template>Processing: <sr/></template>
	</category>
	
	<!-- Synonym handling with SR -->
	<category>
		<pattern>HI</pattern>
		<template><srai>HELLO</srai></template>
	</category>
	
	<category>
		<pattern>BYE</pattern>
		<template><srai>GOODBYE</srai></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:              "test-integration",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	// Test conversation with SR tags
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "greeting hello",
			expected: "Nice to meet you! Hi! How can I help you?",
		},
		{
			input:    "greeting goodbye",
			expected: "Nice to meet you! See you later!",
		},
		{
			input:    "greeting how are you",
			expected: "Nice to meet you! I'm doing great, thanks for asking!",
		},
		{
			input:    "say hello",
			expected: "You asked me to say: Hi! How can I help you?",
		},
		{
			input:    "process hello",
			expected: "Processing: Hi! How can I help you?",
		},
	}

	for i, tt := range tests {
		t.Run("Step_"+string(rune('A'+i)), func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input '%s': %v", tt.input, err)
			}

			if response != tt.expected {
				t.Errorf("Input: '%s'\nExpected: '%s'\nGot: '%s'", tt.input, tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorSRTagWithTreeProcessor tests using TreeProcessor directly
func TestTreeProcessorSRTagWithTreeProcessor(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hi there!"},
		{Pattern: "GOODBYE", Template: "Bye now!"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session
	session := &ChatSession{
		ID:           "test-tree-processor",
		Variables:    map[string]string{"star1": "HELLO"},
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
	}

	// Create variable context
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
		Topic:          "",
		KnowledgeBase:  kb,
		RecursionDepth: 0,
	}

	// Create tree processor
	tp := NewTreeProcessor(g)

	tests := []struct {
		name     string
		template string
		star1    string
		expected string
	}{
		{
			name:     "Simple SR reference",
			template: "Response: <sr/>",
			star1:    "HELLO",
			expected: "Response: Hi there!",
		},
		{
			name:     "SR with uppercase",
			template: "<uppercase><sr/></uppercase>",
			star1:    "HELLO",
			expected: "HI THERE!",
		},
		{
			name:     "SR with formal",
			template: "<formal><sr/></formal>",
			star1:    "HELLO",
			expected: "Hi There!",
		},
		{
			name:     "Multiple SR tags",
			template: "First: <sr/>, Second: <sr/>",
			star1:    "GOODBYE",
			expected: "First: Bye now!, Second: Bye now!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update star1
			session.Variables["star1"] = tt.star1

			result, err := tp.ProcessTemplate(tt.template, make(map[string]string), ctx)
			if err != nil {
				t.Fatalf("Failed to process template: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorSRTagComplexPatterns tests SR tag with complex wildcard patterns
func TestTreeProcessorSRTagComplexPatterns(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML with complex patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<!-- Base patterns -->
	<category>
		<pattern>I LIKE *</pattern>
		<template>That's great! <star/> is wonderful.</template>
	</category>
	
	<category>
		<pattern>I HATE *</pattern>
		<template>Sorry to hear you don't like <star/>.</template>
	</category>
	
	<!-- Pattern with SR that processes wildcard -->
	<category>
		<pattern>TELL ME *</pattern>
		<template>Let me tell you about it: <sr/></template>
	</category>
	
	<!-- Multiple wildcards -->
	<category>
		<pattern>COMPARE * AND *</pattern>
		<template>First: <star/>, Second: <star index="2"/></template>
	</category>
	
	<!-- Nested SR usage -->
	<category>
		<pattern>REPEAT *</pattern>
		<template><sr/></template>
	</category>
	
	<!-- Pattern that itself uses SR -->
	<category>
		<pattern>FORWARD *</pattern>
		<template>Forwarding: <sr/></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:              "test-complex",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	tests := []struct {
		input        string
		expectedPart string // Part of expected response
		description  string
	}{
		{
			input:        "tell me i like pizza",
			expectedPart: "That's great! pizza is wonderful",
			description:  "SR should process 'I LIKE PIZZA'",
		},
		{
			input:        "tell me i hate broccoli",
			expectedPart: "Sorry to hear you don't like broccoli",
			description:  "SR should process 'I HATE BROCCOLI'",
		},
		{
			input:        "repeat i like chocolate",
			expectedPart: "That's great! chocolate is wonderful",
			description:  "REPEAT with SR should forward to I LIKE",
		},
		{
			input:        "forward i like ice cream",
			expectedPart: "Forwarding:",
			description:  "FORWARD with SR should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input '%s': %v", tt.input, err)
			}

			if !strings.Contains(response, tt.expectedPart) {
				t.Errorf("Input: '%s'\nExpected to contain: '%s'\nGot: '%s'",
					tt.input, tt.expectedPart, response)
			}
		})
	}
}

// TestTreeProcessorSRTagEdgeCases tests edge cases for SR tag
func TestTreeProcessorSRTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML with edge case patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST</pattern>
		<template>Basic test response</template>
	</category>
	
	<!-- Pattern with SR but no wildcard -->
	<category>
		<pattern>NO WILDCARD</pattern>
		<template>No wildcard here: <sr/></template>
	</category>
	
	<!-- Pattern with SR and empty wildcard match -->
	<category>
		<pattern>EMPTY *</pattern>
		<template>Empty wildcard: <sr/></template>
	</category>
	
	<!-- SR in think tag -->
	<category>
		<pattern>THINK *</pattern>
		<template><think><sr/></think>Processed in think</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:              "test-edge-cases",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	tests := []struct {
		input       string
		expected    string
		description string
	}{
		{
			input:       "no wildcard",
			expected:    "No wildcard here:",
			description: "SR without wildcard returns empty (AIML spec compliant)",
		},
		{
			input:       "think test",
			expected:    "Processed in think",
			description: "SR in think tag should not produce output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input '%s': %v", tt.input, err)
			}

			if response != tt.expected {
				t.Errorf("Input: '%s'\nExpected: '%s'\nGot: '%s'",
					tt.input, tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorSRTagWildcardPreservation tests that SR preserves original wildcards
func TestTreeProcessorSRTagWildcardPreservation(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>INNER</pattern>
		<template>Inner response</template>
	</category>
	
	<category>
		<pattern>OUTER *</pattern>
		<template>Before SR: <star/>, After SR: <sr/>, Still: <star/></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:              "test-wildcard-preservation",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	input := "outer inner"
	response, err := g.ProcessInput(input, session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	// The SR should not affect the original star wildcards
	// Before: should show "inner", After: should show result of SR processing "inner", Still: should show "inner" again
	if !strings.Contains(response, "Before SR: inner") {
		t.Errorf("Expected original wildcard before SR, got: %s", response)
	}

	if !strings.Contains(response, "After SR: Inner response") {
		t.Errorf("Expected SR result after SR, got: %s", response)
	}

	if !strings.Contains(response, "Still: inner") {
		t.Errorf("Expected original wildcard restored after SR, got: %s", response)
	}
}
