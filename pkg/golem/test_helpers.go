package golem

import "testing"

// NewForTesting creates a new Golem instance for testing with isolated persistent learning
// It uses the test's temporary directory to ensure test isolation and automatic cleanup
func NewForTesting(t *testing.T, verbose bool) *Golem {
	g := New(verbose)
	// Override the persistent learning manager to use a temporary directory
	// This ensures each test has its own isolated learned categories storage
	g.persistentLearning = NewPersistentLearningManager(t.TempDir())
	return g
}
