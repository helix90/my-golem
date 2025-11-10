package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// RecursiveProcessor handles recursive processing (SRAI, SRAIX)
type RecursiveProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewRecursiveProcessor creates a new recursive processor
func NewRecursiveProcessor(g *golem.Golem) *RecursiveProcessor {
	condition := golem.ProcessorCondition{
		RequiresContext: true,
		RequiresKB:      true,
		SkipIfEmpty:     true,
	}

	base := golem.NewBaseProcessor(
		"recursive",
		golem.ProcessorTypeRecursive,
		golem.PriorityNormal,
		condition,
	)

	return &RecursiveProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all recursive tags
func (p *RecursiveProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process SR tags (shorthand for <srai><star/></srai>) AFTER wildcard replacement
	// Note: SR tags should be converted to SRAI format before wildcard replacement
	// but we need to process them after to work with the actual wildcard values
	response = p.golem.ProcessSRTagsWithContext(response, wildcards, ctx)

	// Process SRAI tags (recursive)
	response = p.golem.ProcessSRAITagsWithContext(response, ctx)

	// Process SRAIX tags (external services)
	response = p.golem.ProcessSRAIXTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if recursive processing should run
func (p *RecursiveProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any recursive tags
	recursiveTags := []string{
		"<srai", "</srai>",
		"<sraix", "</sraix>",
		"<sr", "</sr>",
	}

	for _, tag := range recursiveTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
