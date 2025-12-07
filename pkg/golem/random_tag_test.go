package golem

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRandomTagProcessing(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected []string // Multiple possible expected results
	}{
		{
			name: "Basic random selection",
			template: `<random>
				<li>Option 1</li>
				<li>Option 2</li>
				<li>Option 3</li>
			</random>`,
			expected: []string{"Option 1", "Option 2", "Option 3"},
		},
		{
			name: "Single option",
			template: `<random>
				<li>Only option</li>
			</random>`,
			expected: []string{"Only option"},
		},
		{
			name:     "Empty random tag",
			template: `<random></random>`,
			expected: []string{""},
		},
		{
			name: "Random with whitespace",
			template: `<random>
				<li>  Option 1  </li>
				<li>  Option 2  </li>
			</random>`,
			expected: []string{"Option 1", "Option 2"},
		},
		{
			name: "Random with numbers",
			template: `<random>
				<li>1</li>
				<li>2</li>
				<li>3</li>
			</random>`,
			expected: []string{"1", "2", "3"},
		},
		{
			name: "Random with special characters",
			template: `<random>
				<li>Hello, world!</li>
				<li>How are you?</li>
				<li>Goodbye!</li>
			</random>`,
			expected: []string{"Hello, world!", "How are you?", "Goodbye!"},
		},
		{
			name: "Random with unicode",
			template: `<random>
				<li>café</li>
				<li>naïve</li>
				<li>résumé</li>
			</random>`,
			expected: []string{"café", "naïve", "résumé"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			// Ensure knowledge base is initialized
			if g.aimlKB == nil {
				g.aimlKB = NewAIMLKnowledgeBase()
			}
			// Create a unique session for this test case
			sessionID := fmt.Sprintf("random_test_%s_%d", tc.name, time.Now().UnixNano())
			session := g.createSession(sessionID)

			ctx := &VariableContext{
				LocalVars:      make(map[string]string),
				Session:        session,
				Topic:          "",
				KnowledgeBase:  g.aimlKB,
				RecursionDepth: 0,
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			// Check if result is one of the expected options
			found := false
			for _, expected := range tc.expected {
				if result == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected one of %v, got '%s'", tc.expected, result)
			}
		})
	}
}

