package golem

import (
	"strings"
	"testing"
	"time"
)

// TestThinkTagAIML2Compliance tests that <think> tag behavior complies with AIML2 specification
// According to AIML2: <think> processes its content but produces no output
func TestThinkTagAIML2Compliance(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		template    string
		expectedOut string
		description string
	}{
		{
			name:        "Think returns empty string",
			template:    "<think>This should not appear</think>",
			expectedOut: "",
			description: "Think tag with only text should return empty string",
		},
		{
			name:        "Think with set returns empty string",
			template:    "<think><set name=\"x\">value</set></think>",
			expectedOut: "",
			description: "Think tag with set operation should return empty string",
		},
		{
			name:        "Think processes but doesn't output",
			template:    "Start<think>Hidden</think>End",
			expectedOut: "StartEnd",
			description: "Think tag content should not appear in output",
		},
		{
			name:        "Think with complex content returns empty",
			template:    "<think><uppercase>hidden</uppercase> text <set name=\"y\">val</set></think>",
			expectedOut: "",
			description: "Think tag with complex nested content should return empty string",
		},
		{
			name:        "Multiple think tags",
			template:    "<think>A</think>X<think>B</think>Y<think>C</think>Z",
			expectedOut: "XYZ",
			description: "Multiple think tags should all be suppressed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_compliance_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expectedOut {
				t.Errorf("%s: Expected '%s', got '%s'", tt.description, tt.expectedOut, result)
			}

			// Ensure no think tag artifacts in output
			if strings.Contains(result, "<think") || strings.Contains(result, "</think>") {
				t.Errorf("Output contains think tag artifacts: %s", result)
			}
		})
	}
}

// TestThinkTagVariableScopes tests <think> with all variable scopes
func TestThinkTagVariableScopes(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name      string
		template  string
		checkVar  string
		checkVal  string
		scope     string
	}{
		{
			name:     "Session variable with name attribute",
			template: "<think><set name=\"session_var\">session_value</set></think>",
			checkVar: "session_var",
			checkVal: "session_value",
			scope:    "session",
		},
		{
			name:     "Local variable with var attribute",
			template: "<think><set var=\"local_var\">local_value</set></think>",
			checkVar: "local_var",
			checkVal: "local_value",
			scope:    "local",
		},
		{
			name:     "Multiple scope operations",
			template: "<think><set name=\"n1\">v1</set><set var=\"v1\">v2</set></think>",
			checkVar: "n1",
			checkVal: "v1",
			scope:    "both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_scopes_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			// Think tag should produce no output
			if result != "" {
				t.Errorf("Expected empty output, got '%s'", result)
			}

			// Check variable was set in appropriate scope
			if tt.scope == "session" || tt.scope == "both" {
				if session.Variables[tt.checkVar] != tt.checkVal {
					t.Errorf("Expected session variable '%s' = '%s', got '%s'",
						tt.checkVar, tt.checkVal, session.Variables[tt.checkVar])
				}
			}
		})
	}
}

