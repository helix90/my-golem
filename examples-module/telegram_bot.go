//go:build ignore

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/helix90/my-golem/pkg/golem"
)

// TelegramBot represents a Telegram bot integrated with Golem AIML engine
type TelegramBot struct {
	golem     *golem.Golem
	bot       *bot.Bot
	users     *golem.UserStore
	reminders *golem.ReminderStore
	sessions  map[int64]*golem.ChatSession // Chat ID -> Session mapping
	// sessionsMu guards sessions, since handlers run concurrently
	sessionsMu sync.Mutex
	aimlPath   string
	verbose    bool

	// buttonsMu guards postbacks and replies
	buttonsMu sync.Mutex
	// postbacks maps the callback data of inline buttons, which Telegram
	// limits to 64 bytes, to the input each button sends
	postbacks     map[string]string
	postbackOrder []string
	// replies maps, per chat, the text of the suggested replies last shown
	// to the input each sends
	replies map[int64]map[string]string
}

// maxPostbacks bounds how many inline buttons stay usable
const maxPostbacks = 5000

// Config is read from the environment by main
type Config struct {
	Token         string
	AIMLPath      string
	Verbose       bool
	UserStorePath string
	UserStoreKey  string
	RemindersPath string
	APIServerURL  string // for tests; empty uses Telegram's
}

// NewTelegramBot creates a new Telegram bot instance
func NewTelegramBot(cfg Config) (*TelegramBot, error) {
	g := golem.New(cfg.Verbose)

	users, err := golem.OpenUserStore(cfg.UserStorePath, cfg.UserStoreKey)
	if err != nil {
		return nil, fmt.Errorf("failed to open user store: %v", err)
	}
	g.SetUserStore(users)
	if !users.Encrypted() {
		log.Printf("⚠️  USER_STORE_KEY is not set: saved passwords are kept unencrypted in %s", cfg.UserStorePath)
	}

	reminders, err := golem.OpenReminderStore(cfg.RemindersPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open reminders: %v", err)
	}
	g.SetReminderStore(reminders)

	tb := &TelegramBot{
		golem:     g,
		users:     users,
		reminders: reminders,
		sessions:  make(map[int64]*golem.ChatSession),
		aimlPath:  cfg.AIMLPath,
		verbose:   cfg.Verbose,
		postbacks: make(map[string]string),
		replies:   make(map[int64]map[string]string),
	}
	if cfg.AIMLPath != "" {
		if err := tb.loadAIML(); err != nil {
			return nil, err
		}
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(tb.handleOther),
		bot.WithCallbackQueryDataHandler("", bot.MatchTypePrefix, tb.handleCallback),
	}
	if cfg.APIServerURL != "" {
		opts = append(opts, bot.WithServerURL(cfg.APIServerURL), bot.WithSkipGetMe())
	}
	telegramBot, err := bot.New(cfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %v", err)
	}
	tb.bot = telegramBot
	return tb, nil
}

// loadAIML loads the knowledge base and moves any user data the earlier
// persistence AIML kept in learned categories into the user store
func (tb *TelegramBot) loadAIML() error {
	if err := tb.golem.Execute("load", []string{tb.aimlPath}); err != nil {
		return fmt.Errorf("failed to load AIML: %v", err)
	}
	moved, err := tb.golem.MigrateLegacyUserData()
	if err != nil {
		return fmt.Errorf("failed to move saved user data to the user store: %v", err)
	}
	if moved > 0 {
		log.Printf("🔒 Moved %d saved user values from learned categories to the user store", moved)
	}
	return nil
}

