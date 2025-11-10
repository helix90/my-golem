package main

import (
	"fmt"
	"log"
	"os"

	"github.com/helix90/my-golem/pkg/golem"
)

// TestTelegramIntegration demonstrates how to use Golem with Telegram bot
func main() {
	fmt.Println("🧪 Testing Golem AIML Integration for Telegram Bot")
	fmt.Println("=================================================")
	fmt.Println()

	// Create Golem instance
	g := golem.New(true)

	// Load test AIML data
	aimlPath := "testdata"
	if _, err := os.Stat(aimlPath); os.IsNotExist(err) {
		fmt.Printf("❌ Test AIML path '%s' not found. Please ensure testdata directory exists.\n", aimlPath)
		return
	}

	err := g.Execute("load", []string{aimlPath})
	if err != nil {
		log.Fatalf("❌ Failed to load AIML: %v", err)
	}

	fmt.Printf("✅ Loaded AIML from: %s\n", aimlPath)
	fmt.Println()

	// Create a session (simulating a Telegram chat)
	session := g.CreateSession("test_telegram_chat")

	// Test various inputs that a Telegram bot might receive
	testInputs := []string{
		"Hello",
		"What is your name?",
		"How are you?",
		"Tell me a joke",
		"What can you do?",
		"Goodbye",
	}

	fmt.Println("🤖 Testing Bot Responses:")
	fmt.Println("========================")

	for i, input := range testInputs {
		fmt.Printf("%d. User: %s\n", i+1, input)

		response, err := g.ProcessInput(input, session)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   Bot: %s\n", response)
		}
		fmt.Println()
	}

	// Test session information
	fmt.Println("📊 Session Information:")
	fmt.Println("=====================")
	fmt.Printf("Session ID: %s\n", session.ID)
	fmt.Printf("Message Count: %d\n", len(session.History))
	fmt.Printf("Topic: %s\n", session.Topic)
	fmt.Printf("Variables: %v\n", session.Variables)
	fmt.Println()

	// Test command-like inputs
	fmt.Println("🔧 Testing Command-like Inputs:")
	fmt.Println("===============================")

	commandTests := []string{
		"/start",
		"/help",
		"/status",
		"/session",
		"/clear",
	}

	for i, cmd := range commandTests {
		fmt.Printf("%d. Command: %s\n", i+1, cmd)

		// In a real Telegram bot, these would be handled by the command handler
		// Here we just show how they would be processed
		fmt.Printf("   (Would be handled by command handler in Telegram bot)\n")
		fmt.Println()
	}

	fmt.Println("✅ Integration test completed!")
	fmt.Println()
	fmt.Println("🚀 To run the actual Telegram bot:")
	fmt.Println("1. Get a bot token from @BotFather on Telegram")
	fmt.Println("2. Set environment variables:")
	fmt.Println("   export TELEGRAM_BOT_TOKEN='your_token_here'")
	fmt.Println("   export AIML_PATH='testdata'")
	fmt.Println("3. Run: go run examples/telegram_bot.go")
	fmt.Println()
	fmt.Println("📖 For more information, see examples/TELEGRAM_BOT_README.md")
}
