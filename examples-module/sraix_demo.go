package main

import (
	"fmt"
	"log"

	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	// Create a new Golem instance with verbose logging
	g := golem.New(true)

	// Add a simple SRAIX configuration for testing
	config := &golem.SRAIXConfig{
		Name:             "echo_service",
		BaseURL:          "https://httpbin.org/post", // Test service that echoes back requests
		Method:           "POST",
		Timeout:          10,
		ResponseFormat:   "json",
		ResponsePath:     "json.input", // Extract the input from the echoed response
		FallbackResponse: "Echo service is unavailable",
		IncludeWildcards: true,
	}

	err := g.AddSRAIXConfig(config)
	if err != nil {
		log.Fatalf("Failed to add SRAIX config: %v", err)
	}

	// Create a simple AIML knowledge base with SRAIX
	kb := golem.NewAIMLKnowledgeBase()
	kb.Categories = []golem.Category{
		{
			Pattern:  "ECHO *",
			Template: "Echo service says: <sraix service=\"echo_service\">Echo: <star/></sraix>",
		},
		{
			Pattern:  "HELLO",
			Template: "Hello! I can echo your messages using SRAIX. Try saying 'ECHO hello world'",
		},
		{
			Pattern:  "HELP",
			Template: "I can help you test SRAIX functionality. Try these commands:\n- HELLO\n- ECHO your message here\n- HELP",
		},
	}

	g.SetKnowledgeBase(kb)

	fmt.Println("=== SRAIX Demo ===")
	fmt.Println("This demo shows SRAIX (external service integration) in action.")
	fmt.Println("The bot will use an external echo service to process your messages.")
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
		response := g.ProcessTemplate("ECHO "+input, make(map[string]string))
		fmt.Printf("Bot: %s\n\n", response)
	}

	fmt.Println("Goodbye!")
}
