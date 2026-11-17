package golem

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// GetVersion reads the version string from the VERSION file
// Returns "1.0.0" as default if the file cannot be read
func GetVersion() string {
	// Try to find the VERSION file starting from the current directory and walking up
	versionFile := findVersionFile()
	if versionFile == "" {
		return "1.0.0" // Default fallback
	}

	content, err := os.ReadFile(versionFile)
	if err != nil {
		return "1.0.0" // Default fallback if read fails
	}

	version := strings.TrimSpace(string(content))
	if version == "" {
		return "1.0.0" // Default if file is empty
	}

	return version
}

// findVersionFile searches for the VERSION file starting from the current directory
// and walking up to the project root
func findVersionFile() string {
	// Start from current directory and walk up
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Try up to 5 levels up from current directory
	for i := 0; i < 5; i++ {
		versionPath := filepath.Join(currentDir, "VERSION")
		if _, err := os.Stat(versionPath); err == nil {
			return versionPath
		}

		// Go up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached root, stop
			break
		}
		currentDir = parentDir
	}

	return ""
}

// Note: SentenceSplitter and WordBoundaryDetector are defined in aiml_native.go

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// absFloat returns the absolute value of a float64
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// CountWildcards counts all types of wildcards in a pattern
func CountWildcards(pattern string) int {
	count := 0
	count += strings.Count(pattern, "*")
	count += strings.Count(pattern, "_")
	count += strings.Count(pattern, "^")
	count += strings.Count(pattern, "#")
	count += strings.Count(pattern, "$")
	return count
}

// CountWildcardsByType counts wildcards by type and returns a map
func CountWildcardsByType(pattern string) map[string]int {
	return map[string]int{
		"star":       strings.Count(pattern, "*"),
		"underscore": strings.Count(pattern, "_"),
		"caret":      strings.Count(pattern, "^"),
		"hash":       strings.Count(pattern, "#"),
		"dollar":     strings.Count(pattern, "$"),
	}
}

// CalculateLength calculates the length of content by different types
func CalculateLength(content, lengthType string) string {
	switch strings.ToLower(lengthType) {
	case "words":
		// Count words
		words := strings.Fields(content)
		return strconv.Itoa(len(words))
	case "sentences":
		// Count sentences (split by sentence-ending punctuation)
		sentences := splitSentences(content)
		return strconv.Itoa(len(sentences))
	case "characters", "chars":
		// Count characters (including spaces)
		return strconv.Itoa(len(content))
	case "letters":
		// Count only letters
		letterCount := 0
		for _, r := range content {
			if unicode.IsLetter(r) {
				letterCount++
			}
		}
		return strconv.Itoa(letterCount)
	case "words_no_punctuation":
		// Count words excluding punctuation
		words := strings.Fields(content)
		wordCount := 0
		for _, word := range words {
			hasLetter := false
			for _, r := range word {
				if unicode.IsLetter(r) {
					hasLetter = true
					break
				}
			}
			if hasLetter {
				wordCount++
			}
		}
		return strconv.Itoa(wordCount)
	default:
		// Default to character count
		return strconv.Itoa(len(content))
	}
}

// splitSentences splits text into sentences
func splitSentences(text string) []string {
	// Simple sentence splitting on periods, exclamation marks, and question marks
	re := regexp.MustCompile(`[.!?]+`)
	sentences := re.Split(text, -1)
	var result []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence != "" {
			result = append(result, sentence)
		}
	}
	return result
}

// CalculateMemoryUsage calculates memory usage for a slice of strings
func CalculateMemoryUsage(items []string) int {
	totalBytes := 0
	for _, item := range items {
		totalBytes += len(item) + 24 // 24 bytes overhead per string
	}
	return totalBytes
}

// CalculatePatternSpecificity calculates pattern specificity (0.0 = very general, 1.0 = very specific)
func CalculatePatternSpecificity(pattern string) float64 {
	// Count wildcards
	wildcardCount := CountWildcards(pattern)

	// Count words
	words := strings.Fields(pattern)
	wordCount := len(words)

	// Calculate specificity using the formula: (wordCount - wildcardCount) / wordCount
	if wordCount == 0 {
		return 0.0
	}

	specificity := float64(wordCount-wildcardCount) / float64(wordCount)

	// Ensure result is between 0.0 and 1.0
	if specificity > 1.0 {
		specificity = 1.0
	}
	if specificity < 0.0 {
		specificity = 0.0
	}

	return specificity
}

// CalculateOverlapPercentage calculates the percentage of overlap between two strings
func CalculateOverlapPercentage(str1, str2 string) float64 {
	if str1 == str2 {
		return 100.0
	}

	// Convert to lowercase for case-insensitive comparison
	s1 := strings.ToLower(str1)
	s2 := strings.ToLower(str2)

	// Calculate character-level overlap
	shorter := s1
	longer := s2
	if len(s2) < len(s1) {
		shorter = s2
		longer = s1
	}

	if len(longer) == 0 {
		return 0.0
	}

	// Count matching characters
	matches := 0
	for i, char := range shorter {
		if i < len(longer) && rune(longer[i]) == char {
			matches++
		}
	}

	// Calculate percentage
	percentage := float64(matches) / float64(len(longer)) * 100.0
	return percentage
}
