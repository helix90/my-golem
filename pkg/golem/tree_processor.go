package golem

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TreeProcessor handles processing of AST nodes for AIML tag processing
type TreeProcessor struct {
	golem       *Golem
	ctx         *VariableContext
	starCounter int                // Tracks auto-incrementing star index for <star/> tags without explicit index
	metrics     *ProcessorRegistry // Tracks metrics for different tag types/operations
	loop        bool               // Set by <loop/> inside the <li> a condition is processing
	// kbReadLocked is set when the caller holds golem.kbMu for reading, so
	// that <sraix> can release it for the length of its request
	kbReadLocked bool
	// reauthorizing is set while a <sraix> answers its service's reauth
	// input, so that a 401 there does not start another
	reauthorizing bool
}

// NewTreeProcessor creates a new tree processor
func NewTreeProcessor(golem *Golem) *TreeProcessor {
	// Count calls to the logical operations the tree processor performs
	metrics := NewProcessorRegistry("wildcard", "variable", "data", "logic", "format")

	return &TreeProcessor{
		golem:   golem,
		metrics: metrics,
	}
}

// trackMetric tracks metrics for a specific processor type
func (tp *TreeProcessor) trackMetric(processorName string) {
	if tp.metrics != nil {
		tp.metrics.recordCall(processorName)
	}
}

// ProcessTemplate processes a template using tree-based approach
func (tp *TreeProcessor) ProcessTemplate(template string, wildcards map[string]string, ctx *VariableContext) (string, error) {
	// Nested calls (e.g. <srai>, <sr/>) reuse this processor, so restore the
	// caller's context (and its recursion depth) when this call returns
	oldCtx := tp.ctx
	defer func() {
		tp.ctx = oldCtx
	}()

	// Reset star counter for auto-incrementing <star/> tags
	tp.starCounter = 0

	// Track that wildcard processing might occur if wildcards are present
	if len(wildcards) > 0 {
		tp.trackMetric("wildcard")
	}

	// Parse template into AST
	parser := NewASTParser(template)
	ast, err := parser.Parse()
	if err != nil {
		return template, err
	}

	// Store wildcards in context for <star/>, <thatstar/> and learn tag processing
	if ctx != nil {
		// Save current wildcards to restore them later
		oldWildcards := ctx.Wildcards
		ctx.Wildcards = wildcards

		defer func() {
			ctx.Wildcards = oldWildcards
		}()
	}

	// Process the AST
	tp.ctx = ctx
	result := tp.processNode(ast)

	// Smart whitespace trimming:
	// 1. Always trim trailing whitespace
	// 2. Trim leading whitespace, but preserve leading spaces/tabs if there's no leading newline
	//    (this allows indent tags to work while removing template formatting whitespace)
	result = strings.TrimRight(result, " \t\n\r")

	// If result starts with newline, trim all leading whitespace (template formatting)
	// Otherwise, just trim leading newlines but keep leading spaces/tabs (indent tag output)
	if len(result) > 0 && (result[0] == '\n' || result[0] == '\r') {
		result = strings.TrimLeft(result, " \t\n\r")
	} else {
		result = strings.TrimLeft(result, "\n\r")
	}

	// Return result as-is
	// XML entities were already decoded during parsing, and we don't re-encode them
	// to avoid double-escaping issues
	return result, nil
}

// processNode processes a single AST node
func (tp *TreeProcessor) processNode(node *ASTNode) string {
	switch node.Type {
	case NodeTypeText:
		// If this is a text node with children, process children
		if len(node.Children) > 0 {
			return tp.processChildren(node)
		}
		// Decode entities (&amp;, &lt;, ...) on output. Content itself keeps
		// the source text so that re-serialized templates (e.g. in <learn>)
		// still parse correctly.
		return html.UnescapeString(node.Content)
	case NodeTypeComment:
		return "" // Comments are not output
	case NodeTypeCDATA:
		return node.Content // CDATA is output as-is
	case NodeTypeSelfClosingTag:
		return tp.processSelfClosingTag(node)
	case NodeTypeTag:
		return tp.processTag(node)
	default:
		return ""
	}
}

// attributeChildTags lists, per tag, the attributes that AIML 2.0 also
// allows as child elements, e.g. <get><name>color</name></get>
var attributeChildTags = map[string]map[string]bool{
	"get":       {"name": true, "var": true},
	"set":       {"name": true, "var": true},
	"bot":       {"name": true},
	"map":       {"name": true},
	"condition": {"name": true, "var": true, "value": true},
	"li":        {"name": true, "var": true, "value": true},
	"date":      {"format": true, "jformat": true, "locale": true, "timezone": true},
	"sraix":     {"bot": true, "botid": true, "host": true, "default": true, "hint": true, "service": true},
	"star":      {"index": true},
	"that":      {"index": true},
	"input":     {"index": true},
	"request":   {"index": true},
	"response":  {"index": true},
}

// liftAttributeChildren returns the node with any attribute child elements
// (see attributeChildTags) evaluated and moved into its attributes. The node
// itself is not modified; if it has no such children it is returned as is.
func (tp *TreeProcessor) liftAttributeChildren(node *ASTNode) *ASTNode {
	names := attributeChildTags[node.TagName]
	if names == nil {
		return node
	}

	var lifted *ASTNode
	for i, child := range node.Children {
		isAttribute := child.Type == NodeTypeTag && names[child.TagName]
		if isAttribute && lifted == nil {
			copied := *node
			lifted = &copied
			lifted.Attributes = make(map[string]string, len(node.Attributes)+1)
			for k, v := range node.Attributes {
				lifted.Attributes[k] = v
			}
			lifted.Children = append([]*ASTNode(nil), node.Children[:i]...)
		}
		if isAttribute {
			var value strings.Builder
			for _, valueChild := range child.Children {
				value.WriteString(tp.processNode(valueChild))
			}
			lifted.Attributes[child.TagName] = strings.TrimSpace(value.String())
		} else if lifted != nil {
			lifted.Children = append(lifted.Children, child)
		}
	}
	if lifted == nil {
		return node
	}
	return lifted
}

// processTag processes a tag node
func (tp *TreeProcessor) processTag(node *ASTNode) string {
	node = tp.liftAttributeChildren(node)

	// Some tags need to process their children selectively (not all at once)
	// For those tags, skip pre-processing children
	skipChildProcessing := false
	switch node.TagName {
	case "random", "condition", "learn", "learnf":
		skipChildProcessing = true
	}

	// Process children first to handle nested tags (unless tag handles its own children)
	var content string
	if !skipChildProcessing {
		processedChildren := make([]string, len(node.Children))
		for i, child := range node.Children {
			processedChildren[i] = tp.processNode(child)
		}
		// Join processed children
		content = strings.Join(processedChildren, "")
	}

	// Process the tag based on its name
	// Check for that wildcard tags with embedded index (e.g., that_star1, that_underscore2)
	if strings.HasPrefix(node.TagName, "that_star") && len(node.TagName) > 9 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_star")
	}
	if strings.HasPrefix(node.TagName, "that_underscore") && len(node.TagName) > 15 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_underscore")
	}
	if strings.HasPrefix(node.TagName, "that_caret") && len(node.TagName) > 10 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_caret")
	}
	if strings.HasPrefix(node.TagName, "that_hash") && len(node.TagName) > 9 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_hash")
	}
	if strings.HasPrefix(node.TagName, "that_dollar") && len(node.TagName) > 11 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_dollar")
	}
	if strings.HasPrefix(node.TagName, "thatstar") && len(node.TagName) > 8 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_star")
	}

	switch node.TagName {
	case "srai":
		return tp.processSRAITag(node, content)
	case "sraix":
		return tp.processSRAIXTag(node, content)
	case "think":
		return tp.processThinkTag(node, content)
	case "set":
		return tp.processSetTag(node, content)
	case "get":
		return tp.processGetTag(node, content)
	case "bot":
		return tp.processBotTag(node, content)
	case "star":
		return tp.processStarTag(node, content)
	case "sr":
		return tp.processSRTag(node, content)
	case "that":
		return tp.processThatTag(node, content)
	case "that_star":
		return tp.processThatStarTag(node, content)
	case "that_underscore":
		return tp.processThatUnderscoreTag(node, content)
	case "that_caret":
		return tp.processThatCaretTag(node, content)
	case "that_hash":
		return tp.processThatHashTag(node, content)
	case "that_dollar":
		return tp.processThatDollarTag(node, content)
	case "thatstar":
		return tp.processThatStarTag(node, content)
	case "topicstar":
		return tp.processTopicStarTag(node, content)
	case "topic":
		return tp.processTopicTag(node, content)
	case "random":
		return tp.processRandomTag(node, content)
	case "li":
		return tp.processListItemTag(node, content)
	case "condition":
		return tp.processConditionTag(node, content)
	case "map":
		return tp.processMapTag(node, content)
	case "list":
		return tp.processListTag(node, content)
	case "array":
		return tp.processArrayTag(node, content)
	case "learn":
		return tp.processLearnTag(node, content)
	case "learnf":
		return tp.processLearnfTag(node, content)
	case "uppercase":
		return tp.processUppercaseTag(node, content)
	case "lowercase":
		return tp.processLowercaseTag(node, content)
	case "formal":
		return tp.processFormalTag(node, content)
	case "capitalize":
		return tp.processCapitalizeTag(node, content)
	case "explode":
		return tp.processExplodeTag(node, content)
	case "reverse":
		return tp.processReverseTag(node, content)
	case "acronym":
		return tp.processAcronymTag(node, content)
	case "trim":
		return tp.processTrimTag(node, content)
	case "substring":
		return tp.processSubstringTag(node, content)
	case "replace":
		return tp.processReplaceTag(node, content)
	case "pluralize":
		return tp.processPluralizeTag(node, content)
	case "shuffle":
		return tp.processShuffleTag(node, content)
	case "length":
		return tp.processLengthTag(node, content)
	case "count":
		return tp.processCountTag(node, content)
	case "split":
		return tp.processSplitTag(node, content)
	case "join":
		return tp.processJoinTag(node, content)
	case "unique":
		return tp.processUniqueTag(node, content)
	case "indent":
		return tp.processIndentTag(node, content)
	case "dedent":
		return tp.processDedentTag(node, content)
	case "repeat":
		return tp.processRepeatTag(node, content)
	case "first":
		return tp.processFirstTag(node, content)
	case "rest":
		return tp.processRestTag(node, content)
	case "loop":
		return tp.processLoopTag(node, content)
	case "input":
		return tp.processInputTag(node, content)
	case "eval":
		return tp.processEvalTag(node, content)
	case "person":
		return tp.processPersonTag(node, content)
	case "person2":
		return tp.processPerson2Tag(node, content)
	case "gender":
		return tp.processGenderTag(node, content)
	case "sentence":
		return tp.processSentenceTag(node, content)
	case "word":
		return tp.processWordTag(node, content)
	case "date":
		return tp.processDateTag(node, content)
	case "time":
		return tp.processTimeTag(node, content)
	case "subj":
		return tp.processSubjTag(node, content)
	case "pred":
		return tp.processPredTag(node, content)
	case "obj":
		return tp.processObjTag(node, content)
	case "uniq":
		return tp.processUniqTag(node, content)
	case "size":
		return tp.processSizeTag(node, content)
	case "version":
		return tp.processVersionTag(node, content)
	case "id":
		return tp.processIdTag(node, content)
	case "botid":
		return tp.processBotIdTag(node, content)
	case "request":
		return tp.processRequestTag(node, content)
	case "response":
		return tp.processResponseTag(node, content)
	case "normalize":
		return tp.processNormalizeTag(node, content)
	case "denormalize":
		return tp.processDenormalizeTag(node, content)
	case "unlearn":
		return tp.processUnlearnTag(node, content)
	case "unlearnf":
		return tp.processUnlearnfTag(node, content)
	case "var":
		return tp.processVarTag(node, content)
	case "gossip":
		return tp.processGossipTag(node, content)
	case "javascript":
		return tp.processJavascriptTag(node, content)
	case "system":
		return tp.processSystemTag(node, content)
	case "jsonformat":
		return tp.processJsonFormatTag(node, content)
	case "weatherformat":
		return tp.processWeatherFormatTag(node, content)
	default:
		// Unknown tag (such as HTML), return as-is with its attributes and processed content
		return fmt.Sprintf("<%s%s>%s</%s>", node.TagName, formatAttributes(node.Attributes), content, node.TagName)
	}
}

// processSelfClosingTag processes self-closing tags
func (tp *TreeProcessor) processSelfClosingTag(node *ASTNode) string {
	// Check for that wildcard tags with embedded index
	if strings.HasPrefix(node.TagName, "that_star") && len(node.TagName) > 9 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_star")
	}
	if strings.HasPrefix(node.TagName, "that_underscore") && len(node.TagName) > 15 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_underscore")
	}
	if strings.HasPrefix(node.TagName, "that_caret") && len(node.TagName) > 10 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_caret")
	}
	if strings.HasPrefix(node.TagName, "that_hash") && len(node.TagName) > 9 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_hash")
	}
	if strings.HasPrefix(node.TagName, "that_dollar") && len(node.TagName) > 11 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_dollar")
	}
	if strings.HasPrefix(node.TagName, "thatstar") && len(node.TagName) > 8 {
		return tp.processThatWildcardWithEmbeddedIndex(node, "that_star")
	}

	switch node.TagName {
	case "star":
		return tp.processStarTag(node, "")
	case "sr":
		return tp.processSRTag(node, "")
	case "input":
		return tp.processInputTag(node, "")
	case "loop":
		return tp.processLoopTag(node, "")
	case "date":
		return tp.processDateTag(node, "")
	case "time":
		return tp.processTimeTag(node, "")
	case "subj":
		return tp.processSubjTag(node, "")
	case "pred":
		return tp.processPredTag(node, "")
	case "obj":
		return tp.processObjTag(node, "")
	case "uniq":
		return tp.processUniqTag(node, "")
	case "size":
		return tp.processSizeTag(node, "")
	case "version":
		return tp.processVersionTag(node, "")
	case "id":
		return tp.processIdTag(node, "")
	case "botid":
		return tp.processBotIdTag(node, "")
	case "request":
		return tp.processRequestTag(node, "")
	case "response":
		return tp.processResponseTag(node, "")
	case "get":
		return tp.processGetTag(node, "")
	case "that":
		return tp.processThatTag(node, "")
	case "that_star":
		return tp.processThatStarTag(node, "")
	case "that_underscore":
		return tp.processThatUnderscoreTag(node, "")
	case "that_caret":
		return tp.processThatCaretTag(node, "")
	case "that_hash":
		return tp.processThatHashTag(node, "")
	case "that_dollar":
		return tp.processThatDollarTag(node, "")
	case "thatstar":
		return tp.processThatStarTag(node, "")
	case "topicstar":
		return tp.processTopicStarTag(node, "")
	case "bot":
		return tp.processBotTag(node, "")
	case "repeat":
		return tp.processRepeatTag(node, "")
	case "topic":
		return tp.processTopicTag(node, "")
	default:
		// Unknown self-closing tag, return as-is
		return fmt.Sprintf("<%s%s/>", node.TagName, formatAttributes(node.Attributes))
	}
}

