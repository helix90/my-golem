package golem

import (
	"regexp"
	"strings"
	"testing"
)

func TestTimeBasedGreetingWithCondition(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test time-based greetings using <time> with %H format and <condition> tags
	// This tests the integration of time processing with conditional logic
	template := `<condition name="time_of_day">
	<li value="morning">Good Morning! The time is <time format="%H:%M"/>.</li>
	<li value="afternoon">Good Afternoon! The time is <time format="%H:%M"/>.</li>
	<li value="evening">Good Evening! The time is <time format="%H:%M"/>.</li>
	<li value="night">Good Night! The time is <time format="%H:%M"/>.</li>
	<li>Hello! The time is <time format="%H:%M"/>.</li>
</condition>`

	// Test different time periods by setting the time_of_day variable
	testCases := []struct {
		timeOfDay      string
		expectedPrefix string
		description    string
	}{
		{"morning", "Good Morning!", "Morning greeting (6-11 AM)"},
		{"afternoon", "Good Afternoon!", "Afternoon greeting (12-17 PM)"},
		{"evening", "Good Evening!", "Evening greeting (18-21 PM)"},
		{"night", "Good Night!", "Night greeting (22-5 AM)"},
		{"", "Hello!", "Default greeting when time_of_day is not set"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Set the time_of_day variable
			if tc.timeOfDay != "" {
				kb.Variables["time_of_day"] = tc.timeOfDay
			} else {
				delete(kb.Variables, "time_of_day")
			}

			// Process the template
			result := g.processConditionTags(template, nil)

			// Verify the greeting prefix is correct
			if !strings.HasPrefix(result, tc.expectedPrefix) {
				t.Errorf("Expected result to start with '%s', got '%s'", tc.expectedPrefix, result)
			}

			// Verify that time tag was processed (should contain a time format like HH:MM)
			if !strings.Contains(result, ":") {
				t.Errorf("Expected result to contain time format, got '%s'", result)
			}

			// Verify the time format is correct (should be HH:MM)
			timeRegex := regexp.MustCompile(`\d{1,2}:\d{2}`)
			if !timeRegex.MatchString(result) {
				t.Errorf("Expected result to contain time in HH:MM format, got '%s'", result)
			}

			// Verify the complete expected structure
			expectedSuffix := "The time is " + timeRegex.FindString(result) + "."
			if !strings.HasSuffix(result, expectedSuffix) {
				t.Errorf("Expected result to end with '%s', got '%s'", expectedSuffix, result)
			}
		})
	}
}

func TestTimeBasedGreetingWithDirectTimeCondition(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Create a session for testing
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    "2024-01-01T00:00:00Z",
		LastActivity: "2024-01-01T00:00:00Z",
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}

	// Test a more complex scenario where we use the actual hour value from <time format="%H">
	// and then use that in a condition to determine the greeting
	template := `<set name="current_hour"><time format="%H"/></set><condition name="current_hour">
	<li value="06">Good Morning! It's 6 AM.</li>
	<li value="07">Good Morning! It's 7 AM.</li>
	<li value="08">Good Morning! It's 8 AM.</li>
	<li value="09">Good Morning! It's 9 AM.</li>
	<li value="10">Good Morning! It's 10 AM.</li>
	<li value="11">Good Morning! It's 11 AM.</li>
	<li value="12">Good Afternoon! It's 12 PM.</li>
	<li value="13">Good Afternoon! It's 1 PM.</li>
	<li value="14">Good Afternoon! It's 2 PM.</li>
	<li value="15">Good Afternoon! It's 3 PM.</li>
	<li value="16">Good Afternoon! It's 4 PM.</li>
	<li value="17">Good Afternoon! It's 5 PM.</li>
	<li value="18">Good Evening! It's 6 PM.</li>
	<li value="19">Good Evening! It's 7 PM.</li>
	<li value="20">Good Evening! It's 8 PM.</li>
	<li value="21">Good Evening! It's 9 PM.</li>
	<li value="22">Good Night! It's 10 PM.</li>
	<li value="23">Good Night! It's 11 PM.</li>
	<li value="00">Good Night! It's 12 AM.</li>
	<li value="01">Good Night! It's 1 AM.</li>
	<li value="02">Good Night! It's 2 AM.</li>
	<li value="03">Good Night! It's 3 AM.</li>
	<li value="04">Good Night! It's 4 AM.</li>
	<li value="05">Good Night! It's 5 AM.</li>
	<li>Hello! The current hour is <get name="current_hour"/>.</li>
</condition>`

	// Process the template using session context
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session)

	// Verify that the result contains a greeting
	greetingPatterns := []string{
		"Good Morning!", "Good Afternoon!", "Good Evening!", "Good Night!", "Hello!",
	}

	foundGreeting := false
	for _, pattern := range greetingPatterns {
		if strings.Contains(result, pattern) {
			foundGreeting = true
			break
		}
	}

	if !foundGreeting {
		t.Errorf("Expected result to contain a greeting, got '%s'", result)
	}

	// Verify that the time was processed and stored in the session variable
	if session.Variables["current_hour"] == "" {
		t.Errorf("Expected current_hour variable to be set in session")
	}

	// Verify the hour format is correct (should be 2 digits)
	hourRegex := regexp.MustCompile(`^\d{2}$`)
	if !hourRegex.MatchString(session.Variables["current_hour"]) {
		t.Errorf("Expected current_hour to be 2 digits, got '%s'", session.Variables["current_hour"])
	}

	// Verify the hour is in valid range (00-23)
	hour := session.Variables["current_hour"]
	if hour < "00" || hour > "23" {
		t.Errorf("Expected current_hour to be between 00-23, got '%s'", hour)
	}
}

