package golem

import (
	"testing"
)

// TestPerson2TagProcessing tests the <person2> tag processing
func TestPerson2TagProcessing(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic first person to third person",
			template: "<person2>I am happy</person2>",
			expected: "they are happy",
		},
		{
			name:     "Multiple pronouns",
			template: "<person2>I gave my book to myself</person2>",
			expected: "they gave their book to themselves",
		},
		{
			name:     "Plural first person",
			template: "<person2>We are going to our house</person2>",
			expected: "They are going to their house",
		},
		{
			name:     "Possessive pronouns",
			template: "<person2>This is mine and that is ours</person2>",
			expected: "This is theirs and that is theirs",
		},
		{
			name:     "Contractions",
			template: "<person2>I'm going to give you my car</person2>",
			expected: "they're going to give you their car",
		},
		{
			name:     "Complex contractions",
			template: "<person2>We've been working on our project</person2>",
			expected: "They've been working on their project",
		},
		{
			name:     "Mixed case",
			template: "<person2>I think My idea is better than Ours</person2>",
			expected: "they think Their idea is better than Theirs",
		},
		{
			name:     "Empty person2 tag",
			template: "<person2></person2>",
			expected: "",
		},
		{
			name:     "Person2 tag with only whitespace",
			template: "<person2>   </person2>",
			expected: "",
		},
		{
			name:     "Multiple person2 tags",
			template: "<person2>I am</person2> <person2>we are</person2>",
			expected: "they are they are",
		},
		{
			name:     "No pronouns to substitute",
			template: "<person2>The cat is sleeping</person2>",
			expected: "The cat is sleeping",
		},
		{
			name:     "Mixed pronouns and non-pronouns",
			template: "<person2>I think the cat likes my food</person2>",
			expected: "they think the cat likes their food",
		},
		{
			name:     "Complex sentence",
			template: "<person2>I told myself that we should give our money to ourselves</person2>",
			expected: "they told themselves that they should give their money to themselves",
		},
		{
			name:     "Possessive contractions",
			template: "<person2>I'm going to my friend's house</person2>",
			expected: "they're going to their friend's house",
		},
		{
			name:     "Reflexive pronouns",
			template: "<person2>I hurt myself and we hurt ourselves</person2>",
			expected: "they hurt themselves and they hurt themselves",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPerson2WithWildcards tests person2 tag with wildcards
func TestPerson2WithWildcards(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name      string
		template  string
		wildcards map[string]string
		expected  string
	}{
		{
			name:      "Person2 with wildcard",
			template:  "<person2>I like <star/></person2>",
			wildcards: map[string]string{"star1": "pizza"},
			expected:  "they like pizza",
		},
		{
			name:      "Person2 with multiple wildcards",
			template:  "<person2>I gave <star/> to <star index=\"2\"/></person2>",
			wildcards: map[string]string{"star1": "my book", "star2": "myself"},
			expected:  "they gave their book to themselves",
		},
		{
			name:      "Wildcard in person2 tag",
			template:  "<person2><star/></person2>",
			wildcards: map[string]string{"star1": "I am happy"},
			expected:  "they are happy",
		},
		{
			name:      "Mixed person2 and wildcards",
			template:  "<person2>I think</person2> <star/> <person2>we should</person2>",
			wildcards: map[string]string{"star1": "that"},
			expected:  "they think that they should",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, tt.wildcards)
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPerson2WithVariables tests person2 tag with variables
func TestPerson2WithVariables(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output
	kb := NewAIMLKnowledgeBase()

	// Add a category that uses variables
	kb.Categories = []Category{
		{
			Pattern:  "TEST PERSON2",
			Template: "<person2>I think <get name=\"name\"/> is <get name=\"adjective\"/></person2>",
		},
		{
			Pattern:  "TEST PERSON2 COMPLEX",
			Template: "<person2>We believe our <get name=\"item\"/> is better than theirs</person2>",
		},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	// Create a session with variables
	session := &ChatSession{
		ID:        "test-session",
		History:   []string{},
		Variables: make(map[string]string),
	}

	// Set variables
	session.Variables["name"] = "John"
	session.Variables["adjective"] = "smart"
	session.Variables["item"] = "idea"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Person2 with variable",
			input:    "TEST PERSON2",
			expected: "they think John is smart",
		},
		{
			name:     "Person2 with complex variable",
			input:    "TEST PERSON2 COMPLEX",
			expected: "They believe their idea is better than theirs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput() error = %v", err)
			}
			if response != tt.expected {
				t.Errorf("ProcessInput() = %v, want %v", response, tt.expected)
			}
		})
	}
}

