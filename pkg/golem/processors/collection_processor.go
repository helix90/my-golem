package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// CollectionProcessor handles collection processing (list, array, map)
type CollectionProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewCollectionProcessor creates a new collection processor
func NewCollectionProcessor(g *golem.Golem) *CollectionProcessor {
	condition := golem.ProcessorCondition{
		RequiresContext: true,
		SkipIfEmpty:     true,
	}

	base := golem.NewBaseProcessor(
		"collection",
		golem.ProcessorTypeCollection,
		golem.PriorityNormal,
		condition,
	)

	return &CollectionProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all collection-related tags
func (p *CollectionProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process map tags
	response = p.golem.ProcessMapTagsWithContext(response, ctx)

	// Process list tags
	response = p.golem.ProcessListTagsWithContext(response, ctx)

	// Process array tags
	response = p.golem.ProcessArrayTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if collection processing should run
func (p *CollectionProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any collection tags
	collectionTags := []string{
		"<map", "</map>",
		"<list", "</list>",
		"<array", "</array>",
		"<item", "</item>",
		"<index", "</index>",
		"<size", "</size>",
	}

	for _, tag := range collectionTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
