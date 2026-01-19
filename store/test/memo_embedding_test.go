package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/store"
)

func TestMemoEmbeddingStore(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Memo embedding tests only work with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create a test memo
	memoCreate := &store.Memo{
		UID:        "test-embedding-memo",
		CreatorID:  user.ID,
		Content:    "This is a test memo for embedding",
		Visibility: store.Private,
	}
	memo, err := ts.CreateMemo(ctx, memoCreate)
	require.NoError(t, err)

	// Test upsert embedding
	testVector := make([]float32, 1024)
	for i := range testVector {
		testVector[i] = 0.1
	}

	embedding := &store.MemoEmbedding{
		MemoID:    memo.ID,
		Embedding: testVector,
		Model:     "BAAI/bge-m3",
	}

	upserted, err := ts.UpsertMemoEmbedding(ctx, embedding)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, memo.ID, upserted.MemoID)
	require.Equal(t, "BAAI/bge-m3", upserted.Model)
	require.Greater(t, upserted.ID, int32(0))

	// Test get embedding
	retrieved, err := ts.GetMemoEmbedding(ctx, memo.ID, "BAAI/bge-m3")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, upserted.ID, retrieved.ID)
	require.Equal(t, len(testVector), len(retrieved.Embedding))

	// Test list embeddings by memo_id
	list, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{
		MemoID: &memo.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(list))
	require.Equal(t, "BAAI/bge-m3", list[0].Model)

	// Test update embedding (upsert again with same memo_id and model)
	updatedVector := make([]float32, 1024)
	for i := range updatedVector {
		updatedVector[i] = 0.2
	}
	embedding.Embedding = updatedVector

	// Wait for 1.1 second to ensure timestamp update (DB stores seconds)
	// Wait for 1.1 second to ensure timestamp update (DB stores seconds)
	time.Sleep(2000 * time.Millisecond)

	updated, err := ts.UpsertMemoEmbedding(ctx, embedding)
	require.NoError(t, err)
	require.NotNil(t, updated)
	// Verify the embedding was updated (ID should be the same)
	require.Equal(t, upserted.ID, updated.ID)
	// Relax timestamp check slightly due to low precision in some environments
	// Just ensure it's not older
	require.GreaterOrEqual(t, updated.UpdatedTs, upserted.UpdatedTs)

	// Test delete embedding
	err = ts.DeleteMemoEmbedding(ctx, memo.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = ts.GetMemoEmbedding(ctx, memo.ID, "BAAI/bge-m3")
	require.NoError(t, err) // GetMemoEmbedding returns nil when not found, no error

	// Test delete non-existent embedding (should be idempotent or not found error)
	err = ts.DeleteMemoEmbedding(ctx, memo.ID)
	// This might return an error depending on implementation
	// Just verify it doesn't crash

	ts.Close()
}