func TestTimeBasedGreetingWithNestedConditions(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test nested conditions with time-based greetings
	// This tests more complex conditional logic with time processing
	template := `<set name="current_hour"><time format="%H"/></set><condition name="user_type">
	<li value="admin">Welcome admin! <condition name="current_hour">
		<li value="06">Good morning, admin! It's <time format="%H:%M"/>.</li>
		<li value="12">Good afternoon, admin! It's <time format="%H:%M"/>.</li>
		<li value="18">Good evening, admin! It's <time format="%H:%M"/>.</li>
		<li>Hello admin! It's <time format="%H:%M"/>.</li>
	</condition></li>
	<li value="user">Hello user! <condition name="current_hour">
		<li value="06">Good morning! It's <time format="%H:%M"/>.</li>
		<li value="12">Good afternoon! It's <time format="%H:%M"/>.</li>
		<li value="18">Good evening! It's <time format="%H:%M"/>.</li>
		<li>Hello! It's <time format="%H:%M"/>.</li>
	</condition></li>
	<li>Welcome guest! It's <time format="%H:%M"/>.</li>
</condition>`

	// Test with admin user
	session1 := &ChatSession{
		ID:           "test-session-1",
		Variables:    map[string]string{"user_type": "admin"},
		History:      make([]string, 0),
		CreatedAt:    "2024-01-01T00:00:00Z",
		LastActivity: "2024-01-01T00:00:00Z",
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	result := g.ProcessTemplateWithSession(template, make(map[string]string), session1)

	// Verify admin greeting
	if !strings.Contains(result, "Welcome admin!") {
		t.Errorf("Expected result to contain admin greeting, got '%s'", result)
	}

	// Verify time was processed
	if !strings.Contains(result, ":") {
		t.Errorf("Expected result to contain time format, got '%s'", result)
	}

	// Test with user type
	session2 := &ChatSession{
		ID:           "test-session-2",
		Variables:    map[string]string{"user_type": "user"},
		History:      make([]string, 0),
		CreatedAt:    "2024-01-01T00:00:00Z",
		LastActivity: "2024-01-01T00:00:00Z",
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	result = g.ProcessTemplateWithSession(template, make(map[string]string), session2)

	// Verify user greeting
	if !strings.Contains(result, "Hello user!") {
		t.Errorf("Expected result to contain user greeting, got '%s'", result)
	}

	// Test with no user type (guest)
	session3 := &ChatSession{
		ID:           "test-session-3",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    "2024-01-01T00:00:00Z",
		LastActivity: "2024-01-01T00:00:00Z",
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}
	result = g.ProcessTemplateWithSession(template, make(map[string]string), session3)

	// Verify guest greeting
	if !strings.Contains(result, "Welcome guest!") {
		t.Errorf("Expected result to contain guest greeting, got '%s'", result)
	}
}

func TestTimeBasedGreetingWithAIMLIntegration(t *testing.T) {
	g := NewForTesting(t, false)

	// Load test AIML with time-based greeting patterns
	aimlContent := `<?xml version="1.0" encoding="UTF-8"?>
<aiml version="2.0">
<category>
<pattern>HELLO</pattern>
<template><set name="current_hour"><time format="%H"/></set><condition name="current_hour">
	<li value="06">Good Morning! It's <time format="%H:%M"/>.</li>
	<li value="07">Good Morning! It's <time format="%H:%M"/>.</li>
	<li value="08">Good Morning! It's <time format="%H:%M"/>.</li>
	<li value="09">Good Morning! It's <time format="%H:%M"/>.</li>
	<li value="10">Good Morning! It's <time format="%H:%M"/>.</li>
	<li value="11">Good Morning! It's <time format="%H:%M"/>.</li>
	<li value="12">Good Afternoon! It's <time format="%H:%M"/>.</li>
	<li value="13">Good Afternoon! It's <time format="%H:%M"/>.</li>
	<li value="14">Good Afternoon! It's <time format="%H:%M"/>.</li>
	<li value="15">Good Afternoon! It's <time format="%H:%M"/>.</li>
	<li value="16">Good Afternoon! It's <time format="%H:%M"/>.</li>
	<li value="17">Good Afternoon! It's <time format="%H:%M"/>.</li>
	<li value="18">Good Evening! It's <time format="%H:%M"/>.</li>
	<li value="19">Good Evening! It's <time format="%H:%M"/>.</li>
	<li value="20">Good Evening! It's <time format="%H:%M"/>.</li>
	<li value="21">Good Evening! It's <time format="%H:%M"/>.</li>
	<li value="22">Good Night! It's <time format="%H:%M"/>.</li>
	<li value="23">Good Night! It's <time format="%H:%M"/>.</li>
	<li value="00">Good Night! It's <time format="%H:%M"/>.</li>
	<li value="01">Good Night! It's <time format="%H:%M"/>.</li>
	<li value="02">Good Night! It's <time format="%H:%M"/>.</li>
	<li value="03">Good Night! It's <time format="%H:%M"/>.</li>
	<li value="04">Good Night! It's <time format="%H:%M"/>.</li>
	<li value="05">Good Night! It's <time format="%H:%M"/>.</li>
	<li>Hello! It's <time format="%H:%M"/>.</li>
</condition></template>
</category>
</aiml>`

	err := g.LoadAIMLFromString(aimlContent)
	if err != nil {
		t.Fatalf("Failed to load AIML: %v", err)
	}

	// Create a session for testing
	session := &ChatSession{
		ID:           "test-session",
		Variables:    make(map[string]string),
		History:      make([]string, 0),
		CreatedAt:    "2024-01-01T00:00:00Z",
		LastActivity: "2024-01-01T00:00:00Z",
		Topic:        "",
		ThatHistory:  make([]string, 0),
	}

	// Test the greeting with different times
	response, err := g.ProcessInput("hello", session)
	if err != nil {
		t.Fatalf("Failed to process input: %v", err)
	}

	// Verify that the result contains a greeting
	greetingPatterns := []string{
		"Good Morning!", "Good Afternoon!", "Good Evening!", "Good Night!", "Hello!",
	}

	foundGreeting := false
	for _, pattern := range greetingPatterns {
		if strings.Contains(response, pattern) {
			foundGreeting = true
			break
		}
	}

	if !foundGreeting {
		t.Errorf("Expected result to contain a greeting, got '%s'", response)
	}

	// Verify that time was processed
	if !strings.Contains(response, ":") {
		t.Errorf("Expected result to contain time format, got '%s'", response)
	}

	// Verify the time format is correct (should be HH:MM)
	timeRegex := regexp.MustCompile(`\d{1,2}:\d{2}`)
	if !timeRegex.MatchString(response) {
		t.Errorf("Expected result to contain time in HH:MM format, got '%s'", response)
	}
}
