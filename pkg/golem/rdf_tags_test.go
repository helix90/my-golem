package golem

import (
	"strings"
	"testing"
)

func TestRDFTagsProcessing(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "Basic uniq tag",
			template: "<uniq>test content</uniq>",
			expected: "test content",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Basic subj tag",
			template: "<subj>cat</subj>",
			expected: "cat",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Basic pred tag",
			template: "<pred>hasPlural</pred>",
			expected: "hasPlural",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Basic obj tag",
			template: "<obj>cats</obj>",
			expected: "cats",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Uniq with subj/pred/obj",
			template: "<uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq>",
			expected: "cat hasPlural cats",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Uniq with wildcard",
			template: "<uniq><subj><star/></subj><pred>sound</pred><obj>meow</obj></uniq>",
			expected: "test input sound meow",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Uniq with variable",
			template: "<uniq><subj><get name=\"animal\"></get></subj><pred>hasPlural</pred><obj><get name=\"plural\"></get></obj></uniq>",
			expected: "dog hasPlural dogs",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="animal">dog</set>`, map[string]string{}, ctx)
				g.ProcessTemplateWithContext(`<set name="plural">dogs</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "Uniq with formatting",
			template: "<uniq><subj><uppercase>cat</uppercase></subj><pred>hasPlural</pred><obj><formal>cats</formal></obj></uniq>",
			expected: "CAT hasPlural Cats",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Uniq with person tag",
			template: "<uniq><subj><person>I</person></subj><pred>am</pred><obj><person>a cat</person></obj></uniq>",
			expected: "you am a cat",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Multiple uniq tags",
			template: "<uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq> <uniq><subj>dog</subj><pred>hasPlural</pred><obj>dogs</obj></uniq>",
			expected: "cat hasPlural cats dog hasPlural dogs",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Nested uniq tags",
			template: "<uniq><subj><uniq><subj>cat</subj><pred>is</pred><obj>animal</obj></uniq></subj><pred>hasPlural</pred><obj>cats</obj></uniq>",
			expected: "cat is animal hasPlural cats",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Empty uniq tag",
			template: "<uniq></uniq>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Empty subj tag",
			template: "<subj></subj>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Empty pred tag",
			template: "<pred></pred>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Empty obj tag",
			template: "<obj></obj>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Whitespace only uniq tag",
			template: "<uniq>   </uniq>",
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

			result := g.ProcessTemplateWithContext(tc.template, map[string]string{"star1": "test input"}, ctx.Session)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestRDFTagsWithOtherTags(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "RDF with uppercase",
			template: "<uppercase><uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq></uppercase>",
			expected: "CAT HASPLURAL CATS",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF with formal",
			template: "<formal><uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq></formal>",
			expected: "Cat Hasplural Cats",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF with person",
			template: "<person><uniq><subj>I</subj><pred>am</pred><obj>a cat</obj></uniq></person>",
			expected: "you are a cat",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF with first and rest",
			template: "<uniq><subj><first>cat dog bird</first></subj><pred>hasPlural</pred><obj><rest>cat dog bird</rest></obj></uniq>",
			expected: "cat hasPlural dog bird",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF with condition",
			template: "<condition name=\"test\" value=\"true\"><uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq></condition>",
			expected: "cat hasPlural cats",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "RDF with multiple processing",
			template: "<uppercase><formal><person><uniq><subj>I</subj><pred>am</pred><obj>a cat</obj></uniq></person></formal></uppercase>",
			expected: "YOU ARE A CAT",
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

func TestRDFTagsIntegration(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "RDF in complex template",
			template: "Knowledge: <uniq><subj><get name=\"animal\"></get></subj><pred>hasPlural</pred><obj><get name=\"plural\"></get></obj></uniq>",
			expected: "Knowledge: cat hasPlural cats",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="animal">cat</set>`, map[string]string{}, ctx)
				g.ProcessTemplateWithContext(`<set name="plural">cats</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "RDF with list operations",
			template: "Facts: <uniq><subj><list name=\"animals\" operation=\"get\"></list></subj><pred>are</pred><obj>pets</obj></uniq>",
			expected: "Facts: cat are pets",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<list name="animals" operation="add">cat</list>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "RDF with map operations",
			template: "Data: <uniq><subj><map name=\"data\" key=\"subject\"></map></subj><pred>is</pred><obj><map name=\"data\" key=\"object\"></map></obj></uniq>",
			expected: "Data: cat is animal",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<map name="data" key="subject" operation="set">cat</map>`, map[string]string{}, ctx)
				g.ProcessTemplateWithContext(`<map name="data" key="object" operation="set">animal</map>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "RDF with SRAI",
			template: "Response: <uniq><subj>cat</subj><pred>says</pred><obj><srai>HELLO</srai></obj></uniq>",
			expected: "Response: cat says Hello! How can I help you today?",
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
			name:     "RDF with multiple processing",
			template: "Result: <uppercase><formal><uniq><subj><get name=\"subject\"></get></subj><pred>hasPlural</pred><obj><get name=\"object\"></get></obj></uniq></formal></uppercase>",
			expected: "Result: CAT HASPLURAL CATS",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="subject">cat</set>`, map[string]string{}, ctx)
				g.ProcessTemplateWithContext(`<set name="object">cats</set>`, map[string]string{}, ctx)
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

func TestRDFTagsEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "Malformed uniq tag (missing closing)",
			template: "<uniq>test content",
			expected: "", // Tree processor handles unclosed tags gracefully
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Malformed subj tag (missing closing)",
			template: "<subj>cat",
			expected: "", // Tree processor handles unclosed tags gracefully
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Malformed pred tag (missing closing)",
			template: "<pred>hasPlural",
			expected: "", // Tree processor handles unclosed tags gracefully
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Malformed obj tag (missing closing)",
			template: "<obj>cats",
			expected: "", // Tree processor handles unclosed tags gracefully
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF tag with attributes",
			template: "<uniq attr=\"value\">test content</uniq>",
			expected: "test content",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Empty template",
			template: "",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF tag with special characters",
			template: "<uniq><subj>cat@world#123</subj><pred>hasPlural</pred><obj>cats@world#123</obj></uniq>",
			expected: "cat@world#123 hasPlural cats@world#123",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF tag with unicode",
			template: "<uniq><subj>café</subj><pred>hasPlural</pred><obj>cafés</obj></uniq>",
			expected: "café hasPlural cafés",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "Very long RDF content",
			template: "<uniq><subj>" + strings.Repeat("cat ", 100) + "</subj><pred>hasPlural</pred><obj>" + strings.Repeat("cats ", 100) + "</obj></uniq>",
			expected: strings.TrimSpace(strings.Repeat("cat ", 100)) + " hasPlural " + strings.TrimSpace(strings.Repeat("cats ", 100)),
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF with empty string",
			template: "<uniq><subj></subj><pred></pred><obj></obj></uniq>",
			expected: "",
			setup:    func(*Golem, *ChatSession) {},
		},
		{
			name:     "RDF with whitespace only",
			template: "<uniq><subj>   </subj><pred>   </pred><obj>   </obj></uniq>",
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

func TestRDFTagsPerformance(t *testing.T) {
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

	// Test with many RDF tags
	template := strings.Repeat("<uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq> ", 1000)
	expected := strings.TrimSpace(strings.Repeat("cat hasPlural cats ", 1000))

	result := g.ProcessTemplateWithContext(template, map[string]string{}, ctx.Session)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestRDFTagsWithConditionals(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		expected string
		setup    func(*Golem, *ChatSession)
	}{
		{
			name:     "RDF in condition true",
			template: "<condition name=\"test\" value=\"true\"><uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq></condition>",
			expected: "cat hasPlural cats",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "RDF in condition false",
			template: "<condition name=\"test\" value=\"false\"><uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq></condition>",
			expected: "",
			setup: func(g *Golem, ctx *ChatSession) {
				g.ProcessTemplateWithContext(`<set name="test">true</set>`, map[string]string{}, ctx)
			},
		},
		{
			name:     "RDF with multiple conditions",
			template: "<condition name=\"test1\" value=\"true\"><uniq><subj>cat</subj><pred>hasPlural</pred><obj>cats</obj></uniq></condition> <condition name=\"test2\" value=\"true\"><uniq><subj>dog</subj><pred>hasPlural</pred><obj>dogs</obj></uniq></condition>",
			expected: "cat hasPlural cats",
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