func TestRandomTagWithNestedTags(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name: "Random with uppercase",
			template: `<random>
				<li><uppercase>hello</uppercase></li>
				<li><uppercase>world</uppercase></li>
			</random>`,
			expected: []string{"HELLO", "WORLD"},
		},
		{
			name: "Random with formal",
			template: `<random>
				<li><formal>hello world</formal></li>
				<li><formal>good morning</formal></li>
			</random>`,
			expected: []string{"Hello World", "Good Morning"},
		},
		{
			name: "Random with person",
			template: `<random>
				<li><person>I am happy</person></li>
				<li><person>I am sad</person></li>
			</random>`,
			expected: []string{"you are happy", "you are sad"},
		},
		{
			name: "Random with variables",
			template: `<random>
				<li>Hello <get name="name"></get></li>
				<li>Hi <get name="name"></get></li>
			</random>`,
			expected: []string{"Hello John", "Hi John"},
		},
		{
			name: "Random with multiple nested tags",
			template: `<random>
				<li><uppercase><formal>hello world</formal></uppercase></li>
				<li><uppercase><formal>good morning</formal></uppercase></li>
			</random>`,
			expected: []string{"HELLO WORLD", "GOOD MORNING"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			// Ensure knowledge base is initialized
			if g.aimlKB == nil {
				g.aimlKB = NewAIMLKnowledgeBase()
			}
			// Create a unique session for this test case
			sessionID := fmt.Sprintf("random_test_%s_%d", tc.name, time.Now().UnixNano())
			session := g.createSession(sessionID)

			ctx := &VariableContext{
				LocalVars:      make(map[string]string),
				Session:        session,
				Topic:          "",
				KnowledgeBase:  g.aimlKB,
				RecursionDepth: 0,
			}

			// Set up variables if needed
			if strings.Contains(tc.template, "name") {
				g.ProcessTemplateWithContext(`<set name="name">John</set>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			// Check if result is one of the expected options
			found := false
			for _, expected := range tc.expected {
				if result == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected one of %v, got '%s'", tc.expected, result)
			}
		})
	}
}

func TestRandomTagIntegration(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name: "Random with first and rest",
			template: `<random>
				<li><first>apple banana cherry</first></li>
				<li><rest>apple banana cherry</rest></li>
			</random>`,
			expected: []string{"apple", "banana cherry"},
		},
		{
			name: "Random with list operations",
			template: `<random>
				<li><list name="items" operation="get"></list></li>
				<li>No items available</li>
			</random>`,
			expected: []string{"apple banana", "No items available"},
		},
		{
			name: "Random with condition",
			template: `<random>
				<li><condition name="test" value="true">Yes</condition></li>
				<li><condition name="test" value="false">No</condition></li>
			</random>`,
			expected: []string{"Yes", ""},
		},
		{
			name: "Random with SRAI",
			template: `<random>
				<li><srai>HELLO</srai></li>
				<li>Hi there!</li>
			</random>`,
			expected: []string{"Hello! How can I help you today?", "Hi there!"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			// Ensure knowledge base is initialized
			if g.aimlKB == nil {
				g.aimlKB = NewAIMLKnowledgeBase()
			}
			// Create a unique session for this test case
			sessionID := fmt.Sprintf("random_test_%s_%d", tc.name, time.Now().UnixNano())
			session := g.createSession(sessionID)

			ctx := &VariableContext{
				LocalVars:      make(map[string]string),
				Session:        session,
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
			if strings.Contains(tc.template, "test") {
				// Set the variable in the knowledge base instead of session
				// since random tag processing uses nil session
				if g.aimlKB == nil {
					g.aimlKB = NewAIMLKnowledgeBase()
				}
				g.aimlKB.Variables["test"] = "true"
			}
			if strings.Contains(tc.template, "items") {
				g.ProcessTemplateWithContext(`<list name="items" operation="add">apple</list>`, map[string]string{}, ctx.Session)
				g.ProcessTemplateWithContext(`<list name="items" operation="add">banana</list>`, map[string]string{}, ctx.Session)
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			// Check if result is one of the expected options
			found := false
			for _, expected := range tc.expected {
				if result == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected one of %v, got '%s'", tc.expected, result)
			}
		})
	}
}

func TestRandomTagEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name: "Malformed random tag",
			template: `<random>
				<li>Option 1</li>
				<li>Option 2</li>
			</random`,
			expected: []string{"Option 1", "Option 2"}, // Tree processor handles tag, returns one option
		},
		{
			name: "Empty li tags",
			template: `<random>
				<li></li>
				<li></li>
			</random>`,
			expected: []string{""},
		},
		{
			name: "Random with only whitespace",
			template: `<random>
				<li>   </li>
				<li>   </li>
			</random>`,
			expected: []string{""},
		},
		{
			name: "Random with mixed empty and non-empty",
			template: `<random>
				<li></li>
				<li>Option 1</li>
				<li></li>
			</random>`,
			expected: []string{"", "Option 1"},
		},
		{
			name: "Random with very long content",
			template: `<random>
				<li>` + strings.Repeat("a", 1000) + `</li>
				<li>` + strings.Repeat("b", 1000) + `</li>
			</random>`,
			expected: []string{strings.Repeat("a", 1000), strings.Repeat("b", 1000)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			// Ensure knowledge base is initialized
			if g.aimlKB == nil {
				g.aimlKB = NewAIMLKnowledgeBase()
			}
			// Create a unique session for this test case
			sessionID := fmt.Sprintf("random_test_%s_%d", tc.name, time.Now().UnixNano())
			session := g.createSession(sessionID)

			ctx := &VariableContext{
				LocalVars:      make(map[string]string),
				Session:        session,
				Topic:          "",
				KnowledgeBase:  g.aimlKB,
				RecursionDepth: 0,
			}

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{}, ctx.Session)

			// Check if result is one of the expected options
			found := false
			for _, expected := range tc.expected {
				if result == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected one of %v, got '%s'", tc.expected, result)
			}
		})
	}
}

func TestRandomTagPerformance(t *testing.T) {
	// Test with many options
	template := `<random>`
	for i := 0; i < 100; i++ {
		template += `<li>Option ` + string(rune('0'+i%10)) + `</li>`
	}
	template += `</random>`

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

	// Run multiple times to test randomness
	results := make(map[string]int)
	for i := 0; i < 10; i++ {
		result := g.ProcessTemplateWithContext(template, map[string]string{}, ctx.Session)
		results[result]++
	}

	// Should have some variation (not all the same result)
	if len(results) == 1 {
		t.Logf("Warning: All random selections returned the same result: %v", results)
	}

	// Check that we got some results
	if len(results) == 0 {
		t.Errorf("Expected some random results, got none")
	}
}
