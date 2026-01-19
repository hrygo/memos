package store

import "context"

// MemoEmbedding represents the vector embedding of a memo.
type MemoEmbedding struct {
	ID        int32
	MemoID    int32
	Embedding []float32 // 1024-dimensional vector
	Model     string    // Model identifier, e.g., "BAAI/bge-m3"
	CreatedTs int64
	UpdatedTs int64
}

// FindMemoEmbedding is the find condition for memo embeddings.
type FindMemoEmbedding struct {
	MemoID *int32
	Model  *string
}

// MemoWithScore represents a vector search result with similarity score.
type MemoWithScore struct {
	Memo  *Memo
	Score float32 // Similarity score (0-1, higher is more similar)
}

// VectorSearchOptions represents the options for vector search.
type VectorSearchOptions struct {
	UserID int32     // Required, only search memos of this user
	Vector []float32 // Query vector
	Limit  int       // Number of results to return, default 10
}

// UpsertMemoEmbedding inserts or updates a memo embedding.
func (s *Store) UpsertMemoEmbedding(ctx context.Context, embedding *MemoEmbedding) (*MemoEmbedding, error) {
	return s.driver.UpsertMemoEmbedding(ctx, embedding)
}

// GetMemoEmbedding gets the embedding of a specific memo.
func (s *Store) GetMemoEmbedding(ctx context.Context, memoID int32, model string) (*MemoEmbedding, error) {
	list, err := s.driver.ListMemoEmbeddings(ctx, &FindMemoEmbedding{
		MemoID: &memoID,
		Model:  &model,
	})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// ListMemoEmbeddings lists memo embeddings.
func (s *Store) ListMemoEmbeddings(ctx context.Context, find *FindMemoEmbedding) ([]*MemoEmbedding, error) {
	return s.driver.ListMemoEmbeddings(ctx, find)
}

// DeleteMemoEmbedding deletes a memo embedding.
func (s *Store) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
	return s.driver.DeleteMemoEmbedding(ctx, memoID)
}

// VectorSearch performs vector similarity search.
func (s *Store) VectorSearch(ctx context.Context, opts *VectorSearchOptions) ([]*MemoWithScore, error) {
	return s.driver.VectorSearch(ctx, opts)
}
