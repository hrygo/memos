package mysql

import (
	"context"

	"github.com/pkg/errors"

	"github.com/usememos/memos/store"
)

// UpsertMemoEmbedding is not supported for MySQL.
// AI features require PostgreSQL with pgvector extension.
func (d *DB) UpsertMemoEmbedding(ctx context.Context, embedding *store.MemoEmbedding) (*store.MemoEmbedding, error) {
	return nil, errors.New("memo embedding requires PostgreSQL database with pgvector extension")
}

// ListMemoEmbeddings is not supported for MySQL.
func (d *DB) ListMemoEmbeddings(ctx context.Context, find *store.FindMemoEmbedding) ([]*store.MemoEmbedding, error) {
	return nil, errors.New("memo embedding requires PostgreSQL database with pgvector extension")
}

// DeleteMemoEmbedding is not supported for MySQL.
func (d *DB) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
	// Return nil (success) to allow cascade delete to work
	return nil
}

// VectorSearch is not supported for MySQL.
// MySQL does not have native vector similarity search capabilities like pgvector.
func (d *DB) VectorSearch(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
	return nil, errors.New("vector search is not supported for MySQL database, please use PostgreSQL with pgvector extension")
}

// FindMemosWithoutEmbedding is not supported for MySQL.
func (d *DB) FindMemosWithoutEmbedding(ctx context.Context, find *store.FindMemosWithoutEmbedding) ([]*store.Memo, error) {
	return nil, errors.New("memo embedding features require PostgreSQL database with pgvector extension")
}
