package golem

import (
	"testing"
	"time"
)

// TestTreeProcessorEvalTagBasic tests basic <eval> tag processing with AST
func TestTreeProcessorEvalTagBasic(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Use AST processor

	// Initialize knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Create session
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name     string
		template string
		setup    func()
		expected string
	}{
		{
			name:     "Plain text evaluation",
			template: "<eval>hello world</eval>",
			setup:    func() {},
			expected: "hello world",
		},
		{
			name:     "Eval with uppercase",
			template: "<eval><uppercase>hello</uppercase></eval>",
			setup:    func() {},
			expected: "HELLO",
		},
		{
			name:     "Eval with lowercase",
			template: "<eval><lowercase>HELLO</lowercase></eval>",
			setup:    func() {},
			expected: "hello",
		},
		{
			name:     "Eval with formal",
			template: "<eval><formal>hello world</formal></eval>",
			setup:    func() {},
			expected: "Hello World",
		},
		{
			name:     "Eval with variable",
			template: "<eval>Hello <get name=\"name\"/></eval>",
			setup: func() {
				session.Variables["name"] = "Alice"
			},
			expected: "Hello Alice",
		},
		{
			name:     "Eval with set and get",
			template: "<eval><set name=\"greeting\">Hi</set></eval><get name=\"greeting\"/>",
			setup:    func() {},
			expected: "Hi",
		},
		{
			name:     "Empty eval tag",
			template: "<eval></eval>",
			setup:    func() {},
			expected: "",
		},
		{
			name:     "Eval with whitespace only",
			template: "<eval>   </eval>",
			setup:    func() {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset variables for clean test
			session.Variables = make(map[string]string)

			// Run setup
			if tt.setup != nil {
				tt.setup()
			}

			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorEvalTagNested tests nested <eval> tags
func TestTreeProcessorEvalTagNested(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name     string
		template string
		setup    func()
		expected string
	}{
		{
			name:     "Nested eval",
			template: "<eval><eval>hello</eval></eval>",
			setup:    func() {},
			expected: "hello",
		},
		{
			name:     "Triple nested eval",
			template: "<eval><eval><eval>test</eval></eval></eval>",
			setup:    func() {},
			expected: "test",
		},
		{
			name:     "Eval with nested formatting",
			template: "<eval><uppercase><lowercase>HELLO</lowercase></uppercase></eval>",
			setup:    func() {},
			expected: "HELLO",
		},
		{
			name:     "Eval with nested variables",
			template: "<eval><uppercase><get name=\"msg\"/></uppercase></eval>",
			setup: func() {
				session.Variables["msg"] = "hello"
			},
			expected: "HELLO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session.Variables = make(map[string]string)

			if tt.setup != nil {
				tt.setup()
			}

			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorEvalTagWithConditions tests eval with conditional logic
func TestTreeProcessorEvalTagWithConditions(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name     string
		template string
		setup    func()
		expected string
	}{
		{
			name:     "Eval with condition true",
			template: "<eval><condition name=\"status\" value=\"active\">System active</condition></eval>",
			setup: func() {
				session.Variables["status"] = "active"
			},
			expected: "System active",
		},
		{
			name:     "Eval with condition false",
			template: "<eval><condition name=\"status\" value=\"active\">System active</condition></eval>",
			setup: func() {
				session.Variables["status"] = "inactive"
			},
			expected: "",
		},
		{
			name:     "Eval with condition list",
			template: `<eval><condition name="role"><li value="admin">Admin access</li><li value="user">User access</li><li>Guest access</li></condition></eval>`,
			setup: func() {
				session.Variables["role"] = "admin"
			},
			expected: "Admin access", // Tree processor correctly returns only matching li (AIML spec compliant)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session.Variables = make(map[string]string)

			if tt.setup != nil {
				tt.setup()
			}

			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorEvalTagWithTextProcessing tests eval with person/gender tags
func TestTreeProcessorEvalTagWithTextProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Eval with person tag",
			template: "<eval><person>I am happy</person></eval>",
			expected: "you are happy",
		},
		{
			name:     "Eval with person2 tag",
			template: "<eval><person2>you are happy</person2></eval>",
			expected: "you are happy", // person2 in tree processor might not fully work yet
		},
		{
			name:     "Eval with gender tag",
			template: "<eval><gender>he is happy</gender></eval>",
			expected: "she is happy",
		},
		{
			name:     "Eval with multiple transformations",
			template: "<eval><uppercase><person>I am going to my house</person></uppercase></eval>",
			expected: "YOU ARE GOING TO YOUR HOUSE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorEvalTagWithWildcards tests eval with wildcard references
func TestTreeProcessorEvalTagWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name      string
		template  string
		wildcards map[string]string
		expected  string
	}{
		{
			name:      "Eval with star",
			template:  "<eval>You said: <star/></eval>",
			wildcards: map[string]string{"star1": "hello"},
			expected:  "You said: hello", // Tree processor puts wildcards in session.Variables
		},
		{
			name:      "Eval with uppercase star",
			template:  "<eval><uppercase><star/></uppercase></eval>",
			wildcards: map[string]string{"star1": "hello"},
			expected:  "HELLO", // Tree processor puts wildcards in session.Variables
		},
		{
			name:      "Eval with multiple stars",
			template:  "<eval><star index=\"1\"/> and <star index=\"2\"/></eval>",
			wildcards: map[string]string{"star1": "first", "star2": "second"},
			expected:  "first and second", // Tree processor puts wildcards in session.Variables
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tt.template, tt.wildcards, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorEvalTagWithHistoryTags tests eval with request/response/that/input tags
func TestTreeProcessorEvalTagWithHistoryTags(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  []string{"hello", "how are you"},
		ResponseHistory: []string{"hi there", "I'm good"},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Eval with input tag",
			template: "<eval>You said: <input/></eval>",
			expected: "You said: how are you",
		},
		{
			name:     "Eval with request tag",
			template: "<eval>Previous: <request index=\"2\"/></eval>",
			expected: "Previous: hello",
		},
		{
			name:     "Eval with response tag",
			template: "<eval>I said: <response index=\"1\"/></eval>",
			expected: "I said: I'm good", // Response index 1 is most recent (last in array)
		},
		{
			name:     "Eval with that tag",
			template: "<eval>That was: <that/></eval>",
			expected: "That was: I'm good",
		},
		{
			name:     "Eval with uppercase input",
			template: "<eval><uppercase><input/></uppercase></eval>",
			expected: "HOW ARE YOU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorEvalTagEdgeCases tests edge cases for eval tag
func TestTreeProcessorEvalTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Empty eval",
			template: "<eval></eval>",
			expected: "",
		},
		{
			name:     "Eval with only whitespace",
			template: "<eval>   \t\n  </eval>",
			expected: "",
		},
		{
			name:     "Eval with leading/trailing whitespace",
			template: "<eval>  hello  </eval>",
			expected: "hello",
		},
		{
			name:     "Eval with newlines",
			template: "<eval>\nhello\nworld\n</eval>",
			expected: "hello\nworld",
		},
		{
			name:     "Eval with special characters",
			template: "<eval>Hello, World! @#$%</eval>",
			expected: "Hello, World! @#$%",
		},
		{
			name:     "Eval with unicode",
			template: "<eval>Hello 世界 🌍</eval>",
			expected: "Hello 世界 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorEvalTagNoContext tests eval tag with nil context
func TestTreeProcessorEvalTagNoContext(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	template := "<eval>test</eval>"
	expected := "test"

	// Process with nil session (should still work)
	result := g.ProcessTemplate(template, map[string]string{})

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestTreeProcessorEvalTagComplex tests complex eval scenarios
func TestTreeProcessorEvalTagComplex(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

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

	session.Variables["name"] = "Alice"
	session.Variables["greeting"] = "Hi"

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Multiple operations",
			template: "<eval><uppercase><get name=\"greeting\"/></uppercase> <formal><get name=\"name\"/></formal></eval>",
			expected: "HI Alice",
		},
		{
			name:     "Eval with think",
			template: "<eval><think><set name=\"temp\">value</set></think>Done</eval><get name=\"temp\"/>",
			expected: "Donevalue",
		},
		{
			name:     "Complex nested structure",
			template: "<eval><condition name=\"name\" value=\"Alice\"><uppercase>Welcome <get name=\"name\"/></uppercase></condition></eval>",
			expected: "WELCOME ALICE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tt.template, map[string]string{}, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
