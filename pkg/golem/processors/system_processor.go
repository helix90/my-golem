package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// SystemProcessor handles system information processing
type SystemProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewSystemProcessor creates a new system processor
func NewSystemProcessor(g *golem.Golem) *SystemProcessor {
	condition := golem.ProcessorCondition{
		RequiresContext: true,
		SkipIfEmpty:     true,
	}

	base := golem.NewBaseProcessor(
		"system",
		golem.ProcessorTypeSystem,
		golem.PriorityFinal,
		condition,
	)

	return &SystemProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all system-related tags
func (p *SystemProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process size tags (knowledge base size)
	response = p.golem.ProcessSizeTagsWithContext(response, ctx)

	// Process version tags (AIML version)
	response = p.golem.ProcessVersionTagsWithContext(response, ctx)

	// Process id tags (session ID)
	response = p.golem.ProcessIdTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if system processing should run
func (p *SystemProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any system tags
	systemTags := []string{
		"<size", "</size>",
		"<version", "</version>",
		"<id", "</id>",
	}

	for _, tag := range systemTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
