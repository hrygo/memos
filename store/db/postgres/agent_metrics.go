package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *DB) UpsertAgentMetrics(ctx context.Context, upsert *store.UpsertAgentMetrics) (*store.AgentMetrics, error) {
	if upsert == nil {
		return nil, fmt.Errorf("upsert parameter cannot be nil")
	}

	query := `
		INSERT INTO agent_metrics (hour_bucket, agent_type, request_count, success_count, latency_sum_ms, latency_p50_ms, latency_p95_ms, errors)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (hour_bucket, agent_type) DO UPDATE SET
			request_count = agent_metrics.request_count + EXCLUDED.request_count,
			success_count = agent_metrics.success_count + EXCLUDED.success_count,
			latency_sum_ms = agent_metrics.latency_sum_ms + EXCLUDED.latency_sum_ms,
			latency_p50_ms = EXCLUDED.latency_p50_ms,
			latency_p95_ms = EXCLUDED.latency_p95_ms,
			errors = EXCLUDED.errors
		RETURNING id, hour_bucket, agent_type, request_count, success_count, latency_sum_ms, latency_p50_ms, latency_p95_ms, errors
	`

	var metrics store.AgentMetrics
	err := d.db.QueryRowContext(ctx, query,
		upsert.HourBucket, upsert.AgentType, upsert.RequestCount, upsert.SuccessCount,
		upsert.LatencySumMs, upsert.LatencyP50Ms, upsert.LatencyP95Ms, upsert.Errors,
	).Scan(
		&metrics.ID, &metrics.HourBucket, &metrics.AgentType,
		&metrics.RequestCount, &metrics.SuccessCount, &metrics.LatencySumMs,
		&metrics.LatencyP50Ms, &metrics.LatencyP95Ms, &metrics.Errors,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert agent metrics: %w", err)
	}

	return &metrics, nil
}

func (d *DB) ListAgentMetrics(ctx context.Context, find *store.FindAgentMetrics) ([]*store.AgentMetrics, error) {
	if find == nil {
		return nil, fmt.Errorf("find parameter cannot be nil")
	}

	where, args := []string{"1 = 1"}, []any{}
	argIndex := 1

	if find.AgentType != nil {
		where = append(where, fmt.Sprintf("agent_type = $%d", argIndex))
		args = append(args, *find.AgentType)
		argIndex++
	}
	if find.StartTime != nil {
		where = append(where, fmt.Sprintf("hour_bucket >= $%d", argIndex))
		args = append(args, *find.StartTime)
		argIndex++
	}
	if find.EndTime != nil {
		where = append(where, fmt.Sprintf("hour_bucket <= $%d", argIndex))
		args = append(args, *find.EndTime)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT id, hour_bucket, agent_type, request_count, success_count, latency_sum_ms, latency_p50_ms, latency_p95_ms, errors
		FROM agent_metrics
		WHERE %s
		ORDER BY hour_bucket DESC
	`, strings.Join(where, " AND "))

	limit := find.Limit
	if limit > 0 {
		if limit > 1000 {
			limit = 1000
		}
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*store.AgentMetrics
	for rows.Next() {
		var m store.AgentMetrics
		if err := rows.Scan(
			&m.ID, &m.HourBucket, &m.AgentType,
			&m.RequestCount, &m.SuccessCount, &m.LatencySumMs,
			&m.LatencyP50Ms, &m.LatencyP95Ms, &m.Errors,
		); err != nil {
			return nil, fmt.Errorf("failed to scan agent metrics: %w", err)
		}
		metrics = append(metrics, &m)
	}

	return metrics, rows.Err()
}

func (d *DB) DeleteAgentMetrics(ctx context.Context, delete *store.DeleteAgentMetrics) error {
	if delete == nil {
		return fmt.Errorf("delete parameter cannot be nil")
	}

	if delete.BeforeTime == nil {
		return fmt.Errorf("before_time is required for deletion")
	}

	query := `DELETE FROM agent_metrics WHERE hour_bucket < $1`
	_, err := d.db.ExecContext(ctx, query, *delete.BeforeTime)
	if err != nil {
		return fmt.Errorf("failed to delete agent metrics: %w", err)
	}

	return nil
}

func (d *DB) UpsertToolMetrics(ctx context.Context, upsert *store.UpsertToolMetrics) (*store.ToolMetrics, error) {
	if upsert == nil {
		return nil, fmt.Errorf("upsert parameter cannot be nil")
	}

	query := `
		INSERT INTO tool_metrics (hour_bucket, tool_name, call_count, success_count, latency_sum_ms)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (hour_bucket, tool_name) DO UPDATE SET
			call_count = tool_metrics.call_count + EXCLUDED.call_count,
			success_count = tool_metrics.success_count + EXCLUDED.success_count,
			latency_sum_ms = tool_metrics.latency_sum_ms + EXCLUDED.latency_sum_ms
		RETURNING id, hour_bucket, tool_name, call_count, success_count, latency_sum_ms
	`

	var metrics store.ToolMetrics
	err := d.db.QueryRowContext(ctx, query,
		upsert.HourBucket, upsert.ToolName, upsert.CallCount,
		upsert.SuccessCount, upsert.LatencySumMs,
	).Scan(
		&metrics.ID, &metrics.HourBucket, &metrics.ToolName,
		&metrics.CallCount, &metrics.SuccessCount, &metrics.LatencySumMs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert tool metrics: %w", err)
	}

	return &metrics, nil
}

func (d *DB) ListToolMetrics(ctx context.Context, find *store.FindToolMetrics) ([]*store.ToolMetrics, error) {
	if find == nil {
		return nil, fmt.Errorf("find parameter cannot be nil")
	}

	where, args := []string{"1 = 1"}, []any{}
	argIndex := 1

	if find.ToolName != nil {
		where = append(where, fmt.Sprintf("tool_name = $%d", argIndex))
		args = append(args, *find.ToolName)
		argIndex++
	}
	if find.StartTime != nil {
		where = append(where, fmt.Sprintf("hour_bucket >= $%d", argIndex))
		args = append(args, *find.StartTime)
		argIndex++
	}
	if find.EndTime != nil {
		where = append(where, fmt.Sprintf("hour_bucket <= $%d", argIndex))
		args = append(args, *find.EndTime)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT id, hour_bucket, tool_name, call_count, success_count, latency_sum_ms
		FROM tool_metrics
		WHERE %s
		ORDER BY hour_bucket DESC
	`, strings.Join(where, " AND "))

	limit := find.Limit
	if limit > 0 {
		if limit > 1000 {
			limit = 1000
		}
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tool metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*store.ToolMetrics
	for rows.Next() {
		var m store.ToolMetrics
		if err := rows.Scan(
			&m.ID, &m.HourBucket, &m.ToolName,
			&m.CallCount, &m.SuccessCount, &m.LatencySumMs,
		); err != nil {
			return nil, fmt.Errorf("failed to scan tool metrics: %w", err)
		}
		metrics = append(metrics, &m)
	}

	return metrics, rows.Err()
}

func (d *DB) DeleteToolMetrics(ctx context.Context, delete *store.DeleteToolMetrics) error {
	if delete == nil {
		return fmt.Errorf("delete parameter cannot be nil")
	}

	if delete.BeforeTime == nil {
		return fmt.Errorf("before_time is required for deletion")
	}

	query := `DELETE FROM tool_metrics WHERE hour_bucket < $1`
	_, err := d.db.ExecContext(ctx, query, *delete.BeforeTime)
	if err != nil {
		return fmt.Errorf("failed to delete tool metrics: %w", err)
	}

	return nil
}
