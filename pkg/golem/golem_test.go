package golem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	g := NewForTesting(t, false)
	if g == nil {
		t.Fatal("New() returned nil")
	}
	if g.verbose {
		t.Error("Expected verbose to be false")
	}

	g = New(true)
	if !g.verbose {
		t.Errorf("Expected verbose to be true, got %v", g.verbose)
	}
}

func TestProcessData(t *testing.T) {
	g := NewForTesting(t, false)
	input := "test input"
	result, err := g.ProcessData(input)
	if err != nil {
		t.Fatalf("ProcessData returned error: %v", err)
	}
	expected := "Processed: test input"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestAnalyzeData(t *testing.T) {
	g := NewForTesting(t, false)
	input := "test data"
	result, err := g.AnalyzeData(input)
	if err != nil {
		t.Fatalf("AnalyzeData returned error: %v", err)
	}
	if result["input"] != input {
		t.Errorf("Expected input %q, got %q", input, result["input"])
	}
	if result["status"] != "analyzed" {
		t.Errorf("Expected status 'analyzed', got %q", result["status"])
	}
}

func TestGenerateOutput(t *testing.T) {
	g := NewForTesting(t, false)
	data := "test data"
	result, err := g.GenerateOutput(data)
	if err != nil {
		t.Fatalf("GenerateOutput returned error: %v", err)
	}
	expected := "Generated output for: test data"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExecute(t *testing.T) {
	g := NewForTesting(t, false)

	// Test unknown command
	err := g.Execute("unknown", []string{})
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	// Test process command
	err = g.Execute("process", []string{"test.txt"})
	if err != nil {
		t.Errorf("Process command failed: %v", err)
	}

	// Test analyze command
	err = g.Execute("analyze", []string{"test.json"})
	if err != nil {
		t.Errorf("Analyze command failed: %v", err)
	}

	// Test generate command
	err = g.Execute("generate", []string{})
	if err != nil {
		t.Errorf("Generate command failed: %v", err)
	}

	// Test load command with non-existent file
	err = g.Execute("load", []string{"nonexistent.txt"})
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoadFile(t *testing.T) {
	g := NewForTesting(t, false)

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!\nThis is a test file."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test loading existing file
	content, err := g.LoadFile(testFile)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if content != testContent {
		t.Errorf("Expected content %q, got %q", testContent, content)
	}

	// Test loading non-existent file
	_, err = g.LoadFile("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test loading file in subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subDir, "subfile.txt")
	subContent := "Content in subdirectory"
	err = os.WriteFile(subFile, []byte(subContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create subdirectory file: %v", err)
	}

	content, err = g.LoadFile(subFile)
	if err != nil {
		t.Fatalf("LoadFile failed for subdirectory file: %v", err)
	}
	if content != subContent {
		t.Errorf("Expected content %q, got %q", subContent, content)
	}
}

func TestLoadCommand(t *testing.T) {
	g := NewForTesting(t, false)

	// Test load command without arguments
	err := g.Execute("load", []string{})
	if err == nil {
		t.Error("Expected error for load command without arguments")
	}

	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Test content for load command"

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test load command with existing file
	err = g.Execute("load", []string{testFile})
	if err != nil {
		t.Errorf("Load command failed: %v", err)
	}
}
