package golem

import (
	"testing"
)

// TestLocalVariableToSessionVariable tests that local variables can be copied to session variables
func TestLocalVariableToSessionVariable(t *testing.T) {
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
  <category>
    <pattern>TEST VAR COPY *</pattern>
    <template>
      <think>
        <set var="localvar"><star/></set>
        <set name="sessionvar"><get var="localvar"/></set>
      </think>
      Local: <get var="localvar"/>, Session: <get name="sessionvar"/>
    </template>
  </category>
</aiml>`

	g := New(false)
	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test_local_var")

	response, err := g.ProcessInput("test var copy HELLO", session)
	if err != nil {
		t.Fatalf("Error processing input: %v", err)
	}

	t.Logf("Response: %s", response)
	t.Logf("Session variable: %s", session.Variables["sessionvar"])

	// Check that the session variable was set
	if session.Variables["sessionvar"] != "HELLO" {
		t.Errorf("sessionvar = %s, expected 'HELLO'", session.Variables["sessionvar"])
	}

	// Check that the response includes both values
	if response != "Local: HELLO, Session: HELLO" {
		t.Errorf("Response = %s, expected 'Local: HELLO, Session: HELLO'", response)
	}
}

// TestLocalVarWithSameName tests that local and session variables with the same name don't conflict
func TestLocalVarWithSameName(t *testing.T) {
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
  <category>
    <pattern>TEST SAME NAME *</pattern>
    <template>
      <think>
        <set var="location"><star/></set>
        <set name="location"><get var="location"/></set>
      </think>
      Local: <get var="location"/>, Session: <get name="location"/>
    </template>
  </category>
</aiml>`

	g := New(false)
	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test_same_name")

	response, err := g.ProcessInput("test same name Seattle", session)
	if err != nil {
		t.Fatalf("Error processing input: %v", err)
	}

	t.Logf("Response: %s", response)
	t.Logf("Session variable location: %s", session.Variables["location"])

	// Check that the session variable was set correctly
	if session.Variables["location"] != "Seattle" {
		t.Errorf("location = '%s', expected 'Seattle'", session.Variables["location"])
	}
}

// TestMultipleLocalVariables tests that multiple local variables work correctly
func TestMultipleLocalVariables(t *testing.T) {
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
  <category>
    <pattern>TEST MULTI VAR * *</pattern>
    <template>
      <think>
        <set var="var1"><star index="1"/></set>
        <set var="var2"><star index="2"/></set>
        <set name="session1"><get var="var1"/></set>
        <set name="session2"><get var="var2"/></set>
      </think>
      Var1: <get var="var1"/>, Var2: <get var="var2"/>, Session1: <get name="session1"/>, Session2: <get name="session2"/>
    </template>
  </category>
</aiml>`

	g := New(false)
	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test_multi_var")

	response, err := g.ProcessInput("test multi var FIRST SECOND", session)
	if err != nil {
		t.Fatalf("Error processing input: %v", err)
	}

	t.Logf("Response: %s", response)
	t.Logf("Session variables: session1=%s, session2=%s", session.Variables["session1"], session.Variables["session2"])

	// Check session variables
	if session.Variables["session1"] != "FIRST" {
		t.Errorf("session1 = %s, expected 'FIRST'", session.Variables["session1"])
	}
	if session.Variables["session2"] != "SECOND" {
		t.Errorf("session2 = %s, expected 'SECOND'", session.Variables["session2"])
	}
}

// TestLocalVarAfterGetVar tests that a local variable remains accessible after being used in a <get var>
func TestLocalVarAfterGetVar(t *testing.T) {
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
  <category>
    <pattern>TEST VAR PERSIST *</pattern>
    <template>
      <think>
        <set var="value1"><star/></set>
        <set var="value2"><get var="value1"/></set>
        <set var="value3"><get var="value1"/></set>
        <set name="session1"><get var="value1"/></set>
        <set name="session2"><get var="value2"/></set>
        <set name="session3"><get var="value3"/></set>
      </think>
      V1: <get var="value1"/>, V2: <get var="value2"/>, V3: <get var="value3"/>
    </template>
  </category>
</aiml>`

	g := New(false)
	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	session := g.CreateSession("test_var_persist")

	response, err := g.ProcessInput("test var persist TESTVALUE", session)
	if err != nil {
		t.Fatalf("Error processing input: %v", err)
	}

	t.Logf("Response: %s", response)
	t.Logf("Session variables: s1=%s, s2=%s, s3=%s",
		session.Variables["session1"],
		session.Variables["session2"],
		session.Variables["session3"])

	// All three session variables should have the same value
	if session.Variables["session1"] != "TESTVALUE" {
		t.Errorf("session1 = '%s', expected 'TESTVALUE'", session.Variables["session1"])
	}
	if session.Variables["session2"] != "TESTVALUE" {
		t.Errorf("session2 = '%s', expected 'TESTVALUE'", session.Variables["session2"])
	}
	if session.Variables["session3"] != "TESTVALUE" {
		t.Errorf("session3 = '%s', expected 'TESTVALUE'", session.Variables["session3"])
	}
}
