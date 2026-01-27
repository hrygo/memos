package sqlite

import (
	"context"

	"github.com/usememos/memos/store"
)

func (d *DB) UpsertAgentMetrics(_ context.Context, _ *store.UpsertAgentMetrics) (*store.AgentMetrics, error) {
	return nil, errAIFeatureNotSupported
}

func (d *DB) ListAgentMetrics(_ context.Context, _ *store.FindAgentMetrics) ([]*store.AgentMetrics, error) {
	return nil, errAIFeatureNotSupported
}

func (d *DB) DeleteAgentMetrics(_ context.Context, _ *store.DeleteAgentMetrics) error {
	return errAIFeatureNotSupported
}

func (d *DB) UpsertToolMetrics(_ context.Context, _ *store.UpsertToolMetrics) (*store.ToolMetrics, error) {
	return nil, errAIFeatureNotSupported
}

func (d *DB) ListToolMetrics(_ context.Context, _ *store.FindToolMetrics) ([]*store.ToolMetrics, error) {
	return nil, errAIFeatureNotSupported
}

func (d *DB) DeleteToolMetrics(_ context.Context, _ *store.DeleteToolMetrics) error {
	return errAIFeatureNotSupported
}
