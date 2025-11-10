package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// TextProcessor handles all text-related processing (person, gender, sentence, word)
type TextProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewTextProcessor creates a new text processor
func NewTextProcessor(g *golem.Golem) *TextProcessor {
	condition := golem.ProcessorCondition{
		SkipIfEmpty: true,
	}

	base := golem.NewBaseProcessor(
		"text",
		golem.ProcessorTypeText,
		golem.PriorityNormal,
		condition,
	)

	return &TextProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all text-related tags
func (p *TextProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process person tags (pronoun substitution)
	response = p.golem.ProcessPersonTagsWithContext(response, ctx)

	// Process gender tags (gender pronoun substitution)
	response = p.golem.ProcessGenderTagsWithContext(response, ctx)

	// Process person2 tags (first-to-third person pronoun substitution)
	response = p.golem.ProcessPerson2TagsWithContext(response, ctx)

	// Process sentence tags (sentence-level processing)
	response = p.golem.ProcessSentenceTagsWithContext(response, ctx)

	// Process word tags (word-level processing)
	response = p.golem.ProcessWordTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if text processing should run
func (p *TextProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any text processing tags
	textTags := []string{
		"<person", "</person>",
		"<gender", "</gender>",
		"<person2", "</person2>",
		"<sentence", "</sentence>",
		"<word", "</word>",
	}

	for _, tag := range textTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
