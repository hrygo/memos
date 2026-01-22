// Package ocr provides tests for OCR functionality
package ocr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "tesseract", config.TesseractPath)
	assert.Equal(t, "", config.DataPath)
	assert.Equal(t, "chi_sim+eng", config.Languages)
}

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		client := NewClient(nil)
		assert.NotNil(t, client)
		assert.Equal(t, "chi_sim+eng", client.config.Languages)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &Config{
			TesseractPath: "/usr/bin/tesseract",
			Languages:     "eng",
		}
		client := NewClient(config)
		assert.NotNil(t, client)
		assert.Equal(t, "eng", client.config.Languages)
		assert.Equal(t, "/usr/bin/tesseract", client.config.TesseractPath)
	})
}

// TestIsSupported tests MIME type support checking
func TestIsSupported(t *testing.T) {
	client := NewClient(nil)

	supportedTypes := []string{
		"image/png",
		"image/jpeg",
		"IMAGE/JPG", // Case insensitive
		"image/gif",
		"image/bmp",
		"image/webp",
	}

	for _, mimeType := range supportedTypes {
		t.Run(mimeType, func(t *testing.T) {
			assert.True(t, client.IsSupported(mimeType), "MIME type %s should be supported", mimeType)
		})
	}

	unsupportedTypes := []string{
		"application/pdf",
		"text/plain",
		"image/tiff",
		"",
	}

	for _, mimeType := range unsupportedTypes {
		t.Run(mimeType, func(t *testing.T) {
			assert.False(t, client.IsSupported(mimeType), "MIME type %s should not be supported", mimeType)
		})
	}
}

// TestGetLanguageName tests language code to name mapping
func TestGetLanguageName(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"eng", "English"},
		{"chi_sim", "Chinese Simplified"},
		{"chi_tra", "Chinese Traditional"},
		{"jpn", "Japanese"},
		{"kor", "Korean"},
		{"fra", "French"},
		{"deu", "German"},
		{"spa", "Spanish"},
		{"rus", "Russian"},
		{"ara", "Arabic"},
		{"hin", "Hindi"},
		{"unknown", "unknown"}, // Unknown codes return as-is
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := GetLanguageName(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestResultValidate tests OCR result validation
func TestResultValidate(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := &Result{Text: "Some text"}
		err := result.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty result", func(t *testing.T) {
		result := &Result{Text: ""}
		err := result.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

// TestMerge tests merging multiple OCR results
func TestMerge(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := Merge([]*Result{})
		assert.NotNil(t, result)
		assert.Equal(t, "", result.Text)
	})

	t.Run("single result", func(t *testing.T) {
		original := &Result{Text: "Single text"}
		result := Merge([]*Result{original})
		assert.Same(t, original, result)
	})

	t.Run("multiple results", func(t *testing.T) {
		results := []*Result{
			{Text: "First text"},
			{Text: "Second text"},
			{Text: "Third text"},
		}
		result := Merge(results)
		assert.Contains(t, result.Text, "First text")
		assert.Contains(t, result.Text, "Second text")
		assert.Contains(t, result.Text, "Third text")
	})
}

// TestFormatOutput tests output formatting
func TestFormatOutput(t *testing.T) {
	result := &Result{
		Text:      "Sample text",
		Languages: "eng",
	}

	t.Run("text format", func(t *testing.T) {
		output, err := FormatOutput(result, "text")
		assert.NoError(t, err)
		assert.Equal(t, "Sample text", output)
	})

	t.Run("empty format defaults to text", func(t *testing.T) {
		output, err := FormatOutput(result, "")
		assert.NoError(t, err)
		assert.Equal(t, "Sample text", output)
	})

	t.Run("json format", func(t *testing.T) {
		output, err := FormatOutput(result, "json")
		assert.NoError(t, err)
		assert.Contains(t, output, "Sample text")
		assert.Contains(t, output, "eng")
	})

	t.Run("unsupported format", func(t *testing.T) {
		_, err := FormatOutput(result, "xml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
}

// TestParseHOCR tests hOCR parsing
func TestParseHOCR(t *testing.T) {
	hocr := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
<head>
<title>hOCR</title>
</head>
<body>
<div class='ocr_page' title='bbox 0 0 100 100'>
<p class='ocr_par' lang='eng'>
<span class='ocr_line' title='bbox 10 10 90 20'>
<span class='ocrx_word' title='bbox 10 10 30 15'>Hello</span>
<span class='ocrx_word' title='bbox 35 10 55 15'>World</span>
</span>
</p>
</div>
</body>
</html>`

	result, err := ParseHOCR(hocr)
	assert.NoError(t, err)
	assert.Contains(t, result.Text, "Hello")
	assert.Contains(t, result.Text, "World")
}

// TestExtractText_UnsupportedMIMEType tests error handling for unsupported types
func TestExtractText_UnsupportedMIMEType(t *testing.T) {
	client := NewClient(nil)
	ctx := context.Background()

	_, err := client.ExtractText(ctx, []byte("test data"), "application/pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// TestSupportedMimeTypes tests the supported MIME types constant
func TestSupportedMimeTypes(t *testing.T) {
	expected := []string{
		"image/png",
		"image/jpeg",
		"image/jpg",
		"image/gif",
		"image/bmp",
		"image/webp",
	}

	assert.Equal(t, expected, SupportedMimeTypes)
}

// BenchmarkIsSupported benchmarks MIME type checking
func BenchmarkIsSupported(b *testing.B) {
	client := NewClient(nil)
	mimeTypes := []string{
		"image/png",
		"image/jpeg",
		"application/pdf",
		"text/plain",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, mimeType := range mimeTypes {
			client.IsSupported(mimeType)
		}
	}
}
