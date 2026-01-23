// Package agent provides metrics collection for the scheduler agent.
// This module tracks execution performance, tool usage, and business metrics
// for monitoring and observability.
package agent

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"
)

// AgentMetrics collects and tracks agent execution metrics.
// All operations are thread-safe for concurrent access.
type AgentMetrics struct {
	mu sync.RWMutex

	// Execution metrics
	executionDuration []time.Duration // Recent execution durations
	iterationCount    []int           // Recent iteration counts
	maxDurationSamples int            // Max samples to keep

	// Success/failure counters
	totalExecutions     atomic.Int64
	successfulExecutions atomic.Int64
	failedExecutions    atomic.Int64

	// Tool metrics
	toolCalls    map[string]*atomic.Int64   // Tool call counts
	toolFailures map[string]*atomic.Int64   // Tool failure counts
	toolLatency  map[string][]time.Duration // Tool call latencies

	// Business metrics
	schedulesCreated  atomic.Int64
	conflictsDetected atomic.Int64
	conflictsResolved atomic.Int64
	findFreeTimeCalls atomic.Int64

	// Error class metrics
	transientErrors  atomic.Int64
	permanentErrors  atomic.Int64
	conflictErrors   atomic.Int64

	// Cache metrics
	cacheHits atomic.Int64
	cacheMisses atomic.Int64
}

// NewAgentMetrics creates a new metrics collector.
func NewAgentMetrics() *AgentMetrics {
	m := &AgentMetrics{
		executionDuration:  make([]time.Duration, 0, 100),
		iterationCount:     make([]int, 0, 100),
		maxDurationSamples: 100,
		toolCalls:          make(map[string]*atomic.Int64),
		toolFailures:       make(map[string]*atomic.Int64),
		toolLatency:        make(map[string][]time.Duration),
	}
	return m
}

// RecordExecution records a completed agent execution.
func (m *AgentMetrics) RecordExecution(duration time.Duration, iterations int, success bool) {
	m.totalExecutions.Add(1)
	if success {
		m.successfulExecutions.Add(1)
	} else {
		m.failedExecutions.Add(1)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Keep only the last N samples
	if len(m.executionDuration) >= m.maxDurationSamples {
		m.executionDuration = m.executionDuration[1:]
		m.iterationCount = m.iterationCount[1:]
	}
	m.executionDuration = append(m.executionDuration, duration)
	m.iterationCount = append(m.iterationCount, iterations)
}

// RecordToolCall records a tool execution attempt.
func (m *AgentMetrics) RecordToolCall(tool string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize counters if needed
	if m.toolCalls[tool] == nil {
		m.toolCalls[tool] = &atomic.Int64{}
		m.toolFailures[tool] = &atomic.Int64{}
		m.toolLatency[tool] = make([]time.Duration, 0, 50)
	}

	m.toolCalls[tool].Add(1)
	if !success {
		m.toolFailures[tool].Add(1)
	}

	// Track latency (keep last 50 samples)
	if len(m.toolLatency[tool]) >= 50 {
		m.toolLatency[tool] = m.toolLatency[tool][1:]
	}
	m.toolLatency[tool] = append(m.toolLatency[tool], duration)
}

// RecordScheduleCreated records a schedule creation.
func (m *AgentMetrics) RecordScheduleCreated() {
	m.schedulesCreated.Add(1)
}

// RecordConflictDetected records a conflict detection.
func (m *AgentMetrics) RecordConflictDetected() {
	m.conflictsDetected.Add(1)
}

// RecordConflictResolved records a conflict resolution.
func (m *AgentMetrics) RecordConflictResolved() {
	m.conflictsResolved.Add(1)
}

// RecordFindFreeTimeCall records a find_free_time tool call.
func (m *AgentMetrics) RecordFindFreeTimeCall() {
	m.findFreeTimeCalls.Add(1)
}

// RecordErrorClass records an error by its class.
func (m *AgentMetrics) RecordErrorClass(class ErrorClass) {
	switch class {
	case ErrorClassTransient:
		m.transientErrors.Add(1)
	case ErrorClassPermanent:
		m.permanentErrors.Add(1)
	case ErrorClassConflict:
		m.conflictErrors.Add(1)
	}
}

// RecordCacheHit records a cache hit.
func (m *AgentMetrics) RecordCacheHit() {
	m.cacheHits.Add(1)
}

// RecordCacheMiss records a cache miss.
func (m *AgentMetrics) RecordCacheMiss() {
	m.cacheMisses.Add(1)
}

// GetSuccessRate returns the success rate as a percentage (0-100).
func (m *AgentMetrics) GetSuccessRate() float64 {
	total := m.totalExecutions.Load()
	if total == 0 {
		return 0
	}
	successful := m.successfulExecutions.Load()
	return float64(successful) / float64(total) * 100
}

// GetAverageDuration returns the average execution duration.
func (m *AgentMetrics) GetAverageDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.executionDuration) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range m.executionDuration {
		sum += d
	}
	return sum / time.Duration(len(m.executionDuration))
}

