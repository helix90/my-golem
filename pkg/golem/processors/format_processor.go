package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// FormatProcessor handles all text formatting operations
type FormatProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewFormatProcessor creates a new format processor
func NewFormatProcessor(g *golem.Golem) *FormatProcessor {
	condition := golem.ProcessorCondition{
		SkipIfEmpty: true,
	}

	base := golem.NewBaseProcessor(
		"format",
		golem.ProcessorTypeFormat,
		golem.PriorityLate,
		condition,
	)

	return &FormatProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all text formatting tags
func (p *FormatProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process topic tags (current topic references) - before text processing
	response = p.golem.ProcessTopicTagsWithContext(response, ctx)

	// Process repeat tags first (before text formatting) so they can be processed by other tags
	response = p.golem.ProcessRepeatTagsWithContext(response, ctx)

	// Process all text formatting tags
	response = p.golem.ProcessUppercaseTagsWithContext(response, ctx)
	response = p.golem.ProcessLowercaseTagsWithContext(response, ctx)
	response = p.golem.ProcessFormalTagsWithContext(response, ctx)
	response = p.golem.ProcessExplodeTagsWithContext(response, ctx)
	response = p.golem.ProcessCapitalizeTagsWithContext(response, ctx)
	response = p.golem.ProcessReverseTagsWithContext(response, ctx)
	response = p.golem.ProcessAcronymTagsWithContext(response, ctx)
	response = p.golem.ProcessTrimTagsWithContext(response, ctx)
	response = p.golem.ProcessSubstringTagsWithContext(response, ctx)
	response = p.golem.ProcessReplaceTagsWithContext(response, ctx)
	response = p.golem.ProcessPluralizeTagsWithContext(response, ctx)
	response = p.golem.ProcessShuffleTagsWithContext(response, ctx)
	response = p.golem.ProcessLengthTagsWithContext(response, ctx)
	response = p.golem.ProcessCountTagsWithContext(response, ctx)
	response = p.golem.ProcessSplitTagsWithContext(response, ctx)
	response = p.golem.ProcessJoinTagsWithContext(response, ctx)
	response = p.golem.ProcessIndentTagsWithContext(response, ctx)
	response = p.golem.ProcessDedentTagsWithContext(response, ctx)
	response = p.golem.ProcessUniqueTagsWithContext(response, ctx)

	// Process normalize tags (text normalization)
	response = p.golem.ProcessNormalizeTagsWithContext(response, ctx)

	// Process denormalize tags (text denormalization)
	response = p.golem.ProcessDenormalizeTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if format processing should run
func (p *FormatProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any formatting tags
	formatTags := []string{
		"<uppercase", "</uppercase>",
		"<lowercase", "</lowercase>",
		"<formal", "</formal>",
		"<explode", "</explode>",
		"<capitalize", "</capitalize>",
		"<reverse", "</reverse>",
		"<acronym", "</acronym>",
		"<trim", "</trim>",
		"<substring", "</substring>",
		"<replace", "</replace>",
		"<pluralize", "</pluralize>",
		"<shuffle", "</shuffle>",
		"<length", "</length>",
		"<count", "</count>",
		"<split", "</split>",
		"<join", "</join>",
		"<indent", "</indent>",
		"<dedent", "</dedent>",
		"<unique", "</unique>",
		"<repeat", "</repeat>",
		"<normalize", "</normalize>",
		"<denormalize", "</denormalize>",
		"<topic", "</topic>",
	}

	for _, tag := range formatTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
