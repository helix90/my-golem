package golem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestSRAIXEnvironmentVariableSubstitution tests that ${ENV_VAR} placeholders are substituted
func TestSRAIXEnvironmentVariableSubstitution(t *testing.T) {
	// Set test environment variable
	testAPIKey := "test-api-key-12345"
	os.Setenv("TEST_SRAIX_API_KEY", testAPIKey)
	defer os.Unsetenv("TEST_SRAIX_API_KEY")

	// Track the actual URL that was requested
	var requestedURL string

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()

		// Send a test response
		response := map[string]interface{}{
			"result": "success",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create Golem instance
	g := NewForTesting(t, false)

	// Configure SRAIX service with environment variable in URL template
	config := &SRAIXConfig{
		Name:           "test_env_service",
		URLTemplate:    server.URL + "/${TEST_SRAIX_API_KEY}/data",
		Method:         "GET",
		Timeout:        10,
		ResponseFormat: "json",
		ResponsePath:   "result",
	}
	err := g.AddSRAIXConfig(config)
	if err != nil {
		t.Fatalf("Failed to add SRAIX config: %v", err)
	}

	// Create AIML that uses the SRAIX service
	aiml := `
<aiml version="2.0">
    <category>
        <pattern>TEST ENV</pattern>
        <template><sraix service="test_env_service">test input</sraix></template>
    </category>
</aiml>`

	err = g.LoadAIMLFromString(aiml)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Process input
	session := g.CreateSession("test-session")
	response, err := g.ProcessInput("TEST ENV", session)
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}

	// Verify the response
	if response != "success" {
		t.Errorf("Expected response 'success', got '%s'", response)
	}

	// Verify the environment variable was substituted in the URL
	if !strings.Contains(requestedURL, testAPIKey) {
		t.Errorf("Expected URL to contain API key '%s', but got: %s", testAPIKey, requestedURL)
	}

	// Verify the ${TEST_SRAIX_API_KEY} placeholder was replaced
	if strings.Contains(requestedURL, "${TEST_SRAIX_API_KEY}") {
		t.Errorf("Environment variable placeholder was not substituted in URL: %s", requestedURL)
	}

	// Verify the URL structure is correct
	expectedPath := "/" + testAPIKey + "/data"
	if !strings.Contains(requestedURL, expectedPath) {
		t.Errorf("Expected URL path '%s', but got: %s", expectedPath, requestedURL)
	}
}

// TestSRAIXEnvironmentVariableWithSessionVariables tests combining env vars with session vars
func TestSRAIXEnvironmentVariableWithSessionVariables(t *testing.T) {
	// Set test environment variable
	testAPIKey := "weather-api-key-789"
	os.Setenv("TEST_WEATHER_API_KEY", testAPIKey)
	defer os.Unsetenv("TEST_WEATHER_API_KEY")

	// Track the actual URL that was requested
	var requestedURL string

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()

		// Send a test weather response
		response := map[string]interface{}{
			"weather": "sunny",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create Golem instance
	g := NewForTesting(t, false)

	// Configure SRAIX service with both env var and session var placeholders
	config := &SRAIXConfig{
		Name:           "test_weather_service",
		URLTemplate:    server.URL + "/forecast/${TEST_WEATHER_API_KEY}/{lat},{lon}",
		Method:         "GET",
		Timeout:        10,
		ResponseFormat: "json",
		ResponsePath:   "weather",
	}
	err := g.AddSRAIXConfig(config)
	if err != nil {
		t.Fatalf("Failed to add SRAIX config: %v", err)
	}

	// Create AIML that uses the SRAIX service
	aiml := `
<aiml version="2.0">
    <category>
        <pattern>GET WEATHER</pattern>
        <template><sraix service="test_weather_service">weather query</sraix></template>
    </category>
</aiml>`

	err = g.LoadAIMLFromString(aiml)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create session and set location variables
	session := g.CreateSession("test-session")
	session.Variables["latitude"] = "42.3601"
	session.Variables["longitude"] = "-71.0589"

	// Process input
	response, err := g.ProcessInput("GET WEATHER", session)
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}

	// Verify the response
	if response != "sunny" {
		t.Errorf("Expected response 'sunny', got '%s'", response)
	}

	// Verify the environment variable was substituted
	if !strings.Contains(requestedURL, testAPIKey) {
		t.Errorf("Expected URL to contain API key '%s', but got: %s", testAPIKey, requestedURL)
	}

	// Verify the session variables were substituted
	if !strings.Contains(requestedURL, "42.3601") || !strings.Contains(requestedURL, "-71.0589") {
		t.Errorf("Expected URL to contain coordinates, but got: %s", requestedURL)
	}

	// Verify the complete URL structure
	expectedPath := "/forecast/" + testAPIKey + "/42.3601,-71.0589"
	if !strings.Contains(requestedURL, expectedPath) {
		t.Errorf("Expected URL path '%s', but got: %s", expectedPath, requestedURL)
	}
}

// TestSRAIXEnvironmentVariableNotSet tests behavior when env var is not set
func TestSRAIXEnvironmentVariableNotSet(t *testing.T) {
	// Make sure the env var is NOT set
	os.Unsetenv("NONEXISTENT_API_KEY")

	// Track the actual URL that was requested
	var requestedURL string

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Create Golem instance
	g := NewForTesting(t, false)

	// Configure SRAIX service with undefined environment variable
	config := &SRAIXConfig{
		Name:           "test_missing_env",
		URLTemplate:    server.URL + "/${NONEXISTENT_API_KEY}/data",
		Method:         "GET",
		Timeout:        10,
		ResponseFormat: "text",
	}
	err := g.AddSRAIXConfig(config)
	if err != nil {
		t.Fatalf("Failed to add SRAIX config: %v", err)
	}

	// Create AIML
	aiml := `
<aiml version="2.0">
    <category>
        <pattern>TEST MISSING ENV</pattern>
        <template><sraix service="test_missing_env">test</sraix></template>
    </category>
</aiml>`

	err = g.LoadAIMLFromString(aiml)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Process input
	session := g.CreateSession("test-session")
	_, err = g.ProcessInput("TEST MISSING ENV", session)
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}

	// Verify the environment variable placeholder was replaced with empty string
	// This means the URL will have an empty segment, which is valid behavior
	expectedPath := "//data" // Empty API key results in //
	if !strings.Contains(requestedURL, expectedPath) {
		t.Errorf("Expected URL to have empty env var segment, but got: %s", requestedURL)
	}
}
