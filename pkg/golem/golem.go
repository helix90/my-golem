package golem

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
	LogLevelTrace
)

// ContextConfig represents configuration for context management
type ContextConfig struct {
	MaxThatDepth         int     // Maximum depth for that history (default: 20)
	MaxRequestDepth      int     // Maximum depth for request history (default: 20)
	MaxResponseDepth     int     // Maximum depth for response history (default: 20)
	MaxTotalContext      int     // Maximum total context items (default: 100)
	CompressionThreshold int     // Threshold for context compression (default: 50)
	WeightDecay          float64 // Weight decay factor for older context (default: 0.9)
	EnableCompression    bool    // Enable context compression (default: true)
	EnableAnalytics      bool    // Enable context analytics (default: true)
	EnablePruning        bool    // Enable smart context pruning (default: true)
}

// ContextItem represents a single context item with metadata
type ContextItem struct {
	Content    string                 `json:"content"`
	Type       string                 `json:"type"` // "that", "request", "response"
	Index      int                    `json:"index"`
	Weight     float64                `json:"weight"`
	Tags       []string               `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  string                 `json:"created_at"`
	LastUsed   string                 `json:"last_used"`
	UsageCount int                    `json:"usage_count"`
}

// ContextAnalytics represents analytics data for context usage
type ContextAnalytics struct {
	TotalItems       int            `json:"total_items"`
	ThatItems        int            `json:"that_items"`
	RequestItems     int            `json:"request_items"`
	ResponseItems    int            `json:"response_items"`
	AverageWeight    float64        `json:"average_weight"`
	MostUsedItems    []string       `json:"most_used_items"`
	LeastUsedItems   []string       `json:"least_used_items"`
	TagDistribution  map[string]int `json:"tag_distribution"`
	MemoryUsage      int            `json:"memory_usage_bytes"`
	CompressionRatio float64        `json:"compression_ratio"`
	LastPruned       string         `json:"last_pruned"`
	PruningCount     int            `json:"pruning_count"`
}

// TemplateProcessingMetrics represents metrics for template processing
type TemplateProcessingMetrics struct {
	TotalProcessed     int                `json:"total_processed"`
	AverageProcessTime float64            `json:"average_process_time_ms"`
	CacheHits          int                `json:"cache_hits"`
	CacheMisses        int                `json:"cache_misses"`
	CacheHitRate       float64            `json:"cache_hit_rate"`
	TagProcessingTimes map[string]float64 `json:"tag_processing_times"`
	ErrorCount         int                `json:"error_count"`
	LastProcessed      string             `json:"last_processed"`
	MemoryPeak         int                `json:"memory_peak_bytes"`
	ParallelOps        int                `json:"parallel_operations"`
}

// TemplateCache represents a cache for processed templates
type TemplateCache struct {
	Cache      map[string]string `json:"cache"`
	Timestamps map[string]string `json:"timestamps"`
	Hits       map[string]int    `json:"hits"`
	MaxSize    int               `json:"max_size"`
	TTL        int64             `json:"ttl_seconds"`
	// Mutex for thread safety
	mutex sync.RWMutex
}

// RegexCache represents a cache for compiled regex patterns
type RegexCache struct {
	Patterns    map[string]*regexp.Regexp `json:"-"` // Don't serialize compiled regexes
	Hits        map[string]int            `json:"hits"`
	Misses      int                       `json:"misses"`
	MaxSize     int                       `json:"max_size"`
	TTL         int64                     `json:"ttl_seconds"`
	Timestamps  map[string]time.Time      `json:"timestamps"`
	AccessOrder []string                  `json:"access_order"` // For LRU eviction
	mutex       sync.RWMutex              // Mutex for thread-safe access
}

// TextNormalizationCache represents a cache for text normalization results
type TextNormalizationCache struct {
	Results     map[string]string    `json:"results"`
	Hits        map[string]int       `json:"hits"`
	Misses      int                  `json:"misses"`
	MaxSize     int                  `json:"max_size"`
	TTL         int64                `json:"ttl_seconds"`
	Timestamps  map[string]time.Time `json:"timestamps"`
	AccessOrder []string             `json:"access_order"` // For LRU eviction
	// Mutex for thread safety
	mutex sync.RWMutex
}

// VariableResolutionCache represents a cache for variable resolution results
type VariableResolutionCache struct {
	Results     map[string]string    `json:"results"`
	Hits        map[string]int       `json:"hits"`
	Misses      int                  `json:"misses"`
	MaxSize     int                  `json:"max_size"`
	TTL         int64                `json:"ttl_seconds"`
	Timestamps  map[string]time.Time `json:"timestamps"`
	AccessOrder []string             `json:"access_order"` // For LRU eviction
	// Scope tracking for cache invalidation
	ScopeHashes map[string]string `json:"scope_hashes"` // Maps cache key to scope hash
}

// TemplateTagProcessingCache represents a cache for template tag processing results
type TemplateTagProcessingCache struct {
	Results     map[string]string    `json:"results"`
	Hits        map[string]int       `json:"hits"`
	Misses      int                  `json:"misses"`
	MaxSize     int                  `json:"max_size"`
	TTL         int64                `json:"ttl_seconds"`
	Timestamps  map[string]time.Time `json:"timestamps"`
	AccessOrder []string             `json:"access_order"` // For LRU eviction
	// Tag type tracking for cache invalidation
	TagTypes map[string]string `json:"tag_types"` // Maps cache key to tag type
	// Context tracking for cache invalidation
	ContextHashes map[string]string `json:"context_hashes"` // Maps cache key to context hash
	// Mutex for thread safety
	mutex sync.RWMutex
}

// PatternMatchingCache represents a cache for pattern matching results
type PatternMatchingCache struct {
	// Pattern priority cache
	PatternPriorities map[string]PatternPriorityInfo `json:"pattern_priorities"`
	// Wildcard match results cache
	WildcardMatches map[string]WildcardMatchResult `json:"wildcard_matches"`
	// Set regex cache
	SetRegexes map[string]string `json:"set_regexes"`
	// Exact match key cache
	ExactMatchKeys map[string]string `json:"exact_match_keys"`
	// Cache statistics
	Hits        map[string]int       `json:"hits"`
	Misses      int                  `json:"misses"`
	MaxSize     int                  `json:"max_size"`
	TTL         int64                `json:"ttl_seconds"`
	Timestamps  map[string]time.Time `json:"timestamps"`
	AccessOrder []string             `json:"access_order"` // For LRU eviction
	// Knowledge base state tracking for invalidation
	KnowledgeBaseHash string `json:"knowledge_base_hash"`
	// Set state tracking for invalidation
	SetHashes map[string]string `json:"set_hashes"` // Maps set name to content hash
	// Mutex for thread safety
	mutex sync.RWMutex
}

// WildcardMatchResult represents the result of a wildcard pattern match
type WildcardMatchResult struct {
	Matched   bool              `json:"matched"`
	Wildcards map[string]string `json:"wildcards"`
	Pattern   string            `json:"pattern"`
	Input     string            `json:"input"`
	Regex     string            `json:"regex"`
}

// TemplateProcessingConfig represents configuration for template processing
type TemplateProcessingConfig struct {
	EnableCaching     bool  `json:"enable_caching"`
	CacheSize         int   `json:"cache_size"`
	CacheTTL          int64 `json:"cache_ttl_seconds"`
	EnableParallel    bool  `json:"enable_parallel"`
	MaxParallelOps    int   `json:"max_parallel_operations"`
	EnableMetrics     bool  `json:"enable_metrics"`
	EnableValidation  bool  `json:"enable_validation"`
	EnableDebugging   bool  `json:"enable_debugging"`
	MemoryLimit       int   `json:"memory_limit_bytes"`
	ProcessingTimeout int64 `json:"processing_timeout_ms"`
}

// ChatSession represents a single chat session
type ChatSession struct {
	ID              string
	Variables       map[string]string
	History         []string
	CreatedAt       string
	LastActivity    string
	Topic           string   // Current conversation topic
	ThatHistory     []string // History of bot responses for that matching
	RequestHistory  []string // History of user requests for <request> tag
	ResponseHistory []string // History of bot responses for <response> tag

	// Enhanced context management
	ContextConfig   *ContextConfig         // Context configuration
	ContextWeights  map[string]float64     // Weights for different context levels
	ContextUsage    map[string]int         // Usage count for each context item
	ContextTags     map[string][]string    // Tags for context categorization
	ContextMetadata map[string]interface{} // Additional context metadata

	// Session-specific learning
	LearnedCategories []Category            // Categories learned in this session
	LearningStats     *SessionLearningStats // Learning statistics for this session
}

// SessionLearningStats represents learning statistics for a session
type SessionLearningStats struct {
	TotalLearned     int            `json:"total_learned"`
	TotalUnlearned   int            `json:"total_unlearned"`
	LastLearned      time.Time      `json:"last_learned"`
	LastUnlearned    time.Time      `json:"last_unlearned"`
	LearningSources  map[string]int `json:"learning_sources"` // Source -> count
	PatternTypes     map[string]int `json:"pattern_types"`    // Pattern type -> count
	TemplateLengths  []int          `json:"template_lengths"` // Template length distribution
	ValidationErrors int            `json:"validation_errors"`
	LearningRate     float64        `json:"learning_rate"` // Categories per minute
}

// Golem represents the main library instance
//
// CRITICAL ARCHITECTURAL NOTE:
// This struct maintains state across multiple operations:
// - aimlKB: Loaded AIML knowledge base (persists across commands)
// - sessions: Active chat sessions (persists across commands)
// - currentID: Currently active session (persists across commands)
//
// CLI USAGE PATTERN:
// - Single command mode: Creates new instance per command (state lost)
// - Interactive mode: Single persistent instance (state preserved)
// - Library mode: User manages instance lifecycle (state controlled by user)
//
// DO NOT modify the state management without understanding the implications
// for all three usage patterns.
type Golem struct {
	verbose   bool
	logLevel  LogLevel
	logger    *log.Logger
	aimlKB    *AIMLKnowledgeBase
	sessions  map[string]*ChatSession
	currentID string
	sessionID int
	oobMgr    *OOBManager
	sraixMgr  *SRAIXManager
	// Mutex for thread-safe session management
	sessionMutex sync.RWMutex
	// Text processing components
	sentenceSplitter     *SentenceSplitter
	wordBoundaryDetector *WordBoundaryDetector
	// Template processing components
	templateCache   *TemplateCache
	templateConfig  *TemplateProcessingConfig
	templateMetrics *TemplateProcessingMetrics
	// Regex compilation caches
	patternRegexCache  *RegexCache
	tagProcessingCache *RegexCache
	normalizationCache *RegexCache
	// Consolidated template processor
	consolidatedProcessor *ConsolidatedTemplateProcessor
	// Text normalization result cache
	textNormalizationCache *TextNormalizationCache
	// Variable resolution cache
	variableResolutionCache *VariableResolutionCache
	// That pattern cache
	thatPatternCache *ThatPatternCache
	// Template tag processing cache
	templateTagProcessingCache *TemplateTagProcessingCache
	// Pattern matching cache
	patternMatchingCache *PatternMatchingCache
	// Persistent learning components
	persistentLearning *PersistentLearningManager
	// Enhanced context resolution components
	fuzzyMatcher    *FuzzyContextMatcher
	semanticMatcher *SemanticContextMatcher
	// Random seed for deterministic shuffling
	randomSeed int64
	// Tree-based processing components
	treeProcessor     *TreeProcessor
	useTreeProcessing bool // Feature flag for tree-based processing
}

// NewRegexCache creates a new regex cache
func NewRegexCache(maxSize int, ttlSeconds int64) *RegexCache {
	return &RegexCache{
		Patterns:    make(map[string]*regexp.Regexp),
		Hits:        make(map[string]int),
		Misses:      0,
		MaxSize:     maxSize,
		TTL:         ttlSeconds,
		Timestamps:  make(map[string]time.Time),
		AccessOrder: make([]string, 0),
	}
}

// GetCompiledRegex returns a compiled regex pattern from cache or compiles and caches it
func (cache *RegexCache) GetCompiledRegex(pattern string) (*regexp.Regexp, error) {
	// Check if pattern exists in cache
	cache.mutex.RLock()
	if compiled, exists := cache.Patterns[pattern]; exists {
		// Check TTL
		if time.Since(cache.Timestamps[pattern]).Seconds() < float64(cache.TTL) {
			cache.mutex.RUnlock()
			// Update access order for LRU (requires write lock)
			cache.mutex.Lock()
			cache.updateAccessOrder(pattern)
			cache.Hits[pattern]++
			cache.mutex.Unlock()
			return compiled, nil
		}
		cache.mutex.RUnlock()
		// TTL expired, remove from cache (requires write lock)
		cache.mutex.Lock()
		cache.removePattern(pattern)
		cache.mutex.Unlock()
	} else {
		cache.mutex.RUnlock()
	}

	// Compile the pattern
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		cache.mutex.Lock()
		cache.Misses++
		cache.mutex.Unlock()
		return nil, err
	}

	// Cache the compiled pattern
	cache.mutex.Lock()
	cache.setPattern(pattern, compiled)
	cache.Misses++
	cache.mutex.Unlock()
	return compiled, nil
}

// updateAccessOrder updates the LRU access order
// Note: This method assumes the caller holds the appropriate lock
func (cache *RegexCache) updateAccessOrder(pattern string) {
	// Remove from current position
	for i, p := range cache.AccessOrder {
		if p == pattern {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
	// Add to end (most recently used)
	cache.AccessOrder = append(cache.AccessOrder, pattern)
}

// setPattern adds a pattern to the cache with LRU eviction
// Note: This method assumes the caller holds the appropriate lock
func (cache *RegexCache) setPattern(pattern string, compiled *regexp.Regexp) {
	// Evict if cache is full
	if len(cache.Patterns) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.Patterns[pattern] = compiled
	cache.Timestamps[pattern] = time.Now()
	cache.updateAccessOrder(pattern)
}

// removePattern removes a pattern from the cache
// Note: This method assumes the caller holds the appropriate lock
func (cache *RegexCache) removePattern(pattern string) {
	delete(cache.Patterns, pattern)
	delete(cache.Timestamps, pattern)
	delete(cache.Hits, pattern)

	// Remove from access order
	for i, p := range cache.AccessOrder {
		if p == pattern {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
}

// evictLRU removes the least recently used pattern
// Note: This method assumes the caller holds the appropriate lock
func (cache *RegexCache) evictLRU() {
	if len(cache.AccessOrder) == 0 {
		return
	}

	// Remove the first (oldest) pattern
	oldestPattern := cache.AccessOrder[0]
	cache.removePattern(oldestPattern)
}

// GetCacheStats returns cache statistics
func (cache *RegexCache) GetCacheStats() map[string]interface{} {
	totalRequests := cache.Misses
	for _, hits := range cache.Hits {
		totalRequests += hits
	}

	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(len(cache.Hits)) / float64(totalRequests)
	}

	return map[string]interface{}{
		"patterns":       len(cache.Patterns),
		"max_size":       cache.MaxSize,
		"ttl_seconds":    cache.TTL,
		"hits":           cache.Hits,
		"misses":         cache.Misses,
		"hit_rate":       hitRate,
		"total_requests": totalRequests,
	}
}

// ClearCache clears the regex cache
func (cache *RegexCache) ClearCache() {
	cache.Patterns = make(map[string]*regexp.Regexp)
	cache.Hits = make(map[string]int)
	cache.Misses = 0
	cache.Timestamps = make(map[string]time.Time)
	cache.AccessOrder = make([]string, 0)
}

// NewTextNormalizationCache creates a new text normalization cache
func NewTextNormalizationCache(maxSize int, ttlSeconds int64) *TextNormalizationCache {
	return &TextNormalizationCache{
		Results:     make(map[string]string),
		Hits:        make(map[string]int),
		Misses:      0,
		MaxSize:     maxSize,
		TTL:         ttlSeconds,
		Timestamps:  make(map[string]time.Time),
		AccessOrder: make([]string, 0),
	}
}

// GetNormalizedText returns a normalized text result from cache or normalizes and caches it
func (cache *TextNormalizationCache) GetNormalizedText(golem *Golem, input string, normalizationType string) (string, error) {
	// Create cache key with normalization type
	cacheKey := normalizationType + ":" + input

	cache.mutex.RLock()
	if result, exists := cache.Results[cacheKey]; exists {
		// Check TTL
		if time.Since(cache.Timestamps[cacheKey]).Seconds() < float64(cache.TTL) {
			cache.mutex.RUnlock()
			// Need write lock to update access order
			cache.mutex.Lock()
			cache.updateAccessOrder(cacheKey)
			cache.Hits[cacheKey]++
			cache.mutex.Unlock()
			return result, nil
		}
		// TTL expired, remove from cache
		cache.mutex.RUnlock()
		cache.mutex.Lock()
		cache.removeResult(cacheKey)
		cache.mutex.Unlock()
	} else {
		cache.mutex.RUnlock()
	}

	// Normalize the text based on type
	var result string

	switch normalizationType {
	case "NormalizePattern":
		result = NormalizePattern(input)
		// Apply loaded substitutions for pattern normalization
		if golem != nil && golem.aimlKB != nil && len(golem.aimlKB.Substitutions) > 0 {
			result = golem.applyLoadedSubstitutions(result)
		}
	case "NormalizeForMatchingCasePreserving":
		result = NormalizeForMatchingCasePreserving(input)
	case "NormalizeThatPattern":
		result = NormalizeThatPattern(input)
	case "normalizeForMatching":
		if golem != nil {
			result = golem.normalizeForMatchingWithSubstitutions(input)
		} else {
			result = normalizeForMatching(input)
		}
	case "expandContractions":
		result = expandContractions(input)
	default:
		return "", fmt.Errorf("unknown normalization type: %s", normalizationType)
	}

	// Cache the result
	cache.mutex.Lock()
	cache.setResult(cacheKey, result)
	cache.Misses++
	cache.mutex.Unlock()
	return result, nil
}

// updateAccessOrder updates the LRU access order
func (cache *TextNormalizationCache) updateAccessOrder(key string) {
	// Remove from current position
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
	// Add to end (most recently used)
	cache.AccessOrder = append(cache.AccessOrder, key)
}

// setResult adds a result to the cache with LRU eviction
// Note: This method assumes the caller holds the appropriate lock
func (cache *TextNormalizationCache) setResult(key string, result string) {
	// Evict if cache is full
	if len(cache.Results) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.Results[key] = result
	cache.Timestamps[key] = time.Now()
	cache.updateAccessOrder(key)
}

// removeResult removes a result from the cache
func (cache *TextNormalizationCache) removeResult(key string) {
	delete(cache.Results, key)
	delete(cache.Timestamps, key)
	delete(cache.Hits, key)

	// Remove from access order
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
}

// evictLRU removes the least recently used result
func (cache *TextNormalizationCache) evictLRU() {
	if len(cache.AccessOrder) == 0 {
		return
	}

	// Remove the first (oldest) result
	oldestKey := cache.AccessOrder[0]
	cache.removeResult(oldestKey)
}

// GetCacheStats returns cache statistics
func (cache *TextNormalizationCache) GetCacheStats() map[string]interface{} {
	totalRequests := cache.Misses
	for _, hits := range cache.Hits {
		totalRequests += hits
	}

	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(len(cache.Hits)) / float64(totalRequests)
	}

	return map[string]interface{}{
		"results":        len(cache.Results),
		"max_size":       cache.MaxSize,
		"ttl_seconds":    cache.TTL,
		"hits":           cache.Hits,
		"misses":         cache.Misses,
		"hit_rate":       hitRate,
		"total_requests": totalRequests,
	}
}

// ClearCache clears the text normalization cache
func (cache *TextNormalizationCache) ClearCache() {
	cache.Results = make(map[string]string)
	cache.Hits = make(map[string]int)
	cache.Misses = 0
	cache.Timestamps = make(map[string]time.Time)
	cache.AccessOrder = make([]string, 0)
}

// NewVariableResolutionCache creates a new variable resolution cache
func NewVariableResolutionCache(maxSize int, ttlSeconds int64) *VariableResolutionCache {
	return &VariableResolutionCache{
		Results:     make(map[string]string),
		Hits:        make(map[string]int),
		Misses:      0,
		MaxSize:     maxSize,
		TTL:         ttlSeconds,
		Timestamps:  make(map[string]time.Time),
		AccessOrder: make([]string, 0),
		ScopeHashes: make(map[string]string),
	}
}

// GetResolvedVariable returns a resolved variable value from cache or resolves and caches it
func (cache *VariableResolutionCache) GetResolvedVariable(varName string, ctx *VariableContext) (string, bool) {
	// Create cache key with variable name and scope context
	cacheKey := cache.generateCacheKey(varName, ctx)

	// Check if result exists in cache
	if result, exists := cache.Results[cacheKey]; exists {
		// Check TTL
		if time.Since(cache.Timestamps[cacheKey]).Seconds() < float64(cache.TTL) {
			// Check if scope has changed (cache invalidation)
			currentScopeHash := cache.generateScopeHash(ctx)
			if cache.ScopeHashes[cacheKey] == currentScopeHash {
				// Update access order for LRU
				cache.updateAccessOrder(cacheKey)
				cache.Hits[cacheKey]++
				return result, true
			}
			// Scope changed, remove from cache
			cache.removeResult(cacheKey)
		} else {
			// TTL expired, remove from cache
			cache.removeResult(cacheKey)
		}
	}

	// Variable not found in cache
	cache.Misses++
	return "", false
}

// SetResolvedVariable caches a resolved variable value
func (cache *VariableResolutionCache) SetResolvedVariable(varName string, value string, ctx *VariableContext) {
	cacheKey := cache.generateCacheKey(varName, ctx)
	scopeHash := cache.generateScopeHash(ctx)

	// Cache the result
	cache.setResult(cacheKey, value, scopeHash)
}

// generateCacheKey creates a cache key for variable resolution
func (cache *VariableResolutionCache) generateCacheKey(varName string, ctx *VariableContext) string {
	// Include variable name and session ID for uniqueness
	sessionID := ""
	if ctx.Session != nil {
		sessionID = ctx.Session.ID
	}
	return fmt.Sprintf("%s:%s", varName, sessionID)
}

// generateScopeHash creates a hash of the variable scope for cache invalidation
func (cache *VariableResolutionCache) generateScopeHash(ctx *VariableContext) string {
	var scopeData []string

	// Include local variables
	if ctx.LocalVars != nil {
		for k, v := range ctx.LocalVars {
			scopeData = append(scopeData, fmt.Sprintf("local:%s=%s", k, v))
		}
	}

	// Include session variables
	if ctx.Session != nil && ctx.Session.Variables != nil {
		for k, v := range ctx.Session.Variables {
			scopeData = append(scopeData, fmt.Sprintf("session:%s=%s", k, v))
		}
	}

	// Include global variables
	if ctx.KnowledgeBase != nil && ctx.KnowledgeBase.Variables != nil {
		for k, v := range ctx.KnowledgeBase.Variables {
			scopeData = append(scopeData, fmt.Sprintf("global:%s=%s", k, v))
		}
	}

	// Include properties
	if ctx.KnowledgeBase != nil && ctx.KnowledgeBase.Properties != nil {
		for k, v := range ctx.KnowledgeBase.Properties {
			scopeData = append(scopeData, fmt.Sprintf("properties:%s=%s", k, v))
		}
	}

	// Include current topic (important for topic-scoped variables)
	if ctx.Session != nil {
		currentTopic := ctx.Session.GetSessionTopic()
		if currentTopic != "" {
			scopeData = append(scopeData, fmt.Sprintf("topic:%s", currentTopic))
		}
	}

	// Sort for consistent hashing
	sort.Strings(scopeData)
	return strings.Join(scopeData, "|")
}

// updateAccessOrder updates the LRU access order
func (cache *VariableResolutionCache) updateAccessOrder(key string) {
	// Remove from current position
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
	// Add to end (most recently used)
	cache.AccessOrder = append(cache.AccessOrder, key)
}

// setResult adds a result to the cache with LRU eviction
func (cache *VariableResolutionCache) setResult(key string, result string, scopeHash string) {
	// Evict if cache is full
	if len(cache.Results) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.Results[key] = result
	cache.Timestamps[key] = time.Now()
	cache.ScopeHashes[key] = scopeHash
	cache.updateAccessOrder(key)
}

// removeResult removes a result from the cache
func (cache *VariableResolutionCache) removeResult(key string) {
	delete(cache.Results, key)
	delete(cache.Timestamps, key)
	delete(cache.Hits, key)
	delete(cache.ScopeHashes, key)

	// Remove from access order
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
}

// evictLRU removes the least recently used result
func (cache *VariableResolutionCache) evictLRU() {
	if len(cache.AccessOrder) == 0 {
		return
	}

	// Remove the first (oldest) result
	oldestKey := cache.AccessOrder[0]
	cache.removeResult(oldestKey)
}

// GetCacheStats returns cache statistics
func (cache *VariableResolutionCache) GetCacheStats() map[string]interface{} {
	totalRequests := cache.Misses
	for _, hits := range cache.Hits {
		totalRequests += hits
	}

	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(len(cache.Hits)) / float64(totalRequests)
	}

	return map[string]interface{}{
		"results":        len(cache.Results),
		"max_size":       cache.MaxSize,
		"ttl_seconds":    cache.TTL,
		"hits":           cache.Hits,
		"misses":         cache.Misses,
		"hit_rate":       hitRate,
		"total_requests": totalRequests,
		"scope_hashes":   len(cache.ScopeHashes),
	}
}

// ClearCache clears the variable resolution cache
func (cache *VariableResolutionCache) ClearCache() {
	cache.Results = make(map[string]string)
	cache.Hits = make(map[string]int)
	cache.Misses = 0
	cache.Timestamps = make(map[string]time.Time)
	cache.AccessOrder = make([]string, 0)
	cache.ScopeHashes = make(map[string]string)
}

// NewTemplateTagProcessingCache creates a new template tag processing cache
func NewTemplateTagProcessingCache(maxSize int, ttlSeconds int64) *TemplateTagProcessingCache {
	return &TemplateTagProcessingCache{
		Results:       make(map[string]string),
		Hits:          make(map[string]int),
		Misses:        0,
		MaxSize:       maxSize,
		TTL:           ttlSeconds,
		Timestamps:    make(map[string]time.Time),
		AccessOrder:   make([]string, 0),
		TagTypes:      make(map[string]string),
		ContextHashes: make(map[string]string),
	}
}

// GetProcessedTag returns a processed tag result from cache or processes and caches it
func (cache *TemplateTagProcessingCache) GetProcessedTag(tagType, content string, ctx *VariableContext) (string, bool) {
	// Create cache key with tag type, content, and context
	cacheKey := cache.generateCacheKey(tagType, content, ctx)

	// Check if result exists in cache
	cache.mutex.RLock()
	if result, exists := cache.Results[cacheKey]; exists {
		// Check TTL
		if time.Since(cache.Timestamps[cacheKey]).Seconds() < float64(cache.TTL) {
			// Check if context has changed (cache invalidation)
			currentContextHash := cache.generateContextHash(ctx)
			if cache.ContextHashes[cacheKey] == currentContextHash {
				cache.mutex.RUnlock()
				// Update access order for LRU (requires write lock)
				cache.mutex.Lock()
				cache.updateAccessOrder(cacheKey)
				cache.Hits[cacheKey]++
				cache.mutex.Unlock()
				return result, true
			}
			cache.mutex.RUnlock()
			// Context changed, remove from cache (requires write lock)
			cache.mutex.Lock()
			cache.removeResult(cacheKey)
			cache.mutex.Unlock()
		} else {
			cache.mutex.RUnlock()
			// TTL expired, remove from cache (requires write lock)
			cache.mutex.Lock()
			cache.removeResult(cacheKey)
			cache.mutex.Unlock()
		}
	} else {
		cache.mutex.RUnlock()
	}

	// Tag not found in cache
	cache.mutex.Lock()
	cache.Misses++
	cache.mutex.Unlock()
	return "", false
}

// SetProcessedTag caches a processed tag result
func (cache *TemplateTagProcessingCache) SetProcessedTag(tagType, content, result string, ctx *VariableContext) {
	cacheKey := cache.generateCacheKey(tagType, content, ctx)
	contextHash := cache.generateContextHash(ctx)

	// Cache the result
	cache.mutex.Lock()
	cache.setResult(cacheKey, result, tagType, contextHash)
	cache.mutex.Unlock()
}

// generateCacheKey creates a cache key for template tag processing
func (cache *TemplateTagProcessingCache) generateCacheKey(tagType, content string, ctx *VariableContext) string {
	// Include tag type, content, and session ID for uniqueness
	sessionID := ""
	if ctx.Session != nil {
		sessionID = ctx.Session.ID
	}
	return fmt.Sprintf("%s:%s:%s", tagType, content, sessionID)
}

// generateContextHash creates a hash of the variable context for cache invalidation
func (cache *TemplateTagProcessingCache) generateContextHash(ctx *VariableContext) string {
	var contextData []string

	// Include local variables
	if ctx.LocalVars != nil {
		for k, v := range ctx.LocalVars {
			contextData = append(contextData, fmt.Sprintf("local:%s=%s", k, v))
		}
	}

	// Include session variables
	if ctx.Session != nil && ctx.Session.Variables != nil {
		for k, v := range ctx.Session.Variables {
			contextData = append(contextData, fmt.Sprintf("session:%s=%s", k, v))
		}
	}

	// Include global variables
	if ctx.KnowledgeBase != nil && ctx.KnowledgeBase.Variables != nil {
		for k, v := range ctx.KnowledgeBase.Variables {
			contextData = append(contextData, fmt.Sprintf("global:%s=%s", k, v))
		}
	}

	// Include properties
	if ctx.KnowledgeBase != nil && ctx.KnowledgeBase.Properties != nil {
		for k, v := range ctx.KnowledgeBase.Properties {
			contextData = append(contextData, fmt.Sprintf("properties:%s=%s", k, v))
		}
	}

	// Include arrays, sets, maps, lists state
	if ctx.KnowledgeBase != nil {
		if ctx.KnowledgeBase.Arrays != nil {
			for k, v := range ctx.KnowledgeBase.Arrays {
				contextData = append(contextData, fmt.Sprintf("arrays:%s=%v", k, v))
			}
		}
		if ctx.KnowledgeBase.Sets != nil {
			for k, v := range ctx.KnowledgeBase.Sets {
				contextData = append(contextData, fmt.Sprintf("sets:%s=%v", k, v))
			}
		}
		if ctx.KnowledgeBase.Maps != nil {
			for k, v := range ctx.KnowledgeBase.Maps {
				contextData = append(contextData, fmt.Sprintf("maps:%s=%v", k, v))
			}
		}
		if ctx.KnowledgeBase.Lists != nil {
			for k, v := range ctx.KnowledgeBase.Lists {
				contextData = append(contextData, fmt.Sprintf("lists:%s=%v", k, v))
			}
		}
	}

	// Sort for consistent hashing
	sort.Strings(contextData)
	return strings.Join(contextData, "|")
}

// updateAccessOrder updates the LRU access order
// Note: This method assumes the caller holds the appropriate lock
func (cache *TemplateTagProcessingCache) updateAccessOrder(key string) {
	// Remove from current position
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
	// Add to end (most recently used)
	cache.AccessOrder = append(cache.AccessOrder, key)
}

// setResult adds a result to the cache with LRU eviction
// Note: This method assumes the caller holds the appropriate lock
func (cache *TemplateTagProcessingCache) setResult(key string, result string, tagType string, contextHash string) {
	// Evict if cache is full
	if len(cache.Results) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.Results[key] = result
	cache.Timestamps[key] = time.Now()
	cache.TagTypes[key] = tagType
	cache.ContextHashes[key] = contextHash
	cache.updateAccessOrder(key)
}

// removeResult removes a result from the cache
// Note: This method assumes the caller holds the appropriate lock
func (cache *TemplateTagProcessingCache) removeResult(key string) {
	delete(cache.Results, key)
	delete(cache.Timestamps, key)
	delete(cache.Hits, key)
	delete(cache.TagTypes, key)
	delete(cache.ContextHashes, key)

	// Remove from access order
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
}

// evictLRU removes the least recently used result
// Note: This method assumes the caller holds the appropriate lock
func (cache *TemplateTagProcessingCache) evictLRU() {
	if len(cache.AccessOrder) == 0 {
		return
	}

	// Remove the first (oldest) result
	oldestKey := cache.AccessOrder[0]
	cache.removeResult(oldestKey)
}

// GetCacheStats returns cache statistics
func (cache *TemplateTagProcessingCache) GetCacheStats() map[string]interface{} {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	totalRequests := cache.Misses
	for _, hits := range cache.Hits {
		totalRequests += hits
	}

	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(len(cache.Hits)) / float64(totalRequests)
	}

	return map[string]interface{}{
		"results":        len(cache.Results),
		"max_size":       cache.MaxSize,
		"ttl_seconds":    cache.TTL,
		"hits":           cache.Hits,
		"misses":         cache.Misses,
		"hit_rate":       hitRate,
		"total_requests": totalRequests,
		"tag_types":      len(cache.TagTypes),
		"context_hashes": len(cache.ContextHashes),
	}
}

// ClearCache clears the template tag processing cache
func (cache *TemplateTagProcessingCache) ClearCache() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.Results = make(map[string]string)
	cache.Hits = make(map[string]int)
	cache.Misses = 0
	cache.Timestamps = make(map[string]time.Time)
	cache.AccessOrder = make([]string, 0)
	cache.TagTypes = make(map[string]string)
	cache.ContextHashes = make(map[string]string)
}

// InvalidateTagType invalidates cache entries for a specific tag type
func (cache *TemplateTagProcessingCache) InvalidateTagType(tagType string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// Remove all results that have this tag type
	for key, cachedTagType := range cache.TagTypes {
		if cachedTagType == tagType {
			cache.removeResult(key)
		}
	}
}

// InvalidateContext invalidates cache entries for a specific context
func (cache *TemplateTagProcessingCache) InvalidateContext(context string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// Remove all results that have this context hash
	for key, contextHash := range cache.ContextHashes {
		if contextHash == context {
			cache.removeResult(key)
		}
	}
}

// New creates a new Golem instance
func New(verbose bool) *Golem {
	logger := log.New(os.Stdout, "[GOLEM] ", log.LstdFlags)

	// Set log level based on verbose flag
	// When verbose is enabled, show Info level and above (Info, Warn, Error)
	// When verbose is disabled, show only Error level
	logLevel := LogLevelError
	if verbose {
		logLevel = LogLevelInfo
	}

	// Create OOB manager and register built-in handlers
	oobMgr := NewOOBManager(verbose, logger)

	// Register built-in OOB handlers
	oobMgr.RegisterHandler(&SystemInfoHandler{})
	oobMgr.RegisterHandler(&SessionInfoHandler{})

	// Properties handler will be registered when AIML is loaded
	// since it needs access to the knowledge base

	// Create SRAIX manager
	sraixMgr := NewSRAIXManager(logger, verbose)

	// Create text processing components
	sentenceSplitter := NewSentenceSplitter()
	wordBoundaryDetector := NewWordBoundaryDetector()

	// Create template processing components
	templateCache := &TemplateCache{
		Cache:      make(map[string]string),
		Timestamps: make(map[string]string),
		Hits:       make(map[string]int),
		MaxSize:    1000,
		TTL:        3600, // 1 hour
	}

	templateConfig := &TemplateProcessingConfig{
		EnableCaching:     true,
		CacheSize:         1000,
		CacheTTL:          3600,
		EnableParallel:    true,
		MaxParallelOps:    4,
		EnableMetrics:     true,
		EnableValidation:  true,
		EnableDebugging:   verbose,
		MemoryLimit:       50 * 1024 * 1024, // 50MB
		ProcessingTimeout: 5000,             // 5 seconds
	}

	templateMetrics := &TemplateProcessingMetrics{
		TotalProcessed:     0,
		AverageProcessTime: 0.0,
		CacheHits:          0,
		CacheMisses:        0,
		CacheHitRate:       0.0,
		TagProcessingTimes: make(map[string]float64),
		ErrorCount:         0,
		LastProcessed:      "",
		MemoryPeak:         0,
		ParallelOps:        0,
	}

	// Create persistent learning manager with default storage path
	persistentLearning := NewPersistentLearningManager("./learned_categories")

	// Create regex compilation caches
	patternRegexCache := NewRegexCache(500, 3600)  // 500 patterns, 1 hour TTL
	tagProcessingCache := NewRegexCache(200, 7200) // 200 patterns, 2 hours TTL
	normalizationCache := NewRegexCache(100, 1800) // 100 patterns, 30 minutes TTL

	// Create text normalization result cache
	textNormalizationCache := NewTextNormalizationCache(1000, 1800) // 1000 results, 30 minutes TTL

	// Create variable resolution cache
	variableResolutionCache := NewVariableResolutionCache(500, 900) // 500 results, 15 minutes TTL

	// Create that pattern cache
	thatPatternCache := NewThatPatternCache(200) // 200 patterns

	// Create template tag processing cache
	templateTagProcessingCache := NewTemplateTagProcessingCache(1000, 1800) // 1000 results, 30 minutes TTL

	// Create pattern matching cache
	patternMatchingCache := NewPatternMatchingCache(2000, 3600) // 2000 results, 1 hour TTL

	// Create tree processor (will be initialized after Golem is created)
	var treeProcessor *TreeProcessor

	return &Golem{
		verbose:                    verbose,
		logLevel:                   logLevel,
		logger:                     logger,
		sessions:                   make(map[string]*ChatSession),
		sessionID:                  1,
		oobMgr:                     oobMgr,
		sraixMgr:                   sraixMgr,
		sentenceSplitter:           sentenceSplitter,
		wordBoundaryDetector:       wordBoundaryDetector,
		templateCache:              templateCache,
		templateConfig:             templateConfig,
		templateMetrics:            templateMetrics,
		patternRegexCache:          patternRegexCache,
		tagProcessingCache:         tagProcessingCache,
		normalizationCache:         normalizationCache,
		textNormalizationCache:     textNormalizationCache,
		variableResolutionCache:    variableResolutionCache,
		thatPatternCache:           thatPatternCache,
		templateTagProcessingCache: templateTagProcessingCache,
		patternMatchingCache:       patternMatchingCache,
		persistentLearning:         persistentLearning,
		treeProcessor:              treeProcessor,
		useTreeProcessing:          true, // Tree-based AST processing is now the default (correct AIML behavior)
	}
}

// LogError logs an error message
func (g *Golem) LogError(format string, args ...interface{}) {
	if g.logLevel >= LogLevelError {
		g.logger.Printf("[ERROR] "+format, args...)
	}
}

// LogWarn logs a warning message
func (g *Golem) LogWarn(format string, args ...interface{}) {
	if g.logLevel >= LogLevelWarn {
		g.logger.Printf("[WARN] "+format, args...)
	}
}

// LogInfo logs an info message
func (g *Golem) LogInfo(format string, args ...interface{}) {
	if g.logLevel >= LogLevelInfo {
		g.logger.Printf("[INFO] "+format, args...)
	}
}

// LogDebug logs a debug message
func (g *Golem) LogDebug(format string, args ...interface{}) {
	if g.logLevel >= LogLevelDebug {
		g.logger.Printf("[DEBUG] "+format, args...)
	}
}

// LogTrace logs a trace message
func (g *Golem) LogTrace(format string, args ...interface{}) {
	if g.logLevel >= LogLevelTrace {
		g.logger.Printf("[TRACE] "+format, args...)
	}
}

// EnableTreeProcessing enables tree-based tag processing
func (g *Golem) EnableTreeProcessing() {
	g.useTreeProcessing = true
	if g.treeProcessor == nil {
		g.treeProcessor = NewTreeProcessor(g)
	}
	g.LogInfo("Tree-based tag processing enabled")
}

// DisableTreeProcessing disables tree-based tag processing (reverts to regex-based)
func (g *Golem) DisableTreeProcessing() {
	g.useTreeProcessing = false
	g.LogInfo("Tree-based tag processing disabled, using regex-based processing")
}

// IsTreeProcessingEnabled returns whether tree-based processing is enabled
func (g *Golem) IsTreeProcessingEnabled() bool {
	return g.useTreeProcessing
}

// SetPersistentLearningPath sets the path for persistent learning storage
func (g *Golem) SetPersistentLearningPath(path string) {
	if g.persistentLearning != nil {
		g.persistentLearning.SetStoragePath(path)
	}
}

// GetPersistentLearningInfo returns information about persistent learning
func (g *Golem) GetPersistentLearningInfo() (map[string]interface{}, error) {
	if g.persistentLearning == nil {
		return nil, fmt.Errorf("persistent learning not initialized")
	}
	return g.persistentLearning.GetPersistentCategoryInfo()
}

// LoadPersistentCategories loads categories from persistent storage
func (g *Golem) LoadPersistentCategories() error {
	if g.persistentLearning == nil {
		return fmt.Errorf("persistent learning not initialized")
	}

	if g.aimlKB == nil {
		return fmt.Errorf("no knowledge base available")
	}

	categories, err := g.persistentLearning.LoadPersistentCategories()
	if err != nil {
		return fmt.Errorf("failed to load persistent categories: %v", err)
	}

	// Add categories to the knowledge base
	for _, category := range categories {
		normalizedPattern := NormalizePattern(category.Pattern)

		// Check if category already exists
		if existingCategory, exists := g.aimlKB.Patterns[normalizedPattern]; exists {
			// Update existing category
			*existingCategory = category
		} else {
			// Add new category
			g.aimlKB.Categories = append(g.aimlKB.Categories, category)
			g.aimlKB.Patterns[normalizedPattern] = &g.aimlKB.Categories[len(g.aimlKB.Categories)-1]
		}
	}

	g.LogInfo("Loaded %d persistent categories", len(categories))
	return nil
}

// SavePersistentCategories saves all current categories to persistent storage
func (g *Golem) SavePersistentCategories(source string) error {
	if g.persistentLearning == nil {
		return fmt.Errorf("persistent learning not initialized")
	}

	if g.aimlKB == nil {
		return fmt.Errorf("no knowledge base available")
	}

	return g.persistentLearning.SavePersistentCategories(g.aimlKB.Categories, source)
}

// GetSessionLearningStats returns learning statistics for a session
func (g *Golem) GetSessionLearningStats(sessionID string) (*SessionLearningStats, error) {
	g.sessionMutex.RLock()
	session, exists := g.sessions[sessionID]
	g.sessionMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if session.LearningStats == nil {
		return &SessionLearningStats{}, nil
	}

	return session.LearningStats, nil
}

// GetSessionLearnedCategories returns categories learned in a session
func (g *Golem) GetSessionLearnedCategories(sessionID string) ([]Category, error) {
	g.sessionMutex.RLock()
	session, exists := g.sessions[sessionID]
	g.sessionMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session.LearnedCategories, nil
}

// ClearSessionLearning clears all learned categories from a session
func (g *Golem) ClearSessionLearning(sessionID string) error {
	g.sessionMutex.RLock()
	session, exists := g.sessions[sessionID]
	g.sessionMutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Remove all session-learned categories from knowledge base
	for _, category := range session.LearnedCategories {
		normalizedPattern := NormalizePattern(category.Pattern)
		delete(g.aimlKB.Patterns, normalizedPattern)

		// Remove from categories slice
		for i, cat := range g.aimlKB.Categories {
			if NormalizePattern(cat.Pattern) == normalizedPattern {
				g.aimlKB.Categories = append(g.aimlKB.Categories[:i], g.aimlKB.Categories[i+1:]...)
				break
			}
		}
	}

	// Clear session learning data
	session.LearnedCategories = []Category{}
	if session.LearningStats != nil {
		session.LearningStats.TotalLearned = 0
		session.LearningStats.TotalUnlearned = 0
		session.LearningStats.LearningSources = make(map[string]int)
		session.LearningStats.PatternTypes = make(map[string]int)
		session.LearningStats.TemplateLengths = []int{}
		session.LearningStats.ValidationErrors = 0
		session.LearningStats.LearningRate = 0.0
	}

	g.LogInfo("Cleared all learning data for session: %s", sessionID)
	return nil
}

// GetLearningSummary returns a summary of learning across all sessions
func (g *Golem) GetLearningSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"total_sessions":   len(g.sessions),
		"total_categories": len(g.aimlKB.Categories),
		"session_stats":    make(map[string]interface{}),
		"global_stats": map[string]interface{}{
			"total_learned":     0,
			"total_unlearned":   0,
			"validation_errors": 0,
		},
	}

	globalStats := summary["global_stats"].(map[string]interface{})

	for sessionID, session := range g.sessions {
		if session.LearningStats != nil {
			sessionStats := map[string]interface{}{
				"total_learned":     session.LearningStats.TotalLearned,
				"total_unlearned":   session.LearningStats.TotalUnlearned,
				"validation_errors": session.LearningStats.ValidationErrors,
				"learning_rate":     session.LearningStats.LearningRate,
				"pattern_types":     session.LearningStats.PatternTypes,
				"learning_sources":  session.LearningStats.LearningSources,
			}

			summary["session_stats"].(map[string]interface{})[sessionID] = sessionStats

			// Add to global stats
			globalStats["total_learned"] = globalStats["total_learned"].(int) + session.LearningStats.TotalLearned
			globalStats["total_unlearned"] = globalStats["total_unlearned"].(int) + session.LearningStats.TotalUnlearned
			globalStats["validation_errors"] = globalStats["validation_errors"].(int) + session.LearningStats.ValidationErrors
		}
	}

	return summary
}

// SetLogLevel sets the logging level
func (g *Golem) SetLogLevel(level LogLevel) {
	g.logLevel = level
}

// GetLogLevel returns the current logging level
func (g *Golem) GetLogLevel() LogLevel {
	return g.logLevel
}

// LogVerbose logs a message only if verbose mode is enabled (for backward compatibility)
// This is a convenience function that maps to LogDebug for backward compatibility
func (g *Golem) LogVerbose(format string, args ...interface{}) {
	if g.verbose {
		g.LogDebug(format, args...)
	}
}

/*
Logging Usage Examples:

Replace verbose logging patterns like this:

OLD PATTERN:
	if g.verbose {
		g.logger.Printf("Loading AIML from string")
	}

NEW PATTERN (using level-based logging):
	g.LogDebug("Loading AIML from string")

OLD PATTERN:
	if g.verbose {
		g.logger.Printf("Total categories: %d", len(g.aimlKB.Categories))
	}

NEW PATTERN:
	g.LogDebug("Total categories: %d", len(g.aimlKB.Categories))

OLD PATTERN:
	if g.verbose {
		g.logger.Printf("Failed to parse learnf content: %v", err)
	}

NEW PATTERN (for errors):
	g.LogError("Failed to parse learnf content: %v", err)

Available log levels:
- LogError: Error messages (always shown)
- LogWarn: Warning messages (shown when verbose enabled)
- LogInfo: Informational messages (shown when verbose enabled)
- LogDebug: Debug messages (shown when log level set to Debug or Trace)
- LogTrace: Very detailed trace messages (shown when log level set to Trace)

Verbose flag behavior:
- --verbose: Shows Info, Warn, and Error messages
- No --verbose: Shows only Error messages

Set log level manually:
	g.SetLogLevel(LogLevelDebug)
*/

// Execute runs the specified command with arguments
//
// IMPORTANT: This method operates on the current Golem instance state.
// - In CLI single-command mode: New instance created per command (state lost)
// - In CLI interactive mode: Same instance used across commands (state preserved)
// - In library mode: User controls instance lifecycle (state managed by user)
//
// Commands that modify state (load, session create/switch, properties set):
// - Will persist in interactive mode and library mode
// - Will be lost in single-command mode
func (g *Golem) Execute(command string, args []string) error {
	g.LogInfo("Executing command: %s with args: %v", command, args)

	switch command {
	case "load":
		return g.loadCommand(args)
	case "chat":
		return g.chatCommand(args)
	case "session":
		return g.sessionCommand(args)
	case "properties":
		return g.propertiesCommand(args)
	case "oob":
		return g.oobCommand(args)
	case "sraix":
		return g.sraixCommand(args)
	case "process":
		return g.processCommand(args)
	case "analyze":
		return g.analyzeCommand(args)
	case "generate":
		return g.generateCommand(args)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// LoadCommand handles the load command
// loadAllRelatedFiles loads all .aiml, .map, and .set files from the same directory as the given file
func (g *Golem) loadAllRelatedFiles(filePath string) error {
	dir := filepath.Dir(filePath)

	g.LogInfo("Loading all related files from directory: %s", dir)

	// Load AIML files from directory
	aimlKB, err := g.LoadAIMLFromDirectory(dir)
	if err != nil {
		// If no AIML files found, create an empty knowledge base
		if strings.Contains(err.Error(), "no AIML files found") {
			aimlKB = NewAIMLKnowledgeBase()
			// Load default properties
			err = g.loadDefaultProperties(aimlKB)
			if err != nil {
				return fmt.Errorf("failed to load default properties: %v", err)
			}
		} else {
			return fmt.Errorf("failed to load AIML files from directory: %v", err)
		}
	}

	// Load maps from directory
	maps, err := g.LoadMapsFromDirectory(dir)
	if err != nil {
		return fmt.Errorf("failed to load map files from directory: %v", err)
	}

	// Load sets from directory
	sets, err := g.LoadSetsFromDirectory(dir)
	if err != nil {
		return fmt.Errorf("failed to load set files from directory: %v", err)
	}

	// Load substitutions from directory
	substitutions, err := g.LoadSubstitutionsFromDirectory(dir)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load substitutions from directory: %v", err)
	}

	// Load properties from directory
	properties, err := g.LoadPropertiesFromDirectory(dir)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load properties from directory: %v", err)
	}

	// Load pdefaults from directory
	pdefaults, err := g.LoadPDefaultsFromDirectory(dir)
	if err != nil {
		// Log the error but don't fail the entire operation
		g.LogInfo("Warning: failed to load pdefaults from directory: %v", err)
	}

	// Merge maps into knowledge base
	for mapName, mapData := range maps {
		aimlKB.Maps[mapName] = mapData
	}

	// Merge sets into knowledge base
	for setName, setMembers := range sets {
		aimlKB.AddSetMembers(setName, setMembers)
	}

	// Merge substitutions into knowledge base
	for subName, subData := range substitutions {
		aimlKB.Substitutions[subName] = subData
	}

	// Merge properties into knowledge base
	for _, propData := range properties {
		for key, value := range propData {
			aimlKB.Properties[key] = value
		}
	}

	// Merge pdefaults into knowledge base (as default user properties)
	for pdefaultName, pdefaultData := range pdefaults {
		for key, value := range pdefaultData {
			// Store pdefaults as a special type of property with prefix
			aimlKB.Properties["pdefault."+pdefaultName+"."+key] = value
		}
	}

	g.LogInfo("About to set knowledge base with %d properties", len(aimlKB.Properties))

	// Set the knowledge base using SetKnowledgeBase to trigger SRAIX configuration
	g.SetKnowledgeBase(aimlKB)

	g.LogInfo("Knowledge base set successfully")

	// Print summary
	fmt.Printf("Successfully loaded all related files from directory: %s\n", dir)
	fmt.Printf("Loaded %d categories\n", len(aimlKB.Categories))
	fmt.Printf("Loaded %d maps\n", len(maps))
	fmt.Printf("Loaded %d sets\n", len(sets))
	fmt.Printf("Loaded %d substitution files\n", len(substitutions))
	fmt.Printf("Loaded %d properties files\n", len(properties))
	fmt.Printf("Loaded %d pdefaults files\n", len(pdefaults))

	return nil
}

func (g *Golem) loadCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("load command requires a filename or directory path")
	}

	path := args[0]
	g.LogInfo("Loading: %s", path)

	// Check if path exists and get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %v", path, err)
	}

	// Check if path exists
	fileInfo, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		// Load all related files from directory (AIML, maps, sets, properties, etc.)
		// Use loadAllRelatedFiles which properly triggers SRAIX configuration
		dummyFilePath := filepath.Join(absPath, "dummy.aiml")
		err := g.loadAllRelatedFiles(dummyFilePath)
		if err != nil {
			return fmt.Errorf("failed to load files from directory: %v", err)
		}
	} else if strings.HasSuffix(strings.ToLower(absPath), ".aiml") {
		// Load single AIML file and all related files from the same directory
		err := g.loadAllRelatedFiles(absPath)
		if err != nil {
			return fmt.Errorf("failed to load AIML file and related files: %v", err)
		}
	} else if strings.HasSuffix(strings.ToLower(absPath), ".map") {
		// Load single map file and all related files from the same directory
		err := g.loadAllRelatedFiles(absPath)
		if err != nil {
			return fmt.Errorf("failed to load map file and related files: %v", err)
		}
	} else if strings.HasSuffix(strings.ToLower(absPath), ".set") {
		// Load single set file and all related files from the same directory
		err := g.loadAllRelatedFiles(absPath)
		if err != nil {
			return fmt.Errorf("failed to load set file and related files: %v", err)
		}
	} else {
		// Read file contents (non-AIML file)
		content, err := g.LoadFile(absPath)
		if err != nil {
			return fmt.Errorf("failed to load file %s: %v", absPath, err)
		}

		fmt.Printf("Successfully loaded file: %s\n", absPath)
		fmt.Printf("File size: %d bytes\n", len(content))

		if g.verbose {
			// Show first 200 characters of content
			preview := content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("Content preview: %s\n", preview)
		}
	}

	return nil
}

// ChatCommand handles the chat command
func (g *Golem) chatCommand(args []string) error {
	if g.aimlKB == nil {
		return fmt.Errorf("no AIML knowledge base loaded. Use 'load' command first")
	}

	if len(args) == 0 {
		return fmt.Errorf("chat command requires input text")
	}

	// Get or create current session
	session := g.getCurrentSession()
	if session == nil {
		session = g.createSession("")
	}

	input := strings.Join(args, " ")
	g.LogInfo("Processing chat input in session %s: %s", session.ID, input)

	// Check for OOB messages first
	if oobMsg, isOOB := ParseOOBMessage(input); isOOB {
		response, err := g.oobMgr.ProcessOOB(oobMsg.Raw, session)
		if err != nil {
			fmt.Printf("OOB Error: %v\n", err)
			session.History = append(session.History, "User: "+input)
			session.History = append(session.History, "Golem: OOB Error: "+err.Error())
			return nil
		}
		fmt.Printf("OOB: %s\n", response)
		session.History = append(session.History, "User: "+input)
		session.History = append(session.History, "Golem: OOB: "+response)
		return nil
	}

	// Add to history
	session.History = append(session.History, "User: "+input)

	// Add to request history for <request> tag support
	session.AddToRequestHistory(input)

	// Match pattern and get response
	category, wildcards, err := g.aimlKB.MatchPattern(input)
	if err != nil {
		response := g.aimlKB.GetProperty("default_response")
		if response == "" {
			response = "I don't understand: " + input
		}
		fmt.Printf("Golem: %s\n", response)
		session.History = append(session.History, "Golem: "+response)
		return nil
	}

	// Process template with session context
	response := g.ProcessTemplateWithSession(category.Template, wildcards, session)
	fmt.Printf("Golem: %s\n", response)
	session.History = append(session.History, "Golem: "+response)

	// Add to response history for <response> tag support
	session.AddToResponseHistory(response)

	return nil
}

// PropertiesCommand handles the properties command
func (g *Golem) propertiesCommand(args []string) error {
	if g.aimlKB == nil {
		return fmt.Errorf("no AIML knowledge base loaded. Use 'load' command first")
	}

	if len(args) == 0 {
		// Show all properties
		fmt.Println("Bot Properties:")
		fmt.Println(strings.Repeat("=", 50))
		for key, value := range g.aimlKB.Properties {
			fmt.Printf("%-20s: %s\n", key, value)
		}
		return nil
	}

	if len(args) == 1 {
		// Show specific property
		key := args[0]
		value := g.aimlKB.GetProperty(key)
		if value == "" {
			fmt.Printf("Property '%s' not found\n", key)
		} else {
			fmt.Printf("%s: %s\n", key, value)
		}
		return nil
	}

	if len(args) == 2 {
		// Set property
		key := args[0]
		value := args[1]
		g.aimlKB.SetProperty(key, value)
		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	}

	return fmt.Errorf("usage: properties [key] [value]")
}

// ProcessCommand handles the process command
func (g *Golem) processCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("process command requires input file")
	}

	inputFile := args[0]
	g.LogInfo("Processing file: %s", inputFile)

	// Process the input file (placeholder implementation)
	fmt.Printf("Processing file: %s\n", inputFile)
	return nil
}

// AnalyzeCommand handles the analyze command
func (g *Golem) analyzeCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("analyze command requires input file")
	}

	inputFile := args[0]
	g.LogInfo("Analyzing file: %s", inputFile)

	// Analyze the input file (placeholder implementation)
	fmt.Printf("Analyzing file: %s\n", inputFile)
	return nil
}

// GenerateCommand handles the generate command
func (g *Golem) generateCommand(args []string) error {
	outputFile := "output.txt"

	// Parse optional output file argument
	if len(args) > 0 && args[0] == "--output" && len(args) > 1 {
		outputFile = args[1]
	}

	g.LogInfo("Generating output to: %s", outputFile)

	// Generate output (placeholder implementation)
	fmt.Printf("Generating output to: %s\n", outputFile)
	return nil
}

// ProcessData is a library function that can be used by other programs
func (g *Golem) ProcessData(input string) (string, error) {
	g.LogInfo("Processing data: %s", input)

	// Process data (placeholder implementation)
	result := fmt.Sprintf("Processed: %s", input)
	return result, nil
}

// ProcessInput processes user input with full context support
func (g *Golem) ProcessInput(input string, session *ChatSession) (string, error) {
	if g.aimlKB == nil {
		return "", fmt.Errorf("no AIML knowledge base loaded")
	}

	g.LogInfo("Processing input: %s", input)

	// Normalize input
	normalizedInput := g.CachedNormalizePattern(input)

	// Get current topic and that context
	currentTopic := session.GetSessionTopic()
	lastThat := session.GetLastThat()

	// Normalize the that context for matching using enhanced that normalization
	normalizedThat := ""
	if lastThat != "" {
		normalizedThat = g.CachedNormalizeThatPattern(lastThat)
	}

	// Try to match pattern with full context (using index 0 for last response)
	category, wildcards, err := g.aimlKB.MatchPatternWithTopicAndThatIndexOriginalCached(g, normalizedInput, input, currentTopic, normalizedThat, 0)
	if err != nil {
		return "", err
	}

	// Capture that context from template before processing (for next input)
	// This needs to be done before the template is processed because <set> tags might change the content
	nextThatContext := g.extractThatContextFromTemplate(category.Template)

	// Process template with context
	response := g.ProcessTemplateWithContext(category.Template, wildcards, session)

	// Add to history
	session.History = append(session.History, input)
	session.LastActivity = time.Now().Format(time.RFC3339)

	// Add to request history for <request> tag support
	session.AddToRequestHistory(input)

	// Add the extracted that context to history for future context matching
	if nextThatContext != "" {
		session.AddToThatHistory(nextThatContext)
	}

	// Add to response history for <response> tag support
	session.AddToResponseHistory(response)

	return response, nil
}

// ProcessInputWithThatIndex processes user input with specific that context index
func (g *Golem) ProcessInputWithThatIndex(input string, session *ChatSession, thatIndex int) (string, error) {
	if g.aimlKB == nil {
		return "", fmt.Errorf("no AIML knowledge base loaded")
	}

	g.LogInfo("Processing input with that index %d: %s", thatIndex, input)

	// Normalize input
	normalizedInput := g.CachedNormalizePattern(input)

	// Get current topic and that context by index
	currentTopic := session.GetSessionTopic()
	thatContext := session.GetThatByIndex(thatIndex)

	g.LogInfo("That context for index %d: '%s'", thatIndex, thatContext)
	g.LogInfo("That history: %v", session.ThatHistory)

	// Normalize the that context for matching using enhanced that normalization
	normalizedThat := ""
	if thatContext != "" {
		normalizedThat = g.CachedNormalizeThatPattern(thatContext)
	}

	// Try to match pattern with full context and specific that index
	category, wildcards, err := g.aimlKB.MatchPatternWithTopicAndThatIndexOriginalCached(g, normalizedInput, input, currentTopic, normalizedThat, thatIndex)
	if err != nil {
		return "", err
	}

	// Capture that context from template before processing (for next input)
	// This needs to be done before the template is processed because <set> tags might change the content
	nextThatContext := g.extractThatContextFromTemplate(category.Template)

	// Process template with context
	response := g.ProcessTemplateWithContext(category.Template, wildcards, session)

	// Add to history
	session.History = append(session.History, input)
	session.LastActivity = time.Now().Format(time.RFC3339)

	// Add to request history for <request> tag support
	session.AddToRequestHistory(input)

	// Add the extracted that context to history for future context matching
	if nextThatContext != "" {
		session.AddToThatHistory(nextThatContext)
	}

	// Add to response history for <response> tag support
	session.AddToResponseHistory(response)

	return response, nil
}

// extractThatContextFromTemplate extracts the that context from a template
// This is used to capture the that context before <set> tags are processed
func (g *Golem) extractThatContextFromTemplate(template string) string {
	// For that context, we need to extract only the content that comes after <set> tags
	// This is because <set> tags are processed and removed, but the that context
	// should only include the content that remains after processing

	// Find <set name="topic"> tags and extract content after them
	topicSetRegex := regexp.MustCompile(`<set\s+name="topic">(.*?)</set>`)
	matches := topicSetRegex.FindAllStringSubmatch(template, -1)

	if len(matches) > 0 {
		// If there are <set> tags, extract only the content after the last one
		lastMatch := matches[len(matches)-1]
		lastMatchEnd := strings.Index(template, lastMatch[0]) + len(lastMatch[0])

		// Get the content after the last <set> tag
		thatContext := strings.TrimSpace(template[lastMatchEnd:])

		return thatContext
	}

	// If no <set> tags, return the entire template
	processedTemplate := strings.TrimSpace(template)

	return processedTemplate
}

// AnalyzeData is a library function that can be used by other programs
func (g *Golem) AnalyzeData(input string) (map[string]interface{}, error) {
	g.LogInfo("Analyzing data: %s", input)

	// Analyze the input file (placeholder implementation)
	result := map[string]interface{}{
		"input":  input,
		"status": "analyzed",
		"length": len(input),
	}
	return result, nil
}

// GenerateOutput is a library function that can be used by other programs
func (g *Golem) GenerateOutput(data interface{}) (string, error) {
	g.LogInfo("Generating output for data: %v", data)

	// Generate output (placeholder implementation)
	result := fmt.Sprintf("Generated output for: %v", data)
	return result, nil
}

// LoadFile is a library function that loads a file and returns its contents
func (g *Golem) LoadFile(filename string) (string, error) {
	g.LogInfo("Loading file: %s", filename)

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Read the file contents
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return string(content), nil
}

// SetKnowledgeBase sets the AIML knowledge base
func (g *Golem) SetKnowledgeBase(kb *AIMLKnowledgeBase) {
	g.aimlKB = kb

	// Register properties handler now that we have a knowledge base
	propertiesHandler := &PropertiesHandler{aimlKB: kb}
	g.oobMgr.RegisterHandler(propertiesHandler)

	// Load persistent learned categories if available
	if g.persistentLearning != nil && kb != nil {
		g.LogInfo("Loading persistent learned categories...")
		persistentCategories, err := g.persistentLearning.LoadPersistentCategories()
		if err != nil {
			g.LogWarn("Failed to load persistent categories: %v", err)
		} else if len(persistentCategories) > 0 {
			g.LogInfo("Loaded %d persistent learned categories", len(persistentCategories))
			// Ensure Patterns map is initialized
			if g.aimlKB.Patterns == nil {
				g.aimlKB.Patterns = make(map[string]*Category)
			}
			// Add persistent categories to the knowledge base
			for _, category := range persistentCategories {
				// Normalize pattern and add to knowledge base
				normalizedPattern := NormalizePattern(category.Pattern)
				key := normalizedPattern
				if category.That != "" {
					key += "|THAT:" + NormalizePattern(category.That)
					if category.ThatIndex != 0 {
						key += fmt.Sprintf("|THATINDEX:%d", category.ThatIndex)
					}
				}
				if category.Topic != "" {
					key += "|TOPIC:" + strings.ToUpper(category.Topic)
				}

				// Add category to knowledge base
				g.aimlKB.Categories = append(g.aimlKB.Categories, category)
				g.aimlKB.Patterns[key] = &g.aimlKB.Categories[len(g.aimlKB.Categories)-1]
			}
		} else {
			g.LogInfo("No persistent learned categories found")
		}
	}

	// Configure SRAIX services from properties if SRAIX manager exists
	if g.sraixMgr != nil && kb != nil && kb.Properties != nil {
		g.LogInfo("Configuring SRAIX from %d properties...", len(kb.Properties))
		if err := g.sraixMgr.ConfigureFromProperties(kb.Properties); err != nil {
			g.LogWarn("Failed to configure SRAIX from properties: %v", err)
		} else {
			g.LogInfo("SRAIX configuration complete")
		}
	} else {
		g.LogInfo("Skipping SRAIX configuration: sraixMgr=%v, kb=%v, properties=%v",
			g.sraixMgr != nil, kb != nil, kb != nil && kb.Properties != nil)
	}
}

// GetKnowledgeBase returns the current AIML knowledge base
func (g *Golem) GetKnowledgeBase() *AIMLKnowledgeBase {
	return g.aimlKB
}

// SessionCommand handles session management
func (g *Golem) sessionCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session command requires subcommand: create, list, switch, delete, current")
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		return g.createSessionCommand(args[1:])
	case "list":
		return g.listSessionsCommand()
	case "switch":
		return g.switchSessionCommand(args[1:])
	case "delete":
		return g.deleteSessionCommand(args[1:])
	case "current":
		return g.currentSessionCommand()
	default:
		return fmt.Errorf("unknown session subcommand: %s", subcommand)
	}
}

// createSessionCommand creates a new chat session
func (g *Golem) createSessionCommand(args []string) error {
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	session := g.createSession(sessionID)
	fmt.Printf("Created session: %s\n", session.ID)
	return nil
}

// listSessionsCommand lists all active sessions
func (g *Golem) listSessionsCommand() error {
	g.sessionMutex.RLock()
	defer g.sessionMutex.RUnlock()

	if len(g.sessions) == 0 {
		fmt.Println("No active sessions")
		return nil
	}

	fmt.Println("Active Sessions:")
	fmt.Println(strings.Repeat("=", 50))
	for id, session := range g.sessions {
		marker := ""
		if id == g.currentID {
			marker = " (current)"
		}
		fmt.Printf("%-10s: Created %s, %d messages%s\n",
			id, session.CreatedAt, len(session.History), marker)
	}
	return nil
}

// switchSessionCommand switches to a different session
func (g *Golem) switchSessionCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session switch requires session ID")
	}

	sessionID := args[0]
	g.sessionMutex.RLock()
	_, exists := g.sessions[sessionID]
	g.sessionMutex.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	g.sessionMutex.Lock()
	g.currentID = sessionID
	g.sessionMutex.Unlock()
	fmt.Printf("Switched to session: %s\n", sessionID)
	return nil
}

// deleteSessionCommand deletes a session
func (g *Golem) deleteSessionCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session delete requires session ID")
	}

	sessionID := args[0]
	g.sessionMutex.Lock()
	defer g.sessionMutex.Unlock()

	if _, exists := g.sessions[sessionID]; !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	delete(g.sessions, sessionID)
	if g.currentID == sessionID {
		g.currentID = ""
	}
	fmt.Printf("Deleted session: %s\n", sessionID)
	return nil
}

// currentSessionCommand shows current session info
func (g *Golem) currentSessionCommand() error {
	g.sessionMutex.RLock()
	defer g.sessionMutex.RUnlock()

	if g.currentID == "" {
		fmt.Println("No current session")
		return nil
	}

	session := g.sessions[g.currentID]
	fmt.Printf("Current session: %s\n", session.ID)
	fmt.Printf("Created: %s\n", session.CreatedAt)
	fmt.Printf("Messages: %d\n", len(session.History))
	return nil
}

// createSession creates a new chat session
// CreateSession creates a new chat session with the given ID
func (g *Golem) CreateSession(sessionID string) *ChatSession {
	return g.createSession(sessionID)
}

func (g *Golem) createSession(sessionID string) *ChatSession {
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", g.sessionID)
		g.sessionID++
	}

	now := time.Now().Format(time.RFC3339)
	session := &ChatSession{
		ID:                sessionID,
		Variables:         make(map[string]string),
		History:           []string{},
		CreatedAt:         now,
		LastActivity:      now,
		RequestHistory:    []string{},
		ResponseHistory:   []string{},
		ThatHistory:       []string{},
		LearnedCategories: []Category{},
		LearningStats: &SessionLearningStats{
			TotalLearned:     0,
			TotalUnlearned:   0,
			LearningSources:  make(map[string]int),
			PatternTypes:     make(map[string]int),
			TemplateLengths:  []int{},
			ValidationErrors: 0,
			LearningRate:     0.0,
		},
	}

	// Initialize enhanced context management
	session.InitializeContextConfig()

	g.sessionMutex.Lock()
	g.sessions[sessionID] = session
	g.currentID = sessionID
	g.sessionMutex.Unlock()
	return session
}

// getCurrentSession returns the current session
func (g *Golem) getCurrentSession() *ChatSession {
	g.sessionMutex.RLock()
	defer g.sessionMutex.RUnlock()

	if g.currentID == "" {
		return nil
	}
	return g.sessions[g.currentID]
}

// ProcessTemplateWithSession processes a template with session context
func (g *Golem) ProcessTemplateWithSession(template string, wildcards map[string]string, session *ChatSession) string {
	// Ensure knowledge base is initialized for variable/collection operations
	if g.aimlKB == nil {
		g.aimlKB = NewAIMLKnowledgeBase()
	}

	// Create variable context for template processing with session
	ctx := &VariableContext{
		LocalVars:      make(map[string]string),
		Session:        session,
		Topic:          "", // Topic tracking will be implemented in future version
		KnowledgeBase:  g.aimlKB,
		RecursionDepth: 0,
	}

	return g.processTemplateWithContext(template, wildcards, ctx)
}

// replaceSessionVariableTags replaces <get name="var"/> tags with session variables
func (g *Golem) replaceSessionVariableTags(template string, session *ChatSession) string {
	// Find all <get name="var"/> tags
	getTagRegex := regexp.MustCompile(`<get name="([^"]+)"/>`)
	matches := getTagRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			varValue := session.Variables[varName]
			if varValue != "" {
				template = strings.ReplaceAll(template, match[0], varValue)
			}
		}
	}

	return template
}