// Tag processing methods

// matchSRAI matches srai input the same way ProcessInput matches user input:
// against the session's current topic and the bot's last response (<that>).
// The topic is read at this point so that a <set name="topic"> earlier in the
// same template applies.
func (tp *TreeProcessor) matchSRAI(kb *AIMLKnowledgeBase, input string) (*Category, map[string]string, string, error) {
	topic := tp.ctx.Topic
	that := ""
	if tp.ctx.Session != nil {
		topic = tp.ctx.Session.GetSessionTopic()
		if lastThat := tp.ctx.Session.GetLastThat(); lastThat != "" {
			that = tp.golem.normalizeThatContext(lastThat)
		}
	}
	category, wildcards, err := kb.MatchPatternWithTopicAndThatIndexOriginalCached(tp.golem, tp.golem.CachedNormalizePattern(input), input, topic, that, 0)
	return category, wildcards, topic, err
}

func (tp *TreeProcessor) processSRAITag(_ *ASTNode, content string) string {
	// Process SRAI tag - recursive AIML processing (Symbolic Reduction and Inference)
	// Check recursion depth to prevent infinite recursion
	if tp.ctx == nil || tp.ctx.RecursionDepth >= MaxSRAIRecursionDepth {
		tp.golem.LogWarn("SRAI recursion depth limit reached (%d), stopping recursion", MaxSRAIRecursionDepth)
		return content
	}

	// Trim and normalize the content
	sraiContent := strings.TrimSpace(content)

	tp.golem.LogInfo("Processing SRAI: '%s' (depth: %d)", sraiContent, tp.ctx.RecursionDepth)

	kb := tp.ctx.KnowledgeBase
	if kb == nil {
		kb = tp.golem.aimlKB
	}

	// Try to match the SRAI content as a new AIML pattern
	if kb != nil {
		category, wildcards, topic, err := tp.matchSRAI(kb, sraiContent)
		tp.golem.LogInfo("SRAI pattern match: content='%s', err=%v, category=%v, wildcards=%v",
			sraiContent, err, category != nil, wildcards)

		if err == nil && category != nil {
			// Create a new context with incremented recursion depth
			// Preserve all context except increment recursion depth
			// Local variables (<set var>) belong to their category, so the
			// matched template gets its own
			newCtx := &VariableContext{
				LocalVars:      make(map[string]string),
				Session:        tp.ctx.Session,
				Topic:          topic,
				KnowledgeBase:  kb,
				RecursionDepth: tp.ctx.RecursionDepth + 1,
			}

			// Process the matched template with the new context; its <star/>
			// tags refer to its own match
			response := tp.golem.processTemplateWithContext(category.Template, wildcards, newCtx)
			tp.golem.LogInfo("SRAI result: '%s' -> '%s'", sraiContent, response)
			return response
		} else {
			// No match found, return the content as-is
			tp.golem.LogInfo("SRAI no match for: '%s'", sraiContent)
			return sraiContent
		}
	}

	// No knowledge base, return content as-is
	return sraiContent
}

func (tp *TreeProcessor) processSRAIXTag(node *ASTNode, content string) string {
	// Process SRAIX tag - external service integration (SRAI eXtended)

	// Get and evaluate attributes first (needed for fallback even if SRAIX manager not configured)
	serviceName := ""
	if val, exists := node.Attributes["service"]; exists {
		serviceName = strings.TrimSpace(tp.evaluateAttributeValue(val))
	}

	botName := ""
	if val, exists := node.Attributes["bot"]; exists {
		botName = strings.TrimSpace(tp.evaluateAttributeValue(val))
	}

	botID := ""
	if val, exists := node.Attributes["botid"]; exists {
		botID = strings.TrimSpace(tp.evaluateAttributeValue(val))
	}

	hostName := ""
	if val, exists := node.Attributes["host"]; exists {
		hostName = strings.TrimSpace(tp.evaluateAttributeValue(val))
	}

	defaultResponse := ""
	if val, exists := node.Attributes["default"]; exists {
		defaultResponse = strings.TrimSpace(tp.evaluateAttributeValue(val))
	}

	hintText := ""
	if val, exists := node.Attributes["hint"]; exists {
		hintText = strings.TrimSpace(tp.evaluateAttributeValue(val))
	}

	// The content is already processed by the AST
	sraixContent := strings.TrimSpace(content)

	// Decode XML entities before sending to external service
	// AIML uses XML encoding (&amp;, &lt;, etc.) but external services expect plain text
	sraixContent = html.UnescapeString(sraixContent)

	// The content and the response are logged only by size, since they
	// often carry credentials or tokens
	tp.golem.LogInfo("Processing SRAIX: service='%s', bot='%s', botid='%s', host='%s', hint='%s', content %d bytes",
		serviceName, botName, botID, hostName, hintText, len(sraixContent))

	// The tag's default comes first, then the service's fallback, then the
	// content (AIML2 spec behavior)
	fallBack := func(config *SRAIXConfig) string {
		if defaultResponse != "" {
			return defaultResponse
		}
		if config != nil && config.FallbackResponse != "" {
			return config.FallbackResponse
		}
		return sraixContent
	}

	// The bot attribute names the service when service does not
	targetService := serviceName
	if targetService == "" {
		targetService = botName
	}
	if targetService == "" {
		tp.golem.LogInfo("SRAIX tag missing service or bot attribute")
		return fallBack(nil)
	}
	config, exists := tp.golem.sraixMgr.GetConfig(targetService)
	if !exists {
		tp.golem.LogInfo("SRAIX service '%s' not configured", targetService)
		return fallBack(nil)
	}

	// The tag's attributes, and the predicates the service names, fill
	// placeholders such as {hint} and {access_token}
	requestParams := make(map[string]string)
	for key, value := range map[string]string{"botid": botID, "host": hostName, "hint": hintText} {
		if value != "" {
			requestParams[key] = value
		}
	}
	requestParams, _ = tp.sraixParams(config, requestParams)

	// Pattern wildcards are available to URL templates as {STAR1}, {STAR2}, ...
	var wildcards map[string]string
	if tp.ctx != nil {
		wildcards = tp.ctx.Wildcards
	}

	response, err := tp.requestSRAIX(config, sraixContent, wildcards, requestParams)
	if err != nil && isUnauthorized(err) && config.Reauth != "" && !tp.reauthorizing {
		// The credentials have expired: answer the service's reauth input,
		// which logs in again, and retry if it changed the params
		tp.golem.LogInfo("SRAIX service '%s' replied 401; answering %q and retrying", targetService, config.Reauth)
		tp.reauthorizing = true
		tp.processSRAITag(nil, config.Reauth)
		tp.reauthorizing = false
		retryParams, changed := tp.sraixParams(config, requestParams)
		if changed {
			response, err = tp.requestSRAIX(config, sraixContent, wildcards, retryParams)
		}
	}
	if err != nil {
		tp.golem.LogInfo("SRAIX request failed: %v", err)
		return fallBack(config)
	}
	for predicate, value := range response.set {
		tp.setPredicate(predicate, value)
	}

	tp.golem.LogInfo("SRAIX result: service='%s', %d bytes, %d predicates set", targetService, len(response.text), len(response.set))
	return response.text
}

// sraixParams returns a copy of params with the predicates the service
// names (its Params) filled in from their current values, and whether any
// of those differ from params
func (tp *TreeProcessor) sraixParams(config *SRAIXConfig, params map[string]string) (map[string]string, bool) {
	result := make(map[string]string, len(params)+len(config.Params))
	for key, value := range params {
		result[key] = value
	}
	changed := false
	for placeholder, predicate := range config.Params {
		value, exists := tp.predicateValue(predicate)
		if !exists || value == "" {
			if _, had := result[placeholder]; had {
				delete(result, placeholder)
				changed = true
			}
			continue
		}
		if result[placeholder] != value {
			result[placeholder] = value
			changed = true
		}
	}
	return result, changed
}

// requestSRAIX makes a SRAIX request without holding the knowledge base lock.
// A request can take as long as its timeout; holding the lock meanwhile
// would block a reload or a load, and, once one of those was waiting, every
// other conversation. The rest of the template sees the knowledge base as it
// is when the request returns.
func (tp *TreeProcessor) requestSRAIX(config *SRAIXConfig, input string, wildcards, params map[string]string) (sraixResponse, error) {
	if tp.kbReadLocked {
		tp.golem.kbMu.RUnlock()
		defer tp.golem.kbMu.RLock()
	}
	return tp.golem.sraixMgr.send(config, input, wildcards, params)
}

func (tp *TreeProcessor) processThinkTag(_ *ASTNode, content string) string {
	// Process think tag - evaluates content but produces no output
	// The content parameter already contains the fully processed result of all child nodes
	// (variables have been set, operations performed, etc.)
	// We simply return empty string to suppress output

	tp.golem.LogInfo("Think tag: processed '%s' (no output)", content)

	// Think tags don't output anything
	return ""
}

// evaluateAttributeValue evaluates an attribute value if it contains AIML tags
// For example, name="<star/>" will be evaluated to the actual wildcard value
func (tp *TreeProcessor) evaluateAttributeValue(value string) string {
	// Quick check: if it doesn't contain '<', it's a plain string
	if !strings.Contains(value, "<") {
		return value
	}

	// Check if it contains AIML tags
	if strings.Contains(value, "<star") || strings.Contains(value, "<get") ||
		strings.Contains(value, "<bot") || strings.Contains(value, "<that") ||
		strings.Contains(value, "<input") || strings.Contains(value, "<id") {
		// Parse and evaluate the attribute value as AIML
		parser := NewASTParser(value)
		root, err := parser.Parse()
		if err != nil {
			// If parsing fails, return the original value
			return value
		}

		// Process the parsed tree
		var result strings.Builder
		for _, node := range root.Children {
			result.WriteString(tp.processNode(node))
		}
		return strings.TrimSpace(result.String())
	}

	return value
}

func (tp *TreeProcessor) processSetTag(node *ASTNode, content string) string {
	// Process set tag - can be either variable assignment OR collection operations
	// Check for both 'name' (session predicates) and 'var' (local variables)
	name, nameExists := node.Attributes["name"]
	varName, varExists := node.Attributes["var"]

	if !nameExists && !varExists {
		return content
	}

	// Determine which attribute to use and whether this is a local variable
	isLocalVar := false
	varKey := ""
	if varExists {
		isLocalVar = true
		varKey = varName
	} else {
		varKey = name
	}

	// Evaluate the name/var if it contains AIML tags (like <star/>)
	varKey = tp.evaluateAttributeValue(varKey)

	// Local variables don't support collection operations, skip collection logic for them
	if !isLocalVar {
		// Check if this is a collection operation (has operation attribute)
		operation, hasOperation := node.Attributes["operation"]

		if hasOperation {
			// Distinguish between variable operations and Set collection operations
			// Variable operations: assign (explicit variable assignment)
			// Set collection operations: add, remove, delete, clear, size, length, contains, has, get
			if operation == "assign" {
				// Explicit variable assignment - treat as no-operation case below
				hasOperation = false
			} else {
				// This is a Set collection operation
				return tp.processSetCollectionTag(node, varKey, operation, content)
			}
		}

		// No operation attribute - check if a Set collection with this name already exists
		// If yes, treat it as "get" operation; if no, treat as variable assignment
		if tp.ctx != nil && tp.ctx.KnowledgeBase != nil && tp.ctx.KnowledgeBase.hasSetCollection(varKey) {
			// Set collection exists, treat this as a "get" operation
			return tp.processSetCollectionTag(node, varKey, "get", content)
		}
	}

	// No operation and no existing Set collection - this is variable assignment (original behavior)
	// Process the content to get the value
	value := content // Content is already processed by processNode

	// Set the variable in context
	if tp.ctx != nil {
		// Local variables are stored in LocalVars
		if isLocalVar {
			if tp.ctx.LocalVars == nil {
				tp.ctx.LocalVars = make(map[string]string)
			}
			tp.ctx.LocalVars[varKey] = value
			tp.golem.noteSecret(varKey, value)
		} else {
			// Session predicates are stored in Session.Variables
			// Special handling for "topic" - update session topic
			if varKey == "topic" && tp.ctx.Session != nil {
				tp.ctx.Session.SetSessionTopic(value)
				tp.ctx.Topic = value // Update context topic as well
			}

			// Set in session variables if session exists
			if tp.ctx.Session != nil {
				if tp.ctx.Session.Variables == nil {
					tp.ctx.Session.Variables = make(map[string]string)
				}
				tp.ctx.Session.Variables[varKey] = value
				tp.golem.noteSecret(varKey, value)
			} else if tp.ctx.KnowledgeBase != nil {
				// No session - set in knowledge base variables (global)
				tp.ctx.KnowledgeBase.setVariable(varKey, value)
			} else {
				// Fallback to local variables as last resort
				if tp.ctx.LocalVars == nil {
					tp.ctx.LocalVars = make(map[string]string)
				}
				tp.ctx.LocalVars[varKey] = value
			}
		}
	}

	// Like the AIML spec, <set> outputs the value it set; wrap it in <think>
	// to set a variable silently
	return value
}

