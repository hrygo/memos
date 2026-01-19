package ai

import (
	"context"
	"testing"
)

// TestNewEmbeddingService tests service creation.
func TestNewEmbeddingService(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *EmbeddingConfig
		expectError bool
	}{
		{
			name: "SiliconFlow config",
			cfg: &EmbeddingConfig{
				Provider:   "siliconflow",
				Model:      "BAAI/bge-m3",
				Dimensions: 1024,
				APIKey:     "test-key",
				BaseURL:    "https://api.siliconflow.cn/v1",
			},
			expectError: false,
		},
		{
			name: "OpenAI config",
			cfg: &EmbeddingConfig{
				Provider:   "openai",
				Model:      "text-embedding-3-small",
				Dimensions: 1536,
				APIKey:     "test-key",
				BaseURL:    "https://api.openai.com/v1",
			},
			expectError: false,
		},
		{
			name: "Ollama config",
			cfg: &EmbeddingConfig{
				Provider:   "ollama",
				Model:      "nomic-embed-text",
				Dimensions: 768,
				BaseURL:    "http://localhost:11434",
			},
			expectError: false,
		},
		{
			name: "Unsupported provider",
			cfg: &EmbeddingConfig{
				Provider: "unsupported",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEmbeddingService(tt.cfg)
			if (err != nil) != tt.expectError {
				t.Errorf("NewEmbeddingService() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestEmbeddingService_Dimensions tests Dimensions method.
func TestEmbeddingService_Dimensions(t *testing.T) {
	cfg := &EmbeddingConfig{
		Provider:   "siliconflow",
		Model:      "BAAI/bge-m3",
		Dimensions: 1024,
		APIKey:     "test-key",
		BaseURL:    "https://api.siliconflow.cn/v1",
	}

	service, err := NewEmbeddingService(cfg)
	if err != nil {
		t.Fatalf("NewEmbeddingService() error = %v", err)
	}

	if service.Dimensions() != 1024 {
		t.Errorf("Dimensions() = %d, want 1024", service.Dimensions())
	}
}

// TestEmbeddingService_Embed_Empty tests empty text handling.
func TestEmbeddingService_Embed_Empty(t *testing.T) {
	// Mock service that returns empty result
	service := &embeddingService{
		embedder:   &mockEmbedder{returnEmpty: true},
		dimensions: 1024,
	}

	_, err := service.Embed(context.Background(), "test")
	if err == nil {
		t.Error("Expected error for empty embedding result, got nil")
	}
}

// mockEmbedder is a mock embedder for testing.
type mockEmbedder struct {
	returnEmpty bool
	dimensions  int
}

func newMockEmbedder(dimensions int) *mockEmbedder {
	return &mockEmbedder{dimensions: dimensions}
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.returnEmpty {
		return [][]float32{}, nil
	}
	dim := m.dimensions
	if dim == 0 {
		dim = 1024
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, dim)
		for j := range result[i] {
			result[i][j] = 0.1
		}
	}
	return result, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.returnEmpty {
		return []float32{}, nil
	}
	dim := m.dimensions
	if dim == 0 {
		dim = 1024
	}
	result := make([]float32, dim)
	for i := range result {
		result[i] = 0.1
	}
	return result, nil
}

// TestEmbeddingService_EmptyString tests empty string input.
func TestEmbeddingService_EmptyString(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	vector, err := service.Embed(context.Background(), "")
	if err != nil {
		t.Fatalf("Embed() with empty string should succeed, got error: %v", err)
	}
	if len(vector) != 1024 {
		t.Errorf("Embed() with empty string returned vector of length %d, want 1024", len(vector))
	}
}

// TestEmbeddingService_VeryLongString tests very long string input.
func TestEmbeddingService_VeryLongString(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	// Create a very long string (10000 characters)
	longText := ""
	for i := 0; i < 10000; i++ {
		longText += "a"
	}

	vector, err := service.Embed(context.Background(), longText)
	if err != nil {
		t.Fatalf("Embed() with long string should succeed, got error: %v", err)
	}
	if len(vector) != 1024 {
		t.Errorf("Embed() with long string returned vector of length %d, want 1024", len(vector))
	}
}

// TestEmbeddingService_EmbedBatch_Empty tests empty batch.
func TestEmbeddingService_EmbedBatch_Empty(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	vectors, err := service.EmbedBatch(context.Background(), []string{})
	if err != nil {
		t.Fatalf("EmbedBatch() with empty slice should succeed, got error: %v", err)
	}
	if len(vectors) != 0 {
		t.Errorf("EmbedBatch() with empty slice returned %d vectors, want 0", len(vectors))
	}
}

// TestEmbeddingService_EmbedBatch_Multiple tests batch with multiple texts.
func TestEmbeddingService_EmbedBatch_Multiple(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(768),
		dimensions: 768,
	}

	texts := []string{"first text", "second text", "third text"}
	vectors, err := service.EmbedBatch(context.Background(), texts)
	if err != nil {
		t.Fatalf("EmbedBatch() failed: %v", err)
	}
	if len(vectors) != 3 {
		t.Errorf("EmbedBatch() returned %d vectors, want 3", len(vectors))
	}
	for i, vector := range vectors {
		if len(vector) != 768 {
			t.Errorf("Vector %d has length %d, want 768", i, len(vector))
		}
	}
}

// TestEmbeddingService_EmbedBatch_WithEmptyString tests batch with empty string.
func TestEmbeddingService_EmbedBatch_WithEmptyString(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	texts := []string{"text", "", "another text"}
	vectors, err := service.EmbedBatch(context.Background(), texts)
	if err != nil {
		t.Fatalf("EmbedBatch() with empty string should succeed, got error: %v", err)
	}
	if len(vectors) != 3 {
		t.Errorf("EmbedBatch() returned %d vectors, want 3", len(vectors))
	}
}

// TestEmbeddingService_EmbedBatch_LargeBatch tests large batch processing.
func TestEmbeddingService_EmbedBatch_LargeBatch(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	// Create a large batch (100 texts)
	texts := make([]string, 100)
	for i := range texts {
		texts[i] = "test text"
	}

	vectors, err := service.EmbedBatch(context.Background(), texts)
	if err != nil {
		t.Fatalf("EmbedBatch() with large batch should succeed, got error: %v", err)
	}
	if len(vectors) != 100 {
		t.Errorf("EmbedBatch() returned %d vectors, want 100", len(vectors))
	}
}

// TestEmbeddingService_DifferentDimensions tests different vector dimensions.
func TestEmbeddingService_DifferentDimensions(t *testing.T) {
	dimensions := []int{256, 384, 768, 1024, 1536, 3072}

	for _, dim := range dimensions {
		t.Run("", func(t *testing.T) {
			t.Logf("Testing dimension: %d", dim)
			service := &embeddingService{
				embedder:   newMockEmbedder(dim),
				dimensions: dim,
			}

			if service.Dimensions() != dim {
				t.Errorf("Dimensions() = %d, want %d", service.Dimensions(), dim)
			}

			vector, err := service.Embed(context.Background(), "test")
			if err != nil {
				t.Fatalf("Embed() failed: %v", err)
			}
			if len(vector) != dim {
				t.Errorf("Embed() returned vector of length %d, want %d", len(vector), dim)
			}
		})
	}
}

// TestEmbeddingService_UnicodeText tests unicode text input.
func TestEmbeddingService_UnicodeText(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	testCases := []string{
		"Hello 世界",
		"Привет мир",
		"مرحبا بالعالم",
		"🎉🎊🎈",
		"日本語テスト",
		"ελληνικά",
	}

	for _, text := range testCases {
		t.Run(text, func(t *testing.T) {
			vector, err := service.Embed(context.Background(), text)
			if err != nil {
				t.Fatalf("Embed() with unicode text '%s' failed: %v", text, err)
			}
			if len(vector) != 1024 {
				t.Errorf("Embed() returned vector of length %d, want 1024", len(vector))
			}
		})
	}
}

// TestEmbeddingService_SpecialCharacters tests special characters.
func TestEmbeddingService_SpecialCharacters(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	testCases := []string{
		"text\nwith\nnewlines",
		"text\twith\ttabs",
		"text with \r\n carriage return",
		"text \"with\" quotes",
		"text 'with' single quotes",
		"text <with> html",
		"text &amp; with entities",
		"text/with/slashes",
		"text\\with\\backslashes",
	}

	for _, text := range testCases {
		t.Run(text, func(t *testing.T) {
			vector, err := service.Embed(context.Background(), text)
			if err != nil {
				t.Fatalf("Embed() with special characters failed: %v", err)
			}
			if len(vector) != 1024 {
				t.Errorf("Embed() returned vector of length %d, want 1024", len(vector))
			}
		})
	}
}

// mockErrorEmbedder is a mock embedder that returns errors.
type mockErrorEmbedder struct {
	returnError bool
}

func (m *mockErrorEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.returnError {
		return nil, &mockEmbeddingError{Message: "embedding service unavailable"}
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 1024)
		for j := range result[i] {
			result[i][j] = 0.1
		}
	}
	return result, nil
}

func (m *mockErrorEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.returnError {
		return nil, &mockEmbeddingError{Message: "embedding service unavailable"}
	}
	result := make([]float32, 1024)
	for i := range result {
		result[i] = 0.1
	}
	return result, nil
}

// mockEmbeddingError is a mock error for testing.
type mockEmbeddingError struct {
	Message string
}

func (e *mockEmbeddingError) Error() string {
	return e.Message
}

// TestEmbeddingService_Embed_ErrorHandling tests error handling.
func TestEmbeddingService_Embed_ErrorHandling(t *testing.T) {
	service := &embeddingService{
		embedder:   &mockErrorEmbedder{returnError: true},
		dimensions: 1024,
	}

	_, err := service.Embed(context.Background(), "test")
	if err == nil {
		t.Error("Expected error from Embed(), got nil")
	}
}

// TestEmbeddingService_EmbedBatch_ErrorHandling tests batch error handling.
func TestEmbeddingService_EmbedBatch_ErrorHandling(t *testing.T) {
	service := &embeddingService{
		embedder:   &mockErrorEmbedder{returnError: true},
		dimensions: 1024,
	}

	_, err := service.EmbedBatch(context.Background(), []string{"test1", "test2"})
	if err == nil {
		t.Error("Expected error from EmbedBatch(), got nil")
	}
}

// TestEmbeddingService_NilContext tests with nil context (should panic or handle gracefully).
func TestEmbeddingService_NilContext(t *testing.T) {
	service := &embeddingService{
		embedder:   newMockEmbedder(1024),
		dimensions: 1024,
	}

	// This should handle nil context gracefully or panic
	// The behavior depends on the underlying implementation
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic with nil context
			t.Logf("Recovered from panic with nil context: %v", r)
		}
	}()

	_, err := service.Embed(nil, "test")
	// If it doesn't panic, check if error is returned
	if err != nil {
		t.Logf("Got error with nil context: %v", err)
	}
}
