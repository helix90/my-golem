package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// DataProcessor handles data processing (date, time, random, etc.)
type DataProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewDataProcessor creates a new data processor
func NewDataProcessor(g *golem.Golem) *DataProcessor {
	condition := golem.ProcessorCondition{
		SkipIfEmpty: true,
	}

	base := golem.NewBaseProcessor(
		"data",
		golem.ProcessorTypeData,
		golem.PriorityNormal,
		condition,
	)

	return &DataProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all data-related tags
func (p *DataProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process date and time tags
	response = p.golem.ProcessDateTimeTags(response)

	// Process random tags
	response = p.golem.ProcessRandomTags(response)

	return response, nil
}

// ShouldProcess determines if data processing should run
func (p *DataProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any data tags
	dataTags := []string{
		"<date", "</date>",
		"<time", "</time>",
		"<random", "</random>",
		"<li", "</li>",
	}

	for _, tag := range dataTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
