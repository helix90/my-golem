package golem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnhancedDirectoryLoading(t *testing.T) {
	// Test loading the test data directory
	g := NewForTesting(t, true)

	// Get the absolute path to test data directory
	testDataPath, err := filepath.Abs("../../testdata/loader_test")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check if the directory exists
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Fatalf("Test data directory not found: %s", testDataPath)
	}

	// Load the entire test data directory
	kb, err := g.LoadAIMLFromDirectory(testDataPath)
	if err != nil {
		t.Fatalf("Failed to load test data directory: %v", err)
	}

	// Verify that we have loaded content
	if len(kb.Categories) == 0 {
		t.Error("Expected to load some AIML categories")
	}

	// Check that we have maps loaded
	if len(kb.Maps) == 0 {
		t.Error("Expected to load some maps")
	}

	// Check that we have sets loaded
	if len(kb.Sets) == 0 {
		t.Error("Expected to load some sets")
	}

	// Check that we have substitutions loaded
	if len(kb.Substitutions) == 0 {
		t.Error("Expected to load some substitutions")
	}

	// Check that we have properties loaded
	if len(kb.Properties) == 0 {
		t.Error("Expected to load some properties")
	}

	t.Logf("Loaded %d categories", len(kb.Categories))
	t.Logf("Loaded %d maps", len(kb.Maps))
	t.Logf("Loaded %d sets", len(kb.Sets))
	t.Logf("Loaded %d substitution files", len(kb.Substitutions))
	t.Logf("Loaded %d properties", len(kb.Properties))

	// Verify specific file types are loaded
	expectedSubstitutions := []string{"normal"}
	for _, subName := range expectedSubstitutions {
		if _, exists := kb.Substitutions[subName]; !exists {
			t.Errorf("Expected substitution file '%s' to be loaded", subName)
		}
	}

	// Check for specific properties files
	expectedProperties := []string{"test"}
	for _, propName := range expectedProperties {
		// Properties are merged into the main Properties map, so we check for specific keys
		found := false
		for key := range kb.Properties {
			if key == "name" || key == "version" || key == "gender" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected properties from '%s' to be loaded", propName)
		}
	}

	// Check for pdefaults (stored with pdefault prefix)
	foundPDefaults := false
	for key := range kb.Properties {
		if len(key) > 8 && key[:8] == "pdefault" {
			foundPDefaults = true
			break
		}
	}
	if !foundPDefaults {
		t.Error("Expected pdefaults to be loaded with pdefault prefix")
	}
}

func TestLoadSubstitutionFile(t *testing.T) {
	g := NewForTesting(t, true)

	// Get the absolute path to the substitution file
	substitutionPath, err := filepath.Abs("../../testdata/loader_test/normal.substitution")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check if the file exists
	if _, err := os.Stat(substitutionPath); os.IsNotExist(err) {
		t.Fatalf("Normal.substitution file not found: %s", substitutionPath)
	}

	// Test loading a specific substitution file
	substitutions, err := g.LoadSubstitutionFromFile(substitutionPath)
	if err != nil {
		t.Fatalf("Failed to load normal.substitution: %v", err)
	}

	if len(substitutions) == 0 {
		t.Error("Expected to load some substitution rules")
	}

	// Check for specific substitution rules
	expectedRules := []string{"%20", "&", "(", ")"}
	for _, rule := range expectedRules {
		if _, exists := substitutions[rule]; !exists {
			t.Errorf("Expected substitution rule '%s' to be loaded", rule)
		}
	}

	t.Logf("Loaded %d substitution rules", len(substitutions))
}

func TestLoadPropertiesFile(t *testing.T) {
	g := NewForTesting(t, true)

	// Get the absolute path to the properties file
	propertiesPath, err := filepath.Abs("../../testdata/loader_test/test.properties")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check if the file exists
	if _, err := os.Stat(propertiesPath); os.IsNotExist(err) {
		t.Fatalf("Test.properties file not found: %s", propertiesPath)
	}

	// Test loading a specific properties file
	properties, err := g.LoadPropertiesFromFile(propertiesPath)
	if err != nil {
		t.Fatalf("Failed to load test.properties: %v", err)
	}

	if len(properties) == 0 {
		t.Error("Expected to load some properties")
	}

	// Check for specific properties
	expectedProps := []string{"name", "version", "gender", "birthplace"}
	for _, prop := range expectedProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Expected property '%s' to be loaded", prop)
		}
	}

	t.Logf("Loaded %d properties", len(properties))
}