// OOBCommand handles OOB-related commands
func (g *Golem) oobCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("oob command requires subcommand: list, test, register")
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		return g.listOOBHandlers()
	case "test":
		return g.testOOBCommand(args[1:])
	case "register":
		return g.registerOOBHandler(args[1:])
	default:
		return fmt.Errorf("unknown oob subcommand: %s", subcommand)
	}
}

// listOOBHandlers lists all registered OOB handlers
func (g *Golem) listOOBHandlers() error {
	handlers := g.oobMgr.ListHandlers()
	if len(handlers) == 0 {
		fmt.Println("No OOB handlers registered")
		return nil
	}

	fmt.Println("Registered OOB Handlers:")
	fmt.Println(strings.Repeat("=", 40))
	for _, name := range handlers {
		if handler, exists := g.oobMgr.GetHandler(name); exists {
			fmt.Printf("%-20s: %s\n", name, handler.GetDescription())
		}
	}
	return nil
}

// testOOBCommand tests an OOB message
func (g *Golem) testOOBCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("oob test requires a message")
	}

	message := strings.Join(args, " ")
	session := g.getCurrentSession()
	if session == nil {
		session = g.createSession("")
	}

	response, err := g.oobMgr.ProcessOOB(message, session)
	if err != nil {
		fmt.Printf("OOB Error: %v\n", err)
		return nil
	}

	fmt.Printf("OOB Response: %s\n", response)
	return nil
}

