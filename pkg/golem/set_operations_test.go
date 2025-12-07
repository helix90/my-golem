package golem

import (
	"strings"
	"testing"
)

func TestSetTagBasicOperations(t *testing.T) {
	g := NewForTesting(t, false)

	// Test basic set operations
	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>ADD TO SET</pattern>
        <template>I'll add that to your set. <set name="fruits" operation="add">apple</set></template>
    </category>
    <category>
        <pattern>SHOW SET</pattern>
        <template>Your set contains: <set name="fruits"></set></template>
    </category>
    <category>
        <pattern>SET SIZE</pattern>
        <template>Your set has <set name="fruits" operation="size"></set> items.</template>
    </category>
    <category>
        <pattern>CHECK CONTAINS</pattern>
        <template>Set contains apple: <set name="fruits" operation="contains">apple</set></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.CreateSession("test-session")

	// Test adding items to set
	response, _ := g.ProcessInput("ADD TO SET", session)
	expected := "I'll add that to your set."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test showing set
	response, _ = g.ProcessInput("SHOW SET", session)
	expected = "Your set contains: apple"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test set size
	response, _ = g.ProcessInput("SET SIZE", session)
	expected = "Your set has 1 items."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test contains check
	response, _ = g.ProcessInput("CHECK CONTAINS", session)
	expected = "Set contains apple: true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestSetTagMultipleOperations(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SETUP SET</pattern>
        <template>
            <set name="colors" operation="add">red</set>
            <set name="colors" operation="add">green</set>
            <set name="colors" operation="add">blue</set>
            Set setup complete.
        </template>
    </category>
    <category>
        <pattern>SHOW COLORS</pattern>
        <template>Colors: <set name="colors"></set></template>
    </category>
    <category>
        <pattern>ADD YELLOW</pattern>
        <template>
            <set name="colors" operation="add">yellow</set>
            Added yellow to the set.
        </template>
    </category>
    <category>
        <pattern>REMOVE GREEN</pattern>
        <template>
            <set name="colors" operation="remove">green</set>
            Removed green from the set.
        </template>
    </category>
    <category>
        <pattern>CLEAR SET</pattern>
        <template>
            <set name="colors" operation="clear"></set>
            Set cleared.
        </template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.CreateSession("test-session")

	// Setup set
	response, _ := g.ProcessInput("SETUP SET", session)
	if !strings.Contains(response, "Set setup complete") {
		t.Errorf("Expected response to contain 'Set setup complete', got '%s'", response)
	}

	// Show initial set
	response, _ = g.ProcessInput("SHOW COLORS", session)
	expected := "Colors: red green blue"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Add yellow
	response, _ = g.ProcessInput("ADD YELLOW", session)
	if !strings.Contains(response, "Added yellow to the set") {
		t.Errorf("Expected response to contain 'Added yellow to the set', got '%s'", response)
	}

	// Show set after adding yellow
	response, _ = g.ProcessInput("SHOW COLORS", session)
	expected = "Colors: red green blue yellow"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Remove green
	response, _ = g.ProcessInput("REMOVE GREEN", session)
	if !strings.Contains(response, "Removed green from the set") {
		t.Errorf("Expected response to contain 'Removed green from the set', got '%s'", response)
	}

	// Show set after removing green
	response, _ = g.ProcessInput("SHOW COLORS", session)
	expected = "Colors: red blue yellow"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Clear set
	response, _ = g.ProcessInput("CLEAR SET", session)
	if !strings.Contains(response, "Set cleared") {
		t.Errorf("Expected response to contain 'Set cleared', got '%s'", response)
	}

	// Show empty set
	response, _ = g.ProcessInput("SHOW COLORS", session)
	if !strings.HasPrefix(response, "Colors:") {
		t.Errorf("Expected response to start with 'Colors:', got '%s'", response)
	}
}

