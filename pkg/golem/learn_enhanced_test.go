package golem

import (
	"strings"
	"testing"
)

// TestEnhancedLearnSessionManagement tests enhanced session management for learn
func TestEnhancedLearnSessionManagement(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session
	session := g.CreateSession("test_session")
	if session.LearningStats == nil {
		t.Fatal("Learning stats not initialized")
	}

	// Test learning with session context
	template := `<learn>
		<category>
			<pattern>ENHANCED SESSION TEST</pattern>
			<template>This is enhanced session learning</template>
		</category>
	</learn>`

	ctx := &VariableContext{
		Session: session,
	}

	result := g.processTemplateWithContext(template, make(map[string]string), ctx)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if the category was added to knowledge base
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}

	// Check if learning stats were updated
	if session.LearningStats.TotalLearned != 1 {
		t.Errorf("Expected 1 learned category, got %d", session.LearningStats.TotalLearned)
	}

	// Check if category was added to session learned categories
	if len(session.LearnedCategories) != 1 {
		t.Errorf("Expected 1 session learned category, got %d", len(session.LearnedCategories))
	}

	// Check pattern type categorization
	if session.LearningStats.PatternTypes["literal"] != 1 {
		t.Errorf("Expected 1 literal pattern, got %d", session.LearningStats.PatternTypes["literal"])
	}
}

// TestEnhancedLearnStatistics tests learning statistics tracking
func TestEnhancedLearnStatistics(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("stats_test")

	// Learn multiple categories with different pattern types
	templates := []string{
		`<learn><category><pattern>LITERAL PATTERN</pattern><template>Literal response</template></category></learn>`,
		`<learn><category><pattern>WILDCARD *</pattern><template>Wildcard response for <star/></template></category></learn>`,
		`<learn><category><pattern>ALTERNATION (a|b)</pattern><template>Alternation response</template></category></learn>`,
	}

	ctx := &VariableContext{Session: session}

	for _, template := range templates {
		_ = g.processTemplateWithContext(template, make(map[string]string), ctx)
	}

	// Check total learned
	if session.LearningStats.TotalLearned != 3 {
		t.Errorf("Expected 3 learned categories, got %d", session.LearningStats.TotalLearned)
	}

	// Check pattern type distribution
	expectedPatternTypes := map[string]int{
		"literal":     1,
		"wildcard":    1,
		"alternation": 1,
	}

	for patternType, expected := range expectedPatternTypes {
		if session.LearningStats.PatternTypes[patternType] != expected {
			t.Errorf("Expected %d %s patterns, got %d", expected, patternType, session.LearningStats.PatternTypes[patternType])
		}
	}

	// Check learning sources
	if session.LearningStats.LearningSources["learn"] != 3 {
		t.Errorf("Expected 3 learn operations, got %d", session.LearningStats.LearningSources["learn"])
	}

	// Check template lengths
	if len(session.LearningStats.TemplateLengths) != 3 {
		t.Errorf("Expected 3 template lengths, got %d", len(session.LearningStats.TemplateLengths))
	}
}

// TestEnhancedLearnValidationErrors tests validation error tracking
func TestEnhancedLearnValidationErrors(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("validation_test")

	// Try to learn invalid category
	template := `<learn>
		<category>
			<pattern>INVALID PATTERN WITH SCRIPT<script>alert('xss')</script></pattern>
			<template>Invalid response</template>
		</category>
	</learn>`

	ctx := &VariableContext{Session: session}
	_ = g.processTemplateWithContext(template, make(map[string]string), ctx)

	// Check validation errors were tracked
	if session.LearningStats.ValidationErrors != 1 {
		t.Errorf("Expected 1 validation error, got %d", session.LearningStats.ValidationErrors)
	}

	// Check no categories were learned
	if session.LearningStats.TotalLearned != 0 {
		t.Errorf("Expected 0 learned categories, got %d", session.LearningStats.TotalLearned)
	}
}

