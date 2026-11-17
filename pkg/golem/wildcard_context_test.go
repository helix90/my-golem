package golem

import "testing"

// Wildcards live in the template's context, not in session variables, so a
// predicate named like a wildcard neither shadows it nor is changed by it
func TestWildcardsAreNotSessionVariables(t *testing.T) {
	g := NewForTesting(t, false)
	if err := g.LoadAIMLFromString(`<aiml version="2.0">
<category><pattern>ECHO *</pattern><template>[<star/>][<get name="star1"/>]</template></category>
</aiml>`); err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}
	session := g.CreateSession("wildcard-context")
	session.Variables["star1"] = "predicate"

	response, err := g.ProcessInput("echo hello", session)
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}
	if response != "[hello][predicate]" {
		t.Errorf("Expected the wildcard and the predicate separately, got %q", response)
	}
	if session.Variables["star1"] != "predicate" {
		t.Errorf("Expected the predicate unchanged, got %q", session.Variables["star1"])
	}
}
