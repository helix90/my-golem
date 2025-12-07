package golem

import (
	"regexp"
	"strings"
	"testing"
)

// normalizeWhitespace normalizes whitespace for comparison
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	// Trim leading/trailing whitespace
	return strings.TrimSpace(s)
}

// TestLearnTagProcessing tests basic learn tag processing
func TestLearnTagProcessing(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose logging for tests
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
		{Pattern: "TEACH ME *", Template: `<learn>
			<category>
				<pattern>I KNOW *</pattern>
				<template>Yes, I know about <star/>.</template>
			</category>
		</learn>`},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test learn tag processing
	template := `<learn>
		<category>
			<pattern>TEST PATTERN</pattern>
			<template>Test response</template>
		</category>
	</learn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if the category was added
	if len(kb.Categories) != 3 { // 2 initial + 1 learned
		t.Errorf("Expected 3 categories, got %d", len(kb.Categories))
	}

	// Check if the pattern was indexed
	normalizedPattern := NormalizePattern("TEST PATTERN")
	if _, exists := kb.Patterns[normalizedPattern]; !exists {
		t.Errorf("Learned pattern not indexed: %s", normalizedPattern)
	}
}

// TestLearnfTagProcessing tests learnf tag processing
func TestLearnfTagProcessing(t *testing.T) {
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

	// Test learnf tag processing
	template := `<learnf>
		<category>
			<pattern>PERSISTENT PATTERN</pattern>
			<template>Persistent response</template>
		</category>
	</learnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learnf tag should be removed after processing
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// Check if the category was added
	if len(kb.Categories) != 2 { // 1 initial + 1 learned
		t.Errorf("Expected 2 categories, got %d", len(kb.Categories))
	}

	// Check if the pattern was indexed
	normalizedPattern := NormalizePattern("PERSISTENT PATTERN")
	if _, exists := kb.Patterns[normalizedPattern]; !exists {
		t.Errorf("Learned pattern not indexed: %s", normalizedPattern)
	}
}

// TestLearnWithMultipleCategories tests learning multiple categories at once
func TestLearnWithMultipleCategories(t *testing.T) {
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

	// Test learning multiple categories
	template := `<learn>
		<category>
			<pattern>CATEGORY ONE</pattern>
			<template>Response one</template>
		</category>
		<category>
			<pattern>CATEGORY TWO</pattern>
			<template>Response two</template>
		</category>
	</learn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if both categories were added
	if len(kb.Categories) != 3 { // 1 initial + 2 learned
		t.Errorf("Expected 3 categories, got %d", len(kb.Categories))
	}

	// Check if both patterns were indexed
	pattern1 := NormalizePattern("CATEGORY ONE")
	pattern2 := NormalizePattern("CATEGORY TWO")

	if _, exists := kb.Patterns[pattern1]; !exists {
		t.Errorf("Learned pattern 1 not indexed: %s", pattern1)
	}
	if _, exists := kb.Patterns[pattern2]; !exists {
		t.Errorf("Learned pattern 2 not indexed: %s", pattern2)
	}
}

// TestLearnWithWildcards tests learning categories with wildcards
func TestLearnWithWildcards(t *testing.T) {
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

	// Test learning with wildcards
	template := `<learn>
		<category>
			<pattern>I LIKE *</pattern>
			<template>I'm glad you like <star/>!</template>
		</category>
	</learn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if the category was added
	if len(kb.Categories) != 2 { // 1 initial + 1 learned
		t.Errorf("Expected 2 categories, got %d", len(kb.Categories))
	}

	// Test that the learned pattern works
	learnedPattern := NormalizePattern("I LIKE *")
	if _, exists := kb.Patterns[learnedPattern]; !exists {
		t.Errorf("Learned wildcard pattern not indexed: %s", learnedPattern)
	}

	// Test matching the learned pattern
	category, wildcards, err := kb.MatchPattern("I LIKE PIZZA")
	if err != nil {
		t.Errorf("Failed to match learned pattern: %v", err)
	}
	if category == nil {
		t.Error("No category matched for learned pattern")
	}
	if wildcards["star1"] != "PIZZA" {
		t.Errorf("Expected wildcard 'PIZZA', got '%s'", wildcards["star1"])
	}
}

