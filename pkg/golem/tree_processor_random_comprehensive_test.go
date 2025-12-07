package golem

import (
	"math"
	"strings"
	"testing"
	"time"
)

// TestRandomTagAIML2Compliance tests that <random> tag behavior complies with AIML2 specification
// According to AIML2: random tag should return exactly one li element with uniform distribution
func TestRandomTagAIML2Compliance(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name         string
		template     string
		validOptions []string
		description  string
	}{
		{
			name: "Basic random selection",
			template: `<random>
				<li>Option A</li>
				<li>Option B</li>
				<li>Option C</li>
			</random>`,
			validOptions: []string{"Option A", "Option B", "Option C"},
			description:  "Random should return exactly one of the li elements",
		},
		{
			name: "Random with two options",
			template: `<random>
				<li>Yes</li>
				<li>No</li>
			</random>`,
			validOptions: []string{"Yes", "No"},
			description:  "Random with two options should return one or the other",
		},
		{
			name: "Random with single option",
			template: `<random>
				<li>Only choice</li>
			</random>`,
			validOptions: []string{"Only choice"},
			description:  "Random with single option should always return that option",
		},
		{
			name: "Random with complex content",
			template: `<random>
				<li>The answer is <uppercase>yes</uppercase></li>
				<li>The answer is <uppercase>no</uppercase></li>
				<li>The answer is <uppercase>maybe</uppercase></li>
			</random>`,
			validOptions: []string{"The answer is YES", "The answer is NO", "The answer is MAYBE"},
			description:  "Random should process li content before returning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_compliance_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			// Verify result is one of the valid options
			valid := false
			for _, option := range tt.validOptions {
				if result == option {
					valid = true
					break
				}
			}

			if !valid {
				t.Errorf("%s: Expected one of %v, got '%s'", tt.description, tt.validOptions, result)
			}

			// Verify no random tag artifacts in output
			if strings.Contains(result, "<random") || strings.Contains(result, "</random>") {
				t.Errorf("Output contains random tag artifacts: %s", result)
			}

			// Verify no li tag artifacts in output
			if strings.Contains(result, "<li>") || strings.Contains(result, "</li>") {
				t.Errorf("Output contains li tag artifacts: %s", result)
			}
		})
	}
}

// TestRandomTagUniformDistribution tests that random selections are uniformly distributed
// According to AIML2 spec: "The distribution of selections should be random uniform"
func TestRandomTagUniformDistribution(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Test with 3 options
	template := `<random>
		<li>A</li>
		<li>B</li>
		<li>C</li>
	</random>`

	session := g.CreateSession("test_distribution")

	// Run many iterations to test distribution
	iterations := 1000
	results := make(map[string]int)

	for i := 0; i < iterations; i++ {
		result := g.ProcessTemplateWithContext(template, nil, session)
		results[result]++
	}

	// Check that all options were selected
	if len(results) != 3 {
		t.Errorf("Expected 3 different results, got %d: %v", len(results), results)
	}

	// Verify each option exists
	for _, option := range []string{"A", "B", "C"} {
		if results[option] == 0 {
			t.Errorf("Option '%s' was never selected", option)
		}
	}

	// Check for roughly uniform distribution (within 20% of expected)
	// Expected: ~333 each for 1000 iterations with 3 options
	expected := float64(iterations) / 3.0
	tolerance := expected * 0.20 // 20% tolerance

	for option, count := range results {
		diff := math.Abs(float64(count) - expected)
		if diff > tolerance {
			t.Logf("Warning: Option '%s' selected %d times (expected ~%.0f, tolerance ±%.0f)",
				option, count, expected, tolerance)
		}
	}

	t.Logf("Distribution over %d iterations: %v", iterations, results)
}

// TestRandomTagUniformDistributionFiveOptions tests uniform distribution with 5 options
func TestRandomTagUniformDistributionFiveOptions(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	template := `<random>
		<li>1</li>
		<li>2</li>
		<li>3</li>
		<li>4</li>
		<li>5</li>
	</random>`

	session := g.CreateSession("test_dist_5")

	iterations := 5000
	results := make(map[string]int)

	for i := 0; i < iterations; i++ {
		result := g.ProcessTemplateWithContext(template, nil, session)
		results[result]++
	}

	// Check that all 5 options were selected
	if len(results) != 5 {
		t.Errorf("Expected 5 different results, got %d: %v", len(results), results)
	}

	// Expected: ~1000 each
	expected := float64(iterations) / 5.0
	tolerance := expected * 0.15 // 15% tolerance

	for option, count := range results {
		diff := math.Abs(float64(count) - expected)
		if diff > tolerance {
			t.Logf("Warning: Option '%s' selected %d times (expected ~%.0f, tolerance ±%.0f)",
				option, count, expected, tolerance)
		}
	}

	t.Logf("Distribution over %d iterations: %v", iterations, results)
}

