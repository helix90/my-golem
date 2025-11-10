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
	aimlFilePath := filepath.Join(currentDir, "gender_tag_example.aiml")

	// Load the AIML knowledge base
	kb, err := g.LoadAIML(aimlFilePath)
	if err != nil {
		log.Fatalf("Failed to load AIML file: %v", err)
	}

	// Set the knowledge base on the Golem instance
	g.SetKnowledgeBase(kb)

	// Set some bot properties after loading
	kb.SetProperty("name", "GenderTagBot")
	kb.SetProperty("version", "1.0")
	kb.SetProperty("author", "Helix")

	fmt.Println("Gender Tag Demo Bot is ready. Type 'exit' to quit.")
	fmt.Println("This bot demonstrates the <gender> tag for gender pronoun substitution.")
	fmt.Println("The <gender> tag swaps masculine and feminine pronouns (he/she, him/her, his/hers, etc.)")
	fmt.Println()

	// Create a new chat session
	session := g.CreateSession("user123")

	// Simulate user input
	inputs := []string{
		"hello",
		"tell me about the doctor",
		"tell me about the teacher",
		"what did he say",
		"what did she say",
		"describe the manager",
		"describe the nurse",
		"what does he think",
		"what does she think",
		"how does he feel",
		"how does she feel",
		"tell me about the couple",
		"what is his opinion",
		"what is her opinion",
		"describe the leader",
		"describe the scientist",
		"ask him a question",
		"ask her a question",
		"tell me about something else",
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