// TestLearnErrorHandling tests error handling in learn processing
func TestLearnErrorHandling(t *testing.T) {
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

	// Test with invalid AIML content
	template := `<learn>
		<invalid>This is not valid AIML</invalid>
	</learn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learn tag should be removed even on error
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template on error: %s", result)
	}

	// No new categories should be added
	if len(kb.Categories) != 1 { // Only initial category
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}
}

// TestLearnWithEmptyContent tests learn tags with empty content
func TestLearnWithEmptyContent(t *testing.T) {
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

	// Test with empty learn content
	template := `<learn></learn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learn tag should be removed
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// No new categories should be added
	if len(kb.Categories) != 1 { // Only initial category
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}
}

// TestLearnIntegration tests learn functionality in a complete scenario
func TestLearnIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
		{Pattern: "TEACH ME *", Template: `<learn>
			<category>
				<pattern>I KNOW *</pattern>
				<template>Yes, I know about <star/>.</template>
			</category>
		</learn>I've learned that pattern!`},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test the teaching scenario by matching the pattern first
	category, wildcards, err := kb.MatchPattern("TEACH ME SOMETHING")
	if err != nil {
		t.Errorf("Failed to match teaching pattern: %v", err)
	}
	if category == nil {
		t.Error("No category matched for teaching pattern")
		return
	}

	// Process the matched template
	response := g.ProcessTemplate(category.Template, wildcards)
	expected := "I've learned that pattern!"
	if !strings.Contains(response, expected) {
		t.Errorf("Expected teaching response, got: %s", response)
	}

	// Check if the new pattern was learned
	learnedPattern := NormalizePattern("I KNOW *")
	if _, exists := kb.Patterns[learnedPattern]; !exists {
		t.Errorf("Learned pattern not found: %s", learnedPattern)
	}

	// Test that the learned pattern works
	learnedCategory, learnedWildcards, err := kb.MatchPattern("I KNOW PIZZA")
	if err != nil {
		t.Errorf("Failed to match learned pattern: %v", err)
	}
	if learnedCategory == nil {
		t.Error("No category matched for learned pattern")
	}
	if learnedWildcards["star1"] != "PIZZA" {
		t.Errorf("Expected wildcard 'PIZZA', got '%s'", learnedWildcards["star1"])
	}
}

// TestLearnfIntegration tests learnf functionality in a complete scenario
func TestLearnfIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add initial categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you?"},
		{Pattern: "SAVE KNOWLEDGE *", Template: `<learnf>
			<category>
				<pattern>SAVED *</pattern>
				<template>I remember: <star/></template>
			</category>
		</learnf>Knowledge saved!`},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Test the saving scenario by matching the pattern first
	category, wildcards, err := kb.MatchPattern("SAVE KNOWLEDGE SOMETHING")
	if err != nil {
		t.Errorf("Failed to match saving pattern: %v", err)
	}
	if category == nil {
		t.Error("No category matched for saving pattern")
		return
	}

	// Process the matched template
	response := g.ProcessTemplate(category.Template, wildcards)
	expected := "Knowledge saved!"
	if !strings.Contains(response, expected) {
		t.Errorf("Expected saving response, got: %s", response)
	}

	// Check if the new pattern was learned
	learnedPattern := NormalizePattern("SAVED *")
	if _, exists := kb.Patterns[learnedPattern]; !exists {
		t.Errorf("Learned pattern not found: %s", learnedPattern)
	}

	// Test that the learned pattern works
	learnedCategory2, learnedWildcards2, err := kb.MatchPattern("SAVED INFORMATION")
	if err != nil {
		t.Errorf("Failed to match learned pattern: %v", err)
	}
	if learnedCategory2 == nil {
		t.Error("No category matched for learned pattern")
	}
	if learnedWildcards2["star1"] != "INFORMATION" {
		t.Errorf("Expected wildcard 'INFORMATION', got '%s'", learnedWildcards2["star1"])
	}
}

