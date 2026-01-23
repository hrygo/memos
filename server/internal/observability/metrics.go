package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects and aggregates metrics for AI operations.
type Metrics struct {
	mu sync.Mutex

	// Counters
	requestTotal  atomic.Int64
	requestFailed atomic.Int64
	streamChunks  atomic.Int64

	// Agent-specific metrics
	agentMetrics map[string]*AgentMetrics

	// Duration histogram data (simplified for internal use)
	durations []time.Duration
	maxDurations int
}

// AgentMetrics represents metrics for a specific agent type.
type AgentMetrics struct {
	executionCount atomic.Int64
	totalDuration  atomic.Int64 // milliseconds
	errorCount     atomic.Int64
}

// NewMetrics creates a new metrics collector.
func NewMetrics(maxDurations int) *Metrics {
	if maxDurations <= 0 {
		maxDurations = 1000 // Default to keeping last 1000 durations
	}
	return &Metrics{
		agentMetrics:  make(map[string]*AgentMetrics),
		durations:     make([]time.Duration, 0, maxDurations),
		maxDurations:  maxDurations,
	}
}

// Global metrics instance.
var globalMetrics = NewMetrics(1000)

// GlobalMetrics returns the global metrics instance.
func GlobalMetrics() *Metrics {
	return globalMetrics
}

// RecordRequest records a request.
func (m *Metrics) RecordRequest(agentType string) {
	m.requestTotal.Add(1)
	m.getAgentMetrics(agentType).executionCount.Add(1)
}

// RecordFailure records a failed request.
func (m *Metrics) RecordFailure(agentType string) {
	m.requestFailed.Add(1)
	m.getAgentMetrics(agentType).errorCount.Add(1)
}

// RecordDuration records a request duration.
func (m *Metrics) RecordDuration(agentType string, duration time.Duration) {
	m.mu.Lock()
	if len(m.durations) >= m.maxDurations {
		// Remove oldest duration (FIFO)
		m.durations = m.durations[1:]
	}
	m.durations = append(m.durations, duration)

	am := m.getAgentMetrics(agentType)
	am.totalDuration.Add(int64(duration.Milliseconds()))
	m.mu.Unlock()
}

// RecordStreamChunk records a stream chunk sent.
func (m *Metrics) RecordStreamChunk() {
	m.streamChunks.Add(1)
}

// GetRequestTotal returns the total number of requests.
func (m *Metrics) GetRequestTotal() int64 {
	return m.requestTotal.Load()
}

// GetRequestFailed returns the total number of failed requests.
func (m *Metrics) GetRequestFailed() int64 {
	return m.requestFailed.Load()
}

// GetStreamChunks returns the total number of stream chunks sent.
func (m *Metrics) GetStreamChunks() int64 {
	return m.streamChunks.Load()
}

// GetAgentMetrics returns metrics for a specific agent type.
func (m *Metrics) GetAgentMetrics(agentType string) *AgentMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	am, ok := m.agentMetrics[agentType]
	if !ok {
		// Create and store in map to avoid race condition
		am = &AgentMetrics{}
		m.agentMetrics[agentType] = am
	}
	return am
}

// getAgentMetrics gets or creates agent metrics (internal use, assumes lock held).
func (m *Metrics) getAgentMetrics(agentType string) *AgentMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.agentMetrics[agentType]; !ok {
		m.agentMetrics[agentType] = &AgentMetrics{}
	}
	return m.agentMetrics[agentType]
}

// GetAverageDuration returns the average duration in milliseconds for an agent type.
func (m *Metrics) GetAverageDuration(agentType string) int64 {
	am := m.GetAgentMetrics(agentType)
	count := am.executionCount.Load()
	if count == 0 {
		return 0
	}
	total := am.totalDuration.Load()
	return total / count
}

// GetAllAgentTypes returns all agent types that have been recorded.
func (m *Metrics) GetAllAgentTypes() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	types := make([]string, 0, len(m.agentMetrics))
	for agentType := range m.agentMetrics {
		types = append(types, agentType)
	}
	return types
}

// Reset resets all metrics (useful for testing).
func (m *Metrics) Reset() {
	m.requestTotal.Store(0)
	m.requestFailed.Store(0)
	m.streamChunks.Store(0)

	m.mu.Lock()
	m.agentMetrics = make(map[string]*AgentMetrics)
	m.durations = make([]time.Duration, 0, m.maxDurations)
	m.mu.Unlock()
}

// Snapshot returns a snapshot of current metrics.
func (m *Metrics) Snapshot() *MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentSnapshots := make(map[string]*AgentMetricsSnapshot, len(m.agentMetrics))
	for agentType, am := range m.agentMetrics {
		agentSnapshots[agentType] = &AgentMetricsSnapshot{
			ExecutionCount: am.executionCount.Load(),
			TotalDuration:  am.totalDuration.Load(),
			ErrorCount:     am.errorCount.Load(),
			AverageDuration: func() int64 {
				count := am.executionCount.Load()
				if count == 0 {
					return 0
				}
				return am.totalDuration.Load() / count
			}(),
		}
	}

	return &MetricsSnapshot{
		RequestTotal:   m.requestTotal.Load(),
		RequestFailed:  m.requestFailed.Load(),
		StreamChunks:   m.streamChunks.Load(),
		AgentMetrics:   agentSnapshots,
		DurationCount:  len(m.durations),
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics.
type MetricsSnapshot struct {
	RequestTotal   int64
	RequestFailed  int64
	StreamChunks   int64
	AgentMetrics   map[string]*AgentMetricsSnapshot
	DurationCount  int
}

// AgentMetricsSnapshot represents metrics for a specific agent.
type AgentMetricsSnapshot struct {
	ExecutionCount  int64
	TotalDuration   int64
	ErrorCount      int64
	AverageDuration int64
}

// SuccessRate returns the success rate as a percentage (0-100).
func (s *MetricsSnapshot) SuccessRate() float64 {
	if s.RequestTotal == 0 {
		return 100.0
	}
	return float64(s.RequestTotal-s.RequestFailed) / float64(s.RequestTotal) * 100.0
}
