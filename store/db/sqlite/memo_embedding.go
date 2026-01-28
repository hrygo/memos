package sqlite

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// ============================================================================
// SQLITE AI FEATURES SUPPORT (Limited - High ROI Only)
// ============================================================================
// SQLite does NOT support vector search (no pgvector equivalent).
// Full-text search is provided on a best-effort basis (FTS5 if available).
//
// For full AI features, use PostgreSQL.
// ============================================================================

// UpsertMemoEmbedding is NOT supported for SQLite.
// Vector storage requires PostgreSQL with pgvector extension.
func (d *DB) UpsertMemoEmbedding(ctx context.Context, embedding *store.MemoEmbedding) (*store.MemoEmbedding, error) {
	return nil, errors.New("memo embedding (vector storage) requires PostgreSQL with pgvector extension")
}

// ListMemoEmbeddings is NOT supported for SQLite.
func (d *DB) ListMemoEmbeddings(ctx context.Context, find *store.FindMemoEmbedding) ([]*store.MemoEmbedding, error) {
	return nil, errors.New("memo embedding (vector storage) requires PostgreSQL with pgvector extension")
}

// DeleteMemoEmbedding is NOT supported for SQLite.
func (d *DB) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
	// Return nil (success) to allow cascade delete to work
	return nil
}

// VectorSearch is NOT supported for SQLite.
// Vector similarity search requires PostgreSQL with pgvector extension.
func (d *DB) VectorSearch(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
	return nil, errors.New("vector search requires PostgreSQL with pgvector extension")
}

// FindMemosWithoutEmbedding is NOT supported for SQLite.
func (d *DB) FindMemosWithoutEmbedding(ctx context.Context, find *store.FindMemosWithoutEmbedding) ([]*store.Memo, error) {
	return nil, errors.New("memo embedding features require PostgreSQL with pgvector extension")
}

// BM25Search performs full-text search using SQLite FTS5 if available.
// This is a best-effort implementation - for production use, prefer PostgreSQL.
func (d *DB) BM25Search(ctx context.Context, opts *store.BM25SearchOptions) ([]*store.BM25Result, error) {
	// Try using FTS5 full-text search
	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			bm25(memo_fts) AS score
		FROM memo m
		LEFT JOIN memo_fts ON m.id = memo_fts.rowid
		WHERE m.creator_id = $1
			AND m.row_status = 'NORMAL'
			AND memo_fts MATCH $2
		ORDER BY score DESC, m.updated_ts DESC
		LIMIT $3
	`

	rows, err := d.db.QueryContext(ctx, query, opts.UserID, opts.Query, opts.Limit)
	if err != nil {
		// FTS5 might not be enabled, fall back to LIKE search
		return d.bm25SearchFallback(ctx, opts)
	}
	defer rows.Close()

	results := []*store.BM25Result{}
	for rows.Next() {
		var result store.BM25Result
		var memo store.Memo
		var payloadBytes []byte

		err := rows.Scan(
			&memo.ID,
			&memo.UID,
			&memo.CreatorID,
			&memo.CreatedTs,
			&memo.UpdatedTs,
			&memo.RowStatus,
			&memo.Visibility,
			&memo.Pinned,
			&memo.Content,
			&payloadBytes,
			&result.Score,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan BM25 result")
		}

		// Parse payload
		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		result.Memo = &memo

		// Apply minimum score filter
		if result.Score >= opts.MinScore {
			results = append(results, &result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// bm25SearchFallback provides a simple LIKE-based search when FTS5 is not available.
// This is a fallback implementation with limited functionality.
func (d *DB) bm25SearchFallback(ctx context.Context, opts *store.BM25SearchOptions) ([]*store.BM25Result, error) {
	// Split query into words for LIKE search
	words := []string{}
	fields := strings.Fields(opts.Query)
	for _, word := range fields {
		if len(word) > 0 {
			// Escape LIKE special characters to prevent pattern injection
			escaped := strings.ReplaceAll(strings.ReplaceAll(word, "%", "\\%"), "_", "\\_")
			words = append(words, "%"+escaped+"%")
		}
	}

	if len(words) == 0 {
		return []*store.BM25Result{}, nil
	}

	// Build WHERE clause
	whereClause := strings.Repeat("AND m.content LIKE ? ", len(words))
	args := make([]any, 0, len(words)+1)
	args = append(args, opts.UserID)
	for _, word := range words {
		args = append(args, word)
	}
	args = append(args, opts.Limit)

	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			COUNT(*) AS score
		FROM memo m
		WHERE m.creator_id = ?
			AND m.row_status = 'NORMAL'
			` + whereClause + `
		GROUP BY m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload
		ORDER BY score DESC, m.updated_ts DESC
		LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to BM25 search fallback")
	}
	defer rows.Close()

	results := []*store.BM25Result{}
	for rows.Next() {
		var result store.BM25Result
		var memo store.Memo
		var payloadBytes []byte

		err := rows.Scan(
			&memo.ID,
			&memo.UID,
			&memo.CreatorID,
			&memo.CreatedTs,
			&memo.UpdatedTs,
			&memo.RowStatus,
			&memo.Visibility,
			&memo.Pinned,
			&memo.Content,
			&payloadBytes,
			&result.Score,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan BM25 result")
		}

		// Parse payload
		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		result.Memo = &memo

		// Apply minimum score filter
		if result.Score >= opts.MinScore {
			results = append(results, &result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