// TestEnhancedLearnUnlearnCycle tests learn/unlearn cycle with statistics
func TestEnhancedLearnUnlearnCycle(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("cycle_test")

	// Learn a category
	learnTemplate := `<learn>
		<category>
			<pattern>LEARN UNLEARN CYCLE</pattern>
			<template>This will be learned and then unlearned</template>
		</category>
	</learn>`

	ctx := &VariableContext{Session: session}
	_ = g.processTemplateWithContext(learnTemplate, make(map[string]string), ctx)

	// Check learning stats
	if session.LearningStats.TotalLearned != 1 {
		t.Errorf("Expected 1 learned category, got %d", session.LearningStats.TotalLearned)
	}

	// Unlearn the category
	unlearnTemplate := `<unlearn>
		<category>
			<pattern>LEARN UNLEARN CYCLE</pattern>
			<template>This will be learned and then unlearned</template>
		</category>
	</unlearn>`

	_ = g.processTemplateWithContext(unlearnTemplate, make(map[string]string), ctx)

	// Check unlearning stats
	if session.LearningStats.TotalUnlearned != 1 {
		t.Errorf("Expected 1 unlearned category, got %d", session.LearningStats.TotalUnlearned)
	}

	// Check that category was removed from session learned categories
	if len(session.LearnedCategories) != 0 {
		t.Errorf("Expected 0 session learned categories, got %d", len(session.LearnedCategories))
	}
}

// TestEnhancedLearnLearningRate tests learning rate calculation
func TestEnhancedLearnLearningRate(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("rate_test")

	// Learn multiple categories quickly
	templates := []string{
		`<learn><category><pattern>RATE TEST 1</pattern><template>Rate test 1</template></category></learn>`,
		`<learn><category><pattern>RATE TEST 2</pattern><template>Rate test 2</template></category></learn>`,
		`<learn><category><pattern>RATE TEST 3</pattern><template>Rate test 3</template></category></learn>`,
	}

	ctx := &VariableContext{Session: session}

	for _, template := range templates {
		_ = g.processTemplateWithContext(template, make(map[string]string), ctx)
	}

	// Check learning rate is calculated
	if session.LearningStats.LearningRate <= 0 {
		t.Errorf("Expected positive learning rate, got %f", session.LearningStats.LearningRate)
	}

	// Check total learned
	if session.LearningStats.TotalLearned != 3 {
		t.Errorf("Expected 3 learned categories, got %d", session.LearningStats.TotalLearned)
	}
}

// TestEnhancedLearnSessionIsolation tests session isolation
func TestEnhancedLearnSessionIsolation(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create two sessions
	session1 := g.CreateSession("session1")
	session2 := g.CreateSession("session2")

	// Learn different categories in each session
	template1 := `<learn>
		<category>
			<pattern>SESSION 1 PATTERN</pattern>
			<template>Session 1 response</template>
		</category>
	</learn>`

	template2 := `<learn>
		<category>
			<pattern>SESSION 2 PATTERN</pattern>
			<template>Session 2 response</template>
		</category>
	</learn>`

	ctx1 := &VariableContext{Session: session1}
	ctx2 := &VariableContext{Session: session2}

	_ = g.processTemplateWithContext(template1, make(map[string]string), ctx1)
	_ = g.processTemplateWithContext(template2, make(map[string]string), ctx2)

	// Check each session has its own learned categories
	if len(session1.LearnedCategories) != 1 {
		t.Errorf("Session 1: Expected 1 learned category, got %d", len(session1.LearnedCategories))
	}

	if len(session2.LearnedCategories) != 1 {
		t.Errorf("Session 2: Expected 1 learned category, got %d", len(session2.LearnedCategories))
	}

	// Check session 1 doesn't have session 2's category
	found := false
	for _, category := range session1.LearnedCategories {
		if category.Pattern == "SESSION 2 PATTERN" {
			found = true
			break
		}
	}
	if found {
		t.Error("Session 1 should not have session 2's learned category")
	}

	// Check session 2 doesn't have session 1's category
	found = false
	for _, category := range session2.LearnedCategories {
		if category.Pattern == "SESSION 1 PATTERN" {
			found = true
			break
		}
	}
	if found {
		t.Error("Session 2 should not have session 1's learned category")
	}
}

