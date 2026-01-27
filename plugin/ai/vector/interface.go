// Package vector provides the vector retrieval service interface for AI agents.
// This interface is consumed by Team C (Memo Enhancement).
package vector

import "context"

// VectorService defines the vector retrieval service interface.
// Consumers: Team C (Memo Enhancement)
type VectorService interface {
	// StoreEmbedding stores a vector embedding with metadata.
	StoreEmbedding(ctx context.Context, docID string, vector []float32, metadata map[string]any) error

	// SearchSimilar performs similarity search on vectors.
	// filter: filter conditions (user_id, created_after, etc.)
	SearchSimilar(ctx context.Context, vector []float32, limit int, filter map[string]any) ([]VectorResult, error)

	// HybridSearch performs hybrid search combining vector and keyword search.
	HybridSearch(ctx context.Context, query string, limit int) ([]SearchResult, error)
}

// VectorResult represents a vector search result.
type VectorResult struct {
	DocID    string         `json:"doc_id"`
	Score    float32        `json:"score"` // similarity score 0-1
	Metadata map[string]any `json:"metadata"`
}

// SearchResult represents a hybrid search result.
type SearchResult struct {
	Name      string  `json:"name"` // memo UID
	Content   string  `json:"content"`
	Score     float32 `json:"score"`
	MatchType string  `json:"match_type"` // vector/keyword/hybrid
}
