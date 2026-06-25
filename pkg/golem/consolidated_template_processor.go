package golem

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ConsolidatedTemplateProcessor manages the consolidated template processing pipeline
type ConsolidatedTemplateProcessor struct {
	registry *ProcessorRegistry
	golem    *Golem
}

// NewConsolidatedTemplateProcessor creates a new consolidated template processor
func NewConsolidatedTemplateProcessor(g *Golem) *ConsolidatedTemplateProcessor {
	registry := NewProcessorRegistry()

	// Register comprehensive processors in the correct order
	// Processing order: wildcard -> variable -> recursive -> data -> text -> format -> collection -> system
	registry.RegisterProcessor(&ComprehensiveWildcardProcessor{golem: g})
	registry.RegisterProcessor(&ComprehensiveVariableProcessor{golem: g})
	registry.RegisterProcessor(&ComprehensiveRecursiveProcessor{golem: g})
	registry.RegisterProcessor(&ComprehensiveDataProcessor{golem: g})
	registry.RegisterProcessor(&ComprehensiveTextProcessor{golem: g})
	registry.RegisterProcessor(&ComprehensiveFormatProcessor{golem: g})
	registry.RegisterProcessor(&ComprehensiveCollectionProcessor{golem: g})
	registry.RegisterProcessor(&ComprehensiveSystemProcessor{golem: g})

	return &ConsolidatedTemplateProcessor{
		registry: registry,
		golem:    g,
	}
}

// ProcessTemplate processes a template using the consolidated pipeline
func (ctp *ConsolidatedTemplateProcessor) ProcessTemplate(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	startTime := time.Now()

	// Check cache first if enabled
	hasConditionTags := strings.Contains(template, "<condition ")
	if ctp.golem.templateConfig.EnableCaching && !hasConditionTags {
		cacheKey := ctp.golem.generateTemplateCacheKey(template, wildcards, ctx)
		if cached, found := ctp.golem.getFromTemplateCache(cacheKey); found {
			ctp.golem.templateMetrics.CacheHits++
			ctp.golem.updateCacheHitRate()
			ctp.golem.LogDebug("Template cache hit for key: %s", cacheKey)
			return cached, nil
		}
		ctp.golem.templateMetrics.CacheMisses++
		ctp.golem.updateCacheHitRate()
	}

	// Log initial state
	ctp.golem.LogInfo("Template text: '%s'", template)
	ctp.golem.LogInfo("Wildcards: %v", wildcards)

	// Process using the consolidated pipeline
	response, err := ctp.registry.ProcessTemplate(template, wildcards, ctx)
	if err != nil {
		return template, err
	}

	// One additional resolving pass for variables/conditions/collections in case
	// formatting/text passes revealed or produced new tags
	response = ctp.golem.replaceSessionVariableTagsWithContext(response, ctx)
	response = ctp.golem.processConditionTagsWithContext(response, ctx)
	// Re-run collection retrieval/mutations if any remain
	response = ctp.golem.processMapTagsWithContext(response, ctx)
	response = ctp.golem.processListTagsWithContext(response, ctx)
	response = ctp.golem.processArrayTagsWithContext(response, ctx)
	// Re-run SRAI processing in case input tags produced new SRAI tags
	response = ctp.golem.processSRAITagsWithContext(response, ctx)

	ctp.golem.LogInfo("Final response: '%s'", response)

	// Smart trimming: preserve intentional indentation; collapse whitespace-only to empty
	finalResponse := response
	if len(response) > 0 {
		// Collapse whitespace-only output to empty (collections/tests expect empty)
		if strings.TrimSpace(response) == "" {
			finalResponse = ""
		} else if response[0] != ' ' && response[0] != '\t' {
			// If it doesn't start with intentional indentation, trim normally
			finalResponse = strings.TrimSpace(response)
		}
		// If it starts with space/tab and has non-whitespace content, preserve it (intentional indentation)
	}

	// Update metrics
	processingTime := float64(time.Since(startTime).Nanoseconds()) / 1000000.0 // Convert to milliseconds
	ctp.golem.templateMetrics.TotalProcessed++
	ctp.golem.templateMetrics.LastProcessed = time.Now().Format(time.RFC3339)

	// Update average processing time
	if ctp.golem.templateMetrics.TotalProcessed == 1 {
		ctp.golem.templateMetrics.AverageProcessTime = processingTime
	} else {
		ctp.golem.templateMetrics.AverageProcessTime = (ctp.golem.templateMetrics.AverageProcessTime*float64(ctp.golem.templateMetrics.TotalProcessed-1) + processingTime) / float64(ctp.golem.templateMetrics.TotalProcessed)
	}

	// Cache the result if caching is enabled
	if ctp.golem.templateConfig.EnableCaching && !hasConditionTags {
		cacheKey := ctp.golem.generateTemplateCacheKey(template, wildcards, ctx)
		ctp.golem.storeInTemplateCache(cacheKey, finalResponse)
	}

	// Update memory peak
	currentMemory := len([]byte(finalResponse))
	if currentMemory > ctp.golem.templateMetrics.MemoryPeak {
		ctp.golem.templateMetrics.MemoryPeak = currentMemory
	}

	return finalResponse, nil
}

