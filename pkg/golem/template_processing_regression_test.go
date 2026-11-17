package golem

import (
	"strings"
	"testing"
)

func loadTemplateTestAIML(t *testing.T, categories string) (*Golem, *ChatSession) {
	t.Helper()
	g := New(false)
	if err := g.LoadAIMLFromString(`<aiml version="2.0">` + categories + `</aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	return g, g.CreateSession("test")
}

func askTemplate(t *testing.T, g *Golem, session *ChatSession, input string) string {
	t.Helper()
	response, err := g.ProcessInput(input, session)
	if err != nil {
		t.Fatalf("Failed to process %q: %v", input, err)
	}
	return response
}

// Only the chosen <li> of a <random> may run, and only once.
func TestRandomRunsOnlyChosenItem(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>GO</pattern><template><random><li><think><set name="mood">happy</set><list name="runs" operation="add">a</list></think>A</li><li><think><set name="mood">sad</set><list name="runs" operation="add">b</list></think>B</li></random></template></category>`)

	for i := 0; i < 20; i++ {
		response := askTemplate(t, g, session, "go")
		want := map[string]string{"A": "happy", "B": "sad"}[response]
		if mood := session.Variables["mood"]; mood != want {
			t.Fatalf("Response %q but mood %q: the other item ran too", response, mood)
		}
	}
	if runs := len(g.aimlKB.Lists["runs"]); runs != 20 {
		t.Errorf("Expected 20 item runs for 20 inputs, got %d", runs)
	}
}

// An empty <li> is a valid <random> choice.
func TestRandomEmptyItemCanBeChosen(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>GO</pattern><template>x<random><li></li><li>Y</li></random></template></category>`)

	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		seen[askTemplate(t, g, session, "go")] = true
	}
	if !seen["x"] || !seen["xY"] {
		t.Errorf("Expected both 'x' and 'xY' over 50 runs, got %v", seen)
	}
}

// Tags the processor does not know, such as HTML, keep their attributes.
func TestUnknownTagsKeepAttributes(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>LINK</pattern><template>See <a href="https://example.com" target="_blank">here</a></template></category>`)

	response := askTemplate(t, g, session, "link")
	if !strings.Contains(response, `href="https://example.com"`) || !strings.Contains(response, `target="_blank"`) {
		t.Errorf("Expected the link's attributes, got %q", response)
	}
}

// <condition> supports a name on each <li>, value="*" and <loop/>.
func TestConditionFeatures(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>LI NAMES</pattern><template><condition><li name="a" value="1">a is 1</li><li name="b" value="2">b is 2</li><li>neither</li></condition></template></category>
<category><pattern>BOUND</pattern><template><condition name="x"><li value="*">x is bound</li><li>x unbound</li></condition></template></category>
<category><pattern>BLOCK BOUND</pattern><template><condition name="x" value="*">bound</condition></template></category>
<category><pattern>COUNT</pattern><template><condition name="count"><li value="aaa">done</li><li><think><set name="count"><get name="count"/>a</set></think>.<loop/></li></condition></template></category>
<category><pattern>FOREVER</pattern><template><condition><li>x<loop/></li></condition></template></category>`)

	session.Variables["b"] = "2"
	if response := askTemplate(t, g, session, "li names"); response != "b is 2" {
		t.Errorf("Expected 'b is 2', got %q", response)
	}

	if response := askTemplate(t, g, session, "bound"); response != "x unbound" {
		t.Errorf("Expected 'x unbound', got %q", response)
	}
	session.Variables["x"] = "hello"
	if response := askTemplate(t, g, session, "bound"); response != "x is bound" {
		t.Errorf("Expected 'x is bound', got %q", response)
	}
	if response := askTemplate(t, g, session, "block bound"); response != "bound" {
		t.Errorf("Expected 'bound', got %q", response)
	}

	if response := askTemplate(t, g, session, "count"); response != "...done" {
		t.Errorf("Expected '...done', got %q", response)
	}

	if response := askTemplate(t, g, session, "forever"); response != strings.Repeat("x", maxConditionLoops) {
		t.Errorf("Expected the loop to stop after %d iterations, got %d characters", maxConditionLoops, len(response))
	}
}

// Attributes can be written as child elements, as AIML 2.0 allows.
func TestAttributeChildElements(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>GET CHILD</pattern><template>[<get><name>color</name></get>]</template></category>
<category><pattern>SET CHILD</pattern><template><think><set><name>shape</name>round</set></think>[<get name="shape"/>]</template></category>
<category><pattern>DYNAMIC NAME</pattern><template>[<get><name><get name="which"/></name></get>]</template></category>
<category><pattern>COND CHILD</pattern><template><condition><name>color</name><li><value>red</value>is red</li><li>other</li></condition></template></category>`)

	session.Variables["color"] = "red"
	session.Variables["which"] = "color"
	tests := map[string]string{
		"get child":    "[red]",
		"set child":    "[round]",
		"dynamic name": "[red]",
		"cond child":   "is red",
	}
	for input, want := range tests {
		if response := askTemplate(t, g, session, input); response != want {
			t.Errorf("%q: expected %q, got %q", input, want, response)
		}
	}
}

