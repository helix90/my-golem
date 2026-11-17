# Golem Telegram Bot Example

This example demonstrates how to create a Telegram bot using the Golem AIML engine. The bot can have intelligent conversations with users using AIML patterns and templates.

## Features

- 🤖 **Intelligent Conversations**: Powered by the Golem AIML engine
- 💬 **Session Management**: Each chat maintains its own conversation context
- 🔄 **Hot Reloading**: Reload AIML knowledge base without restarting the bot
- 📊 **Status Monitoring**: Built-in commands for bot status and session information
- 🧹 **History Management**: Clear conversation history on demand
- 🔧 **Verbose Logging**: Optional detailed logging for debugging

## Prerequisites

1. **Go 1.19+** installed
2. **Telegram Bot Token** from [@BotFather](https://t.me/botfather)
3. **AIML Knowledge Base** (`.aiml`, `.map`, `.set` files)

## Installation

1. **Install Dependencies**:
   ```bash
   go mod tidy
   go get github.com/go-telegram/bot
   ```

2. **Get Telegram Bot Token**:
   - Message [@BotFather](https://t.me/botfather) on Telegram
   - Use `/newbot` command to create a new bot
   - Follow the instructions to get your bot token

3. **Prepare AIML Files**:
   - Place your AIML files in a directory (e.g., `testdata/`)
   - Include `.aiml`, `.map`, and `.set` files as needed

## Configuration

Set the following environment variables:

```bash
# Required: Your Telegram bot token
export TELEGRAM_BOT_TOKEN="your_bot_token_here"

# Required: Path to AIML files directory
export AIML_PATH="/path/to/your/aiml/files"

# Optional: Enable verbose logging
export VERBOSE="true"
```

## Usage

### Running the Bot

```bash
# Set environment variables
export TELEGRAM_BOT_TOKEN="1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
export AIML_PATH="testdata"
export VERBOSE="true"

# Run the bot
go run examples-module/telegram_bot.go
```

### Bot Commands

Once the bot is running, users can interact with it using these commands:

- `/start` - Welcome message and introduction
- `/help` - Show help information and available commands
- `/status` - Display bot status and statistics
- `/reload` - Reload the AIML knowledge base
- `/session` - Show current session information
- `/clear` - Clear conversation history

### Example Conversation

```
User: /start
Bot: 🤖 Welcome to the Golem AIML Bot!
     I'm powered by the Golem AIML engine and can have conversations with you using artificial intelligence.
     ...

User: Hello
Bot: Hello! How can I help you?

User: What is your name?
Bot: My name is Golem.

User: /status
Bot: 📊 Bot Status
     🤖 AIML Engine: Golem
     📁 Knowledge Base: /path/to/aiml/files
     💬 Active Sessions: 1
     📝 Your Messages: 2
     ...
```

## Code Structure

### Main Components

1. **TelegramBot Struct**: Main bot controller
   - Manages Golem AIML engine
   - Handles session management
   - Processes messages and commands

2. **Session Management**: 
   - Each chat gets its own session
   - Maintains conversation history
   - Stores user variables and context

3. **Message Processing**:
   - Routes commands to command handler
   - Processes regular messages through AIML engine
   - Sends responses back to users

### Key Methods

- `NewTelegramBot()`: Creates a new bot instance
- `handleMessage()`: Processes incoming messages
- `handleCommand()`: Handles bot commands
- `getOrCreateSession()`: Manages chat sessions
- `Start()`: Starts the bot

## Customization

### Adding New Commands

To add new bot commands, extend the `handleCommand()` method:

```go
case "mycommand":
    // Handle your custom command
    _, err := b.SendMessage(ctx, &bot.SendMessageParams{
        ChatID: chatID,
        Text:   "Response to my command",
    })
```

### Customizing AIML Processing

You can modify the message processing in `handleMessage()`:

```go
// Add preprocessing
userInput = strings.ToUpper(userInput)

// Process through Golem
response, err := tb.golem.ProcessInput(userInput, session)

// Add postprocessing
response = strings.TrimSpace(response)
```

### Session Management

The bot maintains separate sessions for each chat:

```go
// Access session for a specific chat
session := tb.getOrCreateSession(chatID)

// Access session variables
session.Variables["custom_var"] = "value"

// Access conversation history
for i, message := range session.History {
    // Even indices: user messages
    // Odd indices: bot responses
}
```

## Error Handling

The bot includes comprehensive error handling:

- **AIML Processing Errors**: Graceful fallback messages
- **Telegram API Errors**: Logged and handled appropriately
- **Session Errors**: Automatic session recovery
- **Configuration Errors**: Clear error messages on startup

## Logging

Enable verbose logging to see detailed information:

```bash
export VERBOSE="true"
```

Verbose mode shows:
- Message processing details
- Session creation and management
- AIML engine operations
- Error details

## Deployment

### Local Development

```bash
# Clone the repository
git clone <repository-url>
cd golem

# Install dependencies
go mod tidy

# Set environment variables
export TELEGRAM_BOT_TOKEN="your_token"
export AIML_PATH="testdata"

# Run the bot
go run examples-module/telegram_bot.go
```

### Production Deployment

1. **Build the binary**:
   ```bash
   go build -o telegram-bot examples-module/telegram_bot.go
   ```

2. **Set up environment variables**:
   ```bash
   export TELEGRAM_BOT_TOKEN="your_production_token"
   export AIML_PATH="/path/to/production/aiml"
   export VERBOSE="false"
   ```

3. **Run as a service**:
   ```bash
   ./telegram-bot
   ```

### Docker Deployment

Create a `Dockerfile`:

```dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o telegram-bot examples-module/telegram_bot.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/telegram-bot .
COPY --from=builder /app/testdata ./testdata
CMD ["./telegram-bot"]
```

Build and run:

```bash
docker build -t golem-telegram-bot .
docker run -e TELEGRAM_BOT_TOKEN="your_token" golem-telegram-bot
```

## Troubleshooting

### Common Issues

1. **Bot Token Invalid**:
   - Verify token from @BotFather
   - Check environment variable is set correctly

2. **AIML Files Not Loading**:
   - Check AIML_PATH is correct
   - Verify files exist and are readable
   - Check file permissions

3. **Bot Not Responding**:
   - Check verbose logging for errors
   - Verify internet connection
   - Check Telegram API status

4. **Session Issues**:
   - Use `/clear` command to reset session
   - Check session management code

### Debug Mode

Enable verbose logging for detailed debugging:

```bash
export VERBOSE="true"
go run examples-module/telegram_bot.go
```

## Contributing

To contribute to this example:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

This example is part of the Golem AIML engine project and follows the same license terms.

## Support

For issues and questions:

1. Check the troubleshooting section
2. Review the verbose logs
3. Open an issue on the repository
4. Check the Golem AIML engine documentation

---

**Happy Bot Building! 🤖✨**
