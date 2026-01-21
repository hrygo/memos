package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
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
	model       llms.Model
	maxTokens   int
	temperature float32
}

// NewLLMService creates a new LLMService.
func NewLLMService(cfg *LLMConfig) (LLMService, error) {
	var model llms.Model
	var err error

	switch cfg.Provider {
	case "deepseek":
		// DeepSeek is compatible with OpenAI API
		model, err = openai.New(
			openai.WithToken(cfg.APIKey),
			openai.WithBaseURL(cfg.BaseURL),
			openai.WithModel(cfg.Model),
		)

	case "openai":
		model, err = openai.New(
			openai.WithToken(cfg.APIKey),
			openai.WithModel(cfg.Model),
		)

	case "ollama":
		model, err = ollama.New(
			ollama.WithModel(cfg.Model),
			ollama.WithServerURL(cfg.BaseURL),
		)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, err
	}

	return &llmService{
		model:       model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}, nil
}

func (s *llmService) Chat(ctx context.Context, messages []Message) (string, error) {
	llmMessages := convertMessages(messages)

	resp, err := s.model.GenerateContent(ctx, llmMessages,
		llms.WithMaxTokens(s.maxTokens),
		llms.WithTemperature(float64(s.temperature)),
	)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return resp.Choices[0].Content, nil
}

func (s *llmService) ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan error) {
	// Add buffer to prevent blocking if receiver exits early
	contentChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		llmMessages := convertMessages(messages)

		// Add timeout protection to prevent goroutine leak
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		_, err := s.model.GenerateContent(ctx, llmMessages,
			llms.WithMaxTokens(s.maxTokens),
			llms.WithTemperature(float64(s.temperature)),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				select {
				case contentChan <- string(chunk):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}),
		)

		if err != nil {
			// Check if context is still valid before sending error
			select {
			case errChan <- err:
			case <-ctx.Done():
				// Context cancelled, unable to send error
			}
		}
	}()

	return contentChan, errChan
}

func convertMessages(messages []Message) []llms.MessageContent {
	llmMessages := make([]llms.MessageContent, len(messages))
	for i, m := range messages {
		role := llms.ChatMessageTypeHuman
		switch m.Role {
		case "system":
			role = llms.ChatMessageTypeSystem
		case "user":
			role = llms.ChatMessageTypeHuman
		case "assistant":
			role = llms.ChatMessageTypeAI
		}

		llmMessages[i] = llms.MessageContent{
			Role:  role,
			Parts: []llms.ContentPart{llms.TextPart(m.Content)},
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