// TestEnhancedLearnSessionCleanup tests session learning cleanup
func TestEnhancedLearnSessionCleanup(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("cleanup_test")

	// Learn some categories
	templates := []string{
		`<learn><category><pattern>CLEANUP TEST 1</pattern><template>Cleanup test 1</template></category></learn>`,
		`<learn><category><pattern>CLEANUP TEST 2</pattern><template>Cleanup test 2</template></category></learn>`,
	}

	ctx := &VariableContext{Session: session}

	for _, template := range templates {
		_ = g.processTemplateWithContext(template, make(map[string]string), ctx)
	}

	// Check categories were learned
	if len(session.LearnedCategories) != 2 {
		t.Errorf("Expected 2 learned categories, got %d", len(session.LearnedCategories))
	}

	// Clear session learning
	err := g.ClearSessionLearning("cleanup_test")
	if err != nil {
		t.Fatalf("Failed to clear session learning: %v", err)
	}

	// Check categories were cleared
	if len(session.LearnedCategories) != 0 {
		t.Errorf("Expected 0 learned categories after cleanup, got %d", len(session.LearnedCategories))
	}

	// Check stats were reset
	if session.LearningStats.TotalLearned != 0 {
		t.Errorf("Expected 0 total learned after cleanup, got %d", session.LearningStats.TotalLearned)
	}
}

// TestEnhancedLearnGetSessionStats tests getting session learning statistics
func TestEnhancedLearnGetSessionStats(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("stats_get_test")

	// Learn some categories
	template := `<learn>
		<category>
			<pattern>STATS GET TEST</pattern>
			<template>Stats get test response</template>
		</category>
	</learn>`

	ctx := &VariableContext{Session: session}
	_ = g.processTemplateWithContext(template, make(map[string]string), ctx)

	// Get session stats
	stats, err := g.GetSessionLearningStats("stats_get_test")
	if err != nil {
		t.Fatalf("Failed to get session stats: %v", err)
	}

	if stats.TotalLearned != 1 {
		t.Errorf("Expected 1 learned category in stats, got %d", stats.TotalLearned)
	}

	// Get session learned categories
	categories, err := g.GetSessionLearnedCategories("stats_get_test")
	if err != nil {
		t.Fatalf("Failed to get session learned categories: %v", err)
	}

	if len(categories) != 1 {
		t.Errorf("Expected 1 learned category, got %d", len(categories))
	}

	if categories[0].Pattern != "STATS GET TEST" {
		t.Errorf("Expected pattern 'STATS GET TEST', got '%s'", categories[0].Pattern)
	}
}

// TestEnhancedLearnGetLearningSummary tests getting learning summary
func TestEnhancedLearnGetLearningSummary(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create multiple sessions and learn categories
	session1 := g.CreateSession("summary_session1")
	session2 := g.CreateSession("summary_session2")

	ctx1 := &VariableContext{Session: session1}
	ctx2 := &VariableContext{Session: session2}

	// Learn in session 1
	template1 := `<learn>
		<category>
			<pattern>SUMMARY SESSION 1</pattern>
			<template>Summary session 1 response</template>
		</category>
	</learn>`

	// Learn in session 2
	template2 := `<learn>
		<category>
			<pattern>SUMMARY SESSION 2</pattern>
			<template>Summary session 2 response</template>
		</category>
	</learn>`

	_ = g.processTemplateWithContext(template1, make(map[string]string), ctx1)
	_ = g.processTemplateWithContext(template2, make(map[string]string), ctx2)

	// Get learning summary
	summary := g.GetLearningSummary()

	// Check summary structure
	if summary["total_sessions"] != 2 {
		t.Errorf("Expected 2 total sessions, got %v", summary["total_sessions"])
	}

	if summary["total_categories"] != 2 {
		t.Errorf("Expected 2 total categories, got %v", summary["total_categories"])
	}

	// Check global stats
	globalStats := summary["global_stats"].(map[string]interface{})
	if globalStats["total_learned"] != 2 {
		t.Errorf("Expected 2 total learned globally, got %v", globalStats["total_learned"])
	}

	// Check session stats
	sessionStats := summary["session_stats"].(map[string]interface{})
	if len(sessionStats) != 2 {
		t.Errorf("Expected 2 session stats entries, got %d", len(sessionStats))
	}
}

