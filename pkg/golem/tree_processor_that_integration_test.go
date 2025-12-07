package golem

import (
	"strings"
	"testing"
	"time"
)

// TestTreeProcessorThatTagIntegration tests <that> tag in a full AIML conversation flow
func TestTreeProcessorThatTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML with that tag patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>HELLO</pattern>
		<template>Hello! How are you today?</template>
	</category>
	
	<category>
		<pattern>FINE</pattern>
		<template>You said you are fine. That's great! What would you like to talk about?</template>
	</category>
	
	<category>
		<pattern>WHAT DID YOU SAY</pattern>
		<template>I said: <that/></template>
	</category>
	
	<category>
		<pattern>REPEAT</pattern>
		<template>Sure, I said: <that/></template>
	</category>
	
	<category>
		<pattern>WHAT DID YOU SAY BEFORE</pattern>
		<template>Before that, I said: <that index="2"/></template>
	</category>
	
	<category>
		<pattern>TELL ME EVERYTHING</pattern>
		<template>Most recent: <that index="1"/>, Before: <that index="2"/>, Earlier: <that index="3"/></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:              "test-integration",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	// Simulate a conversation
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "hello",
			expected: "Hello! How are you today?",
		},
		{
			input:    "fine",
			expected: "You said you are fine. That's great! What would you like to talk about?",
		},
		{
			input:    "what did you say",
			expected: "I said: You said you are fine. That's great! What would you like to talk about?",
		},
		{
			input:    "repeat",
			expected: "Sure, I said: I said: You said you are fine. That's great! What would you like to talk about?",
		},
		{
			input:    "what did you say before",
			expected: "Before that, I said: I said: You said you are fine. That's great! What would you like to talk about?",
		},
		{
			input:    "tell me everything",
			expected: "Most recent: Before that, I said: I said: You said you are fine. That's great! What would you like to talk about?, Before: Sure, I said: I said: You said you are fine. That's great! What would you like to talk about?, Earlier: I said: You said you are fine. That's great! What would you like to talk about?",
		},
	}

	for i, tt := range tests {
		t.Run("Step_"+string(rune('A'+i)), func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input '%s': %v", tt.input, err)
			}

			if response != tt.expected {
				t.Errorf("Input: '%s'\nExpected: '%s'\nGot: '%s'", tt.input, tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorThatTagWithTreeProcessor tests using TreeProcessor directly
func TestTreeProcessorThatTagWithTreeProcessor(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize AIML knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Create a session with conversation history
	session := &ChatSession{
		ID:           "test-tree-processor",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
		ResponseHistory: []string{
			"Welcome to the chatbot!",
			"I can help you with many things.",
			"Just ask me anything.",
		},
		RequestHistory: make([]string, 0),
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple that reference",
			template: "You asked about: <that/>",
			expected: "You asked about: Just ask me anything.",
		},
		{
			name:     "That with uppercase",
			template: "<uppercase><that/></uppercase>",
			expected: "JUST ASK ME ANYTHING.",
		},
		{
			name:     "That with index",
			template: "Response 2 was: <that index=\"2\"/>",
			expected: "Response 2 was: I can help you with many things.",
		},
		{
			name:     "Multiple that references",
			template: "Last: <that/>, Previous: <that index=\"2\"/>",
			expected: "Last: Just ask me anything., Previous: I can help you with many things.",
		},
		{
			name:     "That in random structure",
			template: "<random><li>Recent: <that/></li><li>Last: <that/></li></random>",
			expected: "", // We'll check differently for random
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.ProcessTemplate(tt.template, make(map[string]string), ctx)
			if err != nil {
				t.Fatalf("Failed to process template: %v", err)
			}

			// Special handling for random test - just check it contains expected content
			if tt.name == "That in random structure" {
				// Result should be one of the two options
				if !strings.Contains(result, "Just ask me anything.") {
					t.Errorf("Random result should contain that tag output, got '%s'", result)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorThatTagConversationFlow tests realistic conversation flow
func TestTreeProcessorThatTagConversationFlow(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML with contextual responses using that tag
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>DO YOU LIKE *</pattern>
		<template>I'm not sure about <star/>. Do you like <star/>?</template>
	</category>
	
	<category>
		<pattern>YES</pattern>
		<template>Great! I'll remember that you like what we were discussing: <that/></template>
	</category>
	
	<category>
		<pattern>NO</pattern>
		<template>Okay, I understand you don't like what I mentioned: <that/></template>
	</category>
	
	<category>
		<pattern>WHAT WERE WE TALKING ABOUT</pattern>
		<template>We were discussing: <that index="2"/></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:              "test-conversation",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	// Conversation flow
	t.Run("Ask about pizza", func(t *testing.T) {
		response, err := g.ProcessInput("do you like pizza", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		expected := "I'm not sure about pizza. Do you like pizza?"
		if response != expected {
			t.Errorf("Expected '%s', got '%s'", expected, response)
		}
	})

	t.Run("Answer yes with that reference", func(t *testing.T) {
		response, err := g.ProcessInput("yes", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		// Should reference the previous question
		if !strings.Contains(response, "like") && !strings.Contains(response, "discussing") {
			t.Errorf("Response should reference previous statement: '%s'", response)
		}
	})

	t.Run("Ask about ice cream", func(t *testing.T) {
		response, err := g.ProcessInput("do you like ice cream", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		expected := "I'm not sure about ice cream. Do you like ice cream?"
		if response != expected {
			t.Errorf("Expected '%s', got '%s'", expected, response)
		}
	})

	t.Run("What were we talking about", func(t *testing.T) {
		response, err := g.ProcessInput("what were we talking about", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		// Should reference index 2 (two responses back)
		if !strings.Contains(response, "discussing") {
			t.Errorf("Response should reference earlier statement: '%s'", response)
		}
	})
}

// TestTreeProcessorThatTagWithVariables tests that tag interaction with variables
func TestTreeProcessorThatTagWithVariables(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize AIML knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Create a session with conversation history
	session := &ChatSession{
		ID:              "test-variables",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: []string{"My favorite color is blue."},
		RequestHistory:  make([]string, 0),
	}

	// Set a variable
	session.Variables["topic"] = "colors"

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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "That with set variable",
			template: "<think><set name=\"last_response\"><that/></set></think>Saved: <get name=\"last_response\"/>",
			expected: "Saved: My favorite color is blue.",
		},
		{
			name:     "That with get variable",
			template: "Topic: <get name=\"topic\"/>, Last: <that/>",
			expected: "Topic: colors, Last: My favorite color is blue.",
		},
		{
			name:     "Complex nesting",
			template: "<uppercase><that/></uppercase> in <get name=\"topic\"/>",
			expected: "MY FAVORITE COLOR IS BLUE. in colors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.ProcessTemplate(tt.template, make(map[string]string), ctx)
			if err != nil {
				t.Fatalf("Failed to process template: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorThatTagEmptyHistory tests that tag with empty response history
func TestTreeProcessorThatTagEmptyHistory(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize AIML knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Create a session with empty history
	session := &ChatSession{
		ID:              "test-empty",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0), // Empty!
		RequestHistory:  make([]string, 0),
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "That with empty history",
			template: "Last response: '<that/>'",
			expected: "Last response: ''",
		},
		{
			name:     "That with index on empty history",
			template: "Response: '<that index=\"1\"/>'",
			expected: "Response: ''",
		},
		{
			name:     "Multiple that tags with empty history",
			template: "<that/> and <that index=\"2\"/>",
			expected: " and", // Trailing space trimmed by tree processor
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.ProcessTemplate(tt.template, make(map[string]string), ctx)
			if err != nil {
				t.Fatalf("Failed to process template: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
