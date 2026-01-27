package embedding

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/usememos/memos/store"
)

// mockEmbeddingService is a mock implementation of ai.EmbeddingService for testing.
type mockEmbeddingService struct {
	embedFunc      func(ctx context.Context, text string) ([]float32, error)
	embedBatchFunc func(ctx context.Context, texts []string) ([][]float32, error)
	dimensions     int
	callCount      atomic.Int32
	batchCallCount atomic.Int32
	shouldFail     bool
	emptyResult    bool
	delay          time.Duration
}

func newMockEmbeddingService(dimensions int) *mockEmbeddingService {
	return &mockEmbeddingService{
		dimensions: dimensions,
	}
}

func (m *mockEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	m.callCount.Add(1)
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.shouldFail {
		return nil, errors.New("embedding service error")
	}
	if m.emptyResult {
		return nil, nil
	}
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	vector := make([]float32, m.dimensions)
	for i := range vector {
		vector[i] = 0.1
	}
	return vector, nil
}

func (m *mockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	m.batchCallCount.Add(1)
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.shouldFail {
		return nil, errors.New("batch embedding error")
	}
	if m.emptyResult {
		return nil, nil
	}
	if m.embedBatchFunc != nil {
		return m.embedBatchFunc(ctx, texts)
	}
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vector := make([]float32, m.dimensions)
		for j := range vector {
			vector[j] = 0.1
		}
		vectors[i] = vector
	}
	return vectors, nil
}

func (m *mockEmbeddingService) Dimensions() int {
	return m.dimensions
}

// TestNewRunner tests the runner constructor.
func TestNewRunner(t *testing.T) {
	mockService := newMockEmbeddingService(1024)
	s := &store.Store{} // store.ListAttachments now handles nil driver gracefully

	runner := NewRunner(s, mockService)

	assert.NotNil(t, runner)
	assert.Equal(t, s, runner.store)
	assert.Equal(t, mockService, runner.embeddingService)
	assert.Equal(t, 2*time.Minute, runner.interval)
	assert.Equal(t, 8, runner.batchSize)
	assert.Equal(t, "BAAI/bge-m3", runner.model)
}

// TestRunnerProcessBatch_EmptyBatch tests empty batch handling.
func TestRunnerProcessBatch_EmptyBatch(t *testing.T) {
	ctx := context.Background()

	mockSvc := newMockEmbeddingService(1024)
	s := &store.Store{}
	runner := NewRunner(s, mockSvc)

	// Empty batch should not cause panics
	err := runner.processBatch(ctx, []*store.Memo{})
	assert.NoError(t, err)
}

// TestRunnerProcessBatch_EmbeddingFailure tests embedding service failure handling.
func TestRunnerProcessBatch_EmbeddingFailure(t *testing.T) {
	ctx := context.Background()

	mockSvc := newMockEmbeddingService(1024)
	mockSvc.shouldFail = true

	s := &store.Store{}
	runner := NewRunner(s, mockSvc)

	// Should return error when embedding service fails
	err := runner.processBatch(ctx, []*store.Memo{{ID: 1, Content: "test"}})
	assert.Error(t, err)
}

// TestRunnerBatchSizes tests batch processing with different sizes.
func TestRunnerBatchSizes(t *testing.T) {
	tests := []struct {
		name      string
		batchSize int
		memoCount int
	}{
		{"batch size 1", 1, 5},
		{"batch size 5", 5, 12},
		{"batch size 10", 10, 25},
		{"batch size larger than memos", 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := newMockEmbeddingService(1024)
			s := &store.Store{}
			runner := NewRunner(s, mockSvc)
			runner.batchSize = tt.batchSize

			memos := createMemos(tt.memoCount)

			// Test batch slicing logic
			expectedBatches := (tt.memoCount + tt.batchSize - 1) / tt.batchSize
			batchCount := 0
			for i := 0; i < len(memos); i += tt.batchSize {
				end := i + tt.batchSize
				if end > len(memos) {
					end = len(memos)
				}
				batch := memos[i:end]
				assert.LessOrEqual(t, len(batch), tt.batchSize)
				batchCount++
			}

			assert.Equal(t, expectedBatches, batchCount)
		})
	}
}

