package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/helix90/my-golem/pkg/golem"
)

// TelegramBot represents a Telegram bot integrated with Golem AIML engine
type TelegramBot struct {
	golem    *golem.Golem
	bot      *bot.Bot
	sessions map[int64]*golem.ChatSession // Chat ID -> Session mapping
	aimlPath string
	verbose  bool
}

// NewTelegramBot creates a new Telegram bot instance
func NewTelegramBot(token, aimlPath string, verbose bool) (*TelegramBot, error) {
	// Create Golem instance
	g := golem.New(verbose)

	// Load AIML knowledge base
	if aimlPath != "" {
		err := g.Execute("load", []string{aimlPath})
		if err != nil {
			return nil, fmt.Errorf("failed to load AIML: %v", err)
		}
	}

	// Create bot options - no default handler needed, we register handlers explicitly
	opts := []bot.Option{}

	// Create Telegram bot
	telegramBot, err := bot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %v", err)
	}

	return &TelegramBot{
		golem:    g,
		bot:      telegramBot,
		sessions: make(map[int64]*golem.ChatSession),
		aimlPath: aimlPath,
		verbose:  verbose,
	}, nil
}

// getOrCreateSession gets an existing session or creates a new one for the chat
func (tb *TelegramBot) getOrCreateSession(chatID int64) *golem.ChatSession {
	if session, exists := tb.sessions[chatID]; exists {
		return session
	}

	// Create new session
	sessionID := fmt.Sprintf("telegram_%d", chatID)
	session := tb.golem.CreateSession(sessionID)
	tb.sessions[chatID] = session

	if tb.verbose {
		log.Printf("Created new session for chat %d: %s", chatID, sessionID)
	}

	return session
}

// handleMessage processes incoming messages
func (tb *TelegramBot) handleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	userInput := update.Message.Text

	// Skip empty messages
	if strings.TrimSpace(userInput) == "" {
		return
	}

	// Get or create session for this chat
	session := tb.getOrCreateSession(chatID)

	// Log the interaction
	if tb.verbose {
		log.Printf("Chat %d: User said: %s", chatID, userInput)
	}

	// Process input through Golem AIML engine
	response, err := tb.golem.ProcessInput(userInput, session)
	if err != nil {
		log.Printf("Error processing input for chat %d: %v", chatID, err)
		response = "Sorry, I encountered an error processing your message. Please try again."
	}

	// Send response back to user
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   response,
	})
	if err != nil {
		log.Printf("Failed to send message to chat %d: %v", chatID, err)
	}

	// Log the response
	if tb.verbose {
		log.Printf("Chat %d: Bot replied: %s", chatID, response)
	}
}

