package sqlite

import (
	"context"
	"errors"

	"github.com/usememos/memos/store"
)

// ============================================================================
// EPISODIC MEMORY - NOT SUPPORTED IN SQLITE
// ============================================================================
// Episodic memory is an AI feature that requires PostgreSQL.
// SQLite users should use PostgreSQL for AI features.
// ============================================================================

var errAIFeatureNotSupported = errors.New("AI features (episodic memory, user preferences) are not supported in SQLite; please use PostgreSQL")

func (d *DB) CreateEpisodicMemory(ctx context.Context, create *store.EpisodicMemory) (*store.EpisodicMemory, error) {
	return nil, errAIFeatureNotSupported
}

func (d *DB) ListEpisodicMemories(ctx context.Context, find *store.FindEpisodicMemory) ([]*store.EpisodicMemory, error) {
	return nil, errAIFeatureNotSupported
}

func (d *DB) DeleteEpisodicMemory(ctx context.Context, delete *store.DeleteEpisodicMemory) error {
	return errAIFeatureNotSupported
}

func (d *DB) UpsertUserPreferences(ctx context.Context, upsert *store.UpsertUserPreferences) (*store.UserPreferences, error) {
	return nil, errAIFeatureNotSupported
}

func (d *DB) GetUserPreferences(ctx context.Context, find *store.FindUserPreferences) (*store.UserPreferences, error) {
	return nil, errAIFeatureNotSupported
}
