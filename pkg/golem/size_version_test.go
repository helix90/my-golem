package golem

import (
	"testing"
)

func TestSizeTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Add some test categories
	testCategories := []Category{
		{Pattern: "HELLO", Template: "Hi there!"},
		{Pattern: "HOW ARE YOU", Template: "I'm doing well, thank you!"},
		{Pattern: "WHAT IS YOUR NAME", Template: "I'm GolemBot."},
		{Pattern: "GOODBYE", Template: "See you later!"},
		{Pattern: "THANK YOU", Template: "You're welcome!"},
	}

	// Add categories to knowledge base
	g.aimlKB.Categories = append(g.aimlKB.Categories, testCategories...)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic size tag",
			template: "I have <size/> categories in my knowledge base.",
			expected: "I have 5 categories in my knowledge base.",
		},
		{
			name:     "Multiple size tags",
			template: "Size: <size/>. Total: <size/> patterns.",
			expected: "Size: 5. Total: 5 patterns.",
		},
		{
			name:     "Size tag with other content",
			template: "My knowledge base contains <size/> patterns and I can help you!",
			expected: "My knowledge base contains 5 patterns and I can help you!",
		},
		{
			name:     "Size tag in question",
			template: "How many patterns? <size/> patterns available.",
			expected: "How many patterns? 5 patterns available.",
		},
		{
			name:     "Size tag with mixed content",
			template: "I know <size/> things. What would you like to know?",
			expected: "I know 5 things. What would you like to know?",
		},
		{
			name:     "Empty knowledge base",
			template: "Size: <size/>",
			expected: "Size: 0",
		},
	}

	// Test with empty knowledge base first
	g.aimlKB.Categories = []Category{}
	for _, tt := range tests {
		if tt.name == "Empty knowledge base" {
			t.Run(tt.name, func(t *testing.T) {
				ctx := &VariableContext{
					LocalVars:     make(map[string]string),
					Session:       nil,
					Topic:         "",
					KnowledgeBase: g.aimlKB,
				}
				result := processTemplateForTest(g, tt.template, nil, ctx)
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			})
		}
	}

	// Restore test categories
	g.aimlKB.Categories = append(g.aimlKB.Categories, testCategories...)

	// Test with populated knowledge base
	for _, tt := range tests {
		if tt.name != "Empty knowledge base" {
			t.Run(tt.name, func(t *testing.T) {
				ctx := &VariableContext{
					LocalVars:     make(map[string]string),
					Session:       nil,
					Topic:         "",
					KnowledgeBase: g.aimlKB,
				}
				result := processTemplateForTest(g, tt.template, nil, ctx)
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			})
		}
	}
}

func TestSizeVersionIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test data
	g.aimlKB.Properties["version"] = "2.0"
	testCategories := []Category{
		{Pattern: "HELLO", Template: "Hi there!"},
		{Pattern: "HOW ARE YOU", Template: "I'm doing well!"},
		{Pattern: "WHAT IS YOUR NAME", Template: "I'm GolemBot."},
	}
	g.aimlKB.Categories = testCategories

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Size and version together",
			template: "I have <size/> patterns in AIML <version/>.",
			expected: "I have 3 patterns in AIML 2.0.",
		},
		{
			name:     "Version and size together",
			template: "AIML <version/> with <size/> categories.",
			expected: "AIML 2.0 with 3 categories.",
		},
		{
			name:     "Multiple size and version tags",
			template: "Size: <size/>, Version: <version/>, Total: <size/> patterns.",
			expected: "Size: 3, Version: 2.0, Total: 3 patterns.",
		},
		{
			name:     "Complex template with both tags",
			template: "My knowledge base has <size/> patterns using AIML <version/>. I can help you with <size/> different topics!",
			expected: "My knowledge base has 3 patterns using AIML 2.0. I can help you with 3 different topics!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			// Process both size and version tags
			result := processTemplateForTest(g, tt.template, nil, ctx)
			result = processTemplateForTest(g, result, nil, ctx)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSizeVersionWithOtherTags(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test data
	g.aimlKB.Properties["version"] = "2.0"
	g.aimlKB.Properties["name"] = "TestBot"
	testCategories := []Category{
		{Pattern: "HELLO", Template: "Hi there!"},
		{Pattern: "HOW ARE YOU", Template: "I'm doing well!"},
	}
	g.aimlKB.Categories = testCategories

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Size with bot tag",
			template: "I am <bot name=\"name\"/> with <size/> patterns.",
			expected: "I am TestBot with 2 patterns.",
		},
		{
			name:     "Version with bot tag",
			template: "I am <bot name=\"name\"/> running AIML <version/>.",
			expected: "I am TestBot running AIML 2.0.",
		},
		{
			name:     "All system tags together",
			template: "I am <bot name=\"name\"/> with <size/> patterns in AIML <version/>.",
			expected: "I am TestBot with 2 patterns in AIML 2.0.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			// Process all tags
			result := processTemplateForTest(g, tt.template, nil, ctx)
			result = processTemplateForTest(g, result, nil, ctx)
			result = processTemplateForTest(g, result, nil, ctx)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSizeVersionPerformance(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test data with many categories
	g.aimlKB.Properties["version"] = "2.0"
	for i := 0; i < 1000; i++ {
		category := Category{
			Pattern:  "PATTERN" + string(rune(i)),
			Template: "Response " + string(rune(i)),
		}
		g.aimlKB.Categories = append(g.aimlKB.Categories, category)
	}

	template := "I have <size/> patterns in AIML <version/>."
	expected := "I have 1000 patterns in AIML 2.0."

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       nil,
		Topic:         "",
		KnowledgeBase: g.aimlKB,
	}

	// Process both tags
	result := processTemplateForTest(g, template, nil, ctx)
	result = processTemplateForTest(g, result, nil, ctx)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
