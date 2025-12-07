package golem

import (
	"strings"
	"testing"
)

func TestLoopTagProcessing(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic loop tag removal",
			template: "Hello <loop/> world",
			expected: "Hello  world",
		},
		{
			name:     "Multiple loop tags",
			template: "Start <loop/> middle <loop/> end",
			expected: "Start  middle  end",
		},
		{
			name:     "Loop tag with spaces",
			template: "Hello <loop /> world",
			expected: "Hello  world",
		},
		{
			name:     "Loop tag with multiple spaces",
			template: "Hello <loop   /> world",
			expected: "Hello  world",
		},
		{
			name:     "Loop tag at beginning",
			template: "<loop/>Hello world",
			expected: "Hello world",
		},
		{
			name:     "Loop tag at end",
			template: "Hello world<loop/>",
			expected: "Hello world",
		},
		{
			name:     "Only loop tag",
			template: "<loop/>",
			expected: "",
		},
		{
			name:     "Multiple consecutive loop tags",
			template: "Hello <loop/><loop/><loop/> world",
			expected: "Hello  world",
		},
		{
			name:     "Loop tag with newlines",
			template: "Hello\n<loop/>\nworld",
			expected: "Hello\n\nworld",
		},
		{
			name:     "Loop tag with tabs",
			template: "Hello\t<loop/>\tworld",
			expected: "Hello\t\tworld",
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

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestLoopTagWithOtherTags(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Loop with uppercase",
			template: "Hello <loop/><uppercase>world</uppercase>",
			expected: "Hello WORLD",
		},
		{
			name:     "Loop with formal",
			template: "Hello <loop/><formal>world</formal>",
			expected: "Hello World",
		},
		{
			name:     "Loop with person",
			template: "Hello <loop/><person>I am happy</person>",
			expected: "Hello you are happy",
		},
		{
			name:     "Loop with variables",
			template: "Hello <loop/><get name=\"name\"></get>",
			expected: "Hello John",
		},
		{
			name:     "Loop with first and rest",
			template: "First: <loop/><first>apple banana cherry</first>, Rest: <loop/><rest>apple banana cherry</rest>",
			expected: "First: apple, Rest: banana cherry",
		},
		{
			name:     "Loop with random",
			template: "Choice: <loop/><random><li>Option 1</li><li>Option 2</li></random>",
			expected: "Choice: Option 1", // or Option 2, depending on random selection
		},
		{
			name:     "Loop with condition",
			template: "Result: <loop/><condition name=\"test\" value=\"true\">Yes</condition>",
			expected: "Result: Yes",
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

			// Set up variables if needed
			if strings.Contains(tc.template, "name") {
				g.ProcessTemplateWithContext(`<set name="name">John</set>`, map[string]string{}, ctx.Session)
			}
			if strings.Contains(tc.template, "test") {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			// For random tests, check if result contains expected pattern
			if strings.Contains(tc.template, "random") {
				if !strings.Contains(result, "Choice: Option") {
					t.Errorf("Expected result to contain 'Choice: Option', got '%s'", result)
				}
			} else {
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			}
		})
	}
}

func TestLoopTagIntegration(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Loop in complex template",
			template: "Start <loop/><uppercase><formal>hello world</formal></uppercase> <loop/>End",
			expected: "Start HELLO WORLD End",
		},
		{
			name:     "Loop with list operations",
			template: "Items: <loop/><list name=\"items\" operation=\"get\"></list>",
			expected: "Items: apple banana",
		},
		{
			name:     "Loop with map operations",
			template: "Value: <loop/><map name=\"data\" key=\"key1\"></map>",
			expected: "Value: value1",
		},
		{
			name:     "Loop with SRAI",
			template: "Response: <loop/><srai>HELLO</srai>",
			expected: "Response: Hello! How can I help you today?",
		},
		{
			name:     "Loop with multiple processing",
			template: "Result: <loop/><uppercase><person><get name=\"message\"></get></person></uppercase>",
			expected: "Result: YOU ARE HAPPY",
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
			}

			// Set up variables/collections if needed
			if strings.Contains(tc.template, "message") {
				g.ProcessTemplateWithContext(`<set name="message">I am happy</set>`, map[string]string{}, ctx.Session)
			}
			if strings.Contains(tc.template, "items") {
				g.ProcessTemplateWithContext(`<list name="items" operation="add">apple</list>`, map[string]string{}, ctx.Session)
				g.ProcessTemplateWithContext(`<list name="items" operation="add">banana</list>`, map[string]string{}, ctx.Session)
			}
			if strings.Contains(tc.template, "data") {
				g.ProcessTemplateWithContext(`<map name="data" key="key1" operation="set">value1</map>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestLoopTagEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Malformed loop tag (missing slash)",
			template: "Hello <loop> world",
			expected: "Hello", // Tree processor handles tag, world is consumed, trailing space trimmed
		},
		{
			name:     "Malformed loop tag (extra content)",
			template: "Hello <loop>content</loop> world",
			expected: "Hello  world", // Tree processor handles tag, content inside tag
		},
		{
			name:     "Loop tag with attributes",
			template: "Hello <loop count=\"5\"/> world",
			expected: "Hello  world", // Tree processor handles loop tag
		},
		{
			name:     "Empty template",
			template: "",
			expected: "",
		},
		{
			name:     "Only whitespace",
			template: "   ",
			expected: "",
		},
		{
			name:     "Loop tag with special characters",
			template: "Hello <loop/>@#$% world",
			expected: "Hello @#$% world",
		},
		{
			name:     "Loop tag with unicode",
			template: "Hello <loop/>café naïve world",
			expected: "Hello café naïve world",
		},
		{
			name:     "Very long template with loop",
			template: strings.Repeat("a", 1000) + "<loop/>" + strings.Repeat("b", 1000),
			expected: strings.Repeat("a", 1000) + strings.Repeat("b", 1000),
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

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestLoopTagPerformance(t *testing.T) {
	// Test with many loop tags
	template := "Start "
	for i := 0; i < 100; i++ {
		template += "<loop/>"
	}
	template += " End"

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

	result := g.ProcessTemplateWithContext(template, map[string]string{}, ctx.Session)
	expected := "Start " + strings.Repeat("", 100) + " End"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestLoopTagWithConditionals(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Loop in condition true",
			template: `<condition name="test" value="true">Yes <loop/> more</condition>`,
			expected: "Yes  more",
		},
		{
			name:     "Loop in condition false",
			template: `<condition name="test" value="false">No <loop/> more</condition>`,
			expected: "",
		},
		{
			name:     "Loop with multiple conditions",
			template: `<condition name="test" value="true">Yes <loop/></condition><condition name="test" value="false">No <loop/></condition>`,
			expected: "Yes",
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

			// Set up variables
			g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx.Session)

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