// GetProcessorMetrics returns metrics for all processors
func (ctp *ConsolidatedTemplateProcessor) GetProcessorMetrics() map[string]*ProcessorMetrics {
	return ctp.registry.GetMetrics()
}

// GetProcessor returns a specific processor
func (ctp *ConsolidatedTemplateProcessor) GetProcessor(name string) (TemplateProcessor, bool) {
	return ctp.registry.GetProcessor(name)
}

// GetProcessorsByType returns processors of a specific type
func (ctp *ConsolidatedTemplateProcessor) GetProcessorsByType(processorType ProcessorType) []TemplateProcessor {
	return ctp.registry.GetProcessorsByType(processorType)
}

// ResetMetrics resets all processor metrics
func (ctp *ConsolidatedTemplateProcessor) ResetMetrics() {
	ctp.registry.ResetMetrics()
}

// GetProcessingOrder returns the current processing order
func (ctp *ConsolidatedTemplateProcessor) GetProcessingOrder() []string {
	var order []string
	for _, processor := range ctp.registry.GetAllProcessors() {
		order = append(order, processor.Name())
	}
	return order
}

// SetProcessingOrder allows reordering of processors (for advanced use cases)
func (ctp *ConsolidatedTemplateProcessor) SetProcessingOrder(order []string) error {
	// Validate that all processors exist
	for _, name := range order {
		if _, exists := ctp.registry.processors[name]; !exists {
			return fmt.Errorf("processor %s not found", name)
		}
	}

	ctp.registry.order = order
	return nil
}

// EnableProcessor enables a specific processor
func (ctp *ConsolidatedTemplateProcessor) EnableProcessor(name string) error {
	processor, exists := ctp.registry.GetProcessor(name)
	if !exists {
		return fmt.Errorf("processor %s not found", name)
	}

	// For now, we don't have a disabled state, but this could be extended
	// to support enabling/disabling processors dynamically
	_ = processor
	return nil
}

// DisableProcessor disables a specific processor
func (ctp *ConsolidatedTemplateProcessor) DisableProcessor(name string) error {
	// For now, we don't have a disabled state, but this could be extended
	// to support enabling/disabling processors dynamically
	return fmt.Errorf("processor disabling not yet implemented")
}

// GetProcessorStats returns detailed statistics for all processors
func (ctp *ConsolidatedTemplateProcessor) GetProcessorStats() map[string]interface{} {
	stats := make(map[string]interface{})

	for name, metrics := range ctp.registry.GetMetrics() {
		stats[name] = map[string]interface{}{
			"total_calls":     metrics.TotalCalls,
			"total_time_ms":   metrics.TotalTime.Milliseconds(),
			"average_time_ms": metrics.AverageTime.Milliseconds(),
			"last_call_time":  metrics.LastCallTime.Format(time.RFC3339),
			"error_count":     metrics.ErrorCount,
			"cache_hits":      metrics.CacheHits,
			"cache_misses":    metrics.CacheMisses,
			"hit_rate":        float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses),
		}
	}

	return stats
}