// registerOOBHandler registers a new OOB handler (for advanced users)
func (g *Golem) registerOOBHandler(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("oob register requires handler name and description")
	}

	name := args[0]
	description := strings.Join(args[1:], " ")

	// Create a simple test handler
	handler := &TestOOBHandler{
		name:        name,
		description: description,
	}

	g.oobMgr.RegisterHandler(handler)
	fmt.Printf("Registered custom OOB handler: %s\n", name)
	return nil
}

// TestOOBHandler is a simple test handler for demonstration
type TestOOBHandler struct {
	name        string
	description string
}

func (h *TestOOBHandler) CanHandle(message string) bool {
	return strings.HasPrefix(strings.ToUpper(message), strings.ToUpper(h.name))
}

func (h *TestOOBHandler) Process(message string, session *ChatSession) (string, error) {
	return fmt.Sprintf("Test handler '%s' processed: %s", h.name, message), nil
}

func (h *TestOOBHandler) GetName() string {
	return h.name
}

func (h *TestOOBHandler) GetDescription() string {
	return h.description
}

// SRAIX Management Methods

// AddSRAIXConfig adds a new SRAIX service configuration
func (g *Golem) AddSRAIXConfig(config *SRAIXConfig) error {
	return g.sraixMgr.AddConfig(config)
}

