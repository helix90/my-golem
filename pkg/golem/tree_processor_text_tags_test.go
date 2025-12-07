package golem

import (
	"testing"
	"time"
)

// TestTreeProcessorPersonTag tests the <person> tag with tree processor
func TestTreeProcessorPersonTag(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Enable AST-based processing

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic first person to second person",
			template: "<person>I am happy</person>",
			expected: "you are happy",
		},
		{
			name:     "Basic second person to first person",
			template: "<person>you are happy</person>",
			expected: "I am happy",
		},
		{
			name:     "Multiple pronouns",
			template: "<person>I gave you my book</person>",
			expected: "you gave I your book",
		},
		{
			name:     "Possessive pronouns",
			template: "<person>my car and your bike</person>",
			expected: "your car and my bike",
		},
		{
			name:     "Contractions",
			template: "<person>I'm going to give you my car</person>",
			expected: "you're going to give I your car",
		},
		{
			name:     "Complex sentence",
			template: "<person>I think you should take my advice</person>",
			expected: "you think I should take your advice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_person_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorPerson2Tag tests the <person2> tag with tree processor
func TestTreeProcessorPerson2Tag(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Enable AST-based processing

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
			name:     "Complex sentence",
			template: "<person2>I think my idea is better</person2>",
			expected: "they think their idea is better",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_person2_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorGenderTag tests the <gender> tag with tree processor
func TestTreeProcessorGenderTag(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Enable AST-based processing

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic masculine to feminine",
			template: "<gender>he is a doctor</gender>",
			expected: "she is a doctor",
		},
		{
			name:     "Basic feminine to masculine",
			template: "<gender>she is a teacher</gender>",
			expected: "he is a teacher",
		},
		{
			name:     "Multiple pronouns",
			template: "<gender>he said she was here</gender>",
			expected: "she said he was here",
		},
		{
			name:     "Possessive pronouns",
			template: "<gender>his book and her pen</gender>",
			expected: "her book and his pen",
		},
		{
			name:     "Reflexive pronouns",
			template: "<gender>he helped himself and she helped herself</gender>",
			expected: "she helped herself and he helped himself",
		},
		{
			name:     "Contractions",
			template: "<gender>he's going and she's staying</gender>",
			expected: "she's going and he's staying",
		},
		{
			name:     "Complex sentence",
			template: "<gender>he said his friend told him she would help</gender>",
			expected: "she said her friend told her he would help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_gender_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorTextTagsIntegration tests person, person2, and gender tags in AIML categories
func TestTreeProcessorTextTagsIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TELL ME WHAT I SAID</pattern>
		<template><person>you said hello</person></template>
	</category>
	
	<category>
		<pattern>CONVERT TO THIRD PERSON</pattern>
		<template><person2>I am happy</person2></template>
	</category>
	
	<category>
		<pattern>SWAP GENDER</pattern>
		<template><gender>he said she was here</gender></template>
	</category>
	
	<category>
		<pattern>COMBINED TEST</pattern>
		<template><person><gender>he told you</gender></person></template>
	</category>
	
	<category>
		<pattern>CAPITALIZE SENTENCE</pattern>
		<template><sentence>hello world. how are you?</sentence></template>
	</category>
	
	<category>
		<pattern>CAPITALIZE WORDS</pattern>
		<template><word>hello world how are you</word></template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-text-tags",
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
		name     string
		input    string
		expected string
	}{
		{
			name:     "Person tag in category",
			input:    "tell me what i said",
			expected: "I said hello",
		},
		{
			name:     "Person2 tag in category",
			input:    "convert to third person",
			expected: "they are happy",
		},
		{
			name:     "Gender tag in category",
			input:    "swap gender",
			expected: "she said he was here",
		},
		{
			name:     "Nested person and gender tags",
			input:    "combined test",
			expected: "she told I",
		},
		{
			name:     "Sentence tag in category",
			input:    "capitalize sentence",
			expected: "Hello world. How are you?",
		},
		{
			name:     "Word tag in category",
			input:    "capitalize words",
			expected: "Hello World How Are You",
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
		})
	}
}

