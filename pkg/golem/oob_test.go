package golem

import (
	"fmt"
	"testing"
)

func TestOOBMessageParsing(t *testing.T) {
	// Test XML-style OOB tags
	oobMsg, isOOB := ParseOOBMessage("Hello <oob>SYSTEM INFO</oob> world")
	if !isOOB {
		t.Error("Expected OOB message to be detected")
	}
	if oobMsg.Type != "SYSTEM" {
		t.Errorf("Expected type 'SYSTEM', got '%s'", oobMsg.Type)
	}
	if oobMsg.Content != "INFO" {
		t.Errorf("Expected content 'INFO', got '%s'", oobMsg.Content)
	}

	// Test bracket-style OOB tags
	oobMsg, isOOB = ParseOOBMessage("Hello [OOB]PROPERTIES GET name[/OOB] world")
	if !isOOB {
		t.Error("Expected OOB message to be detected")
	}
	if oobMsg.Type != "PROPERTIES" {
		t.Errorf("Expected type 'PROPERTIES', got '%s'", oobMsg.Type)
	}
	if oobMsg.Content != "GET name" {
		t.Errorf("Expected content 'GET name', got '%s'", oobMsg.Content)
	}

	// Test case insensitive
	oobMsg, isOOB = ParseOOBMessage("Hello <OOB>system info</OOB> world")
	if !isOOB {
		t.Error("Expected OOB message to be detected")
	}
	if oobMsg.Type != "SYSTEM" {
		t.Errorf("Expected type 'SYSTEM', got '%s'", oobMsg.Type)
	}

	// Test non-OOB message
	_, isOOB = ParseOOBMessage("Hello world")
	if isOOB {
		t.Error("Expected non-OOB message to not be detected")
	}
}

func TestOOBManager(t *testing.T) {
	// Create OOB manager
	oobMgr := NewOOBManager(false, nil)

	// Register test handler
	handler := &TestOOBHandler{
		name:        "TEST",
		description: "Test handler",
	}
	oobMgr.RegisterHandler(handler)

	// Test handler registration
	handlers := oobMgr.ListHandlers()
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(handlers))
	}
	if handlers[0] != "TEST" {
		t.Errorf("Expected handler 'TEST', got '%s'", handlers[0])
	}

	// Test handler retrieval
	retrievedHandler, exists := oobMgr.GetHandler("TEST")
	if !exists {
		t.Error("Expected handler to exist")
	}
	if retrievedHandler.GetName() != "TEST" {
		t.Errorf("Expected handler name 'TEST', got '%s'", retrievedHandler.GetName())
	}

	// Test OOB processing
	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{},
		History:   []string{},
	}

	response, err := oobMgr.ProcessOOB("TEST message", session)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := "Test handler 'TEST' processed: TEST message"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test unknown handler
	_, err = oobMgr.ProcessOOB("UNKNOWN message", session)
	if err == nil {
		t.Error("Expected error for unknown handler")
	}
}

func TestBuiltInOOBHandlers(t *testing.T) {
	// Test SystemInfoHandler
	systemHandler := &SystemInfoHandler{}

	if !systemHandler.CanHandle("SYSTEM INFO") {
		t.Error("Expected SystemInfoHandler to handle 'SYSTEM INFO'")
	}
	if !systemHandler.CanHandle("systeminfo") {
		t.Error("Expected SystemInfoHandler to handle 'systeminfo'")
	}
	if systemHandler.CanHandle("OTHER MESSAGE") {
		t.Error("Expected SystemInfoHandler to not handle 'OTHER MESSAGE'")
	}

	response, err := systemHandler.Process("SYSTEM INFO VERSION", nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expectedVersion := fmt.Sprintf("Golem v%s", GetVersion())
	if response != expectedVersion {
		t.Errorf("Expected '%s', got '%s'", expectedVersion, response)
	}

	// Test SessionInfoHandler
	sessionHandler := &SessionInfoHandler{}

	if !sessionHandler.CanHandle("SESSION INFO") {
		t.Error("Expected SessionInfoHandler to handle 'SESSION INFO'")
	}

	session := &ChatSession{
		ID:        "test_session",
		Variables: map[string]string{"name": "TestUser"},
		History:   []string{"User: hello", "Golem: hi"},
	}

	response, err = sessionHandler.Process("SESSION INFO", session)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !contains(response, "test_session") {
		t.Errorf("Expected response to contain session ID, got '%s'", response)
	}
	if !contains(response, "2") {
		t.Errorf("Expected response to contain message count, got '%s'", response)
	}

	// Test PropertiesHandler
	kb := NewAIMLKnowledgeBase()
	kb.Properties = map[string]string{
		"name":    "Golem",
		"version": GetVersion(),
	}
	propertiesHandler := &PropertiesHandler{aimlKB: kb}

	if !propertiesHandler.CanHandle("PROPERTIES") {
		t.Error("Expected PropertiesHandler to handle 'PROPERTIES'")
	}

	response, err = propertiesHandler.Process("PROPERTIES GET name", nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := "name=Golem"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestOOBIntegration(t *testing.T) {
	// Create Golem instance
	g := NewForTesting(t, false)

	// Load AIML to register properties handler
	kb := NewAIMLKnowledgeBase()
	kb.Properties = map[string]string{
		"name": "Golem",
	}
	g.SetKnowledgeBase(kb)

	// Test OOB command
	err := g.oobCommand([]string{"list"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test OOB test command
	err = g.oobCommand([]string{"test", "SYSTEM INFO"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test OOB register command
	err = g.oobCommand([]string{"register", "CUSTOM", "Custom test handler"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify custom handler was registered
	handlers := g.oobMgr.ListHandlers()
	if len(handlers) < 4 { // system_info, session_info, properties, custom
		t.Errorf("Expected at least 4 handlers, got %d", len(handlers))
	}
}

func TestOOBInChat(t *testing.T) {
	// Create Golem instance
	g := NewForTesting(t, false)

	// Load AIML
	kb := NewAIMLKnowledgeBase()
	kb.Properties = map[string]string{
		"name": "Golem",
	}
	g.SetKnowledgeBase(kb)

	// Test OOB message in chat
	err := g.chatCommand([]string{"<oob>SYSTEM INFO</oob>"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test non-OOB message in chat (should work now since we have AIML loaded)
	err = g.chatCommand([]string{"hello"})
	if err != nil {
		t.Errorf("Expected no error for chat with AIML knowledge base, got %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
