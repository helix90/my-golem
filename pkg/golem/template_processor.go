package golem

import (
	"strings"
	"sync"
	"time"
)

// ProcessorType represents the category of a template processor
type ProcessorType string

const (
	// ProcessorTypeWildcard handles wildcard replacement
	ProcessorTypeWildcard ProcessorType = "wildcard"

	// ProcessorTypeProperty handles property and bot tag processing
	ProcessorTypeProperty ProcessorType = "property"

	// ProcessorTypeVariable handles variable-related processing
	ProcessorTypeVariable ProcessorType = "variable"

	// ProcessorTypeRecursive handles recursive processing (SRAI, SRAIX)
	ProcessorTypeRecursive ProcessorType = "recursive"

	// ProcessorTypeLearning handles dynamic learning
	ProcessorTypeLearning ProcessorType = "learning"

	// ProcessorTypeConditional handles conditional processing
	ProcessorTypeConditional ProcessorType = "conditional"

	// ProcessorTypeData handles data processing (date, time, random, etc.)
	ProcessorTypeData ProcessorType = "data"

	// ProcessorTypeCollection handles collection processing (list, array, map)
	ProcessorTypeCollection ProcessorType = "collection"

	// ProcessorTypeText handles text processing (person, gender, sentence, word)
	ProcessorTypeText ProcessorType = "text"

	// ProcessorTypeFormat handles text formatting (uppercase, lowercase, etc.)
	ProcessorTypeFormat ProcessorType = "format"

	// ProcessorTypeHistory handles history processing (request, response, that)
	ProcessorTypeHistory ProcessorType = "history"

	// ProcessorTypeSystem handles system information (version, id, size)
	ProcessorTypeSystem ProcessorType = "system"
)

// ProcessorPriority defines the execution priority of processors
type ProcessorPriority int

const (
	PriorityEarly  ProcessorPriority = 100 // Early processing (think, topic, set)
	PriorityNormal ProcessorPriority = 200 // Normal processing
	PriorityLate   ProcessorPriority = 300 // Late processing (text formatting)
	PriorityFinal  ProcessorPriority = 400 // Final processing (history, system)
)

// ProcessorCondition defines when a processor should run
type ProcessorCondition struct {
	RequiresContext bool   // Whether context is required
	RequiresKB      bool   // Whether knowledge base is required
	RequiresSession bool   // Whether session is required
	SkipIfEmpty     bool   // Skip if template is empty
	CustomCheck     string // Custom condition function name
}

