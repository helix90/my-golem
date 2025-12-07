package golem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEnhancedLearnfBasicFunctionality tests basic enhanced learnf functionality
func TestEnhancedLearnfBasicFunctionality(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test learnf tag processing with persistent storage
	template := `<learnf>
		<category>
			<pattern>ENHANCED PERSISTENT</pattern>
			<template>This is enhanced persistent learning</template>
		</category>
	</learnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learnf tag should be removed after processing
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// Check if the category was added to knowledge base
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}

	// Check if the pattern was indexed
	normalizedPattern := NormalizePattern("ENHANCED PERSISTENT")
	if _, exists := kb.Patterns[normalizedPattern]; !exists {
		t.Errorf("Learned pattern not indexed: %s", normalizedPattern)
	}

	// Check if persistent storage file was created
	storageFile := filepath.Join(tempDir, "learned_categories.json")
	if _, err := os.Stat(storageFile); os.IsNotExist(err) {
		t.Errorf("Persistent storage file not created: %s", storageFile)
	}
}

// TestEnhancedLearnfPersistence tests persistence across restarts
func TestEnhancedLearnfPersistence(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// First instance - learn a category
	g1 := New(false)
	g1.SetPersistentLearningPath(tempDir)
	kb1 := NewAIMLKnowledgeBase()
	g1.SetKnowledgeBase(kb1)

	template := `<learnf>
		<category>
			<pattern>PERSISTENT ACROSS RESTART</pattern>
			<template>This should persist across restarts</template>
		</category>
	</learnf>`

	_ = g1.ProcessTemplate(template, make(map[string]string))

	// Second instance - load persistent categories
	g2 := New(false)
	g2.SetPersistentLearningPath(tempDir)
	kb2 := NewAIMLKnowledgeBase()
	g2.SetKnowledgeBase(kb2)

	// Load persistent categories
	err := g2.LoadPersistentCategories()
	if err != nil {
		t.Fatalf("Failed to load persistent categories: %v", err)
	}

	// Check if the category was loaded
	if len(kb2.Categories) != 1 {
		t.Errorf("Expected 1 persistent category, got %d", len(kb2.Categories))
	}

	// Check if the pattern was indexed
	normalizedPattern := NormalizePattern("PERSISTENT ACROSS RESTART")
	if _, exists := kb2.Patterns[normalizedPattern]; !exists {
		t.Errorf("Persistent pattern not loaded: %s", normalizedPattern)
	}
}

// TestEnhancedLearnfMultipleCategories tests learning multiple categories persistently
func TestEnhancedLearnfMultipleCategories(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test learning multiple categories
	template := `<learnf>
		<category>
			<pattern>MULTI PERSISTENT 1</pattern>
			<template>First persistent category</template>
		</category>
		<category>
			<pattern>MULTI PERSISTENT 2</pattern>
			<template>Second persistent category</template>
		</category>
		<category>
			<pattern>MULTI PERSISTENT 3</pattern>
			<template>Third persistent category</template>
		</category>
	</learnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learnf tag should be removed after processing
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// Check if all categories were added
	if len(kb.Categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(kb.Categories))
	}

	// Check if all patterns were indexed
	patterns := []string{"MULTI PERSISTENT 1", "MULTI PERSISTENT 2", "MULTI PERSISTENT 3"}
	for _, pattern := range patterns {
		normalizedPattern := NormalizePattern(pattern)
		if _, exists := kb.Patterns[normalizedPattern]; !exists {
			t.Errorf("Pattern not indexed: %s", normalizedPattern)
		}
	}
}

// TestEnhancedLearnfUpdateExisting tests updating existing persistent categories
func TestEnhancedLearnfUpdateExisting(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// First learning
	template1 := `<learnf>
		<category>
			<pattern>UPDATE TEST</pattern>
			<template>Original response</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(template1, make(map[string]string))

	// Check original category
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category after first learning, got %d", len(kb.Categories))
	}

	originalTemplate := kb.Categories[0].Template
	if originalTemplate != "Original response" {
		t.Errorf("Expected 'Original response', got '%s'", originalTemplate)
	}

	// Update the category
	template2 := `<learnf>
		<category>
			<pattern>UPDATE TEST</pattern>
			<template>Updated response</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(template2, make(map[string]string))

	// Check updated category
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category after update, got %d", len(kb.Categories))
	}

	updatedTemplate := kb.Categories[0].Template
	if updatedTemplate != "Updated response" {
		t.Errorf("Expected 'Updated response', got '%s'", updatedTemplate)
	}
}

