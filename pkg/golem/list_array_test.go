package golem

import (
	"testing"
)

func TestListTagBasicOperations(t *testing.T) {
	g := NewForTesting(t, false)

	// Test basic list operations
	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>ADD TO LIST</pattern>
        <template>I'll add that to your list. <list name="shopping" operation="add">milk</list></template>
    </category>
    <category>
        <pattern>SHOW LIST</pattern>
        <template>Your list contains: <list name="shopping"></list></template>
    </category>
    <category>
        <pattern>LIST SIZE</pattern>
        <template>Your list has <list name="shopping" operation="size"></list> items.</template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Test adding items to list
	response, _ := g.ProcessInput("ADD TO LIST", session)
	expected := "I'll add that to your list."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test showing list
	response, _ = g.ProcessInput("SHOW LIST", session)
	expected = "Your list contains: milk"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test list size
	response, _ = g.ProcessInput("LIST SIZE", session)
	expected = "Your list has 1 items."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestListTagMultipleOperations(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>ADD ITEMS</pattern>
        <template>
            <list name="fruits" operation="add">apple</list>
            <list name="fruits" operation="add">banana</list>
            <list name="fruits" operation="add">orange</list>
            Added three fruits to your list.
        </template>
    </category>
    <category>
        <pattern>SHOW FRUITS</pattern>
        <template>Fruits: <list name="fruits"></list></template>
    </category>
    <category>
        <pattern>GET FIRST FRUIT</pattern>
        <template>The first fruit is <list name="fruits" index="0"></list></template>
    </category>
    <category>
        <pattern>GET SECOND FRUIT</pattern>
        <template>The second fruit is <list name="fruits" index="1"></list></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Test adding multiple items
	response, _ := g.ProcessInput("ADD ITEMS", session)
	expected := "Added three fruits to your list."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test showing all fruits
	response, _ = g.ProcessInput("SHOW FRUITS", session)
	expected = "Fruits: apple banana orange"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test getting first fruit
	response, _ = g.ProcessInput("GET FIRST FRUIT", session)
	expected = "The first fruit is apple"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test getting second fruit
	response, _ = g.ProcessInput("GET SECOND FRUIT", session)
	expected = "The second fruit is banana"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", response, expected)
	}
}

