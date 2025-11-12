package main

import (
	"fmt"
	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	fmt.Println("=== Testing SRAIX Logging ===")
	fmt.Println("Creating Golem with verbose=true...")

	// Create Golem with verbose mode enabled
	g := golem.New(true)

	// Create knowledge base
	kb := golem.NewAIMLKnowledgeBase()

	// Load List Handler configuration
	fmt.Println("Loading List Handler configuration...")
	err := g.LoadPropertiesFromFile("testdata/list-handler-config.properties", kb)
	if err != nil {
		fmt.Printf("Error loading properties: %v\n", err)
		return
	}

	// Load List Handler AIML
	fmt.Println("Loading List Handler AIML...")
	err = g.LoadAIMLFile("testdata/list-handler-examples.aiml", kb)
	if err != nil {
		fmt.Printf("Error loading AIML: %v\n", err)
		return
	}

	g.SetKnowledgeBase(kb)

	// Set environment variable for List Handler URL
	// export LIST_HANDLER_URL="http://192.168.0.26:8088"
	fmt.Println("\nMake sure to set: export LIST_HANDLER_URL=\"http://192.168.0.26:8088\"")
	fmt.Println("\nAttempting login...")
	fmt.Println("You should see detailed SRAIX logs below:")
	fmt.Println("---")

	// Create session and try login
	session := g.CreateSession("test-user")
	response, err := g.ProcessInput("list login helix FblE2013aa", session)

	fmt.Println("---")
	fmt.Println("Response:", response)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n=== End Test ===")
}