// TestThinkTagWithConditions tests <think> containing <condition> tags
func TestThinkTagWithConditions(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>CONDITIONAL THINK *</pattern>
		<template>
			<think>
				<condition name="level">
					<li value="high"><set name="priority">urgent</set></li>
					<li value="low"><set name="priority">normal</set></li>
					<li><set name="priority">default</set></li>
				</condition>
			</think>
			Priority set.
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-conditions",
		Variables:       map[string]string{"level": "high"},
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	response, err := g.ProcessInput("conditional think test", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	if response != "Priority set." {
		t.Errorf("Expected 'Priority set.', got '%s'", response)
	}

	if session.Variables["priority"] != "urgent" {
		t.Errorf("Expected priority='urgent', got '%s'", session.Variables["priority"])
	}
}

// TestThinkTagWithTopicChanges tests <think> with topic manipulation
func TestThinkTagWithTopicChanges(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>START TOPIC *</pattern>
		<template>
			<think><set name="topic"><star/></set></think>
			Topic changed.
		</template>
	</category>

	<topic name="sports">
		<category>
			<pattern>FAVORITE</pattern>
			<template>I like basketball!</template>
		</category>
	</topic>

	<topic name="music">
		<category>
			<pattern>FAVORITE</pattern>
			<template>I like jazz!</template>
		</category>
	</topic>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test-topics")

	// Change to sports topic
	response, err := g.ProcessInput("start topic sports", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	if response != "Topic changed." {
		t.Errorf("Expected 'Topic changed.', got '%s'", response)
	}

	if session.Variables["topic"] != "sports" {
		t.Errorf("Expected topic='sports', got '%s'", session.Variables["topic"])
	}
}

// TestThinkTagWithTextTransformation tests <think> with text processing tags
func TestThinkTagWithTextTransformation(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name     string
		template string
		checkVar string
		checkVal string
	}{
		{
			name:     "Uppercase in think",
			template: "<think><set name=\"upper\"><uppercase>hello</uppercase></set></think>",
			checkVar: "upper",
			checkVal: "HELLO",
		},
		{
			name:     "Lowercase in think",
			template: "<think><set name=\"lower\"><lowercase>WORLD</lowercase></set></think>",
			checkVar: "lower",
			checkVal: "world",
		},
		{
			name:     "Formal in think",
			template: "<think><set name=\"formal\"><formal>hello world</formal></set></think>",
			checkVar: "formal",
			checkVal: "Hello World",
		},
		{
			name:     "Sentence in think",
			template: "<think><set name=\"sent\"><sentence>hello world</sentence></set></think>",
			checkVar: "sent",
			checkVal: "Hello world",
		},
		{
			name:     "Multiple transformations",
			template: "<think><set name=\"result\"><uppercase><lowercase>TEST</lowercase></uppercase></set></think>",
			checkVar: "result",
			checkVal: "TEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_transform_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != "" {
				t.Errorf("Expected empty output, got '%s'", result)
			}

			if session.Variables[tt.checkVar] != tt.checkVal {
				t.Errorf("Expected %s='%s', got '%s'", tt.checkVar, tt.checkVal, session.Variables[tt.checkVar])
			}
		})
	}
}

// TestThinkTagWithCollections tests <think> with map, list, array operations
func TestThinkTagWithCollections(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Create test map
	kb := NewAIMLKnowledgeBase()
	kb.Maps = make(map[string]map[string]string)
	kb.Maps["colors"] = map[string]string{
		"sky":   "blue",
		"grass": "green",
		"sun":   "yellow",
	}
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		template string
		checkVar string
		checkVal string
	}{
		{
			name:     "Map lookup in think",
			template: "<think><set name=\"color\"><map name=\"colors\">sky</map></set></think>",
			checkVar: "color",
			checkVal: "blue",
		},
		{
			name:     "Multiple map operations",
			template: "<think><set name=\"c1\"><map name=\"colors\">sky</map></set><set name=\"c2\"><map name=\"colors\">grass</map></set></think>",
			checkVar: "c1",
			checkVal: "blue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_collections_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != "" {
				t.Errorf("Expected empty output, got '%s'", result)
			}

			if session.Variables[tt.checkVar] != tt.checkVal {
				t.Errorf("Expected %s='%s', got '%s'", tt.checkVar, tt.checkVal, session.Variables[tt.checkVar])
			}
		})
	}
}

// TestThinkTagWithDateTime tests <think> with date and time tags
func TestThinkTagWithDateTime(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name     string
		template string
		checkVar string
	}{
		{
			name:     "Date in think",
			template: "<think><set name=\"today\"><date/></set></think>",
			checkVar: "today",
		},
		{
			name:     "Date with format in think",
			template: "<think><set name=\"formatted\"><date format=\"%Y\"/></set></think>",
			checkVar: "formatted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_datetime_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != "" {
				t.Errorf("Expected empty output, got '%s'", result)
			}

			// Just check that variable was set (don't check exact value since it's time-dependent)
			if session.Variables[tt.checkVar] == "" {
				t.Errorf("Expected %s to be set, but it's empty", tt.checkVar)
			}
		})
	}
}

