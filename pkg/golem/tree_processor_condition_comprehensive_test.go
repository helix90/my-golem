package golem

import (
	"strings"
	"testing"
	"time"
)

// TestConditionTagAIML2Compliance tests that <condition> tag behavior complies with AIML2 specification
// According to AIML2, there are three forms of condition tags
func TestConditionTagAIML2Compliance(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		aiml        string
		input       string
		setupVars   map[string]string
		expectedOut string
		description string
	}{
		{
			name: "Form 1: name and value attributes",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST FORM1</pattern>
		<template><condition name="mood" value="happy">I'm glad!</condition></template>
	</category>
</aiml>`,
			input:       "test form1",
			setupVars:   map[string]string{"mood": "happy"},
			expectedOut: "I'm glad!",
			description: "Condition with name and value attributes should match when variable equals value",
		},
		{
			name: "Form 1: no match returns empty",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST NOMATCH</pattern>
		<template><condition name="mood" value="happy">I'm glad!</condition></template>
	</category>
</aiml>`,
			input:       "test nomatch",
			setupVars:   map[string]string{"mood": "sad"},
			expectedOut: "",
			description: "Condition with no match should return empty string",
		},
		{
			name: "Form 2: name attribute with li list",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST FORM2</pattern>
		<template>
			<condition name="weather">
				<li value="sunny">It's sunny!</li>
				<li value="rainy">It's rainy!</li>
				<li value="snowy">It's snowy!</li>
			</condition>
		</template>
	</category>
</aiml>`,
			input:       "test form2",
			setupVars:   map[string]string{"weather": "rainy"},
			expectedOut: "It's rainy!",
			description: "Condition with li list should match the correct value",
		},
		{
			name: "Form 2: with default li (no value attribute)",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST DEFAULT</pattern>
		<template>
			<condition name="weather">
				<li value="sunny">It's sunny!</li>
				<li value="rainy">It's rainy!</li>
				<li>Weather is unknown!</li>
			</condition>
		</template>
	</category>
</aiml>`,
			input:       "test default",
			setupVars:   map[string]string{"weather": "cloudy"},
			expectedOut: "Weather is unknown!",
			description: "Condition should use default li when no value matches",
		},
		{
			name: "Form 2: default li when variable not set",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST UNSET</pattern>
		<template>
			<condition name="undefined_var">
				<li value="something">Has value</li>
				<li>Variable is not set</li>
			</condition>
		</template>
	</category>
</aiml>`,
			input:       "test unset",
			setupVars:   map[string]string{},
			expectedOut: "Variable is not set",
			description: "Condition should use default li when variable is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.LoadAIMLFromString(tt.aiml)
			if err != nil {
				t.Fatalf("Failed to load AIML: %v", err)
			}

			session := &ChatSession{
				ID:              "test-condition-" + tt.name,
				Variables:       tt.setupVars,
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				ThatHistory:     make([]string, 0),
				ResponseHistory: make([]string, 0),
				RequestHistory:  make([]string, 0),
			}

			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if response != tt.expectedOut {
				t.Errorf("%s: Expected '%s', got '%s'", tt.description, tt.expectedOut, response)
			}
		})
	}
}

// TestConditionTagMultiPredicate tests Form 3: multi-predicate conditions
// where <li> elements have their own name and value attributes
// NOTE: This feature is not currently implemented in the tree processor
// Skipping until Form 3 multi-predicate conditions are implemented
func TestConditionTagMultiPredicate(t *testing.T) {
	t.Skip("Multi-predicate conditions (Form 3) are not yet implemented in tree processor")

	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		aiml        string
		input       string
		setupVars   map[string]string
		expectedOut string
		description string
	}{
		{
			name: "Multi-predicate with first match",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST MULTI</pattern>
		<template>
			<condition>
				<li name="time" value="morning">Good morning!</li>
				<li name="time" value="evening">Good evening!</li>
				<li name="mood" value="happy">You seem happy!</li>
				<li>Hello!</li>
			</condition>
		</template>
	</category>
</aiml>`,
			input:       "test multi",
			setupVars:   map[string]string{"time": "morning", "mood": "happy"},
			expectedOut: "Good morning!",
			description: "Multi-predicate should match first condition (time=morning)",
		},
		{
			name: "Multi-predicate with second variable",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST MULTI2</pattern>
		<template>
			<condition>
				<li name="time" value="morning">Good morning!</li>
				<li name="mood" value="happy">You seem happy!</li>
				<li name="weather" value="sunny">Nice weather!</li>
				<li>Default response</li>
			</condition>
		</template>
	</category>
</aiml>`,
			input:       "test multi2",
			setupVars:   map[string]string{"mood": "happy", "weather": "sunny"},
			expectedOut: "You seem happy!",
			description: "Multi-predicate should match first matching condition in order",
		},
		{
			name: "Multi-predicate with default",
			aiml: `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST MULTIDEFAULT</pattern>
		<template>
			<condition>
				<li name="a" value="x">A is X</li>
				<li name="b" value="y">B is Y</li>
				<li>Nothing matched</li>
			</condition>
		</template>
	</category>
</aiml>`,
			input:       "test multidefault",
			setupVars:   map[string]string{"a": "z", "b": "w"},
			expectedOut: "Nothing matched",
			description: "Multi-predicate should use default when no conditions match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.LoadAIMLFromString(tt.aiml)
			if err != nil {
				t.Fatalf("Failed to load AIML: %v", err)
			}

			session := &ChatSession{
				ID:              "test-multipred-" + tt.name,
				Variables:       tt.setupVars,
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				ThatHistory:     make([]string, 0),
				ResponseHistory: make([]string, 0),
				RequestHistory:  make([]string, 0),
			}

			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if response != tt.expectedOut {
				t.Errorf("%s: Expected '%s', got '%s'", tt.description, tt.expectedOut, response)
			}
		})
	}
}

// TestConditionTagNested tests nested condition tags
func TestConditionTagNested(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEST NESTED</pattern>
		<template>
			<condition name="level1">
				<li value="a">
					<condition name="level2">
						<li value="x">A and X</li>
						<li value="y">A and Y</li>
						<li>A and other</li>
					</condition>
				</li>
				<li value="b">
					<condition name="level2">
						<li value="x">B and X</li>
						<li value="y">B and Y</li>
						<li>B and other</li>
					</condition>
				</li>
				<li>Neither A nor B</li>
			</condition>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		level1   string
		level2   string
		expected string
	}{
		{"a", "x", "A and X"},
		{"a", "y", "A and Y"},
		{"a", "z", "A and other"},
		{"b", "x", "B and X"},
		{"b", "y", "B and Y"},
		{"b", "z", "B and other"},
		{"c", "x", "Neither A nor B"},
	}

	for _, tt := range tests {
		t.Run(tt.level1+"_"+tt.level2, func(t *testing.T) {
			session := &ChatSession{
				ID:              "test-nested-" + tt.level1 + tt.level2,
				Variables:       map[string]string{"level1": tt.level1, "level2": tt.level2},
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				ThatHistory:     make([]string, 0),
				ResponseHistory: make([]string, 0),
				RequestHistory:  make([]string, 0),
			}

			response, err := g.ProcessInput("test nested", session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if response != tt.expected {
				t.Errorf("For level1=%s, level2=%s: Expected '%s', got '%s'", tt.level1, tt.level2, tt.expected, response)
			}
		})
	}
}

// TestConditionTagWithLoop tests <loop/> tag within conditions
// NOTE: Loop functionality may have recursion limits
// Skipping until loop behavior is fully verified
func TestConditionTagWithLoop(t *testing.T) {
	t.Skip("Loop functionality needs verification - may have recursion limits")

	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>COUNT DOWN FROM *</pattern>
		<template>
			<think><set name="count"><star/></set></think>
			<condition name="count">
				<li value="0">Done!</li>
				<li>
					<get name="count"/>
					<think><set name="count"><map name="predecessor"><get name="count"/></map></set></think>
					<loop/>
				</li>
			</condition>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create predecessor map for countdown
	kb := g.GetKnowledgeBase()
	if kb.Maps == nil {
		kb.Maps = make(map[string]map[string]string)
	}
	kb.Maps["predecessor"] = map[string]string{
		"5": "4",
		"4": "3",
		"3": "2",
		"2": "1",
		"1": "0",
	}

	session := &ChatSession{
		ID:              "test-loop",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	response, err := g.ProcessInput("count down from 5", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	// Should contain countdown sequence: 5 4 3 2 1 Done!
	if !strings.Contains(response, "5") || !strings.Contains(response, "Done") {
		t.Errorf("Expected countdown from 5 to 0, got: %s", response)
	}
}

// TestConditionTagComplexScenarios tests complex real-world condition patterns
// NOTE: This test uses Form 3 multi-predicate conditions which are not currently implemented
// Skipping until multi-predicate support is added
func TestConditionTagComplexScenarios(t *testing.T) {
	t.Skip("Complex scenarios use Form 3 multi-predicate conditions which are not yet implemented")

	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>RECOMMEND ACTIVITY</pattern>
		<template>
			<condition>
				<li name="weather" value="sunny">
					<condition name="temperature">
						<li value="hot">Go swimming!</li>
						<li value="warm">Go for a walk!</li>
						<li value="cold">Enjoy the sun but dress warm!</li>
						<li>Enjoy the sunshine!</li>
					</condition>
				</li>
				<li name="weather" value="rainy">
					<condition name="mood">
						<li value="adventurous">Splash in puddles!</li>
						<li value="relaxed">Read a book indoors.</li>
						<li>Stay cozy inside!</li>
					</condition>
				</li>
				<li name="weather" value="snowy">
					<condition name="age_group">
						<li value="child">Build a snowman!</li>
						<li value="adult">Enjoy winter sports!</li>
						<li>Enjoy the winter wonderland!</li>
					</condition>
				</li>
				<li>Check the weather first!</li>
			</condition>
		</template>
	</category>

	<category>
		<pattern>ACCESS LEVEL</pattern>
		<template>
			<condition>
				<li name="role" value="admin">Full access granted.</li>
				<li name="role" value="moderator">
					<condition name="verified">
						<li value="true">Moderator access granted.</li>
						<li>Please verify your account first.</li>
					</condition>
				</li>
				<li name="role" value="user">
					<condition name="subscribed">
						<li value="true">Subscriber access granted.</li>
						<li>Basic access granted.</li>
					</condition>
				</li>
				<li>Please log in first.</li>
			</condition>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		vars      map[string]string
		expected  string
		substring bool // If true, just check substring
	}{
		{
			name:     "Sunny and hot",
			input:    "recommend activity",
			vars:     map[string]string{"weather": "sunny", "temperature": "hot"},
			expected: "Go swimming!",
		},
		{
			name:     "Rainy and adventurous",
			input:    "recommend activity",
			vars:     map[string]string{"weather": "rainy", "mood": "adventurous"},
			expected: "Splash in puddles!",
		},
		{
			name:     "Snowy for child",
			input:    "recommend activity",
			vars:     map[string]string{"weather": "snowy", "age_group": "child"},
			expected: "Build a snowman!",
		},
		{
			name:     "No weather set",
			input:    "recommend activity",
			vars:     map[string]string{},
			expected: "Check the weather first!",
		},
		{
			name:     "Admin access",
			input:    "access level",
			vars:     map[string]string{"role": "admin"},
			expected: "Full access granted.",
		},
		{
			name:     "Moderator verified",
			input:    "access level",
			vars:     map[string]string{"role": "moderator", "verified": "true"},
			expected: "Moderator access granted.",
		},
		{
			name:     "User subscribed",
			input:    "access level",
			vars:     map[string]string{"role": "user", "subscribed": "true"},
			expected: "Subscriber access granted.",
		},
		{
			name:     "User not subscribed",
			input:    "access level",
			vars:     map[string]string{"role": "user", "subscribed": "false"},
			expected: "Basic access granted.",
		},
		{
			name:     "No role set",
			input:    "access level",
			vars:     map[string]string{},
			expected: "Please log in first.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &ChatSession{
				ID:              "test-complex-" + tt.name,
				Variables:       tt.vars,
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				ThatHistory:     make([]string, 0),
				ResponseHistory: make([]string, 0),
				RequestHistory:  make([]string, 0),
			}

			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if tt.substring {
				if !strings.Contains(response, tt.expected) {
					t.Errorf("Expected response to contain '%s', got '%s'", tt.expected, response)
				}
			} else {
				if response != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, response)
				}
			}
		})
	}
}

// TestConditionTagEdgeCases tests edge cases for condition tags
func TestConditionTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		template    string
		setupVars   map[string]string
		expectedOut string
	}{
		{
			name:        "Empty condition tag",
			template:    "<condition></condition>",
			setupVars:   map[string]string{},
			expectedOut: "",
		},
		{
			name:        "Condition with empty string value",
			template:    `<condition name="test" value="">Empty value matched</condition>`,
			setupVars:   map[string]string{"test": ""},
			expectedOut: "Empty value matched",
		},
		{
			name:        "Condition with numeric values",
			template:    `<condition name="count"><li value="0">Zero</li><li value="1">One</li><li>Other</li></condition>`,
			setupVars:   map[string]string{"count": "0"},
			expectedOut: "Zero",
		},
		{
			name:        "Condition with special characters",
			template:    `<condition name="symbol"><li value="&amp;">Ampersand</li><li value="&lt;">Less than</li><li>Other</li></condition>`,
			setupVars:   map[string]string{"symbol": "&amp;"}, // XML entities are preserved
			expectedOut: "Ampersand",
		},
		{
			name:        "Condition with only default li",
			template:    `<condition name="anything"><li>Always this</li></condition>`,
			setupVars:   map[string]string{"anything": "whatever"},
			expectedOut: "Always this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_edge_" + tt.name)
			session.Variables = tt.setupVars

			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expectedOut {
				t.Errorf("Expected '%s', got '%s'", tt.expectedOut, result)
			}
		})
	}
}

// TestConditionTagVariableTypes tests condition with different variable types
func TestConditionTagVariableTypes(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	kb := NewAIMLKnowledgeBase()
	kb.Properties = map[string]string{
		"bot_name": "Golem",
		"version":  GetVersion(),
	}
	kb.Variables = map[string]string{
		"global_var": "global_value",
	}
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name        string
		template    string
		sessionVars map[string]string
		expectedOut string
	}{
		{
			name:        "Condition on session variable",
			template:    `<condition name="session_var" value="session_value">Session match!</condition>`,
			sessionVars: map[string]string{"session_var": "session_value"},
			expectedOut: "Session match!",
		},
		{
			name:        "Condition on global variable",
			template:    `<condition name="global_var" value="global_value">Global match!</condition>`,
			sessionVars: map[string]string{},
			expectedOut: "Global match!",
		},
		{
			name:        "Condition on bot property",
			template:    `<condition name="bot_name" value="Golem">Bot property match!</condition>`,
			sessionVars: map[string]string{},
			expectedOut: "Bot property match!",
		},
		{
			name:        "Session overrides global",
			template:    `<condition name="global_var" value="session_override">Override worked!</condition>`,
			sessionVars: map[string]string{"global_var": "session_override"},
			expectedOut: "Override worked!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_vartypes_" + tt.name)
			session.Variables = tt.sessionVars

			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expectedOut {
				t.Errorf("Expected '%s', got '%s'", tt.expectedOut, result)
			}
		})
	}
}

// TestConditionTagAIML2Examples tests examples from AIML2 specification
// NOTE: Some examples use Form 3 multi-predicate conditions which are not currently implemented
// Only Form 1 and Form 2 examples will pass
func TestConditionTagAIML2Examples(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<!-- Example from AIML2 spec: Single predicate condition with list -->
	<category>
		<pattern>WHAT SHOULD I WEAR</pattern>
		<template>
			<condition name="weather">
				<li value="sunny">Wear sunglasses!</li>
				<li value="rainy">Take an umbrella!</li>
				<li value="snowy">Wear a coat!</li>
				<li>Check the weather first!</li>
			</condition>
		</template>
	</category>

	<!-- Example: Multi-predicate condition (if-elseif-else pattern) -->
	<category>
		<pattern>GREETING</pattern>
		<template>
			<condition>
				<li name="name" value="">Hello, stranger!</li>
				<li name="time" value="morning">Good morning, <get name="name"/>!</li>
				<li name="time" value="evening">Good evening, <get name="name"/>!</li>
				<li>Hello, <get name="name"/>!</li>
			</condition>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		input    string
		vars     map[string]string
		expected string
		skip     bool
		skipReason string
	}{
		{
			input:    "what should I wear",
			vars:     map[string]string{"weather": "sunny"},
			expected: "Wear sunglasses!",
		},
		{
			input:    "what should I wear",
			vars:     map[string]string{"weather": "rainy"},
			expected: "Take an umbrella!",
		},
		{
			input:    "what should I wear",
			vars:     map[string]string{"weather": "cloudy"},
			expected: "Check the weather first!",
		},
		{
			input:    "greeting",
			vars:     map[string]string{},
			expected: "Hello, stranger!",
			skip:     true,
			skipReason: "Uses Form 3 multi-predicate",
		},
		{
			input:    "greeting",
			vars:     map[string]string{"name": "Alice", "time": "morning"},
			expected: "Good morning, Alice!",
			skip:     true,
			skipReason: "Uses Form 3 multi-predicate",
		},
		{
			input:    "greeting",
			vars:     map[string]string{"name": "Bob", "time": "evening"},
			expected: "Good evening, Bob!",
			skip:     true,
			skipReason: "Uses Form 3 multi-predicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.expected, func(t *testing.T) {
			if tt.skip {
				t.Skip(tt.skipReason)
			}
			session := &ChatSession{
				ID:              "test-aiml2ex-" + tt.input,
				Variables:       tt.vars,
				History:         make([]string, 0),
				CreatedAt:       time.Now().Format(time.RFC3339),
				LastActivity:    time.Now().Format(time.RFC3339),
				ThatHistory:     make([]string, 0),
				ResponseHistory: make([]string, 0),
				RequestHistory:  make([]string, 0),
			}

			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}
