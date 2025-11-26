package golem

import (
	"strings"
	"testing"
)

// TestWeatherFormatTomorrow tests the weatherformat tag with day="tomorrow" attribute
func TestWeatherFormatTomorrow(t *testing.T) {
	// Sample Pirate Weather API response with daily forecast data
	sampleWeatherJSON := `{
		"latitude": 47.6062,
		"longitude": -122.3321,
		"timezone": "America/Los_Angeles",
		"currently": {
			"time": 1700000000,
			"summary": "Partly Cloudy",
			"temperature": 10.5
		},
		"daily": {
			"summary": "Rain throughout the week.",
			"data": [
				{
					"time": 1700000000,
					"summary": "Partly cloudy throughout the day.",
					"temperatureHigh": 15.0,
					"temperatureLow": 8.0
				},
				{
					"time": 1700086400,
					"summary": "Light rain in the morning.",
					"temperatureHigh": 12.5,
					"temperatureLow": 6.2
				}
			]
		}
	}`

	testCases := []struct {
		name           string
		day            string
		expectedPhrases []string // Should contain these phrases
	}{
		{
			name: "TodayWeather",
			day:  "today",
			expectedPhrases: []string{
				"Partly Cloudy",
				"°F", // Has temperature in Fahrenheit
				"59°F", // 15°C high
				"46°F", // 8°C low
			},
		},
		{
			name: "TomorrowWeather",
			day:  "tomorrow",
			expectedPhrases: []string{
				"Tomorrow will be",
				"light rain",
				"54°F", // 12.5°C high
				"43°F", // 6.2°C low
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := New(true)

			// Create AST node with day attribute
			node := &ASTNode{
				TagName: "weatherformat",
				Attributes: map[string]string{
					"day": tc.day,
				},
			}

			// Create tree processor
			tp := &TreeProcessor{
				golem: g,
				ctx:   &VariableContext{},
			}

			// Process the weather format tag
			result := tp.processWeatherFormatTag(node, sampleWeatherJSON)

			t.Logf("Day=%s, Result: %s", tc.day, result)

			// Check that result is not empty or an error
			if result == "" || strings.Contains(result, "unavailable") {
				t.Errorf("Expected valid weather response, got: %s", result)
			}

			// Check that expected phrases are in the result
			for _, phrase := range tc.expectedPhrases {
				if !strings.Contains(result, phrase) {
					t.Errorf("Expected result to contain '%s', but got: %s", phrase, result)
				}
			}
		})
	}
}

// TestWeatherFormatTomorrowEdgeCases tests edge cases for tomorrow's weather
func TestWeatherFormatTomorrowEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		jsonData    string
		day         string
		expectError bool
		errorPhrase string
	}{
		{
			name: "MissingDailyData",
			jsonData: `{
				"currently": {
					"summary": "Cloudy",
					"temperature": 10.0
				}
			}`,
			day:         "tomorrow",
			expectError: true,
			errorPhrase: "no forecast data",
		},
		{
			name: "InsufficientDailyData",
			jsonData: `{
				"currently": {
					"summary": "Cloudy",
					"temperature": 10.0
				},
				"daily": {
					"data": [
						{
							"summary": "Today only",
							"temperatureHigh": 15.0,
							"temperatureLow": 8.0
						}
					]
				}
			}`,
			day:         "tomorrow",
			expectError: true,
			errorPhrase: "insufficient forecast data",
		},
		{
			name: "ValidTomorrowData",
			jsonData: `{
				"currently": {
					"summary": "Sunny",
					"temperature": 20.0
				},
				"daily": {
					"data": [
						{
							"summary": "Sunny today",
							"temperatureHigh": 25.0,
							"temperatureLow": 15.0
						},
						{
							"summary": "Rainy tomorrow",
							"temperatureHigh": 18.0,
							"temperatureLow": 12.0
						}
					]
				}
			}`,
			day:         "tomorrow",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := New(false) // Disable verbose for cleaner test output

			node := &ASTNode{
				TagName: "weatherformat",
				Attributes: map[string]string{
					"day": tc.day,
				},
			}

			tp := &TreeProcessor{
				golem: g,
				ctx:   &VariableContext{},
			}

			result := tp.processWeatherFormatTag(node, tc.jsonData)

			t.Logf("Result: %s", result)

			if tc.expectError {
				if !strings.Contains(result, "unavailable") {
					t.Errorf("Expected error response containing 'unavailable', got: %s", result)
				}
				if tc.errorPhrase != "" && !strings.Contains(result, tc.errorPhrase) {
					t.Errorf("Expected error to contain '%s', got: %s", tc.errorPhrase, result)
				}
			} else {
				if strings.Contains(result, "unavailable") {
					t.Errorf("Did not expect error response, got: %s", result)
				}
				// For valid tomorrow data, check it mentions tomorrow
				if tc.day == "tomorrow" && !strings.Contains(result, "Tomorrow") {
					t.Errorf("Expected response to mention 'Tomorrow', got: %s", result)
				}
			}
		})
	}
}

// TestWeatherTomorrowAIMLPatterns tests the AIML patterns for tomorrow queries
func TestWeatherTomorrowAIMLPatterns(t *testing.T) {
	g := New(false)

	// Load weather AIML
	kb, err := g.LoadAIML("../../testdata/weather.aiml")
	if err != nil {
		t.Fatalf("Failed to load weather.aiml: %v", err)
	}

	g.SetKnowledgeBase(kb)

	// Create a session
	session := g.CreateSession("test-weather-tomorrow")

	testPatterns := []struct {
		input          string
		shouldMatch    bool
		mustNotContain []string // Phrases that indicate pattern didn't match
	}{
		{
			input:          "what is the weather tomorrow",
			shouldMatch:    true,
			mustNotContain: []string{"don't understand", "not sure", "rephrase"},
		},
		{
			input:          "what will the weather be tomorrow",
			shouldMatch:    true,
			mustNotContain: []string{"don't understand", "not sure", "rephrase"},
		},
		{
			input:          "weather tomorrow",
			shouldMatch:    true,
			mustNotContain: []string{"don't understand", "not sure", "rephrase"},
		},
		{
			input:          "tomorrow weather",
			shouldMatch:    true,
			mustNotContain: []string{"don't understand", "not sure", "rephrase"},
		},
		{
			input:          "what is the weather tomorrow in seattle",
			shouldMatch:    true,
			mustNotContain: []string{"don't understand", "not sure", "rephrase"},
		},
		{
			input:          "weather tomorrow in boston",
			shouldMatch:    true,
			mustNotContain: []string{"don't understand", "not sure", "rephrase"},
		},
	}

	for _, tc := range testPatterns {
		t.Run(tc.input, func(t *testing.T) {
			response, err := g.ProcessInput(tc.input, session)
			if err != nil {
				// Pattern not matching at all
				if tc.shouldMatch {
					t.Errorf("Pattern should have matched but got error: %v", err)
				}
				return
			}

			t.Logf("Input: '%s' -> Response: '%s'", tc.input, response)

			if tc.shouldMatch {
				// Check that we didn't get default "I don't understand" responses
				responseLower := strings.ToLower(response)
				for _, phrase := range tc.mustNotContain {
					if strings.Contains(responseLower, phrase) {
						t.Errorf("Pattern matched but response contains '%s': %s", phrase, response)
					}
				}

				// The response should contain something (not be empty)
				if strings.TrimSpace(response) == "" {
					t.Errorf("Pattern matched but response is empty")
				}
			}
		})
	}
}
