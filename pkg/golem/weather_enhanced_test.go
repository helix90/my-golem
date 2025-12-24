package golem

import (
	"strings"
	"testing"
)

func TestEnhancedWeatherFormatWithWindWarnings(t *testing.T) {
	tests := []struct {
		name        string
		weatherJSON string
		day         string
		mustContain []string
	}{
		{
			name: "High wind warning (40+ mph gusts)",
			weatherJSON: `{
				"currently": {
					"summary": "Partly Cloudy",
					"temperature": 10,
					"windSpeed": 18,
					"windGust": 20,
					"precipProbability": 0.3,
					"precipType": "rain"
				},
				"daily": {
					"data": [
						{"temperatureHigh": 15, "temperatureLow": 8}
					]
				}
			}`,
			day: "today",
			mustContain: []string{
				"Partly Cloudy",
				"50°F",
				"10°C",
				"Wind:",
				"40 mph",
				"72 km/h",
				"65 km/h",
				"Rain: 30% chance",
				"HIGH WIND WARNING",
			},
		},
		{
			name: "Windy conditions (25-40 mph gusts)",
			weatherJSON: `{
				"currently": {
					"summary": "Cloudy",
					"temperature": 12,
					"windSpeed": 10,
					"windGust": 14,
					"precipProbability": 0.6,
					"precipIntensity": 4.0,
					"precipType": "rain"
				},
				"daily": {
					"data": [
						{"temperatureHigh": 16, "temperatureLow": 10}
					]
				}
			}`,
			day: "today",
			mustContain: []string{
				"Cloudy",
				"54°F",
				"Wind:",
				"22 mph",
				"31 mph",
				"Rain: 60% chance",
				"moderate",
				"Windy conditions",
			},
		},
		{
			name: "Very humid with precipitation",
			weatherJSON: `{
				"currently": {
					"summary": "Rainy",
					"temperature": 18,
					"humidity": 0.88,
					"precipProbability": 0.75,
					"precipIntensity": 12,
					"precipType": "rain"
				},
				"daily": {
					"data": [
						{"temperatureHigh": 20, "temperatureLow": 15}
					]
				}
			}`,
			day: "today",
			mustContain: []string{
				"Rainy",
				"64°F",
				"Rain: 75% chance",
				"heavy",
				"Humidity: 88%",
				"very humid",
			},
		},
		{
			name: "High UV index warning",
			weatherJSON: `{
				"currently": {
					"summary": "Clear",
					"temperature": 28,
					"uvIndex": 9,
					"humidity": 0.25
				},
				"daily": {
					"data": [
						{"temperatureHigh": 32, "temperatureLow": 22}
					]
				}
			}`,
			day: "today",
			mustContain: []string{
				"Clear",
				"82°F",
				"28°C",
				"Humidity: 25%",
				"dry",
				"VERY HIGH UV INDEX",
				"sunscreen",
				"protective clothing",
			},
		},
		{
			name: "Low visibility warning",
			weatherJSON: `{
				"currently": {
					"summary": "Foggy",
					"temperature": 8,
					"visibility": 0.5,
					"humidity": 0.95
				},
				"daily": {
					"data": [
						{"temperatureHigh": 12, "temperatureLow": 6}
					]
				}
			}`,
			day: "today",
			mustContain: []string{
				"Foggy",
				"46°F",
				"Low visibility",
				"0.5 km",
				"Humidity: 95%",
				"very humid",
			},
		},
		{
			name: "Tomorrow with wind and precipitation",
			weatherJSON: `{
				"daily": {
					"data": [
						{"temperatureHigh": 15, "temperatureLow": 8},
						{
							"summary": "Rainy and windy",
							"temperatureHigh": 12,
							"temperatureLow": 6,
							"windSpeed": 15,
							"windGust": 18,
							"precipProbability": 0.85,
							"precipType": "rain",
							"uvIndex": 2
						}
					]
				}
			}`,
			day: "tomorrow",
			mustContain: []string{
				"Tomorrow will be rainy and windy",
				"54°F",
				"12°C",
				"43°F",
				"6°C",
				"Wind:",
				"34 mph",
				"40 mph",
				"Rain: 85% chance",
				"HIGH WIND WARNING",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing()

			tp := &TreeProcessor{
				golem: g,
				ctx:   &VariableContext{},
			}

			// Create AST node for weatherformat tag
			node := &ASTNode{
				Type:     NodeTypeTag,
				TagName:  "weatherformat",
				Attributes: map[string]string{},
			}

			if tt.day == "tomorrow" {
				node.Attributes["day"] = "tomorrow"
			}

			result := tp.processWeatherFormatTag(node, tt.weatherJSON)

			t.Logf("Weather output: %s", result)

			for _, expected := range tt.mustContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't. Got: %s", expected, result)
				}
			}
		})
	}
}

func TestWeatherWarningEmoji(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	tp := &TreeProcessor{
		golem: g,
		ctx:   &VariableContext{},
	}

	weatherJSON := `{
		"currently": {
			"summary": "Stormy",
			"temperature": 10,
			"windSpeed": 18,
			"windGust": 20,
			"uvIndex": 8,
			"visibility": 0.8
		},
		"daily": {
			"data": [
				{"temperatureHigh": 15, "temperatureLow": 8}
			]
		}
	}`

	node := &ASTNode{
		Type:       NodeTypeTag,
		TagName:    "weatherformat",
		Attributes: map[string]string{},
	}

	result := tp.processWeatherFormatTag(node, weatherJSON)

	t.Logf("Weather output with multiple warnings: %s", result)

	// Should have warning emoji
	if !strings.Contains(result, "⚠️") {
		t.Errorf("Expected warning emoji in output with multiple warnings")
	}

	// Should have multiple warnings
	warnings := []string{
		"HIGH WIND WARNING",
		"VERY HIGH UV INDEX",
		"Low visibility",
	}

	for _, warning := range warnings {
		if !strings.Contains(result, warning) {
			t.Errorf("Expected warning: %s", warning)
		}
	}
}
