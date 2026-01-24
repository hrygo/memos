package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"github.com/usememos/memos/store"
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
	ConversationID    int32
	IsTempConversation bool
	Timestamp         int64
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
// It can return a value that will be collected and returned to the publisher.
type ChatEventListener func(ctx context.Context, event *ChatEvent) (interface{}, error)

// EventBus manages chat event listeners.
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
// Returns a map of listener results indexed by listener index.
// For conversation_start event, it returns the conversation ID.
func (b *EventBus) Publish(ctx context.Context, event *ChatEvent) (map[int]interface{}, error) {
	b.mu.RLock()
	listeners := b.listeners[event.Type]
	b.mu.RUnlock()

	if len(listeners) == 0 {
		return nil, nil
	}

	results := make(map[int]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	for i, listener := range listeners {
		wg.Add(1)
		go func(index int, l ChatEventListener) {
			defer wg.Done()

			// Create timeout context for each listener
			listenerCtx, cancel := context.WithTimeout(ctx, b.timeout)
			defer cancel()

			// Channel to receive result (buffered to avoid blocking)
			resultChan := make(chan listenerResult, 1)

			// Execute listener in separate goroutine
			go func() {
				defer close(resultChan)
				result, err := l(listenerCtx, event)
				resultChan <- listenerResult{result: result, err: err}
			}()

			// Wait for result or timeout
			select {
			case <-listenerCtx.Done():
				if listenerCtx.Err() == context.DeadlineExceeded {
					slog.Default().Warn("Event listener timeout",
						"event_type", event.Type,
						"listener_index", index,
					)
					errOnce.Do(func() { firstErr = fmt.Errorf("listener timeout") })
				}
				// Drain resultChan synchronously to avoid goroutine leak
				// The listener goroutine will exit after sending or closing
				for range resultChan {
				}
			case res, ok := <-resultChan:
				if !ok {
					// Channel closed without result
					return
				}
				if res.err != nil {
					slog.Default().Warn("Event listener failed",
						"event_type", event.Type,
						"listener_index", index,
						"error", res.err,
					)
					errOnce.Do(func() { firstErr = res.err })
				} else if res.result != nil {
					mu.Lock()
					results[index] = res.result
					mu.Unlock()
				}
			}
		}(i, listener)
	}
	wg.Wait()

	return results, firstErr
}

// listenerResult wraps the result from a listener.
type listenerResult struct {
	result interface{}
	err    error
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
	title := s.generateTemporaryTitle(ctx, event.UserID)
	conversation, err := s.store.CreateAIConversation(ctx, &store.AIConversation{
		UID:       shortuuid.New(),
		CreatorID: event.UserID,
		Title:     title,
		ParrotID:  event.AgentType.String(),
		Pinned:    false,
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
func (s *ConversationService) findOrCreateFixedConversation(ctx context.Context, event *ChatEvent) (int32, error) {
	fixedID := calculateFixedConversationID(event.UserID, event.AgentType)

	// Try to find existing
	conversations, err := s.store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &fixedID,
		CreatorID: &event.UserID,
	})
	if err == nil && len(conversations) > 0 {
		// Update timestamp
		s.store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
			ID:        fixedID,
			UpdatedTs: &event.Timestamp,
		})
		return fixedID, nil
	}

	// Create new
	_, err = s.store.CreateAIConversation(ctx, &store.AIConversation{
		ID:        fixedID,
		UID:       shortuuid.New(),
		CreatorID: event.UserID,
		Title:     getFixedConversationTitle(event.AgentType),
		ParrotID:  event.AgentType.String(),
		Pinned:    true,
		CreatedTs: event.Timestamp,
		UpdatedTs: event.Timestamp,
		RowStatus: store.Normal,
	})
	if err != nil {
		return 0, fmt.Errorf("create fixed conversation: %w", err)
	}
	return fixedID, nil
}

// generateTemporaryTitle generates a title for a temporary conversation.
// Returns a title key that the frontend should localize.
func (s *ConversationService) generateTemporaryTitle(ctx context.Context, userID int32) string {
	conversations, _ := s.store.ListAIConversations(ctx, &store.FindAIConversation{
		CreatorID: &userID,
	})
	tempCount := 0
	for _, c := range conversations {
		if !c.Pinned {
			tempCount++
		}
	}
	// Return a title pattern: frontend should localize "chat.new" and substitute {n}
	return fmt.Sprintf("chat.new.%d", tempCount+1)
}

// ConversationStore is the interface needed for conversation persistence.
type ConversationStore interface {
	CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error)
	ListAIConversations(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error)
	UpdateAIConversation(ctx context.Context, update *store.UpdateAIConversation) (*store.AIConversation, error)
	CreateAIMessage(ctx context.Context, create *store.AIMessage) (*store.AIMessage, error)
}

// calculateFixedConversationID calculates the fixed conversation ID for a user and agent type.
// Formula: (UserID << 8) | AgentTypeOffset ensures uniqueness by using bit shifting.
// The lower 8 bits are reserved for agent type offset (max 255 agent types).
func calculateFixedConversationID(userID int32, agentType AgentType) int32 {
	offsets := map[AgentType]int32{
		AgentTypeDefault:  1,
		AgentTypeMemo:     2,
		AgentTypeSchedule: 3,
		AgentTypeAmazing:  4,
		AgentTypeCreative: 5,
	}
	offset := offsets[agentType]
	if offset == 0 {
		offset = 1
	}
	return (userID << 8) | offset
}

// getFixedConversationTitle returns the default title for a fixed conversation.
// Returns a title key that the frontend should localize.
func getFixedConversationTitle(agentType AgentType) string {
	// Title keys for frontend localization
	titles := map[AgentType]string{
		AgentTypeDefault:  "chat.default.title",
		AgentTypeMemo:     "chat.memo.title",
		AgentTypeSchedule: "chat.schedule.title",
		AgentTypeAmazing:  "chat.amazing.title",
		AgentTypeCreative: "chat.creative.title",
	}
	if title, ok := titles[agentType]; ok {
		return title
	}
	return "chat.default.title"
}
