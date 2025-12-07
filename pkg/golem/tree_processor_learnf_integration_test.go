package golem

import (
	"strings"
	"testing"
	"time"
)

// TestTreeProcessorLearnfTagIntegration tests learnf in full conversation flow
func TestTreeProcessorLearnfTagIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Load AIML with learnf teaching patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEACH ME * MEANS *</pattern>
		<template>
			<learnf>
				<category>
					<pattern><star/></pattern>
					<template><star index="2"/></template>
				</category>
			</learnf>
			Okay, I learned that <star/> means <star index="2"/>.
		</template>
	</category>
	
	<category>
		<pattern>TEACH GREETING *</pattern>
		<template>
			<learnf>
				<category>
					<pattern>GREET ME</pattern>
					<template><star/></template>
				</category>
			</learnf>
			I will now greet you with: <star/>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-integration",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	// Teach a new pattern
	t.Run("Teach pattern", func(t *testing.T) {
		response, err := g.ProcessInput("teach me hello means hi there", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "I learned") {
			t.Errorf("Expected learning confirmation, got '%s'", response)
		}
	})

	// Use the learned pattern
	t.Run("Use learned pattern", func(t *testing.T) {
		response, err := g.ProcessInput("hello", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if response != "hi there" {
			t.Errorf("Expected 'hi there', got '%s'", response)
		}
	})

	// Teach a greeting
	t.Run("Teach greeting", func(t *testing.T) {
		response, err := g.ProcessInput("teach greeting Hello, friend!", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "I will now greet you") {
			t.Errorf("Expected teaching confirmation, got '%s'", response)
		}
	})

	// Use the greeting
	t.Run("Use greeting", func(t *testing.T) {
		response, err := g.ProcessInput("greet me", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if response != "Hello, friend!" {
			t.Errorf("Expected 'Hello, friend!', got '%s'", response)
		}
	})
}

// TestTreeProcessorLearnfTagPersistenceIntegration tests persistence across sessions
func TestTreeProcessorLearnfTagPersistenceIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Load AIML with learning capability
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>REMEMBER *</pattern>
		<template>
			<learnf>
				<category>
					<pattern>WHAT DO YOU REMEMBER</pattern>
					<template>I remember: <star/></template>
				</category>
			</learnf>
			I will remember: <star/>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// First session - teach something
	session1 := &ChatSession{
		ID:              "session-1",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	t.Run("Teach in first session", func(t *testing.T) {
		response, err := g.ProcessInput("remember the secret code", session1)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "I will remember") {
			t.Errorf("Expected confirmation, got '%s'", response)
		}
	})

	// Second session - recall should work
	session2 := &ChatSession{
		ID:              "session-2",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	t.Run("Recall in second session", func(t *testing.T) {
		response, err := g.ProcessInput("what do you remember", session2)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "the secret code") {
			t.Errorf("Expected 'the secret code' in response, got '%s'", response)
		}
	})
}

// TestTreeProcessorLearnfTagDynamicIntegration tests dynamic content in learnf
func TestTreeProcessorLearnfTagDynamicIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Load AIML with dynamic learning
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>MY NAME IS *</pattern>
		<template>
			<set name="username"><star/></set>
			<learnf>
				<category>
					<pattern>WHO AM I</pattern>
					<template>You are <get name="username"/></template>
				</category>
			</learnf>
			Nice to meet you, <star/>!
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-dynamic",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	t.Run("Introduce name", func(t *testing.T) {
		response, err := g.ProcessInput("my name is Alice", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "Alice") {
			t.Errorf("Expected name in response, got '%s'", response)
		}
	})

	t.Run("Recall identity", func(t *testing.T) {
		response, err := g.ProcessInput("who am i", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "Alice") {
			t.Errorf("Expected 'Alice' in response, got '%s'", response)
		}
	})
}

