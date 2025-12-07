package golem

import (
	"regexp"
	"strings"
	"testing"
)

// TestUnlearnTagProcessing tests basic unlearn tag processing
func TestUnlearnTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
		{Pattern: "GOODBYE", Template: "Goodbye! Have a great day!"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test unlearn tag processing
	template := `<unlearn>
		<category>
			<pattern>GOODBYE</pattern>
			<template>Goodbye! Have a great day!</template>
		</category>
	</unlearn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The unlearn tag should be removed after processing
	if strings.Contains(result, "<unlearn>") || strings.Contains(result, "</unlearn>") {
		t.Errorf("Unlearn tag not removed from template: %s", result)
	}

	// Check if the category was removed
	if len(kb.Categories) != 1 { // Only 1 category should remain
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}

	// Check if the pattern was removed from index
	normalizedPattern := NormalizePattern("GOODBYE")
	if _, exists := kb.Patterns[normalizedPattern]; exists {
		t.Errorf("Pattern should have been removed from index: %s", normalizedPattern)
	}

	// Check that HELLO pattern still exists
	helloPattern := NormalizePattern("HELLO")
	if _, exists := kb.Patterns[helloPattern]; !exists {
		t.Errorf("HELLO pattern should still exist: %s", helloPattern)
	}
}

// TestUnlearnfTagProcessing tests unlearnf tag processing
func TestUnlearnfTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
		{Pattern: "GOODBYE", Template: "Goodbye! Have a great day!"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test unlearnf tag processing
	template := `<unlearnf>
		<category>
			<pattern>GOODBYE</pattern>
			<template>Goodbye! Have a great day!</template>
		</category>
	</unlearnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The unlearnf tag should be removed after processing
	if strings.Contains(result, "<unlearnf>") || strings.Contains(result, "</unlearnf>") {
		t.Errorf("Unlearnf tag not removed from template: %s", result)
	}

	// Check if the category was removed
	if len(kb.Categories) != 1 { // Only 1 category should remain
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}

	// Check if the pattern was removed from index
	normalizedPattern := NormalizePattern("GOODBYE")
	if _, exists := kb.Patterns[normalizedPattern]; exists {
		t.Errorf("Pattern should have been removed from index: %s", normalizedPattern)
	}
}

// TestUnlearnWithMultipleCategories tests unlearning multiple categories at once
func TestUnlearnWithMultipleCategories(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
		{Pattern: "GOODBYE", Template: "Goodbye! Have a great day!"},
		{Pattern: "THANKS", Template: "You're welcome!"},
		{Pattern: "HELP", Template: "I can help you with that."},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test unlearning multiple categories
	template := `<unlearn>
		<category>
			<pattern>GOODBYE</pattern>
			<template>Goodbye! Have a great day!</template>
		</category>
		<category>
			<pattern>THANKS</pattern>
			<template>You're welcome!</template>
		</category>
	</unlearn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The unlearn tag should be removed after processing
	if strings.Contains(result, "<unlearn>") || strings.Contains(result, "</unlearn>") {
		t.Errorf("Unlearn tag not removed from template: %s", result)
	}

	// Check if the categories were removed
	if len(kb.Categories) != 2 { // Only 2 categories should remain
		t.Errorf("Expected 2 categories, got %d", len(kb.Categories))
	}

	// Check if the patterns were removed from index
	goodbyePattern := NormalizePattern("GOODBYE")
	if _, exists := kb.Patterns[goodbyePattern]; exists {
		t.Errorf("GOODBYE pattern should have been removed from index")
	}

	thanksPattern := NormalizePattern("THANKS")
	if _, exists := kb.Patterns[thanksPattern]; exists {
		t.Errorf("THANKS pattern should have been removed from index")
	}

	// Check that remaining patterns still exist
	helloPattern := NormalizePattern("HELLO")
	if _, exists := kb.Patterns[helloPattern]; !exists {
		t.Errorf("HELLO pattern should still exist")
	}

	helpPattern := NormalizePattern("HELP")
	if _, exists := kb.Patterns[helpPattern]; !exists {
		t.Errorf("HELP pattern should still exist")
	}
}

// TestUnlearnWithWildcards tests unlearning with wildcard patterns
func TestUnlearnWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories with wildcards
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
		{Pattern: "HELLO *", Template: "Hello <star/>! How are you?"},
		{Pattern: "GOODBYE *", Template: "Goodbye <star/>! Have a great day!"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test unlearning wildcard pattern
	template := `<unlearn>
		<category>
			<pattern>HELLO *</pattern>
			<template>Hello <star/>! How are you?</template>
		</category>
	</unlearn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The unlearn tag should be removed after processing
	if strings.Contains(result, "<unlearn>") || strings.Contains(result, "</unlearn>") {
		t.Errorf("Unlearn tag not removed from template: %s", result)
	}

	// Check if the wildcard category was removed
	if len(kb.Categories) != 2 { // Only 2 categories should remain
		t.Errorf("Expected 2 categories, got %d", len(kb.Categories))
	}

	// Check if the wildcard pattern was removed from index
	wildcardPattern := NormalizePattern("HELLO *")
	if _, exists := kb.Patterns[wildcardPattern]; exists {
		t.Errorf("Wildcard pattern should have been removed from index: %s", wildcardPattern)
	}

	// Check that other patterns still exist
	helloPattern := NormalizePattern("HELLO")
	if _, exists := kb.Patterns[helloPattern]; !exists {
		t.Errorf("HELLO pattern should still exist")
	}

	goodbyePattern := NormalizePattern("GOODBYE *")
	if _, exists := kb.Patterns[goodbyePattern]; !exists {
		t.Errorf("GOODBYE * pattern should still exist")
	}
}

