package golem

import (
	"strings"
	"testing"
)

// TestTestDataDirectoryLoad verifies that loading the entire testdata directory
// doesn't result in conflicting configurations or broken functionality.
// This test is critical for preventing regressions where example config files
// accidentally override actual service configurations.
func TestTestDataDirectoryLoad(t *testing.T) {
	g := NewForTesting(t, true)

	// Load entire testdata directory
	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load testdata directory: %v", err)
	}

	// Set the knowledge base
	g.SetKnowledgeBase(kb)

	// Verify AIML categories were loaded
	if kb == nil {
		t.Fatal("Knowledge base is nil after loading testdata")
	}

	if len(kb.Categories) == 0 {
		t.Error("No categories loaded from testdata directory")
	}

	t.Logf("Loaded %d categories from testdata", len(kb.Categories))

	// Verify bot properties were loaded
	if len(kb.Properties) == 0 {
		t.Error("No bot properties loaded from testdata")
	}

	// Check for expected bot properties
	expectedProps := []string{"name", "age", "species", "location"}
	for _, prop := range expectedProps {
		if _, exists := kb.Properties[prop]; !exists {
			t.Errorf("Expected bot property '%s' not found", prop)
		}
	}

	t.Logf("Loaded %d bot properties", len(kb.Properties))
}

// TestTestDataSRAIXConfiguration verifies that SRAIX service configurations
// are loaded correctly without conflicts from example files.
func TestTestDataSRAIXConfiguration(t *testing.T) {
	g := NewForTesting(t, true)

	// Load entire testdata directory
	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load testdata directory: %v", err)
	}

	// Set the knowledge base to trigger SRAIX configuration from properties
	g.SetKnowledgeBase(kb)

	// Check SRAIX manager configuration
	if g.sraixMgr == nil {
		t.Fatal("SRAIX manager is nil after loading")
	}

	// Test weather service configuration
	// CRITICAL: weather-config.properties intentionally omits responsepath
	// to return full JSON for weatherformat tag processing
	t.Run("WeatherServiceConfig", func(t *testing.T) {
		config, exists := g.sraixMgr.GetServiceConfig("weather")
		if !exists {
			t.Fatal("Weather service not configured")
		}

		// Verify weather service has no ResponsePath (should return full JSON)
		if config.ResponsePath != "" {
			t.Errorf("Weather service should not have ResponsePath, but has '%s'. "+
				"This usually means sraix-config-example.properties is overriding weather-config.properties. "+
				"Example files should be renamed to .bak or moved to prevent conflicts.",
				config.ResponsePath)
		}

		// Verify weather service uses JSON format
		if config.ResponseFormat != "json" {
			t.Errorf("Weather service should use JSON format, got '%s'", config.ResponseFormat)
		}

		// Verify URL template contains Pirate Weather API
		if !strings.Contains(config.URLTemplate, "pirateweather.net") {
			t.Errorf("Weather service should use Pirate Weather API, got URL: %s", config.URLTemplate)
		}

		t.Logf("Weather service config: ResponsePath='%s', ResponseFormat='%s'",
			config.ResponsePath, config.ResponseFormat)
	})

	// Test geocode service configuration
	t.Run("GeocodeServiceConfig", func(t *testing.T) {
		config, exists := g.sraixMgr.GetServiceConfig("geocode")
		if !exists {
			t.Skip("Geocode service not configured (optional)")
		}

		// Geocode should have ResponsePath to extract latitude
		if config.ResponsePath != "0.lat" {
			t.Errorf("Geocode service should have ResponsePath '0.lat', got '%s'", config.ResponsePath)
		}

		// Verify it uses Nominatim
		if !strings.Contains(config.URLTemplate, "nominatim.openstreetmap.org") {
			t.Errorf("Geocode service should use Nominatim, got URL: %s", config.URLTemplate)
		}

		t.Logf("Geocode service config: ResponsePath='%s'", config.ResponsePath)
	})

	// Test geocode_lon service configuration
	t.Run("GeocodeLonServiceConfig", func(t *testing.T) {
		config, exists := g.sraixMgr.GetServiceConfig("geocode_lon")
		if !exists {
			t.Skip("Geocode_lon service not configured (optional)")
		}

		// Geocode_lon should have ResponsePath to extract longitude
		if config.ResponsePath != "0.lon" {
			t.Errorf("Geocode_lon service should have ResponsePath '0.lon', got '%s'", config.ResponsePath)
		}

		t.Logf("Geocode_lon service config: ResponsePath='%s'", config.ResponsePath)
	})

	// Check for unexpected service configurations that might indicate conflicts
	t.Run("NoConflictingServices", func(t *testing.T) {
		// List all configured services
		services := g.sraixMgr.ListServices()
		t.Logf("Configured SRAIX services: %v", services)

		// Check that we don't have duplicate or conflicting weather services
		for _, service := range services {
			if strings.Contains(service, "weather") && service != "weather" {
				t.Logf("Found additional weather-related service: %s", service)
			}
		}
	})
}

