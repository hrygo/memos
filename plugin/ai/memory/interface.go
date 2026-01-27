// Package memory provides the unified memory service interface for AI agents.
// This interface is consumed by Team B (Assistant+Schedule) and Team C (Memo Enhancement).
package memory

import (
	"context"
	"time"
)

// MemoryService defines the unified memory service interface.
// Consumers: Team B (Assistant+Schedule), Team C (Memo Enhancement)
type MemoryService interface {
	// ========== Short-term Memory (within session) ==========

	// GetRecentMessages retrieves recent messages from a session.
	// limit: maximum number of messages to return, recommended 10
	GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)

	// AddMessage adds a message to a session.
	AddMessage(ctx context.Context, sessionID string, msg Message) error

	// ========== Long-term Memory (cross-session) ==========

	// SaveEpisode saves an episodic memory.
	SaveEpisode(ctx context.Context, episode EpisodicMemory) error

	// SearchEpisodes searches episodic memories for a specific user.
	// userID: required, ensures multi-tenant data isolation
	// query: search keywords, empty string returns most recent records
	// limit: maximum number of results to return
	SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]EpisodicMemory, error)

	// ========== User Preferences ==========

	// GetPreferences retrieves user preferences.
	GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error)

	// UpdatePreferences updates user preferences.
	UpdatePreferences(ctx context.Context, userID int32, prefs *UserPreferences) error
}

// Message represents a conversation message.
type Message struct {
	Role      string    `json:"role"`      // "user" | "assistant" | "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// EpisodicMemory represents an episodic memory record.
type EpisodicMemory struct {
	ID         int64     `json:"id"`
	UserID     int32     `json:"user_id"`
	Timestamp  time.Time `json:"timestamp"`
	AgentType  string    `json:"agent_type"` // memo/schedule/amazing/assistant
	UserInput  string    `json:"user_input"`
	Outcome    string    `json:"outcome"`    // success/failure
	Summary    string    `json:"summary"`
	Importance float32   `json:"importance"` // 0-1
}

// UserPreferences represents user preferences.
type UserPreferences struct {
	Timezone           string         `json:"timezone"`
	DefaultDuration    int            `json:"default_duration"`    // minutes
	PreferredTimes     []string       `json:"preferred_times"`     // ["09:00", "14:00"]
	FrequentLocations  []string       `json:"frequent_locations"`
	CommunicationStyle string         `json:"communication_style"` // concise/detailed
	TagPreferences     []string       `json:"tag_preferences"`
	CustomSettings     map[string]any `json:"custom_settings"`
}