// TestUnlearnErrorHandling tests error handling in unlearn processing
func TestUnlearnErrorHandling(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test unlearning non-existent category
	template := `<unlearn>
		<category>
			<pattern>NONEXISTENT</pattern>
			<template>This doesn't exist</template>
		</category>
	</unlearn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The unlearn tag should be removed even on error
	if strings.Contains(result, "<unlearn>") || strings.Contains(result, "</unlearn>") {
		t.Errorf("Unlearn tag not removed from template: %s", result)
	}

	// No categories should be removed
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}
}

// TestUnlearnWithEmptyContent tests unlearning with empty content
func TestUnlearnWithEmptyContent(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test unlearning with empty content
	template := `<unlearn></unlearn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The unlearn tag should be removed after processing
	if strings.Contains(result, "<unlearn>") || strings.Contains(result, "</unlearn>") {
		t.Errorf("Unlearn tag not removed from template: %s", result)
	}

	// No categories should be removed
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}
}

// TestUnlearnIntegration tests integration with learning system
func TestUnlearnIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// First learn a category
	learnTemplate := `<learn>
		<category>
			<pattern>LEARNED PATTERN</pattern>
			<template>This was learned</template>
		</category>
	</learn>`

	_ = g.ProcessTemplate(learnTemplate, make(map[string]string))

	// Check if category was learned
	if len(kb.Categories) != 2 {
		t.Errorf("Expected 2 categories after learning, got %d", len(kb.Categories))
	}

	learnedPattern := NormalizePattern("LEARNED PATTERN")
	if _, exists := kb.Patterns[learnedPattern]; !exists {
		t.Errorf("Learned pattern should exist: %s", learnedPattern)
	}

	// Now unlearn the category
	unlearnTemplate := `<unlearn>
		<category>
			<pattern>LEARNED PATTERN</pattern>
			<template>This was learned</template>
		</category>
	</unlearn>`

	_ = g.ProcessTemplate(unlearnTemplate, make(map[string]string))

	// Check if category was unlearned
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category after unlearning, got %d", len(kb.Categories))
	}

	if _, exists := kb.Patterns[learnedPattern]; exists {
		t.Errorf("Learned pattern should have been removed: %s", learnedPattern)
	}
}

