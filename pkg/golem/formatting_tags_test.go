package golem

import (
	"strings"
	"testing"
)

// TestFormalTagProcessing tests the <formal> tag processing functionality
func TestFormalTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic formal formatting",
			template: "<formal>hello world</formal>",
			expected: "Hello World",
		},
		{
			name:     "Single word",
			template: "<formal>test</formal>",
			expected: "Test",
		},
		{
			name:     "Multiple words",
			template: "<formal>this is a test sentence</formal>",
			expected: "This Is A Test Sentence",
		},
		{
			name:     "Already capitalized words",
			template: "<formal>Hello World</formal>",
			expected: "Hello World",
		},
		{
			name:     "Mixed case words",
			template: "<formal>hELLo WoRLd</formal>",
			expected: "Hello World",
		},
		{
			name:     "Numbers and special characters",
			template: "<formal>test123 with-special_chars</formal>",
			expected: "Test123 With-special_chars",
		},
		{
			name:     "Empty content",
			template: "<formal></formal>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<formal>   </formal>",
			expected: "",
		},
		{
			name:     "Multiple formal tags",
			template: "<formal>first</formal> and <formal>second</formal>",
			expected: "First and Second",
		},
		{
			name:     "Nested with other tags",
			template: "<formal><uppercase>hello world</uppercase></formal>",
			expected: "Hello World", // Tree processor evaluates inner tags first: uppercase("hello world")="HELLO WORLD", then formal("HELLO WORLD")="Hello World"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestExplodeTagProcessing tests the <explode> tag processing functionality
func TestExplodeTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic explode",
			template: "<explode>hello</explode>",
			expected: "h e l l o",
		},
		{
			name:     "Single character",
			template: "<explode>a</explode>",
			expected: "a",
		},
		{
			name:     "Multiple words",
			template: "<explode>hi there</explode>",
			expected: "h i   t h e r e",
		},
		{
			name:     "Numbers and special characters",
			template: "<explode>test123</explode>",
			expected: "t e s t 1 2 3",
		},
		{
			name:     "Empty content",
			template: "<explode></explode>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<explode>   </explode>",
			expected: "",
		},
		{
			name:     "Multiple explode tags",
			template: "<explode>hi</explode> <explode>there</explode>",
			expected: "h i t h e r e",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestCapitalizeTagProcessing tests the <capitalize> tag processing functionality
func TestCapitalizeTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic capitalize",
			template: "<capitalize>hello world</capitalize>",
			expected: "Hello world",
		},
		{
			name:     "Single word",
			template: "<capitalize>test</capitalize>",
			expected: "Test",
		},
		{
			name:     "Already capitalized",
			template: "<capitalize>Hello World</capitalize>",
			expected: "Hello world",
		},
		{
			name:     "All uppercase",
			template: "<capitalize>HELLO WORLD</capitalize>",
			expected: "Hello world",
		},
		{
			name:     "Mixed case",
			template: "<capitalize>hELLo WoRLd</capitalize>",
			expected: "Hello world",
		},
		{
			name:     "Numbers and special characters",
			template: "<capitalize>test123 with-special_chars</capitalize>",
			expected: "Test123 with-special_chars",
		},
		{
			name:     "Empty content",
			template: "<capitalize></capitalize>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<capitalize>   </capitalize>",
			expected: "",
		},
		{
			name:     "Multiple capitalize tags",
			template: "<capitalize>first</capitalize> and <capitalize>second</capitalize>",
			expected: "First and Second",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestReverseTagProcessing tests the <reverse> tag processing functionality
func TestReverseTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic reverse",
			template: "<reverse>hello</reverse>",
			expected: "olleh",
		},
		{
			name:     "Single character",
			template: "<reverse>a</reverse>",
			expected: "a",
		},
		{
			name:     "Multiple words",
			template: "<reverse>hello world</reverse>",
			expected: "dlrow olleh",
		},
		{
			name:     "Numbers and special characters",
			template: "<reverse>test123</reverse>",
			expected: "321tset",
		},
		{
			name:     "Empty content",
			template: "<reverse></reverse>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<reverse>   </reverse>",
			expected: "",
		},
		{
			name:     "Multiple reverse tags",
			template: "<reverse>hi</reverse> <reverse>there</reverse>",
			expected: "ih ereht",
		},
		{
			name:     "Unicode characters",
			template: "<reverse>café</reverse>",
			expected: "éfac",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestAcronymTagProcessing tests the <acronym> tag processing functionality
func TestAcronymTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic acronym",
			template: "<acronym>hello world</acronym>",
			expected: "HW",
		},
		{
			name:     "Single word",
			template: "<acronym>test</acronym>",
			expected: "T",
		},
		{
			name:     "Multiple words",
			template: "<acronym>artificial intelligence markup language</acronym>",
			expected: "AIML",
		},
		{
			name:     "Words with numbers",
			template: "<acronym>test123 world456</acronym>",
			expected: "TW",
		},
		{
			name:     "Empty content",
			template: "<acronym></acronym>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<acronym>   </acronym>",
			expected: "",
		},
		{
			name:     "Multiple acronym tags",
			template: "<acronym>first second</acronym> and <acronym>third fourth</acronym>",
			expected: "FS and TF",
		},
		{
			name:     "Special characters",
			template: "<acronym>test-world_here</acronym>",
			expected: "T",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestTrimTagProcessing tests the <trim> tag processing functionality
func TestTrimTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic trim",
			template: "<trim>  hello world  </trim>",
			expected: "hello world",
		},
		{
			name:     "Leading spaces only",
			template: "<trim>  hello world</trim>",
			expected: "hello world",
		},
		{
			name:     "Trailing spaces only",
			template: "<trim>hello world  </trim>",
			expected: "hello world",
		},
		{
			name:     "No spaces",
			template: "<trim>hello world</trim>",
			expected: "hello world",
		},
		{
			name:     "Tabs and newlines",
			template: "<trim>\t\nhello world\n\t</trim>",
			expected: "hello world",
		},
		{
			name:     "Empty content",
			template: "<trim></trim>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<trim>   </trim>",
			expected: "",
		},
		{
			name:     "Multiple trim tags",
			template: "<trim>  first  </trim> and <trim>  second  </trim>",
			expected: "first and second",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestSubstringTagProcessing tests the <substring> tag processing functionality
func TestSubstringTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic substring",
			template: `<substring start="2" end="5">hello world</substring>`,
			expected: "llo",
		},
		{
			name:     "Start from beginning",
			template: `<substring start="0" end="5">hello world</substring>`,
			expected: "hello",
		},
		{
			name:     "End at string length",
			template: `<substring start="6" end="11">hello world</substring>`,
			expected: "world",
		},
		{
			name:     "Single character",
			template: `<substring start="0" end="1">hello</substring>`,
			expected: "h",
		},
		{
			name:     "Empty substring",
			template: `<substring start="2" end="2">hello</substring>`,
			expected: "",
		},
		{
			name:     "Invalid start (negative)",
			template: `<substring start="-1" end="5">hello</substring>`,
			expected: "hello",
		},
		{
			name:     "Invalid end (too large)",
			template: `<substring start="0" end="20">hello</substring>`,
			expected: "hello",
		},
		{
			name:     "Start greater than end",
			template: `<substring start="5" end="2">hello</substring>`,
			expected: "",
		},
		{
			name:     "Empty content",
			template: `<substring start="0" end="5"></substring>`,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestReplaceTagProcessing tests the <replace> tag processing functionality
func TestReplaceTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic replace",
			template: `<replace search="world" replace="universe">hello world</replace>`,
			expected: "hello universe",
		},
		{
			name:     "Multiple occurrences",
			template: `<replace search="test" replace="example">test test test</replace>`,
			expected: "example example example",
		},
		{
			name:     "No match",
			template: `<replace search="xyz" replace="abc">hello world</replace>`,
			expected: "hello world",
		},
		{
			name:     "Empty search",
			template: `<replace search="" replace="abc">hello world</replace>`,
			expected: "abchabceabclabclabcoabc abcwabcoabcrabclabcdabc",
		},
		{
			name:     "Empty replace",
			template: `<replace search="world" replace="">hello world</replace>`,
			expected: "hello",
		},
		{
			name:     "Case sensitive",
			template: `<replace search="World" replace="Universe">hello world</replace>`,
			expected: "hello world",
		},
		{
			name:     "Special characters",
			template: `<replace search="123" replace="456">test123</replace>`,
			expected: "test456",
		},
		{
			name:     "Empty content",
			template: `<replace search="test" replace="example"></replace>`,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestPluralizeTagProcessing tests the <pluralize> tag processing functionality
func TestPluralizeTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic pluralize",
			template: "<pluralize>cat</pluralize>",
			expected: "cats",
		},
		{
			name:     "Already plural",
			template: "<pluralize>cats</pluralize>",
			expected: "cats",
		},
		{
			name:     "Words ending in s",
			template: "<pluralize>bus</pluralize>",
			expected: "buses",
		},
		{
			name:     "Words ending in y",
			template: "<pluralize>city</pluralize>",
			expected: "cities",
		},
		{
			name:     "Words ending in sh",
			template: "<pluralize>wish</pluralize>",
			expected: "wishes",
		},
		{
			name:     "Words ending in ch",
			template: "<pluralize>watch</pluralize>",
			expected: "watches",
		},
		{
			name:     "Words ending in x",
			template: "<pluralize>box</pluralize>",
			expected: "boxes",
		},
		{
			name:     "Words ending in z",
			template: "<pluralize>quiz</pluralize>",
			expected: "quizes",
		},
		{
			name:     "Irregular plurals",
			template: "<pluralize>child</pluralize>",
			expected: "children",
		},
		{
			name:     "Empty content",
			template: "<pluralize></pluralize>",
			expected: "",
		},
		{
			name:     "Multiple words",
			template: "<pluralize>cat dog</pluralize>",
			expected: "cats dogs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestShuffleTagProcessing tests the <shuffle> tag processing functionality
func TestShuffleTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	// Test basic shuffle functionality
	template := "<shuffle>one two three four</shuffle>"
	result := g.ProcessTemplateWithContext(template, make(map[string]string), session)

	// Split result and check that all words are present
	words := strings.Fields(result)
	expectedWords := []string{"one", "two", "three", "four"}

	if len(words) != len(expectedWords) {
		t.Errorf("Expected %d words, got %d", len(expectedWords), len(words))
	}

	// Check that all expected words are present
	wordMap := make(map[string]bool)
	for _, word := range words {
		wordMap[word] = true
	}

	for _, expectedWord := range expectedWords {
		if !wordMap[expectedWord] {
			t.Errorf("Expected word '%s' not found in result", expectedWord)
		}
	}

	// Test edge cases
	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Single word",
			template: "<shuffle>hello</shuffle>",
			expected: "hello",
		},
		{
			name:     "Empty content",
			template: "<shuffle></shuffle>",
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: "<shuffle>   </shuffle>",
			expected: "",
		},
		{
			name:     "Numbers and special characters",
			template: "<shuffle>1 2 3 4</shuffle>",
			expected: "1 2 3 4", // Should contain all numbers
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if tc.name == "Single word" || tc.name == "Empty content" || tc.name == "Whitespace only" {
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			} else if tc.name == "Numbers and special characters" {
				// Check that all numbers are present
				words := strings.Fields(result)
				if len(words) != 4 {
					t.Errorf("Expected 4 words, got %d", len(words))
				}
			}
		})
	}
}