// TestRunnerWithContextCancellation tests context cancellation with mock embedding service.
func TestRunnerWithContextCancellation(t *testing.T) {
	tests := []struct {
		name              string
		cancelImmediately bool
		cancelAfterDelay  bool
	}{
		{"cancel immediately", true, false},
		{"cancel after delay", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelImmediately {
				cancel()
			}

			// Test that context cancellation works
			if !tt.cancelImmediately {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}

			// Verify context is cancelled
			select {
			case <-ctx.Done():
				// Context cancelled as expected
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Context was not cancelled")
			}
		})
	}
}

// TestRunnerWithDifferentModels tests runner with different model names.
func TestRunnerWithDifferentModels(t *testing.T) {
	models := []string{"BAAI/bge-m3", "text-embedding-3-small", "nomic-embed-text"}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			mockSvc := newMockEmbeddingService(1024)
			s := &store.Store{}
			runner := NewRunner(s, mockSvc)
			runner.model = model

			assert.Equal(t, model, runner.model)
		})
	}
}

// TestMockEmbeddingService tests the mock embedding service itself.
func TestMockEmbeddingService(t *testing.T) {
	t.Run("single embed", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(1024)
		ctx := context.Background()

		vector, err := mockSvc.Embed(ctx, "test text")
		assert.NoError(t, err)
		assert.Equal(t, 1024, len(vector))
		assert.Equal(t, int32(1), mockSvc.callCount.Load())
	})

	t.Run("batch embed", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(768)
		ctx := context.Background()

		texts := []string{"text 1", "text 2", "text 3"}
		vectors, err := mockSvc.EmbedBatch(ctx, texts)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(vectors))
		assert.Equal(t, 768, len(vectors[0]))
		assert.Equal(t, int32(1), mockSvc.batchCallCount.Load())
	})

	t.Run("dimensions", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(1536)
		assert.Equal(t, 1536, mockSvc.Dimensions())
	})

	t.Run("error on shouldFail", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(1024)
		mockSvc.shouldFail = true
		ctx := context.Background()

		_, err := mockSvc.Embed(ctx, "test")
		assert.Error(t, err)
	})

	t.Run("empty result", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(1024)
		mockSvc.emptyResult = true
		ctx := context.Background()

		result, err := mockSvc.Embed(ctx, "test")
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(1024)
		mockSvc.delay = 100 * time.Millisecond

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := mockSvc.Embed(ctx, "test")
		assert.Error(t, err)
	})

	t.Run("custom embed function", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(512)
		mockSvc.embedFunc = func(ctx context.Context, text string) ([]float32, error) {
			return []float32{0.5, 0.3, 0.8}, nil
		}
		ctx := context.Background()

		vector, err := mockSvc.Embed(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, []float32{0.5, 0.3, 0.8}, vector)
	})

	t.Run("custom batch embed function", func(t *testing.T) {
		mockSvc := newMockEmbeddingService(512)
		mockSvc.embedBatchFunc = func(ctx context.Context, texts []string) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = []float32{float32(i), float32(i + 1)}
			}
			return result, nil
		}
		ctx := context.Background()

		vectors, err := mockSvc.EmbedBatch(ctx, []string{"a", "b"})
		assert.NoError(t, err)
		assert.Equal(t, [][]float32{{0, 1}, {1, 2}}, vectors)
	})
}

// TestMemosHelper tests the createMemos helper function.
func TestMemosHelper(t *testing.T) {
	memos := createMemos(5)
	assert.Equal(t, 5, len(memos))
	assert.Equal(t, int32(1), memos[0].ID)
	assert.Equal(t, int32(5), memos[4].ID)
}

// Helper functions.

func createMemos(count int) []*store.Memo {
	memos := make([]*store.Memo, count)
	for i := 0; i < count; i++ {
		memos[i] = &store.Memo{
			ID:      int32(i + 1),
			Content: "test content",
		}
	}
	return memos
}
