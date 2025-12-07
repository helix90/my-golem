package golem

import (
	"testing"
	"time"
)

// TestTreeProcessorSRTagBasic tests basic <sr/> tag processing
func TestTreeProcessorSRTagBasic(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base with patterns
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hi there!"},
		{Pattern: "GOODBYE", Template: "See you later!"},
		{Pattern: "HOW ARE YOU", Template: "I'm doing great, thanks!"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session with star1 wildcard
	session := &ChatSession{
		ID:           "test-session",
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

	tests := []struct {
		name     string
		template string
		star1    string
		expected string
	}{
		{
			name:     "SR tag with HELLO",
			template: "You said: <sr/>",
			star1:    "HELLO",
			expected: "You said: Hi there!",
		},
		{
			name:     "SR tag with GOODBYE",
			template: "Response: <sr/>",
			star1:    "GOODBYE",
			expected: "Response: See you later!",
		},
		{
			name:     "SR tag with HOW ARE YOU",
			template: "<sr/>",
			star1:    "HOW ARE YOU",
			expected: "I'm doing great, thanks!",
		},
		{
			name:     "Multiple SR tags",
			template: "<sr/> and <sr/>",
			star1:    "HELLO",
			expected: "Hi there! and Hi there!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update star1 in session
			session.Variables["star1"] = tt.star1

			// Create tree processor
			tp := NewTreeProcessor(g)

			// Parse template into AST
			parser := NewASTParser(tt.template)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			// Process the AST
			tp.ctx = ctx
			result := tp.processNode(ast)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorSRTagNoMatch tests SR tag when pattern doesn't match
func TestTreeProcessorSRTagNoMatch(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base with limited patterns
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hi there!"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session with non-matching wildcard
	session := &ChatSession{
		ID:           "test-session",
		Variables:    map[string]string{"star1": "NONEXISTENT"},
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

	template := "You said: <sr/>"
	expected := "You said: "

	// Create tree processor
	tp := NewTreeProcessor(g)

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Process the AST
	tp.ctx = ctx
	result := tp.processNode(ast)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTreeProcessorSRTagNoWildcard tests SR tag when no wildcard is available
func TestTreeProcessorSRTagNoWildcard(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session without star1 wildcard
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
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

	template := "You said: <sr/>"
	expected := "You said: "

	// Create tree processor
	tp := NewTreeProcessor(g)

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Process the AST
	tp.ctx = ctx
	result := tp.processNode(ast)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTreeProcessorSRTagNoKnowledgeBase tests SR tag without knowledge base
func TestTreeProcessorSRTagNoKnowledgeBase(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with star1 wildcard
	session := &ChatSession{
		ID:           "test-session",
		Variables:    map[string]string{"star1": "HELLO"},
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
	}

	// Create variable context without knowledge base
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
		Topic:          "",
		KnowledgeBase:  nil, // No KB
		RecursionDepth: 0,
	}

	template := "You said: <sr/>"
	expected := "You said: "

	// Create tree processor
	tp := NewTreeProcessor(g)

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Process the AST
	tp.ctx = ctx
	result := tp.processNode(ast)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTreeProcessorSRTagNoSession tests SR tag without session
func TestTreeProcessorSRTagNoSession(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create variable context without session
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        nil, // No session
		Topic:          "",
		KnowledgeBase:  kb,
		RecursionDepth: 0,
	}

	template := "You said: <sr/>"
	expected := "You said: "

	// Create tree processor
	tp := NewTreeProcessor(g)

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Process the AST
	tp.ctx = ctx
	result := tp.processNode(ast)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTreeProcessorSRTagRecursion tests SR tag recursion behavior
func TestTreeProcessorSRTagRecursion(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base with recursive patterns
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{Pattern: "LEVEL1", Template: "Processing level 1: <sr/>"},
		{Pattern: "LEVEL2", Template: "Processing level 2: DONE"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session with star1 wildcard
	session := &ChatSession{
		ID:           "test-session",
		Variables:    map[string]string{"star1": "LEVEL1"},
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

	// Note: This test shows that SR will look up LEVEL1, which has an SR tag,
	// but that SR tag won't have a star1 set, so it returns empty (AIML spec compliant)
	template := "Start: <sr/>"

	// Create tree processor
	tp := NewTreeProcessor(g)

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Process the AST
	tp.ctx = ctx
	result := tp.processNode(ast)

	// The SR in LEVEL1's template will not have star1, so it returns empty string
	// Tree processor trims trailing whitespace
	expected := "Start: Processing level 1:"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTreeProcessorSRTagWithNestedTags tests SR tag with other nested tags
func TestTreeProcessorSRTagWithNestedTags(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hi there!"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session with star1 wildcard
	session := &ChatSession{
		ID:           "test-session",
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "SR with uppercase",
			template: "<uppercase><sr/></uppercase>",
			expected: "HI THERE!",
		},
		{
			name:     "SR with lowercase",
			template: "<lowercase><sr/></lowercase>",
			expected: "hi there!",
		},
		{
			name:     "SR with think and set",
			template: "<think><set name=\"response\"><sr/></set></think>Result: <get name=\"response\"/>",
			expected: "Result: Hi there!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create tree processor
			tp := NewTreeProcessor(g)

			// Parse template into AST
			parser := NewASTParser(tt.template)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			// Process the AST
			tp.ctx = ctx
			result := tp.processNode(ast)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorSRTagMaxRecursionDepth tests SR tag with max recursion depth
func TestTreeProcessorSRTagMaxRecursionDepth(t *testing.T) {
	g := NewForTesting(t, false)

	// Create knowledge base with recursive pattern
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{Pattern: "RECURSIVE", Template: "Loop: <sr/>"},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session with star1 wildcard
	session := &ChatSession{
		ID:           "test-session",
		Variables:    map[string]string{"star1": "RECURSIVE"},
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
	}

	// Create variable context with high recursion depth
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
		Topic:          "",
		KnowledgeBase:  kb,
		RecursionDepth: 99, // Near max
	}

	template := "<sr/>"

	// Create tree processor
	tp := NewTreeProcessor(g)

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Process the AST
	tp.ctx = ctx
	result := tp.processNode(ast)

	// Should hit recursion limit and SR returns empty string (AIML spec compliant)
	// Tree processor trims trailing whitespace
	expected := "Loop:"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
