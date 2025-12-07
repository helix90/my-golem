package golem

import (
	"log"
	"os"
	"testing"
)

func TestSRAIXConfigureFromProperties(t *testing.T) {
	logger := log.New(os.Stdout, "[SRAIX TEST] ", log.LstdFlags)
	sm := NewSRAIXManager(logger, true)

	properties := map[string]string{
		// OpenAI service configuration
		"sraix.openai.baseurl":        "https://api.openai.com/v1/chat/completions",
		"sraix.openai.apikey":         "Bearer sk-test-key-12345",
		"sraix.openai.method":         "POST",
		"sraix.openai.timeout":        "60",
		"sraix.openai.responseformat": "json",
		"sraix.openai.responsepath":   "choices.0.message.content",
		"sraix.openai.fallback":       "AI service temporarily unavailable",
		"sraix.openai.header.Content-Type": "application/json",

		// Weather API service configuration
		"sraix.weather.baseurl":        "https://api.weather.com/v1/forecast",
		"sraix.weather.apikey":         "wx-api-key-67890",
		"sraix.weather.method":         "GET",
		"sraix.weather.timeout":        "30",
		"sraix.weather.responseformat": "json",
		"sraix.weather.responsepath":   "forecast.today",
		"sraix.weather.fallback":       "Weather information unavailable",

		// Custom service with custom headers
		"sraix.custom.baseurl":               "https://custom-api.example.com",
		"sraix.custom.header.X-API-Key":      "custom-key",
		"sraix.custom.header.X-Client-ID":    "client-123",
		"sraix.custom.includewildcards":      "true",

		// Invalid properties (should be ignored/warned)
		"sraix.invalid.":                     "no property name",
		"sraix..baseurl":                     "no service name",
		"not.sraix.property":                 "wrong prefix",
	}

	err := sm.ConfigureFromProperties(properties)
	if err != nil {
		t.Fatalf("ConfigureFromProperties failed: %v", err)
	}

	// Test OpenAI service configuration
	config, exists := sm.GetConfig("openai")
	if !exists {
		t.Error("Expected 'openai' service to be configured")
	}
	if config.BaseURL != "https://api.openai.com/v1/chat/completions" {
		t.Errorf("Expected openai baseURL 'https://api.openai.com/v1/chat/completions', got '%s'", config.BaseURL)
	}
	if config.Method != "POST" {
		t.Errorf("Expected openai method 'POST', got '%s'", config.Method)
	}
	if config.Timeout != 60 {
		t.Errorf("Expected openai timeout 60, got %d", config.Timeout)
	}
	if config.ResponseFormat != "json" {
		t.Errorf("Expected openai responseformat 'json', got '%s'", config.ResponseFormat)
	}
	if config.ResponsePath != "choices.0.message.content" {
		t.Errorf("Expected openai responsepath 'choices.0.message.content', got '%s'", config.ResponsePath)
	}
	if config.FallbackResponse != "AI service temporarily unavailable" {
		t.Errorf("Expected openai fallback 'AI service temporarily unavailable', got '%s'", config.FallbackResponse)
	}
	if config.Headers["Authorization"] != "Bearer sk-test-key-12345" {
		t.Errorf("Expected openai Authorization header 'Bearer sk-test-key-12345', got '%s'", config.Headers["Authorization"])
	}
	if config.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected openai Content-Type header 'application/json', got '%s'", config.Headers["Content-Type"])
	}

	// Test Weather service configuration
	config, exists = sm.GetConfig("weather")
	if !exists {
		t.Error("Expected 'weather' service to be configured")
	}
	if config.BaseURL != "https://api.weather.com/v1/forecast" {
		t.Errorf("Expected weather baseURL 'https://api.weather.com/v1/forecast', got '%s'", config.BaseURL)
	}
	if config.Method != "GET" {
		t.Errorf("Expected weather method 'GET', got '%s'", config.Method)
	}
	if config.Timeout != 30 {
		t.Errorf("Expected weather timeout 30, got %d", config.Timeout)
	}

	// Test Custom service configuration with custom headers
	config, exists = sm.GetConfig("custom")
	if !exists {
		t.Error("Expected 'custom' service to be configured")
	}
	if config.BaseURL != "https://custom-api.example.com" {
		t.Errorf("Expected custom baseURL 'https://custom-api.example.com', got '%s'", config.BaseURL)
	}
	if config.Headers["X-API-Key"] != "custom-key" {
		t.Errorf("Expected custom X-API-Key header 'custom-key', got '%s'", config.Headers["X-API-Key"])
	}
	if config.Headers["X-Client-ID"] != "client-123" {
		t.Errorf("Expected custom X-Client-ID header 'client-123', got '%s'", config.Headers["X-Client-ID"])
	}
	if !config.IncludeWildcards {
		t.Error("Expected custom IncludeWildcards to be true")
	}

	// Verify invalid properties were not configured
	_, exists = sm.GetConfig("invalid")
	if exists {
		t.Error("Did not expect 'invalid' service to be configured")
	}
	_, exists = sm.GetConfig("")
	if exists {
		t.Error("Did not expect empty service name to be configured")
	}
}

