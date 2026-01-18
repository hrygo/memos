package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Config holds the AI provider configuration.
type Config struct {
	BaseURL         string
	APIKey          string
	EmbeddingModel  string
	ChatModel       string
	MaxRetries      int
	Timeout         time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:         "https://api.openai.com/v1",
		APIKey:          "",
		EmbeddingModel:  "text-embedding-3-small",
		ChatModel:       "gpt-4o-mini",
		MaxRetries:      3,
		Timeout:         30 * time.Second,
	}
}

// Provider provides AI capabilities including LLM and Embedding.
type Provider struct {
	llm    *openai.LLM
	config *Config
}

// NewProvider creates a new AI provider.
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Apply defaults for unset values
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "text-embedding-3-small"
	}
	if cfg.ChatModel == "" {
		cfg.ChatModel = "gpt-4o-mini"
	}

	opts := []openai.Option{
		openai.WithToken(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create openai client: %w", err)
	}

	return &Provider{
		llm:    llm,
		config: cfg,
	}, nil
}

// Embedding generates an embedding vector for the given text.
func (p *Provider) Embedding(ctx context.Context, text string) ([]float32, error) {
	var result []float32
	err := p.doWithRetry(ctx, func() error {
		embeddings, err := p.llm.CreateEmbedding(ctx, []string{text})
		if err != nil {
			return err
		}
		if len(embeddings) == 0 || len(embeddings[0]) == 0 {
			return fmt.Errorf("empty embedding response")
		}
		result = embeddings[0]
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return result, nil
}

// Chat performs a chat completion.
func (p *Provider) Chat(ctx context.Context, messages []llms.MessageContent) (string, error) {
	var result string
	err := p.doWithRetry(ctx, func() error {
		// Build content string from messages
		var content string
		for _, msg := range messages {
			for _, part := range msg.Parts {
				if text, ok := part.(llms.TextContent); ok {
					content += text.Text
				}
			}
		}

		response, err := llms.GenerateFromSinglePrompt(ctx, p.llm, content, llms.WithModel(p.config.ChatModel))
		if err != nil {
			return err
		}
		result = response
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to complete chat: %w", err)
	}

	return result, nil
}

// ChatStream performs a streaming chat completion.
// TODO: Implement proper streaming support with langchaingo.
// For now, this returns a non-streaming response.
func (p *Provider) ChatStream(ctx context.Context, messages []llms.MessageContent) (string, error) {
	return p.Chat(ctx, messages)
}

// ListModels lists available models from the provider.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	return []string{
		p.config.EmbeddingModel,
		p.config.ChatModel,
	}, nil
}

// Validate validates the provider configuration by testing API connectivity.
func (p *Provider) Validate(ctx context.Context) error {
	if p.config.APIKey == "" {
		return fmt.Errorf("API key is required, set MEMOS_AI_API_KEY environment variable")
	}

	// Test embedding generation with a simple request
	_, err := p.Embedding(ctx, "test")
	if err != nil {
		return fmt.Errorf("embedding validation failed: %w", err)
	}

	slog.Info("AI provider validated successfully",
		"embedding_model", p.config.EmbeddingModel,
		"chat_model", p.config.ChatModel)

	return nil
}

// doWithRetry executes a function with exponential backoff retry.
func (p *Provider) doWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < p.config.MaxRetries-1 {
				waitTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				slog.Debug("AI request failed, retrying",
					"attempt", attempt+1,
					"wait_time", waitTime,
					"error", err)
				select {
				case <-time.After(waitTime):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return lastErr
}

// NewProviderFromEnv creates a provider from environment variables.
func NewProviderFromEnv() (*Provider, error) {
	return NewProvider(&Config{
		BaseURL:         getEnv("MEMOS_AI_BASE_URL", "https://api.openai.com/v1"),
		APIKey:          getEnv("MEMOS_AI_API_KEY", ""),
		EmbeddingModel:  getEnv("MEMOS_AI_EMBEDDING_MODEL", "text-embedding-3-small"),
		ChatModel:       getEnv("MEMOS_AI_CHAT_MODEL", "gpt-4o-mini"),
		MaxRetries:      3,
		Timeout:         30 * time.Second,
	})
}

// getEnv gets an environment variable with a fallback.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
