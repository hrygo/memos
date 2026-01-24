// Package agent provides conversation context management for multi-turn dialogues.
// This module maintains state across conversation turns to enable handling
// of refinements like "change it to 3pm" without re-specifying the full context.
package agent

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/usememos/memos/plugin/ai/schedule"
	"github.com/usememos/memos/store"
)

// ConversationContext maintains state across conversation turns.
type ConversationContext struct {
	mu sync.RWMutex

	// Identification
	SessionID string
	UserID    int32
	Timezone  string

	// Conversation history
	Turns []ConversationTurn

	// Working state - current work in progress
	WorkingState *WorkingState

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConversationTurn represents a single turn in the conversation.
type ConversationTurn struct {
	UserInput   string        // What the user said
	AgentOutput string        // How the agent responded
	ToolCalls   []ToolCallRecord // Tools called during this turn
	Timestamp   time.Time     // When this turn occurred
}

// ToolCallRecord records a tool invocation.
type ToolCallRecord struct {
	Tool      string
	Input     string
	Output    string
	Success   bool
	Duration  time.Duration
	Timestamp time.Time
}

// WorkingState tracks the agent's current understanding and work in progress.
type WorkingState struct {
	// Proposed schedule (what user wants to create)
	ProposedSchedule *ScheduleDraft

	// Conflicts detected (if any) - stores the conflicting schedules
	Conflicts []*store.Schedule

	// Last intent (what the user wanted to do last)
	LastIntent string

	// Last tool used (for context)
	LastToolUsed string

	// Current step in the workflow
	CurrentStep WorkflowStep
}

// ScheduleDraft represents a partially specified schedule.
type ScheduleDraft struct {
	Title       string
	Description string
	Location    string
	StartTime   *time.Time
	EndTime     *time.Time
	AllDay      bool
	Timezone    string
	Recurrence  *schedule.RecurrenceRule
	Confidence  map[string]float32 // Confidence score for each field (0-1)

	// Raw input that generated this draft
	OriginalInput string
}

// WorkflowStep represents the current step in the scheduling workflow.
type WorkflowStep string

const (
	StepIdle            WorkflowStep = "idle"
	StepParsing         WorkflowStep = "parsing"
	StepConflictCheck   WorkflowStep = "conflict_check"
	StepConflictResolve WorkflowStep = "conflict_resolve"
	StepConfirming      WorkflowStep = "confirming"
	StepCompleted       WorkflowStep = "completed"
)

// NewConversationContext creates a new conversation context.
func NewConversationContext(sessionID string, userID int32, timezone string) *ConversationContext {
	return &ConversationContext{
		SessionID:    sessionID,
		UserID:       userID,
		Timezone:     timezone,
		Turns:        make([]ConversationTurn, 0),
		WorkingState: &WorkingState{CurrentStep: StepIdle},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// AddTurn adds a new turn to the conversation history.
func (c *ConversationContext) AddTurn(userInput, agentOutput string, toolCalls []ToolCallRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	turn := ConversationTurn{
		UserInput:   userInput,
		AgentOutput: agentOutput,
		ToolCalls:   toolCalls,
		Timestamp:   time.Now(),
	}

	c.Turns = append(c.Turns, turn)
	c.UpdatedAt = time.Now()

	// Keep only last 10 turns to manage memory
	if len(c.Turns) > 10 {
		c.Turns = c.Turns[len(c.Turns)-10:]
	}
}

// UpdateWorkingState updates the working state with new information.
func (c *ConversationContext) UpdateWorkingState(state *WorkingState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.WorkingState = state
	c.UpdatedAt = time.Now()
}

// GetWorkingState returns a deep copy of the current working state.
func (c *ConversationContext) GetWorkingState() *WorkingState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.WorkingState == nil {
		return nil
	}

	// Return a deep copy to avoid race conditions
	result := &WorkingState{
		LastIntent:   c.WorkingState.LastIntent,
		LastToolUsed: c.WorkingState.LastToolUsed,
		CurrentStep:  c.WorkingState.CurrentStep,
	}

	// Deep copy ProposedSchedule
	if c.WorkingState.ProposedSchedule != nil {
		result.ProposedSchedule = &ScheduleDraft{
			Title:         c.WorkingState.ProposedSchedule.Title,
			Description:   c.WorkingState.ProposedSchedule.Description,
			Location:      c.WorkingState.ProposedSchedule.Location,
			AllDay:        c.WorkingState.ProposedSchedule.AllDay,
			Timezone:      c.WorkingState.ProposedSchedule.Timezone,
			OriginalInput: c.WorkingState.ProposedSchedule.OriginalInput,
		}
		if c.WorkingState.ProposedSchedule.StartTime != nil {
			t := *c.WorkingState.ProposedSchedule.StartTime
			result.ProposedSchedule.StartTime = &t
		}
		if c.WorkingState.ProposedSchedule.EndTime != nil {
			t := *c.WorkingState.ProposedSchedule.EndTime
			result.ProposedSchedule.EndTime = &t
		}
		if c.WorkingState.ProposedSchedule.Recurrence != nil {
			// RecurrenceRule contains simple types, shallow copy is sufficient
			result.ProposedSchedule.Recurrence = c.WorkingState.ProposedSchedule.Recurrence
		}
		if c.WorkingState.ProposedSchedule.Confidence != nil {
			result.ProposedSchedule.Confidence = make(map[string]float32, len(c.WorkingState.ProposedSchedule.Confidence))
			for k, v := range c.WorkingState.ProposedSchedule.Confidence {
				result.ProposedSchedule.Confidence[k] = v
			}
		}
	}

	// Deep copy Conflicts slice
	if len(c.WorkingState.Conflicts) > 0 {
		result.Conflicts = make([]*store.Schedule, len(c.WorkingState.Conflicts))
		copy(result.Conflicts, c.WorkingState.Conflicts)
	}

	return result
}

// GetLastTurn returns a copy of the most recent conversation turn.
func (c *ConversationContext) GetLastTurn() *ConversationTurn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return nil
	}

	// Return a copy, not a pointer to the slice element
	last := c.Turns[len(c.Turns)-1]
	return &last
}

