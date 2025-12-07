package golem

import (
	"fmt"
	"testing"
	"time"
)

func TestThatTagMatching(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with that tags
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE MOVIES</that>
<template>Great! I love movies too.</template>
</category>
<category>
<pattern>YES</pattern>
<that>DO YOU LIKE BOOKS</that>
<template>Excellent! Books are wonderful.</template>
</category>
<category>
<pattern>YES</pattern>
<template>That's good to hear!</template>
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

	// Test 1: No that context - should match general YES
	response, err := g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "That's good to hear!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Set that context and test specific matching
	session.AddToThatHistory("DO YOU LIKE MOVIES")
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Great! I love movies too."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Change that context
	session.AddToThatHistory("DO YOU LIKE BOOKS")
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Excellent! Books are wonderful."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestTopicTagMatching(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with topic tags
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>LET'S TALK ABOUT MOVIES</pattern>
<template>Sure! <set name="topic">movies</set> What would you like to know about movies?</template>
</category>
<category>
<pattern>WHAT IS YOUR FAVORITE MOVIE</pattern>
<topic>movies</topic>
<template>I love classic films like "Casablanca".</template>
</category>
<category>
<pattern>WHAT IS YOUR FAVORITE MOVIE</pattern>
<template>I don't have a favorite movie right now.</template>
</category>
<category>
<pattern>LET'S TALK ABOUT BOOKS</pattern>
<template>Great! <set name="topic">books</set> What would you like to know about books?</template>
</category>
<category>
<pattern>WHAT IS YOUR FAVORITE BOOK</pattern>
<topic>books</topic>
<template>I love "To Kill a Mockingbird".</template>
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

	// Test 1: Set movies topic
	response, err := g.ProcessInput("let's talk about movies", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Sure!  What would you like to know about movies?" // Extra space from set tag
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify topic was set
	if session.GetSessionTopic() != "movies" {
		t.Errorf("Expected topic 'movies', got '%s'", session.GetSessionTopic())
	}

	// Test 2: Ask about favorite movie in movies topic
	response, err = g.ProcessInput("what is your favorite movie", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "I love classic films like \"Casablanca\"."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Change to books topic
	response, err = g.ProcessInput("let's talk about books", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Great!  What would you like to know about books?" // Extra space from set tag
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify topic was changed
	if session.GetSessionTopic() != "books" {
		t.Errorf("Expected topic 'books', got '%s'", session.GetSessionTopic())
	}

	// Test 4: Ask about favorite book in books topic
	response, err = g.ProcessInput("what is your favorite book", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "I love \"To Kill a Mockingbird\"."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 5: Ask about favorite movie without movies topic (should get general response)
	session.SetSessionTopic("")
	response, err = g.ProcessInput("what is your favorite movie", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "I don't have a favorite movie right now."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatAndTopicCombined(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with both that and topic tags
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>LET'S TALK ABOUT MOVIES</pattern>
<template>Sure! <set name="topic">movies</set> What would you like to know about movies?</template>
</category>
<category>
<pattern>YES</pattern>
<that>WHAT WOULD YOU LIKE TO KNOW ABOUT MOVIES</that>
<topic>movies</topic>
<template>Tell me about action movies!</template>
</category>
<category>
<pattern>YES</pattern>
<that>TELL ME ABOUT ACTION MOVIES</that>
<topic>movies</topic>
<template>I'm interested in learning more about movies.</template>
</category>
<category>
<pattern>YES</pattern>
<template>That's good to hear!</template>
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

	// Test 1: Set movies topic
	response, err := g.ProcessInput("let's talk about movies", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Sure!  What would you like to know about movies?" // Extra space from set tag
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Respond with YES in movies topic context
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Tell me about action movies!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Continue with movies topic context
	response, err = g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "I'm interested in learning more about movies."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatHistoryManagement(t *testing.T) {
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

	// Test adding responses to that history
	session.AddToThatHistory("Hello there!")
	session.AddToThatHistory("How are you?")
	session.AddToThatHistory("What's your name?")

	// Test getting last that
	lastThat := session.GetLastThat()
	expected := "What's your name?"
	if lastThat != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lastThat)
	}

	// Test getting full history
	history := session.GetThatHistory()
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}

	// Test history limit (should keep only last 10)
	for i := 0; i < 15; i++ {
		session.AddToThatHistory("Response " + fmt.Sprintf("%d", i))
	}

	history = session.GetThatHistory()
	if len(history) != 10 {
		t.Errorf("Expected history length 10, got %d", len(history))
	}

	// Last response should be "Response 14" (the 15th response, but only last 10 are kept)
	lastThat = session.GetLastThat()
	expected = "Response 14"
	if lastThat != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lastThat)
	}
}

func TestTopicManagement(t *testing.T) {
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

	// Test setting and getting topic
	session.SetSessionTopic("movies")
	topic := session.GetSessionTopic()
	expected := "movies"
	if topic != expected {
		t.Errorf("Expected '%s', got '%s'", expected, topic)
	}

	// Test changing topic
	session.SetSessionTopic("books")
	topic = session.GetSessionTopic()
	expected = "books"
	if topic != expected {
		t.Errorf("Expected '%s', got '%s'", expected, topic)
	}

	// Test clearing topic
	session.SetSessionTopic("")
	topic = session.GetSessionTopic()
	expected = ""
	if topic != expected {
		t.Errorf("Expected '%s', got '%s'", expected, topic)
	}
}

func TestThatTagWithWildcards(t *testing.T) {
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
<pattern>YES</pattern>
<template>That's good to hear!</template>
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

	// Test 1: Set that context with wildcard
	session.AddToThatHistory("DO YOU LIKE PIZZA")
	response, err := g.ProcessInput("yes", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Great! I love PIZZA too."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Change that context
	session.AddToThatHistory("DO YOU LIKE BOOKS")
	response, err = g.ProcessInput("no", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "That's okay, not everyone likes BOOKS."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestTopicTagWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with topic tags containing wildcards
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>LET'S TALK ABOUT *</pattern>
<template>Sure! <set name="topic"><star/></set> What would you like to know about <star/>?</template>
</category>
<category>
<pattern>WHAT IS YOUR FAVORITE *</pattern>
<topic>*</topic>
<template>I love <star/>! It's one of my favorites.</template>
</category>
<category>
<pattern>WHAT IS YOUR FAVORITE *</pattern>
<template>I don't have a favorite <star/> right now.</template>
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

	// Test 1: Set topic with wildcard
	response, err := g.ProcessInput("let's talk about music", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Sure!  What would you like to know about music?" // Extra space from empty set tag
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify topic was set
	if session.GetSessionTopic() != "music" {
		t.Errorf("Expected topic 'music', got '%s'", session.GetSessionTopic())
	}

	// Test 2: Ask about favorite with topic context
	response, err = g.ProcessInput("what is your favorite song", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "I love song! It's one of my favorites."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestThatTopicRegression(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with various tags to ensure no regression
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>HELLO</pattern>
<template>Hello! How can I help you?</template>
</category>
<category>
<pattern>WHAT IS YOUR NAME</pattern>
<template>My name is <get name="name"/>.</template>
</category>
<category>
<pattern>SET MY NAME TO *</pattern>
<template>Nice to meet you, <set name="name"><star/></set>!</template>
</category>
<category>
<pattern>RANDOM TEST</pattern>
<template><random><li>Option 1</li><li>Option 2</li><li>Option 3</li></random></template>
</category>
<category>
<pattern>CONDITION TEST</pattern>
<template><condition name="name"><li value="John">Hello John!</li><li value="Jane">Hello Jane!</li><li>Hello there!</li></condition></template>
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

	// Test 1: Basic greeting (no that/topic)
	response, err := g.ProcessInput("hello", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected := "Hello! How can I help you?"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Set name
	response, err = g.ProcessInput("set my name to John", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Nice to meet you, !"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Get name
	response, err = g.ProcessInput("what is your name", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "My name is John."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 4: Random tag
	response, err = g.ProcessInput("random test", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	if response != "Option 1" && response != "Option 2" && response != "Option 3" {
		t.Errorf("Expected one of 'Option 1', 'Option 2', 'Option 3', got '%s'", response)
	}

	// Test 5: Condition tag
	response, err = g.ProcessInput("condition test", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}
	expected = "Hello John!"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}
