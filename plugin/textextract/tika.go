// Package textextract provides full-text extraction functionality using Apache Tika.
// This is used to extract text from PDF, Office, and other document formats.
package textextract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"log/slog"
)

// Supported MIME types for text extraction
var SupportedMimeTypes = []string{
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

// Config holds the text extraction configuration
type Config struct {
	// TikaServerURL is the URL of the Tika server (e.g., http://localhost:9998)
	TikaServerURL string
	// TikaJarPath is the path to tika-app.jar (for embedded mode)
	TikaJarPath string
	// JavaPath is the path to the java executable
	JavaPath string
	// Timeout is the HTTP timeout for Tika server requests
	Timeout time.Duration
	// UseEmbedded determines whether to use embedded Tika (java -jar tika-app.jar)
	UseEmbedded bool
}

// DefaultConfig returns the default text extraction configuration
func DefaultConfig() *Config {
	return &Config{
		TikaServerURL: "http://localhost:9998",
		TikaJarPath:   "",
		JavaPath:      "java",
		Timeout:       30 * time.Second,
		UseEmbedded:   false,
	}
}

// ConfigFromEnv creates extraction config from environment variables
func ConfigFromEnv() *Config {
	config := DefaultConfig()

	if url := os.Getenv("MEMOS_TEXTEXTRACT_TIKA_URL"); url != "" {
		config.TikaServerURL = url
	}
	if path := os.Getenv("MEMOS_TEXTEXTRACT_TIKA_JAR"); path != "" {
		config.TikaJarPath = path
	}
	if path := os.Getenv("MEMOS_TEXTEXTRACT_JAVA_PATH"); path != "" {
		config.JavaPath = path
	}
	if timeout := os.Getenv("MEMOS_TEXTEXTRACT_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.Timeout = d
		}
	}
	if useEmbedded := os.Getenv("MEMOS_TEXTEXTRACT_EMBEDDED"); useEmbedded == "true" || useEmbedded == "1" {
		config.UseEmbedded = true
	}

	return config
}