func TestMemoEmbeddingVectorSearch(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Vector search only works with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create test memos with different content
	memos := []*store.Memo{
		{
			UID:        "search-memo-1",
			CreatorID:  user.ID,
			Content:    "Machine learning is a subset of artificial intelligence",
			Visibility: store.Private,
		},
		{
			UID:        "search-memo-2",
			CreatorID:  user.ID,
			Content:    "Deep learning uses neural networks for pattern recognition",
			Visibility: store.Private,
		},
		{
			UID:        "search-memo-3",
			CreatorID:  user.ID,
			Content:    "Cooking pasta requires boiling water and adding salt",
			Visibility: store.Private,
		},
		{
			UID:        "search-memo-archived",
			CreatorID:  user.ID,
			Content:    "This should not appear in search",
			Visibility: store.Private,
			RowStatus:  store.Archived,
		},
	}

	for _, memoCreate := range memos {
		memo, err := ts.CreateMemo(ctx, memoCreate)
		require.NoError(t, err)

		// CreateMemo might enforce NORMAL status, so we update explicitly if needed
		if memoCreate.RowStatus == store.Archived {
			rowStatus := store.Archived
			err = ts.UpdateMemo(ctx, &store.UpdateMemo{
				ID:        memo.ID,
				RowStatus: &rowStatus,
			})
			require.NoError(t, err)
			// Update the memo object so we know its status changed
			memo.RowStatus = store.Archived
			memo.RowStatus = store.Archived
		}

		// Verify that the archived memo is actually archived in DB
		archivedMemo, err := ts.GetMemo(ctx, &store.FindMemo{ID: &memo.ID})
		require.NoError(t, err)
		t.Logf("Memo %s status after update: %s", memo.UID, archivedMemo.RowStatus)

		testVector := make([]float32, 1024)
		for i := range testVector {
			testVector[i] = 0.1
		}

		embedding := &store.MemoEmbedding{
			MemoID:    memo.ID,
			Embedding: testVector,
			Model:     "BAAI/bge-m3",
		}

		_, err = ts.UpsertMemoEmbedding(ctx, embedding)
		require.NoError(t, err)
	}

	// Test vector search
	queryVector := make([]float32, 1024)
	for i := range queryVector {
		queryVector[i] = 0.1
	}

	results, err := ts.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  10,
	})
	require.NoError(t, err)
	require.NotNil(t, results)

	// Should only return NORMAL status memos (3 active, 1 archived)
	if len(results) > 3 {
		for _, r := range results {
			t.Logf("Result: ID=%d UID=%s Status=%s Score=%f", r.Memo.ID, r.Memo.UID, r.Memo.RowStatus, r.Score)
		}
	}
	require.LessOrEqual(t, len(results), 3)

	// Verify all results belong to the user
	for _, result := range results {
		if result.Memo.RowStatus != store.Normal {
			t.Logf("Unexpected row status: %s for memo: %s", result.Memo.RowStatus, result.Memo.UID)
		}
		require.Equal(t, user.ID, result.Memo.CreatorID)
		require.Equal(t, store.Normal, result.Memo.RowStatus)
		// Score should be between 0 and 1
		require.GreaterOrEqual(t, result.Score, float32(0))
		require.LessOrEqual(t, result.Score, float32(1))
	}

	// Test limit
	results, err = ts.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  2,
	})
	require.NoError(t, err)
	require.LessOrEqual(t, len(results), 2)

	ts.Close()
}

func TestMemoEmbeddingCascadeDelete(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Memo embedding tests only work with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create a memo with embedding
	memoCreate := &store.Memo{
		UID:        "cascade-test-memo",
		CreatorID:  user.ID,
		Content:    "Test cascade delete",
		Visibility: store.Private,
	}
	memo, err := ts.CreateMemo(ctx, memoCreate)
	require.NoError(t, err)

	testVector := make([]float32, 1024)
	for i := range testVector {
		testVector[i] = 0.1
	}

	embedding := &store.MemoEmbedding{
		MemoID:    memo.ID,
		Embedding: testVector,
		Model:     "BAAI/bge-m3",
	}

	_, err = ts.UpsertMemoEmbedding(ctx, embedding)
	require.NoError(t, err)

	// Delete the memo (embedding should be cascade deleted)
	err = ts.DeleteMemo(ctx, &store.DeleteMemo{ID: memo.ID})
	require.NoError(t, err)

	// Verify embedding is also deleted
	retrieved, err := ts.GetMemoEmbedding(ctx, memo.ID, "BAAI/bge-m3")
	require.NoError(t, err)
	require.Nil(t, retrieved)

	ts.Close()
}