func (tp *TreeProcessor) processSetCollectionTag(_ *ASTNode, name string, operation string, content string) string {
	// Process Set collection operations (unique values with insertion order)
	// Check for required knowledge base
	if tp.ctx == nil || tp.ctx.KnowledgeBase == nil || tp.ctx.KnowledgeBase.SetCollections == nil {
		tp.golem.LogInfo("Set collection: no knowledge base available")
		return ""
	}
	tp.ctx.KnowledgeBase.mu.Lock()
	defer tp.ctx.KnowledgeBase.mu.Unlock()

	// Get or create the set
	if tp.ctx.KnowledgeBase.SetCollections[name] == nil {
		tp.ctx.KnowledgeBase.SetCollections[name] = NewSetCollection()
		tp.golem.LogInfo("Created new set collection '%s'", name)
	}

	setData := tp.ctx.KnowledgeBase.SetCollections[name]
	item := strings.TrimSpace(content)

	tp.golem.LogInfo("Set collection tag: name='%s', operation='%s', item='%s'", name, operation, item)
	tp.golem.LogInfo("Before operation: set '%s' has %d items", name, len(setData.Items))

	switch operation {
	case "add", "insert":
		// Add item to set (only if not already present) maintaining insertion order
		if item != "" && !setData.Index[item] {
			setData.Items = append(setData.Items, item)
			setData.Index[item] = true
			tp.golem.LogInfo("Added '%s' to set '%s'", item, name)
		}
		return "" // Add operations don't return content

	case "remove", "delete":
		// Remove item from set
		if item != "" && setData.Index[item] {
			// Remove from index
			delete(setData.Index, item)
			// Remove from items slice
			for i, v := range setData.Items {
				if v == item {
					setData.Items = append(setData.Items[:i], setData.Items[i+1:]...)
					break
				}
			}
			tp.golem.LogInfo("Removed '%s' from set '%s'", item, name)
		} else {
			tp.golem.LogInfo("Item '%s' not found in set '%s'", item, name)
		}
		return "" // Remove operations don't return content

	case "clear":
		// Clear all items from set
		tp.ctx.KnowledgeBase.SetCollections[name] = NewSetCollection()
		tp.golem.LogInfo("Cleared set '%s'", name)
		return "" // Clear operations don't return content

	case "size", "length":
		// Return the size of the set
		size := strconv.Itoa(len(setData.Items))
		tp.golem.LogInfo("Set '%s' size: %s", name, size)
		return size

	case "contains", "has":
		// Check if set contains item (case-insensitive)
		contains := false
		itemLower := strings.ToLower(item)
		for _, setItem := range setData.Items {
			if strings.ToLower(setItem) == itemLower {
				contains = true
				break
			}
		}
		result := "false"
		if contains {
			result = "true"
		}
		tp.golem.LogInfo("Set '%s' contains '%s': %s", name, item, result)
		return result

	case "get", "":
		// Return all items in set (space-separated, in insertion order)
		result := strings.Join(setData.Items, " ")
		tp.golem.LogInfo("Got all items from set '%s': '%s'", name, result)
		return result

	default:
		// Unknown operation, return all items
		tp.golem.LogInfo("Unknown operation '%s', returning all items", operation)
		return strings.Join(setData.Items, " ")
	}
}

func (tp *TreeProcessor) processGetTag(node *ASTNode, content string) string {
	// Process get tag - variable retrieval
	// Check for both 'name' (session predicates) and 'var' (local variables)
	name, nameExists := node.Attributes["name"]
	varName, varExists := node.Attributes["var"]

	if !nameExists && !varExists {
		return content
	}

	// Determine which attribute to use and whether this is a local variable
	isLocalVar := false
	varKey := ""
	if varExists {
		isLocalVar = true
		varKey = varName
	} else {
		varKey = name
	}

	// Evaluate the name/var if it contains AIML tags (like <star/>)
	varKey = tp.evaluateAttributeValue(varKey)

	// Get the variable value from context
	if tp.ctx != nil {
		// If explicitly asking for local variable, check only LocalVars
		if isLocalVar {
			if tp.ctx.LocalVars != nil {
				if value, exists := tp.ctx.LocalVars[varKey]; exists {
					return value
				}
			}
			// Local variable not found, return empty
			return ""
		}

		if value, exists := tp.predicateValue(varKey); exists {
			return value
		}
	}

	// If variable not found, return the processed content as default
	return content
}

// predicateValue returns the value <get name="..."/> gives, looking in
// order at local variables (for compatibility), session predicates, and the
// topic and global variables and bot properties of the knowledge base
func (tp *TreeProcessor) predicateValue(name string) (string, bool) {
	if tp.ctx == nil {
		return "", false
	}
	if value, exists := tp.ctx.LocalVars[name]; exists {
		return value, true
	}
	if tp.ctx.Session != nil {
		if value, exists := tp.ctx.Session.Variables[name]; exists {
			return value, true
		}
	}
	if tp.ctx.KnowledgeBase != nil {
		return tp.ctx.KnowledgeBase.lookupVariable(tp.ctx.Topic, name)
	}
	return "", false
}

// setPredicate sets a session predicate, or a global variable when there is
// no session
func (tp *TreeProcessor) setPredicate(name, value string) {
	if tp.ctx == nil {
		return
	}
	if tp.ctx.Session != nil {
		if tp.ctx.Session.Variables == nil {
			tp.ctx.Session.Variables = make(map[string]string)
		}
		tp.ctx.Session.Variables[name] = value
		tp.golem.noteSecret(name, value)
	} else if tp.ctx.KnowledgeBase != nil {
		tp.ctx.KnowledgeBase.setVariable(name, value)
	}
}

func (tp *TreeProcessor) processBotTag(node *ASTNode, content string) string {
	// Process bot tag - bot property access
	name, exists := node.Attributes["name"]
	if !exists {
		return content
	}

	// Get bot property from knowledge base
	if tp.ctx != nil && tp.ctx.KnowledgeBase != nil {
		if value, exists := tp.ctx.KnowledgeBase.Properties[name]; exists {
			return value
		}
	}

	// Property not found
	return ""
}

func (tp *TreeProcessor) processStarTag(node *ASTNode, _ string) string {
	// Process star tag - wildcard reference
	// <star/> without index always refers to star1 (first wildcard)
	// <star index="2"/> refers to star2 (second wildcard), etc.
	// If no pattern wildcards exist, falls back to that pattern wildcards
	index := 1
	if idx, exists := node.Attributes["index"]; exists {
		// Explicit index provided
		if parsed, err := strconv.Atoi(idx); err == nil {
			index = parsed
		}
	}

	key := fmt.Sprintf("star%d", index)

	// Get wildcard value from the context
	if tp.ctx != nil {
		if tp.ctx.Wildcards != nil {
			if value, exists := tp.ctx.Wildcards[key]; exists {
				return value
			}
		}

		// Fallback: If no pattern wildcard found, check for that pattern wildcards
		// This allows <star/> to reference wildcards from <that> patterns when
		// the main pattern has no wildcards
		thatKey := fmt.Sprintf("that_star%d", index)
		if tp.ctx.Session != nil {
			if value, exists := tp.ctx.Session.Variables[thatKey]; exists {
				tp.golem.LogDebug("Star tag: falling back to that wildcard %s", thatKey)
				return value
			}
		}
		if tp.ctx.Wildcards != nil {
			if value, exists := tp.ctx.Wildcards[thatKey]; exists {
				tp.golem.LogDebug("Star tag: falling back to that wildcard %s", thatKey)
				return value
			}
		}
	}

	return ""
}

func (tp *TreeProcessor) processSRTag(_ *ASTNode, _ string) string {
	// Process SR tag - shorthand for <srai><star/></srai>, so it shares srai's
	// matching, context handling and recursion limit
	if tp.ctx == nil {
		tp.golem.LogDebug("SR tag: no context available")
		return ""
	}

	starContent := tp.processStarTag(&ASTNode{Type: NodeTypeSelfClosingTag, TagName: "star", Attributes: map[string]string{}}, "")
	tp.golem.LogDebug("SR tag: star1 content='%s'", starContent)
	if starContent == "" {
		tp.golem.LogDebug("SR tag: no star content available")
		return ""
	}

	return tp.processSRAITag(nil, starContent)
}

func (tp *TreeProcessor) processThatTag(node *ASTNode, _ string) string {
	// Process that tag - previous response reference
	// <that/> or <that> with no index returns the most recent response (index 1)
	// <that index="N"/> returns the Nth most recent response

	if tp.ctx == nil || tp.ctx.Session == nil {
		return ""
	}

	// Get the index attribute, default to 1 (most recent)
	index := 1
	if indexStr, exists := node.Attributes["index"]; exists {
		if parsed, err := strconv.Atoi(indexStr); err == nil && parsed > 0 {
			index = parsed
		}
	}

	// Get the response by index
	response := tp.ctx.Session.GetResponseByIndex(index)

	tp.golem.LogDebug("That tag: index=%d, response='%s'", index, response)

	return response
}

func (tp *TreeProcessor) processThatStarTag(node *ASTNode, _ string) string {
	// Process that_star tag - wildcard reference from that pattern
	// <that_star1/> refers to the first star wildcard in the that pattern
	// <that_star index="2"/> refers to the second star wildcard, etc.
	return tp.processThatWildcardTag(node, "that_star")
}

func (tp *TreeProcessor) processTopicStarTag(node *ASTNode, _ string) string {
	// Process topicstar tag - wildcard reference from the topic pattern
	// <topicstar index="2"/> refers to the second wildcard, etc.
	return tp.processThatWildcardTag(node, "topic_star")
}

func (tp *TreeProcessor) processThatUnderscoreTag(node *ASTNode, _ string) string {
	// Process that_underscore tag - underscore wildcard from that pattern
	return tp.processThatWildcardTag(node, "that_underscore")
}

func (tp *TreeProcessor) processThatCaretTag(node *ASTNode, _ string) string {
	// Process that_caret tag - caret wildcard from that pattern
	return tp.processThatWildcardTag(node, "that_caret")
}

func (tp *TreeProcessor) processThatHashTag(node *ASTNode, _ string) string {
	// Process that_hash tag - hash wildcard from that pattern
	return tp.processThatWildcardTag(node, "that_hash")
}

func (tp *TreeProcessor) processThatDollarTag(node *ASTNode, _ string) string {
	// Process that_dollar tag - dollar wildcard from that pattern
	return tp.processThatWildcardTag(node, "that_dollar")
}

func (tp *TreeProcessor) processThatWildcardTag(node *ASTNode, wildcardType string) string {
	// Generic handler for that wildcard tags
	// wildcardType is "that_star", "that_underscore", "that_caret", "that_hash", or "that_dollar"

	// Get the index attribute, default to 1
	index := 1
	if idx, exists := node.Attributes["index"]; exists {
		if parsed, err := strconv.Atoi(idx); err == nil && parsed > 0 {
			index = parsed
		}
	}

	// Build the wildcard key (e.g., "that_star1", "that_underscore2")
	key := fmt.Sprintf("%s%d", wildcardType, index)

	// Get wildcard value from context
	if tp.ctx != nil {
		if tp.ctx.Wildcards != nil {
			if value, exists := tp.ctx.Wildcards[key]; exists {
				tp.golem.LogDebug("That wildcard tag: key=%s, value='%s' (from context)", key, value)
				return value
			}
		}
	}

	tp.golem.LogDebug("That wildcard tag: key=%s not found", key)
	return ""
}

func (tp *TreeProcessor) processThatWildcardWithEmbeddedIndex(node *ASTNode, wildcardType string) string {
	// Handle tags with embedded index like <that_star1/>, <that_underscore2/>, etc.
	// Extract the index from the tag name

	// Remove the wildcard type prefix to get the index suffix
	suffix := strings.TrimPrefix(node.TagName, wildcardType)

	// Parse the index from the suffix
	index := 1
	if len(suffix) > 0 {
		if parsed, err := strconv.Atoi(suffix); err == nil && parsed > 0 {
			index = parsed
		}
	}

	// Build the wildcard key
	key := fmt.Sprintf("%s%d", wildcardType, index)

	// Get wildcard value from context
	if tp.ctx != nil {
		if tp.ctx.Wildcards != nil {
			if value, exists := tp.ctx.Wildcards[key]; exists {
				tp.golem.LogDebug("That wildcard tag (embedded index): key=%s, value='%s' (from context)", key, value)
				return value
			}
		}
	}

	tp.golem.LogDebug("That wildcard tag (embedded index): key=%s not found", key)
	return ""
}

func (tp *TreeProcessor) processTopicTag(node *ASTNode, _ string) string {
	// Process topic tag - topic reference
	index := 1
	if idx, exists := node.Attributes["index"]; exists {
		if parsed, err := strconv.Atoi(idx); err == nil {
			index = parsed
		}
	}

	// Get topic value
	if tp.ctx != nil {
		if index == 1 {
			return tp.ctx.Topic
		}
	}

	return ""
}

func (tp *TreeProcessor) processRandomTag(node *ASTNode, content string) string {
	// Process random tag - random selection from list items. Only the chosen
	// item is processed, so the others' <set>, <srai> and <learn> do not run.
	// An empty item is a valid choice (it says nothing).
	var items []*ASTNode
	for _, child := range node.Children {
		if child.Type == NodeTypeTag && child.TagName == "li" {
			items = append(items, child)
		}
	}

	if len(items) == 0 {
		return content
	}

	// Select random item
	index := tp.golem.randomIntTree(len(items))
	return strings.TrimSpace(tp.processNode(items[index]))
}

func (tp *TreeProcessor) processListItemTag(_ *ASTNode, content string) string {
	// Process list item tag - processTag has already processed the children
	return content
}

// maxConditionLoops limits how many times <loop/> can repeat a condition
const maxConditionLoops = 100

// conditionValue returns the value of the variable a condition or <li>
// tests: a session predicate for name, a local variable for var
func (tp *TreeProcessor) conditionValue(node *ASTNode) (string, bool) {
	if varName, exists := node.Attributes["var"]; exists {
		return tp.ctx.LocalVars[tp.evaluateAttributeValue(varName)], true
	}
	if name, exists := node.Attributes["name"]; exists {
		return tp.golem.resolveVariable(tp.evaluateAttributeValue(name), tp.ctx), true
	}
	return "", false
}

// conditionMatches reports whether a variable's value matches an expected
// value. "*" matches any value; otherwise values are compared ignoring case.
func conditionMatches(actual, expected string) bool {
	if expected == "*" {
		return actual != ""
	}
	return strings.EqualFold(actual, expected)
}

