package embedding

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/store"
)

type Runner struct {
	store            *store.Store
	embeddingService ai.EmbeddingService
	interval         time.Duration
	batchSize        int
	model            string
}

// NewRunner creates a vector embedding runner.
// Parameters optimized for 2C2G: smaller batch size reduces memory peaks,
// longer interval reduces CPU contention.
func NewRunner(store *store.Store, embeddingService ai.EmbeddingService) *Runner {
	return &Runner{
		store:            store,
		embeddingService: embeddingService,
		interval:         2 * time.Minute,
		batchSize:        8,
		model:            "BAAI/bge-m3",
	}
}

// Run starts the background task.
func (r *Runner) Run(ctx context.Context) {
	// Process once on startup
	r.processNewMemos(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.processNewMemos(ctx)
		case <-ctx.Done():
			slog.Info("embedding runner stopped")
			return
		}
	}
}

// RunOnce processes memos once (for manual trigger).
func (r *Runner) RunOnce(ctx context.Context) {
	r.processNewMemos(ctx)
}

func (r *Runner) processNewMemos(ctx context.Context) {
	// Find memos without embeddings
	memos, err := r.findMemosWithoutEmbedding(ctx)
	if err != nil {
		slog.Error("failed to find memos without embedding", "error", err)
		return
	}

	if len(memos) == 0 {
		return
	}

	slog.Info("processing memos for embedding", "count", len(memos))

	// Process in batches
	for i := 0; i < len(memos); i += r.batchSize {
		end := i + r.batchSize
		if end > len(memos) {
			end = len(memos)
		}
		batch := memos[i:end]

		if err := r.processBatch(ctx, batch); err != nil {
			slog.Error("failed to process batch", "error", err)
			continue
		}
		slog.Info("batch processed", "count", len(batch), "progress", fmt.Sprintf("%d/%d", end, len(memos)))
	}
}

func (r *Runner) findMemosWithoutEmbedding(ctx context.Context) ([]*store.Memo, error) {
	return r.store.FindMemosWithoutEmbedding(ctx, &store.FindMemosWithoutEmbedding{
		Model: r.model,
		Limit: r.batchSize * 20, // Fetch more data, but process in small batches
	})
}

func (r *Runner) processBatch(ctx context.Context, memos []*store.Memo) error {
	// Extract content
	texts := make([]string, len(memos))
	for i, m := range memos {
		texts[i] = m.Content
	}

	// Generate vectors in batch
	vectors, err := r.embeddingService.EmbedBatch(ctx, texts)
	if err != nil {
		return err
	}

	// Store vectors
	for i, m := range memos {
		_, err := r.store.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
			MemoID:    m.ID,
			Embedding: vectors[i],
			Model:     r.model,
		})
		if err != nil {
			slog.Error("failed to upsert embedding", "memoID", m.ID, "error", err)
		}
	}

	return nil
}