// TestEnhancedLearnfErrorHandling tests error handling in enhanced learnf
func TestEnhancedLearnfErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test learnf with invalid content
	template := `<learnf>
		<category>
			<pattern>INVALID PATTERN WITH SCRIPT<script>alert('xss')</script></pattern>
			<template>Invalid response</template>
		</category>
	</learnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learnf tag should be removed even on error
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// No categories should be added due to validation failure
	if len(kb.Categories) != 0 {
		t.Errorf("Expected 0 categories due to validation failure, got %d", len(kb.Categories))
	}
}

// TestEnhancedLearnfWithUnlearnf tests learnf/unlearnf cycle
func TestEnhancedLearnfWithUnlearnf(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Learn a category
	learnTemplate := `<learnf>
		<category>
			<pattern>LEARN UNLEARN CYCLE</pattern>
			<template>This will be learned and then unlearned</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(learnTemplate, make(map[string]string))

	// Check if category was learned
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category after learning, got %d", len(kb.Categories))
	}

	// Unlearn the category
	unlearnTemplate := `<unlearnf>
		<category>
			<pattern>LEARN UNLEARN CYCLE</pattern>
			<template>This will be learned and then unlearned</template>
		</category>
	</unlearnf>`

	_ = g.ProcessTemplate(unlearnTemplate, make(map[string]string))

	// Check if category was unlearned
	if len(kb.Categories) != 0 {
		t.Errorf("Expected 0 categories after unlearning, got %d", len(kb.Categories))
	}
}

// TestEnhancedLearnfStorageInfo tests getting storage information
func TestEnhancedLearnfStorageInfo(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Learn some categories
	template := `<learnf>
		<category>
			<pattern>INFO TEST 1</pattern>
			<template>First info test</template>
		</category>
		<category>
			<pattern>INFO TEST 2</pattern>
			<template>Second info test</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(template, make(map[string]string))

	// Get storage info
	info, err := g.GetPersistentLearningInfo()
	if err != nil {
		t.Fatalf("Failed to get persistent learning info: %v", err)
	}

	// Check info
	if info["total_categories"] != 2 {
		t.Errorf("Expected 2 total categories, got %v", info["total_categories"])
	}

	if info["version"] != "1.3.0" {
		t.Errorf("Expected version 1.3.0, got %v", info["version"])
	}

	if info["storage_path"] != tempDir {
		t.Errorf("Expected storage path %s, got %v", tempDir, info["storage_path"])
	}
}

// TestEnhancedLearnfBackupCreation tests backup creation
func TestEnhancedLearnfBackupCreation(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Learn a category
	template := `<learnf>
		<category>
			<pattern>BACKUP TEST</pattern>
			<template>This should create a backup</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(template, make(map[string]string))

	// Check if backup directory was created
	backupDir := filepath.Join(tempDir, "backups")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Errorf("Backup directory not created: %s", backupDir)
	}

	// Check if backup files exist
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "learned_categories_*.json"))
	if err != nil {
		t.Fatalf("Failed to list backup files: %v", err)
	}

	if len(backupFiles) == 0 {
		t.Errorf("No backup files created")
	}
}

// TestEnhancedLearnfAutoSave tests auto-save functionality
func TestEnhancedLearnfAutoSave(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)

	// Configure auto-save with very short interval
	if g.persistentLearning != nil {
		g.persistentLearning.SetAutoSave(true, 100*time.Millisecond)
	}

	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Learn a category
	template := `<learnf>
		<category>
			<pattern>AUTO SAVE TEST</pattern>
			<template>This should trigger auto-save</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(template, make(map[string]string))

	// Wait for auto-save
	time.Sleep(200 * time.Millisecond)

	// Check if storage file was created
	storageFile := filepath.Join(tempDir, "learned_categories.json")
	if _, err := os.Stat(storageFile); os.IsNotExist(err) {
		t.Errorf("Auto-save storage file not created: %s", storageFile)
	}
}

// TestEnhancedLearnfWithWildcards tests learnf with wildcard patterns
func TestEnhancedLearnfWithWildcards(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test learning wildcard patterns
	template := `<learnf>
		<category>
			<pattern>WILDCARD PERSISTENT *</pattern>
			<template>Wildcard persistent response for <star/></template>
		</category>
		<category>
			<pattern>ANOTHER * PATTERN</pattern>
			<template>Another wildcard: <star/></template>
		</category>
	</learnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learnf tag should be removed after processing
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// Check if wildcard categories were added
	if len(kb.Categories) != 2 {
		t.Errorf("Expected 2 wildcard categories, got %d", len(kb.Categories))
	}

	// Check if wildcard patterns were indexed
	wildcardPattern1 := NormalizePattern("WILDCARD PERSISTENT *")
	if _, exists := kb.Patterns[wildcardPattern1]; !exists {
		t.Errorf("Wildcard pattern not indexed: %s", wildcardPattern1)
	}

	wildcardPattern2 := NormalizePattern("ANOTHER * PATTERN")
	if _, exists := kb.Patterns[wildcardPattern2]; !exists {
		t.Errorf("Wildcard pattern not indexed: %s", wildcardPattern2)
	}
}

