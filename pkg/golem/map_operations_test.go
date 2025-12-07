package golem

import (
	"strings"
	"testing"
)

func TestMapTagBasicOperations(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SETUP MAP</pattern>
        <template>
            <map name="colors" key="red" operation="set">#FF0000</map>
            <map name="colors" key="green" operation="set">#00FF00</map>
            <map name="colors" key="blue" operation="set">#0000FF</map>
            Map setup complete.
        </template>
    </category>
    <category>
        <pattern>GET COLOR *</pattern>
        <template>Color: <map name="colors" key="<star/>"></map></template>
    </category>
    <category>
        <pattern>SHOW MAP SIZE</pattern>
        <template>Map size: <map name="colors" operation="size"></map></template>
    </category>
    <category>
        <pattern>SHOW MAP KEYS</pattern>
        <template>Keys: <map name="colors" operation="keys"></map></template>
    </category>
    <category>
        <pattern>SHOW MAP VALUES</pattern>
        <template>Values: <map name="colors" operation="values"></map></template>
    </category>
    <category>
        <pattern>SHOW MAP LIST</pattern>
        <template>Map: <map name="colors" operation="list"></map></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	// Setup map
	response, _ := g.ProcessInput("SETUP MAP", session)
	if !strings.Contains(response, "Map setup complete") {
		t.Errorf("Expected response to contain 'Map setup complete', got '%s'", response)
	}

	// Test get operations
	response, _ = g.ProcessInput("GET COLOR red", session)
	expected := "Color: #FF0000"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("GET COLOR green", session)
	expected = "Color: #00FF00"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("GET COLOR blue", session)
	expected = "Color: #0000FF"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test size operation
	response, _ = g.ProcessInput("SHOW MAP SIZE", session)
	expected = "Map size: 3"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test keys operation
	response, _ = g.ProcessInput("SHOW MAP KEYS", session)
	// Keys should be sorted alphabetically
	expected = "Keys: blue green red"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test values operation
	response, _ = g.ProcessInput("SHOW MAP VALUES", session)
	// Values should be sorted alphabetically
	expected = "Values: #0000FF #00FF00 #FF0000"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test list operation
	response, _ = g.ProcessInput("SHOW MAP LIST", session)
	// Pairs should be sorted alphabetically
	expected = "Map: blue:#0000FF green:#00FF00 red:#FF0000"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestMapTagMultipleOperations(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SETUP MAP</pattern>
        <template>
            <map name="scores" key="alice" operation="set">100</map>
            <map name="scores" key="bob" operation="set">85</map>
            <map name="scores" key="charlie" operation="set">92</map>
            Map setup complete.
        </template>
    </category>
    <category>
        <pattern>ADD SCORE *</pattern>
        <template>
            <map name="scores" key="<star/>" operation="set">100</map>
            Added score for <star/>.
        </template>
    </category>
    <category>
        <pattern>REMOVE SCORE *</pattern>
        <template>
            <map name="scores" key="<star/>" operation="remove"></map>
            Removed score for <star/>.
        </template>
    </category>
    <category>
        <pattern>SHOW SCORES</pattern>
        <template>Scores: <map name="scores" operation="list"></map></template>
    </category>
    <category>
        <pattern>CLEAR SCORES</pattern>
        <template>
            <map name="scores" operation="clear"></map>
            Scores cleared.
        </template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	// Setup map
	response, _ := g.ProcessInput("SETUP MAP", session)
	if !strings.Contains(response, "Map setup complete") {
		t.Errorf("Expected response to contain 'Map setup complete', got '%s'", response)
	}

	// Show initial scores
	response, _ = g.ProcessInput("SHOW SCORES", session)
	expected := "Scores: alice:100 bob:85 charlie:92"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", response, expected)
	}

	// Add new score
	response, _ = g.ProcessInput("ADD SCORE david", session)
	if !strings.Contains(response, "Added score for david") {
		t.Errorf("Expected response to contain 'Added score for david', got '%s'", response)
	}

	// Show scores after adding
	response, _ = g.ProcessInput("SHOW SCORES", session)
	expected = "Scores: alice:100 bob:85 charlie:92 david:100"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", response, expected)
	}

	// Remove a score
	response, _ = g.ProcessInput("REMOVE SCORE bob", session)
	if !strings.Contains(response, "Removed score for bob") {
		t.Errorf("Expected response to contain 'Removed score for bob', got '%s'", response)
	}

	// Show scores after removing
	response, _ = g.ProcessInput("SHOW SCORES", session)
	expected = "Scores: alice:100 charlie:92 david:100"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", response, expected)
	}

	// Clear scores
	response, _ = g.ProcessInput("CLEAR SCORES", session)
	if !strings.Contains(response, "Scores cleared") {
		t.Errorf("Expected response to contain 'Scores cleared', got '%s'", response)
	}

	// Show scores after clearing
	response, _ = g.ProcessInput("SHOW SCORES", session)
	expected = "Scores:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", response, expected)
	}
}

