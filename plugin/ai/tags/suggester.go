// Package tags provides intelligent tag suggestion for memos.
// P2-C001: Three-layer progressive tag suggestion system.
package tags

import (
	"context"
	"time"
)

// TagSuggester provides tag suggestions for memo content.
type TagSuggester interface {
	// Suggest returns tag suggestions based on content, user history, and rules.
	Suggest(ctx context.Context, req *SuggestRequest) (*SuggestResponse, error)
}

// SuggestRequest contains parameters for tag suggestion.
type SuggestRequest struct {
	UserID  int32  // User ID for personalized suggestions
	MemoID  string // Optional: memo ID when editing existing memo
	Content string // Memo content
	Title   string // Memo title (optional)
	MaxTags int    // Maximum tags to return (default: 5)
	UseLLM  bool   // Whether to use LLM layer (default: true)
}

// SuggestResponse contains tag suggestions and metadata.
type SuggestResponse struct {
	Tags    []Suggestion  `json:"tags"`
	Latency time.Duration `json:"latency"`
	Sources []string      `json:"sources"` // ["statistics", "rules", "llm"]
}

// Suggestion represents a single tag suggestion.
type Suggestion struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"` // 0.0 - 1.0
	Source     string  `json:"source"`     // "statistics", "rules", "llm"
	Reason     string  `json:"reason,omitempty"`
}

// TagFrequency represents tag usage frequency.
type TagFrequency struct {
	Name  string
	Count int
}

// TagWithSimilarity represents a tag from similar memo.
type TagWithSimilarity struct {
	Name       string
	Similarity float64
}

// Layer represents a single layer in the suggestion pipeline.
type Layer interface {
	// Name returns the layer name for logging/metrics.
	Name() string
	// Suggest returns suggestions from this layer.
	Suggest(ctx context.Context, req *SuggestRequest) []Suggestion
}