// ComprehensiveWildcardProcessor handles wildcard processing (star tags and that wildcards)
type ComprehensiveWildcardProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveWildcardProcessor) Name() string                { return "wildcard" }
func (p *ComprehensiveWildcardProcessor) Type() ProcessorType         { return ProcessorTypeWildcard }
func (p *ComprehensiveWildcardProcessor) Priority() ProcessorPriority { return PriorityEarly }
func (p *ComprehensiveWildcardProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{SkipIfEmpty: true}
}
func (p *ComprehensiveWildcardProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
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
			// If no wildcard value exists, leave the tag as-is (don't replace)
			// This allows <star/> tags to be preserved in learned templates
			break
		}
		starIndex++
	}

	// Process that wildcard tags (that context wildcards)
	response = p.golem.processThatWildcardTagsWithContext(response, ctx)

	return response, nil
}
func (p *ComprehensiveWildcardProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	return strings.Contains(template, "<star") || strings.Contains(template, "<that_")
}
func (p *ComprehensiveWildcardProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveWildcardProcessor) ResetMetrics()                 {}

// ComprehensiveVariableProcessor handles variable processing (property, bot, think, topic, set, condition tags)
type ComprehensiveVariableProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveVariableProcessor) Name() string                { return "variable" }
func (p *ComprehensiveVariableProcessor) Type() ProcessorType         { return ProcessorTypeVariable }
func (p *ComprehensiveVariableProcessor) Priority() ProcessorPriority { return PriorityEarly }
func (p *ComprehensiveVariableProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{RequiresContext: true, SkipIfEmpty: true}
}
func (p *ComprehensiveVariableProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template

	// Replace property tags
	response = p.golem.replacePropertyTags(response)

	// Process bot tags (short form of property access)
	response = p.golem.processBotTagsWithContext(response, ctx)

	// Process think tags FIRST (internal processing, no output)
	// This allows local variables to be set before variable replacement
	response = p.golem.processThinkTagsWithContext(response, ctx)

	// Process topic setting tags first (special handling for topic)
	response = p.golem.processTopicSettingTagsWithContext(response, ctx)

	// Process set tags (before session variable replacement)
	response = p.golem.processSetTagsWithContext(response, ctx)

	// Replace session variable tags using context
	response = p.golem.replaceSessionVariableTagsWithContext(response, ctx)

	// Process condition tags
	response = p.golem.processConditionTagsWithContext(response, ctx)

	return response, nil
}
func (p *ComprehensiveVariableProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	// Check if template contains any variable-related tags
	variableTags := []string{
		"<get", "</get>",
		"<set", "</set>",
		"<bot", "</bot>",
		"<think", "</think>",
		"<topic", "</topic>",
		"<name", "</name>",
		"<value", "</value>",
		"<condition", "</condition>",
	}

	for _, tag := range variableTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
func (p *ComprehensiveVariableProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveVariableProcessor) ResetMetrics()                 {}

// ComprehensiveRecursiveProcessor handles recursive processing (SR, SRAI, SRAIX, learn, unlearn tags)
type ComprehensiveRecursiveProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveRecursiveProcessor) Name() string                { return "recursive" }
func (p *ComprehensiveRecursiveProcessor) Type() ProcessorType         { return ProcessorTypeRecursive }
func (p *ComprehensiveRecursiveProcessor) Priority() ProcessorPriority { return PriorityNormal }
func (p *ComprehensiveRecursiveProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{RequiresContext: true, RequiresKB: true, SkipIfEmpty: true}
}
func (p *ComprehensiveRecursiveProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template

	// Process SR tags (shorthand for <srai><star/></srai>) AFTER wildcard replacement
	response = p.golem.processSRTagsWithContext(response, wildcards, ctx)

	// Process SRAI tags (recursive)
	response = p.golem.processSRAITagsWithContext(response, ctx)

	// Process SRAIX tags (external services)
	response = p.golem.processSRAIXTagsWithContext(response, ctx)

	// Process learn tags (dynamic learning)
	response = p.golem.processLearnTagsWithContext(response, ctx)

	// Process unlearn tags (remove learned categories)
	response = p.golem.processUnlearnTagsWithContext(response, ctx)

	return response, nil
}
func (p *ComprehensiveRecursiveProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	// Check if template contains any recursive tags
	recursiveTags := []string{
		"<srai", "</srai>",
		"<sraix", "</sraix>",
		"<sr", "</sr>",
		"<learn", "</learn>",
		"<unlearn", "</unlearn>",
	}

	for _, tag := range recursiveTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
func (p *ComprehensiveRecursiveProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveRecursiveProcessor) ResetMetrics()                 {}

// ComprehensiveTextProcessor handles text processing (person, gender, sentence, word tags)
type ComprehensiveTextProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveTextProcessor) Name() string                { return "text" }
func (p *ComprehensiveTextProcessor) Type() ProcessorType         { return ProcessorTypeText }
func (p *ComprehensiveTextProcessor) Priority() ProcessorPriority { return PriorityNormal }
func (p *ComprehensiveTextProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{SkipIfEmpty: true}
}
func (p *ComprehensiveTextProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		iteration++
		previousResponse := response

		// Process text processing tags (process from inside out for nested tags)
		// First pass: process innermost tags
		response = p.golem.processPerson2TagsWithContext(response, ctx)
		response = p.golem.processGenderTagsWithContext(response, ctx)
		response = p.golem.processSentenceTagsWithContext(response, ctx)
		response = p.golem.processWordTagsWithContext(response, ctx)

		// Second pass: process outermost tags
		response = p.golem.processPersonTagsWithContext(response, ctx)

		// If no changes occurred, we're done
		if response == previousResponse {
			break
		}
	}

	if iteration >= maxIterations {
		p.golem.LogWarn("Text processor reached maximum iterations (%d), stopping recursion", maxIterations)
	}

	return response, nil
}
func (p *ComprehensiveTextProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	// Check if template contains any text processing tags
	textTags := []string{
		"<person", "</person>",
		"<gender", "</gender>",
		"<person2", "</person2>",
		"<sentence", "</sentence>",
		"<word", "</word>",
	}

	for _, tag := range textTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
func (p *ComprehensiveTextProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveTextProcessor) ResetMetrics()                 {}

// ComprehensiveDataProcessor handles data processing (date, time, random tags)
type ComprehensiveDataProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveDataProcessor) Name() string                { return "data" }
func (p *ComprehensiveDataProcessor) Type() ProcessorType         { return ProcessorTypeData }
func (p *ComprehensiveDataProcessor) Priority() ProcessorPriority { return PriorityNormal }
func (p *ComprehensiveDataProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{SkipIfEmpty: true}
}
func (p *ComprehensiveDataProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template
	maxIterations := 10
	iteration := 0

	// Process tags iteratively to handle nested tags
	for iteration < maxIterations {
		iteration++
		originalResponse := response

		// Process date and time tags
		response = p.golem.processDateTimeTags(response)

		// Process random tags
		response = p.golem.processRandomTags(response)

		// Process first tags
		response = p.processFirstTags(response, ctx)

		// Process rest tags
		response = p.processRestTags(response, ctx)

		// Process loop tags
		response = p.processLoopTags(response, ctx)

		// Process input tags
		response = p.processInputTags(response, ctx)

		// Process eval tags
		response = p.processEvalTags(response, ctx)

		// Process RDF-style tags (process individual tags first, then uniq)
		response = p.processSubjTags(response, ctx)
		response = p.processPredTags(response, ctx)
		response = p.processObjTags(response, ctx)
		response = p.processUniqTags(response, ctx)

		// If no changes were made, we're done
		if response == originalResponse {
			break
		}
	}

	return response, nil
}
func (p *ComprehensiveDataProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	// Check if template contains any data tags
	dataTags := []string{
		"<date", "</date>",
		"<time", "</time>",
		"<random", "</random>",
		"<li", "</li>",
		"<first", "</first>",
		"<rest", "</rest>",
		"<loop",
		"<input",
		"<eval", "</eval>",
		"<uniq", "</uniq>",
		"<subj", "</subj>",
		"<pred", "</pred>",
		"<obj", "</obj>",
	}

	for _, tag := range dataTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
func (p *ComprehensiveDataProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveDataProcessor) ResetMetrics()                 {}

// ComprehensiveFormatProcessor handles text formatting (uppercase, lowercase, formal, etc.)
type ComprehensiveFormatProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveFormatProcessor) Name() string                { return "format" }
func (p *ComprehensiveFormatProcessor) Type() ProcessorType         { return ProcessorTypeFormat }
func (p *ComprehensiveFormatProcessor) Priority() ProcessorPriority { return PriorityLate }
func (p *ComprehensiveFormatProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{SkipIfEmpty: true}
}
func (p *ComprehensiveFormatProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		iteration++
		previousResponse := response

		// Process topic tags (current topic references) - before text processing
		response = p.golem.processTopicTagsWithContext(response, ctx)

		// Process repeat tags first (before text formatting) so they can be processed by other tags
		response = p.golem.processRepeatTagsWithContext(response, ctx)

		// Process all text formatting tags (process from inside out for nested tags)
		// First pass: process innermost tags (order chosen to produce expected semantics)
		// 1) explode first so tokens are created before any casing
		response = p.golem.processExplodeTagsWithContext(response, ctx)
		// 2) substring/replace before case changes to act on original text
		response = p.golem.processSubstringTagsWithContext(response, ctx)
		response = p.golem.processReplaceTagsWithContext(response, ctx)
		// 3) (defer capitalization to second pass to combine with case transforms)
		response = p.golem.processReverseTagsWithContext(response, ctx)
		response = p.golem.processAcronymTagsWithContext(response, ctx)
		response = p.golem.processTrimTagsWithContext(response, ctx)
		response = p.golem.processIndentTagsWithContext(response, ctx)
		response = p.golem.processDedentTagsWithContext(response, ctx)
		response = p.golem.processPluralizeTagsWithContext(response, ctx)
		response = p.golem.processShuffleTagsWithContext(response, ctx)
		response = p.golem.processCountTagsWithContext(response, ctx)
		response = p.golem.processSplitTagsWithContext(response, ctx)
		response = p.golem.processJoinTagsWithContext(response, ctx)
		response = p.golem.processLengthTagsWithContext(response, ctx)
		response = p.golem.processUniqueTagsWithContext(response, ctx)

		// Second pass: process outermost tags (case transforms) then formal last to finalize title casing
		response = p.golem.processCapitalizeTagsWithContext(response, ctx)
		response = p.golem.processFormalTagsWithContext(response, ctx)
		response = p.golem.processUppercaseTagsWithContext(response, ctx)
		response = p.golem.processLowercaseTagsWithContext(response, ctx)

		// Process normalize tags (text normalization)
		response = p.golem.processNormalizeTagsWithContext(response, ctx)

		// Process denormalize tags (text denormalization)
		response = p.golem.processDenormalizeTagsWithContext(response, ctx)

		// If no changes occurred, we're done
		if response == previousResponse {
			break
		}
	}

	if iteration >= maxIterations {
		p.golem.LogWarn("Format processor reached maximum iterations (%d), stopping recursion", maxIterations)
	}

	return response, nil
}
func (p *ComprehensiveFormatProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
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
func (p *ComprehensiveFormatProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveFormatProcessor) ResetMetrics()                 {}

// ComprehensiveSystemProcessor handles system processing (size, version, id, that, request, response tags)
type ComprehensiveSystemProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveSystemProcessor) Name() string                { return "system" }
func (p *ComprehensiveSystemProcessor) Type() ProcessorType         { return ProcessorTypeSystem }
func (p *ComprehensiveSystemProcessor) Priority() ProcessorPriority { return PriorityLate }
func (p *ComprehensiveSystemProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{SkipIfEmpty: true}
}
func (p *ComprehensiveSystemProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template

	// Process system tags (size, version, id, that, etc.)
	response = p.golem.processSizeTagsWithContext(response, ctx)
	response = p.golem.processVersionTagsWithContext(response, ctx)
	response = p.golem.processIdTagsWithContext(response, ctx)
	response = p.golem.processThatTagsWithContext(response, ctx)
	response = p.golem.processRequestTags(response, ctx)
	response = p.golem.processResponseTags(response, ctx)

	return response, nil
}
func (p *ComprehensiveSystemProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	// Check if template contains any system tags
	systemTags := []string{
		"<size", "</size>",
		"<version", "</version>",
		"<id", "</id>",
		"<that", "</that>",
		"<request", "</request>",
		"<response", "</response>",
	}

	for _, tag := range systemTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
func (p *ComprehensiveSystemProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveSystemProcessor) ResetMetrics()                 {}

// ComprehensiveCollectionProcessor handles collection processing (map, list, array tags)
type ComprehensiveCollectionProcessor struct {
	*BaseProcessor
	golem *Golem
}

func (p *ComprehensiveCollectionProcessor) Name() string                { return "collection" }
func (p *ComprehensiveCollectionProcessor) Type() ProcessorType         { return ProcessorTypeCollection }
func (p *ComprehensiveCollectionProcessor) Priority() ProcessorPriority { return PriorityNormal }
func (p *ComprehensiveCollectionProcessor) Condition() ProcessorCondition {
	return ProcessorCondition{SkipIfEmpty: true}
}
func (p *ComprehensiveCollectionProcessor) Process(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	response := template
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		iteration++
		previousResponse := response

		// Process map tags
		response = p.golem.processMapTagsWithContext(response, ctx)

		// Process list tags
		response = p.golem.processListTagsWithContext(response, ctx)

		// Process array tags
		response = p.golem.processArrayTagsWithContext(response, ctx)

		// If no changes occurred, we're done
		if response == previousResponse {
			break
		}
	}

	if iteration >= maxIterations {
		p.golem.LogWarn("Collection processor reached maximum iterations (%d), stopping recursion", maxIterations)
	}

	return response, nil
}
func (p *ComprehensiveCollectionProcessor) ShouldProcess(template string, ctx *VariableContext) bool {
	// Check if template contains any collection tags
	collectionTags := []string{
		"<map", "</map>",
		"<list", "</list>",
		"<array", "</array>",
	}

	for _, tag := range collectionTags {
		if strings.Contains(template, tag) {
			return true
		}
	}

	return false
}
func (p *ComprehensiveCollectionProcessor) GetMetrics() *ProcessorMetrics { return &ProcessorMetrics{} }
func (p *ComprehensiveCollectionProcessor) ResetMetrics()                 {}

