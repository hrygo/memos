package memory

import (
	"context"
	"sync"
	"time"
)

// ShortTermMemory manages in-memory session messages with a sliding window.
// Thread-safe for concurrent access.
type ShortTermMemory struct {
	mu       sync.RWMutex
	sessions map[string]*sessionData
	maxSize  int // Maximum messages per session

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type sessionData struct {
	messages   []Message
	lastAccess time.Time
}

// NewShortTermMemory creates a new short-term memory store.
// maxSize specifies the maximum number of messages to keep per session (default 10).
func NewShortTermMemory(maxSize int) *ShortTermMemory {
	if maxSize <= 0 {
		maxSize = 10
	}
	ctx, cancel := context.WithCancel(context.Background())
	stm := &ShortTermMemory{
		sessions: make(map[string]*sessionData),
		maxSize:  maxSize,
		ctx:      ctx,
		cancel:   cancel,
	}
	// Start cleanup goroutine for stale sessions
	stm.wg.Add(1)
	go stm.cleanupLoop()
	return stm
}

// Close stops the cleanup goroutine and releases resources.
// Should be called when the ShortTermMemory is no longer needed.
func (s *ShortTermMemory) Close() {
	s.cancel()
	s.wg.Wait()
}

// GetMessages retrieves recent messages from a session.
// This also updates the session's lastAccess time.
func (s *ShortTermMemory) GetMessages(sessionID string, limit int) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists || len(session.messages) == 0 {
		return []Message{}
	}

	// Update lastAccess on read (Issue #1 fix)
	session.lastAccess = time.Now()

	messages := session.messages
	if limit > 0 && limit < len(messages) {
		// Return the most recent messages
		messages = messages[len(messages)-limit:]
	}

	// Return a copy to prevent modification
	result := make([]Message, len(messages))
	copy(result, messages)
	return result
}

// AddMessage adds a message to a session.
func (s *ShortTermMemory) AddMessage(sessionID string, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		session = &sessionData{
			messages: make([]Message, 0, s.maxSize),
		}
		s.sessions[sessionID] = session
	}

	// Set timestamp if not provided
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	session.messages = append(session.messages, msg)
	session.lastAccess = time.Now()

	// Sliding window: keep only the most recent messages
	if len(session.messages) > s.maxSize {
		// Remove oldest messages
		session.messages = session.messages[len(session.messages)-s.maxSize:]
	}
}

// ClearSession removes all messages from a session.
func (s *ShortTermMemory) ClearSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// SessionCount returns the number of active sessions.
func (s *ShortTermMemory) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// cleanupLoop periodically removes stale sessions (inactive for > 1 hour).
// Stops when the context is cancelled.
func (s *ShortTermMemory) cleanupLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for sessionID, session := range s.sessions {
				if now.Sub(session.lastAccess) > time.Hour {
					delete(s.sessions, sessionID)
				}
			}
			s.mu.Unlock()
		}
	}
}
