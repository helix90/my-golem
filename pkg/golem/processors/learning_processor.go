package processors

import (
	"strings"

	"github.com/helix90/my-golem/pkg/golem"
)

// LearningProcessor handles dynamic learning operations
type LearningProcessor struct {
	*golem.BaseProcessor
	golem *golem.Golem
}

// NewLearningProcessor creates a new learning processor
func NewLearningProcessor(g *golem.Golem) *LearningProcessor {
	condition := golem.ProcessorCondition{
		RequiresContext: true,
		RequiresKB:      true,
		SkipIfEmpty:     true,
	}

	base := golem.NewBaseProcessor(
		"learning",
		golem.ProcessorTypeLearning,
		golem.PriorityNormal,
		condition,
	)

	return &LearningProcessor{
		BaseProcessor: base,
		golem:         g,
	}
}

// Process processes all learning-related tags
func (p *LearningProcessor) Process(template string, wildcards map[string]string, ctx *golem.VariableContext) (string, error) {
	response := template

	// Process learn tags (dynamic learning)
	response = p.golem.ProcessLearnTagsWithContext(response, ctx)

	// Process unlearn tags (remove learned categories)
	response = p.golem.ProcessUnlearnTagsWithContext(response, ctx)

	return response, nil
}

// ShouldProcess determines if learning processing should run
func (p *LearningProcessor) ShouldProcess(template string, ctx *golem.VariableContext) bool {
	// Check if template contains any learning tags
	learningTags := []string{
		"<learn", "</learn>",
		"<unlearn", "</unlearn>",
	}

	for _, tag := range learningTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
