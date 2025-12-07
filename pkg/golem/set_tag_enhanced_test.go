package golem

import (
	"strings"
	"testing"
)

// TestSetTagEnhancedBasicOperations tests basic set tag operations with enhanced coverage
func TestSetTagEnhancedBasicOperations(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	// Test 1: Basic variable assignment (default operation)
	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>SET USERNAME</pattern>
			<template>Hello <set name="username">John</set>, welcome!</template>
		</category>
		<category>
			<pattern>SET AGE</pattern>
			<template>Setting age: <set name="age" operation="assign">25</set></template>
		</category>
		<category>
			<pattern>GET USERNAME</pattern>
			<template>Your username is <get name="username"/></template>
		</category>
		<category>
			<pattern>GET AGE</pattern>
			<template>Your age is <get name="age"/></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test 1: Set username
	response, _ := g.ProcessInput("SET USERNAME", session)
	expected := "Hello , welcome!" // Set returns empty for regex processor compatibility
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify variable was set
	response, _ = g.ProcessInput("GET USERNAME", session)
	expected = "Your username is John"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Set age with explicit assign operation
	response, _ = g.ProcessInput("SET AGE", session)
	expected = "Setting age:" // Set returns empty for regex processor compatibility
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify variable was set
	response, _ = g.ProcessInput("GET AGE", session)
	expected = "Your age is 25"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedAddOperations tests set add/insert operations with enhanced coverage
func TestSetTagEnhancedAddOperations(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>ADD FRUIT</pattern>
			<template>Adding fruit: <set name="fruits" operation="add">apple</set></template>
		</category>
		<category>
			<pattern>ADD BANANA</pattern>
			<template>Adding more fruits: <set name="fruits" operation="add">banana</set></template>
		</category>
		<category>
			<pattern>ADD DUPLICATE</pattern>
			<template>Adding duplicate: <set name="fruits" operation="add">apple</set></template>
		</category>
		<category>
			<pattern>INSERT ORANGE</pattern>
			<template>Inserting: <set name="fruits" operation="insert">orange</set></template>
		</category>
		<category>
			<pattern>SHOW FRUITS</pattern>
			<template>Fruits: <set name="fruits"></set></template>
		</category>
		<category>
			<pattern>FRUIT COUNT</pattern>
			<template>Count: <set name="fruits" operation="size"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test 1: Add single item to set
	response, _ := g.ProcessInput("ADD FRUIT", session)
	expected := "Adding fruit:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify item was added
	setCollection := g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 1 || setCollection.Items[0] != "apple" {
		t.Errorf("Expected set to contain ['apple'], got %v", setCollection)
	}

	// Test 2: Add multiple items
	response, _ = g.ProcessInput("ADD BANANA", session)
	expected = "Adding more fruits:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify both items are in set
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 2 {
		t.Errorf("Expected set to have 2 items, got %d", len(setCollection.Items))
	}

	// Test 3: Add duplicate item (should not add)
	response, _ = g.ProcessInput("ADD DUPLICATE", session)
	expected = "Adding duplicate:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify duplicate was not added
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 2 {
		t.Errorf("Expected set to still have 2 items, got %d", len(setCollection.Items))
	}

	// Test 4: Insert operation (same as add)
	response, _ = g.ProcessInput("INSERT ORANGE", session)
	expected = "Inserting:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify item was inserted
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 3 {
		t.Errorf("Expected set to have 3 items, got %d", len(setCollection.Items))
	}

	// Test 5: Show all fruits
	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Fruits: apple banana orange"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 6: Count fruits
	response, _ = g.ProcessInput("FRUIT COUNT", session)
	expected = "Count: 3"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedRemoveOperations tests set remove/delete operations with enhanced coverage
func TestSetTagEnhancedRemoveOperations(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>SETUP FRUITS</pattern>
			<template>Setting up: <set name="fruits" operation="add">apple</set><set name="fruits" operation="add">banana</set><set name="fruits" operation="add">orange</set></template>
		</category>
		<category>
			<pattern>REMOVE BANANA</pattern>
			<template>Removing fruit: <set name="fruits" operation="remove">banana</set></template>
		</category>
		<category>
			<pattern>REMOVE GRAPE</pattern>
			<template>Removing non-existent: <set name="fruits" operation="remove">grape</set></template>
		</category>
		<category>
			<pattern>DELETE APPLE</pattern>
			<template>Deleting: <set name="fruits" operation="delete">apple</set></template>
		</category>
		<category>
			<pattern>SHOW FRUITS</pattern>
			<template>Fruits: <set name="fruits"></set></template>
		</category>
		<category>
			<pattern>FRUIT COUNT</pattern>
			<template>Count: <set name="fruits" operation="size"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Setup: Add some items to the set
	response, _ := g.ProcessInput("SETUP FRUITS", session)
	expected := "Setting up:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify setup
	setCollection := g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 3 {
		t.Errorf("Expected set to have 3 items after setup, got %d", len(setCollection.Items))
	}

	// Test 1: Remove existing item
	response, _ = g.ProcessInput("REMOVE BANANA", session)
	expected = "Removing fruit:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify item was removed
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 2 {
		t.Errorf("Expected set to have 2 items, got %d", len(setCollection.Items))
	}

	// Test 2: Remove non-existent item
	response, _ = g.ProcessInput("REMOVE GRAPE", session)
	expected = "Removing non-existent:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify set unchanged
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 2 {
		t.Errorf("Expected set to still have 2 items, got %d", len(setCollection.Items))
	}

	// Test 3: Delete operation (same as remove)
	response, _ = g.ProcessInput("DELETE APPLE", session)
	expected = "Deleting:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify item was deleted
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 1 || setCollection.Items[0] != "orange" {
		t.Errorf("Expected set to contain ['orange'], got %v", setCollection.Items)
	}

	// Test 4: Show remaining fruits
	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Fruits: orange"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 5: Count remaining fruits
	response, _ = g.ProcessInput("FRUIT COUNT", session)
	expected = "Count: 1"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedClearOperation tests set clear operation with enhanced coverage
func TestSetTagEnhancedClearOperation(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>SETUP FRUITS</pattern>
			<template>Setting up: <set name="fruits" operation="add">apple</set><set name="fruits" operation="add">banana</set><set name="fruits" operation="add">orange</set></template>
		</category>
		<category>
			<pattern>CLEAR FRUITS</pattern>
			<template>Clearing set: <set name="fruits" operation="clear"></set></template>
		</category>
		<category>
			<pattern>SHOW FRUITS</pattern>
			<template>Fruits: <set name="fruits"></set></template>
		</category>
		<category>
			<pattern>FRUIT COUNT</pattern>
			<template>Count: <set name="fruits" operation="size"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Setup: Add some items to the set
	response, _ := g.ProcessInput("SETUP FRUITS", session)
	expected := "Setting up:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify setup
	setCollection := g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 3 {
		t.Errorf("Expected set to have 3 items after setup, got %d", len(setCollection.Items))
	}

	// Test clear operation
	response, _ = g.ProcessInput("CLEAR FRUITS", session)
	expected = "Clearing set:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify set was cleared
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 0 {
		t.Errorf("Expected set to be empty, got %v", setCollection.Items)
	}

	// Test show empty set
	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Fruits:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test count empty set
	response, _ = g.ProcessInput("FRUIT COUNT", session)
	expected = "Count: 0"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedSizeOperation tests set size/length operations with enhanced coverage
func TestSetTagEnhancedSizeOperation(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>EMPTY SIZE</pattern>
			<template>Set size: <set name="empty" operation="size"></set></template>
		</category>
		<category>
			<pattern>SETUP FRUITS</pattern>
			<template>Setting up: <set name="fruits" operation="add">apple</set><set name="fruits" operation="add">banana</set><set name="fruits" operation="add">orange</set></template>
		</category>
		<category>
			<pattern>FRUIT SIZE</pattern>
			<template>Set size: <set name="fruits" operation="size"></set></template>
		</category>
		<category>
			<pattern>FRUIT LENGTH</pattern>
			<template>Set length: <set name="fruits" operation="length"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test 1: Empty set size
	response, _ := g.ProcessInput("EMPTY SIZE", session)
	expected := "Set size: 0"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Non-empty set size
	response, _ = g.ProcessInput("SETUP FRUITS", session)
	expected = "Setting up:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("FRUIT SIZE", session)
	expected = "Set size: 3"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Length operation (same as size)
	response, _ = g.ProcessInput("FRUIT LENGTH", session)
	expected = "Set length: 3"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedContainsOperation tests set contains/has operations with enhanced coverage
func TestSetTagEnhancedContainsOperation(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>SETUP FRUITS</pattern>
			<template>Setting up: <set name="fruits" operation="add">apple</set><set name="fruits" operation="add">banana</set><set name="fruits" operation="add">orange</set></template>
		</category>
		<category>
			<pattern>CONTAINS APPLE</pattern>
			<template>Contains apple: <set name="fruits" operation="contains">apple</set></template>
		</category>
		<category>
			<pattern>CONTAINS GRAPE</pattern>
			<template>Contains grape: <set name="fruits" operation="contains">grape</set></template>
		</category>
		<category>
			<pattern>HAS BANANA</pattern>
			<template>Has banana: <set name="fruits" operation="has">banana</set></template>
		</category>
		<category>
			<pattern>CONTAINS APPLE UPPER</pattern>
			<template>Contains APPLE: <set name="fruits" operation="contains">APPLE</set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Setup: Add some items to the set
	response, _ := g.ProcessInput("SETUP FRUITS", session)
	expected := "Setting up:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 1: Contains existing item
	response, _ = g.ProcessInput("CONTAINS APPLE", session)
	expected = "Contains apple: true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Contains non-existing item
	response, _ = g.ProcessInput("CONTAINS GRAPE", session)
	expected = "Contains grape: false"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Has operation (same as contains)
	response, _ = g.ProcessInput("HAS BANANA", session)
	expected = "Has banana: true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 4: Case insensitive matching
	response, _ = g.ProcessInput("CONTAINS APPLE UPPER", session)
	expected = "Contains APPLE: true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedGetOperation tests set get/list operations with enhanced coverage
func TestSetTagEnhancedGetOperation(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>EMPTY SET</pattern>
			<template>Set contents: <set name="empty"></set></template>
		</category>
		<category>
			<pattern>SETUP FRUITS</pattern>
			<template>Setting up: <set name="fruits" operation="add">apple</set><set name="fruits" operation="add">banana</set><set name="fruits" operation="add">orange</set></template>
		</category>
		<category>
			<pattern>SHOW FRUITS</pattern>
			<template>Set contents: <set name="fruits"></set></template>
		</category>
		<category>
			<pattern>GET FRUITS</pattern>
			<template>Fruits: <set name="fruits" operation="get"></set></template>
		</category>
		<category>
			<pattern>LIST FRUITS</pattern>
			<template>Fruit list: <set name="fruits" operation="list"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test 1: Get empty set
	response, _ := g.ProcessInput("EMPTY SET", session)
	expected := "Set contents:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 2: Get non-empty set
	response, _ = g.ProcessInput("SETUP FRUITS", session)
	expected = "Setting up:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Set contents: apple banana orange"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Explicit get operation
	response, _ = g.ProcessInput("GET FRUITS", session)
	expected = "Fruits: apple banana orange"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 4: List operation (same as get)
	response, _ = g.ProcessInput("LIST FRUITS", session)
	expected = "Fruit list: apple banana orange"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedWithWildcards tests set operations with wildcards
func TestSetTagEnhancedWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>ADD *</pattern>
			<template>Adding: <set name="fruits" operation="add"><star/></set></template>
		</category>
		<category>
			<pattern>SET ITEM *</pattern>
			<template>Setting: <set name="item"><star/></set></template>
		</category>
		<category>
			<pattern>GET ITEM</pattern>
			<template>Item is <get name="item"/></template>
		</category>
		<category>
			<pattern>SHOW FRUITS</pattern>
			<template>Fruits: <set name="fruits"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test 1: Add wildcard content
	response, _ := g.ProcessInput("ADD apple", session)
	expected := "Adding:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify wildcard was processed and added
	setCollection := g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 1 || setCollection.Items[0] != "apple" {
		t.Errorf("Expected set to contain ['apple'], got %v", setCollection)
	}

	// Test 2: Variable assignment with wildcard
	response, _ = g.ProcessInput("SET ITEM banana", session)
	expected = "Setting:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify variable was set with wildcard content
	response, _ = g.ProcessInput("GET ITEM", session)
	expected = "Item is banana"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test 3: Show fruits with wildcard
	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Fruits: apple"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedEdgeCases tests edge cases for set operations
func TestSetTagEnhancedEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>ADD EMPTY</pattern>
			<template>Adding empty: <set name="fruits" operation="add"></set></template>
		</category>
		<category>
			<pattern>ADD WHITESPACE</pattern>
			<template>Adding whitespace: <set name="fruits" operation="add">   	
   </set></template>
		</category>
		<category>
			<pattern>ADD SPECIAL</pattern>
			<template>Adding special: <set name="fruits-123" operation="add">apple-pie</set></template>
		</category>
		<category>
			<pattern>SHOW FRUITS</pattern>
			<template>Fruits: <set name="fruits"></set></template>
		</category>
		<category>
			<pattern>SHOW SPECIAL</pattern>
			<template>Special: <set name="fruits-123"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test 1: Empty content with add operation
	response, _ := g.ProcessInput("ADD EMPTY", session)
	expected := "Adding empty:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify empty content was not added
	setCollection := g.aimlKB.SetCollections["fruits"]
	if setCollection != nil && len(setCollection.Items) != 0 {
		t.Errorf("Expected empty set, got %v", setCollection.Items)
	}

	// Test 2: Whitespace content
	response, _ = g.ProcessInput("ADD WHITESPACE", session)
	expected = "Adding whitespace:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify whitespace was trimmed and not added
	setCollection = g.aimlKB.SetCollections["fruits"]
	if setCollection != nil && len(setCollection.Items) != 0 {
		t.Errorf("Expected empty set, got %v", setCollection.Items)
	}

	// Debug: Print actual set contents
	if setCollection != nil {
		t.Logf("Set contents after whitespace test: %v", setCollection.Items)
	}

	// Test 3: Special characters in set names and content
	response, _ = g.ProcessInput("ADD SPECIAL", session)
	expected = "Adding special:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify special characters were handled
	setCollection = g.aimlKB.SetCollections["fruits-123"]
	if setCollection == nil || len(setCollection.Items) != 1 || setCollection.Items[0] != "apple-pie" {
		t.Errorf("Expected set to contain ['apple-pie'], got %v", setCollection)
	}

	// Test 4: Show special set
	response, _ = g.ProcessInput("SHOW SPECIAL", session)
	expected = "Special: apple-pie"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedMultipleOperations tests multiple set operations in sequence
func TestSetTagEnhancedMultipleOperations(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>COMPLEX OPERATIONS</pattern>
			<template>
				Adding fruits: <set name="fruits" operation="add">apple</set>
				Adding more: <set name="fruits" operation="add">banana</set>
				Adding more: <set name="fruits" operation="add">orange</set>
				Size: <set name="fruits" operation="size"></set>
				Contains apple: <set name="fruits" operation="contains">apple</set>
				Contents: <set name="fruits"></set>
				Removing: <set name="fruits" operation="remove">banana</set>
				Final size: <set name="fruits" operation="size"></set>
				Final contents: <set name="fruits"></set>
			</template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test sequence of operations
	response, _ := g.ProcessInput("COMPLEX OPERATIONS", session)

	// Check that all operations were processed
	if !strings.Contains(response, "Size: 3") {
		t.Errorf("Expected size 3, got: %s", response)
	}
	if !strings.Contains(response, "Contains apple: true") {
		t.Errorf("Expected contains apple true, got: %s", response)
	}
	if !strings.Contains(response, "Contents: apple banana orange") {
		t.Errorf("Expected contents with all fruits, got: %s", response)
	}
	if !strings.Contains(response, "Final size: 2") {
		t.Errorf("Expected final size 2, got: %s", response)
	}
	if !strings.Contains(response, "Final contents: apple orange") {
		t.Errorf("Expected final contents with apple and orange, got: %s", response)
	}
}

// TestSetTagEnhancedIntegration tests set tag integration with other AIML features
func TestSetTagEnhancedIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>INTEGRATION TEST</pattern>
			<template>
				Hello <set name="name">John</set>!
				Today is <date/> and you have <set name="fruits" operation="add">apple</set> in your basket.
				Your basket contains: <set name="fruits"></set>
				The time is <time/> and you have <set name="fruits" operation="size"></set> fruits.
			</template>
		</category>
		<category>
			<pattern>GET NAME</pattern>
			<template>Your name is <get name="name"/></template>
		</category>
		<category>
			<pattern>SHOW FRUITS</pattern>
			<template>Fruits: <set name="fruits"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test set operations with other tags
	response, _ := g.ProcessInput("INTEGRATION TEST", session)

	// Check that set operations were processed correctly
	if !strings.Contains(response, "Hello !") {
		t.Errorf("Expected name variable to be set, got: %s", response)
	}
	if !strings.Contains(response, "in your basket.") {
		t.Errorf("Expected apple to be added, got: %s", response)
	}
	if !strings.Contains(response, "Your basket contains: apple") {
		t.Errorf("Expected basket contents, got: %s", response)
	}
	if !strings.Contains(response, "you have 1 fruits.") {
		t.Errorf("Expected fruit count, got: %s", response)
	}

	// Verify variable was set
	response, _ = g.ProcessInput("GET NAME", session)
	expected := "Your name is John"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Verify set was populated
	setCollection := g.aimlKB.SetCollections["fruits"]
	if setCollection == nil || len(setCollection.Items) != 1 || setCollection.Items[0] != "apple" {
		t.Errorf("Expected set to contain ['apple'], got %v", setCollection)
	}

	// Test show fruits
	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Fruits: apple"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

// TestSetTagEnhancedPerformance tests performance with many set operations
func TestSetTagEnhancedPerformance(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.CreateSession("test-session")

	aimlContent := `
	<aiml version="2.0">
		<category>
			<pattern>ADD ITEM *</pattern>
			<template>Adding item: <set name="numbers" operation="add">item<star/></set></template>
		</category>
		<category>
			<pattern>SIZE CHECK</pattern>
			<template>Size: <set name="numbers" operation="size"></set></template>
		</category>
		<category>
			<pattern>SHOW NUMBERS</pattern>
			<template>Numbers: <set name="numbers"></set></template>
		</category>
	</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	// Test adding many items
	for i := 0; i < 10; i++ {
		response, _ := g.ProcessInput("ADD ITEM "+string(rune(i+'0')), session)
		expected := "Adding item:"
		if response != expected {
			t.Errorf("Expected '%s', got '%s'", expected, response)
		}
	}

	// Verify all items were added
	setCollection := g.aimlKB.SetCollections["numbers"]
	if setCollection == nil || len(setCollection.Items) != 10 {
		t.Errorf("Expected 10 items, got %d", len(setCollection.Items))
	}

	// Test size operation
	response, _ := g.ProcessInput("SIZE CHECK", session)
	expected := "Size: 10"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test show numbers
	response, _ = g.ProcessInput("SHOW NUMBERS", session)
	if !strings.Contains(response, "Numbers: item") {
		t.Errorf("Expected numbers to be shown, got: %s", response)
	}
}