// GetAverageIterations returns the average iteration count.
func (m *AgentMetrics) GetAverageIterations() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.iterationCount) == 0 {
		return 0
	}

	var sum int
	for _, i := range m.iterationCount {
		sum += i
	}
	return float64(sum) / float64(len(m.iterationCount))
}

// GetP95Duration returns the 95th percentile execution duration.
func (m *AgentMetrics) GetP95Duration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.executionDuration) == 0 {
		return 0
	}

	// Simple approach: sort and get 95th percentile
	sorted := make([]time.Duration, len(m.executionDuration))
	copy(sorted, m.executionDuration)

	// Simple bubble sort (good enough for small N)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	idx := int(float64(len(sorted)) * 0.95)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// GetToolStats returns statistics for a specific tool.
func (m *AgentMetrics) GetToolStats(tool string) ToolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := ToolStats{Name: tool}

	if counter := m.toolCalls[tool]; counter != nil {
		stats.TotalCalls = counter.Load()
	}
	if counter := m.toolFailures[tool]; counter != nil {
		stats.Failures = counter.Load()
	}

	if stats.TotalCalls > 0 {
		stats.SuccessRate = 100 - (float64(stats.Failures) / float64(stats.TotalCalls) * 100)
	}

	if latencies, ok := m.toolLatency[tool]; ok && len(latencies) > 0 {
		var sum time.Duration
		for _, l := range latencies {
			sum += l
		}
		stats.AverageLatency = sum / time.Duration(len(latencies))
	}

	return stats
}

// GetAllToolStats returns statistics for all tools.
func (m *AgentMetrics) GetAllToolStats() []ToolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make([]ToolStats, 0, len(m.toolCalls))
	for tool := range m.toolCalls {
		stats = append(stats, m.GetToolStats(tool))
	}
	return stats
}

// GetSummary returns a summary of all metrics.
func (m *AgentMetrics) GetSummary() MetricsSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsSummary{
		TotalExecutions:      m.totalExecutions.Load(),
		SuccessfulExecutions: m.successfulExecutions.Load(),
		FailedExecutions:     m.failedExecutions.Load(),
		SuccessRate:          m.GetSuccessRate(),
		AverageDuration:      m.GetAverageDuration(),
		P95Duration:          m.GetP95Duration(),
		AverageIterations:    m.GetAverageIterations(),
		SchedulesCreated:     m.schedulesCreated.Load(),
		ConflictsDetected:    m.conflictsDetected.Load(),
		ConflictsResolved:    m.conflictsResolved.Load(),
		FindFreeTimeCalls:    m.findFreeTimeCalls.Load(),
		TransientErrors:      m.transientErrors.Load(),
		PermanentErrors:      m.permanentErrors.Load(),
		ConflictErrors:       m.conflictErrors.Load(),
		CacheHits:            m.cacheHits.Load(),
		CacheMisses:          m.cacheMisses.Load(),
	}
}

// LogSummary logs the current metrics summary.
func (m *AgentMetrics) LogSummary() {
	summary := m.GetSummary()
	slog.Info("agent_metrics_summary",
		"total_executions", summary.TotalExecutions,
		"success_rate", fmtFloat(summary.SuccessRate),
		"avg_duration_ms", summary.AverageDuration.Milliseconds(),
		"p95_duration_ms", summary.P95Duration.Milliseconds(),
		"avg_iterations", fmtFloat(summary.AverageIterations),
		"schedules_created", summary.SchedulesCreated,
		"conflicts_detected", summary.ConflictsDetected,
		"conflicts_resolved", summary.ConflictsResolved,
		"transient_errors", summary.TransientErrors,
		"permanent_errors", summary.PermanentErrors,
		"conflict_errors", summary.ConflictErrors,
		"cache_hit_rate", fmtFloat(m.GetCacheHitRate()),
	)
}

// GetCacheHitRate returns the cache hit rate as a percentage.
func (m *AgentMetrics) GetCacheHitRate() float64 {
	hits := m.cacheHits.Load()
	misses := m.cacheMisses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// ToolStats represents statistics for a single tool.
type ToolStats struct {
	Name            string
	TotalCalls      int64
	Failures        int64
	SuccessRate     float64
	AverageLatency  time.Duration
}

// MetricsSummary represents a summary of all metrics.
type MetricsSummary struct {
	TotalExecutions      int64
	SuccessfulExecutions int64
	FailedExecutions     int64
	SuccessRate          float64
	AverageDuration      time.Duration
	P95Duration          time.Duration
	AverageIterations    float64
	SchedulesCreated     int64
	ConflictsDetected    int64
	ConflictsResolved    int64
	FindFreeTimeCalls    int64
	TransientErrors      int64
	PermanentErrors      int64
	ConflictErrors       int64
	CacheHits            int64
	CacheMisses          int64
}

// fmtFloat formats a float value with 2 decimal places.
func fmtFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// Global metrics instance (can be shared across agents)
var (
	globalMetrics     *AgentMetrics
	globalMetricsOnce sync.Once
)

// GetGlobalMetrics returns the global metrics instance.
func GetGlobalMetrics() *AgentMetrics {
	globalMetricsOnce.Do(func() {
		globalMetrics = NewAgentMetrics()
	})
	return globalMetrics
}
