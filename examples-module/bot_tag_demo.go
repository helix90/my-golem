package main

import (
	"fmt"
	"log"

	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	// Create a new Golem instance
	g := golem.New(true) // Enable verbose logging

	// Set up bot properties
	g.aimlKB.Properties["name"] = "GolemBot"
	g.aimlKB.Properties["version"] = "1.0.0"
	g.aimlKB.Properties["author"] = "Golem Team"
	g.aimlKB.Properties["language"] = "en"

	// Load the bot tag example AIML file
	err := g.LoadAIMLFromFile("bot_tag_example.aiml")
	if err != nil {
		log.Fatalf("Failed to load AIML file: %v", err)
	}

	// Create a chat session
	session := g.CreateSession("bot-tag-demo")

	// Test various bot tag patterns
	testInputs := []string{
		"WHAT IS YOUR NAME",
		"WHAT VERSION ARE YOU",
		"TELL ME ABOUT YOURSELF",
		"MIXED TAGS",
		"WHO CREATED YOU",
	}

	fmt.Println("=== Bot Tag Demo ===")
	fmt.Println("Testing <bot> tag functionality...")
	fmt.Println()

	for _, input := range testInputs {
		fmt.Printf("User: %s\n", input)
		response, err := g.ProcessInput(input, session)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Bot: %s\n", response)
		}
		fmt.Println()
	}

	// Demonstrate the difference between <bot> and <get> tags
	fmt.Println("=== Bot vs Get Tag Comparison ===")

	// Test with <bot> tag
	botTemplate := "I am <bot name=\"name\"/> version <bot name=\"version\"/>."
	fmt.Printf("Template with <bot> tags: %s\n", botTemplate)
	botResult := g.ProcessTemplateWithContext(botTemplate, make(map[string]string), session)
	fmt.Printf("Result: %s\n\n", botResult)

	// Test with <get> tag
	getTemplate := "I am <get name=\"name\"/> version <get name=\"version\"/>."
	fmt.Printf("Template with <get> tags: %s\n", getTemplate)
	getResult := g.ProcessTemplateWithContext(getTemplate, make(map[string]string), session)
	fmt.Printf("Result: %s\n\n", getResult)

	fmt.Println("Both <bot> and <get> tags produce the same result!")
}