func TestLoadPDefaultsFile(t *testing.T) {
	g := NewForTesting(t, true)

	// Get the absolute path to the pdefaults file
	pdefaultsPath, err := filepath.Abs("../../testdata/loader_test/test.pdefaults")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check if the file exists
	if _, err := os.Stat(pdefaultsPath); os.IsNotExist(err) {
		t.Fatalf("Test.pdefaults file not found: %s", pdefaultsPath)
	}

	// Test loading a specific pdefaults file
	pdefaults, err := g.LoadPDefaultsFromFile(pdefaultsPath)
	if err != nil {
		t.Fatalf("Failed to load test.pdefaults: %v", err)
	}

	if len(pdefaults) == 0 {
		t.Error("Expected to load some pdefaults")
	}

	// Check for specific pdefaults
	expectedDefaults := []string{"address", "age", "baby", "bestfriend"}
	for _, def := range expectedDefaults {
		if _, exists := pdefaults[def]; !exists {
			t.Errorf("Expected pdefault '%s' to be loaded", def)
		}
	}

	t.Logf("Loaded %d pdefaults", len(pdefaults))
}

func TestLoadAllRelatedFiles(t *testing.T) {
	g := NewForTesting(t, true)

	// Get the absolute path to the AIML file
	aimlPath, err := filepath.Abs("../../testdata/loader_test/test.aiml")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Check if the file exists
	if _, err := os.Stat(aimlPath); os.IsNotExist(err) {
		t.Fatalf("Test.aiml file not found: %s", aimlPath)
	}

	// Test loading all related files from a single file
	err = g.loadAllRelatedFiles(aimlPath)
	if err != nil {
		t.Fatalf("Failed to load all related files: %v", err)
	}

	// Verify that the knowledge base was loaded
	if g.aimlKB == nil {
		t.Fatal("Expected knowledge base to be loaded")
	}

	// Check that we have loaded content from all file types
	if len(g.aimlKB.Categories) == 0 {
		t.Error("Expected to load some AIML categories")
	}

	if len(g.aimlKB.Maps) == 0 {
		t.Error("Expected to load some maps")
	}

	if len(g.aimlKB.Sets) == 0 {
		t.Error("Expected to load some sets")
	}

	if len(g.aimlKB.Substitutions) == 0 {
		t.Error("Expected to load some substitutions")
	}

	if len(g.aimlKB.Properties) == 0 {
		t.Error("Expected to load some properties")
	}

	t.Logf("Successfully loaded all related files")
	t.Logf("Categories: %d", len(g.aimlKB.Categories))
	t.Logf("Maps: %d", len(g.aimlKB.Maps))
	t.Logf("Sets: %d", len(g.aimlKB.Sets))
	t.Logf("Substitutions: %d", len(g.aimlKB.Substitutions))
	t.Logf("Properties: %d", len(g.aimlKB.Properties))
}

// TestDirectoryPropertiesMergeInLoadOrder checks that a property set in several
// files takes its value from the file loaded last, every time
func TestDirectoryPropertiesMergeInLoadOrder(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"bot.aiml":               `<?xml version="1.0" encoding="UTF-8"?><aiml version="2.0"><category><pattern>WHAT IS YOUR NAME</pattern><template><bot name="name"/></template></category></aiml>`,
		"a.properties":           `[["name", "First"], ["job", "first job"]]`,
		"b.properties":           `[["name", "Second"]]`,
		"sub/c-early.properties": `[["name", "Third"], ["species", "robot"]]`,
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want := map[string]string{"name": "Third", "job": "first job", "species": "robot"}
	for i := 0; i < 20; i++ {
		g := NewForTesting(t, false)
		kb, err := g.LoadAIMLFromDirectory(dir)
		if err != nil {
			t.Fatalf("LoadAIMLFromDirectory: %v", err)
		}
		for key, value := range want {
			if kb.Properties[key] != value {
				t.Fatalf("LoadAIMLFromDirectory: %s = %q, want %q", key, kb.Properties[key], value)
			}
		}

		g = NewForTesting(t, false)
		if err := g.Execute("load", []string{dir}); err != nil {
			t.Fatalf("load: %v", err)
		}
		for key, value := range want {
			if got := g.GetKnowledgeBase().Properties[key]; got != value {
				t.Fatalf("load: %s = %q, want %q", key, got, value)
			}
		}
	}
}
