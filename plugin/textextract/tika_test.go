// Package textextract provides tests for text extraction functionality
package textextract

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "http://localhost:9998", config.TikaServerURL)
	assert.Equal(t, "", config.TikaJarPath)
	assert.Equal(t, "java", config.JavaPath)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.False(t, config.UseEmbedded)
}

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		client := NewClient(nil)
		assert.NotNil(t, client)
		assert.Equal(t, "http://localhost:9998", client.config.TikaServerURL)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &Config{
			TikaServerURL: "http://example.com:9998",
			Timeout:       60 * time.Second,
		}
		client := NewClient(config)
		assert.NotNil(t, client)
		assert.Equal(t, "http://example.com:9998", client.config.TikaServerURL)
		assert.Equal(t, 60*time.Second, client.config.Timeout)
	})
}

// TestIsSupported tests MIME type support checking
func TestIsSupported(t *testing.T) {
	client := NewClient(nil)

	supportedTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/rtf",
		"text/plain",
		"text/rtf",
		"APPLICATION/PDF", // Case insensitive
	}

	for _, mimeType := range supportedTypes {
		t.Run(mimeType, func(t *testing.T) {
			assert.True(t, client.IsSupported(mimeType), "MIME type %s should be supported", mimeType)
		})
	}

	unsupportedTypes := []string{
		"image/png",
		"image/jpeg",
		"video/mp4",
		"audio/mp3",
		"",
	}

	for _, mimeType := range unsupportedTypes {
		t.Run(mimeType, func(t *testing.T) {
			assert.False(t, client.IsSupported(mimeType), "MIME type %s should not be supported", mimeType)
		})
	}
}

// TestDetectDocumentType tests document type detection
func TestDetectDocumentType(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"application/pdf", "pdf"},
		{"application/msword", "word"},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "word"},
		{"application/vnd.ms-excel", "excel"},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "excel"},
		{"application/vnd.ms-powerpoint", "powerpoint"},
		{"application/vnd.openxmlformats-officedocument.presentationml.presentation", "powerpoint"},
		{"text/plain", "text"},
		{"unknown/type", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := DetectDocumentType(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestResultCalculateStats tests statistics calculation
func TestResultCalculateStats(t *testing.T) {
	result := &Result{}

	t.Run("empty text", func(t *testing.T) {
		result.Text = ""
		result.calculateStats()
		assert.Equal(t, 0, result.CharCount)
		assert.Equal(t, 0, result.WordCount)
	})

	t.Run("simple text", func(t *testing.T) {
		result.Text = "Hello world"
		result.calculateStats()
		assert.Equal(t, 11, result.CharCount)
		assert.Equal(t, 2, result.WordCount)
	})

	t.Run("text with multiple spaces", func(t *testing.T) {
		result.Text = "Hello    world   test"
		result.calculateStats()
		assert.Equal(t, 21, result.CharCount)
		assert.Equal(t, 3, result.WordCount)
	})
}

// TestResultGetSummary tests summary generation
func TestResultGetSummary(t *testing.T) {
	longText := "This is a very long text that should be truncated when the summary is generated. " +
		"It contains multiple sentences to test the truncation functionality properly."

	result := &Result{Text: longText}

	t.Run("no limit", func(t *testing.T) {
		summary := result.GetSummary(0)
		assert.Equal(t, longText, summary)
	})

	t.Run("limit larger than text", func(t *testing.T) {
		summary := result.GetSummary(1000)
		assert.Equal(t, longText, summary)
	})

	t.Run("limit smaller than text", func(t *testing.T) {
		summary := result.GetSummary(50)
		assert.Less(t, len(summary), len(longText))
		assert.True(t, len(summary) <= 53) // 50 + "..."
		assert.Contains(t, summary, "...")
	})
}

// TestMerge tests merging multiple extraction results
func TestMerge(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := Merge([]*Result{})
		assert.NotNil(t, result)
		assert.Equal(t, "", result.Text)
	})

	t.Run("single result", func(t *testing.T) {
		original := &Result{Text: "Single text", WordCount: 2}
		result := Merge([]*Result{original})
		assert.Same(t, original, result)
	})

	t.Run("multiple results", func(t *testing.T) {
		results := []*Result{
			{Text: "First text", WordCount: 2, CharCount: 10},
			{Text: "Second text", WordCount: 2, CharCount: 11},
			{Text: "Third text", WordCount: 2, CharCount: 10},
		}
		result := Merge(results)
		assert.Contains(t, result.Text, "First text")
		assert.Contains(t, result.Text, "Second text")
		assert.Contains(t, result.Text, "Third text")
		assert.Equal(t, 6, result.WordCount)
		assert.Equal(t, 31, result.CharCount)
	})

	t.Run("metadata merging", func(t *testing.T) {
		results := []*Result{
			{
				Text:        "Text 1",
				Metadata:    map[string]string{"title": "Doc 1", "author": "Alice"},
				ContentType: "application/pdf",
			},
			{
				Text:        "Text 2",
				Metadata:    map[string]string{"title": "Doc 2", "pages": "5"},
				ContentType: "application/pdf",
			},
		}
		result := Merge(results)
		assert.Equal(t, "Doc 1", result.Metadata["title"])
		assert.Equal(t, "Alice", result.Metadata["author"])
		assert.Equal(t, "5", result.Metadata["pages"])
	})
}

// TestGetSupportedMimeTypes tests getting supported MIME types
func TestGetSupportedMimeTypes(t *testing.T) {
	types := GetSupportedMimeTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, "application/pdf")
	assert.Contains(t, types, "text/plain")
}

// TestSupportedMimeTypes tests the supported MIME types constant
func TestSupportedMimeTypes(t *testing.T) {
	expected := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/rtf",
		"text/plain",
		"text/rtf",
	}

	assert.Equal(t, expected, SupportedMimeTypes)
}

// TestConfigFromEnv tests environment variable configuration
func TestConfigFromEnv(t *testing.T) {
	// Save original env vars
	origURL := ""
	origJar := ""
	origJava := ""
	origTimeout := ""
	origEmbedded := ""

	// Set test env vars
	t.Setenv("MEMOS_TEXTEXTRACT_TIKA_URL", "http://test:9999")
	t.Setenv("MEMOS_TEXTEXTRACT_TIKA_JAR", "/path/to/tika.jar")
	t.Setenv("MEMOS_TEXTEXTRACT_JAVA_PATH", "/usr/bin/java")
	t.Setenv("MEMOS_TEXTEXTRACT_TIMEOUT", "60s")
	t.Setenv("MEMOS_TEXTEXTRACT_EMBEDDED", "true")

	config := ConfigFromEnv()

	assert.Equal(t, "http://test:9999", config.TikaServerURL)
	assert.Equal(t, "/path/to/tika.jar", config.TikaJarPath)
	assert.Equal(t, "/usr/bin/java", config.JavaPath)
	assert.Equal(t, 60*time.Second, config.Timeout)
	assert.True(t, config.UseEmbedded)

	// Restore
	_ = origURL
	_ = origJar
	_ = origJava
	_ = origTimeout
	_ = origEmbedded
}

// BenchmarkIsSupported benchmarks MIME type checking
func BenchmarkIsSupported(b *testing.B) {
	client := NewClient(nil)
	mimeTypes := []string{
		"application/pdf",
		"image/png",
		"text/plain",
		"application/msword",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, mimeType := range mimeTypes {
			client.IsSupported(mimeType)
		}
	}
}