// handleCommand processes bot commands
func (tb *TelegramBot) handleCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// Check if message starts with /
	if !strings.HasPrefix(update.Message.Text, "/") {
		return
	}

	chatID := update.Message.Chat.ID
	text := update.Message.Text

	// Parse command and arguments
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}

	command := strings.TrimPrefix(parts[0], "/")
	// args := strings.Join(parts[1:], " ") // Available for future use

	switch command {
	case "start":
		welcomeMsg := `🤖 Welcome to the Golem AIML Bot!

I'm powered by the Golem AIML engine and can have conversations with you using artificial intelligence.

Commands:
/start - Show this welcome message
/help - Show help information
/status - Show bot status
/reload - Reload AIML knowledge base
/session - Show session information
/clear - Clear conversation history

Just send me a message to start chatting!`

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   welcomeMsg,
		})
		if err != nil {
			log.Printf("Failed to send welcome message: %v", err)
		}

	case "help":
		helpMsg := `📚 Help Information

This bot uses the Golem AIML engine to process your messages and generate intelligent responses.

Available commands:
/start - Welcome message
/help - This help message
/status - Bot status and statistics
/reload - Reload the AIML knowledge base
/session - Show current session information
/clear - Clear your conversation history

The bot maintains conversation context, so you can have ongoing discussions. Each chat has its own session.`

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   helpMsg,
		})
		if err != nil {
			log.Printf("Failed to send help message: %v", err)
		}

	case "status":
		session := tb.getOrCreateSession(chatID)
		statusMsg := fmt.Sprintf(`📊 Bot Status

🤖 AIML Engine: Golem
📁 Knowledge Base: %s
💬 Active Sessions: %d
📝 Your Messages: %d
🕒 Session Created: %s
🕒 Last Activity: %s

The bot is running and ready to chat!`,
			tb.aimlPath,
			len(tb.sessions),
			len(session.History),
			session.CreatedAt,
			session.LastActivity)

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   statusMsg,
		})
		if err != nil {
			log.Printf("Failed to send status message: %v", err)
		}

	case "reload":
		if tb.aimlPath == "" {
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "❌ No AIML path configured for reloading.",
			})
			if err != nil {
				log.Printf("Failed to send reload error message: %v", err)
			}
			return
		}

		err := tb.golem.Execute("load", []string{tb.aimlPath})
		if err != nil {
			errorMsg := fmt.Sprintf("❌ Failed to reload AIML: %v", err)
			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   errorMsg,
			})
			if err != nil {
				log.Printf("Failed to send reload error message: %v", err)
			}
			return
		}

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "✅ AIML knowledge base reloaded successfully!",
		})
		if err != nil {
			log.Printf("Failed to send reload success message: %v", err)
		}

	case "session":
		session := tb.getOrCreateSession(chatID)
		sessionMsg := fmt.Sprintf(`📋 Session Information

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
			tb.getLastResponse(session),
			session.CreatedAt,
			session.LastActivity,
			tb.formatSessionVariables(session))

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   sessionMsg,
		})
		if err != nil {
			log.Printf("Failed to send session message: %v", err)
		}

	case "clear":
		session := tb.getOrCreateSession(chatID)
		session.History = []string{}
		session.ThatHistory = []string{}
		session.Variables = make(map[string]string)
		session.Topic = ""

		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "🧹 Conversation history cleared! Starting fresh.",
		})
		if err != nil {
			log.Printf("Failed to send clear message: %v", err)
		}

	default:
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❓ Unknown command. Use /help to see available commands.",
		})
		if err != nil {
			log.Printf("Failed to send unknown command message: %v", err)
		}
	}
}

// getLastResponse gets the last bot response from session history
func (tb *TelegramBot) getLastResponse(session *golem.ChatSession) string {
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
func (tb *TelegramBot) formatSessionVariables(session *golem.ChatSession) string {
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

	return strings.Join(vars, "\n")
}

// Start starts the Telegram bot
func (tb *TelegramBot) Start(ctx context.Context) error {
	// Set up command handler (messages starting with /)
	tb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/", bot.MatchTypePrefix, tb.handleCommand)

	// Set up message handler (all other text messages)
	tb.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypeContains, tb.handleMessage)

	log.Printf("🤖 Starting Golem Telegram Bot...")
	log.Printf("📁 AIML Path: %s", tb.aimlPath)
	log.Printf("🔧 Verbose: %v", tb.verbose)

	// Start the bot
	tb.bot.Start(ctx)
	return nil
}

func main() {
	// Get configuration from environment variables
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN environment variable is required")
	}

	aimlPath := os.Getenv("AIML_PATH")
	if aimlPath == "" {
		// Default to testdata directory if available
		if _, err := os.Stat("testdata"); err == nil {
			aimlPath = "testdata"
		} else {
			log.Fatal("❌ AIML_PATH environment variable is required or place AIML files in 'testdata' directory")
		}
	}

	// Check if AIML path exists
	if _, err := os.Stat(aimlPath); os.IsNotExist(err) {
		log.Fatalf("❌ AIML path does not exist: %s", aimlPath)
	}

	// Get absolute path
	absPath, err := filepath.Abs(aimlPath)
	if err != nil {
		log.Fatalf("❌ Failed to get absolute path: %v", err)
	}

	verbose := os.Getenv("VERBOSE") == "true"

	// Create and start the bot
	telegramBot, err := NewTelegramBot(token, absPath, verbose)
	if err != nil {
		log.Fatalf("❌ Failed to create Telegram bot: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the bot
	log.Printf("🚀 Bot is starting...")
	if err := telegramBot.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start bot: %v", err)
	}
}