// GetSRAIXConfig retrieves a SRAIX service configuration
func (g *Golem) GetSRAIXConfig(name string) (*SRAIXConfig, bool) {
	return g.sraixMgr.GetConfig(name)
}

// ListSRAIXConfigs returns all configured SRAIX services
func (g *Golem) ListSRAIXConfigs() map[string]*SRAIXConfig {
	return g.sraixMgr.ListConfigs()
}

// LoadSRAIXConfigsFromFile loads SRAIX configurations from a JSON file
func (g *Golem) LoadSRAIXConfigsFromFile(filename string) error {
	return g.sraixMgr.LoadSRAIXConfigsFromFile(filename)
}

// LoadSRAIXConfigsFromDirectory loads all SRAIX configuration files from a directory
func (g *Golem) LoadSRAIXConfigsFromDirectory(dirPath string) error {
	return g.sraixMgr.LoadSRAIXConfigsFromDirectory(dirPath)
}

// sraixCommand handles SRAIX-related CLI commands
func (g *Golem) sraixCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("sraix command requires subcommand: load, list, test")
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "load":
		return g.sraixLoadCommand(subArgs)
	case "list":
		return g.sraixListCommand()
	case "test":
		return g.sraixTestCommand(subArgs)
	default:
		return fmt.Errorf("unknown sraix subcommand: %s", subcommand)
	}
}