// TestLearnWithSessionContext tests learn functionality with session context
func TestLearnWithSessionContext(t *testing.T) {
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

	// Test learn with session context
	template := `<learn>
		<category>
			<pattern>SESSION SPECIFIC *</pattern>
			<template>Session response for <star/></template>
		</category>
	</learn>`

	ctx := &VariableContext{
		Session: session,
	}

	result := g.processTemplateWithContext(template, make(map[string]string), ctx)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if the category was added
	if len(kb.Categories) != 2 { // 1 initial + 1 learned
		t.Errorf("Expected 2 categories, got %d", len(kb.Categories))
	}

	// Test that the learned pattern works
	learnedPattern := NormalizePattern("SESSION SPECIFIC *")
	if _, exists := kb.Patterns[learnedPattern]; !exists {
		t.Errorf("Learned pattern not indexed: %s", learnedPattern)
	}
}

// TestLearnCategoryValidation tests validation of learned categories
func TestLearnCategoryValidation(t *testing.T) {
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

	// Test with invalid category (empty pattern)
	template := `<learn>
		<category>
			<pattern></pattern>
			<template>Invalid response</template>
		</category>
	</learn>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learn tag should be removed even on validation error
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// No new categories should be added due to validation error
	if len(kb.Categories) != 1 { // Only initial category
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}
}

// TestLearnRegexPatterns tests the regex patterns used for extracting learn tags
func TestLearnRegexPatterns(t *testing.T) {
	// Test the basic learn regex pattern (same as used in aiml_native.go)
	learnRegex := regexp.MustCompile(`(?s)<learn[^>]*>(.*?)</learn>`)

	// Test cases for learn tag extraction
	testCases := []struct {
		name         string
		content      string
		expected     int    // expected number of matches
		contentMatch string // expected content in first match
	}{
		{
			name: "Single learn tag with category",
			content: `<learn>
				<category>
					<pattern>TEST PATTERN</pattern>
					<template>Test response</template>
				</category>
			</learn>`,
			expected:     1,
			contentMatch: "TEST PATTERN",
		},
		{
			name: "Multiple learn tags",
			content: `<learn>
				<category>
					<pattern>PATTERN ONE</pattern>
					<template>Response one</template>
				</category>
			</learn>
			<learn>
				<category>
					<pattern>PATTERN TWO</pattern>
					<template>Response two</template>
				</category>
			</learn>`,
			expected:     2,
			contentMatch: "PATTERN ONE",
		},
		{
			name:         "Empty learn tag",
			content:      `<learn></learn>`,
			expected:     1,
			contentMatch: "",
		},
		{
			name:         "Learn tag with whitespace only",
			content:      `<learn>   </learn>`,
			expected:     1,
			contentMatch: "   ",
		},
		{
			name: "No learn tags",
			content: `<category>
				<pattern>NO LEARN</pattern>
				<template>No learn tag</template>
			</category>`,
			expected:     0,
			contentMatch: "",
		},
		{
			name: "Nested learn tags",
			content: `<learn>
				<category>
					<pattern>OUTER PATTERN</pattern>
					<template><learn>
						<category>
							<pattern>INNER PATTERN</pattern>
							<template>Inner response</template>
						</category>
					</learn></template>
				</category>
			</learn>`,
			expected:     1,
			contentMatch: "OUTER PATTERN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := learnRegex.FindAllStringSubmatch(tc.content, -1)

			if len(matches) != tc.expected {
				t.Errorf("Expected %d matches, got %d", tc.expected, len(matches))
			}

			if tc.expected > 0 && len(matches) > 0 && tc.contentMatch != "" {
				// Check that the expected content is found in the first match
				if len(matches[0]) > 1 {
					actualContent := matches[0][1]
					if !strings.Contains(actualContent, tc.contentMatch) {
						t.Errorf("Expected content to contain '%s', got '%s'", tc.contentMatch, actualContent)
					}
				} else {
					t.Error("Match found but no content captured")
				}
			}
		})
	}
}

// TestLearnfRegexPatterns tests the regex patterns used for extracting learnf tags
func TestLearnfRegexPatterns(t *testing.T) {
	// Test the learnf regex pattern (same as used in aiml_native.go)
	learnfRegex := regexp.MustCompile(`(?s)<learnf[^>]*>(.*?)</learnf>`)

	// Test cases for learnf tag extraction
	testCases := []struct {
		name         string
		content      string
		expected     int    // expected number of matches
		contentMatch string // expected content in first match
	}{
		{
			name: "Single learnf tag with category",
			content: `<learnf>
				<category>
					<pattern>PERSISTENT PATTERN</pattern>
					<template>Persistent response</template>
				</category>
			</learnf>`,
			expected:     1,
			contentMatch: "PERSISTENT PATTERN",
		},
		{
			name: "Multiple learnf tags",
			content: `<learnf>
				<category>
					<pattern>PERSISTENT ONE</pattern>
					<template>Persistent one</template>
				</category>
			</learnf>
			<learnf>
				<category>
					<pattern>PERSISTENT TWO</pattern>
					<template>Persistent two</template>
				</category>
			</learnf>`,
			expected:     2,
			contentMatch: "PERSISTENT ONE",
		},
		{
			name:         "Empty learnf tag",
			content:      `<learnf></learnf>`,
			expected:     1,
			contentMatch: "",
		},
		{
			name: "No learnf tags",
			content: `<category>
				<pattern>NO LEARNF</pattern>
				<template>No learnf tag</template>
			</category>`,
			expected:     0,
			contentMatch: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := learnfRegex.FindAllStringSubmatch(tc.content, -1)

			if len(matches) != tc.expected {
				t.Errorf("Expected %d matches, got %d", tc.expected, len(matches))
			}

			if tc.expected > 0 && len(matches) > 0 && tc.contentMatch != "" {
				// Check that the expected content is found in the first match
				if len(matches[0]) > 1 {
					actualContent := matches[0][1]
					if !strings.Contains(actualContent, tc.contentMatch) {
						t.Errorf("Expected content to contain '%s', got '%s'", tc.contentMatch, actualContent)
					}
				} else {
					t.Error("Match found but no content captured")
				}
			}
		})
	}
}

// TestLearnRegexMultilineHandling tests that the regex properly handles multiline content
func TestLearnRegexMultilineHandling(t *testing.T) {
	learnRegex := regexp.MustCompile(`(?s)<learn[^>]*>(.*?)</learn>`)

	// Test multiline content with various line endings and whitespace
	content := `<learn>
		<category>
			<pattern>MULTILINE PATTERN</pattern>
			<template>This is a multiline
			response with
			multiple lines</template>
		</category>
	</learn>`

	matches := learnRegex.FindAllStringSubmatch(content, -1)

	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}

	if len(matches) > 0 && len(matches[0]) > 1 {
		matchedContent := matches[0][1]
		// Should contain the multiline content
		if !strings.Contains(matchedContent, "MULTILINE PATTERN") {
			t.Error("Multiline content not properly captured")
		}
		if !strings.Contains(matchedContent, "multiple lines") {
			t.Error("Multiline template content not properly captured")
		}
	}
}

// TestLearnRegexEdgeCases tests edge cases for learn tag regex patterns
func TestLearnRegexEdgeCases(t *testing.T) {
	learnRegex := regexp.MustCompile(`(?s)<learn[^>]*>(.*?)</learn>`)

	// Test cases for edge cases
	testCases := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "Learn tag with special characters",
			content: `<learn>
				<category>
					<pattern>SPECIAL &lt; &gt; &amp; PATTERN</pattern>
					<template>Response with &quot;quotes&quot;</template>
				</category>
			</learn>`,
			expected: 1,
		},
		{
			name: "Learn tag with CDATA",
			content: `<learn>
				<category>
					<pattern>CDATA PATTERN</pattern>
					<template><![CDATA[This is CDATA content]]></template>
				</category>
			</learn>`,
			expected: 1,
		},
		{
			name: "Learn tag with comments",
			content: `<learn>
				<category>
					<pattern>COMMENT PATTERN</pattern>
					<template>Response <!-- This is a comment --></template>
				</category>
			</learn>`,
			expected: 1,
		},
		{
			name: "Malformed learn tag (missing closing)",
			content: `<learn>
				<category>
					<pattern>MALFORMED</pattern>
					<template>No closing tag</template>
				</category>`,
			expected: 0, // Should not match malformed tags
		},
		{
			name: "Learn tag with attributes",
			content: `<learn version="1.0">
				<category>
					<pattern>ATTRIBUTE PATTERN</pattern>
					<template>Response</template>
				</category>
			</learn>`,
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := learnRegex.FindAllStringSubmatch(tc.content, -1)

			if len(matches) != tc.expected {
				t.Errorf("Expected %d matches, got %d", tc.expected, len(matches))
			}
		})
	}
}