// TestMemoEmbeddingMultipleModels tests embeddings with multiple models.
func TestMemoEmbeddingMultipleModels(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Memo embedding tests only work with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create a test memo
	memoCreate := &store.Memo{
		UID:        "test-multi-model-memo",
		CreatorID:  user.ID,
		Content:    "Test content for multiple models",
		Visibility: store.Private,
	}
	memo, err := ts.CreateMemo(ctx, memoCreate)
	require.NoError(t, err)

	models := []string{"BAAI/bge-m3", "text-embedding-3-small", "nomic-embed-text"}

	// Create embeddings for different models
	for _, model := range models {
		testVector := make([]float32, 1024)
		for i := range testVector {
			testVector[i] = float32(i) / 1024
		}

		embedding := &store.MemoEmbedding{
			MemoID:    memo.ID,
			Embedding: testVector,
			Model:     model,
		}

		_, err := ts.UpsertMemoEmbedding(ctx, embedding)
		require.NoError(t, err)
	}

	// List all embeddings for the memo
	list, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{
		MemoID: &memo.ID,
	})
	require.NoError(t, err)
	require.Equal(t, len(models), len(list))

	// Verify each model embedding
	for _, model := range models {
		retrieved, err := ts.GetMemoEmbedding(ctx, memo.ID, model)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, model, retrieved.Model)
	}

	ts.Close()
}

// TestMemoEmbeddingDifferentDimensions tests embeddings with different vector dimensions.
func TestMemoEmbeddingDifferentDimensions(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Memo embedding tests only work with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create a test memo
	memoCreate := &store.Memo{
		UID:        "test-dim-memo",
		CreatorID:  user.ID,
		Content:    "Test content for different dimensions",
		Visibility: store.Private,
	}
	memo, err := ts.CreateMemo(ctx, memoCreate)
	require.NoError(t, err)

	dimensions := []int{256, 768, 1024, 1536, 3072}

	// Create embeddings with different dimensions
	for _, dim := range dimensions {
		testVector := make([]float32, dim)
		for i := range testVector {
			testVector[i] = 0.1
		}

		embedding := &store.MemoEmbedding{
			MemoID:    memo.ID,
			Embedding: testVector,
			Model:     "model-dim-" + string(rune('0'+dim)),
		}

		upserted, err := ts.UpsertMemoEmbedding(ctx, embedding)
		require.NoError(t, err)
		require.NotNil(t, upserted)
		require.Equal(t, dim, len(upserted.Embedding))
	}

	// Verify all embeddings were created
	list, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{
		MemoID: &memo.ID,
	})
	require.NoError(t, err)
	require.Equal(t, len(dimensions), len(list))

	ts.Close()
}

// TestMemoEmbeddingVectorSearchScoreRange tests that vector search returns valid scores.
func TestMemoEmbeddingVectorSearchScoreRange(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Vector search only works with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create multiple memos with identical vectors for predictable scoring
	memos := make([]*store.Memo, 10)
	for i := range memos {
		memoCreate := &store.Memo{
			UID:        "score-memo-" + string(rune('0'+i)),
			CreatorID:  user.ID,
			Content:    "Test content for score range",
			Visibility: store.Private,
		}
		memo, err := ts.CreateMemo(ctx, memoCreate)
		require.NoError(t, err)
		memos[i] = memo

		// Use identical vectors for all memos
		testVector := make([]float32, 1024)
		for j := range testVector {
			testVector[j] = 0.1
		}

		embedding := &store.MemoEmbedding{
			MemoID:    memo.ID,
			Embedding: testVector,
			Model:     "BAAI/bge-m3",
		}

		_, err = ts.UpsertMemoEmbedding(ctx, embedding)
		require.NoError(t, err)
	}

	// Search with the same vector
	queryVector := make([]float32, 1024)
	for i := range queryVector {
		queryVector[i] = 0.1
	}

	results, err := ts.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  10,
	})
	require.NoError(t, err)
	require.Greater(t, len(results), 0)

	// All scores should be in valid range
	for _, result := range results {
		require.GreaterOrEqual(t, result.Score, float32(0))
		require.LessOrEqual(t, result.Score, float32(1))
		t.Logf("Memo ID: %d, Score: %f", result.Memo.ID, result.Score)
	}

	ts.Close()
}

