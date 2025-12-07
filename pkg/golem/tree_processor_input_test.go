package golem

import (
	"testing"
	"time"
)

// TestTreeProcessorInputTagBasic tests basic <input/> tag processing
func TestTreeProcessorInputTagBasic(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with request history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  []string{"Hello world", "How are you?", "Nice to meet you!"},
		ResponseHistory: make([]string, 0),
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
			name:     "Self-closing input tag",
			template: "You said: <input/>",
			expected: "You said: Nice to meet you!",
		},
		{
			name:     "Input tag with spaces",
			template: "You said: <input />",
			expected: "You said: Nice to meet you!",
		},
		{
			name:     "Input tag at beginning",
			template: "<input/> is what you said",
			expected: "Nice to meet you! is what you said",
		},
		{
			name:     "Input tag at end",
			template: "You said <input/>",
			expected: "You said Nice to meet you!",
		},
		{
			name:     "Only input tag",
			template: "<input/>",
			expected: "Nice to meet you!",
		},
		{
			name:     "Multiple input tags",
			template: "<input/> and then <input/>",
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

// TestTreeProcessorInputTagEdgeCases tests edge cases for input tag
func TestTreeProcessorInputTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name           string
		requestHistory []string
		template       string
		expected       string
		description    string
	}{
		{
			name:           "Empty request history",
			requestHistory: []string{},
			template:       "You said: <input/>",
			expected:       "You said: ",
			description:    "Should return empty string when no history",
		},
		{
			name:           "Single request in history",
			requestHistory: []string{"Hello"},
			template:       "You said: <input/>",
			expected:       "You said: Hello",
			description:    "Should handle single request correctly",
		},
		{
			name:           "Whitespace in request",
			requestHistory: []string{"  Spaces everywhere  "},
			template:       "You said: <input/>",
			expected:       "You said:   Spaces everywhere  ",
			description:    "Should preserve whitespace in requests",
		},
		{
			name:           "Special characters",
			requestHistory: []string{"Hello, world! What's up?"},
			template:       "You said: <input/>",
			expected:       "You said: Hello, world! What's up?",
			description:    "Should handle special characters",
		},
		{
			name:           "Long request history",
			requestHistory: []string{"First", "Second", "Third", "Fourth", "Fifth"},
			template:       "<input/>",
			expected:       "Fifth",
			description:    "Should always return most recent (last) request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a session with specified request history
			session := &ChatSession{
				ID:              "test-session",
				Variables:       make(map[string]string),
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				Topic:           "",
				RequestHistory:  tt.requestHistory,
				ResponseHistory: make([]string, 0),
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

// TestTreeProcessorInputTagNoSession tests input tag when session is nil
func TestTreeProcessorInputTagNoSession(t *testing.T) {
	g := NewForTesting(t, false)

	// Create variable context without a session
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        nil, // No session
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	template := "You said: <input/>"
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

// TestTreeProcessorInputTagNoContext tests input tag when context is nil
func TestTreeProcessorInputTagNoContext(t *testing.T) {
	g := NewForTesting(t, false)

	template := "You said: <input/>"
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

// TestTreeProcessorInputTagInNestedStructure tests input tag in nested AIML structures
func TestTreeProcessorInputTagInNestedStructure(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with request history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  []string{"hello world"},
		ResponseHistory: make([]string, 0),
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
			name:     "Input tag in uppercase",
			template: "<uppercase><input/></uppercase>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Input tag in lowercase",
			template: "<lowercase><input/></lowercase>",
			expected: "hello world",
		},
		{
			name:     "Input tag with think",
			template: "<think><set name=\"last\"><input/></set></think>Stored: <get name=\"last\"/>",
			expected: "Stored: hello world",
		},
		{
			name:     "Input tag in formal",
			template: "<formal><input/></formal>",
			expected: "Hello World",
		},
		{
			name:     "Multiple nested input tags",
			template: "<uppercase><input/></uppercase> and <lowercase><input/></lowercase>",
			expected: "HELLO WORLD and hello world",
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

// TestTreeProcessorInputTagMaxHistory tests input tag with maximum history size
func TestTreeProcessorInputTagMaxHistory(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a large request history (more than typical max)
	requestHistory := make([]string, 15)
	for i := 0; i < 15; i++ {
		requestHistory[i] = "Request " + string(rune('A'+i))
	}

	// Create a session with large request history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  requestHistory,
		ResponseHistory: make([]string, 0),
	}

	// Create variable context
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
		Topic:          "",
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	template := "<input/>"
	expected := "Request O" // Last item (15th)

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

// TestTreeProcessorInputTagSpecialCharacters tests input tag with special characters
func TestTreeProcessorInputTagSpecialCharacters(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name    string
		request string
	}{
		{
			name:    "HTML-like content",
			request: "<html>tags</html>",
		},
		{
			name:    "Email address",
			request: "user@example.com",
		},
		{
			name:    "Dollar sign",
			request: "Price: $19.99",
		},
		{
			name:    "Quotes",
			request: `I said "hello"`,
		},
		{
			name:    "Apostrophe",
			request: "What's up?",
		},
		{
			name:    "Unicode characters",
			request: "Hello 世界 🌍",
		},
		{
			name:    "Newlines",
			request: "Line 1\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a session with the special request
			session := &ChatSession{
				ID:              "test-session",
				Variables:       make(map[string]string),
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				Topic:           "",
				RequestHistory:  []string{tt.request},
				ResponseHistory: make([]string, 0),
			}

			// Create variable context
			ctx := &VariableContext{
				LocalVars:      make(map[string]string),
				Session:        session,
				Topic:          "",
				KnowledgeBase:  g.aimlKB,
				RecursionDepth: 0,
			}

			template := "<input/>"
			expected := tt.request

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
		})
	}
}

// TestTreeProcessorInputTagWithRequestTag tests difference between input and request tags
func TestTreeProcessorInputTagWithRequestTag(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with request history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  []string{"First", "Second", "Third"},
		ResponseHistory: make([]string, 0),
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
			name:     "Input returns most recent",
			template: "<input/>",
			expected: "Third",
		},
		{
			name:     "Request index 1 (same as input)",
			template: `<request index="1"/>`,
			expected: "Third",
		},
		{
			name:     "Request index 2",
			template: `<request index="2"/>`,
			expected: "Second",
		},
		{
			name:     "Request index 3",
			template: `<request index="3"/>`,
			expected: "First",
		},
		{
			name:     "Input and request together",
			template: `Input: <input/>, Request 1: <request index="1"/>, Request 2: <request index="2"/>`,
			expected: "Input: Third, Request 1: Third, Request 2: Second",
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