// getOrCreateSession gets an existing session or creates a new one for the chat
func (tb *TelegramBot) getOrCreateSession(chatID int64) *golem.ChatSession {
	tb.sessionsMu.Lock()
	defer tb.sessionsMu.Unlock()

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

// sessionCount returns the number of active sessions
func (tb *TelegramBot) sessionCount() int {
	tb.sessionsMu.Lock()
	defer tb.sessionsMu.Unlock()
	return len(tb.sessions)
}

// respond answers input from a user in a chat and sends the answer
func (tb *TelegramBot) respond(ctx context.Context, chatID, userID int64, input string) {
	session := tb.getOrCreateSession(chatID)

	// Identify the sender: the user store keeps what the bot remembers about
	// them, and reminders go back to this chat. In a group the session is
	// shared, so this is whoever sent the current message.
	session.WithLock(func() {
		if userID != 0 {
			session.UserID = strconv.FormatInt(userID, 10)
			session.Variables["telegram_user"] = session.UserID
		}
		session.Variables["telegram_chat"] = strconv.FormatInt(chatID, 10)
	})

	if tb.verbose {
		log.Printf("Chat %d: User said: %s", chatID, redactCredentials(input))
	}

	response, err := tb.golem.ProcessInput(input, session)
	if err != nil {
		log.Printf("Error processing input for chat %d: %v", chatID, err)
		response = "Sorry, I encountered an error processing your message. Please try again."
	}
	if credentialInputRe.MatchString(input) {
		scrubHistory(session, input)
	}
	tb.send(ctx, chatID, response)
}

// scrubHistory replaces input that carries a password wherever the session
// keeps past input, so /session and <input>/<request> never show it
func scrubHistory(session *golem.ChatSession, input string) {
	redacted := redactCredentials(input)
	password := strings.Fields(input)[len(strings.Fields(input))-1]
	session.WithLock(func() {
		for _, history := range []*[]string{&session.History, &session.RequestHistory} {
			for i, entry := range *history {
				if strings.Contains(entry, password) {
					(*history)[i] = strings.ReplaceAll(entry, input, redacted)
					(*history)[i] = strings.ReplaceAll((*history)[i], password, "********")
				}
			}
		}
	})
}

// send sends a response, turning its rich media into Telegram buttons
func (tb *TelegramBot) send(ctx context.Context, chatID int64, response string) {
	rich := golem.ParseRichResponse(response)
	text := rich.Text
	if text == "" {
		if !rich.HasMedia() {
			return
		}
		text = "Choose one:"
	}

	params := &bot.SendMessageParams{ChatID: chatID, Text: text}
	if len(rich.Buttons) > 0 {
		params.ReplyMarkup = tb.inlineKeyboard(rich.Buttons)
	} else if len(rich.Replies) > 0 {
		params.ReplyMarkup = tb.replyKeyboard(chatID, rich.Replies)
	}
	if _, err := tb.bot.SendMessage(ctx, params); err != nil {
		log.Printf("Failed to send message to chat %d: %v", chatID, err)
		return
	}
	// A message carries one keyboard; suggested replies alongside buttons
	// follow in a second message
	if len(rich.Buttons) > 0 && len(rich.Replies) > 0 {
		if _, err := tb.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID, Text: "Or choose:", ReplyMarkup: tb.replyKeyboard(chatID, rich.Replies),
		}); err != nil {
			log.Printf("Failed to send replies to chat %d: %v", chatID, err)
		}
	}
	if tb.verbose {
		log.Printf("Chat %d: Bot replied: %s", chatID, text)
	}
}

