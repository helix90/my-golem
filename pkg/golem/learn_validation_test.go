package golem

import (
	"strings"
	"testing"
)

// TestValidateLearnedCategory tests the comprehensive validation system
func TestValidateLearnedCategory(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	tests := []struct {
		name     string
		category Category
		wantErr  bool
		errMsg   string
	}{
		{
			name: "Valid category",
			category: Category{
				Pattern:  "HELLO *",
				Template: "Hello <star/>! How are you?",
			},
			wantErr: false,
		},
		{
			name: "Empty pattern",
			category: Category{
				Pattern:  "",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "pattern cannot be empty",
		},
		{
			name: "Empty template",
			category: Category{
				Pattern:  "HELLO",
				Template: "",
			},
			wantErr: true,
			errMsg:  "template cannot be empty",
		},
		{
			name: "Pattern too long",
			category: Category{
				Pattern:  generateLongString(1001),
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "pattern too long",
		},
		{
			name: "Template too long",
			category: Category{
				Pattern:  "HELLO",
				Template: generateLongString(10001),
			},
			wantErr: true,
			errMsg:  "template too long",
		},
		{
			name: "Too many wildcards",
			category: Category{
				Pattern:  "* * * * * * * * * * * *",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "too many wildcards",
		},
		{
			name: "Consecutive wildcards",
			category: Category{
				Pattern:  "HELLO **",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "consecutive wildcards",
		},
		{
			name: "Unbalanced parentheses",
			category: Category{
				Pattern:  "HELLO (world",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "unbalanced parentheses",
		},
		{
			name: "Invalid characters in pattern",
			category: Category{
				Pattern:  "HELLO @#$%",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name: "Dangerous content - script tag",
			category: Category{
				Pattern:  "HELLO",
				Template: "<script>alert('xss')</script>",
			},
			wantErr: true,
			errMsg:  "potentially dangerous content",
		},
		{
			name: "Too many SRAI tags",
			category: Category{
				Pattern:  "HELLO",
				Template: "<srai>test</srai><srai>test</srai><srai>test</srai><srai>test</srai><srai>test</srai><srai>test</srai>",
			},
			wantErr: true,
			errMsg:  "too many SRAI tags",
		},
		{
			name: "Too many wildcard references",
			category: Category{
				Pattern:  "HELLO",
				Template: "<star/><star/><star/><star/><star/><star/><star/><star/><star/><star/><star/>",
			},
			wantErr: true,
			errMsg:  "too many wildcard references",
		},
		{
			name: "Pattern too short",
			category: Category{
				Pattern:  "H",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "pattern too short",
		},
		{
			name: "Template too short",
			category: Category{
				Pattern:  "HELLO",
				Template: "H",
			},
			wantErr: true,
			errMsg:  "template too short",
		},
		{
			name: "Pattern too complex",
			category: Category{
				Pattern:  generateWordString(51),
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "pattern too complex",
		},
		{
			name: "Whitespace-only pattern",
			category: Category{
				Pattern:  "   ",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "pattern cannot be empty or whitespace only",
		},
		{
			name: "Whitespace-only template",
			category: Category{
				Pattern:  "HELLO",
				Template: "   ",
			},
			wantErr: true,
			errMsg:  "template cannot be empty or whitespace only",
		},
		{
			name: "Invalid alternation group - single option",
			category: Category{
				Pattern:  "HELLO (world)",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "alternation group must have at least 2 options",
		},
		{
			name: "Invalid alternation group - empty option",
			category: Category{
				Pattern:  "HELLO (world|)",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "empty option in alternation group",
		},
		{
			name: "Unknown AIML tag",
			category: Category{
				Pattern:  "HELLO",
				Template: "<unknown>test</unknown>",
			},
			wantErr: true,
			errMsg:  "unknown AIML tag",
		},
		{
			name: "Unbalanced tags",
			category: Category{
				Pattern:  "HELLO",
				Template: "<random><li>test</li>",
			},
			wantErr: true,
			errMsg:  "unbalanced tags",
		},
		{
			name: "Excessive nesting depth",
			category: Category{
				Pattern:  "HELLO",
				Template: generateNestedTags(21),
			},
			wantErr: true,
			errMsg:  "excessive nesting depth",
		},
		{
			name: "Valid alternation group",
			category: Category{
				Pattern:  "HELLO (world|universe|everyone)",
				Template: "Hello <star/>!",
			},
			wantErr: false,
		},
		{
			name: "Valid complex template",
			category: Category{
				Pattern:  "TELL ME ABOUT *",
				Template: "<think><set name=\"topic\"><star/></set></think>I know about <get name=\"topic\"/>.",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.ValidateLearnedCategory(tt.category)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateLearnedCategory() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !containsStringInError(err.Error(), tt.errMsg) {
					t.Errorf("validateLearnedCategory() error = %v, want error containing %s", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateLearnedCategory() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidatePatternSpecific tests pattern-specific validation
func TestValidatePatternSpecific(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name    string
		pattern string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid pattern with wildcards",
			pattern: "HELLO * WORLD",
			wantErr: false,
		},
		{
			name:    "Valid pattern with alternation",
			pattern: "HELLO (world|universe)",
			wantErr: false,
		},
		{
			name:    "Valid pattern with multiple wildcard types",
			pattern: "HELLO * _ ^ # $",
			wantErr: false,
		},
		{
			name:    "Pattern too long",
			pattern: generateLongString(1001),
			wantErr: true,
			errMsg:  "pattern too long",
		},
		{
			name:    "Too many wildcards",
			pattern: "* * * * * * * * * * * *",
			wantErr: true,
			errMsg:  "too many wildcards",
		},
		{
			name:    "Consecutive wildcards",
			pattern: "HELLO **",
			wantErr: true,
			errMsg:  "consecutive wildcards",
		},
		{
			name:    "Unbalanced parentheses",
			pattern: "HELLO (world",
			wantErr: true,
			errMsg:  "unbalanced parentheses",
		},
		{
			name:    "Invalid characters",
			pattern: "HELLO @#$%",
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name:    "Empty alternation group",
			pattern: "HELLO ()",
			wantErr: true,
			errMsg:  "alternation group must have at least 2 options",
		},
		{
			name:    "Single option alternation group",
			pattern: "HELLO (world)",
			wantErr: true,
			errMsg:  "alternation group must have at least 2 options",
		},
		{
			name:    "Empty option in alternation group",
			pattern: "HELLO (world|)",
			wantErr: true,
			errMsg:  "empty option in alternation group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.validatePatternStructure(tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePattern() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !containsStringInError(err.Error(), tt.errMsg) {
					t.Errorf("validatePattern() error = %v, want error containing %s", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validatePattern() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateTemplate tests template-specific validation
func TestValidateTemplate(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name     string
		template string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Valid template",
			template: "Hello <star/>! How are you?",
			wantErr:  false,
		},
		{
			name:     "Valid template with multiple tags",
			template: "<think><set name=\"topic\"><star/></set></think>I know about <get name=\"topic\"/>.",
			wantErr:  false,
		},
		{
			name:     "Template too long",
			template: generateLongString(10001),
			wantErr:  true,
			errMsg:   "template too long",
		},
		{
			name:     "Unbalanced tags",
			template: "<random><li>test</li>",
			wantErr:  true,
			errMsg:   "unbalanced tags",
		},
		{
			name:     "Unknown AIML tag",
			template: "<unknown>test</unknown>",
			wantErr:  true,
			errMsg:   "unknown AIML tag",
		},
		{
			name:     "Excessive nesting depth",
			template: generateNestedTags(25),
			wantErr:  true,
			errMsg:   "excessive nesting depth",
		},
		{
			name:     "Valid self-closing tags",
			template: "Hello <star/>! The time is <date/>.",
			wantErr:  false,
		},
		{
			name:     "Valid complex template",
			template: "<random><li>Option 1</li><li>Option 2</li></random>",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.validateTemplate(tt.template)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTemplate() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !containsStringInError(err.Error(), tt.errMsg) {
					t.Errorf("validateTemplate() error = %v, want error containing %s", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateTemplate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidateSecurity tests security validation
func TestValidateSecurity(t *testing.T) {
	g := NewForTesting(t, false)

	tests := []struct {
		name     string
		category Category
		wantErr  bool
		errMsg   string
	}{
		{
			name: "Safe content",
			category: Category{
				Pattern:  "HELLO",
				Template: "Hello! How are you?",
			},
			wantErr: false,
		},
		{
			name: "Script tag in pattern",
			category: Category{
				Pattern:  "<script>alert('xss')</script>",
				Template: "Hello!",
			},
			wantErr: true,
			errMsg:  "potentially dangerous content",
		},
		{
			name: "Script tag in template",
			category: Category{
				Pattern:  "HELLO",
				Template: "<script>alert('xss')</script>",
			},
			wantErr: true,
			errMsg:  "potentially dangerous content",
		},
		{
			name: "JavaScript URL",
			category: Category{
				Pattern:  "HELLO",
				Template: "javascript:alert('xss')",
			},
			wantErr: true,
			errMsg:  "potentially dangerous content",
		},
		{
			name: "Too many SRAI tags",
			category: Category{
				Pattern:  "HELLO",
				Template: "<srai>test</srai><srai>test</srai><srai>test</srai><srai>test</srai><srai>test</srai><srai>test</srai>",
			},
			wantErr: true,
			errMsg:  "too many SRAI tags",
		},
		{
			name: "Too many wildcard references",
			category: Category{
				Pattern:  "HELLO",
				Template: "<star/><star/><star/><star/><star/><star/><star/><star/><star/><star/><star/>",
			},
			wantErr: true,
			errMsg:  "too many wildcard references",
		},
		{
			name: "Valid SRAI usage",
			category: Category{
				Pattern:  "HELLO",
				Template: "<srai>GREETING</srai>",
			},
			wantErr: false,
		},
		{
			name: "Valid wildcard usage",
			category: Category{
				Pattern:  "HELLO *",
				Template: "Hello <star/>!",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.validateSecurity(tt.category)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateSecurity() expected error, got nil")
					return
				}
				if tt.errMsg != "" && !containsStringInError(err.Error(), tt.errMsg) {
					t.Errorf("validateSecurity() error = %v, want error containing %s", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateSecurity() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestLearnWithValidation tests that learning respects validation
func TestLearnWithValidation(t *testing.T) {
	g := NewForTesting(t, false)
	kb := NewAIMLKnowledgeBase()
	g.SetKnowledgeBase(kb)

	// Test valid learning
	validTemplate := `<learn>
		<category>
			<pattern>VALID TEST *</pattern>
			<template>Valid response for <star/></template>
		</category>
	</learn>`

	result := g.ProcessTemplate(validTemplate, make(map[string]string))
	if strings.Contains(result, "<learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// Check if category was added
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}

	// Test invalid learning
	invalidTemplate := `<learn>
		<category>
			<pattern></pattern>
			<template>Invalid response</template>
		</category>
	</learn>`

	result = g.ProcessTemplate(invalidTemplate, make(map[string]string))
	if strings.Contains(result, "<learn>") {
		t.Errorf("Learn tag not removed from template: %s", result)
	}

	// No new categories should be added due to validation error
	if len(kb.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(kb.Categories))
	}
}

// Helper functions

func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}

func generateWordString(wordCount int) string {
	words := make([]string, wordCount)
	for i := range words {
		words[i] = "word"
	}
	return strings.Join(words, " ")
}

func generateNestedTags(depth int) string {
	result := ""
	for i := 0; i < depth; i++ {
		result += "<random><li>"
	}
	result += "test"
	for i := 0; i < depth; i++ {
		result += "</li></random>"
	}
	return result
}

func containsStringInError(s, substr string) bool {
	return strings.Contains(s, substr)
}