// sraixLoadCommand loads SRAIX configurations from file or directory
func (g *Golem) sraixLoadCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("sraix load requires a filename or directory path")
	}

	path := args[0]

	// Check if it's a file or directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to access path %s: %v", path, err)
	}

	if info.IsDir() {
		err = g.LoadSRAIXConfigsFromDirectory(path)
		if err != nil {
			return fmt.Errorf("failed to load SRAIX configs from directory: %v", err)
		}
		fmt.Printf("Successfully loaded SRAIX configurations from directory: %s\n", path)
	} else {
		err = g.LoadSRAIXConfigsFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load SRAIX config file: %v", err)
		}
		fmt.Printf("Successfully loaded SRAIX configuration file: %s\n", path)
	}

	// Show loaded configurations
	configs := g.ListSRAIXConfigs()
	fmt.Printf("Loaded %d SRAIX service(s)\n", len(configs))
	for name, config := range configs {
		fmt.Printf("  %s: %s %s\n", name, config.Method, config.BaseURL)
	}

	return nil
}

// sraixListCommand lists all configured SRAIX services
func (g *Golem) sraixListCommand() error {
	configs := g.ListSRAIXConfigs()

	if len(configs) == 0 {
		fmt.Println("No SRAIX services configured")
		return nil
	}

	fmt.Println("Configured SRAIX Services:")
	fmt.Println("==========================================")
	for name, config := range configs {
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("  URL: %s %s\n", config.Method, config.BaseURL)
		fmt.Printf("  Timeout: %ds\n", config.Timeout)
		fmt.Printf("  Format: %s\n", config.ResponseFormat)
		if config.ResponsePath != "" {
			fmt.Printf("  Path: %s\n", config.ResponsePath)
		}
		if config.FallbackResponse != "" {
			fmt.Printf("  Fallback: %s\n", config.FallbackResponse)
		}
		fmt.Printf("  Wildcards: %t\n", config.IncludeWildcards)
		fmt.Println()
	}

	return nil
}

