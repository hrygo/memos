package v1

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/store"
)

// parseTags parses tags from LLM response.
func parseTags(response string, limit int) []string {
	lines := splitLines(response)
	var tags []string

	for _, line := range lines {
		tag := trimSpace(line)
		tag = trimPrefix(tag, "-")
		tag = trimPrefix(tag, "#")
		tag = trimSpace(tag)

		if tag != "" && len(tag) <= 20 {
			tags = append(tags, tag)
			if len(tags) >= limit {
				break
			}
		}
	}

	return tags
}

// splitLines splits a string by newlines.
func splitLines(s string) []string {
	var result []string
	var line []rune
	for _, ch := range s {
		if ch == '\n' {
			result = append(result, string(line))
			line = []rune{}
		} else {
			line = append(line, ch)
		}
	}
	if len(line) > 0 {
		result = append(result, string(line))
	}
	return result
}

// trimSpace trims whitespace from both ends of a string.
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && isSpace(byte(s[start])) {
		start++
	}
	for end > start && isSpace(byte(s[end-1])) {
		end--
	}
	return s[start:end]
}

// isSpace checks if a byte is a whitespace character.
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// trimPrefix removes a prefix from a string if present.
func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) {
		for i := 0; i < len(prefix); i++ {
			if s[i] != prefix[i] {
				return s
			}
		}
		return s[len(prefix):]
	}
	return s
}

// mockLLMService is a mock LLM service for testing.
type mockLLMService struct {
	response string
}

func (m *mockLLMService) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	return m.response, nil
}

func (m *mockLLMService) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, <-chan error) {
	return nil, nil
}

func (m *mockLLMService) ChatWithTools(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{Content: m.response}, nil
}

func (m *mockLLMService) IsEnabled() bool {
	return true
}

// mockEmbeddingService is a mock embedding service.
type mockEmbeddingService struct{}

func (m *mockEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 1024), nil
}

func (m *mockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 1024)
	}
	return result, nil
}

func (m *mockEmbeddingService) Dimensions() int {
	return 1024
}

// mockRerankerService is a mock reranker service.
type mockRerankerService struct{}

func (m *mockRerankerService) IsEnabled() bool {
	return false
}

func (m *mockRerankerService) Rerank(ctx context.Context, query string, documents []string, topN int) ([]ai.RerankResult, error) {
	return nil, nil
}

// TestSuggestTags_EmptyContent tests empty content error.
func TestSuggestTags_EmptyContent(t *testing.T) {
	ctx := context.Background()
	st := createStore(t)
	llm := &mockLLMService{response: "tag1\ntag2\ntag3"}
	service := createTestAIService(st, llm)

	req := &v1pb.SuggestTagsRequest{
		Content: "",
	}

	_, err := service.SuggestTags(ctx, req)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// TestSuggestTags_LimitValidation tests limit parameter validation.
func TestSuggestTags_LimitValidation(t *testing.T) {
	ctx := context.Background()
	st := createStore(t)
	llm := &mockLLMService{response: "tag1\ntag2\ntag3"}
	service := createTestAIService(st, llm)

	tests := []struct {
		name        string
		limit       int32
		expectCount int
	}{
		{"default limit (5)", 0, 5},
		{"limit 1", 1, 1},
		{"limit 10", 10, 10},
		{"limit over max (11) should be capped to 10", 11, 10},
		{"limit under min (0) should be set to 5", -1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &v1pb.SuggestTagsRequest{
				Content: "test content",
				Limit:   tt.limit,
			}

			_, err := service.SuggestTags(ctx, req)

			// For now, the method returns empty response
			// We'll verify after implementation
			_ = err
		})
	}
}

// TestSuggestTags_ParseTags tests tag parsing logic.
func TestSuggestTags_ParseTags(t *testing.T) {
	tests := []struct {
		name     string
		response string
		limit    int
		expected []string
	}{
		{
			name:     "simple tags",
			response: "programming\ngo\ntutorial",
			limit:    10,
			expected: []string{"programming", "go", "tutorial"},
		},
		{
			name:     "tags with # prefix",
			response: "#programming\n#coding\n#golang",
			limit:    10,
			expected: []string{"programming", "coding", "golang"},
		},
		{
			name:     "tags with dash prefix",
			response: "- tag1\n- tag2",
			limit:    10,
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "limit works",
			response: "tag1\ntag2\ntag3\ntag4\ntag5",
			limit:    3,
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "long tag is filtered",
			response: "tag1\nverylongtagthatexceeds20charactersshouldbeignored\n",
			limit:    10,
			expected: []string{"tag1"},
		},
		{
			name:     "empty lines are skipped",
			response: "tag1\n\n\ntag2\n\n   \ntag3",
			limit:    10,
			expected: []string{"tag1", "tag2", "tag3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTags(tt.response, tt.limit)
			require.Equal(t, tt.expected, result)
		})
	}
}

// createStore creates a test store.
// TODO: Implement with actual database setup
func createStore(t *testing.T) *store.Store {
	t.Skip("requires database setup")
	return nil
}

// createTestAIService creates an AIService for testing.
func createTestAIService(st *store.Store, llmService ai.LLMService) *AIService {
	return &AIService{
		Store:            st,
		LLMService:       llmService,
		EmbeddingService: &mockEmbeddingService{},
		RerankerService:  &mockRerankerService{},
	}
}
