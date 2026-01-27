package store

import "time"

// EpisodicMemory represents an episodic memory record for AI learning.
type EpisodicMemory struct {
	ID         int64
	UserID     int32
	Timestamp  time.Time
	AgentType  string // memo/schedule/amazing/assistant
	UserInput  string
	Outcome    string // success/failure
	Summary    string
	Importance float32 // 0-1
	CreatedTs  int64
}

// FindEpisodicMemory specifies the conditions for finding episodic memories.
type FindEpisodicMemory struct {
	ID        *int64
	UserID    *int32
	AgentType *string
	Query     *string // For text search in user_input and summary
	Limit     int
	Offset    int
}

// DeleteEpisodicMemory specifies the conditions for deleting episodic memories.
type DeleteEpisodicMemory struct {
	ID     *int64
	UserID *int32
}