// GetLastNTurns returns the last N conversation turns.
func (c *ConversationContext) GetLastNTurns(n int) []ConversationTurn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return nil
	}

	start := 0
	if len(c.Turns) > n {
		start = len(c.Turns) - n
	}

	result := make([]ConversationTurn, len(c.Turns)-start)
	copy(result, c.Turns[start:])
	return result
}

// ExtractRefinement attempts to extract a refinement from user input
// based on the current working state.
// For example: "change to 3pm" when there's a proposed schedule.
func (c *ConversationContext) ExtractRefinement(userInput string) *ScheduleDraft {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if we have a working state with a proposed schedule
	if c.WorkingState == nil || c.WorkingState.ProposedSchedule == nil {
		return nil
	}

	// Check if the input looks like a refinement
	// Refinement patterns:
	// - Time modifications: "change to 3pm", "move to tomorrow", etc.
	// - Title modifications: "call it meeting", "change title to..."
	// - Location modifications: "at the office", "change location to..."

	refinement := &ScheduleDraft{}
	updated := false

	// Copy existing draft
	existing := c.WorkingState.ProposedSchedule

	// Check for time modification patterns
	lowerInput := lower(userInput)
	if contains(lowerInput, []string{"change to", "move to", "reschedule to", "set for"}) {
		// This looks like a time refinement - let parser handle it
		// Just indicate that this is a refinement
		refinement.OriginalInput = userInput
		updated = true
	}

	// Check for simple time patterns like "3pm", "tomorrow", etc.
	if containsAny(lowerInput, []string{"am", "pm", "today", "tomorrow", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}) {
		refinement.OriginalInput = userInput
		updated = true
	}

	// If we detected a refinement, copy over existing fields
	if updated && existing != nil {
		if refinement.Title == "" && existing.Title != "" {
			refinement.Title = existing.Title
		}
		if refinement.Description == "" && existing.Description != "" {
			refinement.Description = existing.Description
		}
		if refinement.Location == "" && existing.Location != "" {
			refinement.Location = existing.Location
		}
		if refinement.StartTime == nil && existing.StartTime != nil {
			t := *existing.StartTime
			refinement.StartTime = &t
		}
		if refinement.EndTime == nil && existing.EndTime != nil {
			t := *existing.EndTime
			refinement.EndTime = &t
		}
		refinement.Timezone = existing.Timezone
		refinement.AllDay = existing.AllDay
		refinement.Recurrence = existing.Recurrence

		slog.Debug("context: extracted refinement",
			"session_id", c.SessionID,
			"user_input", userInput,
			"existing_title", existing.Title,
			"refinement_title", refinement.Title)

		return refinement
	}

	return nil
}

// Clear resets the conversation context.
func (c *ConversationContext) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Turns = make([]ConversationTurn, 0)
	c.WorkingState = &WorkingState{CurrentStep: StepIdle}
	c.UpdatedAt = time.Now()
}

// GetSummary returns a summary of the conversation context.
func (c *ConversationContext) GetSummary() ContextSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := ContextSummary{
		SessionID:    c.SessionID,
		UserID:       c.UserID,
		TurnCount:    len(c.Turns),
		CurrentStep:  StepIdle,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}

	if c.WorkingState != nil {
		summary.CurrentStep = c.WorkingState.CurrentStep
		summary.LastIntent = c.WorkingState.LastIntent
		summary.HasProposedSchedule = c.WorkingState.ProposedSchedule != nil
		summary.ConflictCount = len(c.WorkingState.Conflicts)
	}

	return summary
}

// ToJSON exports the conversation context to JSON for persistence.
func (c *ConversationContext) ToJSON() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ContextSummary provides a quick overview of the context state.
type ContextSummary struct {
	SessionID         string
	UserID            int32
	TurnCount         int
	CurrentStep       WorkflowStep
	LastIntent        string
	HasProposedSchedule bool
	ConflictCount     int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ContextStore manages conversation contexts for multiple sessions.
type ContextStore struct {
	mu       sync.RWMutex
	contexts map[string]*ConversationContext
}

// NewContextStore creates a new context store.
func NewContextStore() *ContextStore {
	return &ContextStore{
		contexts: make(map[string]*ConversationContext),
	}
}

// GetOrCreate retrieves or creates a conversation context.
func (s *ContextStore) GetOrCreate(sessionID string, userID int32, timezone string) *ConversationContext {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, exists := s.contexts[sessionID]; exists {
		return ctx
	}

	ctx := NewConversationContext(sessionID, userID, timezone)
	s.contexts[sessionID] = ctx
	return ctx
}

// Get retrieves a conversation context if it exists.
func (s *ContextStore) Get(sessionID string) *ConversationContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.contexts[sessionID]
}

// Delete removes a conversation context.
func (s *ContextStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.contexts, sessionID)
}

// CleanupOld removes contexts older than the specified duration.
func (s *ContextStore) CleanupOld(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	deleted := 0

	for sessionID, ctx := range s.contexts {
		if ctx.UpdatedAt.Before(cutoff) {
			delete(s.contexts, sessionID)
			deleted++
		}
	}

	return deleted
}

// Helper functions

// lower converts a string to lowercase using the standard library for proper Unicode support.
func lower(s string) string {
	return strings.ToLower(s)
}

func contains(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func containsAny(s string, substrings []string) bool {
	return contains(s, substrings)
}
