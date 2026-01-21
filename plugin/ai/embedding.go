package ai

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// EmbeddingService is the vector embedding service interface.
type EmbeddingService interface {
	// Embed generates vector for a single text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates vectors for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the vector dimension.
	Dimensions() int
}

type embeddingService struct {
	embedder   embeddings.Embedder
	dimensions int
}

// NewEmbeddingService creates a new EmbeddingService.
func NewEmbeddingService(cfg *EmbeddingConfig) (EmbeddingService, error) {
	var embedder embeddings.Embedder
	var err error

	switch cfg.Provider {
	case "siliconflow":
		// SiliconFlow is compatible with OpenAI API
		llm, createErr := openai.New(
			openai.WithToken(cfg.APIKey),
			openai.WithBaseURL(cfg.BaseURL),
			openai.WithEmbeddingModel(cfg.Model),
		)
		if createErr != nil {
			return nil, createErr
		}
		embedder, err = embeddings.NewEmbedder(llm)
		if err != nil {
			return nil, err
		}

	case "openai":
		llm, createErr := openai.New(
			openai.WithToken(cfg.APIKey),
			openai.WithBaseURL(cfg.BaseURL),
			openai.WithEmbeddingModel(cfg.Model),
		)
		if createErr != nil {
			return nil, createErr
		}
		embedder, err = embeddings.NewEmbedder(llm)
		if err != nil {
			return nil, err
		}

	case "ollama":
		llm, createErr := ollama.New(
			ollama.WithModel(cfg.Model),
			ollama.WithServerURL(cfg.BaseURL),
		)
		if createErr != nil {
			return nil, createErr
		}
		embedder, err = embeddings.NewEmbedder(llm)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}

	return &embeddingService{
		embedder:   embedder,
		dimensions: cfg.Dimensions,
	}, nil
}

func (s *embeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := s.embedder.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, errors.New("empty embedding result")
	}
	return vectors[0], nil
}

func (s *embeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	vectors, err := s.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, err
	}
	return vectors, nil
}

func (s *embeddingService) Dimensions() int {
	return s.dimensions
}