// TestUnlearnfIntegration tests integration with persistent learning system
func TestUnlearnfIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// First learn a persistent category
	learnfTemplate := `<learnf>
		<category>
			<pattern>PERSISTENT PATTERN</pattern>
			<template>This was learned persistently</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(learnfTemplate, make(map[string]string))

	// Check if category was learned
	if len(kb.Categories) != 2 {
		t.Errorf("Expected 2 categories after learning, got %d", len(kb.Categories))
	}

	persistentPattern := NormalizePattern("PERSISTENT PATTERN")
	if _, exists := kb.Patterns[persistentPattern]; !exists {
		t.Errorf("Persistent pattern should exist: %s", persistentPattern)
	}

	// Now unlearn the persistent category
	unlearnfTemplate := `<unlearnf>
		<category>
			<pattern>PERSISTENT PATTERN</pattern>
			<template>This was learned persistently</template>
		</category>
	</unlearnf>`

	_ = g.ProcessTemplate(unlearnfTemplate, make(map[string]string))

	// Check if category was unlearned
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category after unlearning, got %d", len(kb.Categories))
	}

	if _, exists := kb.Patterns[persistentPattern]; exists {
		t.Errorf("Persistent pattern should have been removed: %s", persistentPattern)
	}
}

// TestUnlearnWithSessionContext tests unlearn functionality with session context
func TestUnlearnWithSessionContext(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
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
		ID:        "test_session",
		Variables: make(map[string]string),
		History:   []string{},
	}

	// Test unlearn with session context
	template := `<unlearn>
		<category>
			<pattern>HELLO</pattern>
			<template>Hello! How can I help you?</template>
		</category>
	</unlearn>`

	ctx := &VariableContext{
		Session: session,
	}

	result := g.processTemplateWithContext(template, make(map[string]string), ctx)

	// The unlearn tag should be removed after processing
	if strings.Contains(result, "<unlearn>") || strings.Contains(result, "</unlearn>") {
		t.Errorf("Unlearn tag not removed from template: %s", result)
	}

	// Check if the category was removed
	if len(kb.Categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(kb.Categories))
	}
}

// TestUnlearnRegexPatterns tests the regex patterns used for extracting unlearn tags
func TestUnlearnRegexPatterns(t *testing.T) {
	// Test the basic unlearn regex pattern (same as used in aiml_native.go)
	unlearnRegex := regexp.MustCompile(`(?s)<unlearn[^>]*>(.*?)</unlearn>`)

	// Test cases for unlearn tag extraction
	testCases := []struct {
		name            string
		content         string
		expected        int // expected number of matches
		expectedContent string
	}{
		{
			name:            "Single unlearn tag with category",
			content:         `<unlearn><category><pattern>TEST</pattern><template>Test response</template></category></unlearn>`,
			expected:        1,
			expectedContent: "<category><pattern>TEST</pattern><template>Test response</template></category>",
		},
		{
			name:            "Multiple unlearn tags",
			content:         `<unlearn><category><pattern>TEST1</pattern><template>Test1</template></category></unlearn><unlearn><category><pattern>TEST2</pattern><template>Test2</template></category></unlearn>`,
			expected:        2,
			expectedContent: "",
		},
		{
			name:            "Empty unlearn tag",
			content:         `<unlearn></unlearn>`,
			expected:        1,
			expectedContent: "",
		},
		{
			name:            "Unlearn tag with whitespace only",
			content:         `<unlearn>   </unlearn>`,
			expected:        1,
			expectedContent: "",
		},
		{
			name:            "No unlearn tags",
			content:         `<learn><category><pattern>TEST</pattern><template>Test</template></category></learn>`,
			expected:        0,
			expectedContent: "",
		},
		{
			name:            "Nested unlearn tags",
			content:         `<unlearn><unlearn><category><pattern>TEST</pattern><template>Test</template></category></unlearn></unlearn>`,
			expected:        1,
			expectedContent: "<unlearn><category><pattern>TEST</pattern><template>Test</template></category>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := unlearnRegex.FindAllStringSubmatch(tc.content, -1)
			if len(matches) != tc.expected {
				t.Errorf("Expected %d matches, got %d", tc.expected, len(matches))
			}

			if tc.expected > 0 && tc.expectedContent != "" {
				if len(matches) > 0 && len(matches[0]) > 1 {
					content := strings.TrimSpace(matches[0][1])
					if content != tc.expectedContent {
						t.Errorf("Expected content '%s', got '%s'", tc.expectedContent, content)
					}
				}
			}
		})
	}
}

// TestUnlearnfRegexPatterns tests the regex patterns used for extracting unlearnf tags
func TestUnlearnfRegexPatterns(t *testing.T) {
	// Test the basic unlearnf regex pattern (same as used in aiml_native.go)
	unlearnfRegex := regexp.MustCompile(`(?s)<unlearnf[^>]*>(.*?)</unlearnf>`)

	// Test cases for unlearnf tag extraction
	testCases := []struct {
		name            string
		content         string
		expected        int // expected number of matches
		expectedContent string
	}{
		{
			name:            "Single unlearnf tag with category",
			content:         `<unlearnf><category><pattern>TEST</pattern><template>Test response</template></category></unlearnf>`,
			expected:        1,
			expectedContent: "<category><pattern>TEST</pattern><template>Test response</template></category>",
		},
		{
			name:            "Multiple unlearnf tags",
			content:         `<unlearnf><category><pattern>TEST1</pattern><template>Test1</template></category></unlearnf><unlearnf><category><pattern>TEST2</pattern><template>Test2</template></category></unlearnf>`,
			expected:        2,
			expectedContent: "",
		},
		{
			name:            "Empty unlearnf tag",
			content:         `<unlearnf></unlearnf>`,
			expected:        1,
			expectedContent: "",
		},
		{
			name:            "No unlearnf tags",
			content:         `<learnf><category><pattern>TEST</pattern><template>Test</template></category></learnf>`,
			expected:        0,
			expectedContent: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := unlearnfRegex.FindAllStringSubmatch(tc.content, -1)
			if len(matches) != tc.expected {
				t.Errorf("Expected %d matches, got %d", tc.expected, len(matches))
			}

			if tc.expected > 0 && tc.expectedContent != "" {
				if len(matches) > 0 && len(matches[0]) > 1 {
					content := strings.TrimSpace(matches[0][1])
					if content != tc.expectedContent {
						t.Errorf("Expected content '%s', got '%s'", tc.expectedContent, content)
					}
				}
			}
		})
	}
}

// TestUnlearnRegexMultilineHandling tests multiline handling in unlearn regex
func TestUnlearnRegexMultilineHandling(t *testing.T) {
	unlearnRegex := regexp.MustCompile(`(?s)<unlearn[^>]*>(.*?)</unlearn>`)

	multilineContent := `<unlearn>
		<category>
			<pattern>MULTILINE TEST</pattern>
			<template>This is a multiline
			response that spans
			multiple lines</template>
		</category>
	</unlearn>`

	matches := unlearnRegex.FindAllStringSubmatch(multilineContent, -1)
	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}

	if len(matches) > 0 && len(matches[0]) > 1 {
		content := strings.TrimSpace(matches[0][1])
		if !strings.Contains(content, "MULTILINE TEST") {
			t.Errorf("Expected multiline content to contain 'MULTILINE TEST'")
		}
		if !strings.Contains(content, "multiple lines") {
			t.Errorf("Expected multiline content to contain 'multiple lines'")
		}
	}
}

// TestUnlearnRegexEdgeCases tests edge cases in unlearn regex
func TestUnlearnRegexEdgeCases(t *testing.T) {
	unlearnRegex := regexp.MustCompile(`(?s)<unlearn[^>]*>(.*?)</unlearn>`)

	testCases := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "Unlearn tag with special characters",
			content:  `<unlearn><category><pattern>TEST & SPECIAL</pattern><template>Test & response</template></category></unlearn>`,
			expected: 1,
		},
		{
			name:     "Unlearn tag with CDATA",
			content:  `<unlearn><![CDATA[<category><pattern>TEST</pattern><template>Test</template></category>]]></unlearn>`,
			expected: 1,
		},
		{
			name:     "Unlearn tag with comments",
			content:  `<unlearn><!-- comment --><category><pattern>TEST</pattern><template>Test</template></category></unlearn>`,
			expected: 1,
		},
		{
			name:     "Malformed unlearn tag (missing closing)",
			content:  `<unlearn><category><pattern>TEST</pattern><template>Test</template></category>`,
			expected: 0,
		},
		{
			name:     "Unlearn tag with attributes",
			content:  `<unlearn type="test"><category><pattern>TEST</pattern><template>Test</template></category></unlearn>`,
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := unlearnRegex.FindAllStringSubmatch(tc.content, -1)
			if len(matches) != tc.expected {
				t.Errorf("Expected %d matches, got %d", len(matches), tc.expected)
			}
		})
	}
}
