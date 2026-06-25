package golem

import "testing"

// NewForTesting creates a new Golem instance for testing with isolated persistent learning
// It uses the test's temporary directory to ensure test isolation and automatic cleanup
func NewForTesting(t *testing.T, verbose bool) *Golem {
	g := New(verbose)
	g.persistentLearning = NewPersistentLearningManager(t.TempDir())
	return g
}

// NewForTestingWithDir creates a new Golem instance using a specific storage directory.
// Use this when multiple instances in the same test must share persistent learned categories
// (e.g. to simulate a bot restart while keeping the same on-disk state).
func NewForTestingWithDir(_ *testing.T, verbose bool, dir string) *Golem {
	g := New(verbose)
	g.persistentLearning = NewPersistentLearningManager(dir)
	return g
}
