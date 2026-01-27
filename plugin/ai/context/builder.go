// Package context provides context building for LLM prompts.
// It orchestrates short-term memory, long-term memory, and retrieval results
// into an optimized context window for LLM inference.
package context

import (
	"context"
	"time"
)

// ContextBuilder builds optimized context for LLM inference.
type ContextBuilder interface {
	// Build constructs the full context from various sources.
	Build(ctx context.Context, req *ContextRequest) (*ContextResult, error)

	// GetStats returns context building statistics.
	GetStats() *ContextStats
}

// ContextRequest contains parameters for context building.
type ContextRequest struct {
	UserID           int32
	SessionID        string
	CurrentQuery     string
	AgentType        string           // "memo", "schedule", "amazing"
	RetrievalResults []*RetrievalItem // RAG retrieval results
	MaxTokens        int              // Total token budget (default: 4096)
}

// RetrievalItem represents a single retrieval result.
type RetrievalItem struct {
	ID      string
	Content string
	Score   float32
	Source  string
}

// ContextResult contains the built context.
type ContextResult struct {
	SystemPrompt        string
	ConversationContext string
	RetrievalContext    string
	UserPreferences     string
	TotalTokens         int
	TokenBreakdown      *TokenBreakdown
	BuildTime           time.Duration
}

// TokenBreakdown shows how tokens are distributed.
type TokenBreakdown struct {
	SystemPrompt    int
	ShortTermMemory int
	LongTermMemory  int
	Retrieval       int
	UserPrefs       int
}

// ContextStats tracks context building metrics.
type ContextStats struct {
	TotalBuilds      int64
	AverageTokens    float64
	CacheHits        int64
	AverageBuildTime time.Duration
}

// Message represents a conversation message.
type Message struct {
	Role      string // "user" or "assistant"
	Content   string
	Timestamp time.Time
}

// EpisodicMemory represents a stored episode.
type EpisodicMemory struct {
	ID        int64
	Timestamp time.Time
	Summary   string
	AgentType string
	Outcome   string
}

// UserPreferences represents user preferences.
type UserPreferences struct {
	Timezone           string
	DefaultDuration    int
	PreferredTimes     []string
	CommunicationStyle string
}