// A local variable set in a srai'd category must not be visible to the caller.
func TestLocalVariablesStayInTheirCategory(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>GO</pattern><template><think><set var="lv">outer</set></think><srai>INNER</srai>[<get var="lv"/>]</template></category>
<category><pattern>INNER</pattern><template><think><set var="lv">inner</set></think>(<get var="lv"/>)</template></category>`)

	if response := askTemplate(t, g, session, "go"); response != "(inner)[outer]" {
		t.Errorf("Expected '(inner)[outer]', got %q", response)
	}
}

// <set> outputs the value it sets, as the AIML spec says.
func TestSetOutputsValue(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>MY NAME IS *</pattern><template>Nice to meet you, <set name="name"><star/></set>.</template></category>
<category><pattern>QUIET *</pattern><template><think><set name="name"><star/></set></think>ok</template></category>`)

	if response := askTemplate(t, g, session, "my name is Bob"); response != "Nice to meet you, Bob." {
		t.Errorf("Expected 'Nice to meet you, Bob.', got %q", response)
	}
	if response := askTemplate(t, g, session, "quiet Alice"); response != "ok" || session.Variables["name"] != "Alice" {
		t.Errorf("Expected 'ok' and name Alice, got %q and %q", response, session.Variables["name"])
	}
}

// <learn> evaluates <eval> and wildcard tags when learning, and keeps every
// other tag to run when the learned category is used.
func TestLearnEvaluatesOnlyEvalAndWildcards(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>CALL ME *</pattern><template><think><set name="name"><star/></set></think>ok</template></category>
<category><pattern>TEACH WHO</pattern><template><learn><category><pattern>WHO AM I</pattern><template>You are <get name="name"/></template></category></learn>ok</template></category>
<category><pattern>TEACH MOOD</pattern><template><learn><category><pattern>MOOD</pattern><template><think><set name="mood">used</set></think>m</template></category></learn>ok</template></category>
<category><pattern>TEACH FROZEN</pattern><template><learn><category><pattern>FROZEN</pattern><template><eval><get name="name"/></eval></template></category></learn>ok</template></category>
<category><pattern>TEACH * MEANS *</pattern><template><learn><category><pattern><eval><uppercase><star/></uppercase></eval></pattern><template><star index="2"/></template></category></learn>ok</template></category>`)

	askTemplate(t, g, session, "call me alice")
	askTemplate(t, g, session, "teach who")
	askTemplate(t, g, session, "teach frozen")
	askTemplate(t, g, session, "call me bob")
	if response := askTemplate(t, g, session, "who am i"); response != "You are bob" {
		t.Errorf("Expected <get> to run when used: 'You are bob', got %q", response)
	}
	if response := askTemplate(t, g, session, "frozen"); response != "alice" {
		t.Errorf("Expected <eval> to run when learned: 'alice', got %q", response)
	}

	askTemplate(t, g, session, "teach mood")
	if mood, exists := session.Variables["mood"]; exists {
		t.Errorf("Expected <set> not to run when learned, mood is %q", mood)
	}
	askTemplate(t, g, session, "mood")
	if session.Variables["mood"] != "used" {
		t.Errorf("Expected <set> to run when used, mood is %q", session.Variables["mood"])
	}

	// Markup typed by the user is captured as text, not as tags to run
	askTemplate(t, g, session, "teach greet means hi <srai>MOOD</srai>")
	if response := askTemplate(t, g, session, "greet"); response != "hi <srai>MOOD</srai>" {
		t.Errorf("Expected the taught markup as text, got %q", response)
	}
}

// In a learned pattern, <star/> becomes a * wildcard when the teaching
// category has none, and other tags are evaluated when learned
func TestLearnedPatternStarAndTags(t *testing.T) {
	g, session := loadTemplateTestAIML(t, `
<category><pattern>CALL ME *</pattern><template><think><set name="name"><star/></set></think>ok</template></category>
<category><pattern>TEACH LIKES</pattern><template><learn><category><pattern>I LIKE <star/></pattern><template>You like <star/></template></category></learn>ok</template></category>
<category><pattern>TEACH NAME</pattern><template><learn><category><pattern>HI <get name="name"/></pattern><template>That is you</template></category></learn>ok</template></category>`)

	askTemplate(t, g, session, "teach likes")
	if response := askTemplate(t, g, session, "i like green tea"); response != "You like green tea" {
		t.Errorf("Expected <star/> to become a wildcard: 'You like green tea', got %q", response)
	}

	askTemplate(t, g, session, "call me alice")
	askTemplate(t, g, session, "teach name")
	askTemplate(t, g, session, "call me bob")
	if response := askTemplate(t, g, session, "hi alice"); response != "That is you" {
		t.Errorf("Expected <get> in the pattern to run when learned: 'That is you', got %q", response)
	}
}