// inlineKeyboard makes the buttons of a message. A few short buttons, such
// as Yes and No, share a row; otherwise each has its own.
func (tb *TelegramBot) inlineKeyboard(buttons []golem.RichButton) *models.InlineKeyboardMarkup {
	oneRow := len(buttons) <= 3
	for _, b := range buttons {
		if len([]rune(b.Text)) > 18 {
			oneRow = false
		}
	}
	var rows [][]models.InlineKeyboardButton
	for _, b := range buttons {
		button := models.InlineKeyboardButton{Text: b.Text}
		if b.URL != "" {
			button.URL = b.URL
		} else {
			button.CallbackData = tb.rememberPostback(b.Postback)
		}
		if oneRow && len(rows) == 1 {
			rows[0] = append(rows[0], button)
		} else {
			rows = append(rows, []models.InlineKeyboardButton{button})
		}
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// replyKeyboard makes the suggested replies of a message, and remembers what
// each sends, since Telegram sends a reply's text
func (tb *TelegramBot) replyKeyboard(chatID int64, replies []golem.RichButton) *models.ReplyKeyboardMarkup {
	sends := make(map[string]string)
	var row []models.KeyboardButton
	for _, r := range replies {
		row = append(row, models.KeyboardButton{Text: r.Text, RequestLocation: r.RequestLocation})
		if !r.RequestLocation {
			sends[r.Text] = r.Postback
		}
	}
	tb.buttonsMu.Lock()
	tb.replies[chatID] = sends
	tb.buttonsMu.Unlock()
	return &models.ReplyKeyboardMarkup{Keyboard: [][]models.KeyboardButton{row}, ResizeKeyboard: true, OneTimeKeyboard: true}
}

// rememberPostback returns callback data that stands for a postback
func (tb *TelegramBot) rememberPostback(postback string) string {
	id := make([]byte, 8)
	_, _ = rand.Read(id)
	key := hex.EncodeToString(id)
	tb.buttonsMu.Lock()
	defer tb.buttonsMu.Unlock()
	tb.postbacks[key] = postback
	tb.postbackOrder = append(tb.postbackOrder, key)
	if len(tb.postbackOrder) > maxPostbacks {
		delete(tb.postbacks, tb.postbackOrder[0])
		tb.postbackOrder = tb.postbackOrder[1:]
	}
	return key
}

// replyInput returns what a suggested reply sends, if text is one shown in
// the chat
func (tb *TelegramBot) replyInput(chatID int64, text string) (string, bool) {
	tb.buttonsMu.Lock()
	defer tb.buttonsMu.Unlock()
	postback, ok := tb.replies[chatID][text]
	if ok {
		delete(tb.replies, chatID)
	}
	return postback, ok
}

// credentialInputRe matches input that carries a password
var credentialInputRe = regexp.MustCompile(`(?i)^\s*(?:(?:save|store|remember)\s+(?:my\s+)?list\s+credentials|list\s+login)\s+\S+\s+\S+`)

// redactCredentials hides the password in input that carries one, for logs
func redactCredentials(input string) string {
	if credentialInputRe.MatchString(input) {
		fields := strings.Fields(input)
		fields[len(fields)-1] = "********"
		return strings.Join(fields, " ")
	}
	return input
}

// handleMessage processes incoming messages
func (tb *TelegramBot) handleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	msg := update.Message
	if msg.Location != nil {
		tb.handleLocation(ctx, msg)
		return
	}
	input := strings.TrimSpace(msg.Text)
	if input == "" {
		return
	}
	var userID int64
	if msg.From != nil {
		userID = msg.From.ID
	}
	if postback, ok := tb.replyInput(msg.Chat.ID, input); ok && postback != "" {
		input = postback
	}

	tb.respond(ctx, msg.Chat.ID, userID, input)

	// Don't leave a password in the chat history
	if credentialInputRe.MatchString(input) {
		if _, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: msg.ID}); err != nil {
			log.Printf("Could not delete a message with a password in chat %d: %v", msg.Chat.ID, err)
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "🔒 I deleted your message with the password, so it doesn't stay in this chat."})
		}
	}
}

// handleOther handles updates no other handler takes, such as a shared location
func (tb *TelegramBot) handleOther(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil && update.Message.Location != nil {
		tb.handleLocation(ctx, update.Message)
	}
}

// handleLocation saves a location the user shared, through the AIML
func (tb *TelegramBot) handleLocation(ctx context.Context, msg *models.Message) {
	var userID int64
	if msg.From != nil {
		userID = msg.From.ID
	}
	input := fmt.Sprintf("MY COORDINATES ARE %.4f %.4f", msg.Location.Latitude, msg.Location.Longitude)
	tb.respond(ctx, msg.Chat.ID, userID, input)
}

