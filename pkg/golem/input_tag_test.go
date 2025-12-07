package golem

import (
	"strings"
	"testing"
)

func TestInputTagProcessing(t *testing.T) {
	testCases := []struct {
		name           string
		template       string
		requestHistory []string
		expected       string
	}{
		{
			name:           "Basic input tag",
			template:       "You said: <input/>",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello world",
		},
		{
			name:           "Input tag with spaces",
			template:       "You said: <input />",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello world",
		},
		{
			name:           "Input tag with multiple spaces",
			template:       "You said: <input   />",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello world",
		},
		{
			name:           "Multiple input tags",
			template:       "First: <input/>, Second: <input/>",
			requestHistory: []string{"Hello", "World"},
			expected:       "First: World, Second: World",
		},
		{
			name:           "Input tag at beginning",
			template:       "<input/> is what you said",
			requestHistory: []string{"Hello world"},
			expected:       "Hello world is what you said",
		},
		{
			name:           "Input tag at end",
			template:       "You said <input/>",
			requestHistory: []string{"Hello world"},
			expected:       "You said Hello world",
		},
		{
			name:           "Only input tag",
			template:       "<input/>",
			requestHistory: []string{"Hello world"},
			expected:       "Hello world",
		},
		{
			name:           "Empty request history",
			template:       "You said: <input/>",
			requestHistory: []string{},
			expected:       "You said:",
		},
		{
			name:           "Nil request history",
			template:       "You said: <input/>",
			requestHistory: nil,
			expected:       "You said:",
		},
		{
			name:           "Input tag with newlines",
			template:       "You said:\n<input/>\nThank you",
			requestHistory: []string{"Hello world"},
			expected:       "You said:\nHello world\nThank you",
		},
		{
			name:           "Input tag with tabs",
			template:       "You said:\t<input/>\tThank you",
			requestHistory: []string{"Hello world"},
			expected:       "You said:\tHello world\tThank you",
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

			// Set up request history
			ctx.Session.RequestHistory = tc.requestHistory

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestInputTagWithOtherTags(t *testing.T) {
	testCases := []struct {
		name           string
		template       string
		requestHistory []string
		expected       string
	}{
		{
			name:           "Input with uppercase",
			template:       "You said: <uppercase><input/></uppercase>",
			requestHistory: []string{"hello world"},
			expected:       "You said: HELLO WORLD",
		},
		{
			name:           "Input with formal",
			template:       "You said: <formal><input/></formal>",
			requestHistory: []string{"hello world"},
			expected:       "You said: Hello World",
		},
		{
			name:           "Input with person",
			template:       "You said: <person><input/></person>",
			requestHistory: []string{"I am happy"},
			expected:       "You said: you are happy",
		},
		{
			name:           "Input with first and rest",
			template:       "First word: <first><input/></first>, Rest: <rest><input/></rest>",
			requestHistory: []string{"hello world test"},
			expected:       "First word: hello, Rest: world test",
		},
		{
			name:           "Input with random",
			template:       "You said: <random><li><input/></li><li>I heard you</li></random>",
			requestHistory: []string{"hello world"},
			expected:       "You said: hello world", // or "I heard you" depending on random selection
		},
		{
			name:           "Input with condition",
			template:       "Result: <condition name=\"test\" value=\"true\"><input/></condition>",
			requestHistory: []string{"hello world"},
			expected:       "Result: hello world",
		},
		{
			name:           "Input with multiple processing",
			template:       "Result: <uppercase><formal><person><input/></person></formal></uppercase>",
			requestHistory: []string{"I am happy"},
			expected:       "Result: YOU ARE HAPPY",
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

			// Set up request history
			ctx.Session.RequestHistory = tc.requestHistory

			// Set up variables if needed
			if strings.Contains(tc.template, "test") {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			// For random tests, check if result contains expected pattern
			if strings.Contains(tc.template, "random") {
				if !strings.Contains(result, "You said:") {
					t.Errorf("Expected result to contain 'You said:', got '%s'", result)
				}
			} else {
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			}
		})
	}
}

func TestInputTagIntegration(t *testing.T) {
	testCases := []struct {
		name           string
		template       string
		requestHistory []string
		expected       string
	}{
		{
			name:           "Input in complex template",
			template:       "Start <uppercase><formal><input/></formal></uppercase> End",
			requestHistory: []string{"hello world"},
			expected:       "Start HELLO WORLD End",
		},
		{
			name:           "Input with list operations",
			template:       "You said: <input/>, I'll add it to my list: <list name=\"items\" operation=\"add\"><input/></list>",
			requestHistory: []string{"apple"},
			expected:       "You said: apple, I'll add it to my list:",
		},
		{
			name:           "Input with map operations",
			template:       "You said: <input/>, I'll store it: <map name=\"data\" key=\"user_input\" operation=\"set\"><input/></map>",
			requestHistory: []string{"hello world"},
			expected:       "You said: hello world, I'll store it:",
		},
		{
			name:           "Input with SRAI",
			template:       "You said: <input/>, let me respond: <srai><input/></srai>",
			requestHistory: []string{"HELLO"},
			expected:       "You said: HELLO, let me respond: HELLO",
		},
		{
			name:           "Input with multiple processing",
			template:       "Result: <uppercase><formal><person><input/></person></formal></uppercase>",
			requestHistory: []string{"I am going to my house"},
			expected:       "Result: YOU ARE GOING TO YOUR HOUSE",
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

			// Set up request history
			ctx.Session.RequestHistory = tc.requestHistory

			// Set up knowledge base for SRAI test
			if strings.Contains(tc.template, "HELLO") {
				kb := NewAIMLKnowledgeBase()
				kb.Categories = []Category{
					{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
				}
				kb.Patterns = make(map[string]*Category)
				for i := range kb.Categories {
					kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
				}
				g.SetKnowledgeBase(kb)
				// Update the context to use the new knowledge base
				ctx.KnowledgeBase = kb
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestInputTagEdgeCases(t *testing.T) {
	testCases := []struct {
		name           string
		template       string
		requestHistory []string
		expected       string
	}{
		{
			name:           "Malformed input tag (missing slash)",
			template:       "You said: <input>",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello world",
		},
		{
			name:           "Malformed input tag (extra content)",
			template:       "You said: <input>content</input>",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello world",
		},
		{
			name:           "Input tag with attributes",
			template:       "You said: <input index=\"1\"/>",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello world",
		},
		{
			name:           "Empty template",
			template:       "",
			requestHistory: []string{"Hello world"},
			expected:       "",
		},
		{
			name:           "Input tag with special characters",
			template:       "You said: <input/>@#$%",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello world@#$%",
		},
		{
			name:           "Input tag with unicode",
			template:       "You said: <input/>café naïve",
			requestHistory: []string{"Hello world"},
			expected:       "You said: Hello worldcafé naïve",
		},
		{
			name:           "Very long input",
			template:       "You said: <input/>",
			requestHistory: []string{strings.Repeat("a", 1000)},
			expected:       "You said: " + strings.Repeat("a", 1000),
		},
		{
			name:           "Input with empty string",
			template:       "You said: '<input/>'",
			requestHistory: []string{""},
			expected:       "You said: ''",
		},
		{
			name:           "Input with whitespace only",
			template:       "You said: '<input/>'",
			requestHistory: []string{"   "},
			expected:       "You said: '   '",
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

			// Set up request history
			ctx.Session.RequestHistory = tc.requestHistory

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestInputTagPerformance(t *testing.T) {
	// Test with many input tags
	template := "Start "
	for i := 0; i < 100; i++ {
		template += "<input/> "
	}
	template += "End"

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

	// Set up request history
	ctx.Session.RequestHistory = []string{"test input"}

	result := g.ProcessTemplateWithContext(template, map[string]string{}, ctx.Session)
	expected := "Start " + strings.Repeat("test input ", 100) + "End"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestInputTagWithConditionals(t *testing.T) {
	testCases := []struct {
		name           string
		template       string
		requestHistory []string
		expected       string
	}{
		{
			name:           "Input in condition true",
			template:       `<condition name="test" value="true">You said: <input/></condition>`,
			requestHistory: []string{"hello world"},
			expected:       "You said: hello world",
		},
		{
			name:           "Input in condition false",
			template:       `<condition name="test" value="false">You said: <input/></condition>`,
			requestHistory: []string{"hello world"},
			expected:       "",
		},
		{
			name:           "Input with multiple conditions",
			template:       `<condition name="test" value="true">Yes: <input/></condition><condition name="test" value="false">No: <input/></condition>`,
			requestHistory: []string{"hello world"},
			expected:       "Yes: hello world",
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

			// Set up request history
			ctx.Session.RequestHistory = tc.requestHistory

			// Set up variables
			g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx.Session)

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
