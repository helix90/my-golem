package golem

import (
	"testing"
)

// TestExpandContractions tests the contraction expansion function
func TestExpandContractions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic contractions
		{
			name:     "I'm contraction",
			input:    "I'm happy",
			expected: "I am happy",
		},
		{
			name:     "You're contraction",
			input:    "You're welcome",
			expected: "You are welcome",
		},
		{
			name:     "He's contraction",
			input:    "He's going",
			expected: "He is going",
		},
		{
			name:     "She's contraction",
			input:    "She's here",
			expected: "She is here",
		},
		{
			name:     "It's contraction",
			input:    "It's working",
			expected: "It is working",
		},
		{
			name:     "We're contraction",
			input:    "We're ready",
			expected: "We are ready",
		},
		{
			name:     "They're contraction",
			input:    "They're coming",
			expected: "They are coming",
		},

		// Negative contractions
		{
			name:     "Don't contraction",
			input:    "Don't worry",
			expected: "Do not worry",
		},
		{
			name:     "Won't contraction",
			input:    "Won't you come",
			expected: "Will not you come",
		},
		{
			name:     "Can't contraction",
			input:    "Can't do it",
			expected: "Cannot do it",
		},
		{
			name:     "Isn't contraction",
			input:    "Isn't it great",
			expected: "Is not it great",
		},
		{
			name:     "Aren't contraction",
			input:    "Aren't you coming",
			expected: "Are not you coming",
		},
		{
			name:     "Wasn't contraction",
			input:    "Wasn't that fun",
			expected: "Was not that fun",
		},
		{
			name:     "Weren't contraction",
			input:    "Weren't they here",
			expected: "Were not they here",
		},
		{
			name:     "Hasn't contraction",
			input:    "Hasn't he called",
			expected: "Has not he called",
		},
		{
			name:     "Haven't contraction",
			input:    "Haven't you seen",
			expected: "Have not you seen",
		},
		{
			name:     "Hadn't contraction",
			input:    "Hadn't we met",
			expected: "Had not we met",
		},
		{
			name:     "Wouldn't contraction",
			input:    "Wouldn't it be nice",
			expected: "Would not it be nice",
		},
		{
			name:     "Shouldn't contraction",
			input:    "Shouldn't we go",
			expected: "Should not we go",
		},
		{
			name:     "Couldn't contraction",
			input:    "Couldn't you help",
			expected: "Could not you help",
		},
		{
			name:     "Mustn't contraction",
			input:    "Mustn't forget",
			expected: "Must not forget",
		},
		{
			name:     "Shan't contraction",
			input:    "Shan't we dance",
			expected: "Shall not we dance",
		},

		// Future tense contractions
		{
			name:     "I'll contraction",
			input:    "I'll be there",
			expected: "I will be there",
		},
		{
			name:     "You'll contraction",
			input:    "You'll see",
			expected: "You will see",
		},
		{
			name:     "He'll contraction",
			input:    "He'll come",
			expected: "He will come",
		},
		{
			name:     "She'll contraction",
			input:    "She'll help",
			expected: "She will help",
		},
		{
			name:     "It'll contraction",
			input:    "It'll work",
			expected: "It will work",
		},
		{
			name:     "We'll contraction",
			input:    "We'll try",
			expected: "We will try",
		},
		{
			name:     "They'll contraction",
			input:    "They'll understand",
			expected: "They will understand",
		},

		// Perfect tense contractions
		{
			name:     "I've contraction",
			input:    "I've seen it",
			expected: "I have seen it",
		},
		{
			name:     "You've contraction",
			input:    "You've done well",
			expected: "You have done well",
		},
		{
			name:     "We've contraction",
			input:    "We've been here",
			expected: "We have been here",
		},
		{
			name:     "They've contraction",
			input:    "They've arrived",
			expected: "They have arrived",
		},

		// Past tense contractions
		{
			name:     "I'd contraction",
			input:    "I'd like that",
			expected: "I would like that",
		},
		{
			name:     "You'd contraction",
			input:    "You'd better go",
			expected: "You had better go",
		},
		{
			name:     "He'd contraction",
			input:    "He'd prefer it",
			expected: "He had prefer it",
		},
		{
			name:     "She'd contraction",
			input:    "She'd enjoy it",
			expected: "She had enjoy it",
		},
		{
			name:     "It'd contraction",
			input:    "It'd be nice",
			expected: "It had be nice",
		},
		{
			name:     "We'd contraction",
			input:    "We'd love to",
			expected: "We had love to",
		},
		{
			name:     "They'd contraction",
			input:    "They'd appreciate it",
			expected: "They had appreciate it",
		},

		// Other common contractions
		{
			name:     "Let's contraction",
			input:    "Let's go",
			expected: "Let us go",
		},
		{
			name:     "That's contraction",
			input:    "That's great",
			expected: "That is great",
		},
		{
			name:     "There's contraction",
			input:    "There's a problem",
			expected: "There is a problem",
		},
		{
			name:     "Here's contraction",
			input:    "Here's the answer",
			expected: "Here is the answer",
		},
		{
			name:     "What's contraction",
			input:    "What's happening",
			expected: "What is happening",
		},
		{
			name:     "Who's contraction",
			input:    "Who's there",
			expected: "Who is there",
		},
		{
			name:     "Where's contraction",
			input:    "Where's the book",
			expected: "Where is the book",
		},
		{
			name:     "When's contraction",
			input:    "When's the meeting",
			expected: "When is the meeting",
		},
		{
			name:     "Why's contraction",
			input:    "Why's he here",
			expected: "Why is he here",
		},
		{
			name:     "How's contraction",
			input:    "How's it going",
			expected: "How is it going",
		},

		// Possessive contractions
		{
			name:     "Y'all contraction",
			input:    "Y'all come back",
			expected: "You all come back",
		},
		{
			name:     "Ma'am contraction",
			input:    "Yes ma'am",
			expected: "Yes madam",
		},
		{
			name:     "O'clock contraction",
			input:    "It's 3 o'clock",
			expected: "It is 3 of the clock",
		},

		// Case variations
		{
			name:     "Mixed case contractions",
			input:    "I'M happy, you're welcome, DON'T worry",
			expected: "I AM happy, you are welcome, DO NOT worry",
		},
		{
			name:     "Lowercase contractions",
			input:    "i'm happy, you're welcome, don't worry",
			expected: "i am happy, you are welcome, do not worry",
		},

		// Multiple contractions in one sentence
		{
			name:     "Multiple contractions",
			input:    "I'm sure you'll understand why we can't do it",
			expected: "I am sure you will understand why we cannot do it",
		},
		{
			name:     "Complex sentence with contractions",
			input:    "I've been thinking that we shouldn't have done that, but it's too late now",
			expected: "I have been thinking that we should not have done that, but it is too late now",
		},

		// Edge cases
		{
			name:     "No contractions",
			input:    "This is a normal sentence",
			expected: "This is a normal sentence",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only contractions",
			input:    "I'm you're he's",
			expected: "I am you are he is",
		},
		{
			name:     "Contractions with punctuation",
			input:    "I'm happy! You're welcome? Don't worry.",
			expected: "I am happy! You are welcome? Do not worry.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandContractions(tt.input)
			if result != tt.expected {
				t.Errorf("expandContractions(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestContractionExpansionInNormalization tests contraction expansion in normalization functions
func TestContractionExpansionInNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "NormalizeForMatchingCasePreserving with contractions",
			input:    "I'm happy you're here",
			expected: "I am happy you are here",
		},
		{
			name:     "normalizeForMatching with contractions",
			input:    "Don't worry, it'll work",
			expected: "DO NOT WORRY IT WILL WORK",
		},
		{
			name:     "NormalizePattern with contractions",
			input:    "I've been thinking",
			expected: "I HAVE BEEN THINKING",
		},
		{
			name:     "NormalizeThatPattern with contractions",
			input:    "I'm sure you'll understand",
			expected: "I AM SURE YOU WILL UNDERSTAND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string

			switch tt.name {
			case "NormalizeForMatchingCasePreserving with contractions":
				result = NormalizeForMatchingCasePreserving(tt.input)
			case "normalizeForMatching with contractions":
				result = normalizeForMatching(tt.input)
			case "NormalizePattern with contractions":
				result = NormalizePattern(tt.input)
			case "NormalizeThatPattern with contractions":
				result = NormalizeThatPattern(tt.input)
			}

			if result != tt.expected {
				t.Errorf("%s(%q) = %q, expected %q", tt.name, tt.input, result, tt.expected)
			}
		})
	}
}

