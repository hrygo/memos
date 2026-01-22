package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Message represents a chat message.
type Message struct {
	Role    string // system, user, assistant
	Content string
}

// LLMService is the LLM service interface.
type LLMService interface {
	// Chat performs synchronous chat.
	Chat(ctx context.Context, messages []Message) (string, error)

	// ChatStream performs streaming chat.
	ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan error)
}

type llmService struct {
	client      *openai.Client
	model       string
	maxTokens   int
	temperature float32
}

// NewLLMService creates a new LLMService.
func NewLLMService(cfg *LLMConfig) (LLMService, error) {
	var clientConfig openai.ClientConfig

	switch cfg.Provider {
	case "deepseek":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.deepseek.com"
		}
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		clientConfig.BaseURL = baseURL

	case "openai":
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		if cfg.BaseURL != "" {
			clientConfig.BaseURL = cfg.BaseURL
		}

	case "siliconflow":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.siliconflow.cn/v1"
		}
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		clientConfig.BaseURL = baseURL

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &llmService{
		client:      client,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}, nil
}

func (s *llmService) Chat(ctx context.Context, messages []Message) (string, error) {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model:       s.model,
		MaxTokens:   s.maxTokens,
		Temperature: s.temperature,
		Messages:    convertMessages(messages),
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM chat failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

func (s *llmService) ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan error) {
	contentChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		// Add timeout protection
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		req := openai.ChatCompletionRequest{
			Model:       s.model,
			MaxTokens:   s.maxTokens,
			Temperature: s.temperature,
			Messages:    convertMessages(messages),
		}

		slog.Debug("LLM ChatStream starting", "model", s.model, "messages", len(messages))
		stream, err := s.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			slog.Error("LLM ChatStream failed to create", "error", err)
			select {
			case errChan <- fmt.Errorf("create stream failed: %w", err):
			case <-ctx.Done():
			}
			return
		}
		defer stream.Close()

		chunkCount := 0
		for {
			response, err := stream.Recv()
			if err != nil {
				if strings.Contains(err.Error(), "EOF") || err.Error() == "EOF" {
					slog.Debug("LLM ChatStream completed", "chunks", chunkCount)
					return
				}
				slog.Error("LLM ChatStream receive error", "error", err, "chunks_so_far", chunkCount)
				select {
				case errChan <- fmt.Errorf("stream recv failed: %w", err):
				case <-ctx.Done():
				}
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			delta := response.Choices[0].Delta.Content
			if delta != "" {
				chunkCount++
				select {
				case contentChan <- delta:
				case <-ctx.Done():
					slog.Warn("LLM ChatStream context cancelled during send", "chunks", chunkCount)
					return
				}
			}

			// Check if stream is finished
			if response.Choices[0].FinishReason != "" {
				slog.Debug("LLM ChatStream finished", "reason", response.Choices[0].FinishReason, "chunks", chunkCount)
				return
			}
		}
	}()

	return contentChan, errChan
}

func convertMessages(messages []Message) []openai.ChatCompletionMessage {
	llmMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		switch m.Role {
		case "system":
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: m.Content,
			}
		case "user":
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: m.Content,
			}
		case "assistant":
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: m.Content,
			}
		default:
			// Default to user for unknown roles
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: m.Content,
			}
		}
	}
	return llmMessages
}

// Helper for creating system prompts
func SystemPrompt(content string) Message {
	return Message{Role: "system", Content: content}
}

// Helper for creating user messages
func UserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// Helper for creating assistant messages
func AssistantMessage(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// FormatMessages formats messages for prompt templates.
func FormatMessages(systemPrompt string, userContent string, history []Message) []Message {
	messages := []Message{}
	if systemPrompt != "" {
		messages = append(messages, SystemPrompt(systemPrompt))
	}
	messages = append(messages, history...)
	messages = append(messages, UserMessage(userContent))
	return messages
}
