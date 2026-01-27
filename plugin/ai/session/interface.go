// Package session provides the session persistence service interface for AI agents.
// This interface is consumed by Team B (Assistant+Schedule).
package session

import "context"

// SessionService defines the session persistence service interface.
// Consumers: Team B (Assistant+Schedule)
type SessionService interface {
	// SaveContext saves the conversation context.
	SaveContext(ctx context.Context, sessionID string, context *ConversationContext) error

	// LoadContext loads the conversation context.
	LoadContext(ctx context.Context, sessionID string) (*ConversationContext, error)

	// ListSessions lists user sessions.
	ListSessions(ctx context.Context, userID int32, limit int) ([]SessionSummary, error)
}

// ConversationContext represents the conversation context.
type ConversationContext struct {
	SessionID string         `json:"session_id"`
	UserID    int32          `json:"user_id"`
	AgentType string         `json:"agent_type"`
	Messages  []Message      `json:"messages"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

// Message represents a conversation message (reused from memory package).
type Message struct {
	Role    string `json:"role"` // "user" | "assistant" | "system"
	Content string `json:"content"`
}

// SessionSummary represents a session summary.
type SessionSummary struct {
	SessionID   string `json:"session_id"`
	AgentType   string `json:"agent_type"`
	LastMessage string `json:"last_message"`
	UpdatedAt   int64  `json:"updated_at"`
}
