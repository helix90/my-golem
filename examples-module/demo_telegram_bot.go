package main

import (
	"fmt"
	"log"
	"os"

	"github.com/helix90/my-golem/pkg/golem"
)

// DemoTelegramBot simulates a Telegram bot for testing purposes
type DemoTelegramBot struct {
	golem    *golem.Golem
	sessions map[string]*golem.ChatSession
	verbose  bool
}

// NewDemoTelegramBot creates a new demo bot instance
func NewDemoTelegramBot(aimlPath string, verbose bool) (*DemoTelegramBot, error) {
	// Create Golem instance
	g := golem.New(verbose)

	// Load AIML knowledge base
	if aimlPath != "" {
		err := g.Execute("load", []string{aimlPath})
		if err != nil {
			return nil, fmt.Errorf("failed to load AIML: %v", err)
		}
	}

	return &DemoTelegramBot{
		golem:    g,
		sessions: make(map[string]*golem.ChatSession),
		verbose:  verbose,
	}, nil
}

// getOrCreateSession gets an existing session or creates a new one
func (dtb *DemoTelegramBot) getOrCreateSession(chatID string) *golem.ChatSession {
	if session, exists := dtb.sessions[chatID]; exists {
		return session
	}

	// Create new session
	sessionID := fmt.Sprintf("demo_%s", chatID)
	session := dtb.golem.CreateSession(sessionID)
	dtb.sessions[chatID] = session

	if dtb.verbose {
		log.Printf("Created new session for chat %s: %s", chatID, sessionID)
	}

	return session
}

// processMessage processes a message and returns the response
func (dtb *DemoTelegramBot) processMessage(chatID, message string) string {
	// Skip empty messages
	if len(message) == 0 {
		return ""
	}

	// Get or create session for this chat
	session := dtb.getOrCreateSession(chatID)

	// Log the interaction
	if dtb.verbose {
		log.Printf("Chat %s: User said: %s", chatID, message)
	}

	// Process input through Golem AIML engine
	response, err := dtb.golem.ProcessInput(message, session)
	if err != nil {
		log.Printf("Error processing input for chat %s: %v", chatID, err)
		return "Sorry, I encountered an error processing your message. Please try again."
	}

	// Log the response
	if dtb.verbose {
		log.Printf("Chat %s: Bot replied: %s", chatID, response)
	}

	return response
}

// handleCommand processes bot commands
func (dtb *DemoTelegramBot) handleCommand(chatID, command, args string) string {
	switch command {
	case "start":
		return `🤖 Welcome to the Golem AIML Bot Demo!

This is a simulation of the Telegram bot. In the real bot, this would be sent via Telegram.

Commands:
/start - Show this welcome message
/help - Show help information
/status - Show bot status
/session - Show session information
/clear - Clear conversation history

Just send me a message to start chatting!`

	case "help":
		return `📚 Help Information

This bot uses the Golem AIML engine to process your messages and generate intelligent responses.

Available commands:
/start - Welcome message
/help - This help message
/status - Bot status and statistics
/session - Show current session information
/clear - Clear your conversation history

The bot maintains conversation context, so you can have ongoing discussions. Each chat has its own session.`

	case "status":
		session := dtb.getOrCreateSession(chatID)
		return fmt.Sprintf(`📊 Bot Status

🤖 AIML Engine: Golem
💬 Active Sessions: %d
📝 Your Messages: %d
🕒 Session Created: %s
🕒 Last Activity: %s

The bot is running and ready to chat!`,
			len(dtb.sessions),
			len(session.History),
			session.CreatedAt,
			session.LastActivity)

	case "session":
		session := dtb.getOrCreateSession(chatID)
		return fmt.Sprintf(`📋 Session Information

🆔 Session ID: %s
💬 Message Count: %d
🎯 Topic: %s
📝 Last Response: %s
🕒 Created: %s
🕒 Last Activity: %s

Session variables:
%s`,
			session.ID,
			len(session.History),
			session.Topic,
			dtb.getLastResponse(session),
			session.CreatedAt,
			session.LastActivity,
			dtb.formatSessionVariables(session))

	case "clear":
		session := dtb.getOrCreateSession(chatID)
		session.History = []string{}
		session.ThatHistory = []string{}
		session.Variables = make(map[string]string)
		session.Topic = ""
		return "🧹 Conversation history cleared! Starting fresh."

	default:
		return "❓ Unknown command. Use /help to see available commands."
	}
}

// getLastResponse gets the last bot response from session history
func (dtb *DemoTelegramBot) getLastResponse(session *golem.ChatSession) string {
	if len(session.History) == 0 {
		return "None"
	}

	// Find the last bot response (even indices are user messages, odd are bot responses)
	for i := len(session.History) - 1; i >= 0; i-- {
		if i%2 == 1 { // Bot response
			return session.History[i]
		}
	}
	return "None"
}

// formatSessionVariables formats session variables for display
func (dtb *DemoTelegramBot) formatSessionVariables(session *golem.ChatSession) string {
	if len(session.Variables) == 0 {
		return "None"
	}

	var vars []string
	for key, value := range session.Variables {
		vars = append(vars, fmt.Sprintf("• %s = %s", key, value))
	}

	if len(vars) == 0 {
		return "None"
	}

	return fmt.Sprintf("%s", vars[0]) // Simplified for demo
}

func main() {
	// Get configuration
	aimlPath := os.Getenv("AIML_PATH")
	if aimlPath == "" {
		// Default to testdata directory if available
		if _, err := os.Stat("testdata"); err == nil {
			aimlPath = "testdata"
		} else {
			log.Fatal("❌ AIML_PATH environment variable is required or place AIML files in 'testdata' directory")
		}
	}

	verbose := os.Getenv("VERBOSE") == "true"

	// Create demo bot
	bot, err := NewDemoTelegramBot(aimlPath, verbose)
	if err != nil {
		log.Fatalf("❌ Failed to create demo bot: %v", err)
	}

	fmt.Println("🤖 Golem AIML Bot Demo")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Printf("📁 AIML Path: %s\n", aimlPath)
	fmt.Printf("🔧 Verbose: %v\n", verbose)
	fmt.Println()
	fmt.Println("This is a simulation of the Telegram bot.")
	fmt.Println("Type 'quit' to exit, or use /help for commands.")
	fmt.Println()

	chatID := "demo_chat"

	// Interactive loop
	for {
		fmt.Print("You: ")
		var input string
		fmt.Scanln(&input)

		if input == "quit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		// Check if it's a command
		if len(input) > 0 && input[0] == '/' {
			parts := []string{input[1:]} // Remove the /
			command := parts[0]
			args := ""
			if len(parts) > 1 {
				args = parts[1]
			}
			response := bot.handleCommand(chatID, command, args)
			fmt.Printf("Bot: %s\n", response)
		} else {
			// Regular message
			response := bot.processMessage(chatID, input)
			fmt.Printf("Bot: %s\n", response)
		}
		fmt.Println()
	}
}
