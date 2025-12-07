package golem

import (
	"strings"
	"testing"
	"time"
)

// TestTreeProcessorInputTagIntegration tests input tag in full AIML conversation flow
func TestTreeProcessorInputTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML with input tag patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>ECHO</pattern>
		<template>You said: <input/></template>
	</category>
	
	<category>
		<pattern>REPEAT</pattern>
		<template>Repeating: <input/></template>
	</category>
	
	<category>
		<pattern>UPPERCASE</pattern>
		<template><uppercase><input/></uppercase></template>
	</category>
	
	<category>
		<pattern>WHAT DID I SAY</pattern>
		<template>You said: <input/></template>
	</category>
	
	<category>
		<pattern>REMEMBER</pattern>
		<template><think><set name="lastinput"><input/></set></think>I'll remember that you said: <input/></template>
	</category>
	
	<category>
		<pattern>WHAT DO YOU REMEMBER</pattern>
		<template>You previously said: <get name="lastinput"/></template>
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

	// Test conversation with input tags
	// Note: <input/> returns the PREVIOUS input (last item in RequestHistory)
	// because ProcessInput adds the current input to history AFTER template processing
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "echo",
			expected: "You said:", // No previous input, trailing space trimmed by tree processor
		},
		{
			input:    "repeat",
			expected: "Repeating: echo", // Previous input was "echo"
		},
		{
			input:    "uppercase",
			expected: "REPEAT", // Previous input was "repeat", uppercased
		},
		{
			input:    "what did i say",
			expected: "You said: uppercase", // Previous input was "uppercase"
		},
		{
			input:    "remember",
			expected: "I'll remember that you said: what did i say", // Previous was "what did i say"
		},
		{
			input:    "what do you remember",
			expected: "You previously said: what did i say", // Variable contains the input from before "remember"
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

// TestTreeProcessorInputTagWithTreeProcessor tests using TreeProcessor directly
func TestTreeProcessorInputTagWithTreeProcessor(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize AIML knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Create a session with request history
	session := &ChatSession{
		ID:           "test-tree-processor",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		RequestHistory: []string{
			"Hello, how are you?",
			"Tell me a story",
			"What's your name?",
		},
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple input reference",
			template: "You asked: <input/>",
			expected: "You asked: What's your name?",
		},
		{
			name:     "Input with uppercase",
			template: "<uppercase><input/></uppercase>",
			expected: "WHAT'S YOUR NAME?",
		},
		{
			name:     "Input with formal",
			template: "<formal><input/></formal>",
			expected: "What's Your Name?",
		},
		{
			name:     "Multiple input tags",
			template: "First: <input/>, Second: <input/>",
			expected: "First: What's your name?, Second: What's your name?",
		},
		{
			name:     "Input with length",
			template: "Length: <length><input/></length>",
			expected: "Length: 17", // "What's your name?" = 17 chars
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

// TestTreeProcessorInputTagConversationFlow tests realistic conversation flow
func TestTreeProcessorInputTagConversationFlow(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML with contextual responses using input tag
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>MY NAME IS *</pattern>
		<template>Nice to meet you, <star/>! You said: <input/></template>
	</category>
	
	<category>
		<pattern>I LIKE *</pattern>
		<template>Cool! You like <star/>. You said: <input/></template>
	</category>
	
	<category>
		<pattern>ECHO MY INPUT</pattern>
		<template>Your input was: <input/></template>
	</category>
	
	<category>
		<pattern>TELL ME</pattern>
		<template>Sure! But first, you said: <input/></template>
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
	// Note: <input/> returns the PREVIOUS input because ProcessInput adds to history AFTER template processing
	t.Run("Introduce name", func(t *testing.T) {
		response, err := g.ProcessInput("my name is Alice", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		expected := "Nice to meet you, Alice! You said:" // No previous input, trailing space trimmed by tree processor
		if response != expected {
			t.Errorf("Expected '%s', got '%s'", expected, response)
		}
	})

	t.Run("Express preference", func(t *testing.T) {
		response, err := g.ProcessInput("i like pizza", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		expected := "Cool! You like pizza. You said: my name is Alice" // Previous input
		if response != expected {
			t.Errorf("Expected '%s', got '%s'", expected, response)
		}
	})

	t.Run("Echo command", func(t *testing.T) {
		response, err := g.ProcessInput("echo my input", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		expected := "Your input was: i like pizza" // Previous input
		if response != expected {
			t.Errorf("Expected '%s', got '%s'", expected, response)
		}
	})

	t.Run("Tell me command", func(t *testing.T) {
		response, err := g.ProcessInput("tell me", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		expected := "Sure! But first, you said: echo my input" // Previous input
		if response != expected {
			t.Errorf("Expected '%s', got '%s'", expected, response)
		}
	})
}

// TestTreeProcessorInputTagWithVariables tests input tag interaction with variables
func TestTreeProcessorInputTagWithVariables(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize AIML knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Create a session with request history
	session := &ChatSession{
		ID:              "test-variables",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		RequestHistory:  []string{"Hello world"},
		ResponseHistory: make([]string, 0),
	}

	// Set a variable
	session.Variables["greeting"] = "Hi"

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
			name:     "Input with set variable",
			template: "<think><set name=\"user_input\"><input/></set></think>Saved: <get name=\"user_input\"/>",
			expected: "Saved: Hello world",
		},
		{
			name:     "Input with get variable",
			template: "Greeting: <get name=\"greeting\"/>, Input: <input/>",
			expected: "Greeting: Hi, Input: Hello world",
		},
		{
			name:     "Complex nesting",
			template: "<uppercase><input/></uppercase> and <get name=\"greeting\"/>",
			expected: "HELLO WORLD and Hi",
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

// TestTreeProcessorInputTagEmptyHistory tests input tag with empty request history
func TestTreeProcessorInputTagEmptyHistory(t *testing.T) {
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
		RequestHistory:  make([]string, 0), // Empty!
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Input with empty history",
			template: "Last input: '<input/>'",
			expected: "Last input: ''",
		},
		{
			name:     "Multiple input tags with empty history",
			template: "<input/> and <input/>",
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

// TestTreeProcessorInputTagComparisonWithRequest tests difference between input and request tags
func TestTreeProcessorInputTagComparisonWithRequest(t *testing.T) {
	g := NewForTesting(t, false)

	// Load AIML that uses both input and request tags
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>COMPARE</pattern>
		<template>Input: <input/>, Request 1: <request index="1"/>, Request 2: <request index="2"/></template>
	</category>
	
	<category>
		<pattern>SHOW ALL</pattern>
		<template>Current: <input/>, Previous: <request index="2"/>, Before that: <request index="3"/></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:              "test-comparison",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  []string{"first", "second"}, // Pre-populate with history
	}

	// Test that input always returns most recent FROM HISTORY (previous input during ProcessInput)
	t.Run("Compare tags", func(t *testing.T) {
		response, err := g.ProcessInput("compare", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		// At template processing time, history is ["first", "second"]
		// Input returns "second" (last in history)
		// Request 1 returns "second" (index 1, most recent)
		// Request 2 returns "first" (index 2, 2nd most recent)
		if !strings.Contains(response, "Input: second") {
			t.Errorf("Response should contain 'Input: second', got: %s", response)
		}
		if !strings.Contains(response, "Request 1: second") {
			t.Errorf("Response should contain 'Request 1: second', got: %s", response)
		}
		if !strings.Contains(response, "Request 2: first") {
			t.Errorf("Response should contain 'Request 2: first', got: %s", response)
		}
	})

	// Add more history manually (simulating another exchange)
	session.AddToRequestHistory("fourth")

	t.Run("Show all with more history", func(t *testing.T) {
		response, err := g.ProcessInput("show all", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		// At template processing time, history is ["first", "second", "compare", "fourth"]
		// Current (input) = "fourth" (last in history before "show all" is added)
		// Request 2 = "compare" (2nd most recent in history)
		// Request 3 = "second" (3rd most recent in history)
		if !strings.Contains(response, "Current: fourth") {
			t.Errorf("Response should contain 'Current: fourth', got: %s", response)
		}
		if !strings.Contains(response, "Previous: compare") {
			t.Errorf("Response should contain 'Previous: compare', got: %s", response)
		}
		if !strings.Contains(response, "Before that: second") {
			t.Errorf("Response should contain 'Before that: second', got: %s", response)
		}
	})
}