// sraixTestCommand tests a SRAIX service
func (g *Golem) sraixTestCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("sraix test requires service name and test input")
	}

	serviceName := args[0]
	testInput := strings.Join(args[1:], " ")

	config, exists := g.GetSRAIXConfig(serviceName)
	if !exists {
		return fmt.Errorf("SRAIX service '%s' not found", serviceName)
	}

	fmt.Printf("Testing SRAIX service '%s' with input: '%s'\n", serviceName, testInput)
	fmt.Printf("Service URL: %s %s\n", config.Method, config.BaseURL)
	fmt.Println("Making request...")

	// Make the SRAIX request
	response, err := g.sraixMgr.ProcessSRAIX(serviceName, testInput, make(map[string]string))
	if err != nil {
		fmt.Printf("SRAIX request failed: %v\n", err)
		return nil
	}

	fmt.Printf("Response: %s\n", response)
	return nil
}

// GetTemplateProcessingMetrics returns current template processing metrics
func (g *Golem) GetTemplateProcessingMetrics() *TemplateProcessingMetrics {
	return g.templateMetrics
}

// GetTemplateProcessingConfig returns current template processing configuration
func (g *Golem) GetTemplateProcessingConfig() *TemplateProcessingConfig {
	return g.templateConfig
}

// UpdateTemplateProcessingConfig updates template processing configuration
func (g *Golem) UpdateTemplateProcessingConfig(config *TemplateProcessingConfig) {
	g.templateConfig = config
	// Update cache settings
	if g.templateCache != nil {
		g.templateCache.MaxSize = config.CacheSize
		g.templateCache.TTL = config.CacheTTL
	}
}

// ClearTemplateCache clears the template cache
func (g *Golem) ClearTemplateCache() {
	if g.templateCache != nil {
		g.templateCache.Cache = make(map[string]string)
		g.templateCache.Timestamps = make(map[string]string)
		g.templateCache.Hits = make(map[string]int)
	}
}

// GetRegexCacheStats returns regex cache statistics
func (g *Golem) GetRegexCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if g.patternRegexCache != nil {
		stats["pattern_regex"] = g.patternRegexCache.GetCacheStats()
	}
	if g.tagProcessingCache != nil {
		stats["tag_processing"] = g.tagProcessingCache.GetCacheStats()
	}
	if g.normalizationCache != nil {
		stats["normalization"] = g.normalizationCache.GetCacheStats()
	}

	return stats
}

// GetTextNormalizationCacheStats returns text normalization cache statistics
func (g *Golem) GetTextNormalizationCacheStats() map[string]interface{} {
	if g.textNormalizationCache != nil {
		return g.textNormalizationCache.GetCacheStats()
	}
	return map[string]interface{}{
		"results":        0,
		"max_size":       0,
		"ttl_seconds":    0,
		"hits":           map[string]int{},
		"misses":         0,
		"hit_rate":       0.0,
		"total_requests": 0,
	}
}