func TestSRAIXGolemIntegration(t *testing.T) {
	g := NewForTesting(t, true)

	// Create a knowledge base with SRAIX properties
	kb := &AIMLKnowledgeBase{
		Properties: map[string]string{
			"sraix.testservice.baseurl":        "https://api.test.com",
			"sraix.testservice.apikey":         "test-key",
			"sraix.testservice.method":         "POST",
			"sraix.testservice.timeout":        "45",
			"sraix.testservice.responseformat": "json",
			"sraix.testservice.responsepath":   "result",
			"sraix.testservice.fallback":       "Test service unavailable",
		},
	}

	// Set the knowledge base (should trigger SRAIX configuration)
	g.SetKnowledgeBase(kb)

	// Verify the service was configured
	if g.sraixMgr == nil {
		t.Fatal("SRAIX manager is nil")
	}

	config, exists := g.sraixMgr.GetConfig("testservice")
	if !exists {
		t.Fatal("Expected 'testservice' to be configured from properties")
	}

	if config.BaseURL != "https://api.test.com" {
		t.Errorf("Expected baseURL 'https://api.test.com', got '%s'", config.BaseURL)
	}
	if config.Headers["Authorization"] != "test-key" {
		t.Errorf("Expected Authorization header 'test-key', got '%s'", config.Headers["Authorization"])
	}
	if config.Method != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", config.Method)
	}
	if config.Timeout != 45 {
		t.Errorf("Expected timeout 45, got %d", config.Timeout)
	}
	if config.ResponseFormat != "json" {
		t.Errorf("Expected responseformat 'json', got '%s'", config.ResponseFormat)
	}
	if config.ResponsePath != "result" {
		t.Errorf("Expected responsepath 'result', got '%s'", config.ResponsePath)
	}
	if config.FallbackResponse != "Test service unavailable" {
		t.Errorf("Expected fallback 'Test service unavailable', got '%s'", config.FallbackResponse)
	}
}

func TestSRAIXBuildConfigDefaults(t *testing.T) {
	logger := log.New(os.Stdout, "[SRAIX TEST] ", log.LstdFlags)
	sm := NewSRAIXManager(logger, false)

	// Minimal configuration (only baseurl)
	props := map[string]string{
		"baseurl": "https://minimal-api.example.com",
	}

	config, err := sm.buildConfigFromProperties("minimal", props)
	if err != nil {
		t.Fatalf("buildConfigFromProperties failed: %v", err)
	}

	// Check defaults
	if config.Method != "POST" {
		t.Errorf("Expected default method 'POST', got '%s'", config.Method)
	}
	if config.Timeout != 30 {
		t.Errorf("Expected default timeout 30, got %d", config.Timeout)
	}
	if config.ResponseFormat != "text" {
		t.Errorf("Expected default responseformat 'text', got '%s'", config.ResponseFormat)
	}
}

func TestSRAIXBuildConfigErrors(t *testing.T) {
	logger := log.New(os.Stdout, "[SRAIX TEST] ", log.LstdFlags)
	sm := NewSRAIXManager(logger, false)

	// Missing baseurl
	props := map[string]string{
		"method": "GET",
	}

	_, err := sm.buildConfigFromProperties("nobaseurl", props)
	if err == nil {
		t.Error("Expected error for missing baseurl, got nil")
	}
}
