package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/hrygo/divinesense/store"
	"github.com/lithammer/shortuuid/v4"
)

// emptyMetadata is the default empty JSON object for message metadata.
const emptyMetadata = "{}"

// ChatEvent represents a chat event that can be processed by listeners.
type ChatEvent struct {
	Type      ChatEventType
	UserID    int32
	AgentType AgentType
	MessageID string
	// For UserMessage event
	UserMessage string
	// For AssistantResponse event
	AssistantResponse string
	// For Separator event
	SeparatorContent string
	// Context information
	ConversationID     int32
	IsTempConversation bool
	Timestamp          int64
}

// ChatEventType represents the type of chat event.
type ChatEventType string

const (
	// EventConversationStart is fired when a conversation should be created/retrieved
	EventConversationStart ChatEventType = "conversation_start"
	// EventUserMessage is fired when a user sends a message
	EventUserMessage ChatEventType = "user_message"
	// EventAssistantResponse is fired when an assistant responds
	EventAssistantResponse ChatEventType = "assistant_response"
	// EventSeparator is fired when a separator (---) is sent
	EventSeparator ChatEventType = "separator"
)

// ChatEventListener is a function that processes chat events.
//
// IMPORTANT: Listeners MUST respect context cancellation.
// The context passed to listeners has a timeout (default 5s).
// Listeners should check ctx.Done() periodically in long-running operations.
// Failure to respect context will result in the listener continuing to run
// in the background after timeout, which is a resource leak.
//
// Example:
//
//	func myListener(ctx context.Context, event *ChatEvent) (interface{}, error) {
//		// Check context before expensive operation
//		select {
//		case <-ctx.Done():
//			return nil, ctx.Err()
//		default:
//		}
//		// Do work...
//		return result, nil
//	}
type ChatEventListener func(ctx context.Context, event *ChatEvent) (interface{}, error)

// EventBus manages chat event listeners.
//
// Listeners are invoked concurrently with a per-listener timeout.
// Results are collected and returned as a map indexed by listener index.
type EventBus struct {
	listeners map[ChatEventType][]ChatEventListener
	mu        sync.RWMutex
	timeout   time.Duration
}

// NewEventBus creates a new event bus with configurable timeout.
func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[ChatEventType][]ChatEventListener),
		timeout:   5 * time.Second, // Default timeout per listener
	}
}

// SetTimeout sets the timeout for event listeners.
func (b *EventBus) SetTimeout(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.timeout = d
}

// Subscribe registers a listener for a specific event type.
func (b *EventBus) Subscribe(eventType ChatEventType, listener ChatEventListener) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners[eventType] = append(b.listeners[eventType], listener)
}

// Publish emits an event to all registered listeners.
//
// Listeners are executed concurrently, each with its own timeout context.
// Returns a map of listener results indexed by listener index.
// For conversation_start event, it returns the conversation ID.
//
// If any listener returns an error, the first error is returned,
// but all listeners are still executed (fire-and-forget semantics).
func (b *EventBus) Publish(ctx context.Context, event *ChatEvent) (map[int]interface{}, error) {
	// Get listeners for this event type
	b.mu.RLock()
	listeners := make([]ChatEventListener, len(b.listeners[event.Type]))
	copy(listeners, b.listeners[event.Type])
	b.mu.RUnlock()

	if len(listeners) == 0 {
		return nil, nil
	}

	results := make(map[int]interface{})
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	for i, listener := range listeners {
		wg.Add(1)
		go func(index int, l ChatEventListener) {
			defer wg.Done()

			// Panic recovery: prevent one listener's panic from affecting others
			defer func() {
				if r := recover(); r != nil {
					slog.Default().Error("Event listener panic",
						"event_type", event.Type,
						"listener_index", index,
						"panic", r,
					)
					errOnce.Do(func() { firstErr = fmt.Errorf("listener panic: %v", r) })
				}
			}()

			// Create timeout context for this listener
			listenerCtx, cancel := context.WithTimeout(ctx, b.timeout)
			defer cancel()

			// Execute listener directly (no nested goroutine)
			// The listener MUST respect listenerCtx cancellation
			result, err := l(listenerCtx, event)

			// Check if timeout occurred (listener ran too long)
			if listenerCtx.Err() == context.DeadlineExceeded {
				slog.Default().Warn("Event listener timeout",
					"event_type", event.Type,
					"listener_index", index,
					"timeout", b.timeout,
					"had_result", result != nil,
				)
				errOnce.Do(func() { firstErr = fmt.Errorf("listener timeout") })
				// Still store result if available (listener completed just after timeout)
				if result != nil {
					resultsMu.Lock()
					results[index] = result
					resultsMu.Unlock()
				}
				return
			}

			// Check for other context errors (cancellation)
			if listenerCtx.Err() != nil {
				slog.Default().Warn("Event listener context error",
					"event_type", event.Type,
					"listener_index", index,
					"error", listenerCtx.Err(),
				)
				errOnce.Do(func() { firstErr = listenerCtx.Err() })
				return
			}

			// Handle listener errors
			if err != nil {
				slog.Default().Warn("Event listener failed",
					"event_type", event.Type,
					"listener_index", index,
					"error", err,
				)
				errOnce.Do(func() { firstErr = err })
				return
			}

			// Store successful result
			if result != nil {
				resultsMu.Lock()
				results[index] = result
				resultsMu.Unlock()
			}
		}(i, listener)
	}

	wg.Wait()
	return results, firstErr
}