// processFirstTags processes <first> tags to get the first element of a list
func (p *ComprehensiveDataProcessor) processFirstTags(template string, ctx *VariableContext) string {
	// Check for malformed tags first - these should be left unchanged
	// Pattern: <first><rest>content</first></rest> (malformed nesting - no nested tags in content)
	malformedPattern := regexp.MustCompile(`<first><rest>[^<]*</first></rest>`)
	if malformedPattern.MatchString(template) {
		// This contains malformed tags, leave them unchanged
		return template
	}

	// Find <first> tags with proper nesting support
	firstTags := p.findNestedTags(template, "first")

	for _, tag := range firstTags {
		content := strings.TrimSpace(tag.Content)
		if content == "" {
			template = strings.ReplaceAll(template, tag.Full, "")
			continue
		}

		// Process the content through the full template pipeline
		processedContent := p.golem.processTemplateWithContext(content, map[string]string{}, ctx)
		processedContent = strings.TrimSpace(processedContent)

		// Split by spaces to get list elements
		elements := strings.Fields(processedContent)
		if len(elements) == 0 {
			template = strings.ReplaceAll(template, tag.Full, "")
		} else {
			// Return the first element
			template = strings.ReplaceAll(template, tag.Full, elements[0])
		}
	}

	return template
}