// TestContractionExpansionWithPatternMatching tests that contraction expansion works with pattern matching
func TestContractionExpansionWithPatternMatching(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories with contractions in patterns
	kb.Categories = []Category{
		{Pattern: "I AM HAPPY", Template: "That's great!"},
		{Pattern: "YOU ARE WELCOME", Template: "You're very kind!"},
		{Pattern: "DO NOT WORRY", Template: "I won't worry!"},
		{Pattern: "I WILL BE THERE", Template: "I'll see you then!"},
		{Pattern: "I HAVE SEEN IT", Template: "I've noticed that too!"},
		{Pattern: "LET US GO", Template: "Let's do it!"},
		{Pattern: "WHAT IS HAPPENING", Template: "What's going on?"},
		{Pattern: "HOW IS IT GOING", Template: "How's everything?"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "I'm happy should match I AM HAPPY",
			input:    "I'm happy",
			expected: "That's great!",
		},
		{
			name:     "You're welcome should match YOU ARE WELCOME",
			input:    "You're welcome",
			expected: "You're very kind!",
		},
		{
			name:     "Don't worry should match DO NOT WORRY",
			input:    "Don't worry",
			expected: "I won't worry!",
		},
		{
			name:     "I'll be there should match I WILL BE THERE",
			input:    "I'll be there",
			expected: "I'll see you then!",
		},
		{
			name:     "I've seen it should match I HAVE SEEN IT",
			input:    "I've seen it",
			expected: "I've noticed that too!",
		},
		{
			name:     "Let's go should match LET US GO",
			input:    "Let's go",
			expected: "Let's do it!",
		},
		{
			name:     "What's happening should match WHAT IS HAPPENING",
			input:    "What's happening",
			expected: "What's going on?",
		},
		{
			name:     "How's it going should match HOW IS IT GOING",
			input:    "How's it going",
			expected: "How's everything?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_session")
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Errorf("Error processing input: %v", err)
				return
			}

			if response != tt.expected {
				t.Errorf("ProcessInput(%q) = %q, expected %q", tt.input, response, tt.expected)
			}
		})
	}
}

