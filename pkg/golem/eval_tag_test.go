package golem

import (
	"strings"
	"testing"
)

func TestEvalTagProcessing(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "Basic eval tag",
			template: "<eval>hello world</eval>",
			expected: "hello world",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with variable",
			template: "<eval><get name=\"test\"></get></eval>",
			expected: "hello world",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">hello world</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval with wildcard",
			template: "<eval><star/></eval>",
			expected: "test input",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with formatting",
			template: "<eval><uppercase>hello world</uppercase></eval>",
			expected: "HELLO WORLD",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with person tag",
			template: "<eval><person>I am going to my house</person></eval>",
			expected: "you are going to your house",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with condition",
			template: "<eval><condition name=\"test\" value=\"true\">yes</condition></eval>",
			expected: "yes",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval with random",
			template: "<eval><random><li>option1</li><li>option2</li></random></eval>",
			expected: "option1", // Will be one of the options
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with first/rest",
			template: "<eval><first>apple banana cherry</first></eval>",
			expected: "apple",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with input",
			template: "<eval>You said: <input/></eval>",
			expected: "You said: hello",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with empty content",
			template: "<eval></eval>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with whitespace only",
			template: "<eval>   </eval>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with nested eval",
			template: "<eval><eval>hello</eval></eval>",
			expected: "hello",
			setup:    func(*Golem, *ChatSession) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			// Set up request history for input tag tests
			ctx.Session.RequestHistory = []string{"hello"}

			// Run setup if provided
			if tc.setup != nil {
				tc.setup(g, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{"star1": "test input"}, ctx.Session)

			// For random tests, check if result is one of the expected options
			if tc.name == "Eval with random" {
				expectedOptions := []string{"option1", "option2"}
				found := false
				for _, option := range expectedOptions {
					if result == option {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected one of %v, got '%s'", expectedOptions, result)
				}
			} else {
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			}
		})
	}
}

func TestEvalTagWithOtherTags(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "Eval with uppercase",
			template: "<uppercase><eval>hello world</eval></uppercase>",
			expected: "HELLO WORLD",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with formal",
			template: "<formal><eval>hello world</eval></formal>",
			expected: "Hello World",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with person",
			template: "<person><eval>I am going to my house</eval></person>",
			expected: "you are going to your house",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with first and rest",
			template: "<eval><first>apple banana cherry</first> <rest>apple banana cherry</rest></eval>",
			expected: "apple banana cherry",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with condition",
			template: "<condition name=\"test\" value=\"true\"><eval>yes</eval></condition>",
			expected: "yes",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval with multiple processing",
			template: "<uppercase><formal><person><eval>I am going to my house</eval></person></formal></uppercase>",
			expected: "YOU ARE GOING TO YOUR HOUSE",
			setup:    func(*Golem, *ChatSession) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			// Run setup if provided
			if tc.setup != nil {
				tc.setup(g, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestEvalTagIntegration(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "Eval in complex template",
			template: "Result: <eval><get name=\"data\"></get></eval>",
			expected: "Result: processed data",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="data">processed data</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval with list operations",
			template: "Items: <eval><list name=\"items\" operation=\"get\"></list></eval>",
			expected: "Items: apple",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<list name="items" operation="add">apple</list>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval with map operations",
			template: "Value: <eval><map name=\"data\" key=\"key1\"></map></eval>",
			expected: "Value: value1",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<map name="data" key="key1" operation="set">value1</map>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval with SRAI",
			template: "Response: <eval><srai>HELLO</srai></eval>",
			expected: "Response: Hello! How can I help you today?",
			setup: func(g *Golem, ctx *ChatSession) {
				// Set up knowledge base for SRAI
				kb := NewAIMLKnowledgeBase()
				kb.Categories = []Category{
					{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
				}
				kb.Patterns = make(map[string]*Category)
				for i := range kb.Categories {
					kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
				}
				g.SetKnowledgeBase(kb)
			},
		},
		{
			name:     "Eval with multiple processing",
			template: "Result: <uppercase><formal><eval><get name=\"message\"></get></eval></formal></uppercase>",
			expected: "Result: HELLO WORLD",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="message">hello world</set>`, map[string]string{}, ctx)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			// Run setup if provided
			if tc.setup != nil {
				tc.setup(g, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestEvalTagEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "Malformed eval tag (missing closing)",
			template: "<eval>hello world",
			expected: "<eval>hello world",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Malformed eval tag (extra content)",
			template: "<eval>hello world</eval> extra",
			expected: "hello world extra",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval tag with attributes",
			template: "<eval attr=\"value\">hello world</eval>",
			expected: "hello world", // Tree processor evaluates content even with attributes
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Empty template",
			template: "",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval tag with special characters",
			template: "<eval>hello@world#123</eval>",
			expected: "hello@world#123",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval tag with unicode",
			template: "<eval>café naïve</eval>",
			expected: "café naïve",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Very long eval content",
			template: "<eval>" + strings.Repeat("hello ", 100) + "</eval>",
			expected: strings.TrimSpace(strings.Repeat("hello ", 100)),
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with empty string",
			template: "<eval></eval>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Eval with whitespace only",
			template: "<eval>   </eval>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			// Run setup if provided
			if tc.setup != nil {
				tc.setup(g, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestEvalTagPerformance(t *testing.T) {
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

	// Test with many eval tags
	template := strings.Repeat("<eval>test</eval> ", 1000)
	expected := strings.TrimSpace(strings.Repeat("test ", 1000))

	result := g.ProcessTemplateWithContext(template, map[string]string{}, ctx.Session)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestEvalTagWithConditionals(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "Eval in condition true",
			template: "<condition name=\"test\" value=\"true\"><eval>yes</eval></condition>",
			expected: "yes",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval in condition false",
			template: "<condition name=\"test\" value=\"false\"><eval>yes</eval></condition>",
			expected: "",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Eval with multiple conditions",
			template: "<condition name=\"test1\" value=\"true\"><eval>yes</eval></condition> <condition name=\"test2\" value=\"true\"><eval>maybe</eval></condition>",
			expected: "yes",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test1">true</set>`, map[string]string{}, ctx)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			// Run setup if provided
			if tc.setup != nil {
				tc.setup(g, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
