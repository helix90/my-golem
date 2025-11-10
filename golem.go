// Package golem provides a Go implementation of an AIML (Artificial Intelligence Markup Language) interpreter.
// This is a convenience package that re-exports the main functionality from pkg/golem.
package golem

import (
	"github.com/helix90/my-golem/pkg/golem"
)

// New creates a new Golem instance with the specified verbosity level.
// This is a convenience function that calls golem.New().
func New(verbose bool) *golem.Golem {
	return golem.New(verbose)
}

// Re-export commonly used types and functions for convenience
type (
	// Golem represents the main AIML interpreter instance
	Golem = golem.Golem

	// ChatSession represents a chat session
	ChatSession = golem.ChatSession

	// VariableContext represents the context for variable resolution
	VariableContext = golem.VariableContext
)

// Re-export commonly used constants
const (
	// Variable scopes
	ScopeGlobal  = golem.ScopeGlobal
	ScopeSession = golem.ScopeSession
	ScopeLocal   = golem.ScopeLocal
)

// Re-export commonly used functions
var (
	// NewAIMLKnowledgeBase creates a new AIML knowledge base
	NewAIMLKnowledgeBase = golem.NewAIMLKnowledgeBase
)
