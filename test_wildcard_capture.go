package main

import (
	"fmt"
	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	fmt.Println("=== Testing Wildcard Capture ===")

	g := golem.New(true) // Enable verbose to see pattern matching

	kb, err := g.LoadAIML("test_wildcard.aiml")
	if err != nil {
		fmt.Printf("Error loading AIML: %v\n", err)
		return
	}

	g.SetKnowledgeBase(kb)

	// Debug: Print the loaded patterns
	fmt.Println("\n=== Loaded Patterns ===")
	for pattern := range kb.Patterns {
		fmt.Printf("Pattern: '%s'\n", pattern)
	}
	fmt.Println("======================\n")

	session := g.CreateSession("test")

	fmt.Println("\nTest 1: 'test helix FblE2013aa'")
	response, _ := g.ProcessInput("test helix FblE2013aa", session)
	fmt.Printf("Response: %s\n", response)

	fmt.Println("\nTest 2: 'test user pass'")
	response, _ = g.ProcessInput("test user pass", session)
	fmt.Printf("Response: %s\n", response)
}