// TestContractionExpansionPerformance tests the performance of contraction expansion
func TestContractionExpansionPerformance(t *testing.T) {
	// Test with a long string containing many contractions
	input := "I'm happy you're here and we're going to have a great time. Don't worry about anything because we've got everything under control. I'll make sure everything goes smoothly and you'll see that it's going to be fantastic. We've been planning this for weeks and we're confident it'll work out perfectly. Let's make sure we don't forget anything important. What's the plan for tomorrow? How's everyone feeling about this? That's great to hear! There's nothing to worry about. Here's what we need to do. Who's in charge of what? When's the deadline? Why's this so important? I'd like to know more about this. You'd better be ready. He'd prefer if we started early. She'd enjoy this more if we had music. It'd be nice to have some refreshments. We'd love to help with that. They'd appreciate it if we finished on time. Y'all are doing great! Yes ma'am, we understand. It's 3 o'clock and we're ready to go!"

	// Run the function multiple times to test performance
	for i := 0; i < 1000; i++ {
		result := expandContractions(input)
		if result == "" {
			t.Error("Contraction expansion returned empty string")
		}
	}
}

// TestContractionExpansionEdgeCases tests edge cases for contraction expansion
func TestContractionExpansionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Apostrophe not in contraction",
			input:    "rock 'n' roll",
			expected: "rock 'n' roll",
		},
		{
			name:     "Multiple apostrophes",
			input:    "I'm you're he's she's it's we're they're",
			expected: "I am you are he is she is it is we are they are",
		},
		{
			name:     "Contraction at start of sentence",
			input:    "I'm starting",
			expected: "I am starting",
		},
		{
			name:     "Contraction at end of sentence",
			input:    "I can't",
			expected: "I cannot",
		},
		{
			name:     "Contraction in middle of sentence",
			input:    "I think I'll go",
			expected: "I think I will go",
		},
		{
			name:     "Whitespace around contractions",
			input:    "  I'm  happy  ",
			expected: "  I am  happy  ",
		},
		{
			name:     "Contractions with numbers",
			input:    "It's 3 o'clock",
			expected: "It is 3 of the clock",
		},
		{
			name:     "Nested contractions (should not happen but test robustness)",
			input:    "I'm you're he's",
			expected: "I am you are he is",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandContractions(tt.input)
			if result != tt.expected {
				t.Errorf("expandContractions(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
