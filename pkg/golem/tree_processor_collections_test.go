package golem

import (
	"strings"
	"testing"
)

// TestTreeProcessorMapTag tests the native AST implementation of the <map> tag
func TestTreeProcessorMapTag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Map set and get operations",
			aiml: `
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
</aiml>`,
			input:    "GET COLOR red",
			expected: "Color: #FF0000",
		},
		{
			name: "Map size operation",
			aiml: `
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
        <pattern>SHOW MAP SIZE</pattern>
        <template>Map size: <map name="colors" operation="size"></map></template>
    </category>
</aiml>`,
			input:    "SHOW MAP SIZE",
			expected: "Map size: 3",
		},
		{
			name: "Map keys operation",
			aiml: `
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
        <pattern>SHOW MAP KEYS</pattern>
        <template>Keys: <map name="colors" operation="keys"></map></template>
    </category>
</aiml>`,
			input:    "SHOW MAP KEYS",
			expected: "Keys: blue green red",
		},
		{
			name: "Map values operation",
			aiml: `
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
        <pattern>SHOW MAP VALUES</pattern>
        <template>Values: <map name="colors" operation="values"></map></template>
    </category>
</aiml>`,
			input:    "SHOW MAP VALUES",
			expected: "Values: #0000FF #00FF00 #FF0000",
		},
		{
			name: "Map list operation",
			aiml: `
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
        <pattern>SHOW MAP LIST</pattern>
        <template>Map: <map name="colors" operation="list"></map></template>
    </category>
</aiml>`,
			input:    "SHOW MAP LIST",
			expected: "Map: blue:#0000FF green:#00FF00 red:#FF0000",
		},
		{
			name: "Map contains operation",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SETUP MAP</pattern>
        <template>
            <map name="fruits" key="apple" operation="set">red</map>
            <map name="fruits" key="banana" operation="set">yellow</map>
            Map setup complete.
        </template>
    </category>
    <category>
        <pattern>CHECK FRUIT *</pattern>
        <template>
            <map name="fruits" key="<star/>" operation="contains"></map>
        </template>
    </category>
</aiml>`,
			input:    "CHECK FRUIT apple",
			expected: "true",
		},
		{
			name: "Map backward compatibility",
			aiml: `
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
</aiml>`,
			input:    "GET red",
			expected: "#FF0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			// Setup if needed
			if strings.Contains(tt.aiml, "SETUP MAP") {
				_, _ = g.ProcessInput("SETUP MAP", session)
			}

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorListTag tests the native AST implementation of the <list> tag
func TestTreeProcessorListTag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "List add and get operations",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>ADD TO LIST</pattern>
        <template>I'll add that to your list. <list name="shopping" operation="add">milk</list></template>
    </category>
    <category>
        <pattern>SHOW LIST</pattern>
        <template>Your list contains: <list name="shopping"></list></template>
    </category>
</aiml>`,
			input:    "SHOW LIST",
			expected: "Your list contains: milk",
		},
		{
			name: "List size operation",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>ADD TO LIST</pattern>
        <template>I'll add that to your list. <list name="shopping" operation="add">milk</list></template>
    </category>
    <category>
        <pattern>LIST SIZE</pattern>
        <template>Your list has <list name="shopping" operation="size"></list> items.</template>
    </category>
</aiml>`,
			input:    "LIST SIZE",
			expected: "Your list has 1 items.",
		},
		{
			name: "List insert operation",
			aiml: `
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
</aiml>`,
			input:    "SHOW NUMBERS",
			expected: "Numbers: one two three four",
		},
		{
			name: "List remove by value",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SETUP LIST</pattern>
        <template>
            <list name="numbers" operation="add">one</list>
            <list name="numbers" operation="add">two</list>
            <list name="numbers" operation="add">three</list>
            <list name="numbers" operation="add">four</list>
            List setup complete.
        </template>
    </category>
    <category>
        <pattern>REMOVE TWO</pattern>
        <template>
            <list name="numbers" operation="remove">two</list>
            Removed two from the list.
        </template>
    </category>
    <category>
        <pattern>SHOW NUMBERS</pattern>
        <template>Numbers: <list name="numbers"></list></template>
    </category>
</aiml>`,
			input:    "SHOW NUMBERS",
			expected: "Numbers: one three four",
		},
		{
			name: "List get by index",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SETUP LIST</pattern>
        <template>
            <list name="fruits" operation="add">apple</list>
            <list name="fruits" operation="add">banana</list>
            <list name="fruits" operation="add">orange</list>
            List setup complete.
        </template>
    </category>
    <category>
        <pattern>GET FIRST FRUIT</pattern>
        <template>The first fruit is <list name="fruits" index="0"></list></template>
    </category>
</aiml>`,
			input:    "GET FIRST FRUIT",
			expected: "The first fruit is apple",
		},
		{
			name: "List clear operation",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SETUP LIST</pattern>
        <template>
            <list name="numbers" operation="add">one</list>
            <list name="numbers" operation="add">two</list>
            <list name="numbers" operation="add">three</list>
            List setup complete.
        </template>
    </category>
    <category>
        <pattern>CLEAR LIST</pattern>
        <template>
            <list name="numbers" operation="clear"></list>
            List cleared.
        </template>
    </category>
    <category>
        <pattern>SHOW NUMBERS</pattern>
        <template>Numbers: <list name="numbers"></list></template>
    </category>
</aiml>`,
			input:    "SHOW NUMBERS",
			expected: "Numbers:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			// Setup if needed
			if strings.Contains(tt.aiml, "SETUP LIST") {
				_, _ = g.ProcessInput("SETUP LIST", session)
			}
			if strings.Contains(tt.aiml, "ADD TO LIST") {
				_, _ = g.ProcessInput("ADD TO LIST", session)
			}
			if strings.Contains(tt.aiml, "INSERT THREE") {
				_, _ = g.ProcessInput("INSERT THREE", session)
			}
			if strings.Contains(tt.aiml, "REMOVE TWO") {
				_, _ = g.ProcessInput("REMOVE TWO", session)
			}
			if strings.Contains(tt.aiml, "CLEAR LIST") {
				_, _ = g.ProcessInput("CLEAR LIST", session)
			}

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorArrayTag tests the native AST implementation of the <array> tag
func TestTreeProcessorArrayTag(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Array set and get operations",
			aiml: `
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
</aiml>`,
			input:    "SHOW ARRAY",
			expected: "Scores: 100 85 92",
		},
		{
			name: "Array get by index",
			aiml: `
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
        <pattern>GET FIRST SCORE</pattern>
        <template>First score: <array name="scores" index="0"></array></template>
    </category>
</aiml>`,
			input:    "GET FIRST SCORE",
			expected: "First score: 100",
		},
		{
			name: "Array size operation",
			aiml: `
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
        <pattern>ARRAY SIZE</pattern>
        <template>Array has <array name="scores" operation="size"></array> elements.</template>
    </category>
</aiml>`,
			input:    "ARRAY SIZE",
			expected: "Array has 3 elements.",
		},
		{
			name: "Array sparse indices",
			aiml: `
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
        <pattern>GET INDEX 5</pattern>
        <template>Index 5: <array name="sparse" index="5"></array></template>
    </category>
    <category>
        <pattern>GET INDEX 10</pattern>
        <template>Index 10: <array name="sparse" index="10"></array></template>
    </category>
</aiml>`,
			input:    "GET INDEX 10",
			expected: "Index 10: value10",
		},
		{
			name: "Array clear operation",
			aiml: `
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
        <pattern>CLEAR DATA</pattern>
        <template>
            <array name="data" operation="clear"></array>
            Array cleared.
        </template>
    </category>
    <category>
        <pattern>SHOW DATA</pattern>
        <template>Data: <array name="data"></array></template>
    </category>
</aiml>`,
			input:    "SHOW DATA",
			expected: "Data:", // Trailing space trimmed by tree processor
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			// Setup if needed
			if strings.Contains(tt.aiml, "SET ARRAY") {
				_, _ = g.ProcessInput("SET ARRAY", session)
			}
			if strings.Contains(tt.aiml, "SET HIGH INDEX") {
				_, _ = g.ProcessInput("SET HIGH INDEX", session)
			}
			if strings.Contains(tt.aiml, "FILL ARRAY") {
				_, _ = g.ProcessInput("FILL ARRAY", session)
			}
			if strings.Contains(tt.aiml, "CLEAR DATA") {
				_, _ = g.ProcessInput("CLEAR DATA", session)
			}

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorCollectionsWithVariables tests collection tags with variable content
func TestTreeProcessorCollectionsWithVariables(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		inputs   []string
		expected string
	}{
		{
			name: "Map with wildcard key",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SET COLOR * TO *</pattern>
        <template>
            <map name="colors" key="<star/>" operation="set"><star index="2"/></map>
            Set color <star/> to <star index="2"/>.
        </template>
    </category>
    <category>
        <pattern>GET COLOR *</pattern>
        <template>Color: <map name="colors" key="<star/>"></map></template>
    </category>
</aiml>`,
			inputs:   []string{"SET COLOR red TO #FF0000", "GET COLOR red"},
			expected: "Color: #FF0000",
		},
		{
			name: "List with variable content",
			aiml: `
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
</aiml>`,
			inputs:   []string{"MY NAME IS ALICE", "SHOW USERS"},
			expected: "Users: ALICE",
		},
		{
			name: "Array with variable content",
			aiml: `
<aiml version="2.0">
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
</aiml>`,
			inputs:   []string{"SET MY SCORE 95", "SHOW SCORES"},
			expected: "Scores: 95",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			var response string
			for _, input := range tt.inputs {
				response, _ = g.ProcessInput(input, session)
			}

			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}

// TestTreeProcessorCollectionsIntegration tests complex integration scenarios
func TestTreeProcessorCollectionsIntegration(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		inputs   []string
		expected []string
	}{
		{
			name: "Map complete workflow",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SETUP MAP</pattern>
        <template><map name="scores" key="alice" operation="set">100</map><map name="scores" key="bob" operation="set">85</map><map name="scores" key="charlie" operation="set">92</map>Map setup complete.</template>
    </category>
    <category>
        <pattern>REMOVE SCORE *</pattern>
        <template><map name="scores" key="<star/>" operation="remove"></map>Removed score for <star/>.</template>
    </category>
    <category>
        <pattern>SHOW SCORES</pattern>
        <template>Scores: <map name="scores" operation="list"></map></template>
    </category>
</aiml>`,
			inputs:   []string{"SETUP MAP", "SHOW SCORES", "REMOVE SCORE bob", "SHOW SCORES"},
			expected: []string{"Map setup complete.", "Scores: alice:100 bob:85 charlie:92", "Removed score for bob.", "Scores: alice:100 charlie:92"},
		},
		{
			name: "List complete workflow",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>ADD ITEMS</pattern>
        <template><list name="fruits" operation="add">apple</list><list name="fruits" operation="add">banana</list><list name="fruits" operation="add">orange</list>Added three fruits.</template>
    </category>
    <category>
        <pattern>SHOW FRUITS</pattern>
        <template>Fruits: <list name="fruits"></list></template>
    </category>
    <category>
        <pattern>REMOVE BANANA</pattern>
        <template><list name="fruits" operation="remove">banana</list>Removed banana.</template>
    </category>
    <category>
        <pattern>FRUIT SIZE</pattern>
        <template>Size: <list name="fruits" operation="size"></list></template>
    </category>
</aiml>`,
			inputs:   []string{"ADD ITEMS", "SHOW FRUITS", "REMOVE BANANA", "FRUIT SIZE"},
			expected: []string{"Added three fruits.", "Fruits: apple banana orange", "Removed banana.", "Size: 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			for i, input := range tt.inputs {
				response, _ := g.ProcessInput(input, session)
				if response != tt.expected[i] {
					t.Errorf("Input %d ('%s'): Expected '%s', got '%s'", i, input, tt.expected[i], response)
				}
			}
		})
	}
}

// TestTreeProcessorCollectionsEdgeCases tests edge cases and error handling
func TestTreeProcessorCollectionsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		aiml     string
		input    string
		expected string
	}{
		{
			name: "Map empty size",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>EMPTY MAP SIZE</pattern>
        <template>Size: <map name="empty" operation="size"></map></template>
    </category>
</aiml>`,
			input:    "EMPTY MAP SIZE",
			expected: "Size: 0",
		},
		{
			name: "List empty operations",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>EMPTY LIST</pattern>
        <template>List: <list name="empty"></list></template>
    </category>
</aiml>`,
			input:    "EMPTY LIST",
			expected: "List:", // Trailing space trimmed by tree processor
		},
		{
			name: "Array empty operations",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>EMPTY ARRAY</pattern>
        <template>Array: <array name="empty"></array></template>
    </category>
</aiml>`,
			input:    "EMPTY ARRAY",
			expected: "Array:", // Trailing space trimmed by tree processor
		},
		{
			name: "Map key not found",
			aiml: `
<aiml version="2.0">
    <category>
        <pattern>SETUP MAP</pattern>
        <template>
            <map name="colors" key="red" operation="set">#FF0000</map>
            Map setup.
        </template>
    </category>
    <category>
        <pattern>GET *</pattern>
        <template><map name="colors"><star/></map></template>
    </category>
</aiml>`,
			input:    "GET blue",
			expected: "blue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewForTesting(t, false)
			g.EnableTreeProcessing() // Enable tree processing for native AST
			_ = g.LoadAIMLFromString(tt.aiml)
			session := g.CreateSession("test-session")

			// Setup if needed
			if strings.Contains(tt.aiml, "SETUP MAP") {
				_, _ = g.ProcessInput("SETUP MAP", session)
			}

			response, _ := g.ProcessInput(tt.input, session)
			if response != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, response)
			}
		})
	}
}