// processChildren processes a node's children and joins the results
func (tp *TreeProcessor) processChildren(node *ASTNode) string {
	var result strings.Builder
	for _, child := range node.Children {
		result.WriteString(tp.processNode(child))
	}
	return result.String()
}

func (tp *TreeProcessor) processConditionTag(node *ASTNode, _ string) string {
	// Process condition tag - conditional logic (native implementation).
	// A <li> containing <loop/> repeats the condition, up to maxConditionLoops times.
	if tp.ctx == nil {
		return ""
	}

	var result strings.Builder
	for i := 0; i < maxConditionLoops; i++ {
		output, loop := tp.processConditionOnce(node)
		result.WriteString(output)
		if !loop {
			return result.String()
		}
	}
	tp.golem.LogWarn("Condition loop limit reached (%d), stopping", maxConditionLoops)
	return result.String()
}

// processConditionOnce evaluates a condition once. It reports whether the
// chosen <li> contained <loop/>.
func (tp *TreeProcessor) processConditionOnce(node *ASTNode) (string, bool) {
	actualValue, hasName := tp.conditionValue(node)

	// Type 1: Simple condition with value attribute
	if expectedValue, hasExpectedValue := node.Attributes["value"]; hasExpectedValue {
		if conditionMatches(actualValue, tp.evaluateAttributeValue(expectedValue)) {
			return tp.processChildren(node), false
		}
		return "", false // No match
	}

	// Type 2: Multiple <li> conditions. Each <li> tests the condition's
	// variable, or its own name/var if it has one.
	var defaultLi *ASTNode
	hasLi := false
	for _, child := range node.Children {
		if child.Type != NodeTypeTag || child.TagName != "li" {
			continue
		}
		hasLi = true
		li := tp.liftAttributeChildren(child)

		liValue, hasValue := li.Attributes["value"]
		// If no value, this is the default case - save it for later.
		// value="" is not a default: it matches an empty variable
		if !hasValue {
			if defaultLi == nil {
				defaultLi = li
			}
			continue
		}

		value := actualValue
		if liActual, liHasName := tp.conditionValue(li); liHasName {
			value = liActual
		}
		if conditionMatches(value, tp.evaluateAttributeValue(liValue)) {
			return tp.processConditionItem(li)
		}
	}

	// No match found, use default <li> if available
	if defaultLi != nil {
		return tp.processConditionItem(defaultLi)
	}

	// Type 3: No <li> elements and no value - just check if variable has a value
	if !hasLi && hasName && actualValue != "" {
		return tp.processChildren(node), false
	}

	return "", false // No match
}

// processConditionItem processes the chosen <li> of a condition. It reports
// whether the item contained <loop/>; a <loop/> inside a nested condition
// belongs to that condition.
func (tp *TreeProcessor) processConditionItem(li *ASTNode) (string, bool) {
	outerLoop := tp.loop
	tp.loop = false
	output := strings.TrimSpace(tp.processChildren(li))
	loop := tp.loop
	tp.loop = outerLoop
	return output, loop
}

