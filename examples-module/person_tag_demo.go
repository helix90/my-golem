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
	aimlFilePath := filepath.Join(currentDir, "person_tag_example.aiml")

	// Load the AIML knowledge base
	kb, err := g.LoadAIML(aimlFilePath)
	if err != nil {
		log.Fatalf("Failed to load AIML file: %v", err)
	}

	// Set the knowledge base on the Golem instance
	g.SetKnowledgeBase(kb)

	// Set some bot properties after loading
	kb.SetProperty("name", "PersonTagBot")
	kb.SetProperty("version", "1.0")
	kb.SetProperty("author", "Helix")

	fmt.Println("Person Tag Demo Bot is ready. Type 'exit' to quit.")
	fmt.Println("This bot demonstrates the <person> tag for pronoun substitution.")
	fmt.Println()

	// Create a new chat session
	session := g.CreateSession("user123")

	// Simulate user input
	inputs := []string{
		"WHAT DID I SAY",
		"WHAT DO YOU THINK",
		"TELL ME ABOUT YOURSELF",
		"WHAT ARE YOUR PLANS",
		"COMPLEX RESPONSE",
		"CONTRACTIONS",
		"POSSESSIVES",
		"REFLEXIVE",
		"MIXED PRONOUNS",
		"NO PRONOUNS",
		"HELLO", // Should not match, as no category for HELLO
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