// processRestTags processes <rest> tags to get all elements except the first from a list
func (p *ComprehensiveDataProcessor) processRestTags(template string, ctx *VariableContext) string {
	// Check for malformed tags first - these should be left unchanged
	// Pattern: <first><rest>content</first></rest> (malformed nesting - no nested tags in content)
	malformedPattern := regexp.MustCompile(`<first><rest>[^<]*</first></rest>`)
	if malformedPattern.MatchString(template) {
		// This contains malformed tags, leave them unchanged
		return template
	}

	// Process <rest> tags iteratively to handle nested tags
	maxIterations := 10
	iteration := 0

	for iteration < maxIterations {
		iteration++
		originalTemplate := template

		// Find <rest> tags with proper nesting support
		restTags := p.findNestedTags(template, "rest")

		if len(restTags) == 0 {
			// No more <rest> tags found
			break
		}

		for _, tag := range restTags {
			content := strings.TrimSpace(tag.Content)
			if content == "" {
				template = strings.ReplaceAll(template, tag.Full, "")
				continue
			}

			// Process the content through the full template pipeline
			processedContent := p.golem.processTemplateWithContext(content, map[string]string{}, ctx)
			processedContent = strings.TrimSpace(processedContent)

			// Split by spaces to get list elements
			elements := strings.Fields(processedContent)
			if len(elements) <= 1 {
				template = strings.ReplaceAll(template, tag.Full, "")
			} else {
				// Return all elements except the first
				rest := strings.Join(elements[1:], " ")
				template = strings.ReplaceAll(template, tag.Full, rest)
			}
		}

		// If no changes were made, we're done
		if template == originalTemplate {
			break
		}
	}

	return template
}

