package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// VariableProcessor handles all variable-related processing
type VariableProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewVariableProcessor creates a new variable processor
func NewVariableProcessor(g *golem.Golem) *VariableProcessor {
	condition := golem.ProcessorCondition{
		RequiresContext: true,
		SkipIfEmpty:     true,
	}

	base := golem.NewBaseProcessor(
		"variable",
		golem.ProcessorTypeVariable,
		golem.PriorityEarly,
		condition,
	)

	return &VariableProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all variable-related tags
func (p *VariableProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Replace property tags
	response = p.golem.ReplacePropertyTags(response)

	// Process bot tags (short form of property access)
	response = p.golem.ProcessBotTagsWithContext(response, ctx)

	// Process think tags FIRST (internal processing, no output)
	// This allows local variables to be set before variable replacement
	response = p.golem.ProcessThinkTagsWithContext(response, ctx)

	// Process topic setting tags first (special handling for topic)
	response = p.golem.ProcessTopicSettingTagsWithContext(response, ctx)

	// Process set tags (before session variable replacement)
	response = p.golem.ProcessSetTagsWithContext(response, ctx)

	// Replace session variable tags using context
	response = p.golem.ReplaceSessionVariableTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if variable processing should run
func (p *VariableProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any variable-related tags
	variableTags := []string{
		"<get", "</get>",
		"<set", "</set>",
		"<bot", "</bot>",
		"<think", "</think>",
		"<topic", "</topic>",
		"<name", "</name>",
		"<value", "</value>",
	}

	for _, tag := range variableTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