// ConversationService handles conversation persistence independently.
// It listens to chat events and saves conversations/messages to the database.
type ConversationService struct {
	store ConversationStore
}

// NewConversationService creates a new conversation service.
func NewConversationService(store ConversationStore) *ConversationService {
	return &ConversationService{
		store: store,
	}
}

// Subscribe registers event listeners for conversation persistence.
func (s *ConversationService) Subscribe(bus *EventBus) {
	bus.Subscribe(EventConversationStart, s.handleConversationStart)
	bus.Subscribe(EventUserMessage, s.handleUserMessage)
	bus.Subscribe(EventAssistantResponse, s.handleAssistantResponse)
	bus.Subscribe(EventSeparator, s.handleSeparator)
}

// handleConversationStart ensures a conversation exists for the chat.
// Returns the conversation ID.
func (s *ConversationService) handleConversationStart(ctx context.Context, event *ChatEvent) (interface{}, error) {
	if event.ConversationID != 0 {
		// Conversation already specified, just update timestamp
		now := time.Now().Unix()
		_, err := s.store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
			ID:        event.ConversationID,
			UpdatedTs: &now,
		})
		if err != nil {
			slog.Default().Warn("Failed to update conversation timestamp",
				"conversation_id", event.ConversationID,
				"error", err,
			)
		}
		return event.ConversationID, nil
	}

	// Create new conversation (temporary or fixed)
	var id int32
	var err error

	if event.IsTempConversation {
		id, err = s.createTemporaryConversation(ctx, event)
	} else {
		id, err = s.findOrCreateFixedConversation(ctx, event)
	}

	if err != nil {
		slog.Default().Error("Failed to create conversation",
			"user_id", event.UserID,
			"agent_type", event.AgentType,
			"is_temp", event.IsTempConversation,
			"error", err,
		)
		return nil, err
	}

	return id, nil
}

// handleUserMessage saves a user message to the conversation.
func (s *ConversationService) handleUserMessage(ctx context.Context, event *ChatEvent) (interface{}, error) {
	_, err := s.store.CreateAIMessage(ctx, &store.AIMessage{
		UID:            shortuuid.New(),
		ConversationID: event.ConversationID,
		Type:           store.AIMessageTypeMessage,
		Role:           store.AIMessageRoleUser,
		Content:        event.UserMessage,
		Metadata:       emptyMetadata,
		CreatedTs:      event.Timestamp,
	})
	if err != nil {
		slog.Default().Error("Failed to save user message",
			"conversation_id", event.ConversationID,
			"error", err,
		)
	}
	return nil, err
}

// handleAssistantResponse saves an assistant response to the conversation.
func (s *ConversationService) handleAssistantResponse(ctx context.Context, event *ChatEvent) (interface{}, error) {
	_, err := s.store.CreateAIMessage(ctx, &store.AIMessage{
		UID:            shortuuid.New(),
		ConversationID: event.ConversationID,
		Type:           store.AIMessageTypeMessage,
		Role:           store.AIMessageRoleAssistant,
		Content:        event.AssistantResponse,
		Metadata:       emptyMetadata,
		CreatedTs:      event.Timestamp,
	})
	if err != nil {
		slog.Default().Error("Failed to save assistant message",
			"conversation_id", event.ConversationID,
			"error", err,
		)
	}
	return nil, err
}

// handleSeparator saves a separator message to the conversation.
func (s *ConversationService) handleSeparator(ctx context.Context, event *ChatEvent) (interface{}, error) {
	_, err := s.store.CreateAIMessage(ctx, &store.AIMessage{
		UID:            shortuuid.New(),
		ConversationID: event.ConversationID,
		Type:           store.AIMessageTypeSeparator,
		Role:           store.AIMessageRoleSystem,
		Content:        event.SeparatorContent,
		Metadata:       emptyMetadata,
		CreatedTs:      event.Timestamp,
	})
	if err != nil {
		slog.Default().Error("Failed to save separator message",
			"conversation_id", event.ConversationID,
			"error", err,
		)
	}
	return nil, err
}