// handleCallback handles a tap on an inline button
func (tb *TelegramBot) handleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	query := update.CallbackQuery
	if query == nil {
		return
	}
	tb.buttonsMu.Lock()
	postback, ok := tb.postbacks[query.Data]
	tb.buttonsMu.Unlock()

	answer := &bot.AnswerCallbackQueryParams{CallbackQueryID: query.ID}
	if !ok {
		answer.Text = "That button has expired. Please ask again."
	}
	if _, err := b.AnswerCallbackQuery(ctx, answer); err != nil {
		log.Printf("Failed to answer a button tap: %v", err)
	}
	var chatID int64
	var messageID int
	switch {
	case query.Message.Message != nil:
		chatID, messageID = query.Message.Message.Chat.ID, query.Message.Message.ID
	case query.Message.InaccessibleMessage != nil:
		chatID, messageID = query.Message.InaccessibleMessage.Chat.ID, query.Message.InaccessibleMessage.MessageID
	}
	if !ok || chatID == 0 {
		return
	}

	// Take the buttons off the message, so a question is answered once
	if _, err := b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID: chatID, MessageID: messageID, ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}},
	}); err != nil && tb.verbose {
		log.Printf("Could not remove the buttons of message %d: %v", messageID, err)
	}
	tb.respond(ctx, chatID, query.From.ID, postback)
}

