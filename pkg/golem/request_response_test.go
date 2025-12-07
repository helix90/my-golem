package golem

import (
	"fmt"
	"testing"
	"time"
)

func TestRequestHistoryManagement(t *testing.T) {
	// Create a session
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Test adding requests to history
	session.AddToRequestHistory("Hello there!")
	session.AddToRequestHistory("How are you?")
	session.AddToRequestHistory("What's your name?")

	// Test getting request by index
	lastRequest := session.GetRequestByIndex(1)
	expected := "What's your name?"
	if lastRequest != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lastRequest)
	}

	secondRequest := session.GetRequestByIndex(2)
	expected = "How are you?"
	if secondRequest != expected {
		t.Errorf("Expected '%s', got '%s'", expected, secondRequest)
	}

	firstRequest := session.GetRequestByIndex(3)
	expected = "Hello there!"
	if firstRequest != expected {
		t.Errorf("Expected '%s', got '%s'", expected, firstRequest)
	}

	// Test getting full history
	history := session.GetRequestHistory()
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}

	// Test history limit (should keep only last 10)
	for i := 0; i < 15; i++ {
		session.AddToRequestHistory("Request " + fmt.Sprintf("%d", i))
	}

	history = session.GetRequestHistory()
	if len(history) != 10 {
		t.Errorf("Expected history length 10, got %d", len(history))
	}

	// Last request should be "Request 14" (the 15th request, but only last 10 are kept)
	lastRequest = session.GetRequestByIndex(1)
	expected = "Request 14"
	if lastRequest != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lastRequest)
	}

	// Test invalid index
	invalidRequest := session.GetRequestByIndex(0)
	if invalidRequest != "" {
		t.Errorf("Expected empty string for index 0, got '%s'", invalidRequest)
	}

	invalidRequest = session.GetRequestByIndex(11)
	if invalidRequest != "" {
		t.Errorf("Expected empty string for index 11, got '%s'", invalidRequest)
	}
}

func TestResponseHistoryManagement(t *testing.T) {
	// Create a session
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Test adding responses to history
	session.AddToResponseHistory("Hi there!")
	session.AddToResponseHistory("I'm doing well, thank you!")
	session.AddToResponseHistory("My name is Golem.")

	// Test getting response by index
	lastResponse := session.GetResponseByIndex(1)
	expected := "My name is Golem."
	if lastResponse != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lastResponse)
	}

	secondResponse := session.GetResponseByIndex(2)
	expected = "I'm doing well, thank you!"
	if secondResponse != expected {
		t.Errorf("Expected '%s', got '%s'", expected, secondResponse)
	}

	firstResponse := session.GetResponseByIndex(3)
	expected = "Hi there!"
	if firstResponse != expected {
		t.Errorf("Expected '%s', got '%s'", expected, firstResponse)
	}

	// Test getting full history
	history := session.GetResponseHistory()
	if len(history) != 3 {
		t.Errorf("Expected history length 3, got %d", len(history))
	}

	// Test history limit (should keep only last 10)
	for i := 0; i < 15; i++ {
		session.AddToResponseHistory("Response " + fmt.Sprintf("%d", i))
	}

	history = session.GetResponseHistory()
	if len(history) != 10 {
		t.Errorf("Expected history length 10, got %d", len(history))
	}

	// Last response should be "Response 14" (the 15th response, but only last 10 are kept)
	lastResponse = session.GetResponseByIndex(1)
	expected = "Response 14"
	if lastResponse != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lastResponse)
	}

	// Test invalid index
	invalidResponse := session.GetResponseByIndex(0)
	if invalidResponse != "" {
		t.Errorf("Expected empty string for index 0, got '%s'", invalidResponse)
	}

	invalidResponse = session.GetResponseByIndex(11)
	if invalidResponse != "" {
		t.Errorf("Expected empty string for index 11, got '%s'", invalidResponse)
	}
}