// NestedTag represents a tag with its content and full match
type NestedTag struct {
	Full    string // The complete tag including <tag>content</tag>
	Content string // Just the content between the tags
}

// findNestedTags finds tags with proper nesting support
func (p *ComprehensiveDataProcessor) findNestedTags(template, tagName string) []NestedTag {
	var tags []NestedTag
	startTag := "<" + tagName + ">"
	endTag := "</" + tagName + ">"

	start := 0
	for {
		// Find the next start tag
		startPos := strings.Index(template[start:], startTag)
		if startPos == -1 {
			break
		}
		startPos += start

		// Find the matching end tag by counting nested tags
		depth := 1
		pos := startPos + len(startTag)
		endPos := -1

		for pos < len(template) && depth > 0 {
			nextStart := strings.Index(template[pos:], startTag)
			nextEnd := strings.Index(template[pos:], endTag)

			if nextEnd == -1 {
				// No closing tag found, this is malformed
				break
			}

			if nextStart != -1 && nextStart < nextEnd {
				// Found a nested start tag
				depth++
				pos += nextStart + len(startTag)
			} else {
				// Found an end tag
				depth--
				if depth == 0 {
					endPos = pos + nextEnd
				}
				pos += nextEnd + len(endTag)
			}
		}

		// Check if we found a proper closing tag
		if endPos == -1 {
			// This is malformed, skip it
			start = startPos + len(startTag)
			continue
		}

		if endPos != -1 {
			// Extract the content
			content := template[startPos+len(startTag) : endPos]
			full := template[startPos : endPos+len(endTag)]
			tags = append(tags, NestedTag{Full: full, Content: content})
			start = endPos + len(endTag)
		} else {
			// Malformed tag, include it as-is for edge case handling
			// Find the next start tag or end of string
			nextStart := strings.Index(template[startPos+len(startTag):], startTag)
			nextEnd := strings.Index(template[startPos+len(startTag):], endTag)

			var endPos int
			if nextStart == -1 && nextEnd == -1 {
				// No more tags found, take to end of string
				endPos = len(template)
			} else if nextStart == -1 || (nextEnd != -1 && nextEnd < nextStart) {
				// Found end tag first
				endPos = startPos + len(startTag) + nextEnd + len(endTag)
			} else {
				// Found start tag first, take up to that point
				endPos = startPos + len(startTag) + nextStart
			}

			content := template[startPos:endPos]
			tags = append(tags, NestedTag{Full: content, Content: content})
			start = endPos
		}
	}

	return tags
}