func (tp *TreeProcessor) processMapTag(node *ASTNode, content string) string {
	// Process map tag - mapping operations
	// Check for required knowledge base
	if tp.ctx == nil || tp.ctx.KnowledgeBase == nil || tp.ctx.KnowledgeBase.Maps == nil {
		return content
	}

	// Get the map name
	name, exists := node.Attributes["name"]
	if !exists {
		return content
	}

	// Evaluate the map name if it contains tags
	name = tp.evaluateAttributeValue(name)

	// Get the key attribute (optional)
	keyAttr, hasKeyAttr := node.Attributes["key"]
	if hasKeyAttr {
		keyAttr = tp.evaluateAttributeValue(keyAttr)
	}

	// Get the operation (optional, default is "get")
	operation, hasOperation := node.Attributes["operation"]
	if !hasOperation {
		operation = "get"
	}

	// Determine the key: if key attribute is provided, use it; otherwise use content
	key := keyAttr
	if !hasKeyAttr || key == "" {
		key = strings.TrimSpace(content)
	}

	tp.golem.LogInfo("Map tag: name='%s', key='%s', operation='%s', content='%s'", name, key, operation, content)

	tp.ctx.KnowledgeBase.mu.Lock()
	defer tp.ctx.KnowledgeBase.mu.Unlock()

	// Get or create the map
	if tp.ctx.KnowledgeBase.Maps[name] == nil {
		tp.ctx.KnowledgeBase.Maps[name] = make(map[string]string)
		tp.golem.LogInfo("Created new map '%s'", name)
	}

	mapData := tp.ctx.KnowledgeBase.Maps[name]
	tp.golem.LogInfo("Before operation: map '%s' = %v", name, mapData)

	switch operation {
	case "set", "assign":
		// Set a key-value pair
		if key != "" {
			tp.ctx.KnowledgeBase.Maps[name][key] = strings.TrimSpace(content)
			tp.golem.LogInfo("Set map '%s'['%s'] = '%s'", name, key, strings.TrimSpace(content))
			tp.golem.LogInfo("After set: map '%s' = %v", name, tp.ctx.KnowledgeBase.Maps[name])
			return "" // Set operations don't return content
		}
		return ""

	case "remove", "delete":
		// Remove a key-value pair
		if key != "" {
			if _, exists := tp.ctx.KnowledgeBase.Maps[name][key]; exists {
				delete(tp.ctx.KnowledgeBase.Maps[name], key)
				tp.golem.LogInfo("Removed key '%s' from map '%s'", key, name)
				tp.golem.LogInfo("After remove: map '%s' = %v", name, tp.ctx.KnowledgeBase.Maps[name])
			} else {
				tp.golem.LogInfo("Key '%s' not found in map '%s'", key, name)
			}
		}
		return "" // Remove operations don't return content

	case "clear":
		// Clear all entries
		tp.ctx.KnowledgeBase.Maps[name] = make(map[string]string)
		tp.golem.LogInfo("Cleared map '%s'", name)
		return "" // Clear operations don't return content

	case "size", "length":
		// Return the size of the map
		size := strconv.Itoa(len(tp.ctx.KnowledgeBase.Maps[name]))
		tp.golem.LogInfo("Map '%s' size: %s", name, size)
		return size

	case "contains", "has":
		// Check if map contains key
		contains := false
		if key != "" {
			_, contains = tp.ctx.KnowledgeBase.Maps[name][key]
		}
		result := "false"
		if contains {
			result = "true"
		}
		tp.golem.LogInfo("Map '%s' contains key '%s': %s", name, key, result)
		return result

	case "keys":
		// Return all keys
		keys := make([]string, 0, len(tp.ctx.KnowledgeBase.Maps[name]))
		for k := range tp.ctx.KnowledgeBase.Maps[name] {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Sort for consistent output
		keysString := strings.Join(keys, " ")
		tp.golem.LogInfo("Map '%s' keys: %s", name, keysString)
		return keysString

	case "values":
		// Return all values
		values := make([]string, 0, len(tp.ctx.KnowledgeBase.Maps[name]))
		for _, v := range tp.ctx.KnowledgeBase.Maps[name] {
			values = append(values, v)
		}
		sort.Strings(values) // Sort for consistent output
		valuesString := strings.Join(values, " ")
		tp.golem.LogInfo("Map '%s' values: %s", name, valuesString)
		return valuesString

	case "list":
		// Return all key-value pairs
		pairs := make([]string, 0, len(tp.ctx.KnowledgeBase.Maps[name]))
		for k, v := range tp.ctx.KnowledgeBase.Maps[name] {
			pairs = append(pairs, k+":"+v)
		}
		sort.Strings(pairs) // Sort for consistent output
		pairsString := strings.Join(pairs, " ")
		tp.golem.LogInfo("Map '%s' pairs: %s", name, pairsString)
		return pairsString

	case "get", "":
		// Get value by key (original functionality)
		if key != "" {
			if value, exists := tp.ctx.KnowledgeBase.Maps[name][key]; exists {
				tp.golem.LogInfo("Mapped '%s' -> '%s'", key, value)
				return value
			} else {
				// Key not found in map, return the original key
				tp.golem.LogInfo("Key '%s' not found in map '%s', returning key", key, name)
				return key
			}
		}
		return ""

	default:
		// Unknown operation, treat as get
		tp.golem.LogInfo("Unknown operation '%s', treating as get", operation)
		if key != "" {
			if value, exists := tp.ctx.KnowledgeBase.Maps[name][key]; exists {
				return value
			}
			return key
		}
		return ""
	}
}

func (tp *TreeProcessor) processListTag(node *ASTNode, content string) string {
	// Process list tag - list operations
	// Get the list name
	name, exists := node.Attributes["name"]
	if !exists {
		return content
	}

	// Evaluate the list name if it contains tags
	name = tp.evaluateAttributeValue(name)

	// Get the index attribute (optional)
	indexStr, hasIndex := node.Attributes["index"]
	if hasIndex {
		indexStr = tp.evaluateAttributeValue(indexStr)
	}

	// Get the operation (optional)
	operation, hasOperation := node.Attributes["operation"]
	if !hasOperation {
		operation = ""
	}

	// If no knowledge base, just return empty string for operations
	if tp.ctx == nil || tp.ctx.KnowledgeBase == nil || tp.ctx.KnowledgeBase.Lists == nil {
		tp.golem.LogInfo("List processing: no knowledge base available")
		return ""
	}
	tp.ctx.KnowledgeBase.mu.Lock()
	defer tp.ctx.KnowledgeBase.mu.Unlock()

	// Get or create the list
	if tp.ctx.KnowledgeBase.Lists[name] == nil {
		tp.ctx.KnowledgeBase.Lists[name] = make([]string, 0)
		tp.golem.LogInfo("Created new list '%s'", name)
	}
	list := tp.ctx.KnowledgeBase.Lists[name]
	tp.golem.LogInfo("Processing list tag: name='%s', index='%s', operation='%s', content='%s'", name, indexStr, operation, content)
	tp.golem.LogInfo("Before operation: list '%s' = %v", name, list)

	switch operation {
	case "add", "append":
		// Add item to the end of the list
		list = append(list, content)
		tp.ctx.KnowledgeBase.Lists[name] = list
		tp.golem.LogInfo("Added '%s' to list '%s'", content, name)
		tp.golem.LogInfo("After add: list '%s' = %v", name, list)
		return "" // List operations don't return content

	case "insert":
		// Insert item at specific index
		if hasIndex {
			if index, err := strconv.Atoi(indexStr); err == nil && index >= 0 && index <= len(list) {
				// Insert at the specified index
				list = append(list[:index], append([]string{content}, list[index:]...)...)
				tp.ctx.KnowledgeBase.Lists[name] = list
				tp.golem.LogInfo("Inserted '%s' at index %d in list '%s'", content, index, name)
				tp.golem.LogInfo("After insert: list '%s' = %v", name, list)
			} else {
				// Invalid index, append to end
				list = append(list, content)
				tp.ctx.KnowledgeBase.Lists[name] = list
				tp.golem.LogInfo("Invalid index %s, appended '%s' to list '%s'", indexStr, content, name)
				tp.golem.LogInfo("After append: list '%s' = %v", name, list)
			}
		} else {
			// No index specified, append to end
			list = append(list, content)
			tp.ctx.KnowledgeBase.Lists[name] = list
			tp.golem.LogInfo("No index specified, appended '%s' to list '%s'", content, name)
			tp.golem.LogInfo("After append: list '%s' = %v", name, list)
		}
		return ""

	case "remove", "delete":
		// Remove item from list
		if hasIndex {
			if index, err := strconv.Atoi(indexStr); err == nil && index >= 0 && index < len(list) {
				// Remove at specific index
				list = append(list[:index], list[index+1:]...)
				tp.ctx.KnowledgeBase.Lists[name] = list
				tp.golem.LogInfo("Removed item at index %d from list '%s'", index, name)
				tp.golem.LogInfo("After remove by index: list '%s' = %v", name, list)
			} else {
				// Invalid index, try to remove by value
				for i, item := range list {
					if item == content {
						list = append(list[:i], list[i+1:]...)
						tp.ctx.KnowledgeBase.Lists[name] = list
						tp.golem.LogInfo("Removed '%s' from list '%s'", content, name)
						tp.golem.LogInfo("After remove by value: list '%s' = %v", name, list)
						break
					}
				}
			}
		} else {
			// No index, remove by value
			for i, item := range list {
				if item == content {
					list = append(list[:i], list[i+1:]...)
					tp.ctx.KnowledgeBase.Lists[name] = list
					tp.golem.LogInfo("Removed '%s' from list '%s'", content, name)
					tp.golem.LogInfo("After remove by value: list '%s' = %v", name, list)
					break
				}
			}
		}
		return ""

	case "clear":
		// Clear the list
		tp.ctx.KnowledgeBase.Lists[name] = make([]string, 0)
		tp.golem.LogInfo("Cleared list '%s'", name)
		return ""

	case "size", "length":
		// Return size of list
		size := strconv.Itoa(len(list))
		tp.golem.LogInfo("List '%s' size: %s", name, size)
		return size

	case "get", "":
		// Get item(s) from list
		if hasIndex {
			// Get item at specific index
			if index, err := strconv.Atoi(indexStr); err == nil && index >= 0 && index < len(list) {
				tp.golem.LogInfo("Got list '%s'[%d] = '%s'", name, index, list[index])
				return list[index]
			}
			// Invalid index
			tp.golem.LogInfo("Invalid index %s for list '%s'", indexStr, name)
			return ""
		}
		// No index, return all items joined
		result := strings.Join(list, " ")
		tp.golem.LogInfo("Got all items from list '%s': '%s'", name, result)
		return result

	default:
		// Unknown operation, return all items
		tp.golem.LogInfo("Unknown operation '%s', returning all items", operation)
		return strings.Join(list, " ")
	}
}

func (tp *TreeProcessor) processArrayTag(node *ASTNode, content string) string {
	// Process array tag - array operations
	// Get the array name
	name, exists := node.Attributes["name"]
	if !exists {
		return content
	}

	// Evaluate the array name if it contains tags
	name = tp.evaluateAttributeValue(name)

	// Get the index attribute (optional)
	indexStr, hasIndex := node.Attributes["index"]
	if hasIndex {
		indexStr = tp.evaluateAttributeValue(indexStr)
	}

	// Get the operation (optional, default is "get")
	operation, hasOperation := node.Attributes["operation"]
	if !hasOperation {
		operation = "get"
	}

	// If no knowledge base, just return empty string
	if tp.ctx == nil || tp.ctx.KnowledgeBase == nil || tp.ctx.KnowledgeBase.Arrays == nil {
		tp.golem.LogInfo("Array processing: no knowledge base available")
		return ""
	}
	tp.ctx.KnowledgeBase.mu.Lock()
	defer tp.ctx.KnowledgeBase.mu.Unlock()

	// Get or create the array
	if tp.ctx.KnowledgeBase.Arrays[name] == nil {
		tp.ctx.KnowledgeBase.Arrays[name] = make([]string, 0)
		tp.golem.LogInfo("Created new array '%s'", name)
	}
	array := tp.ctx.KnowledgeBase.Arrays[name]
	tp.golem.LogInfo("Processing array tag: name='%s', index='%s', operation='%s', content='%s'", name, indexStr, operation, content)
	tp.golem.LogInfo("Before operation: array '%s' = %v", name, array)

	switch operation {
	case "set", "assign":
		// Set item at specific index
		if hasIndex {
			if index, err := strconv.Atoi(indexStr); err == nil && index >= 0 {
				// Ensure array is large enough
				for len(array) <= index {
					array = append(array, "")
				}
				array[index] = content
				tp.ctx.KnowledgeBase.Arrays[name] = array
				tp.golem.LogInfo("Set array '%s'[%d] = '%s'", name, index, content)
				tp.golem.LogInfo("After set: array '%s' = %v", name, array)
			} else {
				// Invalid index
				tp.golem.LogInfo("Invalid index %s for array '%s'", indexStr, name)
			}
		} else {
			// No index specified, append to end
			array = append(array, content)
			tp.ctx.KnowledgeBase.Arrays[name] = array
			tp.golem.LogInfo("Appended '%s' to array '%s'", content, name)
			tp.golem.LogInfo("After append: array '%s' = %v", name, array)
		}
		return "" // Set operations don't return content

	case "get", "":
		// Get item at index
		if hasIndex {
			if index, err := strconv.Atoi(indexStr); err == nil && index >= 0 && index < len(array) {
				tp.golem.LogInfo("Got array '%s'[%d] = '%s'", name, index, array[index])
				return array[index]
			}
			// Invalid index
			tp.golem.LogInfo("Invalid index %s for array '%s'", indexStr, name)
			return ""
		}
		// Return all items joined by space
		items := strings.Join(array, " ")
		tp.golem.LogInfo("Got all items from array '%s': '%s'", name, items)
		return items

	case "size", "length":
		// Return the size of the array
		size := strconv.Itoa(len(array))
		tp.golem.LogInfo("Array '%s' size: %s", name, size)
		return size

	case "clear":
		// Clear the array
		tp.ctx.KnowledgeBase.Arrays[name] = make([]string, 0)
		tp.golem.LogInfo("Cleared array '%s'", name)
		tp.golem.LogInfo("After clear: array '%s' = %v", name, tp.ctx.KnowledgeBase.Arrays[name])
		return ""

	default:
		// Unknown operation, treat as get
		tp.golem.LogInfo("Unknown operation '%s', treating as get", operation)
		if hasIndex {
			if index, err := strconv.Atoi(indexStr); err == nil && index >= 0 && index < len(array) {
				return array[index]
			}
			return ""
		}
		return strings.Join(array, " ")
	}
}

func (tp *TreeProcessor) processLearnTag(node *ASTNode, _ string) string {
	// Process learn tag - dynamic learning (session-specific)
	// Process content while evaluating wildcards to capture teaching values
	// This evaluates tags like <get>, <uppercase>, <star/> (when wildcards exist)
	// Wildcards in the teaching pattern are evaluated and captured as literal values
	processedContent := tp.processNodePreservingReferences(node)

	tp.golem.learnCategories(processedContent, tp.learnContext(), false)
	return ""
}

func (tp *TreeProcessor) processLearnfTag(node *ASTNode, _ string) string {
	// Process learnf tag - persistent learning
	// The <learnf> tag adds categories to the persistent knowledge base
	// Unlike <learn>, these persist across sessions
	// Process content while evaluating wildcards to capture teaching values
	// This evaluates tags like <get>, <uppercase>, <star/> (when wildcards exist)
	// Wildcards in the teaching pattern are evaluated and captured as literal values
	processedContent := tp.processNodePreservingReferences(node)

	tp.golem.learnCategories(processedContent, tp.learnContext(), true)
	return ""
}

// learnContext passes the actual wildcards so they can be evaluated during
// learning; this lets patterns like "TEACH ME * MEANS *" capture their values
func (tp *TreeProcessor) learnContext() *VariableContext {
	return &VariableContext{
		Session:       tp.ctx.Session,
		LocalVars:     tp.ctx.LocalVars,
		KnowledgeBase: tp.ctx.KnowledgeBase,
		Wildcards:     tp.ctx.Wildcards,
	}
}

// processNodePreservingReferences processes a node's children while preserving reference tags
// Reference tags (like <star/>, <that/>, <input/>, etc.) are output as their string representation
// Other tags (like <get/>, <uppercase/>, etc.) are processed normally
func (tp *TreeProcessor) processNodePreservingReferences(node *ASTNode) string {
	var result strings.Builder

	for _, child := range node.Children {
		result.WriteString(tp.processChildPreservingReferences(child))
	}

	return result.String()
}

// processChildPreservingReferences renders a node of a <learn> body. Like the
// AIML spec, only <eval> is evaluated when the category is learned; every
// other tag is kept so that it runs when the learned category is used. As an
// extension, wildcard tags (<star/>, ...) are also evaluated when the teaching
// category has wildcards, so that "TEACH * MEANS *" works without <eval>.
func (tp *TreeProcessor) processChildPreservingReferences(node *ASTNode) string {
	switch node.Type {
	case NodeTypeText:
		if len(node.Children) > 0 {
			return tp.processNodePreservingReferences(node)
		}
		return node.Content
	case NodeTypeComment, NodeTypeCDATA:
		return node.String()
	case NodeTypeSelfClosingTag, NodeTypeTag:
	default:
		return tp.processNode(node)
	}

	if node.TagName == "eval" {
		// Evaluate now, and escape the result so it stays text when the
		// learned category is parsed
		return html.EscapeString(strings.TrimSpace(tp.processChildren(node)))
	}

	if node.TagName == "pattern" {
		return "<pattern>" + tp.processLearnedPattern(node) + "</pattern>"
	}

	if learnWildcardTags[node.TagName] && tp.ctx != nil && len(tp.ctx.Wildcards) > 0 {
		// Teaching scenario: capture the wildcard value
		return html.EscapeString(tp.processNode(node))
	}

	// Keep the tag for runtime, rendering its children with the same rules
	// so that an <eval> inside it is still evaluated now
	if node.Type == NodeTypeSelfClosingTag {
		return node.String()
	}
	return fmt.Sprintf("<%s%s>%s</%s>", node.TagName, formatAttributes(node.Attributes),
		tp.processNodePreservingReferences(node), node.TagName)
}

// processLearnedPattern renders the <pattern> of a learned category. Its tags
// are evaluated now, since the pattern must be plain words and wildcards; a
// <star/> becomes a * wildcard unless the teaching category has wildcards,
// in which case it captures their value as above.
func (tp *TreeProcessor) processLearnedPattern(node *ASTNode) string {
	var result strings.Builder
	for _, child := range node.Children {
		switch {
		case child.Type == NodeTypeText && len(child.Children) > 0:
			result.WriteString(tp.processLearnedPattern(child))
		case child.Type == NodeTypeText:
			result.WriteString(child.Content)
		case child.Type == NodeTypeComment || child.Type == NodeTypeCDATA:
			result.WriteString(child.String())
		case child.TagName == "star" && (tp.ctx == nil || len(tp.ctx.Wildcards) == 0):
			result.WriteString("*")
		case child.TagName == "eval":
			result.WriteString(html.EscapeString(strings.TrimSpace(tp.processChildren(child))))
		default:
			result.WriteString(html.EscapeString(tp.processNode(child)))
		}
	}
	return result.String()
}

// learnWildcardTags are the tags that <learn> evaluates when the teaching
// category has wildcards
var learnWildcardTags = map[string]bool{
	"star":      true,
	"thatstar":  true,
	"topicstar": true,
}

// Text processing tags

func (tp *TreeProcessor) processUppercaseTag(_ *ASTNode, content string) string {
	// Process content directly - convert to uppercase
	tp.trackMetric("format") // Track format processor usage
	processedContent := strings.ToUpper(content)

	// Normalize whitespace like the original method
	processedContent = strings.TrimSpace(processedContent)
	if processedContent == "" && len(content) > 0 {
		return content // Preserve whitespace-only content
	}

	// Normalize internal whitespace
	processedContent = regexp.MustCompile(`\s+`).ReplaceAllString(processedContent, " ")

	return processedContent
}

func (tp *TreeProcessor) processLowercaseTag(_ *ASTNode, content string) string {
	// Process content directly - convert to lowercase
	tp.trackMetric("format") // Track format processor usage
	processedContent := strings.ToLower(content)

	// Normalize whitespace like the original method
	processedContent = strings.TrimSpace(processedContent)
	if processedContent == "" && len(content) > 0 {
		return content // Preserve whitespace-only content
	}

	// Normalize internal whitespace
	processedContent = regexp.MustCompile(`\s+`).ReplaceAllString(processedContent, " ")

	return processedContent
}

func (tp *TreeProcessor) processFormalTag(_ *ASTNode, content string) string {
	// Process content directly - capitalize first letter of each word
	tp.trackMetric("format") // Track format processor usage
	words := strings.Fields(content)
	var result []string

	for _, word := range words {
		if len(word) > 0 {
			// Capitalize first letter, lowercase the rest
			capitalized := strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
			result = append(result, capitalized)
		}
	}

	return strings.Join(result, " ")
}

func (tp *TreeProcessor) processCapitalizeTag(_ *ASTNode, content string) string {
	// Process content directly - capitalize first letter, lowercase the rest (AIML spec)
	if content == "" {
		return content
	}

	// Convert to rune slice to handle Unicode properly
	runes := []rune(content)
	if len(runes) == 0 {
		return ""
	}

	// Special case: if input consists of single-character tokens separated by spaces
	tokens := strings.Fields(content)
	if len(tokens) > 1 {
		allSingle := true
		for _, t := range tokens {
			if len([]rune(t)) != 1 {
				allSingle = false
				break
			}
		}
		if allSingle {
			for i := range tokens {
				tokens[i] = strings.ToUpper(tokens[i])
			}
			return strings.Join(tokens, " ")
		}
	}

	// Capitalize first character, lowercase the rest
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	for i := 1; i < len(runes); i++ {
		runes[i] = []rune(strings.ToLower(string(runes[i])))[0]
	}

	return string(runes)
}

func (tp *TreeProcessor) processExplodeTag(_ *ASTNode, content string) string {
	// Process content directly - add spaces between characters
	var result strings.Builder
	for i, char := range content {
		if i > 0 {
			result.WriteRune(' ')
		}
		result.WriteRune(char)
	}
	return result.String()
}

func (tp *TreeProcessor) processReverseTag(_ *ASTNode, content string) string {
	// Process content directly - reverse the string
	runes := []rune(content)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func (tp *TreeProcessor) processAcronymTag(_ *ASTNode, content string) string {
	// Process content directly - convert to acronym (first letter of each word, uppercase)
	words := strings.Fields(content)
	var acronym strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			// Take first letter and uppercase it
			firstLetter := strings.ToUpper(string(word[0]))
			acronym.WriteString(firstLetter)
		}
	}
	return acronym.String()
}

func (tp *TreeProcessor) processTrimTag(_ *ASTNode, content string) string {
	// Process content directly - trim whitespace
	return strings.TrimSpace(content)
}

func (tp *TreeProcessor) processSubstringTag(node *ASTNode, content string) string {
	// Process substring tag - extract substring based on start and end positions
	// Get and evaluate attributes
	startStr, startExists := node.Attributes["start"]
	endStr, endExists := node.Attributes["end"]

	if !startExists || !endExists {
		return content
	}

	// Evaluate attributes (they might contain AIML tags)
	startStr = tp.evaluateAttributeValue(startStr)
	endStr = tp.evaluateAttributeValue(endStr)

	// Trim content
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	// Check cache first
	var result string
	cacheKey := fmt.Sprintf("%s|%s|%s", startStr, endStr, content)
	if tp.golem.templateTagProcessingCache != nil {
		if cached, found := tp.golem.templateTagProcessingCache.GetProcessedTag("substring", cacheKey, tp.ctx); found {
			result = cached
		} else {
			result = tp.golem.extractSubstring(content, startStr, endStr)
			tp.golem.templateTagProcessingCache.SetProcessedTag("substring", cacheKey, result, tp.ctx)
		}
	} else {
		result = tp.golem.extractSubstring(content, startStr, endStr)
	}

	tp.golem.LogDebug("Substring tag: '%s' (start=%s, end=%s) -> '%s'", content, startStr, endStr, result)
	return result
}

func (tp *TreeProcessor) processReplaceTag(node *ASTNode, content string) string {
	// Process replace tag - replace occurrences of search string with replacement string
	// Get and evaluate attributes
	search, searchExists := node.Attributes["search"]
	replace, replaceExists := node.Attributes["replace"]

	if !searchExists || !replaceExists {
		return content
	}

	// Evaluate attributes (they might contain AIML tags)
	search = tp.evaluateAttributeValue(search)
	replace = tp.evaluateAttributeValue(replace)

	// Trim content
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	// Check cache first
	var result string
	cacheKey := fmt.Sprintf("%s|%s|%s", search, replace, content)
	if tp.golem.templateTagProcessingCache != nil {
		if cached, found := tp.golem.templateTagProcessingCache.GetProcessedTag("replace", cacheKey, tp.ctx); found {
			result = cached
		} else {
			result = strings.ReplaceAll(content, search, replace)
			tp.golem.templateTagProcessingCache.SetProcessedTag("replace", cacheKey, result, tp.ctx)
		}
	} else {
		result = strings.ReplaceAll(content, search, replace)
	}

	tp.golem.LogDebug("Replace tag: '%s' (search='%s', replace='%s') -> '%s'", content, search, replace, result)
	return result
}