func TestProcessRequestTags(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with some request history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Add some requests to history
	session.AddToRequestHistory("Hello there!")
	session.AddToRequestHistory("How are you?")
	session.AddToRequestHistory("What's your name?")

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       session,
		Topic:         "",
		KnowledgeBase: nil,
	}

	// Test basic request tag (most recent)
	template := "You said: <request/>"
	result := g.processRequestTags(template, ctx)
	expected := "You said: What's your name?"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test request tag with index
	template = "You said: <request index=\"2\"/>"
	result = g.processRequestTags(template, ctx)
	expected = "You said: How are you?"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test request tag with index 1 (most recent)
	template = "You said: <request index=\"1\"/>"
	result = g.processRequestTags(template, ctx)
	expected = "You said: What's your name?"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test request tag with invalid index
	template = "You said: <request index=\"5\"/>"
	result = g.processRequestTags(template, ctx)
	expected = "You said: "
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test multiple request tags
	template = "First: <request index=\"3\"/>, Last: <request index=\"1\"/>"
	result = g.processRequestTags(template, ctx)
	expected = "First: Hello there!, Last: What's your name?"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with no session
	ctx.Session = nil
	template = "You said: <request/>"
	result = g.processRequestTags(template, ctx)
	expected = "You said: <request/>" // Should remain unchanged
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestProcessResponseTags(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with some response history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Add some responses to history
	session.AddToResponseHistory("Hi there!")
	session.AddToResponseHistory("I'm doing well, thank you!")
	session.AddToResponseHistory("My name is Golem.")

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       session,
		Topic:         "",
		KnowledgeBase: nil,
	}

	// Test basic response tag (most recent)
	template := "I said: <response/>"
	result := g.processResponseTags(template, ctx)
	expected := "I said: My name is Golem."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test response tag with index
	template = "I said: <response index=\"2\"/>"
	result = g.processResponseTags(template, ctx)
	expected = "I said: I'm doing well, thank you!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test response tag with index 1 (most recent)
	template = "I said: <response index=\"1\"/>"
	result = g.processResponseTags(template, ctx)
	expected = "I said: My name is Golem."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test response tag with invalid index
	template = "I said: <response index=\"5\"/>"
	result = g.processResponseTags(template, ctx)
	expected = "I said: "
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test multiple response tags
	template = "First: <response index=\"3\"/>, Last: <response index=\"1\"/>"
	result = g.processResponseTags(template, ctx)
	expected = "First: Hi there!, Last: My name is Golem."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with no session
	ctx.Session = nil
	template = "I said: <response/>"
	result = g.processResponseTags(template, ctx)
	expected = "I said: <response/>" // Should remain unchanged
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestRequestResponseIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Add some conversation history
	session.AddToRequestHistory("Hello there!")
	session.AddToResponseHistory("Hi! How can I help you?")
	session.AddToRequestHistory("What's your name?")
	session.AddToResponseHistory("My name is Golem.")

	// Test template with both request and response tags
	template := "You said: <request index=\"1\"/>, and I replied: <response index=\"1\"/>"
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected := "You said: What's your name?, and I replied: My name is Golem."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test template with older history
	template = "Earlier you said: <request index=\"2\"/>, and I replied: <response index=\"2\"/>"
	result = g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected = "Earlier you said: Hello there!, and I replied: Hi! How can I help you?"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestRequestResponseWithEmptyHistory(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session with no history
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Test template with request/response tags when no history exists
	template := "You said: <request/>, and I replied: <response/>"
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected := "You said: , and I replied:"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestRequestResponseWithRealConversation(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session
	session := &ChatSession{
		ID:              "test-session",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		RequestHistory:  make([]string, 0),
		ResponseHistory: make([]string, 0),
	}

	// Manually add conversation history (simulating a previous conversation)
	session.AddToRequestHistory("Hello there!")
	session.AddToResponseHistory("Hello! Nice to meet you.")
	session.AddToRequestHistory("How are you?")
	session.AddToResponseHistory("I'm doing well, thank you!")
	session.AddToRequestHistory("What's your name?")
	session.AddToResponseHistory("My name is Golem.")

	// Test request tag processing directly
	template := "You said: <request index=\"1\"/>"
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected := "You said: What's your name?"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test response tag processing directly
	template = "I said: <response index=\"1\"/>"
	result = g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected = "I said: My name is Golem."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test both tags together
	template = "You: <request index=\"2\"/>, Me: <response index=\"2\"/>"
	result = g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected = "You: How are you?, Me: I'm doing well, thank you!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
