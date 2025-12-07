package golem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTreeProcessorThinkWithSRAIX tests <think> tag containing <sraix> calls
// This tests the exact pattern used in weather.aiml
func TestTreeProcessorThinkWithSRAIX(t *testing.T) {
	// Create mock geocoding server
	geocodeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{
			{
				"lat": "21.3045470",
				"lon": "-157.8556760",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer geocodeServer.Close()

	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Configure SRAIX services for geocoding
	geocodeConfig := &SRAIXConfig{
		Name:           "geocode",
		URLTemplate:    geocodeServer.URL + "?q={input}",
		Method:         "GET",
		Timeout:        10,
		ResponseFormat: "json",
		ResponsePath:   "0.lat",
	}
	g.AddSRAIXConfig(geocodeConfig)

	geocodeLonConfig := &SRAIXConfig{
		Name:           "geocode_lon",
		URLTemplate:    geocodeServer.URL + "?q={input}",
		Method:         "GET",
		Timeout:        10,
		ResponseFormat: "json",
		ResponsePath:   "0.lon",
	}
	g.AddSRAIXConfig(geocodeLonConfig)

	// Test AIML that matches weather.aiml pattern exactly
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>MY LOCATION IS *</pattern>
		<template>
			<think>
				<set var="location"><star/></set>
				<set var="lat"><sraix service="geocode"><get var="location"/></sraix></set>
				<set var="lon"><sraix service="geocode_lon"><get var="location"/></sraix></set>
				<set name="location"><get var="location"/></set>
				<set name="latitude"><get var="lat"/></set>
				<set name="longitude"><get var="lon"/></set>
			</think>
			I've set your location to <get name="location"/> (coordinates: <get name="latitude"/>, <get name="longitude"/>).
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-think-sraix",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	// Test setting location
	response, err := g.ProcessInput("my location is honolulu", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	// The response should NOT contain <think> tags or any content from inside the think tag
	t.Logf("Response: %s", response)

	// Check that response doesn't contain think tag markers
	if containsThinkTag(response) {
		t.Errorf("Response contains <think> tags or think tag content: %s", response)
	}

	// Check that the response has the expected format
	expectedStart := "I've set your location to honolulu"
	if len(response) < len(expectedStart) || response[:len(expectedStart)] != expectedStart {
		t.Errorf("Expected response to start with '%s', got: %s", expectedStart, response)
	}

	// Check that variables were set correctly
	if session.Variables["location"] != "honolulu" {
		t.Errorf("Expected location='honolulu', got '%s'", session.Variables["location"])
	}

	if session.Variables["latitude"] != "21.3045470" {
		t.Errorf("Expected latitude='21.3045470', got '%s'", session.Variables["latitude"])
	}

	if session.Variables["longitude"] != "-157.8556760" {
		t.Errorf("Expected longitude='-157.8556760', got '%s'", session.Variables["longitude"])
	}

	// The response should show the coordinates
	if !strings.Contains(response, "21.3045470") || !strings.Contains(response, "-157.8556760") {
		t.Errorf("Response should contain coordinates, got: %s", response)
	}
}

// Helper function to check if response contains think tag artifacts
func containsThinkTag(s string) bool {
	return strings.Contains(s, "<think>") || strings.Contains(s, "</think>") || strings.Contains(s, "<think")
}