// ProcessorMetrics tracks performance metrics for a processor
type ProcessorMetrics struct {
	TotalCalls   int64         `json:"total_calls"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	LastCallTime time.Time     `json:"last_call_time"`
	ErrorCount   int64         `json:"error_count"`
	CacheHits    int64         `json:"cache_hits"`
	CacheMisses  int64         `json:"cache_misses"`
}

// TemplateProcessor defines the interface for template processors
type TemplateProcessor interface {
	// Name returns the processor name
	Name() string

	// Type returns the processor type
	Type() ProcessorType

	// Priority returns the processor priority
	Priority() ProcessorPriority

	// Condition returns the processor condition
	Condition() ProcessorCondition

	// Process processes the template content
	Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error)

	// ShouldProcess determines if the processor should run
	ShouldProcess(template string, ctx *VariableContext) bool

	// GetMetrics returns processor metrics
	GetMetrics() *ProcessorMetrics

	// ResetMetrics resets processor metrics
	ResetMetrics()
}

// ProcessorRegistry manages template processors
type ProcessorRegistry struct {
	processors map[string]TemplateProcessor
	order      []string
	metrics    map[string]*ProcessorMetrics
	metricsMu  sync.Mutex // guards metrics, which are updated by concurrent template processing
}

// NewProcessorRegistry creates a new processor registry
func NewProcessorRegistry() *ProcessorRegistry {
	return &ProcessorRegistry{
		processors: make(map[string]TemplateProcessor),
		order:      make([]string, 0),
		metrics:    make(map[string]*ProcessorMetrics),
	}
}

// RegisterProcessor registers a processor
func (r *ProcessorRegistry) RegisterProcessor(processor TemplateProcessor) {
	name := processor.Name()
	r.processors[name] = processor
	r.metrics[name] = &ProcessorMetrics{}

	// Insert processor in correct order based on priority
	inserted := false
	for i, existingName := range r.order {
		if r.processors[existingName].Priority() > processor.Priority() {
			r.order = append(r.order[:i], append([]string{name}, r.order[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		r.order = append(r.order, name)
	}
}

// GetProcessor returns a processor by name
func (r *ProcessorRegistry) GetProcessor(name string) (TemplateProcessor, bool) {
	processor, exists := r.processors[name]
	return processor, exists
}

// GetProcessorsByType returns all processors of a specific type
func (r *ProcessorRegistry) GetProcessorsByType(processorType ProcessorType) []TemplateProcessor {
	var processors []TemplateProcessor
	for _, name := range r.order {
		if processor := r.processors[name]; processor.Type() == processorType {
			processors = append(processors, processor)
		}
	}
	return processors
}

// GetAllProcessors returns all processors in execution order
func (r *ProcessorRegistry) GetAllProcessors() []TemplateProcessor {
	var processors []TemplateProcessor
	for _, name := range r.order {
		processors = append(processors, r.processors[name])
	}
	return processors
}

// ProcessTemplate processes a template using all registered processors
func (r *ProcessorRegistry) ProcessTemplate(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template

	for _, processor := range r.GetAllProcessors() {
		if !processor.ShouldProcess(response, ctx) {
			continue
		}

		startTime := time.Now()
		processed, err := processor.Process(response, wildcards, ctx)
		processingTime := time.Since(startTime)

		// Update metrics
		r.metricsMu.Lock()
		metrics := r.metrics[processor.Name()]
		metrics.TotalCalls++
		metrics.TotalTime += processingTime
		metrics.AverageTime = time.Duration(int64(metrics.TotalTime) / metrics.TotalCalls)
		metrics.LastCallTime = time.Now()
		if err != nil {
			metrics.ErrorCount++
		}
		r.metricsMu.Unlock()

		if err != nil {
			return response, err
		}

		response = processed
	}

	return response, nil
}

// GetMetrics returns a snapshot of the metrics for all processors
func (r *ProcessorRegistry) GetMetrics() map[string]*ProcessorMetrics {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	snapshot := make(map[string]*ProcessorMetrics, len(r.metrics))
	for name, metrics := range r.metrics {
		copied := *metrics
		snapshot[name] = &copied
	}
	return snapshot
}

// ResetMetrics resets metrics for all processors
func (r *ProcessorRegistry) ResetMetrics() {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	for _, metrics := range r.metrics {
		*metrics = ProcessorMetrics{}
	}
}

// recordCall counts one call to the named processor
func (r *ProcessorRegistry) recordCall(name string) {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()
	if metrics := r.metrics[name]; metrics != nil {
		metrics.TotalCalls++
		metrics.LastCallTime = time.Now()
	}
}

// BaseProcessor provides common functionality for processors
type BaseProcessor struct {
	name          string
	processorType ProcessorType
	priority      ProcessorPriority
	condition     ProcessorCondition
	metrics       *ProcessorMetrics
}

// NewBaseProcessor creates a new base processor
func NewBaseProcessor(name string, processorType ProcessorType, priority ProcessorPriority, condition ProcessorCondition) *BaseProcessor {
	return &BaseProcessor{
		name:          name,
		processorType: processorType,
		priority:      priority,
		condition:     condition,
		metrics:       &ProcessorMetrics{},
	}
}

// Name returns the processor name
func (p *BaseProcessor) Name() string {
	return p.name
}

// Type returns the processor type
func (p *BaseProcessor) Type() ProcessorType {
	return p.processorType
}

// Priority returns the processor priority
func (p *BaseProcessor) Priority() ProcessorPriority {
	return p.priority
}

// Condition returns the processor condition
func (p *BaseProcessor) Condition() ProcessorCondition {
	return p.condition
}

// GetMetrics returns processor metrics
func (p *BaseProcessor) GetMetrics() *ProcessorMetrics {
	return p.metrics
}

// ResetMetrics resets processor metrics
func (p *BaseProcessor) ResetMetrics() {
	*p.metrics = ProcessorMetrics{}
}

// ShouldProcess determines if the processor should run
func (p *BaseProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	// Check if template is empty and we should skip
	if p.condition.SkipIfEmpty && strings.TrimSpace(template) == "" {
		return false
	}

	// Check context requirements
	if p.condition.RequiresContext && ctx == nil {
		return false
	}

	// Check knowledge base requirements
	if p.condition.RequiresKB && (ctx == nil || ctx.KnowledgeBase == nil) {
		return false
	}

	// Check session requirements
	if p.condition.RequiresSession && (ctx == nil || ctx.Session == nil) {
		return false
	}

	// Custom check would be implemented by specific processors
	return true
}