// cachedTagResult returns compute(), memoized per tag and key in the template
// tag processing cache when one is configured
func (tp *TreeProcessor) cachedTagResult(tag, key string, compute func() string) string {
	cache := tp.golem.templateTagProcessingCache
	if cache == nil {
		return compute()
	}
	if cached, found := cache.GetProcessedTag(tag, key, tp.ctx); found {
		return cached
	}
	result := compute()
	cache.SetProcessedTag(tag, key, result, tp.ctx)
	return result
}

func (tp *TreeProcessor) processPluralizeTag(_ *ASTNode, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	return tp.cachedTagResult("pluralize", content, func() string { return tp.golem.pluralizeText(content) })
}

func (tp *TreeProcessor) processShuffleTag(_ *ASTNode, content string) string {
	// Not cached: each <shuffle> should get a fresh order
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	return tp.golem.shuffleText(content)
}

func (tp *TreeProcessor) processLengthTag(node *ASTNode, content string) string {
	// Process length tag - calculate length of content
	// Get type attribute (optional)
	lengthType := ""
	if val, exists := node.Attributes["type"]; exists {
		lengthType = strings.TrimSpace(tp.evaluateAttributeValue(val))
	}

	// Trim content
	content = strings.TrimSpace(content)
	if content == "" {
		return "0"
	}

	// Check cache first
	var result string
	cacheKey := fmt.Sprintf("%s|%s", lengthType, content)
	if tp.golem.templateTagProcessingCache != nil {
		if cached, found := tp.golem.templateTagProcessingCache.GetProcessedTag("length", cacheKey, tp.ctx); found {
			result = cached
		} else {
			result = tp.golem.calculateLength(content, lengthType)
			tp.golem.templateTagProcessingCache.SetProcessedTag("length", cacheKey, result, tp.ctx)
		}
	} else {
		result = tp.golem.calculateLength(content, lengthType)
	}

	tp.golem.LogDebug("Length tag: '%s' (type='%s') -> '%s'", content, lengthType, result)
	return result
}

func (tp *TreeProcessor) processCountTag(node *ASTNode, content string) string {
	// Process count tag - count occurrences of search string in content
	// Get and evaluate search attribute
	search, searchExists := node.Attributes["search"]
	if !searchExists {
		return "0"
	}

	// Evaluate attribute (might contain AIML tags)
	search = tp.evaluateAttributeValue(search)

	// Trim content
	content = strings.TrimSpace(content)
	if content == "" || search == "" {
		return "0"
	}

	// Check cache first
	var result string
	cacheKey := fmt.Sprintf("%s|%s", search, content)
	if tp.golem.templateTagProcessingCache != nil {
		if cached, found := tp.golem.templateTagProcessingCache.GetProcessedTag("count", cacheKey, tp.ctx); found {
			result = cached
		} else {
			count := strings.Count(content, search)
			result = strconv.Itoa(count)
			tp.golem.templateTagProcessingCache.SetProcessedTag("count", cacheKey, result, tp.ctx)
		}
	} else {
		count := strings.Count(content, search)
		result = strconv.Itoa(count)
	}

	tp.golem.LogDebug("Count tag: '%s' (search='%s') -> '%s'", content, search, result)
	return result
}

func (tp *TreeProcessor) processSplitTag(node *ASTNode, content string) string {
	delimiter := strings.TrimSpace(node.Attributes["delimiter"])
	if delimiter == "" {
		delimiter = " "
	}
	limit := strings.TrimSpace(node.Attributes["limit"])
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	key := fmt.Sprintf("%s|%s|%s", delimiter, limit, content)
	return tp.cachedTagResult("split", key, func() string { return tp.golem.splitText(content, delimiter, limit) })
}

func (tp *TreeProcessor) processJoinTag(node *ASTNode, content string) string {
	delimiter := node.Attributes["delimiter"]
	if delimiter == "" {
		delimiter = " "
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	key := fmt.Sprintf("%s|%s", delimiter, content)
	return tp.cachedTagResult("join", key, func() string { return tp.golem.joinText(content, delimiter) })
}

func (tp *TreeProcessor) processUniqueTag(node *ASTNode, content string) string {
	delimiter := strings.TrimSpace(node.Attributes["delimiter"])
	if delimiter == "" {
		delimiter = " "
	}
	if strings.TrimSpace(content) == "" {
		return ""
	}
	key := fmt.Sprintf("%s|%s", delimiter, content)
	return tp.cachedTagResult("unique", key, func() string { return tp.golem.uniqueText(content, delimiter) })
}

// indentAttributes reads the level (default 1) and char (default space, with
// \t, \n and \r escapes) attributes shared by <indent> and <dedent>
func indentAttributes(node *ASTNode) (int, string) {
	level := 1
	if parsed, err := strconv.Atoi(strings.TrimSpace(node.Attributes["level"])); err == nil && parsed > 0 {
		level = parsed
	}
	char := strings.TrimSpace(node.Attributes["char"])
	if char == "" {
		char = " "
	} else {
		char = strings.ReplaceAll(char, "\\t", "\t")
		char = strings.ReplaceAll(char, "\\n", "\n")
		char = strings.ReplaceAll(char, "\\r", "\r")
	}
	return level, char
}

func (tp *TreeProcessor) processIndentTag(node *ASTNode, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	level, char := indentAttributes(node)
	key := fmt.Sprintf("%d|%s|%s", level, char, content)
	return tp.cachedTagResult("indent", key, func() string { return tp.golem.indentText(content, level, char) })
}

func (tp *TreeProcessor) processDedentTag(node *ASTNode, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	level, char := indentAttributes(node)
	key := fmt.Sprintf("%d|%s|%s", level, char, content)
	return tp.cachedTagResult("dedent", key, func() string { return tp.golem.dedentText(content, level, char) })
}

func (tp *TreeProcessor) processRepeatTag(node *ASTNode, _ string) string {
	// <repeat/> returns the most recent user request from history
	// <repeat times="3"/> repeats it 3 times
	if tp.ctx == nil || tp.ctx.Session == nil {
		return ""
	}

	// Get the most recent request (index 1)
	requestValue := tp.ctx.Session.GetRequestByIndex(1)
	if requestValue == "" {
		return ""
	}

	// Check for times attribute
	times := 1
	if t, exists := node.Attributes["times"]; exists {
		if parsed, err := strconv.Atoi(t); err == nil {
			times = parsed
		}
	}

	// Repeat the request value
	if times <= 0 {
		return ""
	}
	if times == 1 {
		return requestValue
	}

	// Repeat multiple times with spaces between
	result := make([]string, times)
	for i := 0; i < times; i++ {
		result[i] = requestValue
	}
	return strings.Join(result, " ")
}

func (tp *TreeProcessor) processFirstTag(_ *ASTNode, content string) string {
	// Get first word
	words := strings.Fields(content)
	if len(words) == 0 {
		return ""
	}
	return words[0]
}

func (tp *TreeProcessor) processRestTag(_ *ASTNode, content string) string {
	// Get all words except the first
	words := strings.Fields(content)
	if len(words) <= 1 {
		return ""
	}
	return strings.Join(words[1:], " ")
}

func (tp *TreeProcessor) processLoopTag(_ *ASTNode, _ string) string {
	// Loop tag - asks the enclosing condition to evaluate again
	tp.loop = true
	return ""
}

func (tp *TreeProcessor) processInputTag(_ *ASTNode, _ string) string {
	// Process input tag - returns the most recent user input
	// <input/> always returns the current/most recent user input (last item in RequestHistory)
	// This is different from <request> which can take an index attribute
	// Note: during ProcessInput() the current input is added to RequestHistory only
	// after the template is processed, so this returns the *previous* user input

	if tp.ctx == nil || tp.ctx.Session == nil {
		tp.golem.LogDebug("Input tag: no context or session available")
		return ""
	}

	// Get the most recent user input from request history
	if len(tp.ctx.Session.RequestHistory) == 0 {
		tp.golem.LogDebug("Input tag: no request history available")
		return ""
	}

	// Return the last (most recent) item from RequestHistory
	currentInput := tp.ctx.Session.RequestHistory[len(tp.ctx.Session.RequestHistory)-1]

	tp.golem.LogDebug("Input tag: returning '%s'", currentInput)

	return currentInput
}

func (tp *TreeProcessor) processEvalTag(_ *ASTNode, content string) string {
	// Process eval tag - evaluates AIML code dynamically
	// The <eval> tag causes its content to be evaluated as AIML template code
	// In the AST, child nodes are already processed before reaching this point,
	// so the content parameter contains the fully evaluated result
	// This allows for dynamic tag construction and re-evaluation

	// Trim whitespace from the evaluated content
	content = strings.TrimSpace(content)

	// If empty after trimming, return empty string
	if content == "" {
		tp.golem.LogDebug("Eval tag: empty content after evaluation")
		return ""
	}

	tp.golem.LogDebug("Eval tag: evaluated content='%s'", content)

	// Return the evaluated content
	// Note: Unlike the regex processor which re-processes the content through
	// the full template pipeline, the AST naturally handles nested evaluation
	// through its tree traversal, so we simply return the already-evaluated content
	return content
}

func (tp *TreeProcessor) processPersonTag(_ *ASTNode, content string) string {
	// Process person tag - pronoun substitution (1st/2nd person swap)
	// Normalize whitespace
	content = strings.TrimSpace(content)
	content = strings.Join(strings.Fields(content), " ")

	// Check cache first
	var substitutedContent string
	if tp.golem.templateTagProcessingCache != nil {
		if cached, found := tp.golem.templateTagProcessingCache.GetProcessedTag("person", content, tp.ctx); found {
			substitutedContent = cached
		} else {
			substitutedContent = tp.golem.SubstitutePronouns(content)
			tp.golem.templateTagProcessingCache.SetProcessedTag("person", content, substitutedContent, tp.ctx)
		}
	} else {
		substitutedContent = tp.golem.SubstitutePronouns(content)
	}

	tp.golem.LogInfo("Person tag: '%s' -> '%s'", content, substitutedContent)
	return substitutedContent
}

func (tp *TreeProcessor) processPerson2Tag(_ *ASTNode, content string) string {
	// Process person2 tag - first-to-third person pronoun substitution
	// Normalize whitespace
	content = strings.TrimSpace(content)
	content = strings.Join(strings.Fields(content), " ")

	// Check cache first
	var substitutedContent string
	if tp.golem.templateTagProcessingCache != nil {
		if cached, found := tp.golem.templateTagProcessingCache.GetProcessedTag("person2", content, tp.ctx); found {
			substitutedContent = cached
		} else {
			substitutedContent = tp.golem.SubstitutePronouns2(content)
			tp.golem.templateTagProcessingCache.SetProcessedTag("person2", content, substitutedContent, tp.ctx)
		}
	} else {
		substitutedContent = tp.golem.SubstitutePronouns2(content)
	}

	tp.golem.LogInfo("Person2 tag: '%s' -> '%s'", content, substitutedContent)
	return substitutedContent
}

func (tp *TreeProcessor) processGenderTag(_ *ASTNode, content string) string {
	// Process gender tag - swap masculine/feminine pronouns
	// Normalize whitespace
	content = strings.TrimSpace(content)
	content = strings.Join(strings.Fields(content), " ")

	// Check cache first
	var substitutedContent string
	if tp.golem.templateTagProcessingCache != nil {
		if cached, found := tp.golem.templateTagProcessingCache.GetProcessedTag("gender", content, tp.ctx); found {
			substitutedContent = cached
		} else {
			substitutedContent = tp.golem.SubstituteGenderPronouns(content)
			tp.golem.templateTagProcessingCache.SetProcessedTag("gender", content, substitutedContent, tp.ctx)
		}
	} else {
		substitutedContent = tp.golem.SubstituteGenderPronouns(content)
	}

	tp.golem.LogInfo("Gender tag: '%s' -> '%s'", content, substitutedContent)
	return substitutedContent
}

func (tp *TreeProcessor) processJsonFormatTag(node *ASTNode, content string) string {
	// Process jsonformat tag - converts JSON to human-readable format
	// Supports attributes: type="lists|items|list|item"
	content = strings.TrimSpace(content)

	if content == "" {
		return ""
	}

	// Get format type from attributes
	formatType := "auto"
	if node.Attributes != nil {
		if typeAttr, exists := node.Attributes["type"]; exists {
			formatType = typeAttr
		}
	}

	// Try to parse as JSON
	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		// Not valid JSON, return as-is
		tp.golem.LogInfo("JsonFormat: Not valid JSON, returning as-is")
		return content
	}

	// Auto-detect format type if not specified
	if formatType == "auto" {
		switch v := data.(type) {
		case []interface{}:
			if len(v) > 0 {
				if item, ok := v[0].(map[string]interface{}); ok {
					if _, hasContent := item["content"]; hasContent {
						formatType = "items"
					} else if _, hasName := item["name"]; hasName {
						formatType = "lists"
					}
				}
			}
		case map[string]interface{}:
			if _, hasItems := v["items"]; hasItems {
				formatType = "list"
			} else if _, hasContent := v["content"]; hasContent {
				formatType = "item"
			}
		}
	}

	// Format based on type
	switch formatType {
	case "lists":
		return tp.formatListsJSON(data)
	case "list":
		return tp.formatListJSON(data, node.Attributes["button"])
	case "items":
		return tp.formatItemsJSON(data)
	case "item":
		return tp.formatItemJSON(data)
	default:
		return tp.formatGenericJSON(data)
	}
}

func (tp *TreeProcessor) formatListsJSON(data interface{}) string {
	lists, ok := data.([]interface{})
	if !ok {
		return "Invalid lists format"
	}

	if len(lists) == 0 {
		return "You have no lists yet."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("You have %d list", len(lists)))
	if len(lists) != 1 {
		result.WriteString("s")
	}
	result.WriteString(":\n")

	for i, item := range lists {
		list, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id := fmt.Sprintf("%v", list["id"])
		name := fmt.Sprintf("%v", list["name"])
		desc := ""
		if description, exists := list["description"]; exists && description != nil {
			descStr := fmt.Sprintf("%v", description)
			if descStr != "" {
				desc = fmt.Sprintf(" - %s", descStr)
			}
		}

		result.WriteString(fmt.Sprintf("%d. %s (ID: %s)%s\n", i+1, name, id, desc))
	}

	return strings.TrimSpace(result.String())
}

