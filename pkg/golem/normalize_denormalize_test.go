package golem

import (
	"testing"
)

// TestNormalizeTagProcessing tests the <normalize> tag processing
func TestNormalizeTagProcessing(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic normalization",
			template: "<normalize>Hello, World!</normalize>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Normalization with contractions",
			template: "<normalize>I'm happy, aren't you?</normalize>",
			expected: "I AM HAPPY ARE NOT YOU",
		},
		{
			name:     "Normalization with punctuation",
			template: "<normalize>What's up? How are you doing!</normalize>",
			expected: "WHAT IS UP HOW ARE YOU DOING",
		},
		{
			name:     "Normalization with multiple spaces",
			template: "<normalize>Hello    world   with   spaces</normalize>",
			expected: "HELLO WORLD WITH SPACES",
		},
		{
			name:     "Normalization with mixed case",
			template: "<normalize>HeLLo WoRLd</normalize>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Empty normalize tag",
			template: "<normalize></normalize>",
			expected: "",
		},
		{
			name:     "Normalize tag with only whitespace",
			template: "<normalize>   </normalize>",
			expected: "",
		},
		{
			name:     "Multiple normalize tags",
			template: "<normalize>Hello</normalize> <normalize>World</normalize>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Normalize with special characters",
			template: "<normalize>Hello-world_test@example.com</normalize>",
			expected: "HELLO WORLD TEST EXAMPLE COM",
		},
		{
			name:     "Normalize with underscores and hyphens",
			template: "<normalize>test_case-name</normalize>",
			expected: "TEST CASE NAME",
		},
		{
			name:     "Normalize with apostrophes",
			template: "<normalize>don't can't won't</normalize>",
			expected: "DO NOT CANNOT WILL NOT",
		},
		{
			name:     "Normalize with complex contractions",
			template: "<normalize>I'd've done it if I'd known</normalize>",
			expected: "I WOULD HAVE DONE IT IF I HAD KNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDenormalizeTagProcessing tests the <denormalize> tag processing
func TestDenormalizeTagProcessing(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic denormalization",
			template: "<denormalize>HELLO WORLD</denormalize>",
			expected: "Hello world.",
		},
		{
			name:     "Denormalization with multiple sentences",
			template: "<denormalize>HELLO WORLD HOW ARE YOU</denormalize>",
			expected: "Hello world how are you.",
		},
		{
			name:     "Denormalization with existing punctuation",
			template: "<denormalize>HELLO WORLD!</denormalize>",
			expected: "Hello world!",
		},
		{
			name:     "Denormalization with question mark",
			template: "<denormalize>HOW ARE YOU?</denormalize>",
			expected: "How are you?",
		},
		{
			name:     "Denormalization with exclamation",
			template: "<denormalize>GREAT JOB!</denormalize>",
			expected: "Great job!",
		},
		{
			name:     "Empty denormalize tag",
			template: "<denormalize></denormalize>",
			expected: "",
		},
		{
			name:     "Denormalize tag with only whitespace",
			template: "<denormalize>   </denormalize>",
			expected: "",
		},
		{
			name:     "Multiple denormalize tags",
			template: "<denormalize>HELLO</denormalize> <denormalize>WORLD</denormalize>",
			expected: "Hello. World.",
		},
		{
			name:     "Denormalization with mixed case",
			template: "<denormalize>HeLLo WoRLd</denormalize>",
			expected: "Hello world.",
		},
		{
			name:     "Denormalization with multiple spaces",
			template: "<denormalize>HELLO    WORLD</denormalize>",
			expected: "Hello world.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeDenormalizeIntegration tests the integration of normalize and denormalize tags
func TestNormalizeDenormalizeIntegration(t *testing.T) {
	g := NewForTesting(t, false)          // Disable verbose mode for cleaner test output
	g.EnableTreeProcessing() // Enable AST-based processing for nested tag support

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Normalize then denormalize",
			template: "<denormalize><normalize>Hello, World!</normalize></denormalize>",
			expected: "Hello world.",
		},
		{
			name:     "Denormalize then normalize",
			template: "<normalize><denormalize>HELLO WORLD</denormalize></normalize>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Mixed processing order",
			template: "<normalize>Hello</normalize> <denormalize>WORLD</denormalize>",
			expected: "HELLO World.",
		},
		{
			name:     "Nested processing",
			template: "<normalize><denormalize><normalize>Hello, World!</normalize></denormalize></normalize>",
			expected: "HELLO WORLD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeDenormalizeWithWildcards tests normalize and denormalize with wildcards
func TestNormalizeDenormalizeWithWildcards(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name      string
		template  string
		wildcards map[string]string
		expected  string
	}{
		{
			name:      "Normalize with wildcard",
			template:  "<normalize>Hello <star/></normalize>",
			wildcards: map[string]string{"star1": "World!"},
			expected:  "HELLO WORLD",
		},
		{
			name:      "Denormalize with wildcard",
			template:  "<denormalize>HELLO <star/></denormalize>",
			wildcards: map[string]string{"star1": "WORLD"},
			expected:  "Hello world.",
		},
		{
			name:      "Mixed processing with wildcards",
			template:  "<normalize>Hello <star/></normalize> <denormalize>WORLD</denormalize>",
			wildcards: map[string]string{"star1": "World!"},
			expected:  "HELLO WORLD World.",
		},
		{
			name:      "Wildcard in normalize tag",
			template:  "<normalize><star/></normalize>",
			wildcards: map[string]string{"star1": "Hello, World!"},
			expected:  "HELLO WORLD",
		},
		{
			name:      "Wildcard in denormalize tag",
			template:  "<denormalize><star/></denormalize>",
			wildcards: map[string]string{"star1": "HELLO WORLD"},
			expected:  "Hello world.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, tt.wildcards)
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeDenormalizeWithVariables tests normalize and denormalize with variables
func TestNormalizeDenormalizeWithVariables(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output
	kb := NewAIMLKnowledgeBase()

	// Add a category that uses variables
	kb.Categories = []Category{
		{
			Pattern:  "TEST NORMALIZE",
			Template: "<normalize>Hello <get name=\"name\"/></normalize>",
		},
		{
			Pattern:  "TEST DENORMALIZE",
			Template: "<denormalize>HELLO <get name=\"name\"/></denormalize>",
		},
		{
			Pattern:  "TEST MIXED",
			Template: "<normalize>Hello <get name=\"name\"/></normalize> <denormalize>WORLD</denormalize>",
		},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session with variables
	session := &ChatSession{
		ID:        "test-session",
		History:   []string{},
		Variables: make(map[string]string),
	}

	// Set a variable
	session.Variables["name"] = "World!"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normalize with variable",
			input:    "TEST NORMALIZE",
			expected: "HELLO WORLD",
		},
		{
			name:     "Denormalize with variable",
			input:    "TEST DENORMALIZE",
			expected: "Hello world!",
		},
		{
			name:     "Mixed processing with variable",
			input:    "TEST MIXED",
			expected: "HELLO WORLD World.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput() error = %v", err)
			}
			if response != tt.expected {
				t.Errorf("ProcessInput() = %v, want %v", response, tt.expected)
			}
		})
	}
}

// TestNormalizeDenormalizeEdgeCases tests edge cases for normalize and denormalize
func TestNormalizeDenormalizeEdgeCases(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Only punctuation",
			template: "<normalize>!@#$%^&*()</normalize>",
			expected: "",
		},
		{
			name:     "Only spaces",
			template: "<normalize>     </normalize>",
			expected: "",
		},
		{
			name:     "Mixed punctuation and text",
			template: "<normalize>Hello! How are you? I'm fine.</normalize>",
			expected: "HELLO HOW ARE YOU I AM FINE",
		},
		{
			name:     "Numbers and text",
			template: "<normalize>I have 5 cats and 3 dogs!</normalize>",
			expected: "I HAVE 5 CATS AND 3 DOGS",
		},
		{
			name:     "Special characters",
			template: "<normalize>Email: test@example.com Phone: 123-456-7890</normalize>",
			expected: "EMAIL TEST EXAMPLE COM PHONE 123 456 7890",
		},
		{
			name:     "Unicode characters",
			template: "<normalize>Hello 世界!</normalize>",
			expected: "HELLO 世界",
		},
		{
			name:     "Very long text",
			template: "<normalize>This is a very long sentence with many words that should be normalized properly and consistently throughout the entire text content.</normalize>",
			expected: "THIS IS A VERY LONG SENTENCE WITH MANY WORDS THAT SHOULD BE NORMALIZED PROPERLY AND CONSISTENTLY THROUGHOUT THE ENTIRE TEXT CONTENT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeDenormalizePerformance tests performance with multiple tags
func TestNormalizeDenormalizePerformance(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	// Create a template with many normalize and denormalize tags
	template := ""
	expected := ""

	for i := 0; i < 100; i++ {
		template += "<normalize>Hello world</normalize> "
		expected += "HELLO WORLD "
	}

	// Add some denormalize tags
	for i := 0; i < 50; i++ {
		template += "<denormalize>HELLO WORLD</denormalize> "
		expected += "Hello world. "
	}

	expected = expected[:len(expected)-1] // Remove trailing space

	result := g.ProcessTemplate(template, make(map[string]string))
	if result != expected {
		t.Errorf("ProcessTemplate() performance test failed. Expected length: %d, got length: %d", len(expected), len(result))
	}
}
