package golem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestTreeProcessorSRAITag tests the native AST implementation of the <srai> tag
func TestTreeProcessorSRAITag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Basic SRAI redirect",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>HELLO</pattern>
        <template>Hi there!</template>
    </category>
    <category>
        <pattern>GREET</pattern>
        <template><srai>HELLO</srai></template>
    </category>
</aiml>`,
			input:    "GREET",
			expected: "Hi there!",
		},
		{
			name: "SRAI with wildcard",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>MY NAME IS *</pattern>
        <template>Nice to meet you, <star/>!</template>
    </category>
    <category>
        <pattern>I AM *</pattern>
        <template><srai>MY NAME IS <star/></srai></template>
    </category>
</aiml>`,
			input:    "I AM ALICE",
			expected: "Nice to meet you, ALICE!",
		},
		{
			name: "SRAI chain",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>NAME</pattern>
        <template>Golem</template>
    </category>
    <category>
        <pattern>WHAT IS YOUR NAME</pattern>
        <template>My name is <srai>NAME</srai></template>
    </category>
    <category>
        <pattern>WHO ARE YOU</pattern>
        <template><srai>WHAT IS YOUR NAME</srai></template>
    </category>
</aiml>`,
			input:    "WHO ARE YOU",
			expected: "My name is Golem",
		},
		{
			name: "SRAI with multiple redirects",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>HELLO</pattern>
        <template>Hello!</template>
    </category>
    <category>
        <pattern>HI</pattern>
        <template>Hi!</template>
    </category>
    <category>
        <pattern>GREETING</pattern>
        <template><srai>HELLO</srai> and <srai>HI</srai></template>
    </category>
</aiml>`,
			input:    "GREETING",
			expected: "Hello! and Hi!",
		},
		{
			name: "SRAI with variables",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>GET NAME</pattern>
        <template><get name="username"/></template>
    </category>
    <category>
        <pattern>MY NAME IS *</pattern>
        <template><set name="username"><star/></set>Hello <srai>GET NAME</srai>!</template>
    </category>
</aiml>`,
			input:    "MY NAME IS BOB",
			expected: "Hello BOB!",
		},
		{
			name: "SRAI no match",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>HELLO</pattern>
        <template>Hi there!</template>
    </category>
    <category>
        <pattern>TEST</pattern>
        <template>Result: <srai>NONEXISTENT PATTERN</srai></template>
    </category>
</aiml>`,
			input:    "TEST",
			expected: "Result: NONEXISTENT PATTERN",
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

// TestTreeProcessorSRAIRecursion tests recursion depth handling
func TestTreeProcessorSRAIRecursion(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Create a circular SRAI that would cause infinite recursion
	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>A</pattern>
        <template><srai>B</srai></template>
    </category>
    <category>
        <pattern>B</pattern>
        <template><srai>C</srai></template>
    </category>
    <category>
        <pattern>C</pattern>
        <template><srai>D</srai></template>
    </category>
    <category>
        <pattern>D</pattern>
        <template><srai>E</srai></template>
    </category>
    <category>
        <pattern>E</pattern>
        <template><srai>F</srai></template>
    </category>
    <category>
        <pattern>F</pattern>
        <template><srai>G</srai></template>
    </category>
    <category>
        <pattern>G</pattern>
        <template><srai>H</srai></template>
    </category>
    <category>
        <pattern>H</pattern>
        <template><srai>I</srai></template>
    </category>
    <category>
        <pattern>I</pattern>
        <template><srai>J</srai></template>
    </category>
    <category>
        <pattern>J</pattern>
        <template>Final result</template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	// This should hit the recursion limit but not crash
	response, _ := g.ProcessInput("A", session)

	// Should eventually resolve to "Final result"
	if response != "Final result" {
		t.Errorf("Expected 'Final result', got '%s'", response)
	}
}

// TestTreeProcessorSRAIWithWildcards tests SRAI with complex wildcard patterns
func TestTreeProcessorSRAIWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SAY *</pattern>
        <template><star/></template>
    </category>
    <category>
        <pattern>REPEAT * TIMES *</pattern>
        <template><srai>SAY <star index="2"/></srai> repeated <star/> times</template>
    </category>
    <category>
        <pattern>ECHO *</pattern>
        <template><srai>SAY <star/></srai></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	tests := []struct {
		input    string
		expected string
	}{
		{"SAY HELLO", "HELLO"},
		{"ECHO WORLD", "WORLD"},
		{"REPEAT 3 TIMES HELLO", "HELLO repeated 3 times"},
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

// TestTreeProcessorSRAIIntegration tests complex SRAI usage patterns
func TestTreeProcessorSRAIIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>WHAT IS YOUR NAME</pattern>
        <template>I am Golem</template>
    </category>
    <category>
        <pattern>WHAT CAN YOU DO</pattern>
        <template>I can help you with various tasks. <srai>WHAT IS YOUR NAME</srai></template>
    </category>
    <category>
        <pattern>INTRO</pattern>
        <template>Hi there! <srai>WHAT IS YOUR NAME</srai></template>
    </category>
    <category>
        <pattern>GREETING</pattern>
        <template>Welcome! <srai>INTRO</srai></template>
    </category>
    <category>
        <pattern>NORMALIZE * AND *</pattern>
        <template><srai><star/></srai> and <srai><star index="2"/></srai></template>
    </category>
    <category>
        <pattern>HELLO</pattern>
        <template>Hello there</template>
    </category>
    <category>
        <pattern>GOODBYE</pattern>
        <template>Goodbye friend</template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	tests := []struct {
		input    string
		expected string
	}{
		{"WHAT IS YOUR NAME", "I am Golem"},
		{"WHAT CAN YOU DO", "I can help you with various tasks. I am Golem"},
		{"INTRO", "Hi there! I am Golem"},
		{"GREETING", "Welcome! Hi there! I am Golem"},
		{"NORMALIZE HELLO AND GOODBYE", "Hello there and Goodbye friend"},
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

// TestTreeProcessorSRAIXTag tests the native AST implementation of the <sraix> tag
func TestTreeProcessorSRAIXTag(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var requestData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			return
		}

		// Get input
		input, ok := requestData["input"].(string)
		if !ok {
			t.Error("Expected 'input' field in request body")
			return
		}

		// Send response based on input
		var message string
		if strings.Contains(strings.ToLower(input), "weather") {
			message = "It's sunny today!"
		} else if strings.Contains(strings.ToLower(input), "time") {
			message = "It's 3:00 PM"
		} else {
			message = "External response: " + input
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"message": message,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create Golem instance with SRAIX manager
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Configure SRAIX service
	config := &SRAIXConfig{
		Name:           "test_service",
		BaseURL:        server.URL,
		Method:         "POST",
		Timeout:        10,
		ResponseFormat: "json",
		ResponsePath:   "data.message",
	}
	_ = g.AddSRAIXConfig(config)

	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Basic SRAIX call",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>ASK EXTERNAL *</pattern>
        <template><sraix service="test_service"><star/></sraix></template>
    </category>
</aiml>`,
			input:    "ASK EXTERNAL HELLO",
			expected: "External response: HELLO",
		},
		{
			name: "SRAIX with default",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>ASK UNKNOWN *</pattern>
        <template><sraix service="nonexistent" default="Service unavailable"><star/></sraix></template>
    </category>
</aiml>`,
			input:    "ASK UNKNOWN TEST",
			expected: "Service unavailable",
		},
		{
			name: "SRAIX with wildcard content",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>WEATHER IN *</pattern>
        <template><sraix service="test_service">What is the weather in <star/></sraix></template>
    </category>
</aiml>`,
			input:    "WEATHER IN BOSTON",
			expected: "It's sunny today!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorSRAIXWithAttributes tests SRAIX with various attributes
func TestTreeProcessorSRAIXWithAttributes(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var requestData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			return
		}

		// Check for parameters
		var message string
		if botid, ok := requestData["botid"].(string); ok && botid != "" {
			message = "Bot " + botid + " responded"
		} else {
			message = "Default bot response"
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"message": message,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Configure SRAIX service
	config := &SRAIXConfig{
		Name:           "bot_service",
		BaseURL:        server.URL,
		Method:         "POST",
		Timeout:        10,
		ResponseFormat: "json",
		ResponsePath:   "data.message",
	}
	_ = g.AddSRAIXConfig(config)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>ASK BOT *</pattern>
        <template><sraix service="bot_service" botid="123"><star/></sraix></template>
    </category>
    <category>
        <pattern>ASK DEFAULT</pattern>
        <template><sraix service="bot_service">Test query</sraix></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	tests := []struct {
		input    string
		expected string
	}{
		{"ASK BOT HELLO", "Bot 123 responded"},
		{"ASK DEFAULT", "Default bot response"},
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

// TestTreeProcessorSRAIEdgeCases tests edge cases for SRAI processing
func TestTreeProcessorSRAIEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>EMPTY</pattern>
        <template></template>
    </category>
    <category>
        <pattern>TEST EMPTY</pattern>
        <template>Result: <srai>EMPTY</srai> end</template>
    </category>
    <category>
        <pattern>SPACE</pattern>
        <template>   </template>
    </category>
    <category>
        <pattern>TEST SPACE</pattern>
        <template>Before <srai>SPACE</srai> after</template>
    </category>
    <category>
        <pattern>TEST NO MATCH</pattern>
        <template>Result: <srai>NO SUCH PATTERN</srai></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	tests := []struct {
		input    string
		expected string
	}{
		{"TEST EMPTY", "Result:  end"},
		{"TEST SPACE", "Before  after"},
		{"TEST NO MATCH", "Result: NO SUCH PATTERN"},
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
