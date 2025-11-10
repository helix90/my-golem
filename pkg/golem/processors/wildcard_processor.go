package processors

import (
	"fmt"
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// WildcardProcessor handles wildcard replacement in templates
type WildcardProcessor struct {
	*golem.BaseProcessor
}

// NewWildcardProcessor creates a new wildcard processor
func NewWildcardProcessor() *WildcardProcessor {
	condition := golem.ProcessorCondition{
		SkipIfEmpty: true,
	}

	base := golem.NewBaseProcessor(
		"wildcard",
		golem.ProcessorTypeWildcard,
		golem.PriorityEarly,
		condition,
	)

	return &WildcardProcessor{
		BaseProcessor: base,
	}
}

// Process processes wildcard replacements
func (p *WildcardProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Store wildcards in context for that wildcard processing
	if ctx != nil {
		for key, value := range wildcards {
			ctx.LocalVars[key] = value
		}
	}

	// Replace indexed star tags first
	for key, value := range wildcards {
		switch key {
		case "star1":
			response = strings.ReplaceAll(response, "<star index=\"1\"/>", value)
			response = strings.ReplaceAll(response, "<star1/>", value)
		case "star2":
			response = strings.ReplaceAll(response, "<star index=\"2\"/>", value)
			response = strings.ReplaceAll(response, "<star2/>", value)
		case "star3":
			response = strings.ReplaceAll(response, "<star index=\"3\"/>", value)
			response = strings.ReplaceAll(response, "<star3/>", value)
		case "star4":
			response = strings.ReplaceAll(response, "<star index=\"4\"/>", value)
			response = strings.ReplaceAll(response, "<star4/>", value)
		case "star5":
			response = strings.ReplaceAll(response, "<star index=\"5\"/>", value)
			response = strings.ReplaceAll(response, "<star5/>", value)
		case "star6":
			response = strings.ReplaceAll(response, "<star index=\"6\"/>", value)
			response = strings.ReplaceAll(response, "<star6/>", value)
		case "star7":
			response = strings.ReplaceAll(response, "<star index=\"7\"/>", value)
			response = strings.ReplaceAll(response, "<star7/>", value)
		case "star8":
			response = strings.ReplaceAll(response, "<star index=\"8\"/>", value)
			response = strings.ReplaceAll(response, "<star8/>", value)
		case "star9":
			response = strings.ReplaceAll(response, "<star index=\"9\"/>", value)
			response = strings.ReplaceAll(response, "<star9/>", value)
		}
	}

	// Then replace generic <star/> tags sequentially
	starIndex := 1
	for strings.Contains(response, "<star/>") && starIndex <= 9 {
		key := fmt.Sprintf("star%d", starIndex)
		if value, exists := wildcards[key]; exists {
			// Replace only the first occurrence
			response = strings.Replace(response, "<star/>", value, 1)
		} else if len(wildcards) == 1 {
			// If there's only one wildcard captured, use it for all remaining <star/> tags
			for _, value := range wildcards {
				response = strings.Replace(response, "<star/>", value, 1)
				break
			}
		} else {
			// If no wildcard value exists, replace with empty string
			response = strings.Replace(response, "<star/>", "", 1)
		}
		starIndex++
	}

	return response, nil
}

// ShouldProcess determines if wildcard processing should run
func (p *WildcardProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Only process if there are wildcards to replace
	return strings.Contains(template, "<star") || strings.Contains(template, "<star/>")
}
