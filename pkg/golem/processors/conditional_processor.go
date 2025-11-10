package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// ConditionalProcessor handles conditional processing
type ConditionalProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewConditionalProcessor creates a new conditional processor
func NewConditionalProcessor(g *golem.Golem) *ConditionalProcessor {
	condition := golem.ProcessorCondition{
		RequiresContext: true,
		SkipIfEmpty:     true,
	}

	base := golem.NewBaseProcessor(
		"conditional",
		golem.ProcessorTypeConditional,
		golem.PriorityNormal,
		condition,
	)

	return &ConditionalProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all conditional tags
func (p *ConditionalProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process condition tags
	response = p.golem.ProcessConditionTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if conditional processing should run
func (p *ConditionalProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any conditional tags
	conditionalTags := []string{
		"<condition", "</condition>",
		"<if", "</if>",
		"<else", "</else>",
		"<elseif", "</elseif>",
	}

	for _, tag := range conditionalTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