// TestTreeProcessorSentenceTag tests the <sentence> tag with tree processor
func TestTreeProcessorSentenceTag(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Enable AST-based processing

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Single sentence - lowercase",
			template: "<sentence>hello world</sentence>",
			expected: "Hello world",
		},
		{
			name:     "Multiple sentences",
			template: "<sentence>hello world. how are you?</sentence>",
			expected: "Hello world. How are you?",
		},
		{
			name:     "Multiple sentences with exclamation",
			template: "<sentence>wow! that's amazing! really?</sentence>",
			expected: "Wow! That's amazing! Really?",
		},
		{
			name:     "Already capitalized",
			template: "<sentence>Hello World. How Are You?</sentence>",
			expected: "Hello World. How Are You?",
		},
		{
			name:     "Mixed case",
			template: "<sentence>hELLo wORLD. hOW aRE yOU?</sentence>",
			expected: "HELLo wORLD. HOW aRE yOU?",
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
			name:     "Sentences with numbers",
			template: "<sentence>i have 5 apples. you have 3 oranges.</sentence>",
			expected: "I have 5 apples. You have 3 oranges.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_sentence_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorWordTag tests the <word> tag with tree processor
func TestTreeProcessorWordTag(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing() // Enable AST-based processing

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic lowercase words",
			template: "<word>hello world</word>",
			expected: "Hello World",
		},
		{
			name:     "Multiple words",
			template: "<word>the quick brown fox jumps</word>",
			expected: "The Quick Brown Fox Jumps",
		},
		{
			name:     "Already capitalized",
			template: "<word>Hello World</word>",
			expected: "Hello World",
		},
		{
			name:     "Mixed case",
			template: "<word>hELLo wORLD</word>",
			expected: "HELLo WORLD",
		},
		{
			name:     "Single word",
			template: "<word>test</word>",
			expected: "Test",
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
			name:     "Words with numbers",
			template: "<word>i have 5 apples</word>",
			expected: "I Have 5 Apples",
		},
		{
			name:     "Hyphenated words",
			template: "<word>state-of-the-art design</word>",
			expected: "State-Of-The-Art Design",
		},
		{
			name:     "Words with punctuation",
			template: "<word>hello, world! how are you?</word>",
			expected: "Hello, World! How Are You?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test_word_" + tt.name)
			result := g.ProcessTemplateWithContext(tt.template, nil, session)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTreeProcessorTextTagsWithWildcards tests text tags with wildcards
func TestTreeProcessorTextTagsWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>PERSON ECHO *</pattern>
		<template><person><star/></person></template>
	</category>
	
	<category>
		<pattern>PERSON2 ECHO *</pattern>
		<template><person2><star/></person2></template>
	</category>
	
	<category>
		<pattern>GENDER ECHO *</pattern>
		<template><gender><star/></gender></template>
	</category>
	
	<category>
		<pattern>SENTENCE ECHO *</pattern>
		<template><sentence><star/></sentence></template>
	</category>
	
	<category>
		<pattern>WORD ECHO *</pattern>
		<template><word><star/></word></template>
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

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Person with wildcard",
			input:    "person echo I love you",
			expected: "you love I",
		},
		{
			name:     "Person2 with wildcard",
			input:    "person2 echo I am happy",
			expected: "they are happy",
		},
		{
			name:     "Gender with wildcard",
			input:    "gender echo he said hello",
			expected: "she said hello",
		},
		{
			name:     "Sentence with wildcard",
			input:    "sentence echo hello world. how are you?",
			expected: "Hello world. How are you?",
		},
		{
			name:     "Word with wildcard",
			input:    "word echo hello world how are you",
			expected: "Hello World How Are You",
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
		})
	}
}