// deliverReminder sends a reminder when it is due
func (tb *TelegramBot) deliverReminder(r golem.Reminder) {
	ctx := context.Background()
	chatID, err := strconv.ParseInt(r.Chat, 10, 64)
	if err != nil {
		log.Printf("Reminder %s has an invalid chat %q", r.ID, r.Chat)
		return
	}
	if r.Input != "" {
		userID, _ := strconv.ParseInt(r.User, 10, 64)
		tb.respond(ctx, chatID, userID, r.Input)
		return
	}
	tb.send(ctx, chatID, "⏰ Reminder: "+r.Text)
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

	reply := func(text string) {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
			log.Printf("Failed to send /%s reply: %v", command, err)
		}
	}

	switch command {
	case "start":
		reply(`🤖 Welcome to the Golem AIML Bot!

I'm powered by the Golem AIML engine and can have conversations with you using artificial intelligence.

Commands:
/start - Show this welcome message
/help - Show help information
/status - Show bot status
/reload - Reload AIML knowledge base
/session - Show session information
/clear - Clear conversation history
/forgetme - Delete everything I remember about you

Just send me a message to start chatting!`)

	case "help":
		reply(`📚 Help Information

This bot uses the Golem AIML engine to process your messages and generate intelligent responses.

Available commands:
/start - Welcome message
/help - This help message
/status - Bot status and statistics
/reload - Reload the AIML knowledge base
/session - Show current session information
/clear - Clear your conversation history
/forgetme - Delete everything I remember about you, and your reminders

Try "remind me to call mom at 5pm", "send me a daily briefing at 7am", or "what's the weather".
The bot maintains conversation context, so you can have ongoing discussions. Each chat has its own session.`)

	case "status":
		session := tb.getOrCreateSession(chatID)
		var statusMsg string
		session.WithLock(func() {
			statusMsg = fmt.Sprintf(`📊 Bot Status

🤖 AIML Engine: Golem
📁 Knowledge Base: %s
💬 Active Sessions: %d
📝 Your Messages: %d
🕒 Session Created: %s
🕒 Last Activity: %s

The bot is running and ready to chat!`,
				tb.aimlPath,
				tb.sessionCount(),
				len(session.History),
				session.CreatedAt,
				session.LastActivity)
		})
		reply(statusMsg)

	case "reload":
		if tb.aimlPath == "" {
			reply("❌ No AIML path configured for reloading.")
			return
		}
		if err := tb.loadAIML(); err != nil {
			reply(fmt.Sprintf("❌ Failed to reload AIML: %v", err))
			return
		}
		reply("✅ AIML knowledge base reloaded successfully!")

	case "session":
		session := tb.getOrCreateSession(chatID)
		var sessionMsg string
		session.WithLock(func() {
			sessionMsg = fmt.Sprintf(`📋 Session Information

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
		})
		reply(sessionMsg)

	case "clear":
		session := tb.getOrCreateSession(chatID)
		session.WithLock(func() {
			session.History = []string{}
			session.ThatHistory = []string{}
			session.Variables = make(map[string]string)
			session.Topic = ""
		})
		reply("🧹 Conversation history cleared! Starting fresh. (What I remember about you is kept; /forgetme deletes it.)")

	case "forgetme":
		if update.Message.From == nil {
			return
		}
		user := strconv.FormatInt(update.Message.From.ID, 10)
		if err := tb.users.DeleteUser(user); err != nil {
			log.Printf("Failed to delete user %s: %v", user, err)
			reply("❌ Sorry, I couldn't delete your data. Please try again.")
			return
		}
		deleted := 0
		for _, r := range tb.reminders.List(user, true) {
			if ok, _ := tb.reminders.Delete(r.ID); ok {
				deleted++
			}
		}
		session := tb.getOrCreateSession(chatID)
		session.WithLock(func() {
			for name := range session.Variables {
				if tb.users.IsPersistent(name) {
					delete(session.Variables, name)
				}
			}
			delete(session.Variables, "list_access_token")
		})
		reply(fmt.Sprintf("🗑️ Done. I've deleted everything I remembered about you and %d reminder(s).", deleted))

	default:
		reply("❓ Unknown command. Use /help to see available commands.")
	}
}

// getLastResponse gets the last bot response from session history
func (tb *TelegramBot) getLastResponse(session *golem.ChatSession) string {
	if len(session.ThatHistory) == 0 {
		return "None"
	}
	return session.ThatHistory[len(session.ThatHistory)-1]
}

// formatSessionVariables formats session variables for display, hiding
// passwords, tokens and other secrets
func (tb *TelegramBot) formatSessionVariables(session *golem.ChatSession) string {
	if len(session.Variables) == 0 {
		return "None"
	}

	variables := tb.users.RedactedVariables(session.Variables)
	names := make([]string, 0, len(variables))
	for name := range variables {
		names = append(names, name)
	}
	sort.Strings(names)
	vars := make([]string, len(names))
	for i, name := range names {
		vars[i] = fmt.Sprintf("• %s = %s", name, variables[name])
	}
	return strings.Join(vars, "\n")
}

// Start starts the Telegram bot
func (tb *TelegramBot) Start(ctx context.Context) error {
	// Set up command handler (messages starting with /)
	tb.bot.RegisterHandler(bot.HandlerTypeMessageText, "/", bot.MatchTypePrefix, tb.handleCommand)

	// Set up message handler (all other text messages)
	tb.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypeContains, tb.handleMessage)

	// Deliver reminders and daily briefings when they are due
	go tb.reminders.Run(ctx, 20*time.Second, tb.deliverReminder)

	log.Printf("🤖 Starting Golem Telegram Bot...")
	log.Printf("📁 AIML Path: %s", tb.aimlPath)
	log.Printf("🔧 Verbose: %v", tb.verbose)

	// Start the bot
	tb.bot.Start(ctx)
	return nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
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

	cfg := Config{
		Token:         token,
		AIMLPath:      absPath,
		Verbose:       os.Getenv("VERBOSE") == "true",
		UserStorePath: envOr("USER_STORE_PATH", "data/users.json"),
		UserStoreKey:  os.Getenv("USER_STORE_KEY"),
		RemindersPath: envOr("REMINDERS_PATH", "data/reminders.json"),
		APIServerURL:  os.Getenv("TELEGRAM_API_URL"),
	}

	// Create and start the bot
	telegramBot, err := NewTelegramBot(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create Telegram bot: %v", err)
	}

	// Stop cleanly on Ctrl-C or docker stop
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start the bot
	log.Printf("🚀 Bot is starting...")
	if err := telegramBot.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start bot: %v", err)
	}
	log.Printf("👋 Bot stopped")
}
