package golem

import (
	"strings"
	"testing"
)

// TestNoTagsInOutput verifies that XML tags never appear in template processing output
func TestNoTagsInOutput(t *testing.T) {
	g := NewForTesting(t, true)
	
	testCases := []struct {
		name     string
		template string
		wildcards map[string]string
	}{
		{
			name:      "SRAIX with no service configured",
			template:  `<sraix service="pannous">Test query</sraix>`,
			wildcards: map[string]string{},
		},
		{
			name:      "SRAIX with star tag",
			template:  `<sraix service="test"><star/></sraix>`,
			wildcards: map[string]string{"star1": "testvalue"},
		},
		{
			name:      "Complex nested tags",
			template:  `<sraix service="test"><uppercase><star/></uppercase></sraix>`,
			wildcards: map[string]string{"star1": "hello"},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := g.ProcessTemplate(tc.template, tc.wildcards)
			
			// Verify no XML tags in output
			if strings.Contains(result, "<") && strings.Contains(result, ">") {
				t.Errorf("Output contains XML tags: %q", result)
			}
			
			// Verify specifically no sraix tags
			if strings.Contains(result, "<sraix") {
				t.Errorf("Output contains <sraix> tag: %q", result)
			}
		})
	}
}
