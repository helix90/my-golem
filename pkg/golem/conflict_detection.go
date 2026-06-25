package golem

import (
	"fmt"
	"strings"
)

// ConflictDetection provides pattern conflict detection functionality
type ConflictDetection struct {
	golem *Golem
}

// NewConflictDetection creates a new conflict detection instance
func NewConflictDetection(golem *Golem) *ConflictDetection {
	return &ConflictDetection{golem: golem}
}

// DetectConflicts detects all types of pattern conflicts
func (cd *ConflictDetection) DetectConflicts(detector *ThatPatternConflictDetector) []ThatPatternConflict {
	detector.Conflicts = []ThatPatternConflict{}

	// Check for various types of conflicts
	cd.detectOverlapConflicts(detector)
	cd.detectAmbiguityConflicts(detector)
	cd.detectPriorityConflicts(detector)
	cd.detectWildcardConflicts(detector)
	cd.detectSpecificityConflicts(detector)

	return detector.Conflicts
}

// detectOverlapConflicts detects patterns that overlap in matching scope
func (cd *ConflictDetection) detectOverlapConflicts(detector *ThatPatternConflictDetector) {
	for i, pattern1 := range detector.Patterns {
		for j, pattern2 := range detector.Patterns {
			if i >= j {
				continue // Skip same pattern and avoid duplicates
			}

			// Check if patterns overlap
			if cd.patternsOverlap(pattern1, pattern2) {
				conflict := ThatPatternConflict{
					Type:        "overlap",
					Pattern1:    pattern1,
					Pattern2:    pattern2,
					Severity:    cd.calculateOverlapSeverity(pattern1, pattern2),
					Description: fmt.Sprintf("Patterns '%s' and '%s' have overlapping matching scope", pattern1, pattern2),
					Suggestions: cd.generateOverlapSuggestions(pattern1, pattern2),
					Examples:    cd.generateOverlapExamples(pattern1, pattern2),
				}
				detector.Conflicts = append(detector.Conflicts, conflict)
			}
		}
	}
}

// detectAmbiguityConflicts detects patterns that create ambiguous matching
func (cd *ConflictDetection) detectAmbiguityConflicts(detector *ThatPatternConflictDetector) {
	for i, pattern1 := range detector.Patterns {
		for j, pattern2 := range detector.Patterns {
			if i >= j {
				continue
			}

			// Check for ambiguity
			if cd.patternsAreAmbiguous(pattern1, pattern2) {
				conflict := ThatPatternConflict{
					Type:        "ambiguity",
					Pattern1:    pattern1,
					Pattern2:    pattern2,
					Severity:    "high",
					Description: fmt.Sprintf("Patterns '%s' and '%s' create ambiguous matching", pattern1, pattern2),
					Suggestions: cd.generateAmbiguitySuggestions(pattern1, pattern2),
					Examples:    cd.generateAmbiguityExamples(pattern1, pattern2),
				}
				detector.Conflicts = append(detector.Conflicts, conflict)
			}
		}
	}
}

// detectPriorityConflicts detects patterns with unclear priority
func (cd *ConflictDetection) detectPriorityConflicts(detector *ThatPatternConflictDetector) {
	for i, pattern1 := range detector.Patterns {
		for j, pattern2 := range detector.Patterns {
			if i >= j {
				continue
			}

			if cd.patternsHavePriorityConflict(pattern1, pattern2) {
				conflict := ThatPatternConflict{
					Type:        "priority",
					Pattern1:    pattern1,
					Pattern2:    pattern2,
					Severity:    "medium",
					Description: fmt.Sprintf("Patterns '%s' and '%s' have unclear priority", pattern1, pattern2),
					Suggestions: cd.generatePrioritySuggestions(pattern1, pattern2),
					Examples:    cd.generatePriorityExamples(pattern1, pattern2),
				}
				detector.Conflicts = append(detector.Conflicts, conflict)
			}
		}
	}
}

// detectWildcardConflicts detects wildcard-related conflicts
func (cd *ConflictDetection) detectWildcardConflicts(detector *ThatPatternConflictDetector) {
	for i, pattern1 := range detector.Patterns {
		for j, pattern2 := range detector.Patterns {
			if i >= j {
				continue
			}

			if cd.patternsHaveWildcardConflict(pattern1, pattern2) {
				conflict := ThatPatternConflict{
					Type:        "wildcard",
					Pattern1:    pattern1,
					Pattern2:    pattern2,
					Severity:    "low",
					Description: fmt.Sprintf("Patterns '%s' and '%s' have wildcard conflicts", pattern1, pattern2),
					Suggestions: cd.generateWildcardSuggestions(pattern1, pattern2),
					Examples:    cd.generateWildcardExamples(pattern1, pattern2),
				}
				detector.Conflicts = append(detector.Conflicts, conflict)
			}
		}
	}
}