func TestMapTagContainsOperation(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SETUP MAP</pattern>
        <template>
            <map name="fruits" key="apple" operation="set">red</map>
            <map name="fruits" key="banana" operation="set">yellow</map>
            <map name="fruits" key="grape" operation="set">purple</map>
            Map setup complete.
        </template>
    </category>
    <category>
        <pattern>CHECK FRUIT *</pattern>
        <template>
            <map name="fruits" key="<star/>" operation="contains"></map>
        </template>
    </category>
    <category>
        <pattern>SHOW MAP SIZE</pattern>
        <template>Size: <map name="fruits" operation="size"></map></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	// Setup map
	response, _ := g.ProcessInput("SETUP MAP", session)
	if !strings.Contains(response, "Map setup complete") {
		t.Errorf("Expected response to contain 'Map setup complete', got '%s'", response)
	}

	// Test contains operations
	response, _ = g.ProcessInput("CHECK FRUIT apple", session)
	expected := "true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("CHECK FRUIT banana", session)
	expected = "true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("CHECK FRUIT orange", session)
	expected = "false"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("CHECK FRUIT grape", session)
	expected = "true"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test size
	response, _ = g.ProcessInput("SHOW MAP SIZE", session)
	expected = "Size: 3"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestMapTagBackwardCompatibility(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SETUP MAP</pattern>
        <template>
            <map name="colors" key="red" operation="set">#FF0000</map>
            <map name="colors" key="green" operation="set">#00FF00</map>
            Map setup complete.
        </template>
    </category>
    <category>
        <pattern>GET *</pattern>
        <template><map name="colors"><star/></map></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	// Setup map
	response, _ := g.ProcessInput("SETUP MAP", session)
	if !strings.Contains(response, "Map setup complete") {
		t.Errorf("Expected response to contain 'Map setup complete', got '%s'", response)
	}

	// Test backward compatibility (original syntax)
	response, _ = g.ProcessInput("GET red", session)
	expected := "#FF0000"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("GET green", session)
	expected = "#00FF00"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test non-existent key
	response, _ = g.ProcessInput("GET blue", session)
	expected = "blue" // Should return the key if not found
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestMapTagEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>EMPTY MAP SIZE</pattern>
        <template>Size: <map name="empty" operation="size"></map></template>
    </category>
    <category>
        <pattern>EMPTY MAP KEYS</pattern>
        <template>Keys: <map name="empty" operation="keys"></map></template>
    </category>
    <category>
        <pattern>EMPTY MAP VALUES</pattern>
        <template>Values: <map name="empty" operation="values"></map></template>
    </category>
    <category>
        <pattern>EMPTY MAP LIST</pattern>
        <template>Map: <map name="empty" operation="list"></map></template>
    </category>
    <category>
        <pattern>REMOVE NONEXISTENT</pattern>
        <template>
            <map name="empty" key="nonexistent" operation="remove"></map>
            Removed.
        </template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)
	session := g.CreateSession("test-session")

	// Test operations on empty map
	response, _ := g.ProcessInput("EMPTY MAP SIZE", session)
	expected := "Size: 0"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("EMPTY MAP KEYS", session)
	expected = "Keys:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("EMPTY MAP VALUES", session)
	expected = "Values:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("EMPTY MAP LIST", session)
	expected = "Map:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test removing non-existent key
	response, _ = g.ProcessInput("REMOVE NONEXISTENT", session)
	if !strings.Contains(response, "Removed") {
		t.Errorf("Expected response to contain 'Removed', got '%s'", response)
	}
}
