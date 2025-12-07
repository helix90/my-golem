package golem

import (
	"strings"
	"testing"
)

// TestPersonTagAdvancedProcessing tests advanced <person> tag functionality
func TestPersonTagAdvancedProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Complex pronoun substitution",
			template: "<person>I am going to my house with my friends</person>",
			expected: "you are going to your house with your friends",
		},
		{
			name:     "Mixed pronouns and possessives",
			template: "<person>I think my car is better than yours</person>",
			expected: "you think your car is better than mine",
		},
		{
			name:     "Reflexive pronouns",
			template: "<person>I hurt myself while I was working</person>",
			expected: "you hurt yourself while you were working",
		},
		{
			name:     "Contractions",
			template: "<person>I'm going to my house and I'll see you there</person>",
			expected: "you're going to your house and you'll see I there",
		},
		{
			name:     "Possessive contractions",
			template: "<person>I've lost my keys and I can't find them</person>",
			expected: "you've lost your keys and you can't find them",
		},
		{
			name:     "Complex sentence with multiple pronouns",
			template: "<person>I told him that I would help him with his project</person>",
			expected: "you told him that you would help him with his project",
		},
		{
			name:     "Nested person tags",
			template: "<person>I said <person>I love you</person> to my friend</person>",
			expected: "you said I love you to your friend", // Inner swaps I->you, you->I; outer swaps back
		},
		{
			name:     "Empty content",
			template: "<person></person>",
			expected: "",
		},
		{
			name:     "No pronouns",
			template: "<person>The cat sat on the mat</person>",
			expected: "The cat sat on the mat",
		},
		{
			name:     "Mixed case pronouns",
			template: "<person>I AM GOING TO MY HOUSE</person>",
			expected: "you AM GOING TO MY HOUSE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestGenderTagAdvancedProcessing tests advanced <gender> tag functionality
func TestGenderTagAdvancedProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Complex gender substitution",
			template: "<gender>He is going to his house with his friends</gender>",
			expected: "She is going to her house with her friends",
		},
		{
			name:     "Mixed gender pronouns",
			template: "<gender>He thinks his car is better than hers</gender>",
			expected: "She thinks her car is better than his",
		},
		{
			name:     "Reflexive gender pronouns",
			template: "<gender>He hurt himself while he was working</gender>",
			expected: "She hurt herself while she was working",
		},
		{
			name:     "Possessive forms",
			template: "<gender>His book is on his desk</gender>",
			expected: "Her book is on her desk",
		},
		{
			name:     "Complex sentence with multiple gender pronouns",
			template: "<gender>He told her that he would help her with her project</gender>",
			expected: "She told his that she would help his with his project",
		},
		{
			name:     "Nested gender tags",
			template: "<gender>He said <gender>he loves her</gender> to his friend</gender>",
			expected: "She said he loves her to her friend", // Inner swaps he->she, her->him; outer swaps back
		},
		{
			name:     "Empty content",
			template: "<gender></gender>",
			expected: "",
		},
		{
			name:     "No gender pronouns",
			template: "<gender>The cat sat on the mat</gender>",
			expected: "The cat sat on the mat",
		},
		{
			name:     "Mixed case gender pronouns",
			template: "<gender>HE IS GOING TO HIS HOUSE</gender>",
			expected: "SHE IS GOING TO HER HOUSE",
		},
		{
			name:     "They/them pronouns",
			template: "<gender>They are going to their house</gender>",
			expected: "They are going to their house",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestPerson2TagAdvancedProcessing tests advanced <person2> tag functionality
func TestPerson2TagAdvancedProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "First to third person conversion",
			template: "<person2>I am going to my house</person2>",
			expected: "they are going to their house",
		},
		{
			name:     "Complex first to third person",
			template: "<person2>I think my car is better than yours</person2>",
			expected: "they think their car is better than yours",
		},
		{
			name:     "Reflexive pronouns first to third",
			template: "<person2>I hurt myself while I was working</person2>",
			expected: "they hurt themselves while they were working",
		},
		{
			name:     "Contractions first to third",
			template: "<person2>I'm going to my house and I'll see you there</person2>",
			expected: "they're going to their house and they'll see you there",
		},
		{
			name:     "Possessive contractions first to third",
			template: "<person2>I've lost my keys and I can't find them</person2>",
			expected: "they've lost their keys and they can't find them",
		},
		{
			name:     "Complex sentence first to third",
			template: "<person2>I told him that I would help him with his project</person2>",
			expected: "they told him that they would help him with his project",
		},
		{
			name:     "Nested person2 tags",
			template: "<person2>I said <person2>I love you</person2> to my friend</person2>",
			expected: "they said they love you to their friend",
		},
		{
			name:     "Empty content",
			template: "<person2></person2>",
			expected: "",
		},
		{
			name:     "No first person pronouns",
			template: "<person2>The cat sat on the mat</person2>",
			expected: "The cat sat on the mat",
		},
		{
			name:     "Mixed case first person pronouns",
			template: "<person2>I AM GOING TO MY HOUSE</person2>",
			expected: "they AM GOING TO MY HOUSE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestSentenceTagAdvancedProcessing tests advanced <sentence> tag functionality
func TestSentenceTagAdvancedProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Multiple sentences",
			template: "<sentence>hello world. how are you? i am fine!</sentence>",
			expected: "Hello world. How are you? I am fine!",
		},
		{
			name:     "Single sentence",
			template: "<sentence>hello world</sentence>",
			expected: "Hello world",
		},
		{
			name:     "Already capitalized sentences",
			template: "<sentence>Hello World. How Are You?</sentence>",
			expected: "Hello World. How Are You?",
		},
		{
			name:     "Mixed case sentences",
			template: "<sentence>hELLo WoRLd. hOW aRE yOU?</sentence>",
			expected: "HELLo WoRLd. HOW aRE yOU?",
		},
		{
			name:     "Sentences with numbers",
			template: "<sentence>i have 5 apples. they cost $2.50 each.</sentence>",
			expected: "I have 5 apples. They cost $2.50 each.",
		},
		{
			name:     "Sentences with special characters",
			template: "<sentence>visit our website at www.example.com. call us at (555) 123-4567.</sentence>",
			expected: "Visit our website at www.example.com. Call us at (555) 123-4567.",
		},
		{
			name:     "Empty content",
			template: "<sentence></sentence>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<sentence>   </sentence>",
			expected: "",
		},
		{
			name:     "Nested sentence tags",
			template: "<sentence>hello <sentence>world</sentence>.</sentence>",
			expected: "Hello World.",
		},
		{
			name:     "Unicode characters",
			template: "<sentence>café naïve résumé. bonjour le monde!</sentence>",
			expected: "Café naïve résumé. Bonjour le monde!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestWordTagAdvancedProcessing tests advanced <word> tag functionality
func TestWordTagAdvancedProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Multiple words",
			template: "<word>hello world test</word>",
			expected: "Hello World Test",
		},
		{
			name:     "Single word",
			template: "<word>hello</word>",
			expected: "Hello",
		},
		{
			name:     "Already capitalized words",
			template: "<word>Hello World Test</word>",
			expected: "Hello World Test",
		},
		{
			name:     "Mixed case words",
			template: "<word>hELLo WoRLd tEST</word>",
			expected: "HELLo WoRLd TEST",
		},
		{
			name:     "Words with numbers",
			template: "<word>test123 word456 test789</word>",
			expected: "Test123 Word456 Test789",
		},
		{
			name:     "Words with special characters",
			template: "<word>test-word test_word test.word</word>",
			expected: "Test-Word Test_word Test.word",
		},
		{
			name:     "Empty content",
			template: "<word></word>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<word>   </word>",
			expected: "",
		},
		{
			name:     "Nested word tags",
			template: "<word>hello <word>world</word> test</word>",
			expected: "Hello World Test",
		},
		{
			name:     "Unicode characters",
			template: "<word>café naïve résumé</word>",
			expected: "Café Naïve Résumé",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestNormalizeTagAdvancedProcessing tests advanced <normalize> tag functionality
func TestNormalizeTagAdvancedProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Enable AST-based processing
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic normalization",
			template: "<normalize>Hello, World! How are you?</normalize>",
			expected: "HELLO WORLD HOW ARE YOU",
		},
		{
			name:     "Contractions normalization",
			template: "<normalize>I'm going to my house and I'll see you there</normalize>",
			expected: "I AM GOING TO MY HOUSE AND I WILL SEE YOU THERE",
		},
		{
			name:     "Punctuation normalization",
			template: "<normalize>Hello!!! How are you??? I'm fine...</normalize>",
			expected: "HELLO HOW ARE YOU I AM FINE",
		},
		{
			name:     "Numbers and special characters",
			template: "<normalize>I have 5 apples @ $2.50 each (total: $12.50)</normalize>",
			expected: "I HAVE 5 APPLES 2 50 EACH TOTAL 12 50",
		},
		{
			name:     "Multiple spaces normalization",
			template: "<normalize>Hello    world   with    multiple    spaces</normalize>",
			expected: "HELLO WORLD WITH MULTIPLE SPACES",
		},
		{
			name:     "Empty content",
			template: "<normalize></normalize>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<normalize>   </normalize>",
			expected: "",
		},
		{
			name:     "Nested normalize tags",
			template: "<normalize>Hello <normalize>world</normalize> test</normalize>",
			expected: "HELLO WORLD TEST",
		},
		{
			name:     "Unicode characters",
			template: "<normalize>Café naïve résumé</normalize>",
			expected: "CAFÉ NAÏVE RÉSUMÉ",
		},
		{
			name:     "Mixed case normalization",
			template: "<normalize>HeLLo WoRLd!!!</normalize>",
			expected: "HELLO WORLD",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestDenormalizeTagAdvancedProcessing tests advanced <denormalize> tag functionality
func TestDenormalizeTagAdvancedProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic denormalization",
			template: "<denormalize>hello world how are you</denormalize>",
			expected: "Hello world how are you.",
		},
		{
			name:     "Contractions denormalization",
			template: "<denormalize>i am going to my house and i will see you there</denormalize>",
			expected: "I am going to my house and i will see you there.",
		},
		{
			name:     "Question denormalization",
			template: "<denormalize>how are you today</denormalize>",
			expected: "How are you today.",
		},
		{
			name:     "Exclamation denormalization",
			template: "<denormalize>wow that is amazing</denormalize>",
			expected: "Wow that is amazing.",
		},
		{
			name:     "Multiple sentences denormalization",
			template: "<denormalize>hello world how are you i am fine</denormalize>",
			expected: "Hello world how are you i am fine.",
		},
		{
			name:     "Empty content",
			template: "<denormalize></denormalize>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<denormalize>   </denormalize>",
			expected: "",
		},
		{
			name:     "Nested denormalize tags",
			template: "<denormalize>hello <denormalize>world</denormalize> test</denormalize>",
			expected: "Hello world. Test.", // Inner denormalizes "world", outer denormalizes full text
		},
		{
			name:     "Unicode characters",
			template: "<denormalize>cafe naive resume</denormalize>",
			expected: "Cafe naive resume.",
		},
		{
			name:     "Mixed case denormalization",
			template: "<denormalize>HeLLo WoRLd</denormalize>",
			expected: "Hello world.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestTextProcessingAdvancedIntegration tests integration of multiple text processing tags
func TestTextProcessingAdvancedIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Person and gender combination",
			template: "<person><gender>He is going to his house</gender></person>",
			expected: "She is going to her house",
		},
		{
			name:     "Sentence and word combination",
			template: "<sentence><word>hello world</word> test</sentence>",
			expected: "Hello World test",
		},
		{
			name:     "Normalize and denormalize combination",
			template: "<denormalize><normalize>Hello, World! How are you?</normalize></denormalize>",
			expected: "Hello world how are you.",
		},
		{
			name:     "Person2 and sentence combination",
			template: "<sentence><person2>I am going to my house</person2></sentence>",
			expected: "They are going to their house",
		},
		{
			name:     "Complex text processing chain",
			template: "<person><sentence><word>i am going to my house</word></sentence></person>",
			expected: "you Are Going To Your House",
		},
		{
			name:     "Multiple independent text processing tags",
			template: "<person>I am going</person> <gender>he is coming</gender> <sentence>hello world</sentence>",
			expected: "you are going she is coming Hello world",
		},
		{
			name:     "Text processing with wildcards",
			template: "<person><star/></person>",
			expected: "you are going to your house",
		},
	}

	// Set up wildcards for the last test case
	wildcards := map[string]string{
		"star1": "I am going to my house",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result string
			if tc.name == "Text processing with wildcards" {
				result = g.ProcessTemplateWithContext(tc.template, wildcards, session)
			} else {
				result = g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			}
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestTextProcessingAdvancedEdgeCases tests edge cases for text processing tags
func TestTextProcessingAdvancedEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Malformed tag syntax",
			template: "<person>hello world</person",
			expected: "hello world", // Tree processor handles unclosed tags
		},
		{
			name:     "Nested malformed tags",
			template: "<person><gender>hello world</person></gender>",
			expected: "<person><gender>hello world</person></gender>", // Mismatched closing tags preserved
		},
		{
			name:     "Empty tag",
			template: "<person></person>",
			expected: "",
		},
		{
			name:     "Whitespace only tag",
			template: "<person>   </person>",
			expected: "",
		},
		{
			name:     "Very long text",
			template: "<person>" + strings.Repeat("I am going to my house. ", 100) + "</person>",
			expected: strings.TrimSpace(strings.Repeat("you are going to your house. ", 100)),
		},
		{
			name:     "Unicode characters",
			template: "<person>Je vais à ma maison avec mes amis</person>",
			expected: "Je vais à ma maison avec mes amis",
		},
		{
			name:     "Special characters",
			template: "<person>I have $100 @ 5% interest!</person>",
			expected: "you have $100 @ 5% interest!",
		},
		{
			name:     "Mixed languages",
			template: "<person>I am going to mi casa</person>",
			expected: "you are going to mi casa",
		},
		{
			name:     "Numbers and symbols",
			template: "<person>I have 5 apples @ $2.50 each (total: $12.50)</person>",
			expected: "you have 5 apples @ $2.50 each (total: $12.50)",
		},
		{
			name:     "Nested empty tags",
			template: "<person><gender></gender></person>",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestTextProcessingPerformance tests performance of text processing tags
func TestTextProcessingPerformance(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	// Test with large text
	largeText := strings.Repeat("I am going to my house with my friends. ", 1000)
	template := "<person>" + largeText + "</person>"

	// This should complete without hanging
	result := g.ProcessTemplateWithContext(template, make(map[string]string), session)

	// Verify the result is not empty and contains expected transformations
	if result == "" {
		t.Error("Expected non-empty result for large text processing")
	}

	// Check that some transformations occurred
	if !strings.Contains(result, "you are going") {
		t.Error("Expected pronoun transformation in large text processing")
	}
}

// TestTextProcessingWithVariables tests text processing with variable context
func TestTextProcessingWithVariables(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	// Set some variables
	session.Variables["name"] = "John"
	session.Variables["pronoun"] = "he"

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Person tag with variables",
			template: "<person>I am <get name=\"name\"/> and I like my job</person>",
			expected: "you are John and you like your job",
		},
		{
			name:     "Gender tag with variables",
			template: "<gender><get name=\"pronoun\"/> is going to his house</gender>",
			expected: "she is going to her house",
		},
		{
			name:     "Sentence tag with variables",
			template: "<sentence>hello <get name=\"name\"/> how are you</sentence>",
			expected: "Hello John how are you",
		},
		{
			name:     "Word tag with variables",
			template: "<word>hello <get name=\"name\"/> world</word>",
			expected: "Hello John World",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestTextProcessingWithWildcards tests text processing with wildcard context
func TestTextProcessingWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test_session")

	wildcards := map[string]string{
		"star1": "I am going to my house",
		"star2": "He is coming with his friends",
		"star3": "hello world test",
	}

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Person tag with wildcards",
			template: "<person><star/></person>",
			expected: "you are going to your house",
		},
		{
			name:     "Gender tag with wildcards",
			template: "<gender><star/></gender>",
			expected: "I am going to my house",
		},
		{
			name:     "Sentence tag with wildcards",
			template: "<sentence><star/></sentence>",
			expected: "I am going to my house",
		},
		{
			name:     "Word tag with wildcards",
			template: "<word><star/></word>",
			expected: "I Am Going To My House",
		},
		{
			name:     "Multiple wildcards",
			template: "<person><star/> and <gender><star index=\"2\"/></gender></person>",
			expected: "you are going to your house and She is coming with her friends",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, wildcards, session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
