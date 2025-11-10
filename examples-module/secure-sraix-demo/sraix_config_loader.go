package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// SRAIXConfigTemplate represents a configuration template with environment variable placeholders
type SRAIXConfigTemplate struct {
	Name             string            `json:"name"`
	BaseURL          string            `json:"base_url"`
	Method           string            `json:"method"`
	Headers          map[string]string `json:"headers"`
	Timeout          int               `json:"timeout"`
	ResponseFormat   string            `json:"response_format"`
	ResponsePath     string            `json:"response_path"`
	FallbackResponse string            `json:"fallback_response"`
	IncludeWildcards bool              `json:"include_wildcards"`
}

// LoadSRAIXConfigsWithEnvVars loads SRAIX configurations from a template file with environment variable substitution
func LoadSRAIXConfigsWithEnvVars(g *golem.Golem, templateFile string) error {
	// Read the template file
	data, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}

	// Substitute environment variables
	content := string(data)
	content = substituteEnvVars(content)

	// Parse the JSON
	var configs []SRAIXConfigTemplate
	if err := json.Unmarshal([]byte(content), &configs); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// Convert to SRAIXConfig and add to Golem
	for _, template := range configs {
		config := &golem.SRAIXConfig{
			Name:             template.Name,
			BaseURL:          template.BaseURL,
			Method:           template.Method,
			Headers:          template.Headers,
			Timeout:          template.Timeout,
			ResponseFormat:   template.ResponseFormat,
			ResponsePath:     template.ResponsePath,
			FallbackResponse: template.FallbackResponse,
			IncludeWildcards: template.IncludeWildcards,
		}

		if err := g.AddSRAIXConfig(config); err != nil {
			return fmt.Errorf("failed to add SRAIX config %s: %v", config.Name, err)
		}
	}

	return nil
}

// substituteEnvVars replaces ${VAR_NAME} patterns with environment variable values
func substituteEnvVars(content string) string {
	// Find all ${VAR_NAME} patterns
	for {
		start := strings.Index(content, "${")
		if start == -1 {
			break
		}
		
		end := strings.Index(content[start:], "}")
		if end == -1 {
			break
		}
		
		end += start
		varName := content[start+2 : end]
		
		// Get environment variable value
		value := os.Getenv(varName)
		if value == "" {
			fmt.Printf("Warning: Environment variable %s is not set\n", varName)
		}
		
		// Replace the pattern with the value
		content = content[:start] + value + content[end+1:]
	}
	
	return content
}

// LoadSRAIXConfigsFromSecrets loads configurations from a secrets file
func LoadSRAIXConfigsFromSecrets(g *golem.Golem, secretsFile string) error {
	// Read secrets file
	secretsData, err := os.ReadFile(secretsFile)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %v", err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(secretsData, &secrets); err != nil {
		return fmt.Errorf("failed to parse secrets file: %v", err)
	}

	// Example: Create OpenAI configuration using secrets
	if apiKey, exists := secrets["openai_api_key"]; exists {
		config := &golem.SRAIXConfig{
			Name:    "openai_service",
			BaseURL: "https://api.openai.com/v1/chat/completions",
			Method:  "POST",
			Headers: map[string]string{
				"Authorization": "Bearer " + apiKey,
				"Content-Type":  "application/json",
			},
			Timeout:          30,
			ResponseFormat:   "json",
			ResponsePath:     "choices.0.message.content",
			FallbackResponse: "I'm sorry, I couldn't process that request right now.",
		}

		if err := g.AddSRAIXConfig(config); err != nil {
			return fmt.Errorf("failed to add OpenAI config: %v", err)
		}
	}

	// Add more services as needed...

	return nil
}

// Example usage function
func ExampleUsage() {
	g := golem.New(true)

	// Method 1: Load from template with environment variables
	if err := LoadSRAIXConfigsWithEnvVars(g, "sraix_config_template.json"); err != nil {
		fmt.Printf("Error loading configs: %v\n", err)
		return
	}

	// Method 2: Load from secrets file
	if err := LoadSRAIXConfigsFromSecrets(g, "secrets.json"); err != nil {
		fmt.Printf("Error loading secrets: %v\n", err)
		return
	}

	// Method 3: Direct configuration with environment variables
	config := &golem.SRAIXConfig{
		Name:    "direct_env_service",
		BaseURL: "https://api.example.com",
		Headers: map[string]string{
			"Authorization": "Bearer " + os.Getenv("EXAMPLE_API_KEY"),
		},
	}
	g.AddSRAIXConfig(config)

	fmt.Println("SRAIX configurations loaded successfully!")
}
