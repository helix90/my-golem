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
	aimlFilePath := filepath.Join(currentDir, "text_processing_example.aiml")

	// Load the AIML knowledge base
	kb, err := g.LoadAIML(aimlFilePath)
	if err != nil {
		log.Fatalf("Failed to load AIML file: %v", err)
	}

	// Set the knowledge base on the Golem instance
	g.SetKnowledgeBase(kb)

	// Set some bot properties after loading
	kb.SetProperty("name", "TextProcessingBot")
	kb.SetProperty("version", "1.0")
	kb.SetProperty("author", "Helix")

	fmt.Println("Text Processing Demo Bot is ready. Type 'exit' to quit.")
	fmt.Println("This bot demonstrates sentence splitting and word boundary detection.")
	fmt.Println("The <sentence> tag splits text into sentences using intelligent boundary detection.")
	fmt.Println("The <word> tag splits text into words and punctuation tokens.")
	fmt.Println()

	// Create a new chat session
	session := g.CreateSession("user123")

	// Simulate user input
	inputs := []string{
		"hello",
		"split sentences",
		"split complex sentences",
		"split words",
		"split complex words",
		"combined processing",
		"analyze text",
		"process with person",
		"process with gender",
		"empty text",
		"whitespace text",
		"tell me about something else", // Should trigger default response
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
}
