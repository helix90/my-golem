package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// CRITICAL ARCHITECTURAL NOTE:
// The CLI creates a SINGLE Golem instance that persists across all commands.
// This is essential for maintaining state between commands (e.g., loaded AIML knowledge base, active sessions).
//
// DO NOT MODIFY this pattern without understanding the implications:
// - Each command execution creates a new Golem instance (current behavior)
// - This means state is lost between commands (AIML knowledge base, sessions, etc.)
// - This is a known limitation of the current CLI design
//
// For persistent state, use the library directly or implement a persistent CLI mode.

func main() {
	var (
		version = flag.Bool("version", false, "Show version information")
		help    = flag.Bool("help", false, "Show help information")
		verbose = flag.Bool("verbose", false, "Enable verbose output")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *version {
		showVersion()
		return
	}

	// Process command line arguments
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("No command specified. Use -help for usage information.")
		os.Exit(1)
	}

	// Check for interactive mode
	if args[0] == "interactive" || args[0] == "i" {
		runInteractiveMode(*verbose)
		return
	}

	// Initialize the golem library for single command execution
	// NOTE: This creates a new instance for each command, so state is not preserved
	g := golem.New(*verbose)

	// Execute the command
	if err := g.Execute(args[0], args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("Golem - A dual-purpose Go library and CLI tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  golem [flags] <command> [arguments]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -help     Show this help message")
	fmt.Println("  -version  Show version information")
	fmt.Println("  -verbose  Enable verbose output")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  interactive Start interactive mode (persistent state)")
	fmt.Println("  load        Load a file (supports subdirectories and AIML files)")
	fmt.Println("  chat        Chat with loaded AIML knowledge base")
	fmt.Println("  session     Manage chat sessions (create, list, switch, delete)")
	fmt.Println("  properties  Show or set bot properties")
	fmt.Println("  oob         Manage Out-of-Band message handlers")
	fmt.Println("  process     Process input data")
	fmt.Println("  analyze     Analyze data")
	fmt.Println("  generate    Generate output")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  golem interactive                    # Start interactive mode")
	fmt.Println("  golem load data/sample.aiml         # Load AIML file")
	fmt.Println("  golem chat hello                    # Chat (requires loaded AIML)")
	fmt.Println("  golem chat '<oob>SYSTEM INFO</oob>'  # Send OOB message")
	fmt.Println("  golem session create                # Create session")
	fmt.Println("  golem oob list                      # List OOB handlers")
	fmt.Println("  golem oob test SYSTEM INFO          # Test OOB handler")
	fmt.Println()
	fmt.Println("Note: Single commands create new instances (state not preserved)")
	fmt.Println("Use 'interactive' mode for persistent state across commands")
}

func showVersion() {
	fmt.Println("Golem v1.5.0")
	fmt.Println("A dual-purpose Go library and CLI tool")
}

// runInteractiveMode starts an interactive session with persistent state
func runInteractiveMode(verbose bool) {
	fmt.Println("Golem Interactive Mode")
	fmt.Println("=====================")
	fmt.Println("Type 'help' for available commands, 'quit' to exit")
	fmt.Println()

	// Create a single persistent Golem instance
	g := golem.New(verbose)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("golem> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "quit" || line == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if line == "help" {
			showInteractiveHelp()
			continue
		}

		// Parse command and arguments
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		// Execute command with persistent state
		if err := g.Execute(command, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

func showInteractiveHelp() {
	fmt.Println("Interactive Mode Commands:")
	fmt.Println("  load <file>           Load AIML file")
	fmt.Println("  chat <message>        Chat with bot")
	fmt.Println("  chat <oob>msg</oob>   Send OOB message")
	fmt.Println("  session create [id]   Create new session")
	fmt.Println("  session list          List all sessions")
	fmt.Println("  session switch <id>   Switch to session")
	fmt.Println("  session delete <id>   Delete session")
	fmt.Println("  session current       Show current session")
	fmt.Println("  properties            Show all properties")
	fmt.Println("  properties <key>      Show specific property")
	fmt.Println("  properties <key> <val> Set property value")
	fmt.Println("  oob list              List OOB handlers")
	fmt.Println("  oob test <message>    Test OOB handler")
	fmt.Println("  oob register <name> <desc> Register custom handler")
	fmt.Println("  help                  Show this help")
	fmt.Println("  quit/exit             Exit interactive mode")
	fmt.Println()
}