// TestThinkTagNested tests nested <think> tags
func TestThinkTagNested(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		template    string
		expectedOut string
		checkVars   map[string]string
	}{
		{
			name:        "Think within think",
			template:    "<think><set name=\"outer\">out</set><think><set name=\"inner\">in</set></think></think>",
			expectedOut: "",
			checkVars:   map[string]string{"outer": "out", "inner": "in"},
		},
		{
			name:        "Multiple nested levels",
			template:    "Start<think>A<think>B<think>C</think></think></think>End",
			expectedOut: "StartEnd",
			checkVars:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_nested_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expectedOut {
				t.Errorf("Expected '%s', got '%s'", tt.expectedOut, result)
			}

			for varName, varValue := range tt.checkVars {
				if session.Variables[varName] != varValue {
					t.Errorf("Expected %s='%s', got '%s'", varName, varValue, session.Variables[varName])
				}
			}
		})
	}
}

// TestThinkTagWhitespace tests whitespace handling in <think>
func TestThinkTagWhitespace(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		template    string
		expectedOut string
	}{
		{
			name:        "Think with leading whitespace",
			template:    "   <think><set name=\"x\">y</set></think>Text",
			expectedOut: "   Text",
		},
		{
			name:        "Think with trailing whitespace",
			template:    "Text<think><set name=\"x\">y</set></think>   ",
			expectedOut: "Text", // Trailing whitespace is normalized/trimmed
		},
		{
			name:        "Think with internal whitespace",
			template:    "<think>   <set name=\"x\">y</set>   </think>Text",
			expectedOut: "Text",
		},
		{
			name: "Think with newlines",
			template: `Before
<think>
	<set name="x">y</set>
</think>
After`,
			expectedOut: "Before\n\nAfter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_whitespace_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expectedOut {
				t.Errorf("Expected '%q', got '%q'", tt.expectedOut, result)
			}
		})
	}
}

// TestThinkTagComplexScenarios tests complex real-world scenarios
func TestThinkTagComplexScenarios(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>REGISTER * EMAIL * PHONE *</pattern>
		<template>
			<think>
				<set name="username"><star/></set>
				<set name="email"><star index="2"/></set>
				<set name="phone"><star index="3"/></set>
				<set name="registered">true</set>
				<set name="registration_date"><date format="iso"/></set>
			</think>
			Welcome <get name="username"/>! You are now registered.
		</template>
	</category>

	<category>
		<pattern>MY PROFILE</pattern>
		<template>
			Username: <get name="username"/>
			Email: <get name="email"/>
			Phone: <get name="phone"/>
			Registered: <get name="registered"/>
		</template>
	</category>

	<category>
		<pattern>CALCULATE SUM * AND *</pattern>
		<template>
			<think>
				<set var="a"><star/></set>
				<set var="b"><star index="2"/></set>
				<set name="last_calculation">sum</set>
			</think>
			The sum would be calculated here (not implemented in this test).
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test-complex")

	// Test complex registration
	response, err := g.ProcessInput("register john email john@example.com phone 555-1234", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	if !strings.Contains(response, "Welcome john") {
		t.Errorf("Expected welcome message with username, got: %s", response)
	}

	// Verify all variables were set silently
	expectedVars := map[string]string{
		"username":   "john",
		"email":      "john@example.com",
		"phone":      "555-1234",
		"registered": "true",
	}

	for varName, expectedVal := range expectedVars {
		if session.Variables[varName] != expectedVal {
			t.Errorf("Expected %s='%s', got '%s'", varName, expectedVal, session.Variables[varName])
		}
	}

	// Verify registration_date was set
	if session.Variables["registration_date"] == "" {
		t.Error("Expected registration_date to be set")
	}

	// Test profile retrieval
	response, err = g.ProcessInput("my profile", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	if !strings.Contains(response, "john") || !strings.Contains(response, "john@example.com") {
		t.Errorf("Expected profile to contain user data, got: %s", response)
	}
}

// TestThinkTagPerformance tests <think> with many operations
func TestThinkTagPerformance(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Build template with many set operations in think
	var templateParts []string
	templateParts = append(templateParts, "<think>")
	for i := 0; i < 100; i++ {
		templateParts = append(templateParts, "<set name=\"var"+string(rune('0'+i%10))+"\">" + string(rune('a'+i%26)) + "</set>")
	}
	templateParts = append(templateParts, "</think>Done")
	template := strings.Join(templateParts, "")

	session := g.CreateSession("test-performance")
	result := g.ProcessTemplateWithContext(template, nil, session)

	if result != "Done" {
		t.Errorf("Expected 'Done', got '%s'", result)
	}

	// Verify some variables were set
	if len(session.Variables) == 0 {
		t.Error("Expected variables to be set")
	}
}

// TestThinkTagErrorHandling tests <think> with invalid content
func TestThinkTagErrorHandling(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		template    string
		shouldError bool
	}{
		{
			name:        "Think with invalid tag",
			template:    "<think><invalidtag>content</invalidtag></think>Text",
			shouldError: false, // Should process but ignore invalid tag
		},
		{
			name:        "Empty think",
			template:    "<think></think>Text",
			shouldError: false,
		},
		{
			name:        "Think with only whitespace",
			template:    "<think>   </think>Text",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_error_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			// Think should not cause errors, just process what it can
			if tt.shouldError {
				// If we expect an error but got a result, that's a problem
				t.Logf("Template processed without error: %s", result)
			} else {
				// Should not contain think tags in output
				if strings.Contains(result, "<think") || strings.Contains(result, "</think>") {
					t.Errorf("Output contains think tag artifacts: %s", result)
				}
			}
		})
	}
}