// detectSpecificityConflicts detects specificity-related conflicts
func (cd *ConflictDetection) detectSpecificityConflicts(detector *ThatPatternConflictDetector) {
	for i, pattern1 := range detector.Patterns {
		for j, pattern2 := range detector.Patterns {
			if i >= j {
				continue
			}

			if cd.patternsHaveSpecificityConflict(pattern1, pattern2) {
				conflict := ThatPatternConflict{
					Type:        "specificity",
					Pattern1:    pattern1,
					Pattern2:    pattern2,
					Severity:    "medium",
					Description: fmt.Sprintf("Patterns '%s' and '%s' have specificity conflicts", pattern1, pattern2),
					Suggestions: cd.generateSpecificitySuggestions(pattern1, pattern2),
					Examples:    cd.generateSpecificityExamples(pattern1, pattern2),
				}
				detector.Conflicts = append(detector.Conflicts, conflict)
			}
		}
	}
}

// Helper functions for conflict detection
func (cd *ConflictDetection) patternsOverlap(pattern1, pattern2 string) bool {
	// Convert patterns to testable format
	testCases := []string{
		"HELLO WORLD",
		"HELLO",
		"WORLD",
		"HELLO THERE",
		"GOOD MORNING",
		"GOOD AFTERNOON",
		"GOOD EVENING",
		"GOOD NIGHT",
		"WHAT IS YOUR NAME",
		"WHAT DO YOU DO",
		"TELL ME ABOUT YOURSELF",
		"WHO ARE YOU",
		"WHERE ARE YOU FROM",
		"WHAT CAN YOU DO",
		"HELP ME",
		"THANK YOU",
		"GOODBYE",
		"SEE YOU LATER",
		"HAVE A NICE DAY",
		"TAKE CARE",
	}

	matches1 := 0
	matches2 := 0
	overlap := 0

	for _, testCase := range testCases {
		matched1 := cd.testPatternMatch(pattern1, testCase)
		matched2 := cd.testPatternMatch(pattern2, testCase)

		if matched1 {
			matches1++
		}
		if matched2 {
			matches2++
		}
		if matched1 && matched2 {
			overlap++
		}
	}

	// Calculate overlap percentage
	if matches1 > 0 && matches2 > 0 {
		overlapPercentage := float64(overlap) / float64(matches1+matches2-overlap)
		return overlapPercentage > 0.3 // 30% overlap threshold
	}

	return false
}

func (cd *ConflictDetection) patternsAreAmbiguous(pattern1, pattern2 string) bool {
	// Check if patterns could match the same input
	testCases := []string{"HELLO", "HELLO WORLD", "GOOD MORNING"}
	for _, testCase := range testCases {
		if cd.testPatternMatch(pattern1, testCase) && cd.testPatternMatch(pattern2, testCase) {
			// Check if they have similar specificity
			specificity1 := CalculatePatternSpecificity(pattern1)
			specificity2 := CalculatePatternSpecificity(pattern2)

			// If specificity is similar, it's ambiguous
			if absFloat(specificity1-specificity2) < 0.2 {
				return true
			}
		}
	}
	return false
}

func (cd *ConflictDetection) patternsHavePriorityConflict(pattern1, pattern2 string) bool {
	// Check if patterns have similar priority but different specificity
	specificity1 := CalculatePatternSpecificity(pattern1)
	specificity2 := CalculatePatternSpecificity(pattern2)

	// If specificity is very different but both could match, it's a priority conflict
	if absFloat(specificity1-specificity2) > 0.5 {
		// Check if both could match the same input
		testCases := []string{"HELLO", "HELLO WORLD", "GOOD MORNING"}
		for _, testCase := range testCases {
			if cd.testPatternMatch(pattern1, testCase) && cd.testPatternMatch(pattern2, testCase) {
				return true
			}
		}
	}
	return false
}

func (cd *ConflictDetection) patternsHaveWildcardConflict(pattern1, pattern2 string) bool {
	// Check for conflicting wildcard usage
	wildcards1 := CountWildcards(pattern1)
	wildcards2 := CountWildcards(pattern2)

	// If one has many wildcards and the other has few, it might be a conflict
	if abs(wildcards1-wildcards2) > 3 {
		// Check if they could match similar inputs
		testCases := []string{"HELLO", "HELLO WORLD", "GOOD MORNING"}
		matches1 := 0
		matches2 := 0
		for _, testCase := range testCases {
			if cd.testPatternMatch(pattern1, testCase) {
				matches1++
			}
			if cd.testPatternMatch(pattern2, testCase) {
				matches2++
			}
		}
		return matches1 > 0 && matches2 > 0
	}
	return false
}

