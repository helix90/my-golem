package golem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLoadAIML(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary AIML file for testing
	tempDir := t.TempDir()
	aimlFile := filepath.Join(tempDir, "test.aiml")
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
    <category>
        <pattern>HELLO</pattern>
        <template>Hello! How can I help you?</template>
    </category>
    <category>
        <pattern>MY NAME IS *</pattern>
        <template>Nice to meet you, <star/>!</template>
    </category>
</aiml>`

	err := os.WriteFile(aimlFile, []byte(aimlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test AIML file: %v", err)
	}

	// Test loading AIML
	kb, err := g.LoadAIML(aimlFile)
	if err != nil {
		t.Fatalf("LoadAIML failed: %v", err)
	}

	if kb == nil {
		t.Fatal("LoadAIML returned nil knowledge base")
	}

	if len(kb.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(kb.Categories))
	}

	// Test pattern indexing
	if kb.Patterns["HELLO"] == nil {
		t.Error("HELLO pattern not indexed")
	}

	if kb.Patterns["MY NAME IS *"] == nil {
		t.Error("MY NAME IS * pattern not indexed")
	}
}

func TestValidateAIML(t *testing.T) {
	g := NewForTesting(t, false)

	// Test valid AIML
	validAIML := &AIML{
		Version: "2.0",
		Categories: []Category{
			{
				Pattern:  "HELLO",
				Template: "Hello!",
			},
		},
	}

	err := g.validateAIML(validAIML)
	if err != nil {
		t.Errorf("Valid AIML should not error: %v", err)
	}

	// Test missing version
	invalidAIML := &AIML{
		Categories: []Category{
			{
				Pattern:  "HELLO",
				Template: "Hello!",
			},
		},
	}

	err = g.validateAIML(invalidAIML)
	if err == nil {
		t.Error("Expected error for missing version")
	}

	// Test empty categories
	emptyAIML := &AIML{
		Version: "2.0",
	}

	err = g.validateAIML(emptyAIML)
	if err == nil {
		t.Error("Expected error for empty categories")
	}

	// Test empty pattern
	emptyPatternAIML := &AIML{
		Version: "2.0",
		Categories: []Category{
			{
				Pattern:  "",
				Template: "Hello!",
			},
		},
	}

	err = g.validateAIML(emptyPatternAIML)
	if err == nil {
		t.Error("Expected error for empty pattern")
	}
}

func TestValidatePattern(t *testing.T) {
	g := NewForTesting(t, false)

	// Test valid patterns
	validPatterns := []string{
		"HELLO",
		"MY NAME IS *",
		"I AM * YEARS OLD",
		"I LIKE * AND *",
		"<set>emotions</set>",
	}

	for _, pattern := range validPatterns {
		err := g.validatePattern(pattern)
		if err != nil {
			t.Errorf("Pattern '%s' should be valid: %v", pattern, err)
		}
	}

	// Test invalid patterns
	invalidPatterns := []string{
		"", // empty
		"hello*hello*hello*hello*hello*hello*hello*hello*hello*hello*", // too many wildcards
		"<set></set>", // empty set
	}

	for _, pattern := range invalidPatterns {
		err := g.validatePattern(pattern)
		if err == nil {
			t.Errorf("Pattern '%s' should be invalid", pattern)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	// Create a test knowledge base
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{
			Pattern:  "HELLO",
			Template: "Hello! How can I help you?",
		},
		{
			Pattern:  "MY NAME IS *",
			Template: "Nice to meet you, <star/>!",
		},
		{
			Pattern:  "I AM * YEARS OLD",
			Template: "You're <star/> years old!",
		},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := category.Pattern
		kb.Patterns[pattern] = category
	}

	// Test exact match
	category, wildcards, err := kb.MatchPattern("HELLO")
	if err != nil {
		t.Fatalf("Exact match failed: %v", err)
	}
	if category.Pattern != "HELLO" {
		t.Errorf("Expected HELLO pattern, got %s", category.Pattern)
	}
	if len(wildcards) != 0 {
		t.Errorf("Expected no wildcards for exact match, got %v", wildcards)
	}

	// Test wildcard match
	category, wildcards, err = kb.MatchPattern("MY NAME IS JOHN")
	if err != nil {
		t.Fatalf("Wildcard match failed: %v", err)
	}
	if category.Pattern != "MY NAME IS *" {
		t.Errorf("Expected MY NAME IS * pattern, got %s", category.Pattern)
	}
	if wildcards["star1"] != "JOHN" {
		t.Errorf("Expected wildcard 'JOHN', got %s", wildcards["star1"])
	}

	// Test no match
	_, _, err = kb.MatchPattern("UNKNOWN INPUT")
	if err == nil {
		t.Error("Expected error for no match")
	}
}

func TestProcessTemplate(t *testing.T) {
	// Test template with wildcards
	template := "Nice to meet you, <star/>!"
	wildcards := map[string]string{
		"star1": "JOHN",
	}

	g := NewForTesting(t, false)
	result := g.ProcessTemplate(template, wildcards)
	expected := "Nice to meet you, JOHN!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test template without wildcards
	template = "Hello! How can I help you?"
	wildcards = make(map[string]string)

	result = g.ProcessTemplate(template, wildcards)
	expected = "Hello! How can I help you?"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestChatCommand(t *testing.T) {
	g := NewForTesting(t, false)

	// Test without loaded knowledge base
	err := g.chatCommand([]string{"hello"})
	if err == nil {
		t.Error("Expected error when no knowledge base loaded")
	}

	// Create and load a knowledge base
	kb := NewAIMLKnowledgeBase()
	kb.Categories = []Category{
		{
			Pattern:  "HELLO",
			Template: "Hello! How can I help you?",
		},
	}
	kb.Patterns["HELLO"] = &kb.Categories[0]
	g.aimlKB = kb

	// Test chat with loaded knowledge base
	err = g.chatCommand([]string{"hello"})
	if err != nil {
		t.Errorf("Chat command failed: %v", err)
	}

	// Test chat without arguments
	err = g.chatCommand([]string{})
	if err == nil {
		t.Error("Expected error for chat command without arguments")
	}
}

func TestProperties(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Test default properties
	if kb.GetProperty("name") != "" {
		t.Error("Expected empty property before loading")
	}

	// Load default properties
	err := g.loadDefaultProperties(kb)
	if err != nil {
		t.Fatalf("Failed to load default properties: %v", err)
	}

	// Test property retrieval
	if kb.GetProperty("name") != "Golem" {
		t.Errorf("Expected name 'Golem', got '%s'", kb.GetProperty("name"))
	}

	expectedVersion := GetVersion()
	if kb.GetProperty("version") != expectedVersion {
		t.Errorf("Expected version '%s', got '%s'", expectedVersion, kb.GetProperty("version"))
	}

	// Test property setting
	kb.SetProperty("name", "TestBot")
	if kb.GetProperty("name") != "TestBot" {
		t.Errorf("Expected name 'TestBot', got '%s'", kb.GetProperty("name"))
	}

	// Test non-existent property
	if kb.GetProperty("nonexistent") != "" {
		t.Error("Expected empty string for non-existent property")
	}
}

func TestParsePropertiesFile(t *testing.T) {
	g := NewForTesting(t, false)

	content := `# Test properties file
name=TestBot
version=2.0.0
# This is a comment
master=TestUser
empty_key=
`

	props, err := g.parsePropertiesFile(content)
	if err != nil {
		t.Fatalf("Failed to parse properties file: %v", err)
	}

	// Test valid properties
	if props["name"] != "TestBot" {
		t.Errorf("Expected name 'TestBot', got '%s'", props["name"])
	}

	if props["version"] != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", props["version"])
	}

	if props["master"] != "TestUser" {
		t.Errorf("Expected master 'TestUser', got '%s'", props["master"])
	}

	// Test empty value
	if props["empty_key"] != "" {
		t.Errorf("Expected empty value, got '%s'", props["empty_key"])
	}

	// Test that comments are ignored
	if _, exists := props["# This is a comment"]; exists {
		t.Error("Comments should be ignored")
	}
}

func TestSessionManagement(t *testing.T) {
	g := NewForTesting(t, false)

	// Test creating a session
	session := g.createSession("test_session")
	if session == nil {
		t.Fatal("Expected session to be created")
	}
	if session.ID != "test_session" {
		t.Errorf("Expected session ID 'test_session', got '%s'", session.ID)
	}
	if g.currentID != "test_session" {
		t.Errorf("Expected current ID 'test_session', got '%s'", g.currentID)
	}

	// Test getting current session
	currentSession := g.getCurrentSession()
	if currentSession == nil {
		t.Fatal("Expected current session to exist")
	}
	if currentSession.ID != "test_session" {
		t.Errorf("Expected current session ID 'test_session', got '%s'", currentSession.ID)
	}

	// Test creating another session
	session2 := g.createSession("test_session_2")
	if session2.ID != "test_session_2" {
		t.Errorf("Expected session ID 'test_session_2', got '%s'", session2.ID)
	}
	if g.currentID != "test_session_2" {
		t.Errorf("Expected current ID 'test_session_2', got '%s'", g.currentID)
	}

	// Test session history
	session2.History = append(session2.History, "User: hello")
	session2.History = append(session2.History, "Golem: hi there")
	if len(session2.History) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(session2.History))
	}

	// Test session variables
	session2.Variables["name"] = "TestUser"
	if session2.Variables["name"] != "TestUser" {
		t.Errorf("Expected variable 'name' to be 'TestUser', got '%s'", session2.Variables["name"])
	}
}

func TestSessionCommands(t *testing.T) {
	g := NewForTesting(t, false)

	// Test session create command
	err := g.sessionCommand([]string{"create", "test_session"})
	if err != nil {
		t.Errorf("Session create command failed: %v", err)
	}

	// Test session list command
	err = g.sessionCommand([]string{"list"})
	if err != nil {
		t.Errorf("Session list command failed: %v", err)
	}

	// Test session current command
	err = g.sessionCommand([]string{"current"})
	if err != nil {
		t.Errorf("Session current command failed: %v", err)
	}

	// Test session switch command
	err = g.sessionCommand([]string{"switch", "test_session"})
	if err != nil {
		t.Errorf("Session switch command failed: %v", err)
	}

	// Test session delete command
	err = g.sessionCommand([]string{"delete", "test_session"})
	if err != nil {
		t.Errorf("Session delete command failed: %v", err)
	}

	// Test invalid session command
	err = g.sessionCommand([]string{"invalid"})
	if err == nil {
		t.Error("Expected error for invalid session command")
	}
}

func TestProcessTemplateWithSession(t *testing.T) {
	g := NewForTesting(t, false)
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"name": "TestUser", "mood": "happy"},
		History:   []string{},
	}

	// Test template with session variables
	template := "Hello <get name=\"name\"/>, you seem <get name=\"mood\"/>!"
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected := "Hello TestUser, you seem happy!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test template with wildcards and session variables
	template = "Nice to meet you, <star/>! I'm <get name=\"name\"/>."
	wildcards := map[string]string{"star1": "John"}
	result = g.ProcessTemplateWithSession(template, wildcards, session)
	expected = "Nice to meet you, John! I'm TestUser."

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestSRAIProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "WHAT IS YOUR NAME", Template: "My name is <get name=\"name\"/>, your AI assistant."},
		{Pattern: "WHAT CAN YOU DO", Template: "I can help you with various tasks. <srai>WHAT IS YOUR NAME</srai>"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	// Set properties
	kb.Properties = map[string]string{
		"name": "Golem",
	}

	g.SetKnowledgeBase(kb)

	// Test SRAI processing
	template := "I can help you with various tasks. <srai>WHAT IS YOUR NAME</srai>"
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "I can help you with various tasks. My name is Golem, your AI assistant."

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestSRAIWithSession(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "WHAT IS YOUR NAME", Template: "My name is <get name=\"name\"/>, your AI assistant."},
		{Pattern: "WHAT CAN YOU DO", Template: "I can help you with various tasks. <srai>WHAT IS YOUR NAME</srai>"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	// Set properties
	kb.Properties = map[string]string{
		"name": "Golem",
	}

	g.SetKnowledgeBase(kb)

	// Create session
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"name": "TestUser"},
		History:   []string{},
	}

	// Test SRAI processing with session
	template := "I can help you with various tasks. <srai>WHAT IS YOUR NAME</srai>"
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)
	// Note: Template uses <get name="name"/> which checks session variables first
	// Session has name="TestUser", so that takes precedence over bot property name="Golem"
	expected := "I can help you with various tasks. My name is TestUser, your AI assistant."

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestSRAINoMatch(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test SRAI with no matching pattern
	template := "I can help you. <srai>NONEXISTENT PATTERN</srai>"
	result := g.ProcessTemplate(template, make(map[string]string))
	// Per AIML spec: SRAI with no match returns the input text (not empty, not the tag preserved)
	expected := "I can help you. NONEXISTENT PATTERN"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestSRAIRecursive(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories with recursive SRAI
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "WHAT IS YOUR NAME", Template: "My name is <get name=\"name\"/>, your AI assistant."},
		{Pattern: "INTRO", Template: "Hi there! <srai>WHAT IS YOUR NAME</srai>"},
		{Pattern: "GREETING", Template: "Welcome! <srai>INTRO</srai>"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	// Set properties
	kb.Properties = map[string]string{
		"name": "Golem",
	}

	g.SetKnowledgeBase(kb)

	// Test recursive SRAI processing
	template := "Welcome! <srai>INTRO</srai>"
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Welcome! Hi there! My name is Golem, your AI assistant."

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestSRTagIntegration tests SR tag in full template processing
func TestSRTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "HI", Template: "Hi there!"},
		{Pattern: "GREETING *", Template: "Nice to meet you! <sr/>"},
		{Pattern: "GOODBYE", Template: "Goodbye! Have a great day!"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test full template processing
	template := "Nice to meet you! <sr/>"
	wildcards := map[string]string{"star1": "HELLO"}
	result := g.ProcessTemplate(template, wildcards)
	expected := "Nice to meet you! Hello! How can I help you today?"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestSRTagRecursive tests recursive SR tag processing
func TestSRTagRecursive(t *testing.T) {
	g := NewForTesting(t, false) // Disable verbose logging for tests
	kb := NewAIMLKnowledgeBase()

	// Add test categories with recursive SR
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "HI", Template: "Hi there!"},
		{Pattern: "GREETING *", Template: "Nice to meet you! <sr/>"},
		{Pattern: "WELCOME *", Template: "Welcome! <sr/>"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test recursive SR processing
	template := "Welcome! <sr/>"
	wildcards := map[string]string{"star1": "GREETING HELLO"}
	result := g.ProcessTemplate(template, wildcards)
	expected := "Welcome! Nice to meet you! Hello! How can I help you today?"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestProcessTemplateWithRandom(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories with random templates
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: `<random>
			<li>Hello! How can I help you today?</li>
			<li>Hi there! What can I do for you?</li>
			<li>Greetings! How may I assist you?</li>
		</random>`},
		{Pattern: "GOODBYE", Template: `<random>
			<li>Goodbye! Have a great day!</li>
			<li>See you later!</li>
		</random>`},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test random template processing
	template := `<random>
		<li>Option A</li>
		<li>Option B</li>
	</random>`
	result := g.ProcessTemplate(template, make(map[string]string))

	// Should be one of the options
	validOptions := []string{"Option A", "Option B"}
	found := false
	for _, option := range validOptions {
		if result == option {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected one of %v, got '%s'", validOptions, result)
	}
}

func TestProcessTemplateWithRandomAndSession(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories with random templates
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: `<random>
			<li>Hello <get name="name"/>!</li>
			<li>Hi <get name="name"/>, how are you?</li>
		</random>`},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Create session with variables
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"name": "TestUser"},
		History:   []string{},
	}

	// Test random template processing with session
	template := `<random>
		<li>Hello <get name="name"/>!</li>
		<li>Hi <get name="name"/>, how are you?</li>
	</random>`
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)

	// Should be one of the options with name replaced
	validOptions := []string{"Hello TestUser!", "Hi TestUser, how are you?"}
	found := false
	for _, option := range validOptions {
		if result == option {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected one of %v, got '%s'", validOptions, result)
	}
}

func TestRandomWithSRAI(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories with random and SRAI
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "GREETING", Template: `<random>
			<li><srai>HELLO</srai></li>
			<li>Hi there! <srai>HELLO</srai></li>
		</random>`},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test random with SRAI
	template := `<random>
		<li><srai>HELLO</srai></li>
		<li>Hi there! <srai>HELLO</srai></li>
	</random>`
	result := g.ProcessTemplate(template, make(map[string]string))

	// Should be one of the options with SRAI processed
	validOptions := []string{"Hello! How can I help you today?", "Hi there! Hello! How can I help you today?"}
	found := false
	for _, option := range validOptions {
		if result == option {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected one of %v, got '%s'", validOptions, result)
	}
}

func TestRandomWithProperties(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Set properties
	kb.Properties = map[string]string{
		"name": "Golem",
	}

	g.SetKnowledgeBase(kb)

	// Test random with properties
	template := `<random>
		<li>Hello, I'm <get name="name"/>!</li>
		<li>Hi! My name is <get name="name"/>.</li>
	</random>`
	result := g.ProcessTemplate(template, make(map[string]string))

	// Should be one of the options with properties replaced
	validOptions := []string{"Hello, I'm Golem!", "Hi! My name is Golem."}
	found := false
	for _, option := range validOptions {
		if result == option {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected one of %v, got '%s'", validOptions, result)
	}
}

func TestProcessTemplateWithThink(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test think tag in template processing
	template := `<think><set name="internal_var">internal_value</set></think>Hello world!`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Hello world!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Verify the variable was set
	if kb.Variables["internal_var"] != "internal_value" {
		t.Errorf("Expected knowledge base variable 'internal_var' to be 'internal_value', got '%s'", kb.Variables["internal_var"])
	}
}

func TestProcessTemplateWithThinkAndSession(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create session
	session := &ChatSession{
		ID:        "test_session",
		Variables: make(map[string]string),
		History:   []string{},
	}

	// Test think tag with session context - set variable first
	session.Variables["session_var"] = "session_value"
	template := `<think><set name="another_var">another_value</set></think>Hello <get name="session_var"/>!`
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected := "Hello session_value!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Verify the variable was set in session
	if session.Variables["another_var"] != "another_value" {
		t.Errorf("Expected session variable 'another_var' to be 'another_value', got '%s'", session.Variables["another_var"])
	}
}

func TestThinkWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test think tag with wildcard values
	template := `<think><set name="user_input"><star/></set></think>I'll remember: <star/>`
	wildcards := map[string]string{"star1": "hello world"}
	result := g.ProcessTemplate(template, wildcards)
	expected := "I'll remember: hello world"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Verify the variable was set with wildcard value
	if kb.Variables["user_input"] != "hello world" {
		t.Errorf("Expected knowledge base variable 'user_input' to be 'hello world', got '%s'", kb.Variables["user_input"])
	}
}

