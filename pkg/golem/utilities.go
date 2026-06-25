package golem

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// Utilities provides general utility functions
type Utilities struct {
	golem *Golem
}

// NewUtilities creates a new utilities instance
func NewUtilities(golem *Golem) *Utilities {
	return &Utilities{golem: golem}
}

// RandomInt generates a random integer between 0 and max-1
func (u *Utilities) RandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	return rand.Intn(max)
}

// FormatDate formats a date according to the given format
func (u *Utilities) FormatDate(format string) string {
	now := time.Now()

	if format == "" {
		format = "January 2, 2006"
	}

	// Convert common AIML date formats to Go time formats
	goFormat := u.convertToGoTimeFormat(format)

	return now.Format(goFormat)
}

// FormatTime formats a time according to the given format
func (u *Utilities) FormatTime(format string) string {
	now := time.Now()

	if format == "" {
		format = "3:04 PM"
	}

	// Convert common AIML time formats to Go time formats
	goFormat := u.convertToGoTimeFormat(format)

	return now.Format(goFormat)
}

// convertToGoTimeFormat converts AIML time format to Go time format
func (u *Utilities) convertToGoTimeFormat(format string) string {
	// Common AIML to Go time format conversions
	conversions := map[string]string{
		"HH:mm:ss":      "15:04:05",
		"HH:mm":         "15:04",
		"h:mm:ss a":     "3:04:05 PM",
		"h:mm a":        "3:04 PM",
		"h:mm":          "3:04",
		"HH:mm:ss a":    "15:04:05 PM",
		"HH:mm a":       "15:04 PM",
		"yyyy-MM-dd":    "2006-01-02",
		"MM/dd/yyyy":    "01/02/2006",
		"dd/MM/yyyy":    "02/01/2006",
		"MMMM dd, yyyy": "January 02, 2006",
		"MMM dd, yyyy":  "Jan 02, 2006",
		"dd MMM yyyy":   "02 Jan 2006",
		"yyyy":          "2006",
		"MM":            "01",
		"dd":            "02",
		"HH":            "15",
		"mm":            "04",
		"ss":            "05",
		"a":             "PM",
		"aa":            "PM",
		"aaa":           "PM",
	}

	// Check if it's already a Go format
	if u.looksLikeGoTimeFormat(format) {
		return format
	}

	// Try direct conversion
	if goFormat, exists := conversions[format]; exists {
		return goFormat
	}

	// Try to convert common patterns
	result := format
	result = strings.ReplaceAll(result, "yyyy", "2006")
	result = strings.ReplaceAll(result, "MM", "01")
	result = strings.ReplaceAll(result, "dd", "02")
	result = strings.ReplaceAll(result, "HH", "15")
	result = strings.ReplaceAll(result, "mm", "04")
	result = strings.ReplaceAll(result, "ss", "05")
	result = strings.ReplaceAll(result, "a", "PM")

	return result
}

// looksLikeGoTimeFormat checks if a format looks like a Go time format
func (u *Utilities) looksLikeGoTimeFormat(format string) bool {
	goTimeChars := "20060102150405"
	for _, char := range format {
		if !strings.ContainsRune(goTimeChars, char) &&
			!strings.ContainsRune(" -/:.,", char) &&
			!strings.ContainsRune("PMAM", char) {
			return false
		}
	}
	return true
}

// ProcessRandomTags processes random tags in templates
func (u *Utilities) ProcessRandomTags(template string) string {
	// Find all <random> tags
	randomRegex := regexp.MustCompile(`(?s)<random>(.*?)</random>`)
	matches := randomRegex.FindAllStringSubmatch(template, -1)

	result := template
	for _, match := range matches {
		if len(match) > 1 {
			content := match[1]
			selected := u.selectRandomOption(content)
			result = strings.Replace(result, match[0], selected, 1)
		}
	}

	return result
}

// selectRandomOption selects a random option from random tag content
func (u *Utilities) selectRandomOption(content string) string {
	// Split by <li> tags
	liRegex := regexp.MustCompile(`(?s)<li[^>]*>(.*?)</li>`)
	matches := liRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return content
	}

	// Select random option
	randomIndex := u.RandomInt(len(matches))
	return strings.TrimSpace(matches[randomIndex][1])
}

// LoadDefaultProperties loads default properties
func (u *Utilities) LoadDefaultProperties(kb *AIMLKnowledgeBase) error {
	if kb == nil {
		return fmt.Errorf("knowledge base is nil")
	}

	// Set default properties
	defaultProps := map[string]string{
		"name":        "Golem",
		"version":     GetVersion(),
		"language":    "en",
		"encoding":    "UTF-8",
		"author":      "Golem AI",
		"description": "AIML Chatbot",
		"created":     time.Now().Format("2006-01-02"),
	}

	for key, value := range defaultProps {
		kb.SetProperty(key, value)
	}

	return nil
}

// ParsePropertiesFile parses a properties file
func (u *Utilities) ParsePropertiesFile(content string) (map[string]string, error) {
	properties := make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			properties[key] = value
		}
	}

	return properties, nil
}

// ExpandContractions expands common English contractions
func (u *Utilities) ExpandContractions(text string) string {
	contractions := map[string]string{
		"i'm":       "i am",
		"you're":    "you are",
		"he's":      "he is",
		"she's":     "she is",
		"it's":      "it is",
		"we're":     "we are",
		"they're":   "they are",
		"i've":      "i have",
		"you've":    "you have",
		"we've":     "we have",
		"they've":   "they have",
		"i'll":      "i will",
		"you'll":    "you will",
		"he'll":     "he will",
		"she'll":    "she will",
		"we'll":     "we will",
		"they'll":   "they will",
		"i'd":       "i would",
		"you'd":     "you would",
		"he'd":      "he would",
		"she'd":     "she would",
		"we'd":      "we would",
		"they'd":    "they would",
		"i can't":   "i cannot",
		"you can't": "you cannot",
		"won't":     "will not",
		"don't":     "do not",
		"doesn't":   "does not",
		"didn't":    "did not",
		"haven't":   "have not",
		"hasn't":    "has not",
		"hadn't":    "had not",
		"isn't":     "is not",
		"aren't":    "are not",
		"wasn't":    "was not",
		"weren't":   "were not",
		"wouldn't":  "would not",
		"shouldn't": "should not",
		"couldn't":  "could not",
		"mustn't":   "must not",
	}

	result := text
	for contraction, expansion := range contractions {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(contraction) + `\b`)
		result = re.ReplaceAllString(result, expansion)
	}

	return result
}

// NormalizeForMatching normalizes text for pattern matching
func (u *Utilities) NormalizeForMatching(input string) string {
	// Convert to lowercase
	result := strings.ToLower(input)

	// Remove extra whitespace
	result = strings.TrimSpace(result)
	re := regexp.MustCompile(`\s+`)
	result = re.ReplaceAllString(result, " ")

	// Remove punctuation
	re = regexp.MustCompile(`[^\w\s]`)
	result = re.ReplaceAllString(result, "")

	return result
}

// NormalizePattern normalizes an AIML pattern
func (u *Utilities) NormalizePattern(pattern string) string {
	// Convert to lowercase
	result := strings.ToLower(pattern)

	// Remove extra whitespace
	result = strings.TrimSpace(result)
	re := regexp.MustCompile(`\s+`)
	result = re.ReplaceAllString(result, " ")

	// Normalize wildcards
	result = strings.ReplaceAll(result, "*", "*")
	result = strings.ReplaceAll(result, "_", "_")

	return result
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