// ClearRegexCaches clears all regex caches
func (g *Golem) ClearRegexCaches() {
	if g.patternRegexCache != nil {
		g.patternRegexCache.ClearCache()
	}
	if g.tagProcessingCache != nil {
		g.tagProcessingCache.ClearCache()
	}
	if g.normalizationCache != nil {
		g.normalizationCache.ClearCache()
	}
}

// ClearTextNormalizationCache clears the text normalization cache
func (g *Golem) ClearTextNormalizationCache() {
	if g.textNormalizationCache != nil {
		g.textNormalizationCache.ClearCache()
	}
}

// GetVariableResolutionCacheStats returns variable resolution cache statistics
func (g *Golem) GetVariableResolutionCacheStats() map[string]interface{} {
	if g.variableResolutionCache != nil {
		return g.variableResolutionCache.GetCacheStats()
	}
	return map[string]interface{}{
		"results":        0,
		"max_size":       0,
		"ttl_seconds":    0,
		"hits":           map[string]int{},
		"misses":         0,
		"hit_rate":       0.0,
		"total_requests": 0,
		"scope_hashes":   0,
	}
}

// ClearVariableResolutionCache clears the variable resolution cache
func (g *Golem) ClearVariableResolutionCache() {
	if g.variableResolutionCache != nil {
		g.variableResolutionCache.ClearCache()
	}
}

// GetThatPatternCacheStats returns that pattern cache statistics
func (g *Golem) GetThatPatternCacheStats() map[string]interface{} {
	if g.thatPatternCache != nil {
		return g.thatPatternCache.GetCacheStats()
	}
	return map[string]interface{}{
		"patterns":        0,
		"max_size":        0,
		"ttl_seconds":     0,
		"hits":            map[string]int{},
		"misses":          0,
		"hit_rate":        0.0,
		"total_requests":  0,
		"match_results":   0,
		"result_hits":     map[string]int{},
		"result_hit_rate": 0.0,
		"context_hashes":  0,
	}
}

// ClearThatPatternCache clears the that pattern cache
func (g *Golem) ClearThatPatternCache() {
	if g.thatPatternCache != nil {
		g.thatPatternCache.ClearCache()
	}
}

// InvalidateThatPatternContext invalidates that pattern cache entries for a specific context
func (g *Golem) InvalidateThatPatternContext(context string) {
	if g.thatPatternCache != nil {
		g.thatPatternCache.InvalidateContext(context)
	}
}

// GetTemplateTagProcessingCacheStats returns template tag processing cache statistics
func (g *Golem) GetTemplateTagProcessingCacheStats() map[string]interface{} {
	if g.templateTagProcessingCache != nil {
		return g.templateTagProcessingCache.GetCacheStats()
	}
	return map[string]interface{}{
		"results":        0,
		"max_size":       0,
		"ttl_seconds":    0,
		"hits":           map[string]int{},
		"misses":         0,
		"hit_rate":       0.0,
		"total_requests": 0,
		"tag_types":      0,
		"context_hashes": 0,
	}
}

// ClearTemplateTagProcessingCache clears the template tag processing cache
func (g *Golem) ClearTemplateTagProcessingCache() {
	if g.templateTagProcessingCache != nil {
		g.templateTagProcessingCache.ClearCache()
	}
}

// InvalidateTemplateTagType invalidates template tag processing cache entries for a specific tag type
func (g *Golem) InvalidateTemplateTagType(tagType string) {
	if g.templateTagProcessingCache != nil {
		g.templateTagProcessingCache.InvalidateTagType(tagType)
	}
}

// InvalidateTemplateTagContext invalidates template tag processing cache entries for a specific context
func (g *Golem) InvalidateTemplateTagContext(context string) {
	if g.templateTagProcessingCache != nil {
		g.templateTagProcessingCache.InvalidateContext(context)
	}
}

// GetPatternMatchingCacheStats returns pattern matching cache statistics
func (g *Golem) GetPatternMatchingCacheStats() map[string]interface{} {
	if g.patternMatchingCache != nil {
		return g.patternMatchingCache.GetCacheStats()
	}
	return map[string]interface{}{
		"pattern_priorities":  0,
		"wildcard_matches":    0,
		"set_regexes":         0,
		"exact_match_keys":    0,
		"max_size":            0,
		"ttl_seconds":         0,
		"hits":                map[string]int{},
		"misses":              0,
		"hit_rate":            0.0,
		"total_requests":      0,
		"knowledge_base_hash": "",
		"set_hashes":          0,
	}
}

// ClearPatternMatchingCache clears the pattern matching cache
func (g *Golem) ClearPatternMatchingCache() {
	if g.patternMatchingCache != nil {
		g.patternMatchingCache.ClearCache()
	}
}

// InvalidatePatternMatchingKnowledgeBase invalidates pattern matching cache when knowledge base changes
func (g *Golem) InvalidatePatternMatchingKnowledgeBase() {
	if g.patternMatchingCache != nil && g.aimlKB != nil {
		// Generate a simple hash of the knowledge base state
		kbHash := g.generateKnowledgeBaseHash()
		g.patternMatchingCache.InvalidateKnowledgeBase(kbHash)
	}
}

// InvalidatePatternMatchingSet invalidates pattern matching cache when a set changes
func (g *Golem) InvalidatePatternMatchingSet(setName string) {
	if g.patternMatchingCache != nil {
		g.patternMatchingCache.InvalidateSet(setName)
	}
}

// generateKnowledgeBaseHash creates a simple hash of the knowledge base state
func (g *Golem) generateKnowledgeBaseHash() string {
	if g.aimlKB == nil {
		return ""
	}

	// Create a simple hash based on pattern count and set count
	patternCount := len(g.aimlKB.Patterns)
	setCount := 0
	if g.aimlKB.Sets != nil {
		setCount = len(g.aimlKB.Sets)
	}

	return fmt.Sprintf("patterns:%d,sets:%d", patternCount, setCount)
}

// NewPatternMatchingCache creates a new pattern matching cache
func NewPatternMatchingCache(maxSize int, ttlSeconds int64) *PatternMatchingCache {
	return &PatternMatchingCache{
		PatternPriorities: make(map[string]PatternPriorityInfo),
		WildcardMatches:   make(map[string]WildcardMatchResult),
		SetRegexes:        make(map[string]string),
		ExactMatchKeys:    make(map[string]string),
		Hits:              make(map[string]int),
		Misses:            0,
		MaxSize:           maxSize,
		TTL:               ttlSeconds,
		Timestamps:        make(map[string]time.Time),
		AccessOrder:       make([]string, 0),
		KnowledgeBaseHash: "",
		SetHashes:         make(map[string]string),
	}
}

// GetPatternPriority returns a cached pattern priority or calculates and caches it
func (cache *PatternMatchingCache) GetPatternPriority(pattern string) (PatternPriorityInfo, bool) {
	cache.mutex.RLock()
	if priority, exists := cache.PatternPriorities[pattern]; exists {
		// Check TTL
		if time.Since(cache.Timestamps[pattern]).Seconds() < float64(cache.TTL) {
			cache.mutex.RUnlock()
			// Need write lock to update access order
			cache.mutex.Lock()
			cache.updateAccessOrder(pattern)
			cache.Hits[pattern]++
			cache.mutex.Unlock()
			return priority, true
		}
		// TTL expired, remove from cache
		cache.mutex.RUnlock()
		cache.mutex.Lock()
		cache.removePatternPriority(pattern)
		cache.mutex.Unlock()
	} else {
		cache.mutex.RUnlock()
	}

	cache.mutex.Lock()
	cache.Misses++
	cache.mutex.Unlock()
	return PatternPriorityInfo{}, false
}

// SetPatternPriority caches a pattern priority
func (cache *PatternMatchingCache) SetPatternPriority(pattern string, priority PatternPriorityInfo) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// Evict if cache is full
	if len(cache.PatternPriorities) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.PatternPriorities[pattern] = priority
	cache.Timestamps[pattern] = time.Now()
	cache.updateAccessOrder(pattern)
}

// GetWildcardMatch returns a cached wildcard match result
func (cache *PatternMatchingCache) GetWildcardMatch(input, pattern string) (WildcardMatchResult, bool) {
	cacheKey := fmt.Sprintf("%s|%s", input, pattern)

	cache.mutex.RLock()
	if result, exists := cache.WildcardMatches[cacheKey]; exists {
		// Check TTL
		if time.Since(cache.Timestamps[cacheKey]).Seconds() < float64(cache.TTL) {
			cache.mutex.RUnlock()
			// Need write lock to update access order
			cache.mutex.Lock()
			cache.updateAccessOrder(cacheKey)
			cache.Hits[cacheKey]++
			cache.mutex.Unlock()
			return result, true
		}
		// TTL expired, remove from cache
		cache.mutex.RUnlock()
		cache.mutex.Lock()
		cache.removeWildcardMatch(cacheKey)
		cache.mutex.Unlock()
	} else {
		cache.mutex.RUnlock()
	}

	cache.mutex.Lock()
	cache.Misses++
	cache.mutex.Unlock()
	return WildcardMatchResult{}, false
}

// SetWildcardMatch caches a wildcard match result
func (cache *PatternMatchingCache) SetWildcardMatch(input, pattern string, result WildcardMatchResult) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cacheKey := fmt.Sprintf("%s|%s", input, pattern)

	// Evict if cache is full
	if len(cache.WildcardMatches) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.WildcardMatches[cacheKey] = result
	cache.Timestamps[cacheKey] = time.Now()
	cache.updateAccessOrder(cacheKey)
}

// GetSetRegex returns a cached set regex
func (cache *PatternMatchingCache) GetSetRegex(setName string, setContent []string) (string, bool) {
	// Create content hash for set validation
	contentHash := cache.generateSetContentHash(setContent)

	// Check if set regex is cached and content hasn't changed
	if regex, exists := cache.SetRegexes[setName]; exists {
		if cache.SetHashes[setName] == contentHash {
			// Update access order for LRU
			cache.updateAccessOrder(setName)
			cache.Hits[setName]++
			return regex, true
		}
		// Set content changed, remove from cache
		cache.removeSetRegex(setName)
	}

	cache.Misses++
	return "", false
}

// SetSetRegex caches a set regex
func (cache *PatternMatchingCache) SetSetRegex(setName string, setContent []string, regex string) {
	contentHash := cache.generateSetContentHash(setContent)

	// Evict if cache is full
	if len(cache.SetRegexes) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.SetRegexes[setName] = regex
	cache.SetHashes[setName] = contentHash
	cache.Timestamps[setName] = time.Now()
	cache.updateAccessOrder(setName)
}

// GetExactMatchKey returns a cached exact match key
func (cache *PatternMatchingCache) GetExactMatchKey(input, topic, that string, thatIndex int) (string, bool) {
	cacheKey := cache.generateExactMatchKey(input, topic, that, thatIndex)

	if key, exists := cache.ExactMatchKeys[cacheKey]; exists {
		// Check TTL
		if time.Since(cache.Timestamps[cacheKey]).Seconds() < float64(cache.TTL) {
			// Update access order for LRU
			cache.updateAccessOrder(cacheKey)
			cache.Hits[cacheKey]++
			return key, true
		}
		// TTL expired, remove from cache
		cache.removeExactMatchKey(cacheKey)
	}

	cache.Misses++
	return "", false
}

// SetExactMatchKey caches an exact match key
func (cache *PatternMatchingCache) SetExactMatchKey(input, topic, that string, thatIndex int, key string) {
	cacheKey := cache.generateExactMatchKey(input, topic, that, thatIndex)

	// Evict if cache is full
	if len(cache.ExactMatchKeys) >= cache.MaxSize {
		cache.evictLRU()
	}

	cache.ExactMatchKeys[cacheKey] = key
	cache.Timestamps[cacheKey] = time.Now()
	cache.updateAccessOrder(cacheKey)
}

// generateSetContentHash creates a hash of set content for validation
func (cache *PatternMatchingCache) generateSetContentHash(setContent []string) string {
	// Sort content for consistent hashing
	sortedContent := make([]string, len(setContent))
	copy(sortedContent, setContent)
	sort.Strings(sortedContent)
	return strings.Join(sortedContent, "|")
}

// generateExactMatchKey creates a cache key for exact match lookups
func (cache *PatternMatchingCache) generateExactMatchKey(input, topic, that string, thatIndex int) string {
	return fmt.Sprintf("%s|%s|%s|%d", input, topic, that, thatIndex)
}

// updateAccessOrder updates the LRU access order
func (cache *PatternMatchingCache) updateAccessOrder(key string) {
	// Remove from current position
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
	// Add to end (most recently used)
	cache.AccessOrder = append(cache.AccessOrder, key)
}

// removePatternPriority removes a pattern priority from the cache
func (cache *PatternMatchingCache) removePatternPriority(pattern string) {
	delete(cache.PatternPriorities, pattern)
	delete(cache.Timestamps, pattern)
	delete(cache.Hits, pattern)
	cache.removeFromAccessOrder(pattern)
}

// removeWildcardMatch removes a wildcard match from the cache
func (cache *PatternMatchingCache) removeWildcardMatch(cacheKey string) {
	delete(cache.WildcardMatches, cacheKey)
	delete(cache.Timestamps, cacheKey)
	delete(cache.Hits, cacheKey)
	cache.removeFromAccessOrder(cacheKey)
}