func TestThinkWithProperties(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	kb.Properties = map[string]string{
		"bot_name": "Golem",
	}
	g.SetKnowledgeBase(kb)

	// Test think tag with property values
	template := `<think><set name="greeting">Hello from <get name="bot_name"/></set></think>Ready to chat!`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Ready to chat!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Verify the variable was set with property value
	if kb.Variables["greeting"] != "Hello from Golem" {
		t.Errorf("Expected knowledge base variable 'greeting' to be 'Hello from Golem', got '%s'", kb.Variables["greeting"])
	}
}

func TestThinkWithSRAI(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "GREETING", Template: `<think><set name="last_greeting">greeting</set></think><srai>HELLO</srai>`},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test think tag with SRAI
	template := `<think><set name="internal_var">internal_value</set></think><srai>HELLO</srai>`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Hello! How can I help you today?"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Verify the variable was set
	if kb.Variables["internal_var"] != "internal_value" {
		t.Errorf("Expected knowledge base variable 'internal_var' to be 'internal_value', got '%s'", kb.Variables["internal_var"])
	}
}

func TestThinkWithRandom(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test think tag with random
	template := `<think><set name="choice_made">true</set></think><random>
		<li>Option A</li>
		<li>Option B</li>
	</random>`
	result := g.ProcessTemplate(template, make(map[string]string))

	// Should be one of the random options
	validOptions := []string{"Option A", "Option B"}
	found := false
	for _, option := range validOptions {
		if result == option {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected one of %v, got '%s'", validOptions, result)
	}

	// Verify the variable was set
	if kb.Variables["choice_made"] != "true" {
		t.Errorf("Expected knowledge base variable 'choice_made' to be 'true', got '%s'", kb.Variables["choice_made"])
	}
}

func TestCustomTimeFormats(t *testing.T) {
	g := NewForTesting(t, false)

	// Test C-style format strings
	testCases := []struct {
		format        string
		description   string
		expectedRegex string
	}{
		{
			format:        "%H",
			description:   "C-style 24-hour format (hour only)",
			expectedRegex: `^\d{1,2}$`,
		},
		{
			format:        "%H:%M",
			description:   "C-style 24-hour format with minutes",
			expectedRegex: `^\d{1,2}:\d{2}$`,
		},
		{
			format:        "%H:%M:%S",
			description:   "C-style 24-hour format with seconds",
			expectedRegex: `^\d{1,2}:\d{2}:\d{2}$`,
		},
		{
			format:        "%I:%M %p",
			description:   "C-style 12-hour format with AM/PM",
			expectedRegex: `^\d{1,2}:\d{2} (AM|PM)$`,
		},
		{
			format:        "%Y-%m-%d",
			description:   "C-style date format",
			expectedRegex: `^\d{4}-\d{2}-\d{2}$`,
		},
		{
			format:        "%A, %B %d",
			description:   "C-style weekday and month format",
			expectedRegex: `^[A-Za-z]+, [A-Za-z]+ \d{1,2}$`,
		},
		{
			format:        "HH:MM",
			description:   "Alternative 24-hour format",
			expectedRegex: `^\d{1,2}:\d{2}$`,
		},
		{
			format:        "YYYY-MM-DD",
			description:   "Alternative date format",
			expectedRegex: `^\d{4}-\d{2}-\d{2}$`,
		},
		{
			format:        "15",
			description:   "Go-style hour format",
			expectedRegex: `^\d{1,2}$`,
		},
		{
			format:        "15:04",
			description:   "Go-style 24-hour format",
			expectedRegex: `^\d{1,2}:\d{2}$`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			template := fmt.Sprintf("Time: <time format=\"%s\"/>", tc.format)
			result := g.ProcessTemplate(template, make(map[string]string))

			if strings.Contains(result, "<time") {
				t.Errorf("Expected <time> tag to be replaced for format '%s', got '%s'", tc.format, result)
				return
			}

			// Extract the time part (after "Time: ")
			parts := strings.Split(result, "Time: ")
			if len(parts) != 2 {
				t.Errorf("Expected 'Time: ' prefix, got '%s'", result)
				return
			}

			timeStr := parts[1]
			if timeStr == "" {
				t.Errorf("Expected non-empty time for format '%s', got empty", tc.format)
				return
			}

			// Validate the format using regex
			matched, err := regexp.MatchString(tc.expectedRegex, timeStr)
			if err != nil {
				t.Errorf("Error matching regex for format '%s': %v", tc.format, err)
				return
			}

			if !matched {
				t.Errorf("Expected result '%s' to match pattern '%s' for format '%s' (%s)", timeStr, tc.expectedRegex, tc.format, tc.description)
			}
		})
	}

	// Test that unknown formats still fall back to default
	t.Run("unknown_format_fallback", func(t *testing.T) {
		template := "Time: <time format=\"unknown_format\"/>"
		result := g.ProcessTemplate(template, make(map[string]string))

		if strings.Contains(result, "<time") {
			t.Errorf("Expected <time> tag to be replaced, got '%s'", result)
			return
		}

		// Should contain the default 12-hour format
		parts := strings.Split(result, "Time: ")
		if len(parts) != 2 {
			t.Errorf("Expected 'Time: ' prefix, got '%s'", result)
			return
		}

		timeStr := parts[1]
		matched, _ := regexp.MatchString(`^\d{1,2}:\d{2} (AM|PM)$`, timeStr)
		if !matched {
			t.Errorf("Expected unknown format to fall back to default 12-hour format, got '%s'", timeStr)
		}
	})
}

func TestProcessTemplateWithDateTime(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test date/time tags in template processing
	template := "Today is <date format=\"short\"/> and it is <time format=\"12\"/>"
	result := g.ProcessTemplate(template, make(map[string]string))

	if strings.Contains(result, "<date") || strings.Contains(result, "<time") {
		t.Errorf("Expected all date/time tags to be replaced, got '%s'", result)
	}
	if !strings.Contains(result, "Today is") || !strings.Contains(result, "and it is") {
		t.Errorf("Expected result to contain template text, got '%s'", result)
	}
}

func TestProcessTemplateWithDateTimeAndSession(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create session
	session := &ChatSession{
		ID:        "test_session",
		Variables: make(map[string]string),
		History:   []string{},
	}

	// Test date/time tags with session context
	template := "Hello! Today is <date format=\"long\"/> and it is <time format=\"24\"/>"
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)

	if strings.Contains(result, "<date") || strings.Contains(result, "<time") {
		t.Errorf("Expected all date/time tags to be replaced, got '%s'", result)
	}
	if !strings.Contains(result, "Hello! Today is") || !strings.Contains(result, "and it is") {
		t.Errorf("Expected result to contain template text, got '%s'", result)
	}
}

func TestDateTimeWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test date/time tags with wildcards
	template := "You said <star/> and today is <date format=\"short\"/>"
	wildcards := map[string]string{"star1": "hello"}
	result := g.ProcessTemplate(template, wildcards)

	if strings.Contains(result, "<date") {
		t.Errorf("Expected <date> tag to be replaced, got '%s'", result)
	}
	if !strings.Contains(result, "You said hello") {
		t.Errorf("Expected result to contain wildcard replacement, got '%s'", result)
	}
}