// createTemporaryConversation creates a new temporary conversation.
func (s *ConversationService) createTemporaryConversation(ctx context.Context, event *ChatEvent) (int32, error) {
	title := s.generateTemporaryTitle()
	conversation, err := s.store.CreateAIConversation(ctx, &store.AIConversation{
		UID:       shortuuid.New(),
		CreatorID: event.UserID,
		Title:     title,
		ParrotID:  event.AgentType.String(),
		CreatedTs: event.Timestamp,
		UpdatedTs: event.Timestamp,
		RowStatus: store.Normal,
	})
	if err != nil {
		return 0, fmt.Errorf("create temporary conversation: %w", err)
	}
	return conversation.ID, nil
}

// findOrCreateFixedConversation finds or creates a fixed conversation.
// Handles race conditions by catching duplicate key errors.
func (s *ConversationService) findOrCreateFixedConversation(ctx context.Context, event *ChatEvent) (int32, error) {
	fixedID := CalculateFixedConversationID(event.UserID, event.AgentType)

	// Try to find existing first (fast path)
	conversations, err := s.store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &fixedID,
		CreatorID: &event.UserID,
	})
	if err == nil && len(conversations) > 0 {
		// Update timestamp
		_, err = s.store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
			ID:        fixedID,
			UpdatedTs: &event.Timestamp,
		})
		if err != nil {
			slog.Default().Warn("Failed to update fixed conversation timestamp",
				"conversation_id", fixedID,
				"error", err,
			)
		}
		return fixedID, nil
	}

	// Try to create new with fixed ID
	_, err = s.store.CreateAIConversation(ctx, &store.AIConversation{
		ID:        fixedID,
		UID:       shortuuid.New(),
		CreatorID: event.UserID,
		Title:     GetFixedConversationTitle(event.AgentType),
		ParrotID:  event.AgentType.String(),
		CreatedTs: event.Timestamp,
		UpdatedTs: event.Timestamp,
		RowStatus: store.Normal,
	})

	// Handle race condition: if another request created it first, fetch it
	if err != nil {
		// Check if it's a duplicate key / unique constraint violation using proper type checking
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// Race condition - another goroutine created it first
			// Fetch the existing conversation
			conversations, err := s.store.ListAIConversations(ctx, &store.FindAIConversation{
				ID:        &fixedID,
				CreatorID: &event.UserID,
			})
			if err == nil && len(conversations) > 0 {
				return fixedID, nil
			}
			return 0, fmt.Errorf("race condition recovery failed: %w", err)
		}
		return 0, fmt.Errorf("create fixed conversation: %w", err)
	}

	return fixedID, nil
}

// generateTemporaryTitle generates a title for a temporary conversation.
// Returns a title key that the frontend should localize and handle numbering.
// The numbering is handled by the frontend to avoid expensive database queries.
func (s *ConversationService) generateTemporaryTitle() string {
	// Return a simple title key; the frontend will handle display numbering
	// based on the actual list of conversations it receives.
	return "chat.new"
}

// ConversationStore is the interface needed for conversation persistence.
type ConversationStore interface {
	CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error)
	ListAIConversations(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error)
	UpdateAIConversation(ctx context.Context, update *store.UpdateAIConversation) (*store.AIConversation, error)
	CreateAIMessage(ctx context.Context, create *store.AIMessage) (*store.AIMessage, error)
}

// CalculateFixedConversationID calculates the fixed conversation ID for a user and agent type.
// Formula: (UserID << 8) | AgentTypeOffset ensures uniqueness by using bit shifting.
// The lower 8 bits are reserved for agent type offset (max 255 agent types).
// Safe for userID up to 8,388,607 (int32 max / 256).
func CalculateFixedConversationID(userID int32, agentType AgentType) int32 {
	// Boundary check: userID << 8 must not overflow int32
	// Max safe userID = 2^31-1 / 256 = 8388607
	const maxSafeUserID = 8388607
	if userID > maxSafeUserID {
		slog.Default().Warn("User ID exceeds safe range for fixed conversation ID",
			"user_id", userID,
			"max_safe", maxSafeUserID,
		)
		// Use modulo to prevent overflow while maintaining some uniqueness
		userID = userID % maxSafeUserID
	}

	offsets := map[AgentType]int32{
		AgentTypeMemo:     2,
		AgentTypeSchedule: 3,
		AgentTypeAmazing:  4,
	}
	offset := offsets[agentType]
	if offset == 0 {
		offset = 4 // Default to AMAZING offset
	}
	return (userID << 8) | offset
}

// GetFixedConversationTitle returns the default title for a fixed conversation.
// Returns a title key that the frontend should localize.
func GetFixedConversationTitle(agentType AgentType) string {
	// Title keys for frontend localization
	titles := map[AgentType]string{
		AgentTypeMemo:     "chat.memo.title",
		AgentTypeSchedule: "chat.schedule.title",
		AgentTypeAmazing:  "chat.amazing.title",
	}
	if title, ok := titles[agentType]; ok {
		return title
	}
	return "chat.amazing.title"
}