// removeSetRegex removes a set regex from the cache
func (cache *PatternMatchingCache) removeSetRegex(setName string) {
	delete(cache.SetRegexes, setName)
	delete(cache.SetHashes, setName)
	delete(cache.Timestamps, setName)
	delete(cache.Hits, setName)
	cache.removeFromAccessOrder(setName)
}

// removeExactMatchKey removes an exact match key from the cache
func (cache *PatternMatchingCache) removeExactMatchKey(cacheKey string) {
	delete(cache.ExactMatchKeys, cacheKey)
	delete(cache.Timestamps, cacheKey)
	delete(cache.Hits, cacheKey)
	cache.removeFromAccessOrder(cacheKey)
}

// removeFromAccessOrder removes a key from the access order
func (cache *PatternMatchingCache) removeFromAccessOrder(key string) {
	for i, k := range cache.AccessOrder {
		if k == key {
			cache.AccessOrder = append(cache.AccessOrder[:i], cache.AccessOrder[i+1:]...)
			break
		}
	}
}

// evictLRU removes the least recently used item
func (cache *PatternMatchingCache) evictLRU() {
	if len(cache.AccessOrder) == 0 {
		return
	}

	// Remove the first (oldest) item
	oldestKey := cache.AccessOrder[0]

	// Remove from all caches
	cache.removePatternPriority(oldestKey)
	cache.removeWildcardMatch(oldestKey)
	cache.removeSetRegex(oldestKey)
	cache.removeExactMatchKey(oldestKey)
}

// GetCacheStats returns cache statistics
func (cache *PatternMatchingCache) GetCacheStats() map[string]interface{} {
	totalRequests := cache.Misses
	for _, hits := range cache.Hits {
		totalRequests += hits
	}

	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(len(cache.Hits)) / float64(totalRequests)
	}

	return map[string]interface{}{
		"pattern_priorities":  len(cache.PatternPriorities),
		"wildcard_matches":    len(cache.WildcardMatches),
		"set_regexes":         len(cache.SetRegexes),
		"exact_match_keys":    len(cache.ExactMatchKeys),
		"max_size":            cache.MaxSize,
		"ttl_seconds":         cache.TTL,
		"hits":                cache.Hits,
		"misses":              cache.Misses,
		"hit_rate":            hitRate,
		"total_requests":      totalRequests,
		"knowledge_base_hash": cache.KnowledgeBaseHash,
		"set_hashes":          len(cache.SetHashes),
	}
}

// ClearCache clears the pattern matching cache
func (cache *PatternMatchingCache) ClearCache() {
	cache.PatternPriorities = make(map[string]PatternPriorityInfo)
	cache.WildcardMatches = make(map[string]WildcardMatchResult)
	cache.SetRegexes = make(map[string]string)
	cache.ExactMatchKeys = make(map[string]string)
	cache.Hits = make(map[string]int)
	cache.Misses = 0
	cache.Timestamps = make(map[string]time.Time)
	cache.AccessOrder = make([]string, 0)
	cache.KnowledgeBaseHash = ""
	cache.SetHashes = make(map[string]string)
}

// InvalidateKnowledgeBase invalidates all caches when knowledge base changes
func (cache *PatternMatchingCache) InvalidateKnowledgeBase(newHash string) {
	if cache.KnowledgeBaseHash != newHash {
		cache.ClearCache()
		cache.KnowledgeBaseHash = newHash
	}
}

// InvalidateSet invalidates set-related caches when a set changes
func (cache *PatternMatchingCache) InvalidateSet(setName string) {
	cache.removeSetRegex(setName)
	// Also invalidate wildcard matches that might use this set
	for key := range cache.WildcardMatches {
		if strings.Contains(key, setName) {
			cache.removeWildcardMatch(key)
		}
	}
}

// CachedNormalizePattern normalizes AIML patterns with caching
func (g *Golem) CachedNormalizePattern(pattern string) string {
	if g.textNormalizationCache != nil {
		if result, err := g.textNormalizationCache.GetNormalizedText(g, pattern, "NormalizePattern"); err == nil {
			return result
		}
	}
	// Fallback to direct normalization with loaded substitutions
	normalized := NormalizePattern(pattern)
	return g.applyLoadedSubstitutions(normalized)
}

// CachedNormalizeForMatchingCasePreserving normalizes text for pattern matching with case preservation and caching
func (g *Golem) CachedNormalizeForMatchingCasePreserving(input string) string {
	if g.textNormalizationCache != nil {
		if result, err := g.textNormalizationCache.GetNormalizedText(g, input, "NormalizeForMatchingCasePreserving"); err == nil {
			return result
		}
	}
	// Fallback to direct normalization
	return NormalizeForMatchingCasePreserving(input)
}

// CachedNormalizeThatPattern normalizes that patterns with caching
func (g *Golem) CachedNormalizeThatPattern(pattern string) string {
	if g.textNormalizationCache != nil {
		if result, err := g.textNormalizationCache.GetNormalizedText(g, pattern, "NormalizeThatPattern"); err == nil {
			return result
		}
	}
	// Fallback to direct normalization
	return NormalizeThatPattern(pattern)
}

// CachedNormalizeForMatching normalizes text for matching with caching
func (g *Golem) CachedNormalizeForMatching(input string) string {
	if g.textNormalizationCache != nil {
		if result, err := g.textNormalizationCache.GetNormalizedText(g, input, "normalizeForMatching"); err == nil {
			return result
		}
	}
	// Fallback to direct normalization with loaded substitutions
	return g.normalizeForMatchingWithSubstitutions(input)
}

// CachedExpandContractions expands contractions with caching
func (g *Golem) CachedExpandContractions(text string) string {
	if g.textNormalizationCache != nil {
		if result, err := g.textNormalizationCache.GetNormalizedText(g, text, "expandContractions"); err == nil {
			return result
		}
	}
	// Fallback to direct expansion
	return expandContractions(text)
}

// GetTemplateCacheStats returns template cache statistics
func (g *Golem) GetTemplateCacheStats() map[string]interface{} {
	if g.templateCache == nil {
		return map[string]interface{}{
			"cache_size": 0,
			"hits":       0,
			"misses":     0,
			"hit_rate":   0.0,
		}
	}

	totalRequests := g.templateMetrics.CacheHits + g.templateMetrics.CacheMisses
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(g.templateMetrics.CacheHits) / float64(totalRequests)
	}

	return map[string]interface{}{
		"cache_size":      len(g.templateCache.Cache),
		"max_size":        g.templateCache.MaxSize,
		"ttl_seconds":     g.templateCache.TTL,
		"hits":            g.templateMetrics.CacheHits,
		"misses":          g.templateMetrics.CacheMisses,
		"hit_rate":        hitRate,
		"total_processed": g.templateMetrics.TotalProcessed,
		"average_time_ms": g.templateMetrics.AverageProcessTime,
		"error_count":     g.templateMetrics.ErrorCount,
	}
}

// ResetTemplateMetrics resets template processing metrics
func (g *Golem) ResetTemplateMetrics() {
	if g.templateMetrics != nil {
		g.templateMetrics.TotalProcessed = 0
		g.templateMetrics.AverageProcessTime = 0.0
		g.templateMetrics.CacheHits = 0
		g.templateMetrics.CacheMisses = 0
		g.templateMetrics.CacheHitRate = 0.0
		g.templateMetrics.TagProcessingTimes = make(map[string]float64)
		g.templateMetrics.ErrorCount = 0
		g.templateMetrics.LastProcessed = ""
		g.templateMetrics.MemoryPeak = 0
		g.templateMetrics.ParallelOps = 0
	}
}

// generateTemplateCacheKey creates a cache key from template, wildcards and minimal ctx
func (g *Golem) generateTemplateCacheKey(template string, wildcards map[string]string, ctx *VariableContext) string {
	// Build a deterministic key: template + sorted wildcards + session/topic markers + state-dependent data
	var b strings.Builder
	b.WriteString("tpl:")
	b.WriteString(template)
	b.WriteString("|wc:")
	// sort wildcards for deterministic order
	keys := make([]string, 0, len(wildcards))
	for k := range wildcards {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(wildcards[k])
		b.WriteString(";")
	}
	if ctx != nil && ctx.Session != nil {
		b.WriteString("|sid:")
		b.WriteString(ctx.Session.ID)
	}
	if ctx != nil {
		b.WriteString("|topic:")
		b.WriteString(ctx.Topic)
	}

	// Include state-dependent data in cache key
	if ctx != nil && ctx.KnowledgeBase != nil {
		// Include array state for templates that reference arrays
		if strings.Contains(template, "<array ") {
			b.WriteString("|arrays:")
			arrayKeys := make([]string, 0, len(ctx.KnowledgeBase.Arrays))
			for k := range ctx.KnowledgeBase.Arrays {
				arrayKeys = append(arrayKeys, k)
			}
			sort.Strings(arrayKeys)
			for _, k := range arrayKeys {
				b.WriteString(k)
				b.WriteString("=")
				b.WriteString(strings.Join(ctx.KnowledgeBase.Arrays[k], ","))
				b.WriteString(";")
			}
		}

		// Include set state for templates that reference sets
		if strings.Contains(template, "<set ") {
			b.WriteString("|sets:")
			setKeys := make([]string, 0, len(ctx.KnowledgeBase.Sets))
			for k := range ctx.KnowledgeBase.Sets {
				setKeys = append(setKeys, k)
			}
			sort.Strings(setKeys)
			for _, k := range setKeys {
				b.WriteString(k)
				b.WriteString("=")
				b.WriteString(strings.Join(ctx.KnowledgeBase.Sets[k], ","))
				b.WriteString(";")
			}
		}

		// Include map state for templates that reference maps
		if strings.Contains(template, "<map ") {
			b.WriteString("|maps:")
			mapKeys := make([]string, 0, len(ctx.KnowledgeBase.Maps))
			for k := range ctx.KnowledgeBase.Maps {
				mapKeys = append(mapKeys, k)
			}
			sort.Strings(mapKeys)
			for _, k := range mapKeys {
				b.WriteString(k)
				b.WriteString("=")
				// Sort map entries for deterministic order
				entryKeys := make([]string, 0, len(ctx.KnowledgeBase.Maps[k]))
				for ek := range ctx.KnowledgeBase.Maps[k] {
					entryKeys = append(entryKeys, ek)
				}
				sort.Strings(entryKeys)
				for _, ek := range entryKeys {
					b.WriteString(ek)
					b.WriteString(":")
					b.WriteString(ctx.KnowledgeBase.Maps[k][ek])
					b.WriteString(",")
				}
				b.WriteString(";")
			}
		}

		// Include list state for templates that reference lists
		if strings.Contains(template, "<list ") {
			b.WriteString("|lists:")
			listKeys := make([]string, 0, len(ctx.KnowledgeBase.Lists))
			for k := range ctx.KnowledgeBase.Lists {
				listKeys = append(listKeys, k)
			}
			sort.Strings(listKeys)
			for _, k := range listKeys {
				b.WriteString(k)
				b.WriteString("=")
				b.WriteString(strings.Join(ctx.KnowledgeBase.Lists[k], ","))
				b.WriteString(";")
			}
		}
	}

	return b.String()
}

// getFromTemplateCache fetches a cached response if present
func (g *Golem) getFromTemplateCache(key string) (string, bool) {
	if g.templateCache == nil {
		return "", false
	}

	g.templateCache.mutex.RLock()
	v, ok := g.templateCache.Cache[key]
	if ok {
		g.templateCache.mutex.RUnlock()
		// Need write lock to update hits
		g.templateCache.mutex.Lock()
		g.templateCache.Hits[key] = g.templateCache.Hits[key] + 1
		g.templateCache.mutex.Unlock()
	} else {
		g.templateCache.mutex.RUnlock()
	}
	return v, ok
}

// storeInTemplateCache stores a processed template result
func (g *Golem) storeInTemplateCache(key, value string) {
	if g.templateCache == nil {
		return
	}

	g.templateCache.mutex.Lock()
	defer g.templateCache.mutex.Unlock()

	// Evict if over capacity (simple FIFO by timestamps if needed)
	if len(g.templateCache.Cache) >= g.templateCache.MaxSize {
		// naive eviction: remove an arbitrary oldest by timestamp string comparison
		var oldestKey string
		var oldestTs string
		for k, ts := range g.templateCache.Timestamps {
			if oldestTs == "" || ts < oldestTs {
				oldestTs = ts
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(g.templateCache.Cache, oldestKey)
			delete(g.templateCache.Timestamps, oldestKey)
			delete(g.templateCache.Hits, oldestKey)
		}
	}
	g.templateCache.Cache[key] = value
	// store a simple increasing timestamp using TotalProcessed to keep it consistent
	g.templateMetrics.TotalProcessed++
	g.templateCache.Timestamps[key] = strconv.Itoa(g.templateMetrics.TotalProcessed)
	if _, exists := g.templateCache.Hits[key]; !exists {
		g.templateCache.Hits[key] = 0
	}
}

// updateCacheHitRate recomputes cache hit rate metric
func (g *Golem) updateCacheHitRate() {
	total := g.templateMetrics.CacheHits + g.templateMetrics.CacheMisses
	if total > 0 {
		g.templateMetrics.CacheHitRate = float64(g.templateMetrics.CacheHits) / float64(total)
	} else {
		g.templateMetrics.CacheHitRate = 0
	}
}
