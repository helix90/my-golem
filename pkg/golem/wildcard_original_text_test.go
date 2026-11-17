package golem

import "testing"

// A wildcard keeps the user's own text even when a substitution changed the
// number of words in the input
func TestWildcardKeepsOriginalTextAfterSubstitution(t *testing.T) {
	g := NewForTesting(t, false)
	if err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>ECHO *</pattern><template>[<star/>]</template></category>
</aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.aimlKB.Substitutions = map[string]map[string]string{"normal": {"wanna": "want to"}}
	session := g.CreateSession("original-text")

	response, err := g.ProcessInput("echo I Wanna go, OK?", session)
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}
	if response != "[I Wanna go, OK?]" {
		t.Errorf("Expected the original text in <star/>, got %q", response)
	}
}

// When a substitution splits a word, consecutive wildcards still split the
// input on the user's own word boundaries
func TestWildcardsSplitOnOriginalWords(t *testing.T) {
	g := NewForTesting(t, false)
	if err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>SAVE * *</pattern><template>[<star/>][<star index="2"/>]</template></category>
</aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	g.aimlKB.Substitutions = map[string]map[string]string{"normal": {"wanna": "want to"}}
	session := g.CreateSession("original-split")

	response, err := g.ProcessInput("save Wanna x1", session)
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}
	if response != "[Wanna][x1]" {
		t.Errorf("Expected the wildcards split on the user's words, got %q", response)
	}
}