func (cd *ConflictDetection) patternsHaveSpecificityConflict(pattern1, pattern2 string) bool {
	// Check if patterns have very different specificity
	specificity1 := CalculatePatternSpecificity(pattern1)
	specificity2 := CalculatePatternSpecificity(pattern2)

	// If specificity is very different, it might be a conflict
	return absFloat(specificity1-specificity2) > 0.7
}

func (cd *ConflictDetection) testPatternMatch(pattern, input string) bool {
	// Simplified pattern matching for conflict detection
	// This is a basic implementation - in practice, you'd use the full pattern matching logic

	// Convert to uppercase for matching
	pattern = strings.ToUpper(pattern)
	input = strings.ToUpper(input)

	// Handle exact matches
	if pattern == input {
		return true
	}

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		// Simple wildcard matching
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(input, parts[0]) && strings.HasSuffix(input, parts[1])
		}
	}

	return false
}

// Suggestion generation functions
func (cd *ConflictDetection) generateOverlapSuggestions(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Make pattern '%s' more specific", pattern1),
		fmt.Sprintf("Make pattern '%s' more specific", pattern2),
		"Consider using different wildcard types",
		"Add more context to distinguish patterns",
	}
}

func (cd *ConflictDetection) generateAmbiguitySuggestions(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Reorder patterns to prioritize '%s'", pattern1),
		fmt.Sprintf("Reorder patterns to prioritize '%s'", pattern2),
		"Add more specific patterns",
		"Use different pattern structures",
	}
}

func (cd *ConflictDetection) generatePrioritySuggestions(_, _ string) []string {
	return []string{
		"Reorder patterns by specificity",
		"Add priority indicators",
		"Use more specific patterns first",
		"Consider pattern hierarchy",
	}
}

func (cd *ConflictDetection) generateWildcardSuggestions(_, _ string) []string {
	return []string{
		"Use consistent wildcard types",
		"Limit wildcard usage",
		"Use more specific patterns",
		"Consider pattern alternatives",
	}
}

func (cd *ConflictDetection) generateSpecificitySuggestions(_, _ string) []string {
	return []string{
		"Balance pattern specificity",
		"Use consistent pattern complexity",
		"Consider pattern hierarchy",
		"Add more context to patterns",
	}
}

// Example generation functions
func (cd *ConflictDetection) generateOverlapExamples(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Both patterns match 'HELLO WORLD': '%s' and '%s'", pattern1, pattern2),
		"Consider using different wildcard positions",
		"Add topic or that context to distinguish",
	}
}

func (cd *ConflictDetection) generateAmbiguityExamples(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Input 'HELLO' could match either '%s' or '%s'", pattern1, pattern2),
		"Both patterns have similar specificity",
		"Consider reordering or making more specific",
	}
}

func (cd *ConflictDetection) generatePriorityExamples(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Pattern '%s' should come before '%s'", pattern1, pattern2),
		"More specific patterns should have higher priority",
		"Consider pattern ordering",
	}
}

func (cd *ConflictDetection) generateWildcardExamples(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Pattern '%s' uses too many wildcards", pattern1),
		fmt.Sprintf("Pattern '%s' uses too many wildcards", pattern2),
		"Consider using more specific patterns",
	}
}

func (cd *ConflictDetection) generateSpecificityExamples(pattern1, pattern2 string) []string {
	return []string{
		fmt.Sprintf("Pattern '%s' is too general", pattern1),
		fmt.Sprintf("Pattern '%s' is too specific", pattern2),
		"Balance pattern complexity",
	}
}

// Helper functions
func (cd *ConflictDetection) calculateOverlapSeverity(pattern1, pattern2 string) string {
	overlap := cd.calculateOverlapPercentage(pattern1, pattern2)

	if overlap > 0.8 {
		return "critical"
	} else if overlap > 0.6 {
		return "high"
	} else if overlap > 0.4 {
		return "medium"
	} else {
		return "low"
	}
}

func (cd *ConflictDetection) calculateOverlapPercentage(pattern1, pattern2 string) float64 {
	// Simplified overlap calculation
	testCases := []string{"HELLO", "HELLO WORLD", "GOOD MORNING", "WHAT IS YOUR NAME"}
	matches1 := 0
	matches2 := 0
	overlap := 0

	for _, testCase := range testCases {
		matched1 := cd.testPatternMatch(pattern1, testCase)
		matched2 := cd.testPatternMatch(pattern2, testCase)

		if matched1 {
			matches1++
		}
		if matched2 {
			matches2++
		}
		if matched1 && matched2 {
			overlap++
		}
	}

	if matches1+matches2-overlap == 0 {
		return 0
	}

	return float64(overlap) / float64(matches1+matches2-overlap)
}