func TestSetTagDuplicateHandling(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>ADD APPLE</pattern>
        <template>
            <set name="fruits" operation="add">apple</set>
            Added apple.
        </template>
    </category>
    <category>
        <pattern>ADD APPLE AGAIN</pattern>
        <template>
            <set name="fruits" operation="add">apple</set>
            Tried to add apple again.
        </template>
    </category>
    <category>
        <pattern>SHOW FRUITS</pattern>
        <template>Fruits: <set name="fruits"></set></template>
    </category>
    <category>
        <pattern>FRUITS SIZE</pattern>
        <template>Fruits count: <set name="fruits" operation="size"></set></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.CreateSession("test-session")

	// Add apple first time
	response, _ := g.ProcessInput("ADD APPLE", session)
	expected := "Added apple."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Try to add apple again
	response, _ = g.ProcessInput("ADD APPLE AGAIN", session)
	expected = "Tried to add apple again."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show fruits (should only have one apple)
	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Fruits: apple"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Check size (should be 1)
	response, _ = g.ProcessInput("FRUITS SIZE", session)
	expected = "Fruits count: 1"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestSetTagCaseInsensitive(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>ADD ITEM</pattern>
        <template>
            <set name="items" operation="add">Apple</set>
            Added Apple.
        </template>
    </category>
    <category>
        <pattern>CHECK LOWER</pattern>
        <template>Contains apple: <set name="items" operation="contains">apple</set></template>
    </category>
    <category>
        <pattern>CHECK UPPER</pattern>
        <template>Contains APPLE: <set name="items" operation="contains">APPLE</set></template>
    </category>
    <category>
        <pattern>REMOVE LOWER</pattern>
        <template>
            <set name="items" operation="remove">apple</set>
            Removed apple.
        </template>
    </category>
    <category>
        <pattern>SHOW ITEMS</pattern>
        <template>Items: <set name="items"></set></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.CreateSession("test-session")

	// Add Apple (capitalized)
	response, _ := g.ProcessInput("ADD ITEM", session)
	expected := "Added Apple."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Check with lowercase
	response, _ = g.ProcessInput("CHECK LOWER", session)
	expected = "Contains apple: true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Check with uppercase
	response, _ = g.ProcessInput("CHECK UPPER", session)
	expected = "Contains APPLE: true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Remove with lowercase
	response, _ = g.ProcessInput("REMOVE LOWER", session)
	expected = "Removed apple."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show items (should be empty)
	response, _ = g.ProcessInput("SHOW ITEMS", session)
	if !strings.HasPrefix(response, "Items:") {
		t.Errorf("Expected response to start with 'Items:', got '%s'", response)
	}
}

func TestSetTagBackwardCompatibility(t *testing.T) {
	// Test that set operations work with explicit operation attribute
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>TEST SET OPERATIONS</pattern>
        <template>
            <set name="colors" operation="add">red</set>
            <set name="colors" operation="add">blue</set>
            Colors: <set name="colors"></set>
        </template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.CreateSession("test-session")

	// Test set operations
	response, _ := g.ProcessInput("TEST SET OPERATIONS", session)
	if !strings.Contains(response, "Colors: red blue") {
		t.Errorf("Expected response to contain 'Colors: red blue', got '%s'", response)
	}
}

func TestSetTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>EMPTY SET</pattern>
        <template><set name="empty" operation="size"></set> <set name="empty" operation="contains">anything</set></template>
    </category>
    <category>
        <pattern>REMOVE NONEXISTENT</pattern>
        <template><set name="test" operation="add">item1</set><set name="test" operation="remove">nonexistent</set><set name="test"></set></template>
    </category>
    <category>
        <pattern>MULTIPLE OPERATIONS</pattern>
        <template><set name="multi" operation="add">a</set><set name="multi" operation="add">b</set><set name="multi" operation="add">c</set><set name="multi" operation="size"></set></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.CreateSession("test-session")

	// Test empty set operations
	response, _ := g.ProcessInput("EMPTY SET", session)
	if !strings.Contains(response, "0") || !strings.Contains(response, "false") {
		t.Errorf("Expected response to contain '0' and 'false', got '%s'", response)
	}

	// Test removing non-existent item
	response, _ = g.ProcessInput("REMOVE NONEXISTENT", session)
	if !strings.Contains(response, "item1") {
		t.Errorf("Expected response to contain 'item1', got '%s'", response)
	}

	// Test multiple operations
	response, _ = g.ProcessInput("MULTIPLE OPERATIONS", session)
	if !strings.Contains(response, "3") {
		t.Errorf("Expected response to contain '3', got '%s'", response)
	}
}