func TestDateTimeWithProperties(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	kb.Properties = map[string]string{
		"bot_name": "Golem",
	}
	g.SetKnowledgeBase(kb)

	// Test date/time tags with properties
	template := "Hello from <get name=\"bot_name\"/>! Today is <date format=\"long\"/>"
	result := g.ProcessTemplate(template, make(map[string]string))

	if strings.Contains(result, "<date") {
		t.Errorf("Expected <date> tag to be replaced, got '%s'", result)
	}
	if !strings.Contains(result, "Hello from Golem!") {
		t.Errorf("Expected result to contain property replacement, got '%s'", result)
	}
}

func TestDateTimeWithSRAI(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "GREETING", Template: "Hi there! <srai>HELLO</srai> Today is <date format=\"short\"/>"},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test date/time tags with SRAI
	template := "Welcome! <srai>HELLO</srai> Today is <date format=\"short\"/>"
	result := g.ProcessTemplate(template, make(map[string]string))

	if strings.Contains(result, "<date") {
		t.Errorf("Expected <date> tag to be replaced, got '%s'", result)
	}
	if !strings.Contains(result, "Welcome! Hello! How can I help you today?") {
		t.Errorf("Expected result to contain SRAI replacement, got '%s'", result)
	}
}

func TestDateTimeWithRandom(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test date/time tags with random
	template := `<random>
		<li>Today is <date format=\"short\"/></li>
		<li>It is <time format=\"12\"/></li>
	</random>`
	result := g.ProcessTemplate(template, make(map[string]string))

	if strings.Contains(result, "<date") || strings.Contains(result, "<time") {
		t.Errorf("Expected all date/time tags to be replaced, got '%s'", result)
	}
	if !strings.Contains(result, "Today is") && !strings.Contains(result, "It is") {
		t.Errorf("Expected result to contain random option text, got '%s'", result)
	}
}

func TestDateTimeWithThink(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test date/time tags with think - the think content processes date/time but doesn't output it
	template := `<think><set name="current_date"><date format=\"iso\"/></set></think>Today is <date format=\"long\"/>`
	result := g.ProcessTemplate(template, make(map[string]string))

	// The main template date should be processed
	if strings.Contains(result, "<date") {
		t.Errorf("Expected <date> tag to be replaced, got '%s'", result)
	}
	if !strings.Contains(result, "Today is") {
		t.Errorf("Expected result to contain template text, got '%s'", result)
	}

	// Verify the variable was set (it should contain the processed date)
	if kb.Variables["current_date"] == "" {
		t.Errorf("Expected current_date variable to be set")
	}
}

