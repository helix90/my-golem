package golem

import (
	"sync"
	"time"
)

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

// ProcessorRegistry counts calls to the tree processor's logical processors
// (wildcard, data, format, variable, logic)
type ProcessorRegistry struct {
	order     []string
	metrics   map[string]*ProcessorMetrics
	metricsMu sync.Mutex // guards metrics, which are updated by concurrent template processing
}

// NewProcessorRegistry creates a registry for the named processors, kept in the given order
func NewProcessorRegistry(names ...string) *ProcessorRegistry {
	r := &ProcessorRegistry{
		order:   names,
		metrics: make(map[string]*ProcessorMetrics, len(names)),
	}
	for _, name := range names {
		r.metrics[name] = &ProcessorMetrics{}
	}
	return r
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

// GetProcessorStats returns detailed statistics for all processors
func (g *Golem) GetProcessorStats() map[string]interface{} {
	// Ensure TreeProcessor is initialized
	if g.treeProcessor == nil {
		g.treeProcessor = NewTreeProcessor(g)
	}

	// Return TreeProcessor metrics (tree-based AST processing is now the only method)
	if g.treeProcessor.metrics != nil {
		stats := make(map[string]interface{})

		for name, metrics := range g.treeProcessor.metrics.GetMetrics() {
			stats[name] = map[string]interface{}{
				"total_calls":     metrics.TotalCalls,
				"total_time_ms":   metrics.TotalTime.Milliseconds(),
				"average_time_ms": metrics.AverageTime.Milliseconds(),
				"last_call_time":  metrics.LastCallTime.Format(time.RFC3339),
				"error_count":     metrics.ErrorCount,
				"cache_hits":      metrics.CacheHits,
				"cache_misses":    metrics.CacheMisses,
			}
			// Avoid divide by zero
			if metrics.CacheHits+metrics.CacheMisses > 0 {
				hitRate := float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses)
				stats[name].(map[string]interface{})["hit_rate"] = hitRate
			} else {
				stats[name].(map[string]interface{})["hit_rate"] = 0.0
			}
		}

		return stats
	}

	// Return empty map if metrics not initialized
	return make(map[string]interface{})
}

// ResetProcessorMetrics resets metrics for all processors
func (g *Golem) ResetProcessorMetrics() {
	// Ensure TreeProcessor is initialized
	if g.treeProcessor == nil {
		g.treeProcessor = NewTreeProcessor(g)
	}

	// Reset TreeProcessor metrics (tree-based AST processing is now the only method)
	if g.treeProcessor.metrics != nil {
		g.treeProcessor.metrics.ResetMetrics()
	}
}

// GetProcessingOrder returns the current processing order
func (g *Golem) GetProcessingOrder() []string {
	// Ensure TreeProcessor is initialized
	if g.treeProcessor == nil {
		g.treeProcessor = NewTreeProcessor(g)
	}

	// Return TreeProcessor's logical processor order (tree-based AST processing is now the only method)
	if g.treeProcessor.metrics != nil {
		return g.treeProcessor.metrics.order
	}

	// Return empty slice if metrics not initialized
	return []string{}
}
