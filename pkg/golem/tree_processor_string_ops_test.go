package golem

import (
	"testing"
)

// TestTreeProcessorSubstringTag tests the native AST implementation of the <substring> tag
func TestTreeProcessorSubstringTag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Basic substring",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>TEST SUBSTRING</pattern>
        <template><substring start="0" end="5">Hello World</substring></template>
    </category>
</aiml>`,
			input:    "TEST SUBSTRING",
			expected: "Hello",
		},
		{
			name: "Substring with wildcard",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>FIRST THREE OF *</pattern>
        <template><substring start="0" end="3"><star/></substring></template>
    </category>
</aiml>`,
			input:    "FIRST THREE OF TESTING",
			expected: "TES",
		},
		{
			name: "Substring middle portion",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>EXTRACT MIDDLE</pattern>
        <template><substring start="6" end="11">Hello World</substring></template>
    </category>
</aiml>`,
			input:    "EXTRACT MIDDLE",
			expected: "World",
		},
		{
			name: "Substring from start",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>FROM START</pattern>
        <template><substring start="0" end="5">Programming</substring></template>
    </category>
</aiml>`,
			input:    "FROM START",
			expected: "Progr",
		},
		{
			name: "Substring to end",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>TO END</pattern>
        <template><substring start="4" end="100">Test</substring></template>
    </category>
</aiml>`,
			input:    "TO END",
			expected: "",
		},
		{
			name: "Substring with Unicode",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>UNICODE TEST</pattern>
        <template><substring start="0" end="3">Hello 世界</substring></template>
    </category>
</aiml>`,
			input:    "UNICODE TEST",
			expected: "Hel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorReplaceTag tests the native AST implementation of the <replace> tag
func TestTreeProcessorReplaceTag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Basic replace",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>TEST REPLACE</pattern>
        <template><replace search="World" replace="Universe">Hello World</replace></template>
    </category>
</aiml>`,
			input:    "TEST REPLACE",
			expected: "Hello Universe",
		},
		{
			name: "Replace with wildcard",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>CHANGE * TO *</pattern>
        <template><replace search="<star/>" replace="<star index='2'/>">I like APPLES</replace></template>
    </category>
</aiml>`,
			input:    "CHANGE APPLES TO ORANGES",
			expected: "I like ORANGES",
		},
		{
			name: "Replace multiple occurrences",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>REPLACE ALL</pattern>
        <template><replace search="a" replace="X">banana</replace></template>
    </category>
</aiml>`,
			input:    "REPLACE ALL",
			expected: "bXnXnX",
		},
		{
			name: "Replace with empty string",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>REMOVE SPACES</pattern>
        <template><replace search=" " replace="">Hello World Test</replace></template>
    </category>
</aiml>`,
			input:    "REMOVE SPACES",
			expected: "HelloWorldTest",
		},
		{
			name: "Replace case sensitive",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>CASE TEST</pattern>
        <template><replace search="hello" replace="HI">Hello hello HELLO</replace></template>
    </category>
</aiml>`,
			input:    "CASE TEST",
			expected: "Hello HI HELLO",
		},
		{
			name: "Replace with special characters",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SPECIAL CHARS</pattern>
        <template><replace search="." replace="!">Hello. World.</replace></template>
    </category>
</aiml>`,
			input:    "SPECIAL CHARS",
			expected: "Hello! World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorLengthTag tests the native AST implementation of the <length> tag
func TestTreeProcessorLengthTag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Length characters (default)",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>LENGTH TEST</pattern>
        <template><length>Hello World</length></template>
    </category>
</aiml>`,
			input:    "LENGTH TEST",
			expected: "11",
		},
		{
			name: "Length words",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT WORDS</pattern>
        <template><length type="words">Hello World Test</length></template>
    </category>
</aiml>`,
			input:    "COUNT WORDS",
			expected: "3",
		},
		{
			name: "Length characters explicit",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT CHARS</pattern>
        <template><length type="characters">Test</length></template>
    </category>
</aiml>`,
			input:    "COUNT CHARS",
			expected: "4",
		},
		{
			name: "Length with wildcard",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>LENGTH OF *</pattern>
        <template><length><star/></length></template>
    </category>
</aiml>`,
			input:    "LENGTH OF TESTING",
			expected: "7",
		},
		{
			name: "Length empty string",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>EMPTY LENGTH</pattern>
        <template><length></length></template>
    </category>
</aiml>`,
			input:    "EMPTY LENGTH",
			expected: "0",
		},
		{
			name: "Length with Unicode",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>UNICODE LENGTH</pattern>
        <template><length>Hello 世界</length></template>
    </category>
</aiml>`,
			input:    "UNICODE LENGTH",
			expected: "12",
		},
		{
			name: "Length sentences",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT SENTENCES</pattern>
        <template><length type="sentences">Hello world. How are you? I am fine.</length></template>
    </category>
</aiml>`,
			input:    "COUNT SENTENCES",
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorCountTag tests the native AST implementation of the <count> tag
func TestTreeProcessorCountTag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Basic count",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT TEST</pattern>
        <template><count search="l">Hello World</count></template>
    </category>
</aiml>`,
			input:    "COUNT TEST",
			expected: "3",
		},
		{
			name: "Count with wildcard",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT * IN *</pattern>
        <template><count search="<star/>"><star index="2"/></count></template>
    </category>
</aiml>`,
			input:    "COUNT A IN BANANA",
			expected: "3",
		},
		{
			name: "Count word occurrences",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT WORD</pattern>
        <template><count search="the">the quick brown fox jumps over the lazy dog</count></template>
    </category>
</aiml>`,
			input:    "COUNT WORD",
			expected: "2",
		},
		{
			name: "Count no matches",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT NONE</pattern>
        <template><count search="z">Hello World</count></template>
    </category>
</aiml>`,
			input:    "COUNT NONE",
			expected: "0",
		},
		{
			name: "Count case sensitive",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT CASE</pattern>
        <template><count search="H">Hello hello HELLO</count></template>
    </category>
</aiml>`,
			input:    "COUNT CASE",
			expected: "2",
		},
		{
			name: "Count empty search",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT EMPTY</pattern>
        <template><count search="">Test</count></template>
    </category>
</aiml>`,
			input:    "COUNT EMPTY",
			expected: "0",
		},
		{
			name: "Count overlapping",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>COUNT OVERLAP</pattern>
        <template><count search="aa">aaaa</count></template>
    </category>
</aiml>`,
			input:    "COUNT OVERLAP",
			expected: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorStringOpsIntegration tests complex integration scenarios
func TestTreeProcessorStringOpsIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>PROCESS *</pattern>
        <template>
            Original: <star/>
            Length: <length><star/></length>
            First 3: <substring start="0" end="3"><star/></substring>
            Replace A: <replace search="A" replace="X"><star/></replace>
            Count E: <count search="E"><star/></count>
        </template>
    </category>
    <category>
        <pattern>CHAIN *</pattern>
        <template><replace search=" " replace=""><substring start="0" end="5"><star/></substring></replace></template>
    </category>
    <category>
        <pattern>COMPUTE * AND *</pattern>
        <template>Length 1: <length><star/></length>, Length 2: <length><star index="2"/></length></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	tests := []struct {
		input    string
		expected string
	}{
		{"PROCESS TESTING", "Original: TESTING\n            Length: 7\n            First 3: TES\n            Replace A: TESTING\n            Count E: 1"},
		{"CHAIN HELLO WORLD", "HELLO"},
		{"COMPUTE ABC AND DEFGH", "Length 1: 3, Length 2: 5"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Input '%s': expected '%s', got '%s'", tt.input, tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorStringOpsWithVariables tests string operations with variables
func TestTreeProcessorStringOpsWithVariables(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>STORE *</pattern>
        <template><set name="text"><star/></set>Stored: <get name="text"/></template>
    </category>
    <category>
        <pattern>GET LENGTH</pattern>
        <template>Length: <length><get name="text"/></length></template>
    </category>
    <category>
        <pattern>REPLACE IN STORED * WITH *</pattern>
        <template><replace search="<star/>" replace="<star index='2'/>"><get name="text"/></replace></template>
    </category>
    <category>
        <pattern>COUNT * IN STORED</pattern>
        <template><count search="<star/>"><get name="text"/></count></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	// Store text
	response, _ := g.ProcessInput("STORE HELLO WORLD", session)
	if response != "Stored: HELLO WORLD" {
		t.Errorf("Store failed: expected 'Stored: HELLO WORLD', got '%s'", response)
	}

	// Get length
	response, _ = g.ProcessInput("GET LENGTH", session)
	if response != "Length: 11" {
		t.Errorf("Length failed: expected 'Length: 11', got '%s'", response)
	}

	// Replace
	response, _ = g.ProcessInput("REPLACE IN STORED WORLD WITH UNIVERSE", session)
	if response != "HELLO UNIVERSE" {
		t.Errorf("Replace failed: expected 'HELLO UNIVERSE', got '%s'", response)
	}

	// Count
	response, _ = g.ProcessInput("COUNT L IN STORED", session)
	if response != "3" {
		t.Errorf("Count failed: expected '3', got '%s'", response)
	}
}

// TestTreeProcessorStringOpsEdgeCases tests edge cases
func TestTreeProcessorStringOpsEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SUBSTRING BOUNDS</pattern>
        <template><substring start="-5" end="100">Test</substring></template>
    </category>
    <category>
        <pattern>REPLACE NOTHING</pattern>
        <template><replace search="xyz" replace="ABC">Hello</replace></template>
    </category>
    <category>
        <pattern>LENGTH SPACES</pattern>
        <template><length>   </length></template>
    </category>
    <category>
        <pattern>COUNT SELF</pattern>
        <template><count search="aa">aa</count></template>
    </category>
    <category>
        <pattern>EMPTY OPERATIONS</pattern>
        <template><substring start="0" end="0">Test</substring>|<replace search="Z" replace="X">Test</replace>|<length></length>|<count search="a"></count></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	tests := []struct {
		input    string
		expected string
	}{
		{"SUBSTRING BOUNDS", "Test"},
		{"REPLACE NOTHING", "Hello"},
		{"LENGTH SPACES", "0"},
		{"COUNT SELF", "1"},
		{"EMPTY OPERATIONS", "|Test|0|0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Input '%s': expected '%s', got '%s'", tt.input, tt.expected, response)
			}
		})
	}
}
