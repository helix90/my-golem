# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

My-Golem is a comprehensive AIML2 (Artificial Intelligence Markup Language) engine written in Go. It provides both a library for building conversational AI applications and a CLI tool for interactive development. The project implements ~85% of the AIML2 specification with a revolutionary tree-based processing system using Abstract Syntax Trees (AST).

## Development Commands

### Building
```bash
# Build the main binary
make build
# or
go build -o build/golem .

# Build for multiple platforms
make build-all

# Install to GOPATH/bin
make install
```

### Testing
```bash
# Run all tests
make test
# or
go test ./pkg/golem

# Run with verbose output
go test ./pkg/golem -v

# Run specific test categories
go test ./pkg/golem -run TestTreeProcessor
go test ./pkg/golem -run TestThatTag
go test ./pkg/golem -run TestLearning
go test ./pkg/golem -run TestSRAIX

# Run single test
go test ./pkg/golem -run TestTreeProcessor_ProcessTemplate_BasicTags -v

# Test coverage
make test-coverage
# or
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Code Quality
```bash
# Format code
make fmt
# or
go fmt ./...

# Run linter
make lint
# or
golangci-lint run

# Install dependencies
make deps
# or
go mod tidy && go mod download
```

### Running Examples
```bash
# Run library usage example
go run examples/library_usage.go

# Run demos from examples-module
cd examples-module
go run learn_demo.go
go run bot_tag_demo.go
go run list_demo.go

# Telegram bot (requires TELEGRAM_BOT_TOKEN)
export TELEGRAM_BOT_TOKEN="your_token_here"
go run examples-module/telegram_bot.go
```

### CLI Usage
```bash
# Interactive mode (maintains state across commands)
./build/golem interactive
golem> load testdata/sample.aiml
golem> chat hello
golem> session create
golem> quit

