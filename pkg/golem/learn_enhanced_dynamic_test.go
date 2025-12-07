package golem

import (
	"fmt"
	"strings"
	"testing"
)

// TestLearnTagDynamicEvaluation tests the enhanced learn tag with dynamic evaluation
func TestLearnTagDynamicEvaluation(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session for testing
	session := g.CreateSession("test_session")

	// Set up variables for dynamic learning
	g.ProcessTemplateWithContext(`<set name="pattern1">HELLO *</set>`, map[string]string{}, session)
	g.ProcessTemplateWithContext(`<set name="response1">Hi there! How can I help you?</set>`, map[string]string{}, session)

	// Test dynamic learn tag with eval tags
	template := `<learn>
		<category>
			<pattern><eval><get name="pattern1"/></eval></pattern>
			<template><eval><get name="response1"/></eval></template>
		</category>
	</learn>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Test that the learned category works
	testInput := "HELLO WORLD"
	response, err := g.ProcessInput(testInput, session)
	if err != nil {
		t.Errorf("Error processing input: %v", err)
	}

	expectedResponse := "Hi there! How can I help you?"
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}
}

// TestLearnTagMultipleDynamicCategories tests learning multiple categories (AST behavior)
func TestLearnTagMultipleDynamicCategories(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// AST Behavior: Learn multiple categories with literal patterns
	// Wildcards are preserved in the pattern/template structure
	template := `<learn>
		<category>
			<pattern>WHAT IS *</pattern>
			<template>I don't know what <star/> is.</template>
		</category>
		<category>
			<pattern>TELL ME ABOUT *</pattern>
			<template>Let me tell you about <star/>.</template>
		</category>
	</learn>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Test both learned categories
	testCases := []struct {
		input    string
		expected string
	}{
		{"WHAT IS AI", "I don't know what AI is."},
		{"TELL ME ABOUT MACHINE LEARNING", "Let me tell you about MACHINE LEARNING."},
	}

	for _, tc := range testCases {
		response, err := g.ProcessInput(tc.input, session)
		if err != nil {
			t.Errorf("Error processing input '%s': %v", tc.input, err)
			continue
		}

		if response != tc.expected {
			t.Errorf("For input '%s', expected response '%s', got '%s'", tc.input, tc.expected, response)
		}
	}
}

// TestLearnTagWithComplexEval tests learn tag with eval and formatting (AST behavior)
func TestLearnTagWithComplexEval(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// Set up variables with text values (not AIML code)
	session.Variables["greeting_word"] = "hello"
	session.Variables["name_var"] = "friend"

	// AST Behavior: Eval can insert variable values, formatting applies during learning
	template := `<learn>
		<category>
			<pattern>GREET *</pattern>
			<template><eval><uppercase><get name="greeting_word"/></uppercase></eval> <star/>! Nice to meet you.</template>
		</category>
	</learn>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Test the learned category
	testInput := "GREET ALICE"
	response, err := g.ProcessInput(testInput, session)
	if err != nil {
		t.Errorf("Error processing input: %v", err)
	}

	// The eval/uppercase was applied during learning, converting "hello" to "HELLO"
	// The star wildcard gets the runtime value "ALICE"
	expectedResponse := "HELLO ALICE! Nice to meet you."
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}
}

// TestLearnTagWithWildcards tests learn tag with wildcard patterns (AST behavior)
func TestLearnTagWithWildcards(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// AST Behavior: Learn tags should contain literal AIML, not dynamic evaluation
	// of stored text. Wildcards in templates are preserved for runtime matching.
	template := `<learn>
		<category>
			<pattern>* LIKES *</pattern>
			<template><star index="1"/> likes <star index="2"/>.</template>
		</category>
	</learn>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Test the learned category with wildcards
	testInput := "ALICE LIKES CHOCOLATE"
	response, err := g.ProcessInput(testInput, session)
	if err != nil {
		t.Errorf("Error processing input: %v", err)
	}

	expectedResponse := "ALICE likes CHOCOLATE."
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}
}