// TestTreeProcessorLearnfTagWithVariablesIntegration tests learnf with variable manipulation
func TestTreeProcessorLearnfTagWithVariablesIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Load AIML that teaches patterns using variables
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>DEFINE * AS *</pattern>
		<template>
			<set name="term"><star/></set>
			<set name="definition"><star index="2"/></set>
			<learnf>
				<category>
					<pattern>WHAT IS <uppercase><get name="term"/></uppercase></pattern>
					<template><get name="definition"/></template>
				</category>
			</learnf>
			Defined <star/> as: <star index="2"/>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-variables",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	t.Run("Define term", func(t *testing.T) {
		response, err := g.ProcessInput("define robot as a mechanical being", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "Defined") {
			t.Errorf("Expected definition confirmation, got '%s'", response)
		}
	})

	t.Run("Query definition", func(t *testing.T) {
		response, err := g.ProcessInput("what is robot", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "mechanical being") {
			t.Errorf("Expected 'mechanical being' in response, got '%s'", response)
		}
	})
}

// TestTreeProcessorLearnfTagComplexIntegration tests complex learning scenarios
func TestTreeProcessorLearnfTagComplexIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Load AIML with complex learning
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>TEACH UPPERCASE *</pattern>
		<template>
			<learnf>
				<category>
					<pattern>SHOUT <star/></pattern>
					<template><uppercase><star/></uppercase></template>
				</category>
			</learnf>
			I learned to shout: <star/>
		</template>
	</category>
	
	<category>
		<pattern>TEACH ECHO * TWICE</pattern>
		<template>
			<learnf>
				<category>
					<pattern>DOUBLE <star/></pattern>
					<template><star/> <star/></template>
				</category>
			</learnf>
			I will echo <star/> twice.
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-complex",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	t.Run("Teach uppercase", func(t *testing.T) {
		_, err := g.ProcessInput("teach uppercase hello", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
	})

	t.Run("Use uppercase", func(t *testing.T) {
		response, err := g.ProcessInput("shout hello", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if response != "HELLO" {
			t.Errorf("Expected 'HELLO', got '%s'", response)
		}
	})

	t.Run("Teach doubling", func(t *testing.T) {
		_, err := g.ProcessInput("teach echo world twice", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
	})

	t.Run("Use doubling", func(t *testing.T) {
		response, err := g.ProcessInput("double world", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if response != "world world" {
			t.Errorf("Expected 'world world', got '%s'", response)
		}
	})
}

// TestTreeProcessorLearnfTagWithConditionals tests learnf with conditional logic
func TestTreeProcessorLearnfTagWithConditionals(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Load AIML with conditional learning
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>IF * TEACH *</pattern>
		<template>
			<set name="condition"><star/></set>
			<condition name="condition" value="yes">
				<learnf>
					<category>
						<pattern>CONDITIONAL TEST</pattern>
						<template><star index="2"/></template>
					</category>
				</learnf>
				Taught: <star index="2"/>
			</condition>
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-conditional",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	t.Run("Conditional teach with yes", func(t *testing.T) {
		response, err := g.ProcessInput("if yes teach success", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if !strings.Contains(response, "Taught") {
			t.Errorf("Expected teaching confirmation, got '%s'", response)
		}
	})

	t.Run("Test learned pattern", func(t *testing.T) {
		response, err := g.ProcessInput("conditional test", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		if response != "success" {
			t.Errorf("Expected 'success', got '%s'", response)
		}
	})
}

// TestTreeProcessorLearnfTagErrorHandling tests error handling
func TestTreeProcessorLearnfTagErrorHandling(t *testing.T) {
	g := NewForTesting(t, false)
	g.EnableTreeProcessing()

	// Load AIML that might cause errors
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
	<category>
		<pattern>BAD LEARN</pattern>
		<template>
			<learnf>
				This is not valid AIML
			</learnf>
			Attempted to learn
		</template>
	</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := &ChatSession{
		ID:              "test-error",
		Variables:       make(map[string]string),
		History:         make([]string, 0),
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastActivity:    time.Now().Format(time.RFC3339),
		Topic:           "",
		ThatHistory:     make([]string, 0),
		ResponseHistory: make([]string, 0),
		RequestHistory:  make([]string, 0),
	}

	t.Run("Handle bad learn gracefully", func(t *testing.T) {
		response, err := g.ProcessInput("bad learn", session)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}

		// Should handle error gracefully
		if !strings.Contains(response, "Attempted") {
			t.Errorf("Expected error handling, got '%s'", response)
		}
	})
}