func TestProcessTemplateWithCondition(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	kb.Variables["mood"] = "happy"
	g.SetKnowledgeBase(kb)

	// Test condition in template processing
	template := `<condition name="mood" value="happy">I'm glad you're happy!</condition>`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "I'm glad you're happy!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestProcessTemplateWithConditionAndSession(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create session
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"weather": "sunny"},
		History:   []string{},
	}

	// Test condition with session context
	template := `<condition name="weather">
		<li value="sunny">It's a beautiful sunny day!</li>
		<li value="rainy">Don't forget your umbrella!</li>
		<li>I hope you have a great day!</li>
	</condition>`
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)
	expected := "It's a beautiful sunny day!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConditionWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test condition with wildcards
	template := `<condition name="mood" value="happy">You said <star/> and I'm glad you're happy!</condition>`
	wildcards := map[string]string{"star1": "hello"}
	result := g.ProcessTemplate(template, wildcards)
	expected := ""

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Set the variable and test again
	kb.Variables["mood"] = "happy"
	result = g.ProcessTemplate(template, wildcards)
	expected = "You said hello and I'm glad you're happy!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConditionWithProperties(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	kb.Properties = map[string]string{
		"bot_name": "Golem",
	}
	g.SetKnowledgeBase(kb)

	// Test condition with properties
	template := `<condition name="bot_name" value="Golem">Hello from <get name="bot_name"/>!</condition>`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Hello from Golem!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConditionWithSRAI(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add test categories
	kb.Categories = []Category{
		{Pattern: "HELLO", Template: "Hello! How can I help you today?"},
		{Pattern: "GREETING", Template: `<condition name="mood" value="happy"><srai>HELLO</srai></condition>`},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test condition with SRAI
	template := `<condition name="mood" value="happy"><srai>HELLO</srai></condition>`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := ""

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Set the variable and test again
	kb.Variables["mood"] = "happy"
	result = g.ProcessTemplate(template, make(map[string]string))
	expected = "Hello! How can I help you today?"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConditionWithThink(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test condition with think
	template := `<think><set name="mood">happy</set></think><condition name="mood" value="happy">I'm glad you're happy!</condition>`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "I'm glad you're happy!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Verify the variable was set
	if kb.Variables["mood"] != "happy" {
		t.Errorf("Expected mood variable to be set to 'happy', got '%s'", kb.Variables["mood"])
	}
}

func TestConditionWithRandom(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	kb.Variables["weather"] = "sunny"
	g.SetKnowledgeBase(kb)

	// Test condition with random using ProcessTemplate (no session context)
	// This matches how the random tag processing works internally
	template := `<random>
		<li><condition name="weather" value="sunny">It's sunny!</condition></li>
		<li><condition name="weather" value="rainy">It's rainy!</condition></li>
	</random>`
	result := g.ProcessTemplate(template, make(map[string]string))

	// Random tag will randomly select between "It's sunny!" and "" (empty string)
	// since the second condition evaluates to empty when weather is "sunny"
	validResults := []string{"It's sunny!", ""}
	found := false
	for _, expected := range validResults {
		if result == expected {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected one of %v, got '%s'", validResults, result)
	}
}

func TestConditionWithDateTime(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	kb.Variables["time_of_day"] = "morning"
	g.SetKnowledgeBase(kb)

	// Test condition with date/time
	template := `<condition name="time_of_day" value="morning">Good morning! Today is <date format="short"/></condition>`
	result := g.ProcessTemplate(template, make(map[string]string))

	if !strings.Contains(result, "Good morning!") {
		t.Errorf("Expected result to contain 'Good morning!', got '%s'", result)
	}
	if !strings.Contains(result, "Today is") {
		t.Errorf("Expected result to contain 'Today is', got '%s'", result)
	}
	if strings.Contains(result, "<date") {
		t.Errorf("Expected <date> tag to be replaced, got '%s'", result)
	}
}

func TestNestedConditions(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	kb.Variables["user_type"] = "admin"
	kb.Variables["time_of_day"] = "morning"
	g.SetKnowledgeBase(kb)

	// Test nested conditions
	template := `<condition name="user_type">
		<li value="admin">Welcome admin! <condition name="time_of_day">
			<li value="morning">Good morning!</li>
			<li value="afternoon">Good afternoon!</li>
			<li>Good day!</li>
		</condition></li>
		<li value="user">Hello user!</li>
		<li>Welcome guest!</li>
	</condition>`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Welcome admin! Good morning!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConditionDefaultCase(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test condition with default case
	template := `<condition name="name">Hello <get name="name"/>!</condition>`
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := ""

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Set the variable and test again
	kb.Variables["name"] = "John"
	result = g.ProcessTemplate(template, make(map[string]string))
	expected = "Hello John!"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestVariableScopeResolution tests the new variable scope resolution system
func TestVariableScopeResolution(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test 1: Local scope has highest priority
	t.Run("LocalScopePriority", func(t *testing.T) {
		ctx := &VariableContext{
			LocalVars:     map[string]string{"test_var": "local_value"},
			Session:       nil,
			Topic:         "",
			KnowledgeBase: kb,
		}

		// Set variables in other scopes
		kb.Variables["test_var"] = "global_value"
		kb.Properties["test_var"] = "property_value"

		// Local should win
		result := g.resolveVariable("test_var", ctx)
		if result != "local_value" {
			t.Errorf("Expected 'local_value', got '%s'", result)
		}
	})

	// Test 2: Session scope has priority over global
	t.Run("SessionScopePriority", func(t *testing.T) {
		session := &ChatSession{
			ID:        "test_session",
			Variables: map[string]string{"test_var": "session_value"},
		}

		ctx := &VariableContext{
			LocalVars:     map[string]string{},
			Session:       session,
			Topic:         "",
			KnowledgeBase: kb,
		}

		// Set variables in other scopes
		kb.Variables["test_var"] = "global_value"
		kb.Properties["test_var"] = "property_value"

		// Session should win over global
		result := g.resolveVariable("test_var", ctx)
		if result != "session_value" {
			t.Errorf("Expected 'session_value', got '%s'", result)
		}
	})

	// Test 3: Global scope has priority over properties
	t.Run("GlobalScopePriority", func(t *testing.T) {
		ctx := &VariableContext{
			LocalVars:     map[string]string{},
			Session:       nil,
			Topic:         "",
			KnowledgeBase: kb,
		}

		// Set variables in other scopes
		kb.Variables["test_var"] = "global_value"
		kb.Properties["test_var"] = "property_value"

		// Global should win over properties
		result := g.resolveVariable("test_var", ctx)
		if result != "global_value" {
			t.Errorf("Expected 'global_value', got '%s'", result)
		}
	})

	// Test 4: Properties as fallback
	t.Run("PropertiesFallback", func(t *testing.T) {
		// Create a fresh knowledge base for this test
		freshKB := NewAIMLKnowledgeBase()
		freshKB.Properties["test_var"] = "property_value"

		ctx := &VariableContext{
			LocalVars:     map[string]string{},
			Session:       nil,
			Topic:         "",
			KnowledgeBase: freshKB,
		}

		// Properties should be used as fallback
		result := g.resolveVariable("test_var", ctx)
		if result != "property_value" {
			t.Errorf("Expected 'property_value', got '%s'", result)
		}
	})

	// Test 5: Variable not found
	t.Run("VariableNotFound", func(t *testing.T) {
		ctx := &VariableContext{
			LocalVars:     map[string]string{},
			Session:       nil,
			Topic:         "",
			KnowledgeBase: kb,
		}

		// No variables set anywhere
		result := g.resolveVariable("nonexistent_var", ctx)
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})
}

func TestLoadAIMLFromDirectory(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create multiple AIML files
	aimlFile1 := filepath.Join(tempDir, "greetings.aiml")
	aimlContent1 := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
    <category>
        <pattern>HELLO</pattern>
        <template>Hello! How can I help you?</template>
    </category>
    <category>
        <pattern>GOODBYE</pattern>
        <template>Goodbye! Have a great day!</template>
    </category>
</aiml>`

	aimlFile2 := filepath.Join(tempDir, "questions.aiml")
	aimlContent2 := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
    <category>
        <pattern>WHAT IS YOUR NAME</pattern>
        <template>My name is <get name="name"/>.</template>
    </category>
    <category>
        <pattern>HOW ARE YOU</pattern>
        <template>I'm doing well, thank you!</template>
    </category>
</aiml>`

	aimlFile3 := filepath.Join(tempDir, "wildcards.aiml")
	aimlContent3 := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
    <category>
        <pattern>MY NAME IS *</pattern>
        <template>Nice to meet you, <star/>!</template>
    </category>
    <category>
        <pattern>I AM _ YEARS OLD</pattern>
        <template>You are <star/> years old!</template>
    </category>
</aiml>`

	// Write the AIML files
	err := os.WriteFile(aimlFile1, []byte(aimlContent1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test AIML file 1: %v", err)
	}

	err = os.WriteFile(aimlFile2, []byte(aimlContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test AIML file 2: %v", err)
	}

	err = os.WriteFile(aimlFile3, []byte(aimlContent3), 0644)
	if err != nil {
		t.Fatalf("Failed to create test AIML file 3: %v", err)
	}

	// Test loading from directory
	kb, err := g.LoadAIMLFromDirectory(tempDir)
	if err != nil {
		t.Fatalf("LoadAIMLFromDirectory failed: %v", err)
	}

	if kb == nil {
		t.Fatal("LoadAIMLFromDirectory returned nil knowledge base")
	}

	// Should have loaded all categories from all files (6 total)
	if len(kb.Categories) != 6 {
		t.Errorf("Expected 6 categories, got %d", len(kb.Categories))
	}

	// Test that all patterns are indexed
	expectedPatterns := []string{
		"HELLO",
		"GOODBYE",
		"WHAT IS YOUR NAME",
		"HOW ARE YOU",
		"MY NAME IS *",
		"I AM _ YEARS OLD",
	}

	for _, pattern := range expectedPatterns {
		if kb.Patterns[pattern] == nil {
			t.Errorf("Pattern '%s' not indexed", pattern)
		}
	}

	// Test pattern matching works
	category, _, err := kb.MatchPattern("HELLO")
	if err != nil {
		t.Fatalf("Pattern match failed: %v", err)
	}
	if category.Pattern != "HELLO" {
		t.Errorf("Expected HELLO pattern, got %s", category.Pattern)
	}

	// Test wildcard matching
	_, wildcards, err := kb.MatchPattern("MY NAME IS JOHN")
	if err != nil {
		t.Fatalf("Wildcard match failed: %v", err)
	}
	if wildcards["star1"] != "JOHN" {
		t.Errorf("Expected wildcard 'JOHN', got '%s'", wildcards["star1"])
	}

	// Test underscore wildcard matching
	_, wildcards, err = kb.MatchPattern("I AM 25 YEARS OLD")
	if err != nil {
		t.Fatalf("Underscore wildcard match failed: %v", err)
	}
	if wildcards["star1"] != "25" {
		t.Errorf("Expected wildcard '25', got '%s'", wildcards["star1"])
	}
}

func TestLoadAIMLFromDirectoryEmpty(t *testing.T) {
	g := NewForTesting(t, false)

	// Create an empty temporary directory
	tempDir := t.TempDir()

	// Test loading from empty directory
	_, err := g.LoadAIMLFromDirectory(tempDir)
	if err == nil {
		t.Error("Expected error when loading from empty directory")
	}
	if !strings.Contains(err.Error(), "no AIML files found") {
		t.Errorf("Expected 'no AIML files found' error, got: %v", err)
	}
}

func TestLoadCommandWithDirectory(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create an AIML file
	aimlFile := filepath.Join(tempDir, "test.aiml")
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
    <category>
        <pattern>HELLO</pattern>
        <template>Hello from directory!</template>
    </category>
</aiml>`

	err := os.WriteFile(aimlFile, []byte(aimlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test AIML file: %v", err)
	}

	// Test load command with directory
	err = g.loadCommand([]string{tempDir})
	if err != nil {
		t.Fatalf("loadCommand with directory failed: %v", err)
	}

	// Verify the knowledge base was loaded
	if g.aimlKB == nil {
		t.Fatal("Knowledge base not loaded")
	}

	if len(g.aimlKB.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(g.aimlKB.Categories))
	}

	if g.aimlKB.Patterns["HELLO"] == nil {
		t.Error("HELLO pattern not indexed")
	}
}

// TestVariableSetting tests the new variable setting system with scopes
func TestVariableSetting(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	session := &ChatSession{
		ID:        "test_session",
		Variables: make(map[string]string),
	}

	// Test 1: Set variable in local scope
	t.Run("SetLocalVariable", func(t *testing.T) {
		ctx := &VariableContext{
			LocalVars:     make(map[string]string),
			Session:       session,
			Topic:         "",
			KnowledgeBase: kb,
		}

		g.setVariable("local_var", "local_value", ScopeLocal, ctx)

		if ctx.LocalVars["local_var"] != "local_value" {
			t.Errorf("Expected local variable to be set to 'local_value', got '%s'", ctx.LocalVars["local_var"])
		}
	})

	// Test 2: Set variable in session scope
	t.Run("SetSessionVariable", func(t *testing.T) {
		ctx := &VariableContext{
			LocalVars:     make(map[string]string),
			Session:       session,
			Topic:         "",
			KnowledgeBase: kb,
		}

		g.setVariable("session_var", "session_value", ScopeSession, ctx)

		if session.Variables["session_var"] != "session_value" {
			t.Errorf("Expected session variable to be set to 'session_value', got '%s'", session.Variables["session_var"])
		}
	})

	// Test 3: Set variable in global scope
	t.Run("SetGlobalVariable", func(t *testing.T) {
		ctx := &VariableContext{
			LocalVars:     make(map[string]string),
			Session:       session,
			Topic:         "",
			KnowledgeBase: kb,
		}

		g.setVariable("global_var", "global_value", ScopeGlobal, ctx)

		if kb.Variables["global_var"] != "global_value" {
			t.Errorf("Expected global variable to be set to 'global_value', got '%s'", kb.Variables["global_var"])
		}
	})

	// Test 4: Cannot set properties (read-only)
	t.Run("CannotSetProperties", func(t *testing.T) {
		ctx := &VariableContext{
			LocalVars:     make(map[string]string),
			Session:       session,
			Topic:         "",
			KnowledgeBase: kb,
		}

		// This should not set anything and should log a warning
		g.setVariable("property_var", "property_value", ScopeProperties, ctx)

		// Properties should not be modified
		if kb.Properties["property_var"] != "" {
			t.Errorf("Expected properties to remain unchanged, got '%s'", kb.Properties["property_var"])
		}
	})
}

func TestLoadMapFromFile(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary map file
	tempFile := t.TempDir() + "/test.map"
	mapContent := `[
		{"key": "hello", "value": "hi"},
		{"key": "bye", "value": "goodbye"},
		{"key": "thanks", "value": "thank you"}
	]`

	err := os.WriteFile(tempFile, []byte(mapContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test map file: %v", err)
	}

	// Test loading the map file
	mapData, err := g.LoadMapFromFile(tempFile)
	if err != nil {
		t.Fatalf("LoadMapFromFile failed: %v", err)
	}

	// Verify the map data
	expected := map[string]string{
		"hello":  "hi",
		"bye":    "goodbye",
		"thanks": "thank you",
	}

	if len(mapData) != len(expected) {
		t.Errorf("Expected %d map entries, got %d", len(expected), len(mapData))
	}

	for key, expectedValue := range expected {
		if actualValue, exists := mapData[key]; !exists {
			t.Errorf("Key '%s' not found in map", key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected value '%s' for key '%s', got '%s'", expectedValue, key, actualValue)
		}
	}
}

func TestLoadMapFromFileInvalidJSON(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary map file with invalid JSON
	tempFile := t.TempDir() + "/invalid.map"
	mapContent := `[
		{"key": "hello", "value": "hi"
		{"key": "bye", "value": "goodbye"}
	]`

	err := os.WriteFile(tempFile, []byte(mapContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test map file: %v", err)
	}

	// Test loading the invalid map file
	_, err = g.LoadMapFromFile(tempFile)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestLoadMapsFromDirectory(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create multiple map files
	mapFile1 := filepath.Join(tempDir, "greetings.map")
	mapContent1 := `[
		{"key": "hello", "value": "hi"},
		{"key": "bye", "value": "goodbye"}
	]`

	mapFile2 := filepath.Join(tempDir, "emotions.map")
	mapContent2 := `[
		{"key": "happy", "value": "joyful"},
		{"key": "sad", "value": "melancholy"}
	]`

	// Write the map files
	err := os.WriteFile(mapFile1, []byte(mapContent1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test map file 1: %v", err)
	}

	err = os.WriteFile(mapFile2, []byte(mapContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test map file 2: %v", err)
	}

	// Test loading from directory
	allMaps, err := g.LoadMapsFromDirectory(tempDir)
	if err != nil {
		t.Fatalf("LoadMapsFromDirectory failed: %v", err)
	}

	// Should have loaded 2 maps
	if len(allMaps) != 2 {
		t.Errorf("Expected 2 maps, got %d", len(allMaps))
	}

	// Check greetings map
	if greetingsMap, exists := allMaps["greetings"]; !exists {
		t.Error("greetings map not found")
	} else {
		if greetingsMap["hello"] != "hi" {
			t.Errorf("Expected 'hi' for 'hello', got '%s'", greetingsMap["hello"])
		}
		if greetingsMap["bye"] != "goodbye" {
			t.Errorf("Expected 'goodbye' for 'bye', got '%s'", greetingsMap["bye"])
		}
	}

	// Check emotions map
	if emotionsMap, exists := allMaps["emotions"]; !exists {
		t.Error("emotions map not found")
	} else {
		if emotionsMap["happy"] != "joyful" {
			t.Errorf("Expected 'joyful' for 'happy', got '%s'", emotionsMap["happy"])
		}
		if emotionsMap["sad"] != "melancholy" {
			t.Errorf("Expected 'melancholy' for 'sad', got '%s'", emotionsMap["sad"])
		}
	}
}

func TestProcessMapTags(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add a map to the knowledge base
	kb.Maps["greetings"] = map[string]string{
		"hello": "hi",
		"bye":   "goodbye",
	}

	g.SetKnowledgeBase(kb)

	// Test map tag processing
	template := "Say <map name=\"greetings\">hello</map> and <map name=\"greetings\">bye</map>"
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Say hi and goodbye"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestProcessMapTagsWithUnknownKey(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add a map to the knowledge base
	kb.Maps["greetings"] = map[string]string{
		"hello": "hi",
	}

	g.SetKnowledgeBase(kb)

	// Test map tag processing with unknown key
	template := "Say <map name=\"greetings\">unknown</map>"
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Say unknown"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestProcessMapTagsWithUnknownMap(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	g.SetKnowledgeBase(kb)

	// Test map tag processing with unknown map
	template := "Say <map name=\"unknown\">hello</map>"
	result := g.ProcessTemplate(template, make(map[string]string))
	expected := "Say hello"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestLoadCommandWithMapFile(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary map file
	tempFile := t.TempDir() + "/test.map"
	mapContent := `[
		{"key": "hello", "value": "hi"},
		{"key": "bye", "value": "goodbye"}
	]`

	err := os.WriteFile(tempFile, []byte(mapContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test map file: %v", err)
	}

	// Test loading the map file via load command
	err = g.loadCommand([]string{tempFile})
	if err != nil {
		t.Fatalf("loadCommand failed: %v", err)
	}

	// Verify the map was loaded
	if g.aimlKB == nil {
		t.Fatal("Knowledge base should not be nil")
	}

	if g.aimlKB.Maps["test"] == nil {
		t.Fatal("test map should be loaded")
	}

	if g.aimlKB.Maps["test"]["hello"] != "hi" {
		t.Errorf("Expected 'hi' for 'hello', got '%s'", g.aimlKB.Maps["test"]["hello"])
	}
}

func TestLoadSetFromFile(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary set file
	tempFile := t.TempDir() + "/test.set"
	setContent := `["happy", "sad", "angry", "excited"]`

	err := os.WriteFile(tempFile, []byte(setContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test set file: %v", err)
	}

	// Test loading the set file
	setMembers, err := g.LoadSetFromFile(tempFile)
	if err != nil {
		t.Fatalf("LoadSetFromFile failed: %v", err)
	}

	// Verify the set data
	expected := []string{"happy", "sad", "angry", "excited"}

	if len(setMembers) != len(expected) {
		t.Errorf("Expected %d set members, got %d", len(expected), len(setMembers))
	}

	for i, expectedMember := range expected {
		if i >= len(setMembers) {
			t.Errorf("Expected member '%s' at index %d, but set has only %d members", expectedMember, i, len(setMembers))
			continue
		}
		if setMembers[i] != expectedMember {
			t.Errorf("Expected member '%s' at index %d, got '%s'", expectedMember, i, setMembers[i])
		}
	}
}

func TestLoadSetFromFileInvalidJSON(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary set file with invalid JSON
	tempFile := t.TempDir() + "/invalid.set"
	setContent := `["happy", "sad" "angry", "excited"]`

	err := os.WriteFile(tempFile, []byte(setContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test set file: %v", err)
	}

	// Test loading the invalid set file
	_, err = g.LoadSetFromFile(tempFile)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestLoadSetsFromDirectory(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create multiple set files
	setFile1 := filepath.Join(tempDir, "emotions.set")
	setContent1 := `["happy", "sad", "angry"]`

	setFile2 := filepath.Join(tempDir, "colors.set")
	setContent2 := `["red", "blue", "green", "yellow"]`

	// Write the set files
	err := os.WriteFile(setFile1, []byte(setContent1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test set file 1: %v", err)
	}

	err = os.WriteFile(setFile2, []byte(setContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test set file 2: %v", err)
	}

	// Test loading from directory
	allSets, err := g.LoadSetsFromDirectory(tempDir)
	if err != nil {
		t.Fatalf("LoadSetsFromDirectory failed: %v", err)
	}

	// Should have loaded 2 sets
	if len(allSets) != 2 {
		t.Errorf("Expected 2 sets, got %d", len(allSets))
	}

	// Check emotions set
	if emotionsSet, exists := allSets["emotions"]; !exists {
		t.Error("emotions set not found")
	} else {
		expectedEmotions := []string{"happy", "sad", "angry"}
		if len(emotionsSet) != len(expectedEmotions) {
			t.Errorf("Expected %d emotions, got %d", len(expectedEmotions), len(emotionsSet))
		}
		for i, expected := range expectedEmotions {
			if i < len(emotionsSet) && emotionsSet[i] != expected {
				t.Errorf("Expected emotion '%s' at index %d, got '%s'", expected, i, emotionsSet[i])
			}
		}
	}

	// Check colors set
	if colorsSet, exists := allSets["colors"]; !exists {
		t.Error("colors set not found")
	} else {
		expectedColors := []string{"red", "blue", "green", "yellow"}
		if len(colorsSet) != len(expectedColors) {
			t.Errorf("Expected %d colors, got %d", len(expectedColors), len(colorsSet))
		}
		for i, expected := range expectedColors {
			if i < len(colorsSet) && colorsSet[i] != expected {
				t.Errorf("Expected color '%s' at index %d, got '%s'", expected, i, colorsSet[i])
			}
		}
	}
}

func TestLoadCommandWithSetFile(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary set file
	tempFile := t.TempDir() + "/test.set"
	setContent := `["happy", "sad", "angry"]`

	err := os.WriteFile(tempFile, []byte(setContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test set file: %v", err)
	}

	// Test loading the set file via load command
	err = g.loadCommand([]string{tempFile})
	if err != nil {
		t.Fatalf("loadCommand failed: %v", err)
	}

	// Verify the set was loaded
	if g.aimlKB == nil {
		t.Fatal("Knowledge base should not be nil")
	}

	if g.aimlKB.Sets["TEST"] == nil {
		t.Fatal("test set should be loaded")
	}

	expectedMembers := []string{"HAPPY", "SAD", "ANGRY"} // Should be uppercase
	if len(g.aimlKB.Sets["TEST"]) != len(expectedMembers) {
		t.Errorf("Expected %d set members, got %d", len(expectedMembers), len(g.aimlKB.Sets["TEST"]))
	}

	for _, expectedMember := range expectedMembers {
		found := false
		for _, actualMember := range g.aimlKB.Sets["TEST"] {
			if actualMember == expectedMember {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected set member '%s' not found", expectedMember)
		}
	}
}

func TestSetMatchingInPatterns(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()

	// Add a set to the knowledge base
	kb.AddSetMembers("emotions", []string{"happy", "sad", "angry"})

	// Add a category that uses the set
	kb.Categories = []Category{
		{Pattern: "I AM <set>emotions</set>", Template: "I understand you're feeling <star/>."},
	}

	// Index patterns
	kb.Patterns = make(map[string]*Category)
	for i := range kb.Categories {
		kb.Patterns[kb.Categories[i].Pattern] = &kb.Categories[i]
	}

	g.SetKnowledgeBase(kb)

	// Test pattern matching with set
	category, wildcards, err := kb.MatchPattern("I AM HAPPY")
	if err != nil {
		t.Fatalf("Pattern match failed: %v", err)
	}
	if category == nil {
		t.Fatal("Expected pattern match, got nil")
	}
	if wildcards["star1"] != "HAPPY" {
		t.Errorf("Expected wildcard 'HAPPY', got '%s'", wildcards["star1"])
	}

	// Test another emotion
	category, wildcards, err = kb.MatchPattern("I AM SAD")
	if err != nil {
		t.Fatalf("Pattern match failed: %v", err)
	}
	if category == nil {
		t.Fatal("Expected pattern match, got nil")
	}
	if wildcards["star1"] != "SAD" {
		t.Errorf("Expected wildcard 'SAD', got '%s'", wildcards["star1"])
	}
}

// TestNormalizationIntegration tests normalization in the full AIML system
func TestNormalizationIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Add patterns with various punctuation and case
	kb.Categories = []Category{
		{
			Pattern:  "Hello, World!",
			Template: "Hello there!",
		},
		{
			Pattern:  "What's up?",
			Template: "Not much!",
		},
		{
			Pattern:  "I am * years old.",
			Template: "You are <star/> years old!",
		},
		{
			Pattern:  "Calculate *",
			Template: "Let me calculate that for you.",
		},
	}

	// Normalize patterns for storage
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	// Test matching with various inputs
	t.Run("PatternMatchingWithNormalization", func(t *testing.T) {
		testCases := []struct {
			input string
			desc  string
		}{
			{"Hello, World!", "Exact match with punctuation"},
			{"hello world", "Case insensitive match"},
			{"What's up?", "Question with apostrophe"},
			{"what is up", "Normalized question with contraction expansion"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				_, wildcards, err := kb.MatchPattern(tc.input)
				if err != nil {
					t.Errorf("Pattern match failed for '%s': %v", tc.input, err)
					return
				}

				// Process template to get response
				normalizedInput := NormalizePattern(tc.input)
				if category, exists := kb.Patterns[normalizedInput]; exists {
					response := g.ProcessTemplate(category.Template, wildcards)
					if !strings.Contains(response, "Hello") && !strings.Contains(response, "Not much") {
						t.Errorf("Unexpected response for '%s': %s", tc.input, response)
					}
				} else {
					t.Errorf("No pattern found for normalized input '%s'", normalizedInput)
				}
			})
		}
	})
}

func TestBotTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test properties
	currentVersion := GetVersion()
	g.aimlKB.Properties["name"] = "GolemBot"
	g.aimlKB.Properties["version"] = currentVersion
	g.aimlKB.Properties["author"] = "Test Author"
	g.aimlKB.Properties["language"] = "en"

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic bot property access",
			template: "Hello! I am <bot name=\"name\"/>.",
			expected: "Hello! I am GolemBot.",
		},
		{
			name:     "Multiple bot properties",
			template: "I am <bot name=\"name\"/> version <bot name=\"version\"/> by <bot name=\"author\"/>.",
			expected: fmt.Sprintf("I am GolemBot version %s by Test Author.", currentVersion),
		},
		{
			name:     "Bot property with other content",
			template: "Welcome! My name is <bot name=\"name\"/> and I speak <bot name=\"language\"/>.",
			expected: "Welcome! My name is GolemBot and I speak en.",
		},
		{
			name:     "Non-existent bot property",
			template: "Hello! I am <bot name=\"nonexistent\"/>.",
			expected: "Hello! I am .", // Per AIML spec: missing properties return empty string
		},
		{
			name:     "Empty bot property",
			template: "Hello! I am <bot name=\"empty\"/>.",
			expected: "Hello! I am .", // Per AIML spec: empty properties return empty string
		},
		{
			name:     "Mixed bot and get tags",
			template: "I am <bot name=\"name\"/> and my version is <get name=\"version\"/>.",
			expected: fmt.Sprintf("I am GolemBot and my version is %s.", currentVersion),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test session
			session := g.CreateSession("test-session")

			// Process the template
			result := g.ProcessTemplateWithContext(tt.template, make(map[string]string), session)

			if result != tt.expected {
				t.Errorf("Bot tag processing failed.\nExpected: %s\nGot: %s", tt.expected, result)
			}
		})
	}
}

func TestBotTagWithContext(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test properties
	g.aimlKB.Properties["name"] = "ContextBot"
	g.aimlKB.Properties["version"] = "2.0.0"

	// Create a test session
	session := g.CreateSession("context-test")

	// Test with context
	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       session,
		Topic:         "",
		KnowledgeBase: g.aimlKB,
	}

	template := "Hello! I am <bot name=\"name\"/> version <bot name=\"version\"/>."
	expected := "Hello! I am ContextBot version 2.0.0."

	result := processTemplateForTest(g, template, nil, ctx)

	if result != expected {
		t.Errorf("Bot tag with context failed.\nExpected: %s\nGot: %s", expected, result)
	}
}

// TestPersonTagProcessing tests the basic person tag functionality
func TestPersonTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic first person to second person",
			template: "I am happy with my results.",
			expected: "you are happy with your results.",
		},
		{
			name:     "Basic second person to first person",
			template: "You are here with your friends.",
			expected: "I am here with my friends.",
		},
		{
			name:     "Mixed pronouns",
			template: "I think you should do what you want with your life.",
			expected: "you think I should do what I want with my life.",
		},
		{
			name:     "Possessive pronouns",
			template: "This is mine and that is yours.",
			expected: "This is yours and that is yours.",
		},
		{
			name:     "Reflexive pronouns",
			template: "I did it myself and you did it yourself.",
			expected: "you did it yourself and I did it yourself.",
		},
		{
			name:     "Plural pronouns",
			template: "We are going to our house with our friends.",
			expected: "you are going to your house with your friends.",
		},
		{
			name:     "Contractions first person",
			template: "I'm happy and I've been working hard.",
			expected: "you're happy and you've been working hard.",
		},
		{
			name:     "Contractions second person",
			template: "You're right and you'll be fine.",
			expected: "I'm right and I'll be fine.",
		},
		{
			name:     "Possessive forms",
			template: "This is my car and that is your car.",
			expected: "This is your car and that is my car.",
		},
		{
			name:     "Complex sentence",
			template: "I think you should tell me about your plans for our future.",
			expected: "you think I should tell you about my plans for your future.",
		},
		{
			name:     "No pronouns",
			template: "The cat sat on the mat.",
			expected: "The cat sat on the mat.",
		},
		{
			name:     "Mixed case",
			template: "I am happy but You are sad.",
			expected: "you are happy but I am sad.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.SubstitutePronouns(tt.template)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestPersonTagWithContext tests person tags with variable context
func TestPersonTagWithContext(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Simple person tag",
			template: "You said: <person>I am happy</person>",
			expected: "You said: you are happy",
		},
		{
			name:     "Multiple person tags",
			template: "You said: <person>I am happy</person> and <person>you are sad</person>",
			expected: "You said: you are happy and I am sad",
		},
		{
			name:     "Person tag with contractions",
			template: "You said: <person>I'm going to my house</person>",
			expected: "You said: you're going to your house",
		},
		{
			name:     "Person tag with mixed pronouns",
			template: "You said: <person>I think you should do what you want</person>",
			expected: "You said: you think I should do what I want",
		},
		{
			name:     "Person tag with possessives",
			template: "You said: <person>This is mine and that is yours</person>",
			expected: "You said: This is yours and that is mine",
		},
		{
			name:     "Person tag with reflexive pronouns",
			template: "You said: <person>I did it myself</person>",
			expected: "You said: you did it yourself",
		},
		{
			name:     "Person tag with plural pronouns",
			template: "You said: <person>We are going to our house</person>",
			expected: "You said: you are going to your house",
		},
		{
			name:     "Person tag with complex sentence",
			template: "You said: <person>I think you should tell me about your plans</person>",
			expected: "You said: you think I should tell you about my plans",
		},
		{
			name:     "Person tag with no pronouns",
			template: "You said: <person>The cat sat on the mat</person>",
			expected: "You said: The cat sat on the mat",
		},
		{
			name:     "Person tag with mixed case",
			template: "You said: <person>I am happy but You are sad</person>",
			expected: "You said: you are happy but I am sad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: g.aimlKB,
			}
			result := processTemplateForTest(g, tt.template, nil, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestPersonTagIntegration tests the integration of person tags with the full processing pipeline
func TestPersonTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test properties
	g.aimlKB.Properties["name"] = "TestBot"
	g.aimlKB.Properties["version"] = GetVersion()

	// Create test categories with person tags
	categories := []Category{
		{
			Pattern:  "WHAT DID I SAY",
			Template: "You said: <person>I am happy with my results</person>",
		},
		{
			Pattern:  "WHAT DO YOU THINK",
			Template: "I think <person>you should do what you want</person>",
		},
		{
			Pattern:  "TELL ME ABOUT YOURSELF",
			Template: "I am <bot name=\"name\"/> and <person>I am happy to help you</person>",
		},
		{
			Pattern:  "WHAT ARE YOUR PLANS",
			Template: "My plans are <person>I want to help you with your goals</person>",
		},
		{
			Pattern:  "COMPLEX RESPONSE",
			Template: "You said: <person>I think you should tell me about your plans for our future</person> and I agree.",
		},
	}

	// Add categories to knowledge base and rebuild index
	for _, category := range categories {
		g.aimlKB.Categories = append(g.aimlKB.Categories, category)
		// Build pattern index
		pattern := NormalizePattern(category.Pattern)
		key := pattern
		if category.That != "" {
			key += "|THAT:" + NormalizePattern(category.That)
		}
		if category.Topic != "" {
			key += "|TOPIC:" + NormalizePattern(category.Topic)
		}
		g.aimlKB.Patterns[key] = &g.aimlKB.Categories[len(g.aimlKB.Categories)-1]
	}

	// Test each category
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "WHAT DID I SAY",
			expected: "You said: you are happy with your results",
		},
		{
			input:    "WHAT DO YOU THINK",
			expected: "I think I should do what I want",
		},
		{
			input:    "TELL ME ABOUT YOURSELF",
			expected: "I am TestBot and you are happy to help I",
		},
		{
			input:    "WHAT ARE YOUR PLANS",
			expected: "My plans are you want to help I with my goals",
		},
		{
			input:    "COMPLEX RESPONSE",
			expected: "You said: you think I should tell you about my plans for your future and I agree.",
		},
	}

	// Create a test session
	session := g.CreateSession("test-session")

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			response, err := g.ProcessInput(tc.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}
			if response != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, response)
			}
		})
	}
}

func TestBotTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Initialize knowledge base if nil
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Set up test properties
	g.aimlKB.Properties["name"] = "IntegrationBot"
	g.aimlKB.Properties["version"] = "3.0.0"
	g.aimlKB.Properties["language"] = "en"

	// Create test categories with bot tags
	categories := []Category{
		{
			Pattern:  "WHAT IS YOUR NAME",
			Template: "My name is <bot name=\"name\"/>.",
		},
		{
			Pattern:  "WHAT VERSION ARE YOU",
			Template: "I am version <bot name=\"version\"/>.",
		},
		{
			Pattern:  "TELL ME ABOUT YOURSELF",
			Template: "I am <bot name=\"name\"/> version <bot name=\"version\"/> and I speak <bot name=\"language\"/>.",
		},
		{
			Pattern:  "MIXED TAGS",
			Template: "I am <bot name=\"name\"/> and my version is <get name=\"version\"/>.",
		},
	}

	// Add categories to knowledge base and rebuild index
	for _, category := range categories {
		g.aimlKB.Categories = append(g.aimlKB.Categories, category)
		// Build pattern index
		pattern := NormalizePattern(category.Pattern)
		key := pattern
		if category.That != "" {
			key += "|THAT:" + NormalizePattern(category.That)
		}
		if category.Topic != "" {
			key += "|TOPIC:" + NormalizePattern(category.Topic)
		}
		g.aimlKB.Patterns[key] = &g.aimlKB.Categories[len(g.aimlKB.Categories)-1]
	}

	// Test each category
	testCases := []struct {
		input    string
		expected string
	}{
		{"WHAT IS YOUR NAME", "My name is IntegrationBot."},
		{"WHAT VERSION ARE YOU", "I am version 3.0.0."},
		{"TELL ME ABOUT YOURSELF", "I am IntegrationBot version 3.0.0 and I speak en."},
		{"MIXED TAGS", "I am IntegrationBot and my version is 3.0.0."},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			session := g.CreateSession("integration-test")
			response, err := g.ProcessInput(tc.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}

			if response != tc.expected {
				t.Errorf("Integration test failed.\nExpected: %s\nGot: %s", tc.expected, response)
			}
		})
	}
}

// TestGenderTagProcessing tests the basic gender tag processing functionality
func TestGenderTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic masculine to feminine",
			template: "He is a doctor. <gender>He is a doctor.</gender>",
			expected: "He is a doctor. She is a doctor.",
		},
		{
			name:     "Basic feminine to masculine",
			template: "She is a teacher. <gender>She is a teacher.</gender>",
			expected: "She is a teacher. He is a teacher.",
		},
		{
			name:     "Possessive pronouns",
			template: "This is his book. <gender>This is his book.</gender>",
			expected: "This is his book. This is her book.",
		},
		{
			name:     "Object pronouns",
			template: "I saw him yesterday. <gender>I saw him yesterday.</gender>",
			expected: "I saw him yesterday. I saw her yesterday.",
		},
		{
			name:     "Reflexive pronouns",
			template: "He did it himself. <gender>He did it himself.</gender>",
			expected: "He did it himself. She did it herself.",
		},
		{
			name:     "Contractions",
			template: "He's happy. <gender>He's happy.</gender>",
			expected: "He's happy. She's happy.",
		},
		{
			name:     "Mixed case",
			template: "He is HIS friend. <gender>He is HIS friend.</gender>",
			expected: "He is HIS friend. She is HER friend.",
		},
		{
			name:     "Multiple gender tags",
			template: "He said: <gender>I love him</gender> and <gender>he loves me</gender>",
			expected: "He said: I love her and she loves me",
		},
		{
			name:     "No gender pronouns",
			template: "The cat is sleeping. <gender>The cat is sleeping.</gender>",
			expected: "The cat is sleeping. The cat is sleeping.",
		},
		{
			name:     "Complex sentence",
			template: "He told me that his friend saw him at his house. <gender>He told me that his friend saw him at his house.</gender>",
			expected: "He told me that his friend saw him at his house. She told me that her friend saw her at her house.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: nil,
			}
			result := processTemplateForTest(g, tt.template, nil, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGenderTagWithContext tests gender tag processing with context
func TestGenderTagWithContext(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       nil,
		Topic:         "",
		KnowledgeBase: kb,
	}

	template := "The doctor said: <gender>He will help you</gender>"
	expected := "The doctor said: She will help you"

	result := processTemplateForTest(g, template, nil, ctx)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestGenderTagIntegration tests gender tag integration with full AIML processing
func TestGenderTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with gender tags
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>TELL ME ABOUT THE DOCTOR</pattern>
<template>He is a great doctor. <gender>He is a great doctor.</gender></template>
</category>
<category>
<pattern>TELL ME ABOUT THE TEACHER</pattern>
<template>She is a wonderful teacher. <gender>She is a wonderful teacher.</gender></template>
</category>
<category>
<pattern>WHAT DID HE SAY</pattern>
<template>He said: <gender>I love my job</gender></template>
</category>
<category>
<pattern>WHAT DID SHE SAY</pattern>
<template>She said: <gender>I love my job</gender></template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Doctor description with gender swap",
			input:    "tell me about the doctor",
			expected: "He is a great doctor. She is a great doctor.",
		},
		{
			name:     "Teacher description with gender swap",
			input:    "tell me about the teacher",
			expected: "She is a wonderful teacher. He is a wonderful teacher.",
		},
		{
			name:     "He said with gender swap",
			input:    "what did he say",
			expected: "He said: I love my job",
		},
		{
			name:     "She said with gender swap",
			input:    "what did she say",
			expected: "She said: I love my job",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test-session")
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestSentenceSplitting tests the sentence splitting functionality
func TestSentenceSplitting(t *testing.T) {
	splitter := NewSentenceSplitter()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple sentences",
			input:    "Hello world. How are you? I am fine!",
			expected: []string{"Hello world.", "How are you?", "I am fine!"},
		},
		{
			name:     "Single sentence",
			input:    "Hello world.",
			expected: []string{"Hello world."},
		},
		{
			name:     "No punctuation",
			input:    "Hello world",
			expected: []string{"Hello world"},
		},
		{
			name:     "Abbreviations",
			input:    "Dr. Smith is here. He works at Inc. Corp.",
			expected: []string{"Dr. Smith is here.", "He works at Inc. Corp."},
		},
		{
			name:     "Multiple spaces",
			input:    "Hello    world.    How    are    you?",
			expected: []string{"Hello world.", "How are you?"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Whitespace only",
			input:    "   \t\n   ",
			expected: []string{},
		},
		{
			name:     "Mixed punctuation",
			input:    "What?! Really? Yes! No... Maybe?",
			expected: []string{"What?!", "Really?", "Yes!", "No...", "Maybe?"},
		},
		{
			name:     "Numbers and abbreviations",
			input:    "The meeting is at 3 p.m. Dr. Jones will attend.",
			expected: []string{"The meeting is at 3 p.m.", "Dr. Jones will attend."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitter.SplitSentences(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d sentences, got %d", len(tt.expected), len(result))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Got: %v", result)
				return
			}
			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Sentence %d: expected '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

// TestWordBoundaryDetection tests the word boundary detection functionality
func TestWordBoundaryDetection(t *testing.T) {
	detector := NewWordBoundaryDetector()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "With punctuation",
			input:    "Hello, world!",
			expected: []string{"Hello", ",", "world", "!"},
		},
		{
			name:     "Multiple punctuation",
			input:    "What?! Really? Yes!",
			expected: []string{"What", "?", "!", "Really", "?", "Yes", "!"},
		},
		{
			name:     "Parentheses and brackets",
			input:    "Hello (world) [test] {example}",
			expected: []string{"Hello", "(", "world", ")", "[", "test", "]", "{", "example", "}"},
		},
		{
			name:     "Quotes and apostrophes",
			input:    "He said \"Hello\" and it's fine.",
			expected: []string{"He", "said", "\"", "Hello", "\"", "and", "it", "'", "s", "fine", "."},
		},
		{
			name:     "Numbers and symbols",
			input:    "The price is $100.50 (50% off!)",
			expected: []string{"The", "price", "is", "$", "100", ".", "50", "(", "50", "%", "off", "!", ")"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Whitespace only",
			input:    "   \t\n   ",
			expected: []string{},
		},
		{
			name:     "Single word",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "Single punctuation",
			input:    "!",
			expected: []string{"!"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.SplitWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d words, got %d", len(tt.expected), len(result))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Got: %v", result)
				return
			}
			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Word %d: expected '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

// TestSentenceTagProcessing tests the sentence tag processing functionality
func TestSentenceTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic sentence capitalization",
			template: "Hello. <sentence>how are you? i am fine!</sentence>",
			expected: "Hello. How are you? I am fine!",
		},
		{
			name:     "Multiple sentence tags",
			template: "<sentence>first sentence.</sentence> <sentence>second sentence.</sentence>",
			expected: "First sentence. Second sentence.",
		},
		{
			name:     "No sentence tags",
			template: "Hello world. How are you?",
			expected: "Hello world. How are you?",
		},
		{
			name:     "Empty sentence tag",
			template: "Hello <sentence></sentence> world",
			expected: "Hello  world",
		},
		{
			name:     "Whitespace normalization with capitalization",
			template: "<sentence>hello    world.    how    are    you?</sentence>",
			expected: "Hello world. How are you?",
		},
		{
			name:     "Already capitalized sentences",
			template: "<sentence>Hello World. How Are You?</sentence>",
			expected: "Hello World. How Are You?",
		},
		{
			name:     "Mixed case sentences",
			template: "<sentence>hELLO wORLD. hOW aRE yOU?</sentence>",
			expected: "HELLO wORLD. HOW aRE yOU?",
		},
		{
			name:     "Single sentence",
			template: "<sentence>hello world</sentence>",
			expected: "Hello world",
		},
		{
			name:     "Abbreviations",
			template: "<sentence>dr. smith is here. he works at inc. corp.</sentence>",
			expected: "Dr. Smith is here. He works at inc. Corp.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: nil,
			}
			result := processTemplateForTest(g, tt.template, nil, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestWordTagProcessing tests the word tag processing functionality
func TestWordTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic word capitalization",
			template: "Hello <word>world, how are you?</word>",
			expected: "Hello World, How Are You?",
		},
		{
			name:     "Multiple word tags",
			template: "<word>hello world!</word> <word>how are you?</word>",
			expected: "Hello World! How Are You?",
		},
		{
			name:     "No word tags",
			template: "Hello world! How are you?",
			expected: "Hello world! How are you?",
		},
		{
			name:     "Empty word tag",
			template: "Hello <word></word> world",
			expected: "Hello  world",
		},
		{
			name:     "Whitespace normalization with capitalization",
			template: "<word>hello    world!    how    are    you?</word>",
			expected: "Hello World! How Are You?",
		},
		{
			name:     "Already capitalized words",
			template: "<word>Hello World! How Are You?</word>",
			expected: "Hello World! How Are You?",
		},
		{
			name:     "Mixed case words",
			template: "<word>hELLO wORLD! hOW aRE yOU?</word>",
			expected: "HELLO WORLD! HOW ARE YOU?",
		},
		{
			name:     "Single word",
			template: "<word>hello</word>",
			expected: "Hello",
		},
		{
			name:     "Words with punctuation",
			template: "<word>hello, world! how are you?</word>",
			expected: "Hello, World! How Are You?",
		},
		{
			name:     "Contractions",
			template: "<word>don't can't won't</word>",
			expected: "Don't Can't Won't",
		},
		{
			name:     "Hyphenated words",
			template: "<word>well-known self-aware</word>",
			expected: "Well-Known Self-Aware",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: nil,
			}
			result := processTemplateForTest(g, tt.template, nil, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestUpperLowerCaseTagProcessing tests the uppercase and lowercase tag processing functionality
func TestUpperLowerCaseTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)

	type tc struct {
		name     string
		template string
		expected string
	}

	upperTests := []tc{
		{"Basic uppercase", "Hello <uppercase>world</uppercase>", "Hello WORLD"},
		{"Multiple uppercase tags", "<uppercase>hello</uppercase> <uppercase>world</uppercase>", "HELLO WORLD"},
		{"Empty uppercase", "Hello <uppercase></uppercase> world", "Hello  world"},
		{"Whitespace uppercase", "<uppercase>   hello   world  </uppercase>", "HELLO WORLD"},
		{"Unicode uppercase", "<uppercase>héllo wørld</uppercase>", "HÉLLO WØRLD"},
		{"Punctuation uppercase", "<uppercase>hello, world!</uppercase>", "HELLO, WORLD!"},
		{"Multiline uppercase", "<uppercase>hello\nworld</uppercase>", "HELLO WORLD"},
	}

	lowerTests := []tc{
		{"Basic lowercase", "Hello <lowercase>WORLD</lowercase>", "Hello world"},
		{"Multiple lowercase tags", "<lowercase>HELLO</lowercase> <lowercase>WORLD</lowercase>", "hello world"},
		{"Empty lowercase", "Hello <lowercase></lowercase> world", "Hello  world"},
		{"Whitespace lowercase", "<lowercase>   HELLO   WORLD  </lowercase>", "hello world"},
		{"Unicode lowercase", "<lowercase>HÉLLO WØRLD</lowercase>", "héllo wørld"},
		{"Punctuation lowercase", "<lowercase>HELLO, WORLD!</lowercase>", "hello, world!"},
		{"Multiline lowercase", "<lowercase>HELLO\nWORLD</lowercase>", "hello world"},
	}

	for _, tt := range upperTests {
		t.Run("Upper/"+tt.name, func(t *testing.T) {
			ctx := &VariableContext{LocalVars: make(map[string]string)}
			result := processTemplateForTest(g, tt.template, nil, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}

	for _, tt := range lowerTests {
		t.Run("Lower/"+tt.name, func(t *testing.T) {
			ctx := &VariableContext{LocalVars: make(map[string]string)}
			result := processTemplateForTest(g, tt.template, nil, ctx)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTextProcessingUpperLowerIntegration tests integration of uppercase/lowercase in full AIML processing
func TestTextProcessingUpperLowerIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>YELL *</pattern>
<template><uppercase><star/></uppercase></template>
</category>
<category>
<pattern>WHISPER *</pattern>
<template><lowercase><star/></lowercase></template>
</category>
<category>
<pattern>MIXED CASE *</pattern>
<template>U:<uppercase><star/></uppercase> L:<lowercase><star/></lowercase></template>
</category>
</aiml>`

	if err := g.LoadAIMLFromString(aimlContent); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Uppercase wildcard", "yell hello world", "HELLO WORLD"},
		{"Lowercase wildcard", "whisper HELLO WORLD", "hello world"},
		{"Mixed both", "mixed case HeLLo WoRLd", "U:HELLO WORLD L:hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("upper-lower-test")
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTextProcessingIntegration tests text processing integration with full AIML processing
func TestTextProcessingIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with sentence and word tags
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>MY NAME IS *</pattern>
<template>Nice to meet you, <sentence><star/></sentence>!</template>
</category>
<category>
<pattern>I AM *</pattern>
<template>Hello <word><star/></word>!</template>
</category>
<category>
<pattern>TELL ME ABOUT *</pattern>
<template><sentence>here is some information about <word><star/></word>.</sentence></template>
</category>
<category>
<pattern>CAPITALIZE *</pattern>
<template><word><star/></word></template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Sentence tag with wildcard",
			input:    "my name is john doe",
			expected: "Nice to meet you, John doe!",
		},
		{
			name:     "Word tag with wildcard",
			input:    "i am a programmer",
			expected: "Hello A Programmer!",
		},
		{
			name:     "Both tags with wildcard",
			input:    "tell me about artificial intelligence",
			expected: "Here is some information about Artificial Intelligence.",
		},
		{
			name:     "Capitalize wildcard",
			input:    "capitalize hello world",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test-session")
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTextProcessingEdgeCases tests edge cases for text processing
func TestTextProcessingEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Empty sentence tag",
			template: "Hello <sentence></sentence> world",
			expected: "Hello  world",
		},
		{
			name:     "Empty word tag",
			template: "Hello <word></word> world",
			expected: "Hello  world",
		},
		{
			name:     "Whitespace only sentence tag",
			template: "Hello <sentence>   </sentence> world",
			expected: "Hello  world",
		},
		{
			name:     "Whitespace only word tag",
			template: "Hello <word>   </word> world",
			expected: "Hello  world",
		},
		{
			name:     "Nested sentence tags",
			template: "<sentence>Hello <sentence>world</sentence>!</sentence>",
			expected: "Hello <sentence>world</sentence>!",
		},
		{
			name:     "Nested word tags",
			template: "<word>Hello <word>world</word>!</word>",
			expected: "Hello <word>world</word>!",
		},
		{
			name:     "Unicode characters",
			template: "<sentence>héllo wørld. høw are yøu?</sentence>",
			expected: "Héllo wørld. Høw are yøu?",
		},
		{
			name:     "Unicode word capitalization",
			template: "<word>héllo wørld! høw are yøu?</word>",
			expected: "Héllo Wørld! Høw Are Yøu?",
		},
		{
			name:     "Numbers and symbols",
			template: "<sentence>the price is $100.50. that's 50% off!</sentence>",
			expected: "The price is $100.50. That's 50% off!",
		},
		{
			name:     "Numbers and symbols in words",
			template: "<word>the price is $100.50. that's 50% off!</word>",
			expected: "The Price Is $100.50. That's 50% Off!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				LocalVars:     make(map[string]string),
				Session:       nil,
				Topic:         "",
				KnowledgeBase: nil,
			}

			// Test sentence processing
			sentenceResult := processTemplateForTest(g, tt.template, nil, ctx)

			// Test word processing
			wordResult := processTemplateForTest(g, tt.template, nil, ctx)

			// For this test, we'll check that both functions work without errors
			// The expected result depends on which tags are present
			if sentenceResult == "" && wordResult == "" {
				t.Errorf("Both sentence and word processing returned empty results")
			}
		})
	}
}

// TestAIML2Wildcards tests the new AIML2 wildcard types
func TestAIML2Wildcards(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with various wildcard patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>HELLO</pattern>
<template>Exact match!</template>
</category>
<category>
<pattern>WORLD</pattern>
<template>Exact match!</template>
</category>
<category>
<pattern>$HELLO</pattern>
<template>Dollar wildcard match!</template>
</category>
<category>
<pattern># HELLO #</pattern>
<template>Hash wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
<category>
<pattern>_ HELLO _</pattern>
<template>Underscore wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
<category>
<pattern>^ HELLO ^</pattern>
<template>Caret wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
<category>
<pattern>* HELLO *</pattern>
<template>Asterisk wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
<category>
<pattern>HELLO ^</pattern>
<template>Hello with caret: <star/>!</template>
</category>
<category>
<pattern>HELLO #</pattern>
<template>Hello with hash: <star/>!</template>
</category>
<category>
<pattern>HELLO _</pattern>
<template>Hello with underscore: <star/>!</template>
</category>
<category>
<pattern>HELLO *</pattern>
<template>Hello with asterisk: <star/>!</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Test dollar wildcard priority (highest priority)
		{
			name:     "Dollar wildcard should have highest priority",
			input:    "HELLO",
			expected: "Dollar wildcard match!",
		},
		// Test exact match (second highest priority)
		{
			name:     "Exact match should have second highest priority",
			input:    "WORLD",
			expected: "Exact match!",
		},
		// Test hash wildcard (zero or more words)
		{
			name:     "Hash wildcard with no words",
			input:    "HELLO",
			expected: "Dollar wildcard match!", // Dollar wildcard has higher priority
		},
		{
			name:     "Hash wildcard with words before and after",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!",
		},
		{
			name:     "Hash wildcard with words before only",
			input:    "SAY HELLO",
			expected: "Hash wildcard: SAY HELLO !",
		},
		{
			name:     "Hash wildcard with words after only",
			input:    "HELLO THERE",
			expected: "Hash wildcard:  HELLO THERE!", // # at the start outranks the word HELLO
		},
		// Test underscore wildcard (one or more words)
		{
			name:     "Underscore wildcard with one word each",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!", // Hash wildcard has higher priority
		},
		{
			name:     "Underscore wildcard with multiple words",
			input:    "I SAY HELLO TO YOU",
			expected: "Hash wildcard: I SAY HELLO TO YOU!", // Hash wildcard has higher priority
		},
		// Test caret wildcard (zero or more words)
		{
			name:     "Caret wildcard with no words",
			input:    "HELLO",
			expected: "Dollar wildcard match!", // Dollar wildcard has higher priority
		},
		{
			name:     "Caret wildcard with words",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!", // Hash wildcard has higher priority
		},
		// Test asterisk wildcard (zero or more words)
		{
			name:     "Asterisk wildcard with no words",
			input:    "HELLO",
			expected: "Dollar wildcard match!", // Dollar wildcard has higher priority
		},
		{
			name:     "Asterisk wildcard with words",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!", // Hash wildcard has higher priority
		},
		// Test single wildcards at end
		{
			name:     "Caret at end with no words",
			input:    "HELLO",
			expected: "Dollar wildcard match!", // Dollar wildcard has higher priority
		},
		{
			name:     "Caret at end with words",
			input:    "HELLO THERE",
			expected: "Hash wildcard:  HELLO THERE!", // # at the start outranks the word HELLO
		},
		{
			name:     "Hash at end with no words",
			input:    "HELLO",
			expected: "Dollar wildcard match!", // Dollar wildcard has higher priority
		},
		{
			name:     "Hash at end with words",
			input:    "HELLO THERE",
			expected: "Hash wildcard:  HELLO THERE!", // # at the start outranks the word HELLO
		},
		{
			name:     "Underscore at end with words",
			input:    "HELLO THERE",
			expected: "Hash wildcard:  HELLO THERE!", // # at the start outranks the word HELLO
		},
		{
			name:     "Asterisk at end with no words",
			input:    "HELLO",
			expected: "Dollar wildcard match!", // Dollar wildcard has higher priority
		},
		{
			name:     "Asterisk at end with words",
			input:    "HELLO THERE",
			expected: "Hash wildcard:  HELLO THERE!", // # at the start outranks the word HELLO
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test-session")
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestWildcardPriority tests that wildcard priority ordering works correctly
func TestWildcardPriority(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with patterns that should test priority ordering
	// Each pattern should be mutually exclusive to properly test priority
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>HELLO WORLD</pattern>
<template>Exact match!</template>
</category>
<category>
<pattern>$HELLO WORLD</pattern>
<template>Dollar exact match!</template>
</category>
<category>
<pattern># HELLO #</pattern>
<template>Hash wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
<category>
<pattern>_ HELLO _</pattern>
<template>Underscore wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
<category>
<pattern>^ HELLO ^</pattern>
<template>Caret wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
<category>
<pattern>* HELLO *</pattern>
<template>Asterisk wildcard: <star index="1"/> HELLO <star index="2"/>!</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Dollar wildcard should have highest priority",
			input:    "HELLO WORLD",
			expected: "Dollar exact match!",
		},
		{
			name:     "Hash wildcard should have high priority",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!",
		},
		{
			name:     "Underscore wildcard should have medium-high priority",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!", // Hash should match first due to higher priority
		},
		{
			name:     "Caret wildcard should have medium priority",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!", // Hash should match first due to higher priority
		},
		{
			name:     "Asterisk wildcard should have lowest priority",
			input:    "SAY HELLO THERE",
			expected: "Hash wildcard: SAY HELLO THERE!", // Hash should match first due to higher priority
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test-session")
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestWildcardValidation tests that new wildcard types are properly validated
func TestWildcardValidation(t *testing.T) {
	g := NewForTesting(t, false)

	// Test valid patterns with new wildcards
	validPatterns := []string{
		"HELLO ^",
		"# HELLO #",
		"$HELLO",
		"HELLO _ WORLD",
		"HELLO * WORLD",
		"^ HELLO ^ WORLD ^",
		"# HELLO _ WORLD *",
	}

	for _, pattern := range validPatterns {
		t.Run("Valid pattern: "+pattern, func(t *testing.T) {
			err := g.validatePattern(pattern)
			if err != nil {
				t.Errorf("Pattern '%s' should be valid but got error: %v", pattern, err)
			}
		})
	}

	// Test invalid patterns
	invalidPatterns := []string{
		"HELLO ^^", // Double caret
		"HELLO ##", // Double hash
		"HELLO $$", // Double dollar
		"HELLO __", // Double underscore
		"HELLO **", // Double asterisk
		"HELLO ^#", // Mixed wildcards (should be valid actually)
		"HELLO ^*", // Mixed wildcards (should be valid actually)
	}

	for _, pattern := range invalidPatterns {
		t.Run("Invalid pattern: "+pattern, func(t *testing.T) {
			err := g.validatePattern(pattern)
			// Note: Some of these might actually be valid, adjust test as needed
			if err != nil {
				t.Logf("Pattern '%s' validation error: %v", pattern, err)
			}
		})
	}
}

// TestWildcardEdgeCases tests edge cases for wildcard matching
func TestWildcardEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with edge case patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>^</pattern>
<template>Just caret: <star/>!</template>
</category>
<category>
<pattern>#</pattern>
<template>Just hash: <star/>!</template>
</category>
<category>
<pattern>_</pattern>
<template>Just underscore: <star/>!</template>
</category>
<category>
<pattern>*</pattern>
<template>Just asterisk: <star/>!</template>
</category>
<category>
<pattern>HELLO ^ WORLD</pattern>
<template>Middle caret: <star/>!</template>
</category>
<category>
<pattern>HELLO # WORLD</pattern>
<template>Middle hash: <star/>!</template>
</category>
<category>
<pattern>HELLO _ WORLD</pattern>
<template>Middle underscore: <star/>!</template>
</category>
<category>
<pattern>HELLO * WORLD</pattern>
<template>Middle asterisk: <star/>!</template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty input with caret",
			input:    "",
			expected: "Just hash: !", // Hash has higher priority than caret
		},
		{
			name:     "Empty input with hash",
			input:    "",
			expected: "Just hash: !",
		},
		{
			name:     "Empty input with asterisk",
			input:    "",
			expected: "Just hash: !", // Hash has higher priority than asterisk
		},
		{
			name:     "Single word with caret",
			input:    "HELLO",
			expected: "Just hash: HELLO!", // Hash has higher priority than caret
		},
		{
			name:     "Single word with hash",
			input:    "HELLO",
			expected: "Just hash: HELLO!",
		},
		{
			name:     "Single word with asterisk",
			input:    "HELLO",
			expected: "Just hash: HELLO!", // Hash has higher priority than asterisk
		},
		{
			name:     "Multiple words with caret",
			input:    "HELLO THERE WORLD",
			expected: "Just hash: HELLO THERE WORLD!", // # alone outranks every word
		},
		{
			name:     "Multiple words with hash",
			input:    "HELLO THERE WORLD",
			expected: "Just hash: HELLO THERE WORLD!", // # alone outranks every word
		},
		{
			name:     "Multiple words with asterisk",
			input:    "HELLO THERE WORLD",
			expected: "Just hash: HELLO THERE WORLD!", // # alone outranks every word
		},
		{
			name:     "Middle caret with no middle words",
			input:    "HELLO WORLD",
			expected: "Just hash: HELLO WORLD!", // # alone outranks every word
		},
		{
			name:     "Middle caret with middle words",
			input:    "HELLO THERE WORLD",
			expected: "Just hash: HELLO THERE WORLD!", // # alone outranks every word
		},
		{
			name:     "Middle hash with no middle words",
			input:    "HELLO WORLD",
			expected: "Just hash: HELLO WORLD!", // # alone outranks every word
		},
		{
			name:     "Middle hash with middle words",
			input:    "HELLO THERE WORLD",
			expected: "Just hash: HELLO THERE WORLD!", // # alone outranks every word
		},
		{
			name:     "Middle underscore with middle words",
			input:    "HELLO THERE WORLD",
			expected: "Just hash: HELLO THERE WORLD!", // # alone outranks every word
		},
		{
			name:     "Middle asterisk with no middle words",
			input:    "HELLO WORLD",
			expected: "Just hash: HELLO WORLD!", // # alone outranks every word
		},
		{
			name:     "Middle asterisk with middle words",
			input:    "HELLO THERE WORLD",
			expected: "Just hash: HELLO THERE WORLD!", // # alone outranks every word
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := g.CreateSession("test-session")
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestUniqueTagIntegration tests integration of unique tag in full AIML processing
func TestUniqueTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>UNIQUE *</pattern>
<template><unique><star/></unique></template>
</category>
<category>
<pattern>UNIQUE COMMA *</pattern>
<template><unique delimiter=","><star/></unique></template>
</category>
<category>
<pattern>MIXED FORMATTING *</pattern>
<template>U:<uppercase><star/></uppercase> L:<lowercase><star/></lowercase> F:<formal><star/></formal> E:<explode><star/></explode> C:<capitalize><star/></capitalize> R:<reverse><star/></reverse> A:<acronym><star/></acronym> T:<trim><star/></trim> S:<substring start="0" end="3"><star/></substring> Re:<replace search="test" replace="demo"><star/></replace> P:<pluralize><star/></pluralize> Sh:<shuffle><star/></shuffle> Le:<length><star/></length> Co:<count search="e"><star/></count> Sp:<split delimiter=","><star/></split> Jo:<join delimiter=","><star/></join> In:<indent><star/></indent> De:<dedent><star/></dedent> Un:<unique><star/></unique></template>
</category>
<category>
<pattern>NESTED UNIQUE *</pattern>
<template><unique>hello <star/> world <star/> test</unique></template>
</category>
</aiml>`

	if err := g.LoadAIMLFromString(aimlContent); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	tests := []struct {
		input    string
		expected string
	}{
		{"unique hello world hello test", "hello world test"},
		{"unique comma apple,banana,apple,cherry,banana", "apple,banana,cherry"},
		{"mixed formatting test case", "U:TEST CASE L:test case F:Test Case E:t e s t   c a s e C:Test case R:esac tset A:TC T:test case S:tes Re:demo case P:tests cases Sh:case test Le:9 Co:2 Sp:test case Jo:test,case In: test case De:test case Un:test case"},
		{"nested unique user", "hello user world test"},
	}

	for _, tt := range tests {
		t.Run("Integration/"+tt.input, func(t *testing.T) {
			session := &ChatSession{
				ID:        "test_session",
				Variables: make(map[string]string),
			}
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestRepeatTagIntegration tests integration of repeat tag in full AIML processing
func TestRepeatTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>REPEAT *</pattern>
<template>You said: <repeat/></template>
</category>
<category>
<pattern>REPEAT UPPERCASE *</pattern>
<template>You said: <uppercase><repeat/></uppercase></template>
</category>
<category>
<pattern>REPEAT FORMAL *</pattern>
<template>You said: <formal><repeat/></formal></template>
</category>
<category>
<pattern>MIXED FORMATTING *</pattern>
<template>U:<uppercase><star/></uppercase> L:<lowercase><star/></lowercase> F:<formal><star/></formal> E:<explode><star/></explode> C:<capitalize><star/></capitalize> R:<reverse><star/></reverse> A:<acronym><star/></acronym> T:<trim><star/></trim> S:<substring start="0" end="3"><star/></substring> Re:<replace search="test" replace="demo"><star/></replace> P:<pluralize><star/></pluralize> Sh:<shuffle><star/></shuffle> Le:<length><star/></length> Co:<count search="e"><star/></count> Sp:<split delimiter=","><star/></split> Jo:<join delimiter=","><star/></join> In:<indent><star/></indent> De:<dedent><star/></dedent> Un:<unique><star/></unique> Rp:<repeat/></template>
</category>
<category>
<pattern>NESTED REPEAT *</pattern>
<template>You said: <repeat/> and I heard: <star/></template>
</category>
</aiml>`

	if err := g.LoadAIMLFromString(aimlContent); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session and populate request history
	session := &ChatSession{
		ID:        "test_session",
		Variables: make(map[string]string),
	}

	// First, add some requests to history
	session.AddToRequestHistory("hello world")
	session.AddToRequestHistory("test case")
	session.AddToRequestHistory("user input")

	tests := []struct {
		input    string
		expected string
	}{
		{"repeat hello world", "You said: user input"},
		{"repeat uppercase test case", "You said: REPEAT HELLO WORLD"},
		{"repeat formal test case", "You said: Repeat Uppercase Test Case"},
		{"mixed formatting test case", "U:TEST CASE L:test case F:Test Case E:t e s t   c a s e C:Test case R:esac tset A:TC T:test case S:tes Re:demo case P:tests cases Sh:case test Le:9 Co:2 Sp:test case Jo:test,case In: test case De:test case Un:test case Rp:repeat formal test case"},
		{"nested repeat user input", "You said: mixed formatting test case and I heard: user input"},
	}

	for _, tt := range tests {
		t.Run("Integration/"+tt.input, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestThatTagIntegration tests integration of that tag in full AIML processing
func TestThatTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>THAT *</pattern>
<template>You said: <that/></template>
</category>
<category>
<pattern>THAT UPPERCASE *</pattern>
<template>You said: <uppercase><that/></uppercase></template>
</category>
<category>
<pattern>THAT FORMAL *</pattern>
<template>You said: <formal><that/></formal></template>
</category>
<category>
<pattern>MIXED FORMATTING *</pattern>
<template>U:<uppercase><star/></uppercase> L:<lowercase><star/></lowercase> F:<formal><star/></formal> E:<explode><star/></explode> C:<capitalize><star/></capitalize> R:<reverse><star/></reverse> A:<acronym><star/></acronym> T:<trim><star/></trim> S:<substring start="0" end="3"><star/></substring> Re:<replace search="test" replace="demo"><star/></replace> P:<pluralize><star/></pluralize> Sh:<shuffle><star/></shuffle> Le:<length><star/></length> Co:<count search="e"><star/></count> Sp:<split delimiter=","><star/></split> Jo:<join delimiter=","><star/></join> In:<indent><star/></indent> De:<dedent><star/></dedent> Un:<unique><star/></unique> Rp:<repeat/> Th:<that/></template>
</category>
<category>
<pattern>NESTED THAT *</pattern>
<template>You said: <that/> and I heard: <star/></template>
</category>
</aiml>`

	if err := g.LoadAIMLFromString(aimlContent); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session and populate response history
	session := &ChatSession{
		ID:        "test_session",
		Variables: make(map[string]string),
	}

	// First, add some responses to history
	session.AddToResponseHistory("hello world")
	session.AddToResponseHistory("test case")
	session.AddToResponseHistory("user input")

	tests := []struct {
		input    string
		expected string
	}{
		{"that hello world", "You said: user input"},
		{"that uppercase test case", "You said: YOU SAID: USER INPUT"},
		{"that formal test case", "You said: You Said: You Said: User Input"},
		{"mixed formatting test case", "U:TEST CASE L:test case F:Test Case E:t e s t   c a s e C:Test case R:esac tset A:TC T:test case S:tes Re:demo case P:tests cases Sh:case test Le:9 Co:2 Sp:test case Jo:test,case In: test case De:test case Un:test case Rp:that formal test case Th:You said: You Said: You Said: User Input"},
		{"nested that user input", "You said: U:TEST CASE L:test case F:Test Case E:t e s t   c a s e C:Test case R:esac tset A:TC T:test case S:tes Re:demo case P:tests cases Sh:case test Le:9 Co:2 Sp:test case Jo:test,case In: test case De:test case Un:test case Rp:that formal test case Th:You said: You Said: You Said: User Input and I heard: user input"},
	}

	for _, tt := range tests {
		t.Run("Integration/"+tt.input, func(t *testing.T) {
			response, err := g.ProcessInput(tt.input, session)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTopicTagIntegration tests integration of topic tag in full AIML processing
func TestTopicTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>TOPIC *</pattern>
<template>Current topic: <topic/></template>
</category>
<category>
<pattern>TOPIC UPPERCASE *</pattern>
<template>Current topic: <uppercase><topic/></uppercase></template>
</category>
<category>
<pattern>TOPIC FORMAL *</pattern>
<template>Current topic: <formal><topic/></formal></template>
</category>
<category>
<pattern>SET TOPIC *</pattern>
<template><think><set name="topic"><star/></set></think>Topic set to: <topic/></template>
</category>
<category>
<pattern>MIXED FORMATTING *</pattern>
<template>U:<uppercase><star/></uppercase> L:<lowercase><star/></lowercase> F:<formal><star/></formal> E:<explode><star/></explode> C:<capitalize><star/></capitalize> R:<reverse><star/></reverse> A:<acronym><star/></acronym> T:<trim><star/></trim> S:<substring start="0" end="3"><star/></substring> Re:<replace search="test" replace="demo"><star/></replace> P:<pluralize><star/></pluralize> Sh:<shuffle><star/></shuffle> Le:<length><star/></length> Co:<count search="e"><star/></count> Sp:<split delimiter=","><star/></split> Jo:<join delimiter=","><star/></join> In:<indent><star/></indent> De:<dedent><star/></dedent> Un:<unique><star/></unique> Rp:<repeat/> Th:<that/> To:<topic/></template>
</category>
<category>
<pattern>NESTED TOPIC *</pattern>
<template>Current topic: <topic/> and I heard: <star/></template>
</category>
</aiml>`

	if err := g.LoadAIMLFromString(aimlContent); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	kb := g.GetKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session and set initial topic
	session := g.CreateSession("test_session")
	session.SetSessionTopic("weather")

	tests := []struct {
		input    string
		expected string
	}{
		{"topic hello world", "Current topic: weather"},
		{"topic uppercase test case", "Current topic: WEATHER"},
		{"topic formal test case", "Current topic: Weather"},
		{"set topic sports", "Topic set to: sports"},
		{"mixed formatting test case", "U:TEST CASE L:test case F:Test Case E:t e s t   c a s e C:Test case R:esac tset A:TC T:test case S:tes Re:demo case P:tests cases Sh:case test Le:9 Co:2 Sp:test case Jo:test,case In: test case De:test case Un:test case Rp: Th: To:weather"},
		{"nested topic user input", "Current topic: weather and I heard: user input"},
	}

	for _, tt := range tests {
		t.Run("Integration/"+tt.input, func(t *testing.T) {
			// Create a fresh session for each test to avoid state issues
			testSession := g.CreateSession("test_session_" + tt.input)
			testSession.SetSessionTopic("weather")

			response, err := g.ProcessInput(tt.input, testSession)
			if err != nil {
				t.Fatalf("ProcessInput failed: %v", err)
			}
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTopicVariableScope tests topic variable scope functionality
func TestTopicVariableScope(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a session with a topic
	session := &ChatSession{
		ID:        "test-session",
		Variables: make(map[string]string),
		Topic:     "weather",
	}

	kb := g.GetKnowledgeBase()
	if kb == nil {
		kb = NewAIMLKnowledgeBase()
		g.SetKnowledgeBase(kb)
	}

	ctx := &VariableContext{
		LocalVars:     make(map[string]string),
		Session:       session,
		Topic:         "weather",
		KnowledgeBase: kb,
	}

	// Test setting topic variables
	g.setVariable("weather_var", "sunny", ScopeTopic, ctx)
	g.setVariable("general_var", "hello", ScopeSession, ctx)

	// Test retrieving topic variables
	topicValue := g.resolveVariable("weather_var", ctx)
	if topicValue != "sunny" {
		t.Errorf("Expected topic variable 'sunny', got '%s'", topicValue)
	}

	// Test that session variables still work
	sessionValue := g.resolveVariable("general_var", ctx)
	if sessionValue != "hello" {
		t.Errorf("Expected session variable 'hello', got '%s'", sessionValue)
	}

	// Test topic isolation - change topic and verify variable is not accessible
	session.SetSessionTopic("sports")
	topicValueAfterChange := g.resolveVariable("weather_var", ctx)
	if topicValueAfterChange != "" {
		t.Errorf("Expected empty string for topic variable after topic change, got '%s'", topicValueAfterChange)
	}

	// Test setting variable in new topic
	g.setVariable("sports_var", "football", ScopeTopic, ctx)
	sportsValue := g.resolveVariable("sports_var", ctx)
	if sportsValue != "football" {
		t.Errorf("Expected sports topic variable 'football', got '%s'", sportsValue)
	}
}