// TestLearnTagWithConditionalEval tests learn tag with eval for dynamic values (AST behavior)
func TestLearnTagWithConditionalEval(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// Set up variables with actual text values (not AIML code)
	session.Variables["subject"] = "PROGRAMMING"

	// AST Behavior: Eval can be used to insert variable values into patterns at learn time
	// The pattern itself is constructed dynamically, but wildcards are still preserved
	template := `<learn>
		<category>
			<pattern>WHAT IS <eval><get name="subject"/></eval></pattern>
			<template><get name="subject"/> is a skill that can be learned.</template>
		</category>
	</learn>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Test the learned category
	testInput := "WHAT IS PROGRAMMING"
	response, err := g.ProcessInput(testInput, session)
	if err != nil {
		t.Errorf("Error processing input: %v", err)
	}

	expectedResponse := "PROGRAMMING is a skill that can be learned."
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}
}

// TestLearnTagErrorHandling tests error handling in dynamic learn tags
func TestLearnTagErrorHandling(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// Test with invalid eval content
	template := `<learn>
		<category>
			<pattern><eval><get name="nonexistent"/></eval></pattern>
			<template><eval><get name="nonexistent"/></eval></template>
		</category>
	</learn>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learn tag should be removed even on error
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Test with malformed eval content
	template2 := `<learn>
		<category>
			<pattern><eval>INVALID SYNTAX</eval></pattern>
			<template><eval>INVALID SYNTAX</eval></template>
		</category>
	</learn>`

	result2 := g.ProcessTemplateWithContext(template2, map[string]string{}, session)

	// The learn tag should be removed even on error
	if strings.Contains(result2, "<learn>") || strings.Contains(result2, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result2)
	}
}

// TestLearnfTagDynamicEvaluation tests the enhanced learnf tag with dynamic evaluation
func TestLearnfTagDynamicEvaluation(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// AST Behavior: Learnf with literal patterns for persistent learning
	// Wildcards are preserved in the pattern structure
	template := `<learnf>
		<category>
			<pattern>PERSISTENT *</pattern>
			<template>This is a persistent response for <star/>.</template>
		</category>
	</learnf>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learnf tag should be removed after processing
	if strings.Contains(result, "<learnf>") || strings.Contains(result, "</learnf>") {
		t.Errorf("Learnf tag not removed from template: %s", result)
	}

	// Test that the learned category works
	testInput := "PERSISTENT TEST"
	response, err := g.ProcessInput(testInput, session)
	if err != nil {
		t.Errorf("Error processing input: %v", err)
	}

	expectedResponse := "This is a persistent response for TEST."
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}
}

// TestLearnTagIntegrationWithOtherTags tests integration with other tag types
func TestLearnTagIntegrationWithOtherTags(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// Set up variables using other tag types
	g.ProcessTemplateWithContext(`<set name="time_pattern">WHAT TIME IS IT</set>`, map[string]string{}, session)
	g.ProcessTemplateWithContext(`<set name="time_response">The current time is <date/>.</set>`, map[string]string{}, session)

	// Test integration with other tags
	template := `<learn>
		<category>
			<pattern><eval><uppercase><get name="time_pattern"/></uppercase></eval></pattern>
			<template><eval><get name="time_response"/></eval></template>
		</category>
	</learn>`

	result := g.ProcessTemplateWithContext(template, map[string]string{}, session)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Test the learned category
	testInput := "what time is it"
	response, err := g.ProcessInput(testInput, session)
	if err != nil {
		t.Errorf("Error processing input: %v", err)
	}

	// The response should contain a date (we can't predict the exact format)
	if !strings.Contains(response, "The current time is") {
		t.Errorf("Expected response to contain 'The current time is', got '%s'", response)
	}
}

// TestLearnTagPerformance tests performance with multiple dynamic categories
func TestLearnTagPerformance(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("test_session")

	// AST Behavior: Create a large learn template with literal patterns
	// Tests performance of learning many categories at once
	var learnContent strings.Builder
	learnContent.WriteString("<learn>\n")
	for i := 0; i < 10; i++ {
		learnContent.WriteString(fmt.Sprintf(`
		<category>
			<pattern>PATTERN%d *</pattern>
			<template>Response %d for <star/>.</template>
		</category>`, i, i))
	}
	learnContent.WriteString("\n</learn>")

	// Test performance
	result := g.ProcessTemplateWithContext(learnContent.String(), map[string]string{}, session)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template")
	}

	// Test a few learned categories
	testCases := []struct {
		input    string
		expected string
	}{
		{"PATTERN0 TEST", "Response 0 for TEST."},
		{"PATTERN5 TEST", "Response 5 for TEST."},
		{"PATTERN9 TEST", "Response 9 for TEST."},
	}

	for _, tc := range testCases {
		response, err := g.ProcessInput(tc.input, session)
		if err != nil {
			t.Errorf("Error processing input '%s': %v", tc.input, err)
			continue
		}

		if response != tc.expected {
			t.Errorf("For input '%s', expected response '%s', got '%s'", tc.input, tc.expected, response)
		}
	}
}
