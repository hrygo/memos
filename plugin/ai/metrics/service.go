package metrics

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/usememos/memos/store"
)

// ErrMetricsNotConfigured is returned when metrics persistence is not configured.
var ErrMetricsNotConfigured = errors.New("metrics persistence not configured (requires PostgreSQL)")

// Service implements the MetricsService interface with real storage.
type Service struct {
	store      *store.Store
	aggregator *Aggregator
	persister  *Persister
}

// NewService creates a new metrics service.
// If store is nil, metrics will only be aggregated in memory (no persistence).
func NewService(s *store.Store, cfg PersisterConfig) *Service {
	aggregator := NewAggregator()

	svc := &Service{
		store:      s,
		aggregator: aggregator,
	}

	if s != nil {
		svc.persister = NewPersister(s, aggregator, cfg)
		svc.persister.Start()
	} else {
		slog.Warn("metrics service initialized without store (persistence disabled)")
	}

	return svc
}

// Close stops the metrics service and flushes remaining data.
func (s *Service) Close() {
	if s.persister != nil {
		s.persister.Close()
	}
}

// RecordRequest records an agent request metric.
func (s *Service) RecordRequest(_ context.Context, agentType string, latency time.Duration, success bool) {
	s.aggregator.RecordAgentRequest(agentType, latency, success)
}

// RecordToolCall records a tool call metric.
func (s *Service) RecordToolCall(_ context.Context, toolName string, latency time.Duration, success bool) {
	s.aggregator.RecordToolCall(toolName, latency, success)
}

// GetStats retrieves aggregated statistics for the given time range.
func (s *Service) GetStats(ctx context.Context, timeRange TimeRange) (*AgentMetrics, error) {
	// Start with current in-memory stats
	stats := s.aggregator.GetCurrentStats()

	// If no store, return memory-only stats
	if s.store == nil {
		return stats, nil
	}

	// Query persisted metrics from database
	agentMetrics, err := s.store.ListAgentMetrics(ctx, &store.FindAgentMetrics{
		StartTime: &timeRange.Start,
		EndTime:   &timeRange.End,
		Limit:     1000,
	})
	if err != nil {
		// Log error but return in-memory stats
		slog.Warn("failed to query persisted agent metrics", "error", err)
		return stats, nil
	}

	// Aggregate persisted metrics by agent type
	type dbAgg struct {
		totalRequests int64
		totalSuccess  int64
		latencySum    int64
	}
	dbAggs := make(map[string]*dbAgg)

	for _, m := range agentMetrics {
		stats.RequestCount += m.RequestCount
		stats.SuccessCount += m.SuccessCount

		agg, exists := dbAggs[m.AgentType]
		if !exists {
			agg = &dbAgg{}
			dbAggs[m.AgentType] = agg
		}
		agg.totalRequests += m.RequestCount
		agg.totalSuccess += m.SuccessCount
		agg.latencySum += m.LatencySumMs
	}

	// Merge DB aggregations into stats
	for agentType, agg := range dbAggs {
		agentStat, exists := stats.AgentStats[agentType]
		if !exists {
			agentStat = &AgentStat{}
			stats.AgentStats[agentType] = agentStat
		}
		agentStat.Count += agg.totalRequests
		totalReqs := agentStat.Count
		if totalReqs > 0 {
			// Recalculate success rate based on combined totals
			memSuccess := int64(agentStat.SuccessRate * float32(agentStat.Count-agg.totalRequests))
			agentStat.SuccessRate = float32(memSuccess+agg.totalSuccess) / float32(totalReqs)
			// Average latency from DB only (memory stats have their own calculation)
			if agg.latencySum > 0 && agg.totalRequests > 0 {
				agentStat.AvgLatency = time.Duration(agg.latencySum/agg.totalRequests) * time.Millisecond
			}
		}
	}

	return stats, nil
}

// Flush forces an immediate flush of metrics to the database.
func (s *Service) Flush(ctx context.Context) error {
	if s.persister == nil {
		return ErrMetricsNotConfigured
	}
	return s.persister.Flush(ctx)
}

// HasPersistence returns true if metrics persistence is enabled.
func (s *Service) HasPersistence() bool {
	return s.persister != nil
}