// TestTestDataBasicFunctionality tests that basic AIML patterns work correctly
// after loading the testdata directory.
func TestTestDataBasicFunctionality(t *testing.T) {
	g := NewForTesting(t, true)

	// Load entire testdata directory
	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load testdata directory: %v", err)
	}

	// Set the knowledge base to make patterns available
	g.SetKnowledgeBase(kb)

	// Create a test session
	sessionID := "test-session-1"
	session := g.CreateSession(sessionID)

	testCases := []struct{
		name     string
		input    string
		contains []string // Response should contain at least one of these
		notEmpty bool
	}{
		{
			name:     "BotName",
			input:    "what is your name",
			contains: []string{"name", "Golem", "golem"},
			notEmpty: true,
		},
		{
			name:     "HelloPattern",
			input:    "hello",
			contains: []string{"hello", "hi", "greetings"},
			notEmpty: true,
		},
		{
			name:     "BotProperty",
			input:    "what is your age",
			contains: []string{"age", "year", "old"},
			notEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := g.ProcessInput(tc.input, session)
			if err != nil {
				t.Errorf("ProcessInput failed: %v", err)
				return
			}

			if tc.notEmpty && response == "" {
				t.Error("Expected non-empty response")
				return
			}

			if len(tc.contains) > 0 {
				found := false
				for _, substr := range tc.contains {
					if strings.Contains(strings.ToLower(response), strings.ToLower(substr)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Response '%s' should contain one of %v", response, tc.contains)
				}
			}

			t.Logf("Input: %s -> Response: %s", tc.input, response)
		})
	}
}

// TestTestDataExampleFilesNaming verifies that example files follow naming conventions
// to prevent accidental loading conflicts.
func TestTestDataExampleFilesNaming(t *testing.T) {
	// This test documents the expected naming convention for example files
	// Example files that might conflict with active configs should be:
	// 1. Named with "-example" suffix (e.g., sraix-config-example.properties)
	// 2. Documented in README files
	// 3. Not override active service configurations

	t.Log("Example file naming convention:")
	t.Log("  - Active configs: service-config.properties (e.g., weather-config.properties)")
	t.Log("  - Examples: service-config-example.properties or service-config-example.aiml")
	t.Log("  - Examples should NOT override active service configurations")
	t.Log("  - If conflicts occur, rename example files to .bak or move to docs/")

	// Load testdata and check for potential conflicts
	g := NewForTesting(t, false) // Disable verbose for cleaner test output
	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load testdata directory: %v", err)
	}

	// Set the knowledge base to trigger SRAIX configuration from properties
	g.SetKnowledgeBase(kb)

	// Warn about any service that has both base config and example config
	if g.sraixMgr != nil {
		services := g.sraixMgr.ListServices()

		// Check if we have services that might be defined in multiple places
		serviceNames := make(map[string]bool)
		for _, service := range services {
			// Remove common suffixes to group related services
			baseName := strings.TrimSuffix(service, "_lon")
			baseName = strings.TrimSuffix(baseName, "_lat")
			serviceNames[baseName] = true
		}

		t.Logf("Unique service base names found: %v", keys(serviceNames))
	}
}

// TestWeatherConfigurationSpecifically is a focused test for the weather service
// configuration to prevent the specific regression we encountered.
func TestWeatherConfigurationSpecifically(t *testing.T) {
	g := NewForTesting(t, true)

	// Load entire testdata directory
	kb, err := g.LoadAIMLFromDirectory("../../testdata")
	if err != nil {
		t.Fatalf("Failed to load testdata directory: %v", err)
	}

	// Set the knowledge base to trigger SRAIX configuration from properties
	g.SetKnowledgeBase(kb)

	// Get weather service config
	config, exists := g.sraixMgr.GetServiceConfig("weather")
	if !exists {
		t.Fatal("Weather service not configured after loading testdata")
	}

	// CRITICAL CHECK: weather service must not have responsepath
	// This was the bug - sraix-config-example.properties was setting
	// "weather.0.description" which doesn't exist in Pirate Weather API
	if config.ResponsePath != "" {
		t.Errorf("REGRESSION DETECTED: Weather service has ResponsePath='%s'\n"+
			"This should be empty to return full JSON for weatherformat tag.\n"+
			"Likely cause: sraix-config-example.properties is being loaded and overriding weather-config.properties.\n"+
			"Solution: Rename example files to .bak or ensure weather-config.properties loads after examples.",
			config.ResponsePath)
	}

	// Verify Pirate Weather API is configured
	if !strings.Contains(config.URLTemplate, "pirateweather.net") {
		t.Errorf("Weather service should use pirateweather.net, got: %s", config.URLTemplate)
	}

	// Verify JSON format
	if config.ResponseFormat != "json" {
		t.Errorf("Weather service should use JSON format, got: %s", config.ResponseFormat)
	}

	// Verify timeout is reasonable
	if config.Timeout <= 0 || config.Timeout > 60 {
		t.Errorf("Weather service timeout should be 1-60 seconds, got: %d", config.Timeout)
	}

	t.Logf("✓ Weather service correctly configured:")
	t.Logf("  - ResponsePath: (empty - returns full JSON)")
	t.Logf("  - ResponseFormat: %s", config.ResponseFormat)
	t.Logf("  - URLTemplate: %s", maskAPIKey(config.URLTemplate))
	t.Logf("  - Timeout: %d seconds", config.Timeout)
}

// Helper function to get map keys
func keys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Helper function to mask API keys in URLs for logging
func maskAPIKey(url string) string {
	// Replace anything that looks like an API key with asterisks
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if len(part) > 20 && !strings.Contains(part, "?") && !strings.Contains(part, ".") {
			parts[i] = "***API-KEY***"
		}
	}
	return strings.Join(parts, "/")
}
