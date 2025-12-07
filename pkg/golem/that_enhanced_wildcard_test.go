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
			name:        "Invalid starts with wildcard",
			pattern:     "* HELLO WORLD",
			expectError: true,
			description: "Should reject pattern starting with wildcard",
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
			name:        "Invalid characters",
			pattern:     "HELLO @ WORLD",
			expectError: true,
			description: "Should reject invalid characters",
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

func TestEnhancedThatPatternMatching(t *testing.T) {
	tests := []struct {
		name              string
		thatContext       string
		thatPattern       string
		shouldMatch       bool
		expectedWildcards map[string]string
		description       string
	}{
		{
			name:              "Exact match",
			thatContext:       "HELLO WORLD",
			thatPattern:       "HELLO WORLD",
			shouldMatch:       true,
			expectedWildcards: map[string]string{},
			description:       "Should match exact text",
		},
		{
			name:              "Star wildcard match",
			thatContext:       "HELLO BEAUTIFUL WORLD",
			thatPattern:       "HELLO * WORLD",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "BEAUTIFUL"},
			description:       "Should match with star wildcard",
		},
		{
			name:              "Underscore wildcard match",
			thatContext:       "HELLO BEAUTIFUL WORLD",
			thatPattern:       "HELLO _ WORLD",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_underscore1": "BEAUTIFUL"},
			description:       "Should match with underscore wildcard",
		},
		{
			name:              "Caret wildcard match",
			thatContext:       "HELLO BEAUTIFUL WORLD",
			thatPattern:       "HELLO ^ WORLD",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_caret1": "BEAUTIFUL"},
			description:       "Should match with caret wildcard",
		},
		{
			name:              "Hash wildcard match",
			thatContext:       "HELLO BEAUTIFUL WORLD",
			thatPattern:       "HELLO # WORLD",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_hash1": "BEAUTIFUL"},
			description:       "Should match with hash wildcard",
		},
		{
			name:        "Multiple wildcards match",
			thatContext: "HELLO BEAUTIFUL WONDERFUL WORLD",
			thatPattern: "HELLO * * WORLD",
			shouldMatch: true,
			expectedWildcards: map[string]string{
				"that_star1": "BEAUTIFUL",
				"that_star2": "WONDERFUL",
			},
			description: "Should match with multiple wildcards",
		},
		{
			name:        "Mixed wildcard types match",
			thatContext: "HELLO BEAUTIFUL WONDERFUL WORLD",
			thatPattern: "HELLO _ * WORLD",
			shouldMatch: true,
			expectedWildcards: map[string]string{
				"that_underscore1": "BEAUTIFUL",
				"that_star2":       "WONDERFUL",
			},
			description: "Should match with mixed wildcard types",
		},
		{
			name:              "Zero wildcard match",
			thatContext:       "HELLO WORLD",
			thatPattern:       "HELLO * WORLD",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": ""},
			description:       "Should match with zero wildcard",
		},
		{
			name:              "No match - different text",
			thatContext:       "HELLO WORLD",
			thatPattern:       "GOODBYE WORLD",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match different text",
		},
		{
			name:              "No match - underscore requires word",
			thatContext:       "HELLO WORLD",
			thatPattern:       "HELLO _ WORLD",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match when underscore requires word but none present",
		},
		{
			name:              "Case insensitive match",
			thatContext:       "hello world",
			thatPattern:       "HELLO WORLD",
			shouldMatch:       true,
			expectedWildcards: map[string]string{},
			description:       "Should match case insensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, wildcards := matchThatPatternWithWildcards(tt.thatContext, tt.thatPattern)

			if matched != tt.shouldMatch {
				t.Errorf("Expected match=%v, got match=%v for context='%s', pattern='%s'",
					tt.shouldMatch, matched, tt.thatContext, tt.thatPattern)
			}

			if tt.shouldMatch && tt.expectedWildcards != nil {
				if len(wildcards) != len(tt.expectedWildcards) {
					t.Errorf("Expected %d wildcards, got %d: %v",
						len(tt.expectedWildcards), len(wildcards), wildcards)
				}

				for key, expectedValue := range tt.expectedWildcards {
					if actualValue, exists := wildcards[key]; !exists {
						t.Errorf("Expected wildcard '%s' not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected wildcard '%s'='%s', got '%s'",
							key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestThatPatternPriority(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		expected    int
		description string
	}{
		{
			name:        "Exact match",
			pattern:     "HELLO WORLD",
			expected:    2410, // Base 1000 + 500 for no wildcards + 910 for word count (3 words * 5 + 9*100)
			description: "Exact match should have highest priority",
		},
		{
			name:        "Single star wildcard",
			pattern:     "HELLO * WORLD",
			expected:    1835, // Base 1000 + 100 for 8 remaining wildcards + 20 for star + 715 for word count (3 words * 5 + 8*100)
			description: "Single star should have high priority",
		},
		{
			name:        "Single underscore wildcard",
			pattern:     "HELLO _ WORLD",
			expected:    1825, // Base 1000 + 100 for 8 remaining wildcards + 10 for underscore + 715 for word count
			description: "Single underscore should have medium priority",
		},
		{
			name:        "Single caret wildcard",
			pattern:     "HELLO ^ WORLD",
			expected:    2445, // Base 1000 + 100 for 8 remaining wildcards + 30 for caret + 1315 for word count
			description: "Single caret should have medium-high priority",
		},
		{
			name:        "Single hash wildcard",
			pattern:     "HELLO # WORLD",
			expected:    1855, // Base 1000 + 100 for 8 remaining wildcards + 40 for hash + 715 for word count
			description: "Single hash should have high priority",
		},
		{
			name:        "Single dollar wildcard",
			pattern:     "HELLO $ WORLD",
			expected:    1865, // Base 1000 + 100 for 8 remaining wildcards + 50 for dollar + 715 for word count
			description: "Single dollar should have highest wildcard priority",
		},
		{
			name:        "Multiple wildcards",
			pattern:     "HELLO * _ WORLD",
			expected:    1750, // Base 1000 + 100 for 7 remaining wildcards + 20 for star + 10 for underscore + 620 for word count
			description: "Multiple wildcards should have lower priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := calculateThatPatternPriority(tt.pattern)
			if priority != tt.expected {
				t.Errorf("Expected priority %d, got %d for pattern '%s'",
					tt.expected, priority, tt.pattern)
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
<that>DO YOU LIKE $ SPORTS</that>
<template>Outstanding! <that_dollar1/> is perfect!</template>
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
			name:             "Dollar wildcard that pattern",
			thatContext:      "DO YOU LIKE SOCCER SPORTS",
			expectedResponse: "Outstanding! SOCCER is perfect!",
			description:      "Should match dollar wildcard in that pattern",
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

func TestThatPatternWildcardTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		position int
		expected string
	}{
		{
			name:     "Star at position 0",
			pattern:  "HELLO * WORLD",
			position: 0,
			expected: "star",
		},
		{
			name:     "Underscore at position 0",
			pattern:  "HELLO _ WORLD",
			position: 0,
			expected: "underscore",
		},
		{
			name:     "Caret at position 0",
			pattern:  "HELLO ^ WORLD",
			position: 0,
			expected: "caret",
		},
		{
			name:     "Hash at position 0",
			pattern:  "HELLO # WORLD",
			position: 0,
			expected: "hash",
		},
		{
			name:     "Dollar at position 0",
			pattern:  "HELLO $ WORLD",
			position: 0,
			expected: "dollar",
		},
		{
			name:     "Mixed wildcards - first",
			pattern:  "HELLO * _ ^ # $ WORLD",
			position: 0,
			expected: "star",
		},
		{
			name:     "Mixed wildcards - second",
			pattern:  "HELLO * _ ^ # $ WORLD",
			position: 1,
			expected: "underscore",
		},
		{
			name:     "Mixed wildcards - third",
			pattern:  "HELLO * _ ^ # $ WORLD",
			position: 2,
			expected: "caret",
		},
		{
			name:     "Mixed wildcards - fourth",
			pattern:  "HELLO * _ ^ # $ WORLD",
			position: 3,
			expected: "hash",
		},
		{
			name:     "Mixed wildcards - fifth",
			pattern:  "HELLO * _ ^ # $ WORLD",
			position: 4,
			expected: "dollar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineThatWildcardType(tt.pattern, tt.position)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s' for pattern '%s' at position %d",
					tt.expected, result, tt.pattern, tt.position)
			}
		})
	}
}
