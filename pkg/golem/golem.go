package golem

import (
	"container/list"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// TextNormalizationCache represents a cache for text normalization results
type TextNormalizationCache struct {
	Results     map[string]string    `json:"results"`
	Hits        map[string]int       `json:"hits"`
	Misses      int                  `json:"misses"`
	MaxSize     int                  `json:"max_size"`
	TTL         int64                `json:"ttl_seconds"`
	Timestamps  map[string]time.Time `json:"timestamps"`
	AccessOrder []string             `json:"access_order"` // Kept for JSON compat; not populated
	// Mutex for thread safety
	mutex    sync.RWMutex
	lruList  *list.List
	lruIndex map[string]*list.Element
}

// VariableResolutionCache represents a cache for variable resolution results
type VariableResolutionCache struct {
	Results     map[string]string    `json:"results"`
	Hits        map[string]int       `json:"hits"`
	Misses      int                  `json:"misses"`
	MaxSize     int                  `json:"max_size"`
	TTL         int64                `json:"ttl_seconds"`
	Timestamps  map[string]time.Time `json:"timestamps"`
	AccessOrder []string             `json:"access_order"` // Kept for JSON compat; not populated
	// Scope tracking for cache invalidation
	ScopeHashes map[string]string `json:"scope_hashes"` // Maps cache key to scope hash
	lruList     *list.List
	lruIndex    map[string]*list.Element
}

// TemplateTagProcessingCache represents a cache for template tag processing results
type TemplateTagProcessingCache struct {
	Results     map[string]string    `json:"results"`
	Hits        map[string]int       `json:"hits"`
	Misses      int                  `json:"misses"`
	MaxSize     int                  `json:"max_size"`
	TTL         int64                `json:"ttl_seconds"`
	Timestamps  map[string]time.Time `json:"timestamps"`
	AccessOrder []string             `json:"access_order"` // Kept for JSON compat; not populated
	// Tag type tracking for cache invalidation
	TagTypes map[string]string `json:"tag_types"` // Maps cache key to tag type
	// Context tracking for cache invalidation
	ContextHashes map[string]string `json:"context_hashes"` // Maps cache key to context hash
	// Mutex for thread safety
	mutex    sync.RWMutex
	lruList  *list.List
	lruIndex map[string]*list.Element
}

// WildcardMatchResult represents the result of a wildcard pattern match
type WildcardMatchResult struct {
	Matched   bool              `json:"matched"`
	Wildcards map[string]string `json:"wildcards"`
	Pattern   string            `json:"pattern"`
	Input     string            `json:"input"`
	Regex     string            `json:"regex"`
}

// ChatSession represents a single chat session
type ChatSession struct {
	// mu makes concurrent ProcessInput calls on one session run one at a time
	mu sync.Mutex

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

// WithLock runs fn while no ProcessInput call is using the session, so fn can
// read or change its fields safely while other goroutines chat on it. fn must
// not call ProcessInput (or WithLock) on the same session, which would deadlock.
func (session *ChatSession) WithLock(fn func()) {
	session.mu.Lock()
	defer session.mu.Unlock()
	fn()
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
	// kbMu guards replacing aimlKB. ProcessInput holds it for reading for the
	// whole call, so the knowledge base is not swapped (or merged into a copy,
	// losing concurrent <learn>s) while a request is using it.
	kbMu sync.RWMutex
	// Text processing components
	sentenceSplitter     *SentenceSplitter
	wordBoundaryDetector *WordBoundaryDetector
	// Text normalization result cache
	textNormalizationCache *TextNormalizationCache
	// Variable resolution cache
	variableResolutionCache *VariableResolutionCache
	// Template tag processing cache
	templateTagProcessingCache *TemplateTagProcessingCache
	// Persistent learning components
	persistentLearning *PersistentLearningManager
	// Enhanced context resolution components
	fuzzyMatcher    *FuzzyContextMatcher
	semanticMatcher *SemanticContextMatcher
	// Random seed for deterministic shuffling
	randomSeed int64
	// Tree-based processing components
	treeProcessor *TreeProcessor
}

// NewTextNormalizationCache creates a new text normalization cache
func NewTextNormalizationCache(maxSize int, ttlSeconds int64) *TextNormalizationCache {
	return &TextNormalizationCache{
		Results:    make(map[string]string),
		Hits:       make(map[string]int),
		MaxSize:    maxSize,
		TTL:        ttlSeconds,
		Timestamps: make(map[string]time.Time),
		lruList:    list.New(),
		lruIndex:   make(map[string]*list.Element),
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

// updateAccessOrder updates the LRU access order in O(1).
func (cache *TextNormalizationCache) updateAccessOrder(key string) {
	if elem, ok := cache.lruIndex[key]; ok {
		cache.lruList.MoveToBack(elem)
	} else {
		cache.lruIndex[key] = cache.lruList.PushBack(key)
	}
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
	if elem, ok := cache.lruIndex[key]; ok {
		cache.lruList.Remove(elem)
		delete(cache.lruIndex, key)
	}
}

// evictLRU removes the least recently used result in O(1).
func (cache *TextNormalizationCache) evictLRU() {
	front := cache.lruList.Front()
	if front == nil {
		return
	}
	cache.removeResult(front.Value.(string))
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
	cache.lruList = list.New()
	cache.lruIndex = make(map[string]*list.Element)
}

// NewVariableResolutionCache creates a new variable resolution cache
func NewVariableResolutionCache(maxSize int, ttlSeconds int64) *VariableResolutionCache {
	return &VariableResolutionCache{
		Results:     make(map[string]string),
		Hits:        make(map[string]int),
		MaxSize:     maxSize,
		TTL:         ttlSeconds,
		Timestamps:  make(map[string]time.Time),
		ScopeHashes: make(map[string]string),
		lruList:     list.New(),
		lruIndex:    make(map[string]*list.Element),
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

// updateAccessOrder updates the LRU access order in O(1).
func (cache *VariableResolutionCache) updateAccessOrder(key string) {
	if elem, ok := cache.lruIndex[key]; ok {
		cache.lruList.MoveToBack(elem)
	} else {
		cache.lruIndex[key] = cache.lruList.PushBack(key)
	}
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
	if elem, ok := cache.lruIndex[key]; ok {
		cache.lruList.Remove(elem)
		delete(cache.lruIndex, key)
	}
}

// evictLRU removes the least recently used result in O(1).
func (cache *VariableResolutionCache) evictLRU() {
	front := cache.lruList.Front()
	if front == nil {
		return
	}
	cache.removeResult(front.Value.(string))
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
	cache.ScopeHashes = make(map[string]string)
	cache.lruList = list.New()
	cache.lruIndex = make(map[string]*list.Element)
}

// NewTemplateTagProcessingCache creates a new template tag processing cache
func NewTemplateTagProcessingCache(maxSize int, ttlSeconds int64) *TemplateTagProcessingCache {
	return &TemplateTagProcessingCache{
		Results:       make(map[string]string),
		Hits:          make(map[string]int),
		MaxSize:       maxSize,
		TTL:           ttlSeconds,
		Timestamps:    make(map[string]time.Time),
		TagTypes:      make(map[string]string),
		ContextHashes: make(map[string]string),
		lruList:       list.New(),
		lruIndex:      make(map[string]*list.Element),
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

// updateAccessOrder updates the LRU access order in O(1).
// Note: This method assumes the caller holds the appropriate lock
func (cache *TemplateTagProcessingCache) updateAccessOrder(key string) {
	if elem, ok := cache.lruIndex[key]; ok {
		cache.lruList.MoveToBack(elem)
	} else {
		cache.lruIndex[key] = cache.lruList.PushBack(key)
	}
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
	if elem, ok := cache.lruIndex[key]; ok {
		cache.lruList.Remove(elem)
		delete(cache.lruIndex, key)
	}
}

// evictLRU removes the least recently used result in O(1).
// Note: This method assumes the caller holds the appropriate lock
func (cache *TemplateTagProcessingCache) evictLRU() {
	front := cache.lruList.Front()
	if front == nil {
		return
	}
	cache.removeResult(front.Value.(string))
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
	cache.TagTypes = make(map[string]string)
	cache.ContextHashes = make(map[string]string)
	cache.lruList = list.New()
	cache.lruIndex = make(map[string]*list.Element)
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

	// Create persistent learning manager with default storage path
	persistentLearning := NewPersistentLearningManager("./learned_categories")

	// Create text normalization result cache
	textNormalizationCache := NewTextNormalizationCache(1000, 1800) // 1000 results, 30 minutes TTL

	// Create variable resolution cache
	variableResolutionCache := NewVariableResolutionCache(500, 900) // 500 results, 15 minutes TTL

	// Create template tag processing cache
	templateTagProcessingCache := NewTemplateTagProcessingCache(1000, 1800) // 1000 results, 30 minutes TTL

	g := &Golem{
		verbose:                    verbose,
		logLevel:                   logLevel,
		logger:                     logger,
		sessions:                   make(map[string]*ChatSession),
		sessionID:                  1,
		oobMgr:                     oobMgr,
		sraixMgr:                   sraixMgr,
		sentenceSplitter:           sentenceSplitter,
		wordBoundaryDetector:       wordBoundaryDetector,
		textNormalizationCache:     textNormalizationCache,
		variableResolutionCache:    variableResolutionCache,
		templateTagProcessingCache: templateTagProcessingCache,
		persistentLearning:         persistentLearning,
	}
	// Created up front so concurrent template processing never races to
	// initialize it
	g.treeProcessor = NewTreeProcessor(g)
	return g
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
		g.aimlKB.addOrUpdateCategory(category)
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

	return g.persistentLearning.SavePersistentCategories(g.aimlKB.snapshotCategories(), source)
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
		g.aimlKB.removeCategoryByKey(categoryKey(category))
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
		"total_categories": g.aimlKB.categoryCount(),
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
//
// As AIML 2.0 specifies, input is split into sentences, each sentence is
// matched and answered in turn, and the responses are joined.
func (g *Golem) ProcessInput(input string, session *ChatSession) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	g.kbMu.RLock()
	defer g.kbMu.RUnlock()

	if g.aimlKB == nil {
		return "", fmt.Errorf("no AIML knowledge base loaded")
	}

	sentences := inputSentences(input)
	if len(sentences) <= 1 {
		return g.processSentence(input, session)
	}
	var responses []string
	var firstErr error
	for _, sentence := range sentences {
		response, err := g.processSentence(sentence, session)
		if err != nil {
			g.LogInfo("No response for sentence %q: %v", sentence, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if response != "" {
			responses = append(responses, response)
		}
	}
	if len(responses) == 0 && firstErr != nil {
		return "", firstErr
	}
	return strings.Join(responses, " "), nil
}

// inputSentenceEndRe matches the end of a sentence in user input
var inputSentenceEndRe = regexp.MustCompile(`[.!?]+(\s+|$)`)

// inputSentences splits user input into its sentences
func inputSentences(input string) []string {
	var sentences []string
	for _, sentence := range inputSentenceEndRe.Split(strings.TrimSpace(input), -1) {
		if sentence = strings.TrimSpace(sentence); sentence != "" {
			sentences = append(sentences, sentence)
		}
	}
	return sentences
}

// processSentence matches and answers one sentence of input
func (g *Golem) processSentence(input string, session *ChatSession) (string, error) {

	g.LogInfo("Processing input: %s", input)

	// Normalize input
	normalizedInput := g.CachedNormalizePattern(input)

	// Get current topic and that context
	currentTopic := session.GetSessionTopic()
	lastThat := session.GetLastThat()

	// Normalize the that context for matching using enhanced that normalization
	normalizedThat := ""
	if lastThat != "" {
		normalizedThat = g.normalizeThatContext(lastThat)
	}

	// Try to match pattern with full context (using index 0 for last response)
	category, wildcards, err := g.aimlKB.MatchPatternWithTopicAndThatIndexOriginalCached(g, normalizedInput, input, currentTopic, normalizedThat, 0)
	if err != nil {
		return "", err
	}

	// Process template with context
	response := g.ProcessTemplateWithContext(category.Template, wildcards, session)

	// Add to history
	session.History = append(session.History, input)
	session.LastActivity = time.Now().Format(time.RFC3339)

	// Add to request history for <request> tag support
	session.AddToRequestHistory(input)

	// The response is what the next input's <that> is matched against
	if response != "" {
		session.AddToThatHistory(response)
	}

	// Add to response history for <response> tag support
	session.AddToResponseHistory(response)

	return response, nil
}

// ProcessInputWithThatIndex processes user input with specific that context index
func (g *Golem) ProcessInputWithThatIndex(input string, session *ChatSession, thatIndex int) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	g.kbMu.RLock()
	defer g.kbMu.RUnlock()

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
		normalizedThat = g.normalizeThatContext(thatContext)
	}

	// Try to match pattern with full context and specific that index
	category, wildcards, err := g.aimlKB.MatchPatternWithTopicAndThatIndexOriginalCached(g, normalizedInput, input, currentTopic, normalizedThat, thatIndex)
	if err != nil {
		return "", err
	}

	// Process template with context
	response := g.ProcessTemplateWithContext(category.Template, wildcards, session)

	// Add to history
	session.History = append(session.History, input)
	session.LastActivity = time.Now().Format(time.RFC3339)

	// Add to request history for <request> tag support
	session.AddToRequestHistory(input)

	// The response is what the next input's <that> is matched against
	if response != "" {
		session.AddToThatHistory(response)
	}

	// Add to response history for <response> tag support
	session.AddToResponseHistory(response)

	return response, nil
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
	g.kbMu.Lock()
	defer g.kbMu.Unlock()

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
			// Add persistent categories to the knowledge base. addOrUpdateCategory
			// keeps the pattern index valid when Categories is reallocated.
			for _, category := range persistentCategories {
				g.aimlKB.addOrUpdateCategory(category)
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
		g.sessionMutex.Lock()
		sessionID = fmt.Sprintf("session_%d", g.sessionID)
		g.sessionID++
		g.sessionMutex.Unlock()
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
	g.ensureKnowledgeBase()

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

// thatSentenceEndRe matches the end of a sentence inside a bot response
var thatSentenceEndRe = regexp.MustCompile(`[.!?]+\s+`)

// lastSentence returns the last sentence of a bot response, which is what a
// <that> pattern is matched against
func lastSentence(response string) string {
	response = strings.TrimSpace(response)
	ends := thatSentenceEndRe.FindAllStringIndex(response, -1)
	if len(ends) == 0 {
		return response
	}
	return response[ends[len(ends)-1][1]:]
}

// normalizeThatContext normalizes the last sentence of a bot response for
// <that> matching
func (g *Golem) normalizeThatContext(response string) string {
	return g.CachedNormalizeThatPattern(lastSentence(response))
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