// processLoopTags processes <loop/> tags for loop control
func (p *ComprehensiveDataProcessor) processLoopTags(template string, _ *VariableContext) string {
	// Find all <loop/> tags (self-closing)
	loopRegex := regexp.MustCompile(`<loop\s*/>`)
	matches := loopRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		// For now, <loop/> tags are simply removed as they are used for control flow
		// In a more sophisticated implementation, they could be used to control
		// iteration in conditionals or other loop constructs
		template = strings.ReplaceAll(template, match[0], "")
	}

	return template
}

// processInputTags processes <input/> tags to reference the current user input
func (p *ComprehensiveDataProcessor) processInputTags(template string, ctx *VariableContext) string {
	// Find all <input/> tags (self-closing)
	inputRegex := regexp.MustCompile(`<input\s*/>`)
	matches := inputRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		// Get the current user input from the session's request history
		var currentInput string
		if ctx.Session != nil && len(ctx.Session.RequestHistory) > 0 {
			// Get the most recent user input (last item in RequestHistory)
			currentInput = ctx.Session.RequestHistory[len(ctx.Session.RequestHistory)-1]
		}

		// Replace the <input/> tag with the current user input
		template = strings.ReplaceAll(template, match[0], currentInput)
	}

	return template
}

// processEvalTags processes <eval> tags to evaluate content as AIML code
func (p *ComprehensiveDataProcessor) processEvalTags(template string, ctx *VariableContext) string {
	// Find all <eval> tags
	evalRegex := regexp.MustCompile(`<eval>(.*?)</eval>`)
	matches := evalRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		content := strings.TrimSpace(match[1])
		if content == "" {
			template = strings.ReplaceAll(template, match[0], "")
			continue
		}

		// Process the content through the full template pipeline
		// This ensures any variables or other tags are resolved
		processedContent := p.golem.processTemplateWithContext(content, map[string]string{}, ctx)
		processedContent = strings.TrimSpace(processedContent)

		// Replace the entire <eval> tag with the processed content
		template = strings.ReplaceAll(template, match[0], processedContent)
	}

	return template
}