// Client provides text extraction functionality
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new text extraction client
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Result represents the extraction result with metadata
type Result struct {
	Text        string            `json:"text"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ContentType string            `json:"content_type"`
	Author      string            `json:"author,omitempty"`
	Title       string            `json:"title,omitempty"`
	Created     string            `json:"created,omitempty"`
	Modified    string            `json:"modified,omitempty"`
	PageCount   int               `json:"page_count,omitempty"`
	WordCount   int               `json:"word_count,omitempty"`
	CharCount   int               `json:"char_count,omitempty"`
}

// ExtractText extracts text from a document
func (c *Client) ExtractText(ctx context.Context, data []byte, contentType string) (*Result, error) {
	if !c.IsSupported(contentType) {
		return nil, errors.Errorf("unsupported content type: %s", contentType)
	}

	if c.config.UseEmbedded && c.config.TikaJarPath != "" {
		return c.extractEmbedded(ctx, data, contentType)
	}

	return c.extractFromServer(ctx, data, contentType)
}

// extractFromServer extracts text using Tika server
func (c *Client) extractFromServer(ctx context.Context, data []byte, contentType string) (*Result, error) {
	// Try Tika server first
	if c.config.TikaServerURL != "" {
		// Put text extraction request
		req, err := http.NewRequestWithContext(ctx, "PUT",
			c.config.TikaServerURL+"/tika",
			bytes.NewReader(data))
		if err != nil {
			return nil, errors.Wrap(err, "failed to create request")
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", "text/plain")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			slog.Warn("Tika server request failed, trying fallback", "error", err)
			// Fall through to embedded mode if available
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return nil, errors.Errorf("tika server returned status %d: %s", resp.StatusCode, string(body))
			}

			text, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read response")
			}

			result := &Result{
				Text:        string(text),
				ContentType: contentType,
			}
			result.calculateStats()

			// Get metadata
			metadata, err := c.getMetadata(ctx, data, contentType)
			if err == nil {
				result.Metadata = metadata
				result.Author = metadata["Author"]
				result.Title = metadata["title"]
				result.Created = metadata["Creation-Date"]
				result.Modified = metadata["Last-Modified"]
				if pageCount := metadata["xmpTPg:NPages"]; pageCount != "" {
					result.PageCount = parseIntSafely(pageCount)
				}
			}

			return result, nil
		}
	}

	// Fallback to embedded mode
	if c.config.TikaJarPath != "" {
		return c.extractEmbedded(ctx, data, contentType)
	}

	return nil, errors.New("no Tika server or jar available")
}

// extractEmbedded extracts text using embedded Tika (java -jar tika-app.jar)
func (c *Client) extractEmbedded(ctx context.Context, data []byte, contentType string) (*Result, error) {
	// Create temp files
	inputFile, err := os.CreateTemp("", "tika_input_*")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp input file")
	}
	defer func() {
		inputFile.Close()
		os.Remove(inputFile.Name())
	}()

	if _, err := inputFile.Write(data); err != nil {
		return nil, errors.Wrap(err, "failed to write input file")
	}

	outputFile, err := os.CreateTemp("", "tika_output_*")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp output file")
	}
	defer func() {
		outputFile.Close()
		os.Remove(outputFile.Name())
	}()

	// Run tika-app.jar
	args := []string{
		"-jar", c.config.TikaJarPath,
		"-t", // text output
		inputFile.Name(),
	}

	cmd := exec.CommandContext(ctx, c.config.JavaPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		slog.Warn("Tika embedded failed", "error", err, "stderr", stderr.String())
		return nil, errors.Wrap(err, "tika-app.jar failed")
	}

	result := &Result{
		Text:        strings.TrimSpace(stdout.String()),
		ContentType: contentType,
	}
	result.calculateStats()

	return result, nil
}

// getMetadata retrieves document metadata from Tika
func (c *Client) getMetadata(ctx context.Context, data []byte, contentType string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT",
		c.config.TikaServerURL+"/meta",
		bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("metadata request returned status %d", resp.StatusCode)
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		} else if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
			if str, ok := arr[0].(string); ok {
				result[k] = str
			}
		}
	}

	return result, nil
}

// ExtractTextFromFile extracts text from a file
func (c *Client) ExtractTextFromFile(ctx context.Context, filePath string) (*Result, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	contentType := c.detectContentType(filePath, data)
	return c.ExtractText(ctx, data, contentType)
}

// IsAvailable checks if Tika is available
func (c *Client) IsAvailable(ctx context.Context) bool {
	// Try server first
	if c.config.TikaServerURL != "" {
		req, _ := http.NewRequestWithContext(ctx, "GET", c.config.TikaServerURL, nil)
		resp, err := c.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}
	}

	// Try embedded mode
	if c.config.TikaJarPath != "" {
		if _, err := os.Stat(c.config.TikaJarPath); err == nil {
			// Check if java is available
			cmd := exec.CommandContext(ctx, c.config.JavaPath, "-version")
			return cmd.Run() == nil
		}
	}

	return false
}

// IsSupported checks if a MIME type is supported
func (c *Client) IsSupported(contentType string) bool {
	for _, supported := range SupportedMimeTypes {
		if strings.EqualFold(contentType, supported) {
			return true
		}
	}
	return false
}

// GetSupportedMimeTypes returns the list of supported MIME types
func GetSupportedMimeTypes() []string {
	return SupportedMimeTypes
}

// detectContentType detects the content type of a file
func (c *Client) detectContentType(filePath string, data []byte) string {
	// Try by extension first
	ext := strings.ToLower(filepath.Ext(filePath))
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}

	// Try by sniffing
	return http.DetectContentType(data)
}

// calculateStats calculates word and character counts
func (r *Result) calculateStats() {
	r.CharCount = len(r.Text)
	r.WordCount = len(strings.Fields(r.Text))
}

// GetSummary returns a summary of the extracted text
func (r *Result) GetSummary(maxLength int) string {
	if maxLength <= 0 || len(r.Text) <= maxLength {
		return r.Text
	}

	// Try to break at word boundary
	text := r.Text[:maxLength]
	lastSpace := strings.LastIndex(text, " ")
	if lastSpace > maxLength*3/4 {
		text = text[:lastSpace]
	}

	return text + "..."
}

// MarshalJSON implements custom JSON marshaling
func (r *Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		SizeBytes int `json:"size_bytes,omitempty"`
		*Alias
	}{
		SizeBytes: len(r.Text),
		Alias:     (*Alias)(r),
	})
}

// parseIntSafely parses an integer safely
func parseIntSafely(s string) int {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0
	}
	return i
}

// DetectDocumentType detects the type of document from content type
func DetectDocumentType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "application/pdf"):
		return "pdf"
	case strings.Contains(contentType, "word"):
		return "word"
	case strings.Contains(contentType, "excel") || strings.Contains(contentType, "spreadsheet"):
		return "excel"
	case strings.Contains(contentType, "powerpoint") || strings.Contains(contentType, "presentation"):
		return "powerpoint"
	case strings.HasPrefix(contentType, "text/"):
		return "text"
	default:
		return "unknown"
	}
}

// Merge merges multiple extraction results
func Merge(results []*Result) *Result {
	if len(results) == 0 {
		return &Result{}
	}
	if len(results) == 1 {
		return results[0]
	}

	merged := &Result{
		Metadata:    make(map[string]string),
		ContentType: results[0].ContentType,
	}

	var textBuilder strings.Builder
	totalWords := 0
	totalChars := 0

	for _, result := range results {
		if result.Text != "" {
			textBuilder.WriteString(result.Text)
			textBuilder.WriteString("\n\n")
		}
		totalWords += result.WordCount
		totalChars += result.CharCount

		// Merge metadata
		for k, v := range result.Metadata {
			if merged.Metadata[k] == "" {
				merged.Metadata[k] = v
			}
		}
	}

	merged.Text = strings.TrimSpace(textBuilder.String())
	merged.WordCount = totalWords
	merged.CharCount = totalChars

	return merged
}