// TestEnhancedLearnPatternCategorization tests pattern categorization
func TestEnhancedLearnPatternCategorization(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("categorization_test")

	// Test different pattern types
	testCases := []struct {
		pattern      string
		expectedType string
	}{
		{"LITERAL PATTERN", "literal"},
		{"WILDCARD *", "wildcard"},
		{"UNDERSCORE _", "underscore"},
		{"HASH #", "hash"},
		{"DOLLAR $", "dollar"},
		{"ALTERNATION (a|b)", "alternation"},
	}

	ctx := &VariableContext{Session: session}

	for i, tc := range testCases {
		template := `<learn>
			<category>
				<pattern>` + tc.pattern + `</pattern>
				<template>Test response ` + string(rune('0'+i)) + `</template>
			</category>
		</learn>`

		_ = g.processTemplateWithContext(template, make(map[string]string), ctx)
	}

	// Check pattern type distribution
	for _, tc := range testCases {
		if session.LearningStats.PatternTypes[tc.expectedType] != 1 {
			t.Errorf("Expected 1 %s pattern, got %d", tc.expectedType, session.LearningStats.PatternTypes[tc.expectedType])
		}
	}
}

// TestEnhancedLearnWithThatContext tests learn with that context
func TestEnhancedLearnWithThatContext(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("that_context_test")

	// Test learning with that context
	template := `<learn>
		<category>
			<pattern>THAT CONTEXT TEST</pattern>
			<that>WHAT IS YOUR NAME</that>
			<template>My name is Bot, and I remember that context</template>
		</category>
	</learn>`

	ctx := &VariableContext{Session: session}
	result := g.processTemplateWithContext(template, make(map[string]string), ctx)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if category with that context was added
	if len(session.LearnedCategories) != 1 {
		t.Errorf("Expected 1 learned category, got %d", len(session.LearnedCategories))
	}

	category := session.LearnedCategories[0]
	if category.That != "WHAT IS YOUR NAME" {
		t.Errorf("Expected that context 'WHAT IS YOUR NAME', got '%s'", category.That)
	}
}

// TestEnhancedLearnWithTopic tests learn with topic
func TestEnhancedLearnWithTopic(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("topic_test")

	// Test learning with topic
	template := `<learn>
		<category>
			<pattern>TOPIC TEST</pattern>
			<topic>LEARNING TOPIC</topic>
			<template>This is about learning topics</template>
		</category>
	</learn>`

	ctx := &VariableContext{
		Session:       session,
		KnowledgeBase: g.aimlKB,
	}
	result := g.processTemplateWithContext(template, make(map[string]string), ctx)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if category with topic was added
	if len(session.LearnedCategories) != 1 {
		t.Errorf("Expected 1 learned category, got %d", len(session.LearnedCategories))
	}

	category := session.LearnedCategories[0]
	if category.Topic != "LEARNING TOPIC" {
		t.Errorf("Expected topic 'LEARNING TOPIC', got '%s'", category.Topic)
	}
}

// TestEnhancedLearnErrorHandling tests error handling in enhanced learn
func TestEnhancedLearnErrorHandling(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	session := g.CreateSession("error_test")

	// Test learn with empty content
	template := `<learn></learn>`

	ctx := &VariableContext{Session: session}
	result := g.processTemplateWithContext(template, make(map[string]string), ctx)

	// The learn tag should be removed after processing
	if strings.Contains(result, "<learn>") || strings.Contains(result, "</learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// No categories should be learned
	if len(session.LearnedCategories) != 0 {
		t.Errorf("Expected 0 learned categories, got %d", len(session.LearnedCategories))
	}
}

// TestEnhancedLearnConcurrentSessions tests concurrent session learning
func TestEnhancedLearnConcurrentSessions(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create multiple sessions
	sessions := make([]*ChatSession, 3)
	for i := 0; i < 3; i++ {
		sessions[i] = g.CreateSession("concurrent_session_" + string(rune('0'+i)))
	}

	// Learn categories in each session
	for i, session := range sessions {
		template := `<learn>
			<category>
				<pattern>CONCURRENT SESSION ` + string(rune('0'+i)) + `</pattern>
				<template>Concurrent session ` + string(rune('0'+i)) + ` response</template>
			</category>
		</learn>`

		ctx := &VariableContext{Session: session}
		_ = g.processTemplateWithContext(template, make(map[string]string), ctx)
	}

	// Check each session has its own learned category
	for i, session := range sessions {
		if len(session.LearnedCategories) != 1 {
			t.Errorf("Session %d: Expected 1 learned category, got %d", i, len(session.LearnedCategories))
		}

		if session.LearningStats.TotalLearned != 1 {
			t.Errorf("Session %d: Expected 1 total learned, got %d", i, session.LearningStats.TotalLearned)
		}
	}
}
