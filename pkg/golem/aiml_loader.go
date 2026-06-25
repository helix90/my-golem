package golem

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AIMLLoader provides AIML file loading and parsing functionality
type AIMLLoader struct {
	golem *Golem
}

// NewAIMLLoader creates a new AIML loader instance
func NewAIMLLoader(golem *Golem) *AIMLLoader {
	return &AIMLLoader{golem: golem}
}

// LoadAIMLFromString loads AIML content from a string
func (al *AIMLLoader) LoadAIMLFromString(content string) error {
	aiml, err := al.parseAIML(content)
	if err != nil {
		return fmt.Errorf("failed to parse AIML: %v", err)
	}

	if err := al.validateAIML(aiml); err != nil {
		return fmt.Errorf("AIML validation failed: %v", err)
	}

	kb := al.aimlToKnowledgeBase(aiml)
	mergedKB, err := al.mergeKnowledgeBases(al.golem.aimlKB, kb)
	if err != nil {
		return err
	}
	al.golem.aimlKB = mergedKB
	return nil
}

// LoadAIML loads AIML from a file
func (al *AIMLLoader) LoadAIML(filename string) (*AIMLKnowledgeBase, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	aiml, err := al.parseAIML(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse AIML file %s: %v", filename, err)
	}

	if err := al.validateAIML(aiml); err != nil {
		return nil, fmt.Errorf("AIML validation failed for file %s: %v", filename, err)
	}

	kb := al.aimlToKnowledgeBase(aiml)
	mergedKB, err := al.mergeKnowledgeBases(al.golem.aimlKB, kb)
	if err != nil {
		return nil, err
	}
	al.golem.aimlKB = mergedKB
	return mergedKB, nil
}

// LoadAIMLFromDirectory loads all AIML files from a directory
func (al *AIMLLoader) LoadAIMLFromDirectory(dirPath string) (*AIMLKnowledgeBase, error) {
	var allKBs []*AIMLKnowledgeBase

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".aiml") {
			kb, err := al.LoadAIML(path)
			if err != nil {
				return fmt.Errorf("failed to load %s: %v", path, err)
			}
			allKBs = append(allKBs, kb)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Merge all knowledge bases
	var result *AIMLKnowledgeBase
	for _, kb := range allKBs {
		merged, err := al.mergeKnowledgeBases(result, kb)
		if err != nil {
			return nil, err
		}
		result = merged
	}

	return result, nil
}

// parseAIML parses AIML content
func (al *AIMLLoader) parseAIML(content string) (*AIML, error) {
	// Remove comments and XML declaration
	content = al.removeComments(content)
	content = al.removeXMLDeclaration(content)

	var aiml AIML
	err := xml.Unmarshal([]byte(content), &aiml)
	if err != nil {
		return nil, fmt.Errorf("XML parsing error: %v", err)
	}

	return &aiml, nil
}

// removeComments removes XML comments from content
func (al *AIMLLoader) removeComments(content string) string {
	commentPattern := regexp.MustCompile(`<!--.*?-->`)
	return commentPattern.ReplaceAllString(content, "")
}

// removeXMLDeclaration removes XML declaration
func (al *AIMLLoader) removeXMLDeclaration(content string) string {
	declPattern := regexp.MustCompile(`<\?xml[^>]*\?>`)
	return declPattern.ReplaceAllString(content, "")
}

// validateAIML validates AIML structure
func (al *AIMLLoader) validateAIML(aiml *AIML) error {
	if aiml == nil {
		return fmt.Errorf("AIML is nil")
	}

	if len(aiml.Categories) == 0 {
		return fmt.Errorf("no categories found")
	}

	for i, category := range aiml.Categories {
		if category.Pattern == "" {
			return fmt.Errorf("category %d has empty pattern", i)
		}
		if category.Template == "" {
			return fmt.Errorf("category %d has empty template", i)
		}
	}

	return nil
}

// aimlToKnowledgeBase converts AIML to AIMLKnowledgeBase
func (al *AIMLLoader) aimlToKnowledgeBase(aiml *AIML) *AIMLKnowledgeBase {
	kb := NewAIMLKnowledgeBase()

	kb.Categories = append(kb.Categories, aiml.Categories...)

	return kb
}

// mergeKnowledgeBases merges two knowledge bases
func (al *AIMLLoader) mergeKnowledgeBases(kb1, kb2 *AIMLKnowledgeBase) (*AIMLKnowledgeBase, error) {
	if kb1 == nil {
		return kb2, nil
	}
	if kb2 == nil {
		return kb1, nil
	}

	result := &AIMLKnowledgeBase{
		Categories:     make([]Category, 0, len(kb1.Categories)+len(kb2.Categories)),
		Patterns:       make(map[string]*Category),
		Sets:           make(map[string][]string),
		Topics:         make(map[string][]string),
		Variables:      make(map[string]string),
		Properties:     make(map[string]string),
		Maps:           make(map[string]map[string]string),
		Lists:          make(map[string][]string),
		Arrays:         make(map[string][]string),
		SetCollections: make(map[string]*SetCollection),
		Substitutions:  make(map[string]map[string]string),
	}

	// Add categories from both knowledge bases
	result.Categories = append(result.Categories, kb1.Categories...)
	result.Categories = append(result.Categories, kb2.Categories...)

	// Merge patterns
	for k, v := range kb1.Patterns {
		result.Patterns[k] = v
	}
	for k, v := range kb2.Patterns {
		result.Patterns[k] = v
	}

	// Merge sets
	for k, v := range kb1.Sets {
		result.Sets[k] = v
	}
	for k, v := range kb2.Sets {
		result.Sets[k] = v
	}

	// Merge topics
	for k, v := range kb1.Topics {
		result.Topics[k] = v
	}
	for k, v := range kb2.Topics {
		result.Topics[k] = v
	}

	// Merge variables
	for k, v := range kb1.Variables {
		result.Variables[k] = v
	}
	for k, v := range kb2.Variables {
		result.Variables[k] = v
	}

	// Merge properties
	for k, v := range kb1.Properties {
		result.Properties[k] = v
	}
	for k, v := range kb2.Properties {
		result.Properties[k] = v
	}

	// Merge maps
	for k, v := range kb1.Maps {
		result.Maps[k] = v
	}
	for k, v := range kb2.Maps {
		result.Maps[k] = v
	}

	return result, nil
}