// TestThinkTagWithSRAIChaining tests <think> with complex SRAI chains
func TestThinkTagWithSRAIChaining(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>PROCESS *</pattern>
		<template>
			<think>
				<set name="input"><star/></set>
				<set name="processed"><srai>NORMALIZE <star/></srai></set>
			</think>
			Processed: <get name="processed"/>
		</template>
	</category>

	<category>
		<pattern>NORMALIZE *</pattern>
		<template><uppercase><star/></uppercase></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test-srai-chain")

	response, err := g.ProcessInput("process hello", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	if !strings.Contains(response, "HELLO") {
		t.Errorf("Expected processed result to contain 'HELLO', got: %s", response)
	}

	if session.Variables["processed"] != "HELLO" {
		t.Errorf("Expected processed='HELLO', got '%s'", session.Variables["processed"])
	}
}

// TestThinkTagAIML2Examples tests examples from AIML2 specification
func TestThinkTagAIML2Examples(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<!-- Example: State transition with think -->
	<category>
		<pattern>START GAME</pattern>
		<template>
			<think><set name="topic">game</set></think>
			Let's play! What's your move?
		</template>
	</category>

	<!-- Example: Silent variable storage -->
	<category>
		<pattern>MY NAME IS *</pattern>
		<template>
			<think><set name="username"><star/></set></think>
			Nice to meet you, <star/>!
		</template>
	</category>

	<!-- Example: Multiple silent operations -->
	<category>
		<pattern>INITIALIZE</pattern>
		<template>
			<think>
				<set name="status">ready</set>
				<set name="count">0</set>
				<set name="initialized">true</set>
			</think>
			System initialized.
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test-aiml2-examples")

	tests := []struct {
		input       string
		expectedOut string
		checkVars   map[string]string
	}{
		{
			input:       "start game",
			expectedOut: "Let's play! What's your move?",
			checkVars:   map[string]string{"topic": "game"},
		},
		{
			input:       "my name is alice",
			expectedOut: "Nice to meet you, alice!",
			checkVars:   map[string]string{"username": "alice"},
		},
		{
			input:       "initialize",
			expectedOut: "System initialized.",
			checkVars: map[string]string{
				"status":      "ready",
				"count":       "0",
				"initialized": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if response != tt.expectedOut {
				t.Errorf("Expected '%s', got '%s'", tt.expectedOut, response)
			}

			for varName, expectedVal := range tt.checkVars {
				if session.Variables[varName] != expectedVal {
					t.Errorf("Expected %s='%s', got '%s'", varName, expectedVal, session.Variables[varName])
				}
			}
		})
	}
}
