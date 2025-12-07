package golem

import (
	"testing"
	"time"
)

// TestTreeProcessorThatTagBasic tests basic <that/> tag processing
func TestTreeProcessorThatTagBasic(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with response history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: []string{"Hello there!", "How are you?", "Nice to meet you!"},
	}

	// Create variable context
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
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
			name:     "Self-closing that tag",
			template: "You said: <that/>",
			expected: "You said: Nice to meet you!",
		},
		{
			name:     "That tag without index",
			template: "Previous response: <that></that>",
			expected: "Previous response: Nice to meet you!",
		},
		{
			name:     "Multiple that tags",
			template: "<that/> and then <that/>",
			expected: "Nice to meet you! and then Nice to meet you!",
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

// TestTreeProcessorThatTagWithIndex tests <that> tag with index attribute
func TestTreeProcessorThatTagWithIndex(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with response history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: []string{"First response", "Second response", "Third response"},
	}

	// Create variable context
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
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
			name:     "That tag with index 1 (most recent)",
			template: `Response 1: <that index="1"/>`,
			expected: "Response 1: Third response",
		},
		{
			name:     "That tag with index 2",
			template: `Response 2: <that index="2"/>`,
			expected: "Response 2: Second response",
		},
		{
			name:     "That tag with index 3",
			template: `Response 3: <that index="3"/>`,
			expected: "Response 3: First response",
		},
		{
			name:     "Multiple that tags with different indices",
			template: `Latest: <that index="1"/>, Previous: <that index="2"/>`,
			expected: "Latest: Third response, Previous: Second response",
		},
		{
			name:     "Mix of indexed and non-indexed",
			template: `<that/> was most recent, <that index="2"/> was before`,
			expected: "Third response was most recent, Second response was before",
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

// TestTreeProcessorThatTagEdgeCases tests edge cases for <that> tag
func TestTreeProcessorThatTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name            string
		responseHistory []string
		template        string
		expected        string
		description     string
	}{
		{
			name:            "Empty response history",
			responseHistory: []string{},
			template:        "You said: <that/>",
			expected:        "You said: ",
			description:     "Should return empty string when no history",
		},
		{
			name:            "Index out of bounds (too high)",
			responseHistory: []string{"Only response"},
			template:        `Response: <that index="5"/>`,
			expected:        "Response: ",
			description:     "Should return empty string for out of bounds index",
		},
		{
			name:            "Single response in history",
			responseHistory: []string{"Single response"},
			template:        "You said: <that/>",
			expected:        "You said: Single response",
			description:     "Should handle single response correctly",
		},
		{
			name:            "Invalid index (zero)",
			responseHistory: []string{"Response 1", "Response 2"},
			template:        `Response: <that index="0"/>`,
			expected:        "Response: Response 2",
			description:     "Should default to index 1 for invalid index 0",
		},
		{
			name:            "Invalid index (negative)",
			responseHistory: []string{"Response 1", "Response 2"},
			template:        `Response: <that index="-1"/>`,
			expected:        "Response: Response 2",
			description:     "Should default to index 1 for negative index",
		},
		{
			name:            "Whitespace in response",
			responseHistory: []string{"  Trimmed response  "},
			template:        "You said: <that/>",
			expected:        "You said:   Trimmed response  ",
			description:     "Should preserve whitespace in responses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a session with specified response history
			session := &ChatSession{
				ID:              "test-session",
				Variables:       make(map[string]string),
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				Topic:           "",
				ThatHistory:     make([]string, 0),
				ResponseHistory: tt.responseHistory,
			}

			// Create variable context
			ctx := &VariableContext{
				LocalVars:      make(map[string]string),
				Session:        session,
				Topic:          "",
				KnowledgeBase:  g.aimlKB,
				RecursionDepth: 0,
			}

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
				t.Errorf("%s: Expected '%s', got '%s'", tt.description, tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorThatTagNoSession tests <that> tag when session is nil
func TestTreeProcessorThatTagNoSession(t *testing.T) {
	g := NewForTesting(t, false)

	// Create variable context without a session
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        nil, // No session
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	template := "You said: <that/>"
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

// TestTreeProcessorThatTagNoContext tests <that> tag when context is nil
func TestTreeProcessorThatTagNoContext(t *testing.T) {
	g := NewForTesting(t, false)

	template := "You said: <that/>"
	expected := "You said: "

	// Create tree processor
	tp := NewTreeProcessor(g)

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Process the AST with nil context
	tp.ctx = nil
	result := tp.processNode(ast)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTreeProcessorThatTagInNestedStructure tests <that> tag in nested AIML structures
func TestTreeProcessorThatTagInNestedStructure(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with response history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: []string{"I like pizza", "I like pasta", "I like ice cream"},
	}

	// Create variable context
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
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
			name:     "That tag in uppercase",
			template: "<uppercase><that/></uppercase>",
			expected: "I LIKE ICE CREAM",
		},
		{
			name:     "That tag in lowercase",
			template: "<lowercase><that/></lowercase>",
			expected: "i like ice cream",
		},
		{
			name:     "That tag with think",
			template: "<think><set name=\"last\"><that/></set></think>Stored: <get name=\"last\"/>",
			expected: "Stored: I like ice cream",
		},
		{
			name:     "Multiple nested that tags",
			template: "<uppercase><that index=\"1\"/></uppercase> and <lowercase><that index=\"2\"/></lowercase>",
			expected: "I LIKE ICE CREAM and i like pasta",
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

// TestTreeProcessorThatTagMaxHistory tests <that> tag with maximum history size
func TestTreeProcessorThatTagMaxHistory(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a large response history (more than typical max)
	responseHistory := make([]string, 15)
	for i := 0; i < 15; i++ {
		responseHistory[i] = "Response " + string(rune('A'+i))
	}

	// Create a session with large response history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: responseHistory,
	}

	// Create variable context
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
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
			name:     "Most recent",
			template: `<that index="1"/>`,
			expected: "Response O", // 15th item (0-indexed: 14)
		},
		{
			name:     "5th most recent",
			template: `<that index="5"/>`,
			expected: "Response K", // 11th item (0-indexed: 10)
		},
		{
			name:     "15th most recent (oldest)",
			template: `<that index="15"/>`,
			expected: "Response A", // 1st item (0-indexed: 0)
		},
		{
			name:     "Beyond history (should be empty)",
			template: `<that index="20"/>`,
			expected: "",
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

// TestTreeProcessorThatTagSpecialCharacters tests <that> tag with special characters in responses
func TestTreeProcessorThatTagSpecialCharacters(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with responses containing special characters
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
		ResponseHistory: []string{
			"Hello, world!",
			"What's up?",
			"I'm \"quoted\"",
			"Price: $19.99",
			"Email: user@example.com",
			"<html>tags</html>",
		},
	}

	// Create variable context
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
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
			name:     "HTML-like content",
			template: `<that index="1"/>`,
			expected: "<html>tags</html>",
		},
		{
			name:     "Email address",
			template: `<that index="2"/>`,
			expected: "Email: user@example.com",
		},
		{
			name:     "Dollar sign",
			template: `<that index="3"/>`,
			expected: "Price: $19.99",
		},
		{
			name:     "Quotes",
			template: `<that index="4"/>`,
			expected: `I'm "quoted"`,
		},
		{
			name:     "Apostrophe",
			template: `<that index="5"/>`,
			expected: "What's up?",
		},
		{
			name:     "Comma and exclamation",
			template: `<that index="6"/>`,
			expected: "Hello, world!",
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