# Single-command mode (creates new instance per command - state lost)
./build/golem load testdata/sample.aiml
./build/golem chat "hello world"
```

## Architecture Overview

### Critical State Management Pattern

**IMPORTANT**: Golem maintains state (knowledge base, sessions, variables) across operations. Understanding the three usage modes is critical:

1. **Single-Command Mode (CLI)**: Creates new Golem instance per command → state lost between commands
2. **Interactive Mode (CLI)**: Single persistent instance → state preserved across commands
3. **Library Mode**: User controls instance lifecycle → state preserved as long as instance exists

See `ARCHITECTURE.md` for detailed explanation of state persistence patterns.

### Core Components

**Main Engine (`pkg/golem/golem.go`)**:
- `Golem` struct - Central engine with session management and knowledge base
- State-bearing fields: `aimlKB`, `sessions`, `currentID`
- Key methods: `Execute()`, `ProcessInput()`, `CreateSession()`, `SetKnowledgeBase()`, `GetKnowledgeBase()`

**AST-Based Processing Pipeline** (The Revolutionary Feature):
- `ast_parser.go` - Parses AIML templates into Abstract Syntax Trees
- `tree_processor.go` - Processes AST nodes for tag evaluation
- **Eliminates tag-in-tag bugs** by parsing structure before processing
- **95% tag coverage** vs 60% with regex-based processors
- **Performance**: 50-70% faster than regex-based processing

**Template Processing**:
- `template_processor.go` - Legacy regex-based processing (fallback)
- `consolidated_template_processor.go` - Unified processing interface
- `aiml_native.go` - Native AIML processing with full pipeline
- Standardized processing order documented in `TAG_PROCESSING_STANDARDIZATION.md`

**Pattern Matching (`pattern_matching.go`)**:
- Priority-based pattern selection (exact > wildcards)
- Wildcard support: `*` (match 0+ words), `_` (match 1+ words)
- Pattern normalization and caching
- That pattern matching with context history

**Session Management (`session_management.go`)**:
- `ChatSession` struct with isolated state per user
- History tracking: requests, responses, that history
- Variable scopes: session, global, bot properties
- Topic management

**Knowledge Base (`aiml_loader.go`)**:
- `AIMLKnowledgeBase` - Stores patterns and categories
- `Category` - Individual pattern/template pairs
- Pattern indexing for efficient matching
- Support for .aiml, .map, .set, .properties files

**Context Resolution (`context_resolution.go`)**:
- Variable context management
- Wildcard resolution
- That history with indexed access
- Enhanced context with analytics and compression

### Package Organization

- **`/pkg/golem/`** - Main library (104 files: core engine, AST parser, tree processor, feature implementations, 73+ test files)
- **`/pkg/golem/processors/`** - Modular processors (12 files with separate go.mod)
- **`/cmd/golem/`** - CLI with flag parsing and interactive mode
- **`/examples/`** - Basic library usage, Telegram bot integration
- **`/examples-module/`** - Comprehensive demos with documentation
- **`/testdata/`** - Primary test data (sample.aiml, bot.properties, loader tests)

### Configuration File Formats

**Properties Files (.properties)**:
All `.properties` files must use JSON array format:
```json
[
  ["key1", "value1"],
  ["key2", "value2"]
]
```

**NOT** Java properties format (`key=value`) or flat JSON objects. Keys starting with underscore (e.g., `"_comment"`) are ignored and can be used for documentation.

**Bot Properties** (`bot.properties`):
- Contains bot identity, personality, capabilities, and settings
- Automatically loaded from directories via `LoadPropertiesFromDirectory()`
- Merged into `AIMLKnowledgeBase.Properties`

**SRAIX Configuration** (e.g., `weather-config.properties`, `sraix-config-example.properties`):
- Configure external SRAIX services via properties
- Property format: `sraix.servicename.property`
- Available properties: `baseurl`, `urltemplate`, `method`, `timeout`, `responseformat`, `responsepath`, `fallback`, `includewildcards`, `header.<HeaderName>`, `apikey`
- URL template placeholders:
  - `${ENV_VAR}` - Environment variables (e.g., `${PIRATE_WEATHER_API_KEY}`)
  - `{input}` - The SRAIX input text
  - `{apikey}` - API key from headers (Authorization header)
  - `{lat}`, `{lon}` - Automatically from session variables `latitude`/`longitude`
  - `{location}` - Location name from wildcards
  - `{WILDCARD_NAME}` - Any wildcard value in uppercase
- Automatically configured via `ConfigureFromProperties()` when knowledge base is set
- Example: `export PIRATE_WEATHER_API_KEY="your-key"` then use `${PIRATE_WEATHER_API_KEY}` in URL templates

**Loading Files**:
```go
// Load all files from directory (AIML, maps, sets, properties)
g.LoadAIMLFromDirectory("testdata")