// jsonItemCompleted reports whether a list item is done. The list handler
// sends is_completed as 0 or 1; "completed" may be a boolean.
func jsonItemCompleted(item map[string]interface{}) bool {
	for _, key := range []string{"is_completed", "completed"} {
		switch v := item[key].(type) {
		case bool:
			return v
		case float64:
			return v != 0
		case string:
			return v == "1" || strings.EqualFold(v, "true")
		}
	}
	return false
}

// formatListJSON formats a list and its items. With a button template, such
// as "LIST {list} TOGGLE ITEM {id}", each item also gets a rich media
// <button> whose postback is the template with {list}, {id} and {content}
// filled in, so a client can show the items as tick boxes.
func (tp *TreeProcessor) formatListJSON(data interface{}, buttonTemplate string) string {
	list, ok := data.(map[string]interface{})
	if !ok {
		return "Invalid list format"
	}

	name := fmt.Sprintf("%v", list["name"])
	id := fmt.Sprintf("%v", list["id"])

	var result, buttons strings.Builder
	result.WriteString(fmt.Sprintf("List: %s (ID: %s)\n", name, id))

	if description, exists := list["description"]; exists && description != nil {
		descStr := fmt.Sprintf("%v", description)
		if descStr != "" {
			result.WriteString(fmt.Sprintf("Description: %s\n", descStr))
		}
	}

	if items, exists := list["items"]; exists {
		itemsList, ok := items.([]interface{})
		if ok {
			if len(itemsList) == 0 {
				result.WriteString("\nThis list is empty.")
			} else {
				result.WriteString(fmt.Sprintf("\nItems (%d):\n", len(itemsList)))
				for i, item := range itemsList {
					itemMap, ok := item.(map[string]interface{})
					if !ok {
						continue
					}

					content := fmt.Sprintf("%v", itemMap["content"])
					completed := jsonItemCompleted(itemMap)

					checkbox := "☐"
					if completed {
						checkbox = "☑"
					}

					itemID := fmt.Sprintf("%v", itemMap["id"])
					result.WriteString(fmt.Sprintf("%d. %s %s (ID: %s)\n", i+1, checkbox, content, itemID))
					if buttonTemplate != "" {
						postback := strings.NewReplacer("{list}", name, "{id}", itemID, "{content}", content).Replace(buttonTemplate)
						buttons.WriteString(fmt.Sprintf("<button><text>%s %s</text><postback>%s</postback></button>",
							checkbox, html.EscapeString(content), html.EscapeString(postback)))
					}
				}
			}
		}
	}

	return strings.TrimSpace(result.String()) + buttons.String()
}

func (tp *TreeProcessor) formatItemsJSON(data interface{}) string {
	items, ok := data.([]interface{})
	if !ok {
		return "Invalid items format"
	}

	if len(items) == 0 {
		return "No items in this list."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%d item", len(items)))
	if len(items) != 1 {
		result.WriteString("s")
	}
	result.WriteString(":\n")

	for i, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		content := fmt.Sprintf("%v", itemMap["content"])
		completed := jsonItemCompleted(itemMap)

		checkbox := "☐"
		if completed {
			checkbox = "☑"
		}

		itemID := fmt.Sprintf("%v", itemMap["id"])
		result.WriteString(fmt.Sprintf("%d. %s %s (ID: %s)\n", i+1, checkbox, content, itemID))
	}

	return strings.TrimSpace(result.String())
}

func (tp *TreeProcessor) formatItemJSON(data interface{}) string {
	item, ok := data.(map[string]interface{})
	if !ok {
		return "Invalid item format"
	}

	content := fmt.Sprintf("%v", item["content"])
	id := fmt.Sprintf("%v", item["id"])
	completed := jsonItemCompleted(item)

	status := "not completed"
	if completed {
		status = "completed"
	}

	return fmt.Sprintf("Item: %s (ID: %s) - %s", content, id, status)
}

func (tp *TreeProcessor) formatGenericJSON(data interface{}) string {
	// An empty array has no entries to tell what it lists; say so in words
	// rather than showing []. A type attribute gives a better message.
	if items, ok := data.([]interface{}); ok && len(items) == 0 {
		return "There's nothing here yet."
	}
	// For generic JSON, just pretty-print it
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(bytes)
}

func (tp *TreeProcessor) processWeatherFormatTag(node *ASTNode, content string) string {
	// Process weatherformat tag - converts weather API JSON to natural language
	// Supports day="today" (default) or day="tomorrow"
	content = strings.TrimSpace(content)

	if content == "" {
		tp.golem.LogInfo("WeatherFormat: Empty content")
		return "unavailable"
	}

	// Check for day attribute (default to "today")
	day := "today"
	if node != nil && node.Attributes != nil {
		if dayAttr, exists := node.Attributes["day"]; exists {
			day = strings.ToLower(strings.TrimSpace(dayAttr))
		}
	}

	// Log what we received for debugging
	if tp.golem.verbose {
		if len(content) > 200 {
			tp.golem.LogInfo("WeatherFormat: Received content (truncated) for day=%s: %s...", day, content[:200])
		} else {
			tp.golem.LogInfo("WeatherFormat: Received content for day=%s: %s", day, content)
		}
	}

	// Check for fallback messages (non-JSON responses)
	if !strings.HasPrefix(content, "{") && !strings.HasPrefix(content, "[") {
		tp.golem.LogInfo("WeatherFormat: Non-JSON content, returning as-is: %s", content)
		return content
	}

	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		// Not valid JSON, return as-is
		tp.golem.LogInfo("WeatherFormat: Failed to parse JSON: %v, Content: %s", err, content)
		return content
	}

	// Route to appropriate formatter based on day
	if day == "tomorrow" {
		return tp.formatTomorrowWeather(data)
	}
	return tp.formatTodayWeather(data)
}

func (tp *TreeProcessor) formatTodayWeather(data map[string]interface{}) string {
	// Extract weather data for today
	var result strings.Builder
	var warnings []string

	// Get currently data
	currently, hasCurrently := data["currently"].(map[string]interface{})
	if !hasCurrently {
		tp.golem.LogInfo("WeatherFormat: No 'currently' field in response")
		// Check if there's an error message in the response
		if errMsg, hasError := data["error"].(string); hasError {
			return errMsg
		}
		return "unavailable (data format not recognized)"
	}

	// Get temperature (in Celsius since we use units=si)
	temp := 0.0
	hasTemp := false
	if tempVal, exists := currently["temperature"]; exists {
		switch v := tempVal.(type) {
		case float64:
			temp = v
			hasTemp = true
		case int:
			temp = float64(v)
			hasTemp = true
		}
	}

	// Get summary (e.g., "Cloudy", "Clear")
	summary := ""
	if summaryVal, exists := currently["summary"]; exists {
		summary = fmt.Sprintf("%v", summaryVal)
	}

	// If we don't have basic data, log it and return fallback
	if summary == "" && !hasTemp {
		tp.golem.LogInfo("WeatherFormat: Missing both summary and temperature")
		return "unavailable (missing weather data)"
	}

	// Build current conditions
	if hasTemp {
		// Convert Celsius to Fahrenheit
		tempF := (temp * 9 / 5) + 32
		if summary != "" {
			result.WriteString(fmt.Sprintf("%s with a temperature of %.0f°F (%.0f°C)", summary, tempF, temp))
		} else {
			result.WriteString(fmt.Sprintf("Temperature of %.0f°F (%.0f°C)", tempF, temp))
		}
	} else if summary != "" {
		result.WriteString(summary)
	}

	// Get wind information
	windSpeed := 0.0
	windGust := 0.0
	hasWind := false
	hasGust := false

	if windVal, exists := currently["windSpeed"]; exists {
		switch v := windVal.(type) {
		case float64:
			windSpeed = v
			hasWind = true
		case int:
			windSpeed = float64(v)
			hasWind = true
		}
	}

	if gustVal, exists := currently["windGust"]; exists {
		switch v := gustVal.(type) {
		case float64:
			windGust = v
			hasGust = true
		case int:
			windGust = float64(v)
			hasGust = true
		}
	}

	// Convert wind from m/s to mph (1 m/s = 2.237 mph) and km/h
	if hasWind {
		windMph := windSpeed * 2.237
		windKmh := windSpeed * 3.6

		if hasGust {
			gustMph := windGust * 2.237
			gustKmh := windGust * 3.6
			result.WriteString(fmt.Sprintf(". Wind: %.0f mph (%.0f km/h) with gusts up to %.0f mph (%.0f km/h)",
				windMph, windKmh, gustMph, gustKmh))

			// Add wind warnings
			if gustMph >= 40 {
				warnings = append(warnings, "HIGH WIND WARNING: Gusts may cause damage and make driving hazardous")
			} else if gustMph >= 25 {
				warnings = append(warnings, "Windy conditions: Exercise caution outdoors")
			}
		} else {
			result.WriteString(fmt.Sprintf(". Wind: %.0f mph (%.0f km/h)", windMph, windKmh))

			if windMph >= 25 {
				warnings = append(warnings, "Windy conditions: Exercise caution outdoors")
			}
		}
	}

	// Get precipitation information
	precipProb := 0.0
	precipIntensity := 0.0
	precipType := ""

	if probVal, exists := currently["precipProbability"]; exists {
		switch v := probVal.(type) {
		case float64:
			precipProb = v * 100 // Convert to percentage
		case int:
			precipProb = float64(v) * 100
		}
	}

	if intensityVal, exists := currently["precipIntensity"]; exists {
		switch v := intensityVal.(type) {
		case float64:
			precipIntensity = v
		case int:
			precipIntensity = float64(v)
		}
	}

	if typeVal, exists := currently["precipType"]; exists {
		precipType = fmt.Sprintf("%v", typeVal)
	}

	// Report precipitation if probability is significant
	if precipProb >= 20 {
		if precipType != "" && precipType != "none" {
			// Capitalize precipitation type
			precipTypeCapitalized := strings.Title(strings.ToLower(precipType))
			result.WriteString(fmt.Sprintf(". %s: %.0f%% chance", precipTypeCapitalized, precipProb))
		} else {
			result.WriteString(fmt.Sprintf(". Precipitation: %.0f%% chance", precipProb))
		}

		if precipIntensity > 0 {
			intensityMmh := precipIntensity // Already in mm/h
			if intensityMmh >= 10 {
				result.WriteString(" (heavy)")
			} else if intensityMmh >= 2.5 {
				result.WriteString(" (moderate)")
			} else {
				result.WriteString(" (light)")
			}
		}
	}

	// Get humidity
	humidity := 0.0
	if humVal, exists := currently["humidity"]; exists {
		switch v := humVal.(type) {
		case float64:
			humidity = v * 100 // Convert to percentage
		case int:
			humidity = float64(v) * 100
		}

		if humidity >= 80 {
			result.WriteString(fmt.Sprintf(". Humidity: %.0f%% (very humid)", humidity))
		} else if humidity <= 30 {
			result.WriteString(fmt.Sprintf(". Humidity: %.0f%% (dry)", humidity))
		}
	}

	// Get UV Index
	uvIndex := 0.0
	if uvVal, exists := currently["uvIndex"]; exists {
		switch v := uvVal.(type) {
		case float64:
			uvIndex = v
		case int:
			uvIndex = float64(v)
		}

		if uvIndex >= 8 {
			warnings = append(warnings, fmt.Sprintf("VERY HIGH UV INDEX (%.0f): Seek shade, wear sunscreen and protective clothing", uvIndex))
		} else if uvIndex >= 6 {
			warnings = append(warnings, fmt.Sprintf("High UV Index (%.0f): Use sunscreen", uvIndex))
		}
	}

	// Get visibility
	visibility := 0.0
	if visVal, exists := currently["visibility"]; exists {
		switch v := visVal.(type) {
		case float64:
			visibility = v
		case int:
			visibility = float64(v)
		}

		if visibility < 1 {
			visMiles := visibility * 0.621371
			warnings = append(warnings, fmt.Sprintf("Low visibility: %.1f km (%.1f miles)", visibility, visMiles))
		}
	}

	// Try to get daily high/low
	if daily, hasDaily := data["daily"].(map[string]interface{}); hasDaily {
		if dailyData, hasData := daily["data"].([]interface{}); hasData && len(dailyData) > 0 {
			if today, isMap := dailyData[0].(map[string]interface{}); isMap {
				hasHigh := false
				hasLow := false
				highTemp := 0.0
				lowTemp := 0.0

				if highVal, exists := today["temperatureHigh"]; exists {
					switch v := highVal.(type) {
					case float64:
						highTemp = v
						hasHigh = true
					case int:
						highTemp = float64(v)
						hasHigh = true
					}
				}

				if lowVal, exists := today["temperatureLow"]; exists {
					switch v := lowVal.(type) {
					case float64:
						lowTemp = v
						hasLow = true
					case int:
						lowTemp = float64(v)
						hasLow = true
					}
				}

				if hasHigh && hasLow {
					highF := (highTemp * 9 / 5) + 32
					lowF := (lowTemp * 9 / 5) + 32
					result.WriteString(fmt.Sprintf(". Today's high will be %.0f°F (%.0f°C) and low will be %.0f°F (%.0f°C)",
						highF, highTemp, lowF, lowTemp))
				} else if hasHigh {
					highF := (highTemp * 9 / 5) + 32
					result.WriteString(fmt.Sprintf(". Today's high will be %.0f°F (%.0f°C)", highF, highTemp))
				} else if hasLow {
					lowF := (lowTemp * 9 / 5) + 32
					result.WriteString(fmt.Sprintf(". Today's low will be %.0f°F (%.0f°C)", lowF, lowTemp))
				}
			}
		}
	}

	result.WriteString(".")

	// Add warnings if any
	if len(warnings) > 0 {
		result.WriteString(" ⚠️ ")
		result.WriteString(strings.Join(warnings, ". "))
		result.WriteString(".")
	}

	return result.String()
}

