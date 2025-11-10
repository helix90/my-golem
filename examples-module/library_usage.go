package main

import (
	"fmt"
	"log"
	"os"

	"github.com/helix90/my-golem/pkg/golem"
)

func main() {
	// Create a new Golem instance
	g := golem.New(true) // Enable verbose logging

	// Example 1: Process data
	input := "Hello, World!"
	result, err := g.ProcessData(input)
	if err != nil {
		log.Fatalf("Error processing data: %v", err)
	}
	fmt.Printf("Processed result: %s\n", result)

	// Example 2: Analyze data
	analysis, err := g.AnalyzeData(input)
	if err != nil {
		log.Fatalf("Error analyzing data: %v", err)
	}
	fmt.Printf("Analysis result: %+v\n", analysis)

	// Example 3: Generate output
	output, err := g.GenerateOutput(analysis)
	if err != nil {
		log.Fatalf("Error generating output: %v", err)
	}
	fmt.Printf("Generated output: %s\n", output)

	// Example 4: Load file (if it exists)
	if len(os.Args) > 1 {
		filename := os.Args[1]
		fileContent, err := g.LoadFile(filename)
		if err != nil {
			fmt.Printf("Error loading file %s: %v\n", filename, err)
		} else {
			fmt.Printf("Loaded file %s: %s\n", filename, fileContent)
		}
	}
}
