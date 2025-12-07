package golem

import (
	"strings"
	"testing"
)

func TestASTParser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple text",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "Simple tag",
			input:    "<uppercase>hello</uppercase>",
			expected: "<uppercase>hello</uppercase>",
		},
		{
			name:     "Nested tags",
			input:    "<uppercase><lowercase>HELLO</lowercase></uppercase>",
			expected: "<uppercase><lowercase>HELLO</lowercase></uppercase>",
		},
		{
			name:     "Self-closing tag",
			input:    "Hello <star/> world",
			expected: "Hello <star/> world",
		},
		{
			name:     "Tag with attributes",
			input:    `<set name="test" value="hello">content</set>`,
			expected: `<set name="test" value="hello">content</set>`, // Order may vary due to Go map iteration
		},
		{
			name:     "Complex nested structure",
			input:    `<random><li><uppercase>hello</uppercase></li><li><lowercase>WORLD</lowercase></li></random>`,
			expected: `<random><li><uppercase>hello</uppercase></li><li><lowercase>WORLD</lowercase></li></random>`,
		},
		{
			name:     "Malformed tag",
			input:    "<uppercase>hello</lowercase>",
			expected: "<uppercase>hello</lowercase>", // Should be preserved as text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewASTParser(tt.input)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := ast.String()

			// For "Tag with attributes" test, check that attributes are present rather than exact order
			if tt.name == "Tag with attributes" {
				// Check that the result contains the expected attributes
				if !strings.Contains(result, `name="test"`) || !strings.Contains(result, `value="hello"`) {
					t.Errorf("Expected attributes 'name=\"test\"' and 'value=\"hello\"' in result, got %q", result)
				}
				if !strings.Contains(result, "content") {
					t.Errorf("Expected content 'content' in result, got %q", result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestTreeProcessor(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello there!"},
		{Pattern: "HI", Template: "Hi back!"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       &ChatSession{},
		Topic:         "",
		KnowledgeBase: kb,
	}

	tp := NewTreeProcessor(g)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple text processing",
			template: "Hello world",
			expected: "Hello world",
		},
		{
			name:     "Uppercase tag",
			template: "<uppercase>hello world</uppercase>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Nested case tags",
			template: "<uppercase><lowercase>HELLO WORLD</lowercase></uppercase>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Set and get tags",
			template: "<set name=\"test\">hello</set><get name=\"test\">",
			expected: "hello",
		},
		{
			name:     "Random tag with list items",
			template: "<random><li>option1</li><li>option2</li></random>",
			expected: "option1", // or "option2" - either is valid
		},
		{
			name:     "Complex nested processing",
			template: "<uppercase><set name=\"test\">hello</set></uppercase><get name=\"test\">",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.ProcessTemplate(tt.template, map[string]string{}, ctx)
			if err != nil {
				t.Fatalf("ProcessTemplate error: %v", err)
			}

			// For random tags, just check that we got one of the expected options
			if tt.name == "Random tag with list items" {
				if result != "option1" && result != "option2" {
					t.Errorf("Expected 'option1' or 'option2', got %q", result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestTreeProcessorNestedTags(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello there!"},
		{Pattern: "HI", Template: "Hi back!"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       &ChatSession{},
		Topic:         "",
		KnowledgeBase: kb,
	}

	tp := NewTreeProcessor(g)

	// Test complex nested tag scenarios that would be problematic with regex
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Deeply nested tags",
			template: "<uppercase><lowercase><uppercase>hello</uppercase></lowercase></uppercase>",
			expected: "HELLO",
		},
		{
			name:     "Mixed tag types",
			template: "<set name=\"test\"><uppercase>hello</uppercase></set><get name=\"test\">",
			expected: "HELLO",
		},
		{
			name:     "Self-closing tags in content",
			template: "Hello <star/> world <input/>",
			expected: "Hello  world", // Empty tags, trailing space trimmed by tree processor
		},
		{
			name:     "Tags with attributes and content",
			template: `<set name="test" operation="add">hello</set><set name="test"></set>`,
			expected: "hello", // Set collection get operation returns the added item
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.ProcessTemplate(tt.template, map[string]string{}, ctx)
			if err != nil {
				t.Fatalf("ProcessTemplate error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTreeProcessorPerformance(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello there!"},
		{Pattern: "HI", Template: "Hi back!"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       &ChatSession{},
		Topic:         "",
		KnowledgeBase: kb,
	}

	tp := NewTreeProcessor(g)

	// Test performance with complex nested structure
	complexTemplate := `<uppercase><lowercase><set name="test"><random><li>hello</li><li>world</li></random></set></lowercase></uppercase><get name="test">`

	// Run multiple times to test performance
	for i := 0; i < 100; i++ {
		result, err := tp.ProcessTemplate(complexTemplate, map[string]string{}, ctx)
		if err != nil {
			t.Fatalf("ProcessTemplate error: %v", err)
		}

		// Result should be one of the random options
		if result != "hello" && result != "world" {
			t.Errorf("Expected 'hello' or 'world', got %q", result)
		}
	}
}

func TestTreeProcessorErrorHandling(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello there!"},
		{Pattern: "HI", Template: "Hi back!"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       &ChatSession{},
		Topic:         "",
		KnowledgeBase: kb,
	}

	tp := NewTreeProcessor(g)

	// Test error handling with malformed input
	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Unclosed tag",
			template: "<uppercase>hello",
			expected: "<uppercase>hello", // Should be preserved as text
		},
		{
			name:     "Mismatched tags",
			template: "<uppercase>hello</lowercase>",
			expected: "<uppercase>hello</lowercase>", // Should be preserved as text
		},
		{
			name:     "Empty template",
			template: "",
			expected: "",
		},
		{
			name:     "Only whitespace",
			template: "   \n\t  ",
			expected: "", // Whitespace-only collapses to empty (AIML spec compliant)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.ProcessTemplate(tt.template, map[string]string{}, ctx)
			if err != nil {
				t.Fatalf("ProcessTemplate error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestASTNodeHelpers(t *testing.T) {
	// Test AST node helper methods
	parser := NewASTParser(`<random><li>option1</li><li>option2</li></random>`)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Test FindTagsByName
	liTags := ast.FindTagsByName("li")
	if len(liTags) != 2 {
		t.Errorf("Expected 2 li tags, got %d", len(liTags))
	}

	// Test FindFirstTagByName
	randomTag := ast.FindFirstTagByName("random")
	if randomTag == nil {
		t.Error("Expected to find random tag")
	}

	if randomTag.TagName != "random" {
		t.Errorf("Expected tag name 'random', got %q", randomTag.TagName)
	}

	// Test GetTextContent
	textContent := ast.GetTextContent()
	if textContent != "option1option2" {
		t.Errorf("Expected text content 'option1option2', got %q", textContent)
	}
}