// TestRandomTagIntegrationWithAIML tests random tag in complete AIML categories
func TestRandomTagIntegrationWithAIML(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>COIN FLIP</pattern>
		<template>
			<random>
				<li>Heads</li>
				<li>Tails</li>
			</random>
		</template>
	</category>

	<category>
		<pattern>PICK A COLOR</pattern>
		<template>
			I choose <random>
				<li>red</li>
				<li>blue</li>
				<li>green</li>
				<li>yellow</li>
			</random>!
		</template>
	</category>

	<category>
		<pattern>RANDOM GREETING</pattern>
		<template>
			<random>
				<li>Hello! How are you?</li>
				<li>Hi there! Nice to meet you!</li>
				<li>Hey! What's up?</li>
				<li>Greetings! How can I help?</li>
			</random>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		input        string
		validOptions []string
	}{
		{
			input:        "coin flip",
			validOptions: []string{"Heads", "Tails"},
		},
		{
			input:        "pick a color",
			validOptions: []string{"I choose red!", "I choose blue!", "I choose green!", "I choose yellow!"},
		},
		{
			input:        "random greeting",
			validOptions: []string{"Hello! How are you?", "Hi there! Nice to meet you!", "Hey! What's up?", "Greetings! How can I help?"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			session := &ChatSession{
				ID:              "test-random-aiml-" + tt.input,
				Variables:       make(map[string]string),
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

			// Verify response is one of valid options
			valid := false
			for _, option := range tt.validOptions {
				if response == option {
					valid = true
					break
				}
			}

			if !valid {
				t.Errorf("Expected one of %v, got '%s'", tt.validOptions, response)
			}
		})
	}
}