func (tp *TreeProcessor) formatTomorrowWeather(data map[string]interface{}) string {
	// Extract weather forecast for tomorrow from daily data
	var result strings.Builder
	var warnings []string

	// Get daily forecast data
	daily, hasDaily := data["daily"].(map[string]interface{})
	if !hasDaily {
		tp.golem.LogInfo("WeatherFormat (tomorrow): No 'daily' field in response")
		return "unavailable (no forecast data)"
	}

	dailyData, hasData := daily["data"].([]interface{})
	if !hasData || len(dailyData) < 2 {
		tp.golem.LogInfo("WeatherFormat (tomorrow): Not enough daily forecast data (need at least 2 days)")
		return "unavailable (insufficient forecast data)"
	}

	// Get tomorrow's forecast (index 1)
	tomorrow, isMap := dailyData[1].(map[string]interface{})
	if !isMap {
		tp.golem.LogInfo("WeatherFormat (tomorrow): Tomorrow's data is not a map")
		return "unavailable (invalid forecast format)"
	}

	// Extract forecast summary
	summary := ""
	if summaryVal, exists := tomorrow["summary"]; exists {
		summary = fmt.Sprintf("%v", summaryVal)
	}

	// Extract temperature high and low
	hasHigh := false
	hasLow := false
	highTemp := 0.0
	lowTemp := 0.0

	if highVal, exists := tomorrow["temperatureHigh"]; exists {
		switch v := highVal.(type) {
		case float64:
			highTemp = v
			hasHigh = true
		case int:
			highTemp = float64(v)
			hasHigh = true
		}
	}

	if lowVal, exists := tomorrow["temperatureLow"]; exists {
		switch v := lowVal.(type) {
		case float64:
			lowTemp = v
			hasLow = true
		case int:
			lowTemp = float64(v)
			hasLow = true
		}
	}

	// Build the response
	if summary != "" {
		result.WriteString("Tomorrow will be ")
		result.WriteString(strings.ToLower(summary))
	} else {
		result.WriteString("Tomorrow's forecast")
	}

	if hasHigh && hasLow {
		highF := (highTemp * 9 / 5) + 32
		lowF := (lowTemp * 9 / 5) + 32
		result.WriteString(fmt.Sprintf(" with a high of %.0f°F (%.0f°C) and a low of %.0f°F (%.0f°C)",
			highF, highTemp, lowF, lowTemp))
	} else if hasHigh {
		highF := (highTemp * 9 / 5) + 32
		result.WriteString(fmt.Sprintf(" with a high of %.0f°F (%.0f°C)", highF, highTemp))
	} else if hasLow {
		lowF := (lowTemp * 9 / 5) + 32
		result.WriteString(fmt.Sprintf(" with a low of %.0f°F (%.0f°C)", lowF, lowTemp))
	}

	// Get wind information for tomorrow
	windSpeed := 0.0
	windGust := 0.0
	hasWind := false
	hasGust := false

	if windVal, exists := tomorrow["windSpeed"]; exists {
		switch v := windVal.(type) {
		case float64:
			windSpeed = v
			hasWind = true
		case int:
			windSpeed = float64(v)
			hasWind = true
		}
	}

	if gustVal, exists := tomorrow["windGust"]; exists {
		switch v := gustVal.(type) {
		case float64:
			windGust = v
			hasGust = true
		case int:
			windGust = float64(v)
			hasGust = true
		}
	}

	// Convert wind from m/s to mph and km/h
	if hasWind {
		windMph := windSpeed * 2.237
		windKmh := windSpeed * 3.6

		if hasGust {
			gustMph := windGust * 2.237
			gustKmh := windGust * 3.6
			result.WriteString(fmt.Sprintf(". Wind: %.0f mph (%.0f km/h) with gusts up to %.0f mph (%.0f km/h)",
				windMph, windKmh, gustMph, gustKmh))

			// Add wind warnings
			if gustMph >= 40 {
				warnings = append(warnings, "HIGH WIND WARNING: Gusts may cause damage and make driving hazardous")
			} else if gustMph >= 25 {
				warnings = append(warnings, "Windy conditions expected: Exercise caution outdoors")
			}
		} else {
			result.WriteString(fmt.Sprintf(". Wind: %.0f mph (%.0f km/h)", windMph, windKmh))

			if windMph >= 25 {
				warnings = append(warnings, "Windy conditions expected: Exercise caution outdoors")
			}
		}
	}

	// Get precipitation probability
	precipProb := 0.0
	precipType := ""

	if probVal, exists := tomorrow["precipProbability"]; exists {
		switch v := probVal.(type) {
		case float64:
			precipProb = v * 100 // Convert to percentage
		case int:
			precipProb = float64(v) * 100
		}
	}

	if typeVal, exists := tomorrow["precipType"]; exists {
		precipType = fmt.Sprintf("%v", typeVal)
	}

	// Report precipitation if probability is significant
	if precipProb >= 20 {
		if precipType != "" && precipType != "none" {
			// Capitalize precipitation type
			precipTypeCapitalized := strings.Title(strings.ToLower(precipType))
			result.WriteString(fmt.Sprintf(". %s: %.0f%% chance", precipTypeCapitalized, precipProb))
		} else {
			result.WriteString(fmt.Sprintf(". Precipitation: %.0f%% chance", precipProb))
		}
	}

	// Get UV Index for tomorrow
	uvIndex := 0.0
	if uvVal, exists := tomorrow["uvIndex"]; exists {
		switch v := uvVal.(type) {
		case float64:
			uvIndex = v
		case int:
			uvIndex = float64(v)
		}

		if uvIndex >= 8 {
			warnings = append(warnings, fmt.Sprintf("VERY HIGH UV INDEX (%.0f): Seek shade, wear sunscreen and protective clothing", uvIndex))
		} else if uvIndex >= 6 {
			warnings = append(warnings, fmt.Sprintf("High UV Index (%.0f): Use sunscreen", uvIndex))
		}
	}

	result.WriteString(".")

	// Add warnings if any
	if len(warnings) > 0 {
		result.WriteString(" ⚠️ ")
		result.WriteString(strings.Join(warnings, ". "))
		result.WriteString(".")
	}

	return result.String()
}

func (tp *TreeProcessor) processSentenceTag(_ *ASTNode, content string) string {
	// Process sentence tag - capitalize first letter of each sentence
	content = strings.TrimSpace(content)

	if content == "" {
		return ""
	}

	// Capitalize sentences using the existing method
	processedContent := tp.golem.capitalizeSentences(content)

	tp.golem.LogDebug("Sentence tag: '%s' -> '%s'", content, processedContent)
	return processedContent
}

func (tp *TreeProcessor) processWordTag(_ *ASTNode, content string) string {
	// Process word tag - capitalize first letter of each word (Title Case)
	content = strings.TrimSpace(content)

	if content == "" {
		return ""
	}

	// Capitalize words using the existing method
	processedContent := tp.golem.capitalizeWords(content)

	tp.golem.LogDebug("Word tag: '%s' -> '%s'", content, processedContent)
	return processedContent
}

func (tp *TreeProcessor) processDateTag(node *ASTNode, _ string) string {
	// Date tag - current date
	tp.trackMetric("data") // Track data processor usage
	format := "Monday, January 2, 2006"
	if f, exists := node.Attributes["format"]; exists {
		format = f
	}
	// Convert C-style or alternative formats to Go time format
	goFormat := tp.golem.convertToGoTimeFormat(format)
	return time.Now().Format(goFormat)
}

func (tp *TreeProcessor) processTimeTag(node *ASTNode, _ string) string {
	// Time tag - current time
	tp.trackMetric("data") // Track data processor usage
	defaultFormat := "3:04 PM"
	format := defaultFormat
	if f, exists := node.Attributes["format"]; exists {
		format = f
	}
	// Convert C-style or alternative formats to Go time format
	goFormat := tp.golem.convertToGoTimeFormat(format)

	// If format conversion didn't change anything and it's not a recognized Go format,
	// fall back to default
	if goFormat == format && !tp.golem.looksLikeGoTimeFormat(format) && format != defaultFormat {
		goFormat = defaultFormat
	}

	return time.Now().Format(goFormat)
}

// System tags

func (tp *TreeProcessor) processSizeTag(_ *ASTNode, _ string) string {
	// Size tag - knowledge base size
	if tp.ctx != nil && tp.ctx.KnowledgeBase != nil {
		return strconv.Itoa(tp.ctx.KnowledgeBase.categoryCount())
	}
	return "0"
}

func (tp *TreeProcessor) processVersionTag(_ *ASTNode, _ string) string {
	// Version tag - bot version
	if tp.ctx != nil && tp.ctx.KnowledgeBase != nil {
		if version, exists := tp.ctx.KnowledgeBase.Properties["version"]; exists {
			return version
		}
	}
	return "1.0"
}

// processIdTag returns the client ID, as AIML 2.0 specifies: the ID of the
// session the template is answering, or "" without a session
func (tp *TreeProcessor) processIdTag(_ *ASTNode, _ string) string {
	if tp.ctx != nil && tp.ctx.Session != nil {
		return tp.ctx.Session.ID
	}
	return ""
}

// processBotIdTag returns the bot's ID: its "id" property, or "golem" when
// that is not set. This extension keeps what <id/> returned before it
// followed the spec.
func (tp *TreeProcessor) processBotIdTag(_ *ASTNode, _ string) string {
	if tp.ctx != nil && tp.ctx.KnowledgeBase != nil {
		if id, exists := tp.ctx.KnowledgeBase.Properties["id"]; exists {
			return id
		}
	}
	return "golem"
}

func (tp *TreeProcessor) processRequestTag(node *ASTNode, _ string) string {
	// Request tag - previous request
	// Index 1 = most recent, index 2 = 2nd most recent, etc.
	index := 1
	if idx, exists := node.Attributes["index"]; exists {
		if parsed, err := strconv.Atoi(idx); err == nil {
			index = parsed
		}
	}

	if tp.ctx != nil && tp.ctx.Session != nil {
		// Use GetRequestByIndex which properly handles reverse indexing
		return tp.ctx.Session.GetRequestByIndex(index)
	}
	return ""
}

func (tp *TreeProcessor) processResponseTag(node *ASTNode, _ string) string {
	// Response tag - previous response (index 1 = most recent)
	index := 1
	if idx, exists := node.Attributes["index"]; exists {
		if parsed, err := strconv.Atoi(idx); err == nil {
			index = parsed
		}
	}

	if tp.ctx != nil && tp.ctx.Session != nil {
		// Use GetResponseByIndex which handles the index conversion correctly
		// (1-based where 1 is most recent, stored at end of array)
		return tp.ctx.Session.GetResponseByIndex(index)
	}
	return ""
}

// Text processing tags

func (tp *TreeProcessor) processNormalizeTag(_ *ASTNode, content string) string {
	// Normalize tag - text normalization
	// Process the content directly using the normalization function
	return tp.golem.normalizeTextForOutput(content)
}

func (tp *TreeProcessor) processDenormalizeTag(_ *ASTNode, content string) string {
	// Denormalize tag - text denormalization
	// Process the content directly using the denormalization function
	return tp.golem.denormalizeText(content)
}

// Learning tags

func (tp *TreeProcessor) processUnlearnTag(_ *ASTNode, content string) string {
	// Unlearn tag - remove learned categories
	tp.golem.unlearnSessionCategories(content, tp.ctx)
	return ""
}

func (tp *TreeProcessor) processUnlearnfTag(_ *ASTNode, content string) string {
	// Unlearnf tag - remove categories from persistent storage
	if tp.golem.aimlKB == nil {
		tp.golem.LogWarn("Unlearnf: No knowledge base available")
		return ""
	}

	tp.golem.LogInfo("Processing unlearnf: '%s'", content)

	// Parse the AIML content within the unlearnf tag
	categories, err := tp.golem.parseLearnContent(content)
	if err != nil {
		tp.golem.LogError("Failed to parse unlearnf content: %v", err)
		return ""
	}

	// Remove categories from persistent knowledge base
	for _, category := range categories {
		err := tp.golem.removePersistentCategory(category)
		if err != nil {
			tp.golem.LogInfo("Failed to remove persistent category: %v", err)
		}
	}

	// Unlearnf tags don't output content
	return ""
}

// Advanced tags

func (tp *TreeProcessor) processVarTag(node *ASTNode, content string) string {
	// Var tag - variable declaration
	// Similar to set tag but for variable declaration
	name, exists := node.Attributes["name"]
	if !exists {
		return content
	}

	// Process the content to get the value
	value := content

	// Set the variable in context
	if tp.ctx != nil {
		if tp.ctx.LocalVars == nil {
			tp.ctx.LocalVars = make(map[string]string)
		}
		tp.ctx.LocalVars[name] = value
	}

	// Var tags don't output content
	return ""
}

func (tp *TreeProcessor) processGossipTag(_ *ASTNode, _ string) string {
	// Gossip tag - gossip processing
	// For now, return empty string as this functionality needs to be implemented
	return ""
}

func (tp *TreeProcessor) processJavascriptTag(_ *ASTNode, _ string) string {
	// Javascript tag - JavaScript execution
	// For now, return empty string as this functionality needs to be implemented
	return ""
}

func (tp *TreeProcessor) processSystemTag(_ *ASTNode, _ string) string {
	// System tag - system command execution
	// For now, return empty string as this functionality needs to be implemented
	return ""
}

func (tp *TreeProcessor) processSubjTag(_ *ASTNode, content string) string {
	// Subj tag - RDF subject
	// Process content and add trailing space for RDF readability
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	return content + " "
}

func (tp *TreeProcessor) processPredTag(_ *ASTNode, content string) string {
	// Pred tag - RDF predicate
	// Process content and add trailing space for RDF readability
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	return content + " "
}

func (tp *TreeProcessor) processObjTag(_ *ASTNode, content string) string {
	// Obj tag - RDF object
	// Process content without trailing space (it's the last element)
	content = strings.TrimSpace(content)
	return content
}

func (tp *TreeProcessor) processUniqTag(_ *ASTNode, content string) string {
	// Uniq tag - RDF unique/triple container
	// Process content and format with proper spacing
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	// Clean up multiple spaces and format for readability
	words := strings.Fields(content)
	if len(words) == 0 {
		return ""
	}

	return strings.Join(words, " ")
}

// Helper method for random number generation
func (g *Golem) randomIntTree(max int) int {
	// This would use the existing random number generation from the Golem instance
	// For now, return a simple implementation
	return int(time.Now().UnixNano() % int64(max))
}