// processUniqTags processes <uniq> tags for RDF-like predicate relationships
func (p *ComprehensiveDataProcessor) processUniqTags(template string, ctx *VariableContext) string {
	// Find all <uniq> tags (with optional attributes)
	uniqRegex := regexp.MustCompile(`<uniq[^>]*>(.*?)</uniq>`)
	matches := uniqRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		content := strings.TrimSpace(match[1])
		if content == "" {
			template = strings.ReplaceAll(template, match[0], "")
			continue
		}

		// Process the content through the full template pipeline
		// This ensures any subj/pred/obj tags are resolved
		processedContent := p.golem.processTemplateWithContext(content, map[string]string{}, ctx)
		processedContent = strings.TrimSpace(processedContent)

		// Add proper spacing between RDF elements for human readability
		// Split by common RDF patterns and add spaces
		processedContent = p.formatRDFContent(processedContent)

		template = strings.ReplaceAll(template, match[0], processedContent)
	}

	return template
}

// formatRDFContent formats RDF content with proper spacing for human readability
func (p *ComprehensiveDataProcessor) formatRDFContent(content string) string {
	// If content is empty, return as-is
	if strings.TrimSpace(content) == "" {
		return content
	}

	// Clean up multiple spaces and trim
	content = strings.TrimSpace(content)

	// Split content into words and join with single spaces
	words := strings.Fields(content)
	if len(words) == 0 {
		return content
	}

	// Join words with single spaces for human readability
	return strings.Join(words, " ")
}

// processSubjTags processes <subj> tags for subject of RDF triples
func (p *ComprehensiveDataProcessor) processSubjTags(template string, ctx *VariableContext) string {
	// Find all <subj> tags
	subjRegex := regexp.MustCompile(`<subj>(.*?)</subj>`)
	matches := subjRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		content := strings.TrimSpace(match[1])
		if content == "" {
			template = strings.ReplaceAll(template, match[0], "")
			continue
		}

		// Process the content through the full template pipeline
		processedContent := p.golem.processTemplateWithContext(content, map[string]string{}, ctx)
		processedContent = strings.TrimSpace(processedContent)

		// Add trailing space for RDF readability (will be trimmed if not needed)
		processedContent = processedContent + " "

		// Return the processed content as the subject
		template = strings.ReplaceAll(template, match[0], processedContent)
	}

	return template
}

// processPredTags processes <pred> tags for predicate of RDF triples
func (p *ComprehensiveDataProcessor) processPredTags(template string, ctx *VariableContext) string {
	// Find all <pred> tags
	predRegex := regexp.MustCompile(`<pred>(.*?)</pred>`)
	matches := predRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		content := strings.TrimSpace(match[1])
		if content == "" {
			template = strings.ReplaceAll(template, match[0], "")
			continue
		}

		// Process the content through the full template pipeline
		processedContent := p.golem.processTemplateWithContext(content, map[string]string{}, ctx)
		processedContent = strings.TrimSpace(processedContent)

		// Add trailing space for RDF readability (will be trimmed if not needed)
		processedContent = processedContent + " "

		// Return the processed content as the predicate
		template = strings.ReplaceAll(template, match[0], processedContent)
	}

	return template
}

// processObjTags processes <obj> tags for object of RDF triples
func (p *ComprehensiveDataProcessor) processObjTags(template string, ctx *VariableContext) string {
	// Find all <obj> tags
	objRegex := regexp.MustCompile(`<obj>(.*?)</obj>`)
	matches := objRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		content := strings.TrimSpace(match[1])
		if content == "" {
			template = strings.ReplaceAll(template, match[0], "")
			continue
		}

		// Process the content through the full template pipeline
		processedContent := p.golem.processTemplateWithContext(content, map[string]string{}, ctx)
		processedContent = strings.TrimSpace(processedContent)

		// Don't add trailing space for object (it's the last element)
		// Return the processed content as the object
		template = strings.ReplaceAll(template, match[0], processedContent)
	}

	return template
}
