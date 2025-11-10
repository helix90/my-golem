package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// HistoryProcessor handles history processing (request, response, that)
type HistoryProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewHistoryProcessor creates a new history processor
func NewHistoryProcessor(g *golem.Golem) *HistoryProcessor {
	condition := golem.ProcessorCondition{
		RequiresContext: true,
		RequiresSession: true,
		SkipIfEmpty:     true,
	}

	base := golem.NewBaseProcessor(
		"history",
		golem.ProcessorTypeHistory,
		golem.PriorityFinal,
		condition,
	)

	return &HistoryProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all history-related tags
func (p *HistoryProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process that wildcard tags (that context wildcards)
	response = p.golem.ProcessThatWildcardTagsWithContext(response, ctx)

	// Process that tags (bot response references)
	response = p.golem.ProcessThatTagsWithContext(response, ctx)

	// Process request tags (user input history)
	response = p.golem.ProcessRequestTags(response, ctx)

	// Process response tags (bot response history)
	response = p.golem.ProcessResponseTags(response, ctx)

	return response, nil
}

// ShouldProcess determines if history processing should run
func (p *HistoryProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any history tags
	historyTags := []string{
		"<that", "</that>",
		"<request", "</request>",
		"<response", "</response>",
	}

	for _, tag := range historyTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