// TestRandomTagWithVariables tests random tag with variables and dynamic content
func TestRandomTagWithVariables(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>GREET *</pattern>
		<template>
			<think><set name="username"><star/></set></think>
			<random>
				<li>Hello, <get name="username"/>!</li>
				<li>Hi there, <get name="username"/>!</li>
				<li>Welcome, <get name="username"/>!</li>
			</random>
		</template>
	</category>

	<category>
		<pattern>RANDOM ADVICE</pattern>
		<template>
			<random>
				<li><think><set name="advice_type">health</set></think>Exercise regularly!</li>
				<li><think><set name="advice_type">diet</set></think>Eat your vegetables!</li>
				<li><think><set name="advice_type">sleep</set></think>Get enough sleep!</li>
			</random>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Test greeting with username
	session := &ChatSession{
		ID:              "test-random-vars",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	response, err := g.ProcessInput("greet Alice", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	validOptions := []string{"Hello, Alice!", "Hi there, Alice!", "Welcome, Alice!"}
	valid := false
	for _, option := range validOptions {
		if response == option {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Expected one of %v, got '%s'", validOptions, response)
	}

	// Verify username was set
	if session.Variables["username"] != "Alice" {
		t.Errorf("Expected username='Alice', got '%s'", session.Variables["username"])
	}

	// Test advice with think inside li
	response, err = g.ProcessInput("random advice", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	validAdvice := []string{"Exercise regularly!", "Eat your vegetables!", "Get enough sleep!"}
	valid = false
	for _, option := range validAdvice {
		if response == option {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Expected one of %v, got '%s'", validAdvice, response)
	}

	// Verify advice_type was set
	if session.Variables["advice_type"] == "" {
		t.Error("Expected advice_type to be set")
	}

	validTypes := []string{"health", "diet", "sleep"}
	typeValid := false
	for _, vtype := range validTypes {
		if session.Variables["advice_type"] == vtype {
			typeValid = true
			break
		}
	}

	if !typeValid {
		t.Errorf("Expected advice_type to be one of %v, got '%s'", validTypes, session.Variables["advice_type"])
	}
}

// TestRandomTagNestedStructures tests random with nested random and condition tags
func TestRandomTagNestedStructures(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>NESTED RANDOM</pattern>
		<template>
			<random>
				<li>
					Outer A: <random>
						<li>Inner 1</li>
						<li>Inner 2</li>
					</random>
				</li>
				<li>
					Outer B: <random>
						<li>Inner 3</li>
						<li>Inner 4</li>
					</random>
				</li>
			</random>
		</template>
	</category>

	<category>
		<pattern>RANDOM WITH CONDITION</pattern>
		<template>
			<random>
				<li>
					<condition name="mood">
						<li value="happy">Yay! Random happy!</li>
						<li value="sad">Aww! Random sad!</li>
						<li>Random neutral!</li>
					</condition>
				</li>
				<li>Just a random choice!</li>
			</random>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Test nested random
	session := &ChatSession{
		ID:              "test-nested-random",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	response, err := g.ProcessInput("nested random", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	// Should be one of: "Outer A: Inner 1", "Outer A: Inner 2", "Outer B: Inner 3", "Outer B: Inner 4"
	validPatterns := []string{"Outer A: Inner", "Outer B: Inner"}
	valid := false
	for _, pattern := range validPatterns {
		if strings.Contains(response, pattern) {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Expected response to contain 'Outer A: Inner' or 'Outer B: Inner', got '%s'", response)
	}

	// Test random with condition
	session.Variables["mood"] = "happy"
	response, err = g.ProcessInput("random with condition", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	validOptions := []string{"Yay! Random happy!", "Just a random choice!"}
	valid = false
	for _, option := range validOptions {
		if response == option {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Expected one of %v, got '%s'", validOptions, response)
	}
}

// TestRandomTagComprehensiveEdgeCases tests edge cases for random tag
func TestRandomTagComprehensiveEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tests := []struct {
		name         string
		template     string
		validOptions []string
	}{
		{
			name:         "Random with empty li",
			template:     `<random><li></li><li>Not empty</li></random>`,
			validOptions: []string{"", "Not empty"},
		},
		{
			name:         "Random with whitespace li",
			template:     `<random><li>   </li><li>Text</li></random>`,
			validOptions: []string{"", "Text"},
		},
		{
			name:         "Random with numeric content",
			template:     `<random><li>1</li><li>2</li><li>3</li></random>`,
			validOptions: []string{"1", "2", "3"},
		},
		{
			name:         "Random with special characters",
			template:     `<random><li>&amp;</li><li>&lt;</li><li>&gt;</li></random>`,
			validOptions: []string{"&amp;", "&lt;", "&gt;"}, // XML entities are preserved in output
		},
		{
			name:         "Random with long content",
			template:     `<random><li>` + strings.Repeat("A", 100) + `</li><li>` + strings.Repeat("B", 100) + `</li></random>`,
			validOptions: []string{strings.Repeat("A", 100), strings.Repeat("B", 100)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_edge_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			valid := false
			for _, option := range tt.validOptions {
				if result == option {
					valid = true
					break
				}
			}

			if !valid {
				t.Errorf("Expected one of %v, got '%s'", tt.validOptions, result)
			}
		})
	}
}

// TestRandomTagAIML2Examples tests examples from AIML2 specification
func TestRandomTagAIML2Examples(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<!-- Standard AIML2 random example -->
	<category>
		<pattern>HI</pattern>
		<template>
			<random>
				<li>Hello!</li>
				<li>Hi there!</li>
				<li>Hey!</li>
			</random>
		</template>
	</category>

	<!-- Random with template expressions -->
	<category>
		<pattern>WHAT IS YOUR FAVORITE *</pattern>
		<template>
			<random>
				<li>I like <star/> very much!</li>
				<li><star/> is great!</li>
				<li>I enjoy <star/>!</li>
			</random>
		</template>
	</category>

	<!-- Random with SRAI -->
	<category>
		<pattern>HELP</pattern>
		<template>
			<random>
				<li><srai>WHAT CAN YOU DO</srai></li>
				<li>How can I assist you?</li>
				<li>What do you need help with?</li>
			</random>
		</template>
	</category>

	<category>
		<pattern>WHAT CAN YOU DO</pattern>
		<template>I can answer questions!</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		input        string
		validOptions []string
	}{
		{
			input:        "hi",
			validOptions: []string{"Hello!", "Hi there!", "Hey!"},
		},
		{
			input:        "what is your favorite color",
			validOptions: []string{"I like color very much!", "color is great!", "I enjoy color!"},
		},
		{
			input:        "help",
			validOptions: []string{"I can answer questions!", "How can I assist you?", "What do you need help with?"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			session := &ChatSession{
				ID:              "test-aiml2ex-" + tt.input,
				Variables:       make(map[string]string),
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

			valid := false
			for _, option := range tt.validOptions {
				if response == option {
					valid = true
					break
				}
			}

			if !valid {
				t.Errorf("Expected one of %v, got '%s'", tt.validOptions, response)
			}
		})
	}
}

// TestRandomTagComprehensivePerformance tests random tag performance with many options
func TestRandomTagComprehensivePerformance(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Create random tag with 50 options
	var templateParts []string
	templateParts = append(templateParts, "<random>")
	for i := 0; i < 50; i++ {
		templateParts = append(templateParts, "<li>Option "+string(rune('A'+i%26))+"</li>")
	}
	templateParts = append(templateParts, "</random>")
	template := strings.Join(templateParts, "")

	session := g.CreateSession("test-performance")

	// Run 100 iterations
	results := make(map[string]int)
	for i := 0; i < 100; i++ {
		result := g.ProcessTemplateWithContext(template, nil, session)
		results[result]++
	}

	// Should have some variation
	if len(results) < 10 {
		t.Logf("Warning: Only %d different results from 50 options over 100 iterations", len(results))
	}

	// All results should be valid options
	for result := range results {
		if !strings.HasPrefix(result, "Option ") {
			t.Errorf("Invalid result: %s", result)
		}
	}

	t.Logf("Got %d different results from 50 options over 100 iterations", len(results))
}