func TestListTagInsertAndRemove(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SETUP LIST</pattern>
        <template>
            <list name="numbers" operation="add">one</list>
            <list name="numbers" operation="add">two</list>
            <list name="numbers" operation="add">four</list>
            List setup complete.
        </template>
    </category>
    <category>
        <pattern>INSERT THREE</pattern>
        <template>
            <list name="numbers" index="2" operation="insert">three</list>
            Inserted three at position 2.
        </template>
    </category>
    <category>
        <pattern>SHOW NUMBERS</pattern>
        <template>Numbers: <list name="numbers"></list></template>
    </category>
    <category>
        <pattern>REMOVE TWO</pattern>
        <template>
            <list name="numbers" operation="remove">two</list>
            Removed two from the list.
        </template>
    </category>
    <category>
        <pattern>REMOVE AT INDEX</pattern>
        <template>
            <list name="numbers" index="1" operation="remove"></list>
            Removed item at index 1.
        </template>
    </category>
    <category>
        <pattern>CLEAR LIST</pattern>
        <template>
            <list name="numbers" operation="clear"></list>
            List cleared.
        </template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Setup list
	response, _ := g.ProcessInput("SETUP LIST", session)
	expected := "List setup complete."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show initial list
	response, _ = g.ProcessInput("SHOW NUMBERS", session)
	expected = "Numbers: one two four"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Insert three at position 2
	response, _ = g.ProcessInput("INSERT THREE", session)
	expected = "Inserted three at position 2."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show list after insert
	response, _ = g.ProcessInput("SHOW NUMBERS", session)
	expected = "Numbers: one two three four"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Remove by value
	response, _ = g.ProcessInput("REMOVE TWO", session)
	expected = "Removed two from the list."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show list after remove by value
	response, _ = g.ProcessInput("SHOW NUMBERS", session)
	expected = "Numbers: one three four"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Remove at index
	response, _ = g.ProcessInput("REMOVE AT INDEX", session)
	expected = "Removed item at index 1."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show list after remove at index
	response, _ = g.ProcessInput("SHOW NUMBERS", session)
	expected = "Numbers: one four"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Clear list
	response, _ = g.ProcessInput("CLEAR LIST", session)
	expected = "List cleared."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show empty list
	response, _ = g.ProcessInput("SHOW NUMBERS", session)
	expected = "Numbers:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestArrayTagBasicOperations(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SET ARRAY</pattern>
        <template>
            <array name="scores" index="0" operation="set">100</array>
            <array name="scores" index="1" operation="set">85</array>
            <array name="scores" index="2" operation="set">92</array>
            Array set with three scores.
        </template>
    </category>
    <category>
        <pattern>SHOW ARRAY</pattern>
        <template>Scores: <array name="scores"></array></template>
    </category>
    <category>
        <pattern>GET FIRST SCORE</pattern>
        <template>First score: <array name="scores" index="0"></array></template>
    </category>
    <category>
        <pattern>ARRAY SIZE</pattern>
        <template>Array has <array name="scores" operation="size"></array> elements.</template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Test setting array values
	response, _ := g.ProcessInput("SET ARRAY", session)
	expected := "Array set with three scores."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test showing array
	response, _ = g.ProcessInput("SHOW ARRAY", session)
	expected = "Scores: 100 85 92"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test getting first score
	response, _ = g.ProcessInput("GET FIRST SCORE", session)
	expected = "First score: 100"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test array size
	response, _ = g.ProcessInput("ARRAY SIZE", session)
	expected = "Array has 3 elements."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestArrayTagDynamicSizing(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>SET HIGH INDEX</pattern>
        <template>
            <array name="sparse" index="5" operation="set">value5</array>
            <array name="sparse" index="10" operation="set">value10</array>
            Set values at high indices.
        </template>
    </category>
    <category>
        <pattern>SHOW SPARSE</pattern>
        <template>Sparse array: <array name="sparse"></array></template>
    </category>
    <category>
        <pattern>GET INDEX 5</pattern>
        <template>Index 5: <array name="sparse" index="5"></array></template>
    </category>
    <category>
        <pattern>GET INDEX 10</pattern>
        <template>Index 10: <array name="sparse" index="10"></array></template>
    </category>
    <category>
        <pattern>GET INDEX 3</pattern>
        <template>Index 3: <array name="sparse" index="3"></array></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Test setting values at high indices
	response, _ := g.ProcessInput("SET HIGH INDEX", session)
	expected := "Set values at high indices."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test showing sparse array
	response, _ = g.ProcessInput("SHOW SPARSE", session)
	expected = "Sparse array:      value5     value10"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test getting value at index 5
	response, _ = g.ProcessInput("GET INDEX 5", session)
	expected = "Index 5: value5"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test getting value at index 10
	response, _ = g.ProcessInput("GET INDEX 10", session)
	expected = "Index 10: value10"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test getting value at empty index 3
	response, _ = g.ProcessInput("GET INDEX 3", session)
	expected = "Index 3:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestArrayTagClear(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>FILL ARRAY</pattern>
        <template>
            <array name="data" index="0" operation="set">item1</array>
            <array name="data" index="1" operation="set">item2</array>
            <array name="data" index="2" operation="set">item3</array>
            Array filled.
        </template>
    </category>
    <category>
        <pattern>SHOW DATA</pattern>
        <template>Data: <array name="data"></array></template>
    </category>
    <category>
        <pattern>CLEAR DATA</pattern>
        <template>
            <array name="data" operation="clear"></array>
            Array cleared.
        </template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Fill array
	response, _ := g.ProcessInput("FILL ARRAY", session)
	expected := "Array filled."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show filled array
	response, _ = g.ProcessInput("SHOW DATA", session)
	expected = "Data: item1 item2 item3"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Clear array
	response, _ = g.ProcessInput("CLEAR DATA", session)
	expected = "Array cleared."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show empty array
	response, _ = g.ProcessInput("SHOW DATA", session)
	expected = "Data:"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestListAndArrayWithVariables(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>MY NAME IS *</pattern>
        <template>
            <set name="username"><star/></set>
            <list name="users" operation="add"><get name="username"/></list>
            Hello <get name="username"/>, I've added you to the user list.
        </template>
    </category>
    <category>
        <pattern>SHOW USERS</pattern>
        <template>Users: <list name="users"></list></template>
    </category>
    <category>
        <pattern>SET MY SCORE *</pattern>
        <template>
            <set name="score"><star/></set>
            <array name="scores" index="0" operation="set"><get name="score"/></array>
            Your score <get name="score"/> has been recorded.
        </template>
    </category>
    <category>
        <pattern>SHOW SCORES</pattern>
        <template>Scores: <array name="scores"></array></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Test adding user to list
	response, _ := g.ProcessInput("MY NAME IS ALICE", session)
	expected := "Hello ALICE, I've added you to the user list."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test showing users
	response, _ = g.ProcessInput("SHOW USERS", session)
	expected = "Users: ALICE"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test setting score
	response, _ = g.ProcessInput("SET MY SCORE 95", session)
	expected = "Your score 95 has been recorded."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test showing scores
	response, _ = g.ProcessInput("SHOW SCORES", session)
	expected = "Scores: 95"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestListAndArrayErrorHandling(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>TEST INVALID INDEX</pattern>
        <template>
            <list name="test" index="invalid"></list>
            <array name="test" index="invalid"></array>
            Testing invalid indices.
        </template>
    </category>
    <category>
        <pattern>TEST NEGATIVE INDEX</pattern>
        <template>
            <list name="test" index="-1"></list>
            <array name="test" index="-1"></array>
            Testing negative indices.
        </template>
    </category>
    <category>
        <pattern>TEST OUT OF BOUNDS</pattern>
        <template>
            <list name="test" index="999"></list>
            <array name="test" index="999"></array>
            Testing out of bounds.
        </template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Test invalid index
	response, _ := g.ProcessInput("TEST INVALID INDEX", session)
	expected := "Testing invalid indices."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test negative index
	response, _ = g.ProcessInput("TEST NEGATIVE INDEX", session)
	expected = "Testing negative indices."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test out of bounds
	response, _ = g.ProcessInput("TEST OUT OF BOUNDS", session)
	expected = "Testing out of bounds."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}

func TestListAndArrayPersistence(t *testing.T) {
	g := NewForTesting(t, false)

	aimlContent := `
<aiml version="2.0">
    <category>
        <pattern>ADD TO PERSISTENT LIST</pattern>
        <template>
            <list name="persistent" operation="add">item1</list>
            <list name="persistent" operation="add">item2</list>
            Added to persistent list.
        </template>
    </category>
    <category>
        <pattern>SHOW PERSISTENT LIST</pattern>
        <template>Persistent list: <list name="persistent"></list></template>
    </category>
    <category>
        <pattern>SET PERSISTENT ARRAY</pattern>
        <template>
            <array name="persistent" index="0" operation="set">value1</array>
            <array name="persistent" index="1" operation="set">value2</array>
            Set persistent array.
        </template>
    </category>
    <category>
        <pattern>SHOW PERSISTENT ARRAY</pattern>
        <template>Persistent array: <array name="persistent"></array></template>
    </category>
</aiml>`

	_ = g.LoadAIMLFromString(aimlContent)

	session := g.createSession("test-session")

	// Add to persistent list
	response, _ := g.ProcessInput("ADD TO PERSISTENT LIST", session)
	expected := "Added to persistent list."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show persistent list
	response, _ = g.ProcessInput("SHOW PERSISTENT LIST", session)
	expected = "Persistent list: item1 item2"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Set persistent array
	response, _ = g.ProcessInput("SET PERSISTENT ARRAY", session)
	expected = "Set persistent array."
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Show persistent array
	response, _ = g.ProcessInput("SHOW PERSISTENT ARRAY", session)
	expected = "Persistent array: value1 value2"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	// Test persistence across different inputs
	response, _ = g.ProcessInput("SHOW PERSISTENT LIST", session)
	expected = "Persistent list: item1 item2"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}

	response, _ = g.ProcessInput("SHOW PERSISTENT ARRAY", session)
	expected = "Persistent array: value1 value2"
	if response != expected {
		t.Errorf("Expected '%s', got '%s'", expected, response)
	}
}
