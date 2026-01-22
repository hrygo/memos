// Package ocr provides OCR (Optical Character Recognition) functionality using Tesseract.
// This is used to extract text from images for full-text search.
package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"log/slog"
)

// Supported image MIME types for OCR
var SupportedMimeTypes = []string{
	"image/png",
	"image/jpeg",
	"image/jpg",
	"image/gif",
	"image/bmp",
	"image/webp",
}

// Config holds the OCR configuration
type Config struct {
	// TesseractPath is the path to the tesseract executable
	TesseractPath string
	// DataPath is the path to the tessdata directory (optional)
	DataPath string
	// Languages are the languages to use for OCR (e.g., "chi_sim+eng")
	Languages string
}

// DefaultConfig returns the default OCR configuration
func DefaultConfig() *Config {
	return &Config{
		TesseractPath: "tesseract",
		DataPath:      "",
		Languages:     "chi_sim+eng", // Chinese Simplified + English
	}
}

// Client provides OCR functionality
type Client struct {
	config *Config
}

// NewClient creates a new OCR client
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	return &Client{config: config}
}

// ExtractText extracts text from an image using Tesseract OCR
func (c *Client) ExtractText(ctx context.Context, image []byte, mimeType string) (string, error) {
	if !c.isSupported(mimeType) {
		return "", errors.Errorf("unsupported MIME type: %s", mimeType)
	}

	// Create a temporary file for the image
	tmpFile, err := os.CreateTemp("", "ocr_*.png")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	// Write image data to temp file
	if err := os.WriteFile(tmpPath, image, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write temp file")
	}

	// Create output file path (without extension)
	outPath := strings.TrimSuffix(tmpPath, filepath.Ext(tmpPath))

	// Build tesseract command
	args := []string{tmpPath, outPath}
	if c.config.Languages != "" {
		args = append(args, "-l", c.config.Languages)
	}

	// Add tessdata path if configured
	if c.config.DataPath != "" {
		args = append(args, "--tessdata-dir", c.config.DataPath)
	}

	// Run tesseract with timeout support
	cmd := exec.CommandContext(ctx, c.config.TesseractPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		slog.Warn("tesseract command failed", "error", err, "stderr", stderr.String())
		return "", errors.Wrap(err, "tesseract command failed")
	}

	// Read the output text file
	txtPath := outPath + ".txt"
	defer os.Remove(txtPath)

	text, err := os.ReadFile(txtPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read OCR output")
	}

	// Clean up the text
	result := strings.TrimSpace(string(text))
	if result == "" {
		return "", nil
	}

	return result, nil
}

// ExtractTextWithLayout extracts text with layout information (boxes)
func (c *Client) ExtractTextWithLayout(ctx context.Context, image []byte, mimeType string) (*Result, error) {
	text, err := c.ExtractText(ctx, image, mimeType)
	if err != nil {
		return nil, err
	}

	return &Result{
		Text:     text,
		Confidence: 0, // Tesseract doesn't provide overall confidence without hocr
		Languages: c.config.Languages,
	}, nil
}

// ExtractTextToHOCR extracts text with hOCR format (HTML with position info)
func (c *Client) ExtractTextToHOCR(ctx context.Context, image []byte, mimeType string) (string, error) {
	if !c.isSupported(mimeType) {
		return "", errors.Errorf("unsupported MIME type: %s", mimeType)
	}

	// Create a temporary file for the image
	tmpFile, err := os.CreateTemp("", "ocr_*.png")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	// Write image data to temp file
	if err := os.WriteFile(tmpPath, image, 0644); err != nil {
		return "", errors.Wrap(err, "failed to write temp file")
	}

	// Create output file path (without extension)
	outPath := strings.TrimSuffix(tmpPath, filepath.Ext(tmpPath))

	// Build tesseract command with hocr output
	args := []string{tmpPath, outPath, "-l", c.config.Languages, "hocr"}

	// Run tesseract
	cmd := exec.CommandContext(ctx, c.config.TesseractPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		slog.Warn("tesseract hocr command failed", "error", err, "stderr", stderr.String())
		return "", errors.Wrap(err, "tesseract hocr command failed")
	}

	// Read the hocr output file
	hocrPath := outPath + ".hocr"
	defer os.Remove(hocrPath)

	hocr, err := os.ReadFile(hocrPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read hOCR output")
	}

	return string(hocr), nil
}

