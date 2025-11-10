package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	// Create a new Golem instance with verbose logging
	g := golem.New(true)

	// Create a simple AIML knowledge base with learning capabilities
	kb := golem.NewAIMLKnowledgeBase()
	kb.Categories = []golem.Category{
		{
			Pattern:  "HELLO",
			Template: "Hello! I can learn new things. Try teaching me something!",
		},
		{
			Pattern: "TEACH ME *",
			Template: `<learn>
				<category>
					<pattern>I KNOW *</pattern>
					<template>Yes, I know about <star/>!</template>
				</category>
			</learn>I've learned that pattern! Now you can say "I know something" and I'll respond.`,
		},
		{
			Pattern: "SAVE KNOWLEDGE *",
			Template: `<learnf>
				<category>
					<pattern>SAVED *</pattern>
					<template>I remember: <star/></template>
				</category>
			</learnf>Knowledge saved permanently! This will be remembered across sessions.`,
		},
		{
			Pattern: "LEARN GREETINGS",
			Template: `<learn>
				<category>
					<pattern>GOOD MORNING</pattern>
					<template>Good morning! How are you today?</template>
				</category>
				<category>
					<pattern>GOOD EVENING</pattern>
					<template>Good evening! How was your day?</template>
				</category>
				<category>
					<pattern>GOOD NIGHT</pattern>
					<template>Good night! Sleep well!</template>
				</category>
			</learn>I've learned several greeting patterns! Try saying "Good morning", "Good evening", or "Good night".`,
		},
		{
			Pattern:  "WHAT HAVE YOU LEARNED",
			Template: "I've learned many new patterns! I'm constantly growing and adapting. You can teach me new things anytime.",
		},
		{
			Pattern: "HELP",
			Template: `I can learn new things! Try these commands:
- "Teach me something" - Learn a new pattern
- "Save knowledge something" - Learn permanently
- "Learn greetings" - Learn multiple patterns
- "What have you learned" - See my learning status
- "Help" - Show this help message`,
		},
	}

	// Index patterns
	for i := range kb.Categories {
		category := &kb.Categories[i]
		pattern := golem.NormalizePattern(category.Pattern)
		kb.Patterns[pattern] = category
	}

	g.SetKnowledgeBase(kb)

	fmt.Println("=== AIML Learning Demo ===")
	fmt.Println("This demo shows how AIML bots can learn new patterns dynamically.")
	fmt.Println("Type 'quit' to exit.\n")

	// Interactive chat loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "quit" {
			break
		}

		if input == "" {
			continue
		}

		// Process the input through the AIML knowledge base
		category, wildcards, err := kb.MatchPattern(input)
		if err != nil {
			fmt.Printf("Bot: Error processing input: %v\n\n", err)
			continue
		}

		if category == nil {
			fmt.Printf("Bot: I don't understand: %s\n\n", input)
			continue
		}

		// Process the template
		response := g.ProcessTemplate(category.Template, wildcards)
		fmt.Printf("Bot: %s\n\n", response)
	}

	fmt.Println("Goodbye! Thanks for teaching me new things!")
}