// Properties are automatically loaded and merged
// SRAIX services are automatically configured from properties
```

## Key Implementation Patterns

### Tree Processing System (THE CRITICAL FEATURE)

This is Golem's main innovation. When working with tag processing:

**Enable/Disable**:
```go
g := golem.New(true)  // Tree processing is now DEFAULT
// g.EnableTreeProcessing()  // Already enabled by default
g.DisableTreeProcessing()  // Opt-out to use legacy regex (not recommended)
```

**How It Works**:
1. Template → AST Parser (`ast_parser.go`) → AST (tree of nodes)
2. AST → Tree Processor (`tree_processor.go`) → Evaluated result
3. Each tag type has dedicated processing in `processTag()` and `processSelfClosingTag()`

**Supported Tags**: uppercase, lowercase, formal, capitalize, explode, reverse, acronym, trim, set, get, bot, star, that, topic, srai, sraix, think, condition, random, li, size, version, id, request, response, map, list, array, set, first, rest, sentence, word, person, person2, gender, learn, unlearn, unlearnf, subj, pred, obj, uniq (see `TREE_PROCESSING_MIGRATION.md` for AST architecture details)

### Tag Processing Pipeline

All template processing follows standardized order (see `TAG_PROCESSING_STANDARDIZATION.md`):

1. Wildcard replacement (indexed star tags, then generic `<star/>`)
2. SR tags → Property tags → Bot tags
3. **EARLY**: Think tags → Topic setting → Set tags
4. Session variables → SRAI → SRAIX
5. Learn tags → Condition tags
6. Date/time → Random tags
7. Collections (map, list, array)
8. Text processing (person, gender, sentence, word)
9. String operations (uppercase, lowercase, formal, etc.)
10. Request/response history tags

**Critical**: `processTemplateContentForVariable()` uses the same pipeline as `processTemplateWithContext()` to ensure consistency.

### Testing Strategy

Test naming conventions: `*_test.go` (unit), `*_enhanced_test.go` (advanced), `*_integration_test.go` (integration), `tree_processor_*_test.go` (AST), `*_cache_test.go` (performance)

### Adding New AIML Tags

When implementing a new tag:

1. **Add to AST Parser** (`ast_parser.go`):
   - Update tag recognition in `Parse()`
   - Handle self-closing vs paired tags

2. **Add to Tree Processor** (`tree_processor.go`):
   - Add case in `processTag()` or `processSelfClosingTag()`
   - Implement tag logic with access to `tp.golem` and `tp.ctx`

3. **Add Tests**:
   - Create `tree_processor_<tagname>_test.go` with unit tests
   - Create `tree_processor_<tagname>_integration_test.go` for complex scenarios
   - Add test cases to `testdata/sample.aiml`

4. **Update Documentation**:
   - Update tag list in README.md
   - Create implementation guide if complex (see `*_TAG_IMPLEMENTATION.md` files)
   - Update `AIML2_COMPARISON.md` compliance matrix

5. **Follow Processing Order**:
   - Ensure tag is processed at correct stage in pipeline
   - Update `TAG_PROCESSING_STANDARDIZATION.md` if adding new processing stage

### Variable Context Pattern

All tag processing receives `VariableContext`:

```go
type VariableContext struct {
    Session       *ChatSession
    GlobalVars    map[string]string
    BotProperties map[string]string
    Wildcards     map[string]string
}
```

**Access in Tree Processor**:
- `tp.ctx.Session` - Current chat session
- `tp.ctx.Session.Variables` - Session variables
- `tp.ctx.GlobalVars` - Global variables
- `tp.ctx.BotProperties` - Bot properties
- `tp.ctx.Wildcards` - Pattern wildcards

### Learning System

Tags `<learn>`, `<learnf>`, `<unlearn>`, `<unlearnf>` modify the knowledge base in-memory and persist to `learned_categories/` directory. All sessions share the updated knowledge base.

## Debugging

Enable verbose logging:
```go
g := golem.New(true)  // Enable verbose logging
```

Check processing mode:
```go
if g.IsTreeProcessingEnabled() {
    // AST processing active
} else {
    // Regex processing active
}
```

Run debugger demos:
```bash
go run pkg/golem/debugger_demo.go
go run pkg/golem/conflict_demo.go
```

## Important Files

- `ARCHITECTURE.md` - Critical state management patterns
- `TREE_PROCESSING_MIGRATION.md` - AST architecture details
- `TAG_PROCESSING_STANDARDIZATION.md` - Processing pipeline specification
- `README.md` - User-facing documentation
- `AIML2_COMPARISON.md` - Compliance matrix
- `*_TAG_IMPLEMENTATION.md` files - Tag-specific implementation guides

## Performance

Multiple caching layers: regex compilation, pattern matching, template processing, normalization, variable resolution. Reuse Golem instances (don't create per request). **Tree-based AST processing is the default** (50-70% faster than regex). Session history default max: 20 items.

## AIML2 Compliance

**Fully Implemented**: Core AIML elements, tree-based processing, pattern matching, wildcards, template processing, variables (all scopes), learning system, data structures (lists/arrays/maps/sets), context awareness (that/topic), external integration (sraix), OOB handling, comprehensive text processing (95% tags), system info tags, RDF operations

**Partially Implemented**: Advanced pattern matching, enhanced context management

**Not Implemented**: System command execution (`<system>`), JavaScript execution (`<javascript>`), Gossip processing (`<gossip>`)

## Recent Changes (from git log)

- Convert string operation tags to native AST/Tree Processor
- Convert `<srai>` and `<sraix>` tags to native AST/Tree Processor
- Convert collection tags (`<map>`, `<list>`, `<array>`) to native AST/Tree Processor
- Convert `<think>` tag to native AST/Tree Processor
- Convert `<word>` tag to native AST/Tree Processor

The project is actively migrating all tag processing to the AST-based system for consistency and performance.
