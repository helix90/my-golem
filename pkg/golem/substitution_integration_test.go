package golem

import (
	"path/filepath"
	"testing"
)

// TestSubstitutionIntegration tests that loaded substitutions are actually used in text normalization
func TestSubstitutionIntegration(t *testing.T) {
	g := NewForTesting(t, true)

	// Load a test substitution file
	substitutionPath, err := filepath.Abs("../../testdata/loader_test/normal.substitution")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Load the substitution file
	substitutions, err := g.LoadSubstitutionFromFile(substitutionPath)
	if err != nil {
		t.Fatalf("Failed to load substitution file: %v", err)
	}

	// Add the substitutions to the knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	g.aimlKB.Substitutions["normal"] = substitutions

	// Debug: Print some of the loaded substitutions
	t.Logf("Loaded %d substitution rules", len(substitutions))
	for pattern, replacement := range substitutions {
		if pattern == "%20" || pattern == "%28" || pattern == "%29" || pattern == "&" {
			t.Logf("Substitution: '%s' -> '%s'", pattern, replacement)
		}
	}

	// Test cases: input -> expected after normalization with substitutions
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "Hello%20World",
			expected: "HELLO WORLD", // %20 should be replaced with space
			desc:     "URL encoding substitution",
		},
		{
			input:    "Test&Test",
			expected: "TEST TEST", // & should be replaced with space
			desc:     "Ampersand substitution",
		},
		{
			input:    "Hello%28World%29",
			expected: "HELLO LPAREN WORLD RPAREN", // %28 → ( → lparen (uppercased), %29 → ) → rparen (uppercased)
			desc:     "URL encoded parentheses substitution with chaining",
		},
		{
			input:    "Multiple%20%26%28Test%29",
			expected: "MULTIPLE LPAREN TEST RPAREN", // Multiple substitutions apply iteratively
			desc:     "Multiple substitutions with chaining",
		},
		{
			input:    "Hello(World)",
			expected: "HELLO LPAREN WORLD RPAREN", // Direct parentheses should be normalized to LPAREN/RPAREN
			desc:     "Direct parentheses normalization",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Test the normalization function directly
			result := g.CachedNormalizeForMatching(tc.input)
			t.Logf("Input: '%s' -> Result: '%s' (Expected: '%s')", tc.input, result, tc.expected)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestSubstitutionWithoutLoading tests that normalization works without substitutions
func TestSubstitutionWithoutLoading(t *testing.T) {
	g := NewForTesting(t, true)

	// Test that normalization still works without loaded substitutions
	input := "Hello%20World"
	result := g.CachedNormalizeForMatching(input)

	// Without substitutions, %20 should remain as %20
	expected := "HELLO%20WORLD"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestSubstitutionWithEmptyKnowledgeBase tests that normalization works with empty knowledge base
func TestSubstitutionWithEmptyKnowledgeBase(t *testing.T) {
	g := NewForTesting(t, true)
	g.aimlKB = NewAIMLKnowledgeBase() // Empty knowledge base

	// Test that normalization still works with empty substitutions
	input := "Hello%20World"
	result := g.CachedNormalizeForMatching(input)

	// Without substitutions, %20 should remain as %20
	expected := "HELLO%20WORLD"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestApplyLoadedSubstitutionsDirectly tests the applyLoadedSubstitutions function directly
func TestApplyLoadedSubstitutionsDirectly(t *testing.T) {
	g := NewForTesting(t, true)

	// Create test substitutions
	testSubstitutions := map[string]string{
		"%20": " ",
		"&":   " ",
		"(":   "",
		")":   "",
	}

	// Add to knowledge base
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}
	g.aimlKB.Substitutions["test"] = testSubstitutions

	// Test the function directly
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "HELLO%20WORLD",
			expected: "HELLO WORLD",
			desc:     "URL encoding substitution",
		},
		{
			input:    "TEST&TEST",
			expected: "TEST TEST",
			desc:     "Ampersand substitution",
		},
		{
			input:    "HELLO(WORLD)",
			expected: "HELLOWORLD",
			desc:     "Parentheses removal",
		},
		{
			input:    "MULTIPLE%20&TEST",
			expected: "MULTIPLE  TEST",
			desc:     "Multiple substitutions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := g.applyLoadedSubstitutions(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