// TestEnhancedLearnfWithThatContext tests learnf with that context
func TestEnhancedLearnfWithThatContext(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test learning with that context
	template := `<learnf>
		<category>
			<pattern>THAT PERSISTENT</pattern>
			<that>WHAT IS YOUR NAME</that>
			<template>My name is Bot, and I remember that context persistently</template>
		</category>
	</learnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learnf tag should be removed after processing
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// Check if category with that context was added
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category with that context, got %d", len(kb.Categories))
	}

	category := kb.Categories[0]
	if category.That != "WHAT IS YOUR NAME" {
		t.Errorf("Expected that context 'WHAT IS YOUR NAME', got '%s'", category.That)
	}
}

// TestEnhancedLearnfWithTopic tests learnf with topic
func TestEnhancedLearnfWithTopic(t *testing.T) {
	tempDir := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test learning with topic
	template := `<learnf>
		<category>
			<pattern>TOPIC PERSISTENT</pattern>
			<topic>PERSISTENT TOPIC</topic>
			<template>This is about persistent topics</template>
		</category>
	</learnf>`

	result := g.ProcessTemplate(template, make(map[string]string))

	// The learnf tag should be removed after processing
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// Check if category with topic was added
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category with topic, got %d", len(kb.Categories))
	}

	category := kb.Categories[0]
	if category.Topic != "PERSISTENT TOPIC" {
		t.Errorf("Expected topic 'PERSISTENT TOPIC', got '%s'", category.Topic)
	}
}

// TestEnhancedLearnfStoragePathChange tests changing storage path
func TestEnhancedLearnfStoragePathChange(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	g := NewForTesting(t, false)
	g.SetPersistentLearningPath(tempDir1)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Learn a category in first directory
	template := `<learnf>
		<category>
			<pattern>PATH CHANGE TEST</pattern>
			<template>This is in the first directory</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(template, make(map[string]string))

	// Change storage path
	g.SetPersistentLearningPath(tempDir2)

	// Learn another category in second directory
	template2 := `<learnf>
		<category>
			<pattern>PATH CHANGE TEST 2</pattern>
			<template>This is in the second directory</template>
		</category>
	</learnf>`

	_ = g.ProcessTemplate(template2, make(map[string]string))

	// Check if both categories exist in knowledge base
	if len(kb.Categories) != 2 {
		t.Errorf("Expected 2 categories after path change, got %d", len(kb.Categories))
	}

	// Check if storage files exist in both directories
	storageFile1 := filepath.Join(tempDir1, "learned_categories.json")
	storageFile2 := filepath.Join(tempDir2, "learned_categories.json")

	if _, err := os.Stat(storageFile1); os.IsNotExist(err) {
		t.Errorf("Storage file not created in first directory: %s", storageFile1)
	}

	if _, err := os.Stat(storageFile2); os.IsNotExist(err) {
		t.Errorf("Storage file not created in second directory: %s", storageFile2)
	}
}

// TestEnhancedLearnfConcurrentAccess tests concurrent access to persistent storage
func TestEnhancedLearnfConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple instances with same storage path
	g1 := New(false)
	g1.SetPersistentLearningPath(tempDir)
	kb1 := NewAIMLKnowledgeBase()
	g1.SetKnowledgeBase(kb1)

	g2 := New(false)
	g2.SetPersistentLearningPath(tempDir)
	kb2 := NewAIMLKnowledgeBase()
	g2.SetKnowledgeBase(kb2)

	// Learn categories from both instances
	template1 := `<learnf>
		<category>
			<pattern>CONCURRENT 1</pattern>
			<template>From instance 1</template>
		</category>
	</learnf>`

	template2 := `<learnf>
		<category>
			<pattern>CONCURRENT 2</pattern>
			<template>From instance 2</template>
		</category>
	</learnf>`

	_ = g1.ProcessTemplate(template1, make(map[string]string))
	_ = g2.ProcessTemplate(template2, make(map[string]string))

	// Both instances should have their categories
	if len(kb1.Categories) != 1 {
		t.Errorf("Instance 1: Expected 1 category, got %d", len(kb1.Categories))
	}

	if len(kb2.Categories) != 1 {
		t.Errorf("Instance 2: Expected 1 category, got %d", len(kb2.Categories))
	}

	// Check if storage file contains both categories
	storageFile := filepath.Join(tempDir, "learned_categories.json")
	if _, err := os.Stat(storageFile); os.IsNotExist(err) {
		t.Errorf("Storage file not created: %s", storageFile)
	}
}