// TestLengthTagProcessing tests the <length> tag processing functionality
func TestLengthTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic length",
			template: "<length>hello</length>",
			expected: "5",
		},
		{
			name:     "Empty content",
			template: "<length></length>",
			expected: "0",
		},
		{
			name:     "Whitespace only",
			template: "<length>   </length>",
			expected: "0",
		},
		{
			name:     "Numbers and special characters",
			template: "<length>test123!</length>",
			expected: "8",
		},
		{
			name:     "Unicode characters",
			template: "<length>café</length>",
			expected: "5",
		},
		{
			name:     "Multiple words",
			template: "<length>hello world</length>",
			expected: "11",
		},
		{
			name:     "With type parameter",
			template: `<length type="words">hello world test</length>`,
			expected: "3",
		},
		{
			name:     "With type parameter characters",
			template: `<length type="characters">hello world</length>`,
			expected: "11",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestCountTagProcessing tests the <count> tag processing functionality
func TestCountTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic count",
			template: `<count search="test">test test test</count>`,
			expected: "3",
		},
		{
			name:     "No matches",
			template: `<count search="xyz">hello world</count>`,
			expected: "0",
		},
		{
			name:     "Single match",
			template: `<count search="hello">hello world</count>`,
			expected: "1",
		},
		{
			name:     "Empty search",
			template: `<count search="">hello world</count>`,
			expected: "0",
		},
		{
			name:     "Empty content",
			template: `<count search="test"></count>`,
			expected: "0",
		},
		{
			name:     "Case sensitive",
			template: `<count search="Test">test Test TEST</count>`,
			expected: "1",
		},
		{
			name:     "Special characters",
			template: `<count search="123">test123 test456 test123</count>`,
			expected: "2",
		},
		{
			name:     "Overlapping matches",
			template: `<count search="aa">aaaa</count>`,
			expected: "2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestSplitTagProcessing tests the <split> tag processing functionality
func TestSplitTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic split",
			template: `<split delimiter=",">a,b,c</split>`,
			expected: "a b c",
		},
		{
			name:     "Default delimiter (space)",
			template: "<split>a b c</split>",
			expected: "a b c",
		},
		{
			name:     "With limit",
			template: `<split delimiter="," limit="2">a,b,c,d</split>`,
			expected: "a b,c,d",
		},
		{
			name:     "No delimiter matches",
			template: `<split delimiter="x">hello world</split>`,
			expected: "hello world",
		},
		{
			name:     "Empty content",
			template: `<split delimiter=","></split>`,
			expected: "",
		},
		{
			name:     "Empty delimiter",
			template: `<split delimiter="">hello world</split>`,
			expected: "hello world",
		},
		{
			name:     "Multiple delimiters",
			template: `<split delimiter=",">a,b,c,d,e</split>`,
			expected: "a b c d e",
		},
		{
			name:     "Limit greater than splits",
			template: `<split delimiter="," limit="10">a,b,c</split>`,
			expected: "a b c",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestJoinTagProcessing tests the <join> tag processing functionality
func TestJoinTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic join",
			template: `<join delimiter=",">a b c</join>`,
			expected: "a,b,c",
		},
		{
			name:     "Default delimiter (space)",
			template: "<join>a b c</join>",
			expected: "a b c",
		},
		{
			name:     "Custom delimiter",
			template: `<join delimiter="-">hello world test</join>`,
			expected: "hello-world-test",
		},
		{
			name:     "Single word",
			template: `<join delimiter=",">hello</join>`,
			expected: "hello",
		},
		{
			name:     "Empty content",
			template: `<join delimiter=","></join>`,
			expected: "",
		},
		{
			name:     "Whitespace only",
			template: `<join delimiter=",">   </join>`,
			expected: "",
		},
		{
			name:     "Empty delimiter",
			template: `<join delimiter="">a b c</join>`,
			expected: "a b c",
		},
		{
			name:     "Special characters in delimiter",
			template: `<join delimiter=" | ">a b c</join>`,
			expected: "a | b | c",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestIndentTagProcessing tests the <indent> tag processing functionality
func TestIndentTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic indent",
			template: "<indent>hello\nworld</indent>",
			expected: " hello\n world",
		},
		{
			name:     "Custom level",
			template: `<indent level="2">hello\nworld</indent>`,
			expected: "  hello\n  world",
		},
		{
			name:     "Custom character",
			template: `<indent char="\t">hello\nworld</indent>`,
			expected: "\thello\n\tworld",
		},
		{
			name:     "Custom level and character",
			template: `<indent level="3" char="-">hello\nworld</indent>`,
			expected: "---hello\n---world",
		},
		{
			name:     "Single line",
			template: "<indent>hello</indent>",
			expected: " hello",
		},
		{
			name:     "Empty content",
			template: "<indent></indent>",
			expected: "",
		},
		{
			name:     "Already indented",
			template: "<indent>  hello\n  world</indent>",
			expected: "   hello\n   world",
		},
		{
			name:     "Mixed indentation",
			template: "<indent>hello\n  world\n    test</indent>",
			expected: " hello\n   world\n     test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestDedentTagProcessing tests the <dedent> tag processing functionality
func TestDedentTagProcessing(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic dedent",
			template: "<dedent>    hello\n    world</dedent>",
			expected: "   hello\n   world",
		},
		{
			name:     "Custom level",
			template: `<dedent level="2">  hello\n  world</dedent>`,
			expected: "hello\nworld",
		},
		{
			name:     "Custom character",
			template: `<dedent char="\t">\thello\n\tworld</dedent>`,
			expected: "hello\nworld",
		},
		{
			name:     "Custom level and character",
			template: `<dedent level="3" char="-">---hello\n---world</dedent>`,
			expected: "hello\nworld",
		},
		{
			name:     "Single line",
			template: "<dedent>    hello</dedent>",
			expected: "   hello",
		},
		{
			name:     "Empty content",
			template: "<dedent></dedent>",
			expected: "",
		},
		{
			name:     "No indentation",
			template: "<dedent>hello\nworld</dedent>",
			expected: "hello\nworld",
		},
		{
			name:     "Mixed indentation",
			template: "<dedent>    hello\n      world\n        test</dedent>",
			expected: "   hello\n     world\n       test",
		},
		{
			name:     "Partial indentation",
			template: "<dedent>  hello\n    world\n  test</dedent>",
			expected: " hello\n   world\n test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestFormattingTagsIntegration tests integration of multiple formatting tags
func TestFormattingTagsIntegration(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Nested formatting tags",
			template: "<uppercase><formal>hello world</formal></uppercase>",
			expected: "HELLO WORLD",
		},
		{
			name:     "Sequential formatting tags",
			template: "<trim>  <formal>hello world</formal>  </trim>",
			expected: "Hello World",
		},
		{
			name:     "Complex formatting chain",
			template: "<trim><replace search=\"Test\" replace=\"Example\"><formal>this is a test</formal></replace></trim>",
			expected: "This Is A Example",
		},
		{
			name:     "Multiple independent tags",
			template: "<formal>hello</formal> <uppercase>world</uppercase>",
			expected: "Hello WORLD",
		},
		{
			name:     "Formatting with wildcards",
			template: "<formal><star/></formal>",
			expected: "Hello World",
		},
	}

	// Set up wildcards for the last test case
	wildcards := map[string]string{
		"star1": "hello world",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result string
			if tc.name == "Formatting with wildcards" {
				result = g.ProcessTemplateWithContext(tc.template, wildcards, session)
			} else {
				result = g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			}
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestFormattingTagsEdgeCases tests edge cases for formatting tags
func TestFormattingTagsEdgeCases(t *testing.T) {
	g := NewForTesting(t, false)
	session := g.createSession("test_session")

	testCases := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Malformed tag syntax",
			template: "<formal>hello world</formal",
			expected: "Hello World", // Tree processor is lenient with malformed closing tags
		},
		{
			name:     "Nested malformed tags",
			template: "<formal><uppercase>hello world</formal></uppercase>",
			expected: "<formal><uppercase>hello world</formal></uppercase>", // Tree processor preserves malformed nesting as text
		},
		{
			name:     "Empty tag",
			template: "<formal></formal>",
			expected: "",
		},
		{
			name:     "Whitespace only tag",
			template: "<formal>   </formal>",
			expected: "",
		},
		{
			name:     "Very long text",
			template: "<formal>" + strings.Repeat("test ", 10) + "</formal>",
			expected: strings.TrimSpace(strings.Repeat("Test ", 10)),
		},
		{
			name:     "Unicode characters",
			template: "<formal>café naïve résumé</formal>",
			expected: "Café Naïve Résumé",
		},
		{
			name:     "Special characters",
			template: "<formal>test@#$%^&*()</formal>",
			expected: "Test@#$%^&*()",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplateWithContext(tc.template, make(map[string]string), session)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
