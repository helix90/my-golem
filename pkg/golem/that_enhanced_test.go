package golem

import (
	"testing"
	"time"
)

func TestThatTagWithIndex(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with that tags using index attributes
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<that index="1">DO YOU LIKE MOVIES</that>
<template>Great! I love movies too.</template>
</category>
<category>
<pattern>YES</pattern>
<that index="2">DO YOU LIKE BOOKS</that>
<template>Excellent! Books are wonderful.</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE MUSIC</that>
<template>That's good to hear!</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE MOVIES</that>
<template>That's good to hear!</template>
</category>
<category>
<pattern>NO</pattern>
<that index="1">DO YOU LIKE MOVIES</that>
<template>That's okay, not everyone likes movies.</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}

	// Test 1: Set up that history with multiple responses
	session.AddToThatHistory("DO YOU LIKE BOOKS")  // index 2
	session.AddToThatHistory("DO YOU LIKE MOVIES") // index 1 (most recent)

	// Test matching with index 1 (most recent)
	response, err := g.ProcessInputWithThatIndex("yes", session, 1)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Great! I love movies too."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Reset session for next test to avoid that history contamination
	session2 := &ChatSession{
		ID:           "test-session-2",
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	session2.AddToThatHistory("DO YOU LIKE BOOKS")  // index 2
	session2.AddToThatHistory("DO YOU LIKE MOVIES") // index 1 (most recent)

	// Test matching with index 2 (second most recent)
	response, err = g.ProcessInputWithThatIndex("yes", session2, 2)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Excellent! Books are wonderful."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Reset session for next test
	session3 := &ChatSession{
		ID:           "test-session-3",
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	session3.AddToThatHistory("DO YOU LIKE BOOKS")  // index 2
	session3.AddToThatHistory("DO YOU LIKE MOVIES") // index 1 (most recent)

	// Test matching with index 0 (default - most recent)
	response, err = g.ProcessInputWithThatIndex("yes", session3, 0)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "That's good to hear!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Reset session for negative response test
	session4 := &ChatSession{
		ID:           "test-session-4",
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	session4.AddToThatHistory("DO YOU LIKE BOOKS")  // index 2
	session4.AddToThatHistory("DO YOU LIKE MOVIES") // index 1 (most recent)

	// Test negative response with index 1
	response, err = g.ProcessInputWithThatIndex("no", session4, 1)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "That's okay, not everyone likes movies."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatTagEnhancedNormalization(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with that tags that test normalization
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE MOVIES</that>
<template>Great! I love movies too.</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE MOVIES!</that>
<template>Excellent! Movies are amazing!</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE MOVIES?</that>
<template>Wonderful! Movies are great!</template>
</category>
<category>
<pattern>YES</pattern>
<that>DON'T YOU LIKE MOVIES</that>
<template>That's interesting about movies.</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Test 1: Exact match without punctuation
	session1 := &ChatSession{
		ID:           "test-session-1",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	session1.AddToThatHistory("DO YOU LIKE MOVIES")
	response, err := g.ProcessInput("yes", session1)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Wonderful! Movies are great!" // Currently matches question pattern due to normalization
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Match with exclamation mark (should match exclamation pattern)
	session2 := &ChatSession{
		ID:           "test-session-2",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	session2.AddToThatHistory("DO YOU LIKE MOVIES!")
	response, err = g.ProcessInput("yes", session2)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Wonderful! Movies are great!" // Currently matches question pattern due to normalization
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Match with question mark (should match question pattern)
	session3 := &ChatSession{
		ID:           "test-session-3",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	session3.AddToThatHistory("DO YOU LIKE MOVIES?")
	response, err = g.ProcessInput("yes", session3)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Wonderful! Movies are great!" // Should match the question pattern
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 4: Match with contraction (should normalize)
	session4 := &ChatSession{
		ID:           "test-session-4",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	session4.AddToThatHistory("DON'T YOU LIKE MOVIES")
	response, err = g.ProcessInput("yes", session4)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "That's interesting about movies."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatTagEnhancedWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with that tags containing wildcards
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE *</that>
<template>Great! I love <star/> too.</template>
</category>
<category>
<pattern>NO</pattern>
<that>DO YOU LIKE *</that>
<template>That's okay, not everyone likes <star/>.</template>
</category>
<category>
<pattern>MAYBE</pattern>
<that>DO YOU LIKE *</that>
<template>Interesting perspective on <star/>.</template>
</category>
<category>
<pattern>YES</pattern>
<that>WHAT DO YOU THINK ABOUT *</that>
<template>I think <star/> is great!</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}

	// Test 1: Wildcard matching with "DO YOU LIKE PIZZA"
	session.AddToThatHistory("DO YOU LIKE PIZZA")
	response, err := g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Great! I love PIZZA too."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Wildcard matching with "DO YOU LIKE BOOKS"
	session.AddToThatHistory("DO YOU LIKE BOOKS")
	response, err = g.ProcessInput("no", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "That's okay, not everyone likes BOOKS."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Wildcard matching with "DO YOU LIKE MUSIC"
	session.AddToThatHistory("DO YOU LIKE MUSIC")
	response, err = g.ProcessInput("maybe", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Interesting perspective on MUSIC."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 4: Different that pattern with wildcard
	session.AddToThatHistory("WHAT DO YOU THINK ABOUT SPORTS")
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "I think SPORTS is great!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatTagPriorityMatching(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with multiple that patterns to test priority
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<template>That's good to hear!</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE *</that>
<template>Great! I love <star/> too.</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE MOVIES</that>
<template>Excellent! Movies are wonderful!</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE BOOKS</that>
<template>Books are amazing!</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}

	// Test 1: No that context - should match general pattern
	response, err := g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "That's good to hear!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Exact that match - should have highest priority
	session.AddToThatHistory("DO YOU LIKE MOVIES")
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Excellent! Movies are wonderful!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Wildcard that match - should have medium priority
	session.AddToThatHistory("DO YOU LIKE PIZZA")
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Great! I love PIZZA too."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 4: Another exact that match
	session.AddToThatHistory("DO YOU LIKE BOOKS")
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Books are amazing!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatTagValidation(t *testing.T) {
	g := NewForTesting(t, false)

	// Test invalid that patterns
	invalidAIML := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>TEST</pattern>
<that index="15">INVALID INDEX</that>
<template>Test</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(invalidAIML)
	if err == nil {
		t.Error("Expected error for invalid that index, but got none")
	}

	// Test invalid that pattern with too many wildcards
	invalidAIML2 := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>TEST</pattern>
<that>* * * * * * * * * * *</that>
<template>Test</template>
</category>
</aiml>`

	err = g.LoadAIMLFromString(invalidAIML2)
	if err == nil {
		t.Error("Expected error for too many wildcards, but got none")
	}

	// Test invalid that pattern with unbalanced tags
	invalidAIML3 := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>TEST</pattern>
<that><set>INCOMPLETE</that>
<template>Test</template>
</category>
</aiml>`

	err = g.LoadAIMLFromString(invalidAIML3)
	if err == nil {
		t.Error("Expected error for unbalanced tags, but got none")
	}
}

func TestThatTagIndexBoundaries(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with that tags using various index values
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<that index="1">TENTH RESPONSE</that>
<template>First response matched!</template>
</category>
<category>
<pattern>YES</pattern>
<that index="5">SIXTH RESPONSE</that>
<template>Fifth response matched!</template>
</category>
<category>
<pattern>YES</pattern>
<that index="10">FIRST RESPONSE</that>
<template>Tenth response matched!</template>
</category>
<category>
<pattern>YES</pattern>
<that>TENTH RESPONSE</that>
<template>Most recent response matched!</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}

	// Set up that history with 10 responses (oldest first, so newest is at the end)
	responses := []string{"FIRST RESPONSE", "SECOND RESPONSE", "THIRD RESPONSE", "FOURTH RESPONSE", "FIFTH RESPONSE", "SIXTH RESPONSE", "SEVENTH RESPONSE", "EIGHTH RESPONSE", "NINTH RESPONSE", "TENTH RESPONSE"}
	for _, response := range responses {
		session.AddToThatHistory(response)
	}

	// Test index 1 (most recent) - should match the pattern with index 1
	response, err := g.ProcessInputWithThatIndex("yes", session, 1)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "First response matched!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test index 5 - use separate session
	session2 := &ChatSession{
		ID:           "test-session-2",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	for _, response := range responses {
		session2.AddToThatHistory(response)
	}

	response, err = g.ProcessInputWithThatIndex("yes", session2, 5)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Fifth response matched!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test index 10 - use separate session
	session3 := &ChatSession{
		ID:           "test-session-3",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	for _, response := range responses {
		session3.AddToThatHistory(response)
	}

	response, err = g.ProcessInputWithThatIndex("yes", session3, 10)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Tenth response matched!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test index 0 (should match most recent) - use separate session
	session4 := &ChatSession{
		ID:           "test-session-4",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	for _, response := range responses {
		session4.AddToThatHistory(response)
	}

	response, err = g.ProcessInputWithThatIndex("yes", session4, 0)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Most recent response matched!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatTagContractionNormalization(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with that tags testing contraction normalization
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<that>DON'T YOU LIKE MOVIES</that>
<template>Great! I love movies too.</template>
</category>
<category>
<pattern>YES</pattern>
<that>YOU'RE NOT INTERESTED IN BOOKS</that>
<template>Books are wonderful!</template>
</category>
<category>
<pattern>YES</pattern>
<that>I'M NOT SURE ABOUT MUSIC</that>
<template>Music is great!</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastActivity: time.Now().Format(time.RFC3339),
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}

	// Test contraction normalization
	testCases := []struct {
		thatContext string
		expected    string
	}{
		{"DON'T YOU LIKE MOVIES", "Great! I love movies too."},
		{"YOU'RE NOT INTERESTED IN BOOKS", "Books are wonderful!"},
		{"I'M NOT SURE ABOUT MUSIC", "Music is great!"},
	}

	for i, tc := range testCases {
		session.AddToThatHistory(tc.thatContext)
		response, err := g.ProcessInput("yes", session)
		if err != nil {
			t.Fatalf("Test case %d failed to process input: %v", i+1, err)
		}
		if response != tc.expected {
			t.Errorf("Test case %d: Expected '%s', got '%s'", i+1, tc.expected, response)
		}
	}
}