// TestPerson2Integration tests person2 integration with other tags
func TestPerson2Integration(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Person2 with person tag",
			template: "<person>I am happy</person> <person2>I am sad</person2>",
			expected: "you are happy they are sad",
		},
		{
			name:     "Person2 with gender tag",
			template: "<gender>I am a boy</gender> <person2>I am a person</person2>",
			expected: "I am a boy they are a person",
		},
		{
			name:     "Person2 with normalize tag",
			template: "<normalize>I am happy</normalize> <person2>I am sad</person2>",
			expected: "I AM HAPPY they are sad",
		},
		{
			name:     "Person2 with denormalize tag",
			template: "<denormalize>I AM HAPPY</denormalize> <person2>I am sad</person2>",
			expected: "I am happy. they are sad",
		},
		{
			name:     "Nested person2 tags",
			template: "<person2>I think <person2>we should go</person2></person2>",
			expected: "they think they should go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPerson2EdgeCases tests edge cases for person2 tag
func TestPerson2EdgeCases(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Only pronouns",
			template: "<person2>I me my mine we us our ours</person2>",
			expected: "they them their theirs they them their theirs",
		},
		{
			name:     "Pronouns with punctuation",
			template: "<person2>I, me, my, mine!</person2>",
			expected: "I, me, my, mine!",
		},
		{
			name:     "Pronouns with numbers",
			template: "<person2>I have 5 cars and we have 3 houses</person2>",
			expected: "they have 5 cars and they have 3 houses",
		},
		{
			name:     "Pronouns with special characters",
			template: "<person2>I@me.com and we@us.org</person2>",
			expected: "I@me.com and we@us.org",
		},
		{
			name:     "Very long sentence",
			template: "<person2>I think that we should give our money to ourselves because I believe that we deserve it and I know that we can do it together</person2>",
			expected: "they think that they should give their money to themselves because they believe that they deserve it and they know that they can do it together",
		},
		{
			name:     "Pronouns in quotes",
			template: "<person2>I said \"I am happy\" and we said \"we are sad\"</person2>",
			expected: "they said \"I am happy\" and they said \"we are sad\"",
		},
		{
			name:     "Mixed pronouns and non-pronouns",
			template: "<person2>The cat I own and the dog we have</person2>",
			expected: "The cat they own and the dog they have",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPerson2VerbAgreement tests verb agreement after person2 substitution
func TestPerson2VerbAgreement(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic verb agreement",
			template: "<person2>I am happy</person2>",
			expected: "they are happy",
		},
		{
			name:     "Past tense verb agreement",
			template: "<person2>I was there and we were here</person2>",
			expected: "they were there and they were here",
		},
		{
			name:     "Present perfect verb agreement",
			template: "<person2>I have been working and we have been sleeping</person2>",
			expected: "they have been working and they have been sleeping",
		},
		{
			name:     "Third person singular verb agreement",
			template: "<person2>I does not like it and we does not want it</person2>",
			expected: "they do not like it and they do not want it",
		},
		{
			name:     "Negative contractions",
			template: "<person2>I don't like it and we doesn't want it</person2>",
			expected: "they don't like it and they don't want it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ProcessTemplate(tt.template, make(map[string]string))
			if result != tt.expected {
				t.Errorf("ProcessTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPerson2Performance tests performance with multiple person2 tags
func TestPerson2Performance(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	// Create a template with many person2 tags
	template := ""
	expected := ""

	for i := 0; i < 100; i++ {
		template += "<person2>I am happy</person2> "
		expected += "they are happy "
	}

	expected = expected[:len(expected)-1] // Remove trailing space

	result := g.ProcessTemplate(template, make(map[string]string))
	if result != expected {
		t.Errorf("ProcessTemplate() performance test failed. Expected length: %d, got length: %d", len(expected), len(result))
	}
}

// TestSubstitutePronouns2Direct tests the SubstitutePronouns2 function directly
func TestSubstitutePronouns2Direct(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose mode for cleaner test output

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic substitution",
			input:    "I am happy",
			expected: "they are happy",
		},
		{
			name:     "Plural substitution",
			input:    "We are going",
			expected: "They are going",
		},
		{
			name:     "Possessive substitution",
			input:    "This is my book",
			expected: "This is their book",
		},
		{
			name:     "Contraction substitution",
			input:    "I'm going to give you my car",
			expected: "they're going to give you their car",
		},
		{
			name:     "Complex sentence",
			input:    "I think we should give our money to ourselves",
			expected: "they think they should give their money to themselves",
		},
		{
			name:     "No pronouns",
			input:    "The cat is sleeping",
			expected: "The cat is sleeping",
		},
		{
			name:     "Mixed case",
			input:    "I think My idea is better than Ours",
			expected: "they think Their idea is better than Theirs",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only whitespace",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.SubstitutePronouns2(tt.input)
			if result != tt.expected {
				t.Errorf("SubstitutePronouns2() = %v, want %v", result, tt.expected)
			}
		})
	}
}
