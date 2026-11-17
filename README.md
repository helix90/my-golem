# Golem - AIML2 Engine for Go

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![AIML2 Compliance](https://img.shields.io/badge/AIML2-73%25-orange.svg)](AIML2_COMPARISON.md)

Golem is a comprehensive Go library and CLI tool for building conversational AI applications using the AIML2 (Artificial Intelligence Markup Language) specification. It provides both a powerful library for integration into Go applications and a command-line interface for interactive development and testing.

## 🚀 Features

### Core AIML2 Support
- **Pattern Matching**: Advanced pattern matching with wildcards (`*`, `_`) and normalization
- **Tree-Based Processing**: Revolutionary AST-based template processing eliminating tag-in-tag bugs
- **Template Processing**: Full template processing with recursive substitution (`<srai>`)
- **Context Awareness**: Enhanced `<that>` tag support with indexed access to previous responses
- **Topic Management**: Topic-based conversation control
- **Variable Management**: Session, global, and bot variables with scope resolution
- **Learning System**: Dynamic learning with `<learn>` and `<learnf>` tags

### Advanced Features
- **Data Structures**: Complete list and array operations with CRUD functionality
- **External Integration**: `<sraix>` for HTTP/HTTPS service integration
- **Out-of-Band (OOB)**: Custom command handling for external systems
- **Multi-Session Support**: Concurrent chat sessions with isolated state
- **Pronoun Substitution**: `<person>` and `<gender>` tags for natural conversation
- **Conditional Logic**: `<condition>` tags with variable testing
- **Random Responses**: `<random>` and `<li>` for varied responses
- **Date/Time**: `<date>` and `<time>` formatting
- **Maps and Sets**: Key-value mapping and set operations
- **Text Processing**: `<uppercase>`, `<lowercase>`, `<sentence>`, `<word>` tags
- **Enhanced That Support**: Advanced `<that>` pattern matching with debugging tools
- **Pattern Conflict Detection**: Comprehensive analysis for pattern conflicts and optimization

### CLI Tool
- **Interactive Mode**: Persistent state across commands
- **File Loading**: Load AIML files and directories
- **Session Management**: Create, list, switch, and delete sessions
- **Property Management**: View and set bot properties
- **OOB Management**: Register and test custom handlers

## 📦 Installation

### Prerequisites
- Go 1.21 or higher
- Git

### Install from Source
```bash
git clone https://github.com/helix90/golem.git
cd golem
go build -o golem ./cmd/golem
```

### Install as Go Module
```bash
go get github.com/helix90/golem/pkg/golem
```

## 🛠️ Quick Start

### CLI Usage

#### Interactive Mode
```bash
./golem interactive
golem> load examples/sample.aiml
golem> chat hello
golem> session create
golem> quit
```

#### Single Commands
```bash
# Load AIML file
./golem load examples/sample.aiml

# Chat with loaded knowledge base
./golem chat "hello world"

# Create a new session
./golem session create

# Show bot properties
./golem properties
```

### Library Usage

#### Basic Example
```go
package main

import (
    "fmt"
    "log"
    "github.com/helix90/golem/pkg/golem"
)

func main() {
    // Create a new Golem instance
    g := golem.New(true) // Enable verbose logging
    
    // Load AIML knowledge base
    err := g.Execute("load", []string{"examples/sample.aiml"})
    if err != nil {
        log.Fatal(err)
    }
    
    // Create a chat session
    session := g.CreateSession("user123")
    
    // Process user input
    response, err := g.ProcessInput("Hello!", session)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Bot:", response)
}
```

#### Advanced Example with Custom AIML
```go
package main

import (
    "fmt"
    "github.com/helix90/golem/pkg/golem"
)

func main() {
    g := golem.New(false)
    
    // Create custom AIML knowledge base
    kb := golem.NewAIMLKnowledgeBase()
    kb.Categories = []golem.Category{
        {
            Pattern:  "HELLO",
            Template: "Hello! How can I help you today?",
        },
        {
            Pattern: "MY NAME IS *",
            Template: "Nice to meet you, <star/>! I'm Golem.",
        },
        {
            Pattern: "WHAT IS YOUR NAME",
            Template: "My name is Golem, and I'm an AIML2 bot.",
        },
    }
    
    // Index patterns
    for i := range kb.Categories {
        category := &kb.Categories[i]
        pattern := golem.NormalizePattern(category.Pattern)
        kb.Patterns[pattern] = category
    }
    
    g.SetKnowledgeBase(kb)
    
    // Create session and chat
    session := g.CreateSession("demo")
    
    inputs := []string{
        "Hello!",
        "My name is Alice",
        "What is your name?",
    }
    
    for _, input := range inputs {
        response, _ := g.ProcessInput(input, session)
        fmt.Printf("User: %s\nBot: %s\n\n", input, response)
    }
}
```

## 📚 Examples

The `examples-module/` directory contains comprehensive examples:

### Basic Examples
- **`library_usage.go`** - Basic library usage patterns
- **`learn_demo.go`** - Dynamic learning capabilities
- **`bot_tag_demo.go`** - Bot property access

### Advanced Examples
- **`telegram_bot.go`** - Complete Telegram bot integration
- **`sraix_demo.go`** - External service integration
- **`list_demo.go`** - List and array operations
- **`person_tag_demo.go`** - Pronoun substitution
- **`gender_tag_demo.go`** - Gender-based substitution

### Running Examples
```bash
# Basic learning demo
cd examples-module
go run learn_demo.go

# Telegram bot (requires TELEGRAM_BOT_TOKEN)
export TELEGRAM_BOT_TOKEN="your_token_here"
go run telegram_bot.go

# List operations demo
go run list_demo.go
```

## 🏗️ Architecture

### Core Components
- **`Golem`** - Main engine class with session management
- **`AIMLKnowledgeBase`** - Pattern matching and category management
- **`ChatSession`** - Session state and conversation history
- **`Category`** - Individual AIML patterns and templates

### Key Features
- **Pattern Indexing**: Efficient pattern matching with priority-based selection
- **Tree-Based Processing**: AST-based template processing with 95% tag coverage
- **Template Processing**: Recursive template processing with tag support
- **Session Isolation**: Independent conversation contexts
- **Learning System**: Dynamic knowledge base modification
- **OOB Handling**: Extensible command processing

## 🌳 Tree-Based Processing System

Golem features a revolutionary **tree-based processing system** that eliminates the tag-in-tag bugs common in regex-based AIML processors:

### Key Benefits
- **Eliminates Tag-in-Tag Bugs**: AST parsing prevents nested tag processing issues
- **95% Tag Coverage**: Comprehensive support for all major AIML tags
- **Performance Optimization**: Significantly faster than regex-based processing
- **Robust Parsing**: Handles complex nested structures correctly
- **Whitespace Preservation**: Maintains proper text formatting

### Supported Tags
- **Text Processing**: `<uppercase>`, `<lowercase>`, `<formal>`, `<capitalize>`, `<explode>`, `<reverse>`, `<acronym>`, `<trim>`
- **Variables**: `<set>`, `<get>`, `<bot>`, `<star>`, `<that>`, `<topic>`
- **Control Flow**: `<srai>`, `<sraix>`, `<think>`, `<condition>`, `<random>`, `<li>`
- **System Info**: `<size>`, `<version>`, `<id>` (client ID), `<botid>` (bot ID), `<request>`, `<response>`
- **Data Structures**: `<map>`, `<list>`, `<array>`, `<set>`, `<first>`, `<rest>`
- **Text Analysis**: `<sentence>`, `<word>`, `<person>`, `<person2>`, `<gender>`
- **Learning**: `<learn>`, `<unlearn>`, `<unlearnf>`
- **RDF Operations**: `<subj>`, `<pred>`, `<obj>`, `<uniq>`

### Usage
```go
// Tree-based processing is the only template processor
g := golem.New(true)

// Process templates with tree-based system
response, err := g.ProcessInput("Hello <uppercase>world</uppercase>!", session)
// Result: "Hello WORLD!"
```

## 🔍 That Pattern Conflict Detection

Golem includes a comprehensive **That pattern conflict detection system** to help identify and resolve issues with AIML that patterns:

### Conflict Detection Types
- **Overlap Conflicts**: Detect patterns with overlapping matching scope
- **Ambiguity Conflicts**: Identify patterns that create ambiguous matching scenarios
- **Priority Conflicts**: Find patterns with unclear priority ordering
- **Wildcard Conflicts**: Detect conflicting wildcard usage patterns
- **Specificity Conflicts**: Identify patterns with conflicting specificity levels

### Usage Example
```go
package main

import (
    "fmt"
    "github.com/helix90/golem/pkg/golem"
)

func main() {
    // Define patterns to analyze
    patterns := []string{
        "HELLO",
        "HELLO WORLD",
        "* HELLO",
        "GOOD MORNING",
        "GOOD *",
    }
    
    // Create conflict detector
    detector := golem.NewThatPatternConflictDetector(patterns)
    conflicts := detector.DetectConflicts()
    
    // Analyze detected conflicts
    for _, conflict := range conflicts {
        fmt.Printf("Conflict Type: %s\n", conflict.Type)
        fmt.Printf("Severity: %s\n", conflict.Severity)
        fmt.Printf("Description: %s\n", conflict.Description)
        fmt.Printf("Suggestions: %v\n", conflict.Suggestions)
        fmt.Println("---")
    }
}
```

### Advanced Features
- **Pattern Specificity Analysis**: Calculate pattern specificity (0.0-1.0 scale)
- **Wildcard Usage Analysis**: Count and analyze wildcard patterns
- **Overlap Percentage Calculation**: Quantify pattern overlap with severity levels
- **Intelligent Suggestions**: Tailored recommendations for conflict resolution
- **Example Generation**: Real-world examples that trigger conflicts

### Demo System
```bash
# Run conflict detection demo
cd pkg/golem
go run conflict_demo.go
```

## 🧪 Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test ./...

# Run specific test categories
go test ./pkg/golem -run TestThatTag
go test ./pkg/golem -run TestLearning
go test ./pkg/golem -run TestSRAIX
go test ./pkg/golem -run TestThatPatternConflictDetector

# Run with verbose output
go test ./pkg/golem -v
```

## 📋 AIML2 Compliance

Golem implements **85% of the AIML2 specification** with revolutionary tree-based processing, including:

### ✅ Fully Implemented
- Core AIML elements (`<aiml>`, `<category>`, `<pattern>`, `<template>`)
- **Tree-based template processing** with AST parsing
- Pattern matching with wildcards and normalization
- Template processing with recursive substitution
- Variable management (session, global, bot properties)
- Learning system (`<learn>`, `<learnf>`, `<unlearn>`, `<unlearnf>`)
- Data structures (lists, arrays, maps, sets)
- Context awareness (`<that>`, `<topic>`)
- External integration (`<sraix>`)
- Out-of-band message handling
- Comprehensive text processing (95% tag coverage)
- Enhanced that pattern matching with debugging tools
- Pattern conflict detection and analysis
- System information tags (`<size>`, `<version>`, `<id>`, `<botid>`)
- RDF operations (`<subj>`, `<pred>`, `<obj>`, `<uniq>`)

### 🔄 Partially Implemented
- Advanced pattern matching
- Enhanced context management

### ❌ Not Yet Implemented
- System command execution (`<system>`)
- JavaScript execution (`<javascript>`)
- Gossip processing (`<gossip>`)

See [AIML2_COMPARISON.md](AIML2_COMPARISON.md) for detailed compliance information.

## 🐳 Docker Support

Golem includes comprehensive Docker support:

```bash
# Build Docker image
docker build -t golem .

# Run interactive mode
docker run -it golem interactive

# Run with custom AIML files
docker run -v /path/to/aiml:/aiml -it golem load /aiml/sample.aiml
```

## 🤝 Contributing

Contributions are welcome! Please see our contributing guidelines:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- AIML2 specification for the conversational AI standard
- Go community for excellent tooling and libraries
- Contributors and users who help improve Golem

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/helix90/golem/issues)
- **Discussions**: [GitHub Discussions](https://github.com/helix90/golem/discussions)
- **Documentation**: [Wiki](https://github.com/helix90/golem/wiki)

---

**Golem** - Building intelligent conversations with Go and AIML2 🚀
