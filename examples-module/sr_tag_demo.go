package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	// Create a new Golem instance
	g := golem.New(true) // Set to true for verbose logging

	// Define the path to the AIML file
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}
	aimlFilePath := filepath.Join(currentDir, "sr_tag_example.aiml")

	// Load the AIML knowledge base
	kb, err := g.LoadAIML(aimlFilePath)
	if err != nil {
		log.Fatalf("Failed to load AIML file: %v", err)
	}

	// Set the knowledge base on the Golem instance
	g.SetKnowledgeBase(kb)

	// Set some bot properties after loading
	kb.SetProperty("name", "SRTagBot")
	kb.SetProperty("version", "1.0")
	kb.SetProperty("author", "Helix")

	fmt.Println("SR Tag Demo Bot is ready. Type 'exit' to quit.")
	fmt.Println("This bot demonstrates the <sr> tag for recursive pattern matching.")
	fmt.Println("The <sr> tag is shorthand for <srai><star/></srai>")
	fmt.Println()

	// Create a new chat session
	session := g.CreateSession("user123")

	// Simulate user input to demonstrate SR tag functionality
	inputs := []string{
		"GREETING HELLO",         // Should use SR tag to process HELLO
		"GREETING HI",            // Should use SR tag to process HI
		"WELCOME GREETING HELLO", // Should use SR tag recursively
		"SAY HELLO TO WORLD",     // Should use SR tag with multiple wildcards
		"HOWDY",                  // Should use SRAI to redirect to HELLO
		"GOOD MORNING",           // Should use SRAI to redirect to HELLO
		"H3LLO",                  // Should use SRAI for spelling correction
		"H3Y",                    // Should use SRAI for spelling correction
		"HELLO THERE",            // Should use SRAI for phrase normalization
		"HI THERE",               // Should use SRAI for phrase normalization
		"WHO ARE YOU",            // Should use SRAI to redirect to WHAT IS YOUR NAME
		"INTRODUCE YOURSELF",     // Should use SRAI to redirect to WHAT IS YOUR NAME
		"HOW DO YOU DO",          // Should use SRAI to redirect to HOW ARE YOU
		"HOWS IT GOING",          // Should use SRAI to redirect to HOW ARE YOU
		"BYE",                    // Should use SRAI to redirect to GOODBYE
		"SEE YOU LATER",          // Should use SRAI to redirect to GOODBYE
		"FAREWELL",               // Should use SRAI to redirect to GOODBYE
		"TELL ME ABOUT HELLO",    // Should use SR tag to process HELLO
		"TELL ME ABOUT HI",       // Should use SR tag to process HI
		"WHAT",                   // Should use SRAI to redirect to I DON'T UNDERSTAND
		"HUH",                    // Should use SRAI to redirect to I DON'T UNDERSTAND
		"UNKNOWN PATTERN",        // Should not match any pattern
	}

	for _, input := range inputs {
		fmt.Printf("User: %s\n", input)
		response, err := g.ProcessInput(input, session)
		if err != nil {
			fmt.Printf("Bot (Error): %v\n\n", err)
			continue
		}
		fmt.Printf("Bot: %s\n\n", response)
	}

	fmt.Println("Demo completed!")
	fmt.Println("\nKey features demonstrated:")
	fmt.Println("1. <sr/> tag - shorthand for <srai><star/></srai>")
	fmt.Println("2. Recursive pattern matching")
	fmt.Println("3. Synonym handling")
	fmt.Println("4. Spelling correction")
	fmt.Println("5. Phrase normalization")
	fmt.Println("6. Multiple wildcard handling")
}
