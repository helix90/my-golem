package golem

import (
	"testing"
	"time"
)

func TestEnhancedThatPatternValidation(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		expectError bool
		description string
	}{
		{
			name:        "Valid basic pattern",
			pattern:     "HELLO WORLD",
			expectError: false,
			description: "Should accept basic text pattern",
		},
		{
			name:        "Valid star wildcard",
			pattern:     "HELLO * WORLD",
			expectError: false,
			description: "Should accept star wildcard",
		},
		{
			name:        "Valid underscore wildcard",
			pattern:     "HELLO _ WORLD",
			expectError: false,
			description: "Should accept underscore wildcard",
		},
		{
			name:        "Valid caret wildcard",
			pattern:     "HELLO ^ WORLD",
			expectError: false,
			description: "Should accept caret wildcard",
		},
		{
			name:        "Valid hash wildcard",
			pattern:     "HELLO # WORLD",
			expectError: false,
			description: "Should accept hash wildcard",
		},
		{
			name:        "Valid dollar wildcard",
			pattern:     "HELLO $ WORLD",
			expectError: false,
			description: "Should accept dollar wildcard",
		},
		{
			name:        "Valid mixed wildcards",
			pattern:     "HELLO * _ ^ # $ WORLD",
			expectError: false,
			description: "Should accept mixed wildcard types",
		},
		{
			name:        "Valid with punctuation",
			pattern:     "HELLO, WORLD! HOW ARE YOU?",
			expectError: false,
			description: "Should accept punctuation",
		},
		{
			name:        "Valid with contractions",
			pattern:     "I'M FINE, THANKS",
			expectError: false,
			description: "Should accept contractions",
		},
		{
			name:        "Invalid double star",
			pattern:     "HELLO ** WORLD",
			expectError: true,
			description: "Should reject double star wildcard",
		},
		{
			name:        "Invalid double underscore",
			pattern:     "HELLO __ WORLD",
			expectError: true,
			description: "Should reject double underscore wildcard",
		},
		{
			name:        "Invalid wildcard sequence",
			pattern:     "HELLO *_ WORLD",
			expectError: true,
			description: "Should reject invalid wildcard sequence",
		},
		{
			name:        "Valid starts with wildcard",
			pattern:     "* HELLO WORLD",
			expectError: false,
			description: "Should allow pattern starting with wildcard (AIML2)",
		},
		{
			name:        "Valid ends with wildcard",
			pattern:     "HELLO WORLD *",
			expectError: false,
			description: "Should allow pattern ending with wildcard (AIML2)",
		},
		{
			name:        "Invalid too many wildcards",
			pattern:     "A * B * C * D * E * F * G * H * I * J * K",
			expectError: true,
			description: "Should reject too many wildcards",
		},
		{
			name:        "Any characters",
			pattern:     "HELLO @ WORLD",
			expectError: false,
			description: "AIML 2.0 patterns may contain any characters",
		},
		{
			name:        "Empty pattern",
			pattern:     "",
			expectError: true,
			description: "Should reject empty pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateThatPattern(tt.pattern)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for pattern '%s', but got none", tt.pattern)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for pattern '%s', but got: %v", tt.pattern, err)
			}
		})
	}
}

func TestEnhancedThatPatternIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with enhanced that patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE * MOVIES</that>
<template>Great! I love <that_star1/> too.</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE _ PIZZA</that>
<template>Excellent! <that_underscore1/> is amazing!</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE ^ MUSIC</that>
<template>Wonderful! <that_caret1/> is great!</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE # BOOKS</that>
<template>Fantastic! <that_hash1/> is wonderful!</template>
</category>
<category>
<pattern>YES</pattern>
<that>WHAT DO YOU THINK ABOUT * TECHNOLOGY</that>
<template>I think <that_star1/> is interesting.</template>
</category>
<category>
<pattern>YES</pattern>
<that>WHAT DO YOU THINK ABOUT _ ROBOTS</that>
<template>I think <that_underscore1/> is fascinating.</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		name             string
		thatContext      string
		expectedResponse string
		description      string
	}{
		{
			name:             "Star wildcard that pattern",
			thatContext:      "DO YOU LIKE ACTION MOVIES",
			expectedResponse: "Great! I love ACTION too.",
			description:      "Should match star wildcard in that pattern",
		},
		{
			name:             "Underscore wildcard that pattern",
			thatContext:      "DO YOU LIKE PIZZA PIZZA",
			expectedResponse: "Excellent! PIZZA is amazing!",
			description:      "Should match underscore wildcard in that pattern",
		},
		{
			name:             "Caret wildcard that pattern",
			thatContext:      "DO YOU LIKE JAZZ MUSIC",
			expectedResponse: "Wonderful! JAZZ is great!",
			description:      "Should match caret wildcard in that pattern",
		},
		{
			name:             "Hash wildcard that pattern",
			thatContext:      "DO YOU LIKE FICTION BOOKS",
			expectedResponse: "Fantastic! FICTION is wonderful!",
			description:      "Should match hash wildcard in that pattern",
		},
		{
			name:             "Complex star wildcard that pattern",
			thatContext:      "WHAT DO YOU THINK ABOUT ARTIFICIAL INTELLIGENCE TECHNOLOGY",
			expectedResponse: "I think ARTIFICIAL INTELLIGENCE is interesting.",
			description:      "Should match complex star wildcard in that pattern",
		},
		{
			name:             "Complex underscore wildcard that pattern",
			thatContext:      "WHAT DO YOU THINK ABOUT HUMANOID ROBOTS",
			expectedResponse: "I think HUMANOID is fascinating.",
			description:      "Should match complex underscore wildcard in that pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test session
			session := &ChatSession{
				ID:           "test-session",
				Variables:    make(map[string]string),
				History:      make([]string, 0),
				CreatedAt:    time.Now().Format(time.RFC3339),
				LastActivity: time.Now().Format(time.RFC3339),
				Topic:        "",
				ThatHistory:  make([]string, 0),
			}

			// Add the that context to history
			session.AddToThatHistory(tt.thatContext)

			// Process input
			response, err := g.ProcessInput("yes", session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if response != tt.expectedResponse {
				t.Errorf("Expected '%s', got '%s'", tt.expectedResponse, response)
			}
		})
	}
}
