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
				result := g.processSizeTagsWithContext(tt.template, ctx)
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
				result := g.processSizeTagsWithContext(tt.template, ctx)
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			})
		}
	}
}

func TestVersionTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	tests := []struct {
		name        string
		template    string
		version     string
		expected    string
		description string
	}{
		{
			name:        "Basic version tag with version set",
			template:    "I am running AIML version <version/>.",
			version:     "2.0",
			expected:    "I am running AIML version 2.0.",
			description: "Should return the set version",
		},
		{
			name:        "Version tag with custom version",
			template:    "My AIML version is <version/>.",
			version:     "1.0",
			expected:    "My AIML version is 1.0.",
			description: "Should return custom version",
		},
		{
			name:        "Version tag with no version set",
			template:    "Version: <version/>",
			version:     "",
			expected:    "Version: 2.0",
			description: "Should default to 2.0 when no version is set",
		},
		{
			name:        "Multiple version tags",
			template:    "AIML <version/> is compatible with <version/>.",
			version:     "2.1",
			expected:    "AIML 2.1 is compatible with 2.1.",
			description: "Should replace all version tags",
		},
		{
			name:        "Version tag with other content",
			template:    "I support AIML <version/> and can help you!",
			version:     "2.0",
			expected:    "I support AIML 2.0 and can help you!",
			description: "Should work with mixed content",
		},
		{
			name:        "Version tag in question",
			template:    "What version? <version/> is my version.",
			version:     "3.0",
			expected:    "What version? 3.0 is my version.",
			description: "Should work in questions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up knowledge base with version
			if tt.version != "" {
				g.aimlKB.Properties["version"] = tt.version
			} else {
				delete(g.aimlKB.Properties, "version")
			}

			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			result := g.processVersionTagsWithContext(tt.template, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
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
			result := g.processSizeTagsWithContext(tt.template, ctx)
			result = g.processVersionTagsWithContext(result, ctx)

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
			result := g.processBotTagsWithContext(tt.template, ctx)
			result = g.processSizeTagsWithContext(result, ctx)
			result = g.processVersionTagsWithContext(result, ctx)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSizeVersionEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	tests := []struct {
		name        string
		template    string
		expected    string
		description string
	}{
		{
			name:        "No size tags",
			template:    "Hello world!",
			expected:    "Hello world!",
			description: "Should not change template without size tags",
		},
		{
			name:        "No version tags",
			template:    "Hello world!",
			expected:    "Hello world!",
			description: "Should not change template without version tags",
		},
		{
			name:        "Empty template",
			template:    "",
			expected:    "",
			description: "Should handle empty template",
		},
		{
			name:        "Only size tag",
			template:    "<size/>",
			expected:    "0",
			description: "Should return size for template with only size tag",
		},
		{
			name:        "Only version tag",
			template:    "<version/>",
			expected:    "2.0",
			description: "Should return default version for template with only version tag",
		},
		{
			name:        "Size tag with no knowledge base",
			template:    "Size: <size/>",
			expected:    "Size: 0",
			description: "Should return 0 when no knowledge base",
		},
		{
			name:        "Version tag with no knowledge base",
			template:    "Version: <version/>",
			expected:    "Version: 2.0",
			description: "Should return default version when no knowledge base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx *VariableContext

			if tt.name == "Size tag with no knowledge base" || tt.name == "Version tag with no knowledge base" {
				// Test with nil knowledge base
				ctx = &VariableContext{
					LocalVars:     make(map[string]string),
					Session:       nil,
					Topic:         "",
					KnowledgeBase: nil,
				}
			} else {
				// Test with knowledge base
				ctx = &VariableContext{
					LocalVars:     make(map[string]string),
					Session:       nil,
					Topic:         "",
					KnowledgeBase: g.aimlKB,
				}
			}

			// Process both tags
			result := g.processSizeTagsWithContext(tt.template, ctx)
			result = g.processVersionTagsWithContext(result, ctx)

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
	result := g.processSizeTagsWithContext(template, ctx)
	result = g.processVersionTagsWithContext(result, ctx)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestIdTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	tests := []struct {
		name        string
		template    string
		sessionID   string
		expected    string
		description string
	}{
		{
			name:        "Basic id tag with session",
			template:    "Hello user <id/>!",
			sessionID:   "user123",
			expected:    "Hello user user123!",
			description: "Should return the session ID",
		},
		{
			name:        "Multiple id tags",
			template:    "User <id/> has session <id/>.",
			sessionID:   "session456",
			expected:    "User session456 has session session456.",
			description: "Should replace all id tags with session ID",
		},
		{
			name:        "Id tag with other content",
			template:    "Welcome <id/>! How can I help you?",
			sessionID:   "guest789",
			expected:    "Welcome guest789! How can I help you?",
			description: "Should work with mixed content",
		},
		{
			name:        "Id tag in question",
			template:    "What is your ID? <id/> is your session.",
			sessionID:   "admin001",
			expected:    "What is your ID? admin001 is your session.",
			description: "Should work in questions",
		},
		{
			name:        "Id tag with empty session ID",
			template:    "Session: <id/>",
			sessionID:   "",
			expected:    "Session: ",
			description: "Should handle empty session ID",
		},
		{
			name:        "Only id tag",
			template:    "<id/>",
			sessionID:   "test123",
			expected:    "test123",
			description: "Should return session ID for template with only id tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test session
			session := &ChatSession{
				ID:              tt.sessionID,
				Variables:       make(map[string]string),
				History:         []string{},
				CreatedAt:       "now",
				LastActivity:    "now",
				RequestHistory:  []string{},
				ResponseHistory: []string{},
			}

			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       session,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			result := g.processIdTagsWithContext(tt.template, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIdTagWithNoSession(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Id tag with no session",
			template: "Hello <id/>!",
			expected: "Hello !",
		},
		{
			name:     "Multiple id tags with no session",
			template: "User <id/> has session <id/>.",
			expected: "User  has session .",
		},
		{
			name:     "Only id tag with no session",
			template: "<id/>",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil, // No session
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			result := g.processIdTagsWithContext(tt.template, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIdSizeVersionIntegration(t *testing.T) {
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

	// Create a test session
	session := &ChatSession{
		ID:              "integration_test",
		Variables:       make(map[string]string),
		History:         []string{},
		CreatedAt:       "now",
		LastActivity:    "now",
		RequestHistory:  []string{},
		ResponseHistory: []string{},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "All system tags together",
			template: "User <id/> has <size/> patterns in AIML <version/>.",
			expected: "User integration_test has 3 patterns in AIML 2.0.",
		},
		{
			name:     "Id and size together",
			template: "Session <id/> contains <size/> patterns.",
			expected: "Session integration_test contains 3 patterns.",
		},
		{
			name:     "Id and version together",
			template: "User <id/> is using AIML <version/>.",
			expected: "User integration_test is using AIML 2.0.",
		},
		{
			name:     "Complex template with all tags",
			template: "Welcome <id/>! I have <size/> patterns in AIML <version/>. How can I help you?",
			expected: "Welcome integration_test! I have 3 patterns in AIML 2.0. How can I help you?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       session,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			// Process all system tags
			result := g.processIdTagsWithContext(tt.template, ctx)
			result = g.processSizeTagsWithContext(result, ctx)
			result = g.processVersionTagsWithContext(result, ctx)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIdTagWithBotTags(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test data
	g.aimlKB.Properties["name"] = "TestBot"
	g.aimlKB.Properties["version"] = "2.0"

	// Create a test session
	session := &ChatSession{
		ID:              "bot_test_user",
		Variables:       make(map[string]string),
		History:         []string{},
		CreatedAt:       "now",
		LastActivity:    "now",
		RequestHistory:  []string{},
		ResponseHistory: []string{},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Id with bot tag",
			template: "I am <bot name=\"name\"/> and you are <id/>.",
			expected: "I am TestBot and you are bot_test_user.",
		},
		{
			name:     "All system tags together",
			template: "I am <bot name=\"name\"/> version <bot name=\"version\"/>. User <id/> has access to <size/> patterns.",
			expected: "I am TestBot version 2.0. User bot_test_user has access to 0 patterns.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       session,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			// Process all tags
			result := g.processBotTagsWithContext(tt.template, ctx)
			result = g.processIdTagsWithContext(result, ctx)
			result = g.processSizeTagsWithContext(result, ctx)
			result = g.processVersionTagsWithContext(result, ctx)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIdTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	tests := []struct {
		name        string
		template    string
		expected    string
		description string
	}{
		{
			name:        "No id tags",
			template:    "Hello world!",
			expected:    "Hello world!",
			description: "Should not change template without id tags",
		},
		{
			name:        "Empty template",
			template:    "",
			expected:    "",
			description: "Should handle empty template",
		},
		{
			name:        "Id tag with special characters in session ID",
			template:    "Session: <id/>",
			expected:    "Session: user@domain.com",
			description: "Should handle special characters in session ID",
		},
		{
			name:        "Id tag with numeric session ID",
			template:    "User ID: <id/>",
			expected:    "User ID: 12345",
			description: "Should handle numeric session IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var session *ChatSession

			if tt.name == "Id tag with special characters in session ID" {
				session = &ChatSession{
					ID:              "user@domain.com",
					Variables:       make(map[string]string),
					History:         []string{},
					CreatedAt:       "now",
					LastActivity:    "now",
					RequestHistory:  []string{},
					ResponseHistory: []string{},
				}
			} else if tt.name == "Id tag with numeric session ID" {
				session = &ChatSession{
					ID:              "12345",
					Variables:       make(map[string]string),
					History:         []string{},
					CreatedAt:       "now",
					LastActivity:    "now",
					RequestHistory:  []string{},
					ResponseHistory: []string{},
				}
			} else {
				session = nil
			}

			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       session,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}

			result := g.processIdTagsWithContext(tt.template, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
