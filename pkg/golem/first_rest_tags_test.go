package golem

import (
	"strings"
	"testing"
	"time"
)

// TestFirstTagProcessing tests the <first> tag functionality
func TestFirstTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	// Ensure knowledge base is initialized
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        g.createSession("test_session"),
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic first element",
			template: "<first>apple banana cherry</first>",
			expected: "apple",
		},
		{
			name:     "Single element",
			template: "<first>apple</first>",
			expected: "apple",
		},
		{
			name:     "Empty content",
			template: "<first></first>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<first>   </first>",
			expected: "",
		},
		{
			name:     "Multiple spaces",
			template: "<first>  apple  banana  cherry  </first>",
			expected: "apple",
		},
		{
			name:     "Numbers",
			template: "<first>1 2 3 4 5</first>",
			expected: "1",
		},
		{
			name:     "Mixed content",
			template: "<first>hello world 123 test</first>",
			expected: "hello",
		},
		{
			name:     "Special characters",
			template: "<first>@#$ %^& *()</first>",
			expected: "@#$",
		},
		{
			name:     "Unicode characters",
			template: "<first>café naïve résumé</first>",
			expected: "café",
		},
		{
			name:     "With variables",
			template: "<first><get name=\"list\"></get></first>",
			expected: "first",
		},
		{
			name:     "Nested first tags",
			template: "<first><first>apple banana cherry</first></first>",
			expected: "apple",
		},
		{
			name:     "First with other tags",
			template: "<first><uppercase>hello world</uppercase></first>",
			expected: "HELLO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up variables if needed
			if tt.name == "With variables" {
				g.ProcessTemplateWithContext(`<set name="list">first second third</set>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, ctx.Session)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestRestTagProcessing tests the <rest> tag functionality
func TestRestTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	// Ensure knowledge base is initialized
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        g.createSession("test_session"),
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic rest elements",
			template: "<rest>apple banana cherry</rest>",
			expected: "banana cherry",
		},
		{
			name:     "Two elements",
			template: "<rest>apple banana</rest>",
			expected: "banana",
		},
		{
			name:     "Single element",
			template: "<rest>apple</rest>",
			expected: "",
		},
		{
			name:     "Empty content",
			template: "<rest></rest>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<rest>   </rest>",
			expected: "",
		},
		{
			name:     "Multiple spaces",
			template: "<rest>  apple  banana  cherry  </rest>",
			expected: "banana cherry",
		},
		{
			name:     "Numbers",
			template: "<rest>1 2 3 4 5</rest>",
			expected: "2 3 4 5",
		},
		{
			name:     "Mixed content",
			template: "<rest>hello world 123 test</rest>",
			expected: "world 123 test",
		},
		{
			name:     "Special characters",
			template: "<rest>@#$ %^& *()</rest>",
			expected: "%^& *()",
		},
		{
			name:     "Unicode characters",
			template: "<rest>café naïve résumé</rest>",
			expected: "naïve résumé",
		},
		{
			name:     "With variables",
			template: "<rest><get name=\"list\"></get></rest>",
			expected: "second third",
		},
		{
			name:     "Nested rest tags",
			template: "<rest><rest>apple banana cherry</rest></rest>",
			expected: "cherry",
		},
		{
			name:     "Rest with other tags",
			template: "<rest><uppercase>hello world test</uppercase></rest>",
			expected: "WORLD TEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up variables if needed
			if tt.name == "With variables" {
				g.ProcessTemplateWithContext(`<set name="list">first second third</set>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, ctx.Session)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestFirstRestIntegration tests integration of <first> and <rest> tags
func TestFirstRestIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	// Ensure knowledge base is initialized
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        g.createSession("test_session"),
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "First and rest together",
			template: "<first>apple banana cherry</first> and <rest>apple banana cherry</rest>",
			expected: "apple and banana cherry",
		},
		{
			name:     "Nested first and rest",
			template: "<first><rest>apple banana cherry</rest></first>",
			expected: "banana",
		},
		{
			name:     "Rest of first",
			template: "<rest><first>apple banana cherry</first></rest>",
			expected: "",
		},
		{
			name:     "Complex nesting",
			template: "<first><rest><first>apple banana cherry</first></rest></first>",
			expected: "",
		},
		{
			name:     "With variables and processing",
			template: "<first><get name=\"list\"></get></first> - <rest><get name=\"list\"></get></rest>",
			expected: "first - second third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up variables if needed
			if tt.name == "With variables and processing" {
				g.ProcessTemplateWithContext(`<set name="list">first second third</set>`, map[string]string{}, ctx.Session)
			}

			// Process through the full template pipeline
			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, ctx.Session)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestFirstRestWithTemplateProcessing tests <first> and <rest> tags through the full template processing pipeline
func TestFirstRestWithTemplateProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	// Ensure knowledge base is initialized
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        g.createSession("test_session"),
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic template processing",
			template: "<first>apple banana cherry</first>",
			expected: "apple",
		},
		{
			name:     "With variable resolution",
			template: "<first><get name=\"list\"></get></first>",
			expected: "first",
		},
		{
			name:     "With text processing",
			template: "<first><uppercase>hello world</uppercase></first>",
			expected: "HELLO",
		},
		{
			name:     "With formatting",
			template: "<first><formal>hello world test</formal></first>",
			expected: "Hello",
		},
		{
			name:     "Complex processing chain",
			template: "<first><uppercase><formal>hello world test</formal></uppercase></first>",
			expected: "HELLO",
		},
		{
			name:     "Rest with processing",
			template: "<rest><uppercase>hello world test</uppercase></rest>",
			expected: "WORLD TEST",
		},
		{
			name:     "Both first and rest in template",
			template: "First: <first><get name=\"list\"></get></first>, Rest: <rest><get name=\"list\"></get></rest>",
			expected: "First: first, Rest: second third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up variables if needed
			if strings.Contains(tt.template, "<get name=\"list\">") {
				g.ProcessTemplateWithContext(`<set name="list">first second third</set>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, ctx.Session)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestFirstRestEdgeCases tests edge cases for <first> and <rest> tags
func TestFirstRestEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	// Ensure knowledge base is initialized
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        g.createSession("test_session"),
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Malformed first tag",
			template: "<first>apple banana cherry",
			expected: "<first>apple banana cherry",
		},
		{
			name:     "Malformed rest tag",
			template: "<rest>apple banana cherry",
			expected: "<rest>apple banana cherry",
		},
		{
			name:     "Empty first tag",
			template: "<first></first>",
			expected: "",
		},
		{
			name:     "Empty rest tag",
			template: "<rest></rest>",
			expected: "",
		},
		{
			name:     "Nested malformed tags",
			template: "<first><rest>apple banana</first></rest>",
			expected: "<first><rest>apple banana</first></rest>",
		},
		{
			name:     "Very long content",
			template: "<first>" + strings.Repeat("word ", 1000) + "</first>",
			expected: "word",
		},
		{
			name:     "Special characters only",
			template: "<first>!@#$%^&*()</first>",
			expected: "!@#$%^&*()",
		},
		{
			name:     "Numbers and symbols",
			template: "<rest>123 456 789 !@# $%^</rest>",
			expected: "456 789 !@# $%^",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, ctx.Session)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestFirstRestPerformance tests performance of <first> and <rest> tags
func TestFirstRestPerformance(t *testing.T) {
	g := NewForTesting(t, false)
	// Ensure knowledge base is initialized
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        g.createSession("test_session"),
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	// Test with large content
	largeContent := strings.Repeat("word ", 10000)
	template := "<first>" + largeContent + "</first>"

	start := time.Now()
	result := g.ProcessTemplateWithContext(template, map[string]string{}, ctx.Session)
	duration := time.Since(start)

	if result != "word" {
		t.Errorf("Expected 'word', got %q", result)
	}

	if duration > 100*time.Millisecond {
		t.Errorf("Processing took too long: %v", duration)
	}
}
