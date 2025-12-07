package golem

import (
	"testing"
	"time"
)

// TestTreeProcessorThinkTag tests the <think> tag with tree processor
func TestTreeProcessorThinkTag(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Enable AST-based processing

	tests := []struct {
		name         string
		template     string
		expectedOut  string
		expectedVars map[string]string
	}{
		{
			name:         "Basic think with set",
			template:     "<think><set name=\"test\">value</set></think>Output",
			expectedOut:  "Output",
			expectedVars: map[string]string{"test": "value"},
		},
		{
			name:         "Think with multiple sets",
			template:     "<think><set name=\"a\">1</set><set name=\"b\">2</set></think>Result",
			expectedOut:  "Result",
			expectedVars: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:         "Think produces no output",
			template:     "Before<think><set name=\"x\">y</set>Hidden text</think>After",
			expectedOut:  "BeforeAfter",
			expectedVars: map[string]string{"x": "y"},
		},
		{
			name:         "Think with nested tags",
			template:     "<think><set name=\"upper\"><uppercase>hello</uppercase></set></think>Done",
			expectedOut:  "Done",
			expectedVars: map[string]string{"upper": "HELLO"},
		},
		{
			name:         "Empty think tag",
			template:     "<think></think>Output",
			expectedOut:  "Output",
			expectedVars: map[string]string{},
		},
		{
			name:         "Think with get and set",
			template:     "<set name=\"base\">test</set><think><set name=\"result\"><get name=\"base\"/></set></think><get name=\"result\"/>",
			expectedOut:  "test",
			expectedVars: map[string]string{"base": "test", "result": "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_think_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expectedOut {
				t.Errorf("Expected output '%s', got '%s'", tt.expectedOut, result)
			}

			for varName, varValue := range tt.expectedVars {
				if session.Variables[varName] != varValue {
					t.Errorf("Expected variable '%s' = '%s', got '%s'", varName, varValue, session.Variables[varName])
				}
			}
		})
	}
}

// TestTreeProcessorThinkTagIntegration tests think tag integration with AIML categories
func TestTreeProcessorThinkTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>SET MEMORY *</pattern>
		<template><think><set name="memory"><star/></set></think>I will remember that.</template>
	</category>
	
	<category>
		<pattern>WHAT IS MY MEMORY</pattern>
		<template>You told me: <get name="memory"/></template>
	</category>
	
	<category>
		<pattern>COUNT TO *</pattern>
		<template><think><set name="max"><star/></set></think>Counting to <get name="max"/></template>
	</category>
	
	<category>
		<pattern>SILENT SET *</pattern>
		<template><think><set name="silent"><star/></set>This text should not appear</think></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-think",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	tests := []struct {
		name        string
		input       string
		expected    string
		checkVar    string
		expectedVar string
	}{
		{
			name:        "Set memory",
			input:       "set memory hello world",
			expected:    "I will remember that.",
			checkVar:    "memory",
			expectedVar: "hello world",
		},
		{
			name:     "Retrieve memory",
			input:    "what is my memory",
			expected: "You told me: hello world",
		},
		{
			name:        "Count with set",
			input:       "count to 5",
			expected:    "Counting to 5",
			checkVar:    "max",
			expectedVar: "5",
		},
		{
			name:        "Silent set produces no output",
			input:       "silent set secret",
			expected:    "",
			checkVar:    "silent",
			expectedVar: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("Failed to process input: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}

			if tt.checkVar != "" {
				if session.Variables[tt.checkVar] != tt.expectedVar {
					t.Errorf("Expected variable '%s' = '%s', got '%s'", tt.checkVar, tt.expectedVar, session.Variables[tt.checkVar])
				}
			}
		})
	}
}

// TestTreeProcessorThinkTagWithWildcards tests think tag with wildcards
func TestTreeProcessorThinkTagWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>REMEMBER * AS *</pattern>
		<template><think><set name="<star/>"><star index="2"/></set></think>Stored <star/> = <star index="2"/></template>
	</category>
	
	<category>
		<pattern>RECALL *</pattern>
		<template><get name="<star/>"/></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-wildcards",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	// Store a value
	response, err := g.ProcessInput("remember color as blue", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	if response != "Stored color = blue" {
		t.Errorf("Expected 'Stored color = blue', got '%s'", response)
	}

	if session.Variables["color"] != "blue" {
		t.Errorf("Expected variable 'color' = 'blue', got '%s'", session.Variables["color"])
	}

	// Recall the value
	response, err = g.ProcessInput("recall color", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	if response != "blue" {
		t.Errorf("Expected 'blue', got '%s'", response)
	}
}

// TestTreeProcessorThinkTagEdgeCases tests edge cases for think tag
func TestTreeProcessorThinkTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name        string
		template    string
		expectedOut string
	}{
		{
			name:        "Multiple think tags",
			template:    "<think><set name=\"a\">1</set></think>A<think><set name=\"b\">2</set></think>B",
			expectedOut: "AB",
		},
		{
			name:        "Nested content in think",
			template:    "<think>Text <uppercase>hidden</uppercase> more text</think>Visible",
			expectedOut: "Visible",
		},
		{
			name:        "Think at start",
			template:    "<think><set name=\"x\">y</set></think>Output",
			expectedOut: "Output",
		},
		{
			name:        "Think at end",
			template:    "Output<think><set name=\"x\">y</set></think>",
			expectedOut: "Output",
		},
		{
			name:        "Think in middle",
			template:    "Before<think><set name=\"x\">y</set></think>After",
			expectedOut: "BeforeAfter",
		},
		{
			name:        "Only think tag",
			template:    "<think><set name=\"x\">y</set></think>",
			expectedOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_edge_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expectedOut {
				t.Errorf("Expected '%s', got '%s'", tt.expectedOut, result)
			}
		})
	}
}
