package golem

import (
	"strings"
	"testing"
)

// TestEnhancedThatPatternWithSets tests that pattern matching with set support
func TestEnhancedThatPatternWithSets(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Add test sets
	kb.Sets["COLORS"] = []string{"RED", "BLUE", "GREEN", "YELLOW"}
	kb.Sets["ANIMALS"] = []string{"DOG", "CAT", "BIRD", "FISH"}

	// Test that patterns with sets
	tests := []struct {
		name              string
		thatContext       string
		thatPattern       string
		shouldMatch       bool
		expectedWildcards map[string]string
		description       string
	}{
		{
			name:              "Set match - single color",
			thatContext:       "I LIKE RED",
			thatPattern:       "I LIKE <set>COLORS</set>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "RED"},
			description:       "Should match with set wildcard",
		},
		{
			name:              "Set match - single animal",
			thatContext:       "I HAVE A DOG",
			thatPattern:       "I HAVE A <set>ANIMALS</set>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "DOG"},
			description:       "Should match with animal set wildcard",
		},
		{
			name:              "Set match - multiple words",
			thatContext:       "I LIKE BLUE AND GREEN",
			thatPattern:       "I LIKE <set>COLORS</set> AND <set>COLORS</set>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "BLUE", "that_star2": "GREEN"},
			description:       "Should match with multiple set wildcards",
		},
		{
			name:              "Set match with other wildcards",
			thatContext:       "I LIKE BLUE AND SOMETHING ELSE",
			thatPattern:       "I LIKE <set>COLORS</set> AND *",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "BLUE", "that_star2": "SOMETHING ELSE"},
			description:       "Should match set with other wildcards",
		},
		{
			name:              "Set match with underscore wildcard",
			thatContext:       "I HAVE A DOG",
			thatPattern:       "I HAVE _ <set>ANIMALS</set>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_underscore1": "A", "that_star2": "DOG"},
			description:       "Should match set with underscore wildcard",
		},
		{
			name:              "No match - not in set",
			thatContext:       "I LIKE PURPLE",
			thatPattern:       "I LIKE <set>COLORS</set>",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match when word not in set",
		},
		{
			name:              "No match - different pattern",
			thatContext:       "I HATE RED",
			thatPattern:       "I LIKE <set>COLORS</set>",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match different pattern",
		},
		{
			name:              "Set match - case insensitive",
			thatContext:       "i like red",
			thatPattern:       "I LIKE <set>COLORS</set>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "red"},
			description:       "Should match case insensitive with set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, wildcards := matchThatPatternWithWildcardsWithGolem(g, tt.thatContext, tt.thatPattern)

			if matched != tt.shouldMatch {
				t.Errorf("Expected match %v, got %v", tt.shouldMatch, matched)
			}

			if matched && tt.expectedWildcards != nil {
				for key, expectedValue := range tt.expectedWildcards {
					if actualValue, exists := wildcards[key]; !exists {
						t.Errorf("Expected wildcard %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected wildcard %s = '%s', got '%s'", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

// TestEnhancedThatPatternWithTopics tests that pattern matching with topic support
func TestEnhancedThatPatternWithTopics(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Add test topics
	kb.Topics["SPORTS"] = []string{"FOOTBALL", "BASKETBALL", "TENNIS", "SOCCER"}
	kb.Topics["FOOD"] = []string{"PIZZA", "BURGER", "SALAD", "PASTA"}

	// Test that patterns with topics
	tests := []struct {
		name              string
		thatContext       string
		thatPattern       string
		shouldMatch       bool
		expectedWildcards map[string]string
		description       string
	}{
		{
			name:              "Topic match - single sport",
			thatContext:       "I PLAY FOOTBALL",
			thatPattern:       "I PLAY <topic>SPORTS</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "FOOTBALL"},
			description:       "Should match with topic wildcard",
		},
		{
			name:              "Topic match - single food",
			thatContext:       "I EAT PIZZA",
			thatPattern:       "I EAT <topic>FOOD</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "PIZZA"},
			description:       "Should match with food topic wildcard",
		},
		{
			name:              "Topic match - multiple words",
			thatContext:       "I PLAY FOOTBALL AND TENNIS",
			thatPattern:       "I PLAY <topic>SPORTS</topic> AND <topic>SPORTS</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "FOOTBALL", "that_star2": "TENNIS"},
			description:       "Should match with multiple topic wildcards",
		},
		{
			name:              "Topic match with other wildcards",
			thatContext:       "I PLAY FOOTBALL AND SOMETHING ELSE",
			thatPattern:       "I PLAY <topic>SPORTS</topic> AND *",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "FOOTBALL", "that_star2": "SOMETHING ELSE"},
			description:       "Should match topic with other wildcards",
		},
		{
			name:              "Topic match with underscore wildcard",
			thatContext:       "I PLAY A FOOTBALL",
			thatPattern:       "I PLAY _ <topic>SPORTS</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_underscore1": "A", "that_star2": "FOOTBALL"},
			description:       "Should match topic with underscore wildcard",
		},
		{
			name:              "No match - not in topic",
			thatContext:       "I PLAY CHESS",
			thatPattern:       "I PLAY <topic>SPORTS</topic>",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match when word not in topic",
		},
		{
			name:              "No match - different pattern",
			thatContext:       "I HATE FOOTBALL",
			thatPattern:       "I PLAY <topic>SPORTS</topic>",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match different pattern",
		},
		{
			name:              "Topic match - case insensitive",
			thatContext:       "i play football",
			thatPattern:       "I PLAY <topic>SPORTS</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "football"},
			description:       "Should match case insensitive with topic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, wildcards := matchThatPatternWithWildcardsWithGolem(g, tt.thatContext, tt.thatPattern)

			if matched != tt.shouldMatch {
				t.Errorf("Expected match %v, got %v", tt.shouldMatch, matched)
			}

			if matched && tt.expectedWildcards != nil {
				for key, expectedValue := range tt.expectedWildcards {
					if actualValue, exists := wildcards[key]; !exists {
						t.Errorf("Expected wildcard %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected wildcard %s = '%s', got '%s'", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

// TestEnhancedThatPatternWithMixedSetsAndTopics tests that pattern matching with mixed set and topic support
func TestEnhancedThatPatternWithMixedSetsAndTopics(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Add test sets and topics
	kb.Sets["COLORS"] = []string{"RED", "BLUE", "GREEN", "YELLOW"}
	kb.Topics["SPORTS"] = []string{"FOOTBALL", "BASKETBALL", "TENNIS", "SOCCER"}

	// Test that patterns with mixed sets and topics
	tests := []struct {
		name              string
		thatContext       string
		thatPattern       string
		shouldMatch       bool
		expectedWildcards map[string]string
		description       string
	}{
		{
			name:              "Mixed set and topic match",
			thatContext:       "I LIKE RED FOOTBALL",
			thatPattern:       "I LIKE <set>COLORS</set> <topic>SPORTS</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "RED", "that_star2": "FOOTBALL"},
			description:       "Should match with mixed set and topic wildcards",
		},
		{
			name:              "Mixed with other wildcards",
			thatContext:       "I LIKE RED AND SOMETHING FOOTBALL",
			thatPattern:       "I LIKE <set>COLORS</set> * <topic>SPORTS</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{"that_star1": "RED", "that_star2": "AND SOMETHING", "that_star3": "FOOTBALL"},
			description:       "Should match with mixed wildcards and other wildcards",
		},
		{
			name:              "No match - set word not in set",
			thatContext:       "I LIKE PURPLE FOOTBALL",
			thatPattern:       "I LIKE <set>COLORS</set> <topic>SPORTS</topic>",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match when set word not in set",
		},
		{
			name:              "No match - topic word not in topic",
			thatContext:       "I LIKE RED CHESS",
			thatPattern:       "I LIKE <set>COLORS</set> <topic>SPORTS</topic>",
			shouldMatch:       false,
			expectedWildcards: nil,
			description:       "Should not match when topic word not in topic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, wildcards := matchThatPatternWithWildcardsWithGolem(g, tt.thatContext, tt.thatPattern)

			if matched != tt.shouldMatch {
				t.Errorf("Expected match %v, got %v", tt.shouldMatch, matched)
			}

			if matched && tt.expectedWildcards != nil {
				for key, expectedValue := range tt.expectedWildcards {
					if actualValue, exists := wildcards[key]; !exists {
						t.Errorf("Expected wildcard %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected wildcard %s = '%s', got '%s'", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

// TestEnhancedThatPatternWithNonExistentSetsAndTopics tests that pattern matching with non-existent sets and topics
func TestEnhancedThatPatternWithNonExistentSetsAndTopics(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test that patterns with non-existent sets and topics
	tests := []struct {
		name              string
		thatContext       string
		thatPattern       string
		shouldMatch       bool
		expectedWildcards map[string]string
		description       string
	}{
		{
			name:              "Non-existent set falls back to wildcard",
			thatContext:       "I LIKE ANYTHING",
			thatPattern:       "I LIKE <set>NONEXISTENT</set>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{},
			description:       "Should fall back to wildcard for non-existent set",
		},
		{
			name:              "Non-existent topic falls back to wildcard",
			thatContext:       "I PLAY ANYTHING",
			thatPattern:       "I PLAY <topic>NONEXISTENT</topic>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{},
			description:       "Should fall back to wildcard for non-existent topic",
		},
		{
			name:              "Empty set falls back to wildcard",
			thatContext:       "I LIKE ANYTHING",
			thatPattern:       "I LIKE <set>EMPTY</set>",
			shouldMatch:       true,
			expectedWildcards: map[string]string{},
			description:       "Should fall back to wildcard for empty set",
		},
	}

	// Add empty set for testing
	kb.Sets["EMPTY"] = []string{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, wildcards := matchThatPatternWithWildcardsWithGolem(g, tt.thatContext, tt.thatPattern)

			if matched != tt.shouldMatch {
				t.Errorf("Expected match %v, got %v", tt.shouldMatch, matched)
			}

			if matched && tt.expectedWildcards != nil {
				for key, expectedValue := range tt.expectedWildcards {
					if actualValue, exists := wildcards[key]; !exists {
						t.Errorf("Expected wildcard %s not found", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected wildcard %s = '%s', got '%s'", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

// TestEnhancedThatPatternCaching tests that pattern matching caching works with sets and topics
func TestEnhancedThatPatternCaching(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Add test sets
	kb.Sets["COLORS"] = []string{"RED", "BLUE", "GREEN"}

	// Test the same pattern multiple times to verify caching
	thatContext := "I LIKE RED"
	thatPattern := "I LIKE <set>COLORS</set>"

	// First match
	matched1, wildcards1 := matchThatPatternWithWildcardsWithGolem(g, thatContext, thatPattern)
	if !matched1 {
		t.Error("First match should succeed")
	}

	// Second match (should use cache)
	matched2, wildcards2 := matchThatPatternWithWildcardsWithGolem(g, thatContext, thatPattern)
	if !matched2 {
		t.Error("Second match should succeed")
	}

	// Verify wildcards are the same
	if wildcards1["that_star1"] != wildcards2["that_star1"] {
		t.Error("Cached wildcards should be the same")
	}
}

// TestEnhancedThatPatternRegexGeneration tests that regex generation works correctly with sets and topics
func TestEnhancedThatPatternRegexGeneration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Add test sets and topics
	kb.Sets["COLORS"] = []string{"RED", "BLUE", "GREEN"}
	kb.Topics["SPORTS"] = []string{"FOOTBALL", "BASKETBALL"}

	// Test regex generation
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "Set pattern",
			pattern:  "I LIKE <set>COLORS</set>",
			expected: "I\\s*LIKE\\s*(RED|BLUE|GREEN)",
		},
		{
			name:     "Topic pattern",
			pattern:  "I PLAY <topic>SPORTS</topic>",
			expected: "I\\s*PLAY\\s*(FOOTBALL|BASKETBALL)",
		},
		{
			name:     "Mixed pattern",
			pattern:  "I LIKE <set>COLORS</set> <topic>SPORTS</topic>",
			expected: "I\\s*LIKE\\s*(RED|BLUE|GREEN)\\s*(FOOTBALL|BASKETBALL)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex := thatPatternToRegexWithSetsAndTopics(g, tt.pattern, kb)
			// Remove the case insensitive flag and word boundary for comparison
			regex = strings.TrimPrefix(regex, "(?i)")
			regex = strings.TrimSuffix(regex, "$")

			// The regex should contain the expected alternation
			if !strings.Contains(regex, tt.expected) {
				t.Errorf("Expected regex to contain '%s', got '%s'", tt.expected, regex)
			}
		})
	}
}