// IsAvailable checks if Tesseract is available
func (c *Client) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, c.config.TesseractPath, "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// GetVersion returns the Tesseract version
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, c.config.TesseractPath, "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "failed to get tesseract version")
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetAvailableLanguages returns the list of available languages
func (c *Client) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	// Build tesseract command with --list-langs
	args := []string{"--list-langs"}
	if c.config.DataPath != "" {
		args = append(args, "--tessdata-dir", c.config.DataPath)
	}

	cmd := exec.CommandContext(ctx, c.config.TesseractPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "failed to list tesseract languages")
	}

	// Parse output
	lines := strings.Split(stdout.String(), "\n")
	var langs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Error:") {
			langs = append(langs, line)
		}
	}

	return langs, nil
}

// Result represents the OCR result with metadata
type Result struct {
	Text       string   `json:"text"`
	Confidence float64  `json:"confidence,omitempty"`
	Languages  string   `json:"languages,omitempty"`
	Words      []Word   `json:"words,omitempty"`
	Lines      []Line   `json:"lines,omitempty"`
}

// Word represents a single word with position
type Word struct {
	Text        string  `json:"text"`
	Confidence  float64 `json:"confidence,omitempty"`
	BoundingBox *Box    `json:"bounding_box,omitempty"`
}

// Line represents a line of text
type Line struct {
	Text       string  `json:"text"`
	Words      []Word  `json:"words,omitempty"`
	BoundingBox *Box   `json:"bounding_box,omitempty"`
}

// Box represents a bounding box
type Box struct {
	X      int32 `json:"x"`
	Y      int32 `json:"y"`
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

// IsSupported checks if a MIME type is supported for OCR
func (c *Client) IsSupported(mimeType string) bool {
	return c.isSupported(mimeType)
}

func (c *Client) isSupported(mimeType string) bool {
	for _, supported := range SupportedMimeTypes {
		if strings.EqualFold(mimeType, supported) {
			return true
		}
	}
	return false
}

// ParseHOCR parses hOCR output and returns structured data
func ParseHOCR(hocr string) (*Result, error) {
	// Simplified hOCR parsing
	// In production, use a proper HTML parser
	result := &Result{}

	// Extract text content (simplified)
	// hOCR is HTML with special classes and title attributes containing coordinates
	lines := strings.Split(hocr, "\n")
	var textBuilder strings.Builder
	for _, line := range lines {
		// Skip HTML tags and get text content
		if strings.Contains(line, "ocr_line") || strings.Contains(line, "ocrx_word") {
			// Extract text between > and <
			start := strings.LastIndex(line, ">")
			end := strings.LastIndex(line, "<")
			if start != -1 && end != -1 && start < end {
				text := strings.TrimSpace(line[start+1 : end])
				if text != "" {
					textBuilder.WriteString(text)
					textBuilder.WriteString(" ")
				}
			}
		}
	}

	result.Text = strings.TrimSpace(textBuilder.String())
	return result, nil
}

// GetLanguageName returns the full name of a language code
func GetLanguageName(code string) string {
	names := map[string]string{
		"eng":      "English",
		"chi_sim":  "Chinese Simplified",
		"chi_tra":  "Chinese Traditional",
		"jpn":      "Japanese",
		"kor":      "Korean",
		"fra":      "French",
		"deu":      "German",
		"spa":      "Spanish",
		"rus":      "Russian",
		"ara":      "Arabic",
		"hin":      "Hindi",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

// ConfigFromEnv creates OCR config from environment variables
func ConfigFromEnv() *Config {
	config := DefaultConfig()

	if path := os.Getenv("MEMOS_OCR_TESSERACT_PATH"); path != "" {
		config.TesseractPath = path
	}
	if path := os.Getenv("MEMOS_OCR_TESSDATA_PATH"); path != "" {
		config.DataPath = path
	}
	if langs := os.Getenv("MEMOS_OCR_LANGUAGES"); langs != "" {
		config.Languages = langs
	}

	return config
}

// MarshalJSON implements custom JSON marshaling
func (r *Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		WordCount int `json:"word_count,omitempty"`
		*Alias
	}{
		WordCount: len(strings.Fields(r.Text)),
		Alias:     (*Alias)(r),
	})
}

// Validate validates the OCR result
func (r *Result) Validate() error {
	if r.Text == "" {
		return errors.New("OCR result is empty")
	}
	return nil
}

// Merge merges multiple OCR results
func Merge(results []*Result) *Result {
	if len(results) == 0 {
		return &Result{}
	}
	if len(results) == 1 {
		return results[0]
	}

	merged := &Result{
		Languages: results[0].Languages,
	}

	var textBuilder strings.Builder
	for _, result := range results {
		if result.Text != "" {
			textBuilder.WriteString(result.Text)
			textBuilder.WriteString("\n\n")
		}
	}
	merged.Text = strings.TrimSpace(textBuilder.String())

	return merged
}

// FormatOutput formats the OCR output
func FormatOutput(result *Result, format string) (string, error) {
	switch format {
	case "text", "":
		return result.Text, nil
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}
