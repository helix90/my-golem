package main

import (
	"fmt"
	"os"

	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	// Create a new Golem instance with verbose logging
	g := golem.New(true)

	fmt.Println("=== Secure SRAIX Configuration Demo ===")
	fmt.Println("This demo shows different ways to securely handle API keys for SRAIX services.\n")

	// Method 1: Direct configuration with environment variables
	fmt.Println("Method 1: Direct configuration with environment variables")
	if err := setupDirectEnvConfig(g); err != nil {
		fmt.Printf("Error setting up direct env config: %v\n", err)
	} else {
		fmt.Println("✓ Direct environment variable configuration loaded")
	}

	// Method 2: Load from template with environment variable substitution
	fmt.Println("\nMethod 2: Template with environment variable substitution")
	if err := LoadSRAIXConfigsWithEnvVars(g, "sraix_config_template.json"); err != nil {
		fmt.Printf("Error loading template config: %v\n", err)
	} else {
		fmt.Println("✓ Template configuration loaded")
	}

	// Method 3: Load from secrets file
	fmt.Println("\nMethod 3: Secrets file")
	if err := LoadSRAIXConfigsFromSecrets(g, "secrets.json"); err != nil {
		fmt.Printf("Error loading secrets (this is expected if secrets.json doesn't exist): %v\n", err)
	} else {
		fmt.Println("✓ Secrets configuration loaded")
	}

	// Show all loaded configurations
	fmt.Println("\n=== Loaded SRAIX Services ===")
	configs := g.ListSRAIXConfigs()
	if len(configs) == 0 {
		fmt.Println("No SRAIX services configured.")
		fmt.Println("\nTo configure services, you can:")
		fmt.Println("1. Set environment variables (e.g., export OPENAI_API_KEY=your-key)")
		fmt.Println("2. Create a secrets.json file based on secrets.json.example")
		fmt.Println("3. Use the template configuration approach")
		return
	}

	for name, config := range configs {
		fmt.Printf("  %s: %s %s\n", name, config.Method, config.BaseURL)
		// Don't print headers to avoid exposing API keys in logs
	}

	// Create a simple AIML knowledge base for testing
	kb := golem.NewAIMLKnowledgeBase()
	kb.Categories = []golem.Category{
		{
			Pattern:  "ASK AI *",
			Template: "AI Response: <sraix service=\"openai_service\">Please answer: <star/></sraix>",
		},
		{
			Pattern:  "WEATHER IN *",
			Template: "Weather: <sraix service=\"weather_service\">What is the weather in <star/>?</sraix>",
		},
		{
			Pattern:  "TRANSLATE * TO SPANISH",
			Template: "Translation: <sraix service=\"translation_service\">Translate \"<star/>\" to Spanish</sraix>",
		},
		{
			Pattern:  "HELP",
			Template: "I can help you test SRAIX services! Try:\n- ASK AI what is machine learning?\n- WEATHER IN New York\n- TRANSLATE hello TO SPANISH\n- HELP",
		},
	}

	g.SetKnowledgeBase(kb)

	fmt.Println("\n=== Interactive Demo ===")
	fmt.Println("Type 'quit' to exit.\n")

	// Interactive chat loop
	for {
		fmt.Print("You: ")
		var input string
		fmt.Scanln(&input)

		if input == "quit" {
			break
		}

		// Process the input through the AIML knowledge base
		response := g.ProcessTemplate(input, make(map[string]string))
		fmt.Printf("Bot: %s\n\n", response)
	}

	fmt.Println("Goodbye!")
}

// setupDirectEnvConfig demonstrates direct configuration with environment variables
func setupDirectEnvConfig(g *golem.Golem) error {
	// Check if we have the required environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

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
		IncludeWildcards: true,
	}

	return g.AddSRAIXConfig(config)
}