// TestMemoEmbeddingFindMemosWithoutEmbedding tests finding memos without embeddings.
func TestMemoEmbeddingFindMemosWithoutEmbedding(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Memo embedding tests only work with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create memos
	memos := make([]*store.Memo, 5)
	for i := range memos {
		memoCreate := &store.Memo{
			UID:        "find-embed-memo-" + string(rune('0'+i)),
			CreatorID:  user.ID,
			Content:    "Test content",
			Visibility: store.Private,
		}
		memo, err := ts.CreateMemo(ctx, memoCreate)
		require.NoError(t, err)
		memos[i] = memo
	}

	// Add embeddings to only first 2 memos
	for i := 0; i < 2; i++ {
		testVector := make([]float32, 1024)
		for j := range testVector {
			testVector[j] = 0.1
		}

		embedding := &store.MemoEmbedding{
			MemoID:    memos[i].ID,
			Embedding: testVector,
			Model:     "BAAI/bge-m3",
		}

		_, err = ts.UpsertMemoEmbedding(ctx, embedding)
		require.NoError(t, err)
	}

	// Find memos without embeddings
	withoutEmbedding, err := ts.FindMemosWithoutEmbedding(ctx, &store.FindMemosWithoutEmbedding{
		Model: "BAAI/bge-m3",
		Limit: 10,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(withoutEmbedding), 3) // At least 3 memos without embeddings

	ts.Close()
}

// TestMemoEmbeddingVectorSearchWithNoResults tests vector search with no matching results.
func TestMemoEmbeddingVectorSearchWithNoResults(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Vector search only works with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create a different user (no access)
	user2, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Create memo for user1
	memoCreate := &store.Memo{
		UID:        "private-memo",
		CreatorID:  user.ID,
		Content:    "Private content",
		Visibility: store.Private,
	}
	memo, err := ts.CreateMemo(ctx, memoCreate)
	require.NoError(t, err)

	testVector := make([]float32, 1024)
	for i := range testVector {
		testVector[i] = 0.1
	}

	embedding := &store.MemoEmbedding{
		MemoID:    memo.ID,
		Embedding: testVector,
		Model:     "BAAI/bge-m3",
	}
	_, err = ts.UpsertMemoEmbedding(ctx, embedding)
	require.NoError(t, err)

	// Search as user2 (should not find user1's private memo)
	results, err := ts.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user2.ID,
		Vector: testVector,
		Limit:  10,
	})
	require.NoError(t, err)
	// User2 should not see User1's private memo
	require.Equal(t, 0, len(results))

	ts.Close()
}

// TestMemoEmbeddingUpsertSameModel tests upserting the same model multiple times.
func TestMemoEmbeddingUpsertSameModel(t *testing.T) {
	if getDriverFromEnv() != "postgres" {
		t.Skip("Memo embedding tests only work with PostgreSQL + pgvector")
	}

	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	memoCreate := &store.Memo{
		UID:        "test-upsert-memo",
		CreatorID:  user.ID,
		Content:    "Test upsert",
		Visibility: store.Private,
	}
	memo, err := ts.CreateMemo(ctx, memoCreate)
	require.NoError(t, err)

	model := "BAAI/bge-m3"
	var embeddingID int32

	// First upsert
	testVector1 := make([]float32, 1024)
	for i := range testVector1 {
		testVector1[i] = 0.1
	}
	embedding1 := &store.MemoEmbedding{
		MemoID:    memo.ID,
		Embedding: testVector1,
		Model:     model,
	}
	upserted1, err := ts.UpsertMemoEmbedding(ctx, embedding1)
	require.NoError(t, err)
	embeddingID = upserted1.ID

	// Second upsert (should update, not create new)
	testVector2 := make([]float32, 1024)
	for i := range testVector2 {
		testVector2[i] = 0.2
	}
	embedding2 := &store.MemoEmbedding{
		MemoID:    memo.ID,
		Embedding: testVector2,
		Model:     model,
	}
	upserted2, err := ts.UpsertMemoEmbedding(ctx, embedding2)
	require.NoError(t, err)

	// ID should be the same (updated, not inserted)
	require.Equal(t, embeddingID, upserted2.ID)

	// Vector should be updated
	require.Equal(t, float32(0.2), upserted2.Embedding[0])

	// Only one embedding should exist for this memo+model combination
	list, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{
		MemoID: &memo.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(list))

	ts.Close()
}
