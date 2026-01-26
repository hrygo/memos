package v1

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	aichat "github.com/usememos/memos/server/router/api/v1/ai"
)

// getChatEventBus returns the chat event bus, initializing it on first use.
func (s *AIService) getChatEventBus() *aichat.EventBus {
	s.chatEventBusMu.Lock()
	defer s.chatEventBusMu.Unlock()

	if s.chatEventBus == nil {
		s.chatEventBus = aichat.NewEventBus()
		s.conversationService = aichat.NewConversationService(s.Store)
		s.conversationService.Subscribe(s.chatEventBus)
	}

	return s.chatEventBus
}

// getContextBuilder returns the context builder, initializing it on first use.
func (s *AIService) getContextBuilder() *aichat.ContextBuilder {
	s.contextBuilderMu.Lock()
	defer s.contextBuilderMu.Unlock()

	if s.contextBuilder == nil {
		s.contextBuilder = aichat.NewContextBuilder(s.Store)
	}
	return s.contextBuilder
}

// getConversationSummarizer returns the conversation summarizer, initializing on first use.
func (s *AIService) getConversationSummarizer() *aichat.ConversationSummarizer {
	s.conversationSummarizerMu.Lock()
	defer s.conversationSummarizerMu.Unlock()

	if s.conversationSummarizer == nil {
		s.conversationSummarizer = aichat.NewConversationSummarizerWithStore(
			s.Store,
			s.LLMService,
			11, // Default threshold
		)
	}
	return s.conversationSummarizer
}

// Chat streams a chat response with AI agents.
// Emits events for conversation persistence (handled by ConversationService).
func (s *AIService) Chat(req *v1pb.ChatRequest, stream v1pb.AIService_ChatServer) error {
	ctx := stream.Context()

	if !s.IsEnabled() {
		return status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	if !s.IsLLMEnabled() {
		return status.Errorf(codes.Unavailable, "LLM service is not available")
	}

	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	userKey := strconv.FormatInt(int64(user.ID), 10)
	if !globalAILimiter.Allow(userKey) {
		return status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}

	chatReq := aichat.ToChatRequest(req)
	chatReq.UserID = user.ID

	if chatReq.Timezone == "" || !aichat.IsValidTimezone(chatReq.Timezone) {
		chatReq.Timezone = aichat.GetDefaultTimezone()
	}

	// Get event bus (initializes on first use)
	eventBus := s.getChatEventBus()

	// Emit conversation start event to trigger conversation creation
	event := &aichat.ChatEvent{
		Type:               aichat.EventConversationStart,
		UserID:             user.ID,
		AgentType:          chatReq.AgentType,
		ConversationID:     chatReq.ConversationID,
		IsTempConversation: chatReq.IsTempConversation,
		Timestamp:          time.Now().Unix(),
	}
	results, err := eventBus.Publish(ctx, event)
	if err != nil {
		slog.Default().Warn("Conversation persistence issue during start",
			"user_id", user.ID,
			"error", err,
		)
	}

	// Get conversation ID from listener result
	if len(results) > 0 {
		if convID, ok := results[0].(int32); ok && convID != 0 {
			chatReq.ConversationID = convID
		}
	}

	// Handle separator (---) - emit event and return without agent processing
	if req.Message == "---" && chatReq.ConversationID != 0 {
		eventBus.Publish(ctx, &aichat.ChatEvent{
			Type:               aichat.EventSeparator,
			UserID:             user.ID,
			AgentType:          chatReq.AgentType,
			SeparatorContent:   "Context cleared",
			ConversationID:     chatReq.ConversationID,
			IsTempConversation: chatReq.IsTempConversation,
			Timestamp:          time.Now().Unix(),
		})
		return stream.Send(&v1pb.ChatResponse{Done: true})
	}

	// Emit user message event
	eventBus.Publish(ctx, &aichat.ChatEvent{
		Type:               aichat.EventUserMessage,
		UserID:             user.ID,
		AgentType:          chatReq.AgentType,
		UserMessage:        req.Message,
		ConversationID:     chatReq.ConversationID,
		IsTempConversation: chatReq.IsTempConversation,
		Timestamp:          time.Now().Unix(),
	})

	// Build conversation context from backend
	// This ensures SEPARATOR filtering is enforced server-side
	var history []string
	if chatReq.ConversationID != 0 {
		builder := s.getContextBuilder()
		builtContext, err := builder.BuildContext(ctx, chatReq.ConversationID, &aichat.ContextControl{
			// Pending messages: the current user message (not yet persisted)
			PendingMessages: []aichat.Message{
				{
					Content: req.Message,
					Role:    "user",
					Type:    "MESSAGE",
				},
			},
		})
		if err != nil {
			slog.Default().Warn("Failed to build context from backend",
				"conversation_id", chatReq.ConversationID,
				"error", err,
			)
		} else {
			// Exclude the current message from history (it's the last pending message)
			if len(builtContext.Messages) > 0 {
				history = builtContext.Messages[:len(builtContext.Messages)-1]
			}
			slog.Default().Debug("Built context from backend",
				"conversation_id", chatReq.ConversationID,
				"message_count", len(history),
				"token_count", builtContext.TokenCount,
				"separator_pos", builtContext.SeparatorPos,
				"has_pending", builtContext.HasPending,
			)
		}
	}

	// Fallback to frontend-provided history if backend build failed
	// This maintains backward compatibility during migration
	if len(history) == 0 && len(req.History) > 0 {
		slog.Default().Debug("Using frontend-provided history",
			"conversation_id", chatReq.ConversationID,
			"history_count", len(req.History),
		)
		history = req.History
	}

	chatReq.History = history

	// Create handler and process request
	handler := s.createChatHandler()

	// Wrap stream to collect assistant response
	collectingStream := &eventCollectingStream{
		grpcStreamWrapper: &grpcStreamWrapper{stream: stream},
		service:           s,
		eventBus:          eventBus,
		userID:            user.ID,
		agentType:         chatReq.AgentType,
		conversationID:    chatReq.ConversationID,
		isTemp:            chatReq.IsTempConversation,
	}

	if err := handler.Handle(ctx, chatReq, collectingStream); err != nil {
		return aichat.HandleError(err)
	}

	return nil
}

// createChatHandler creates the chat handler.
func (s *AIService) createChatHandler() aichat.Handler {
	factory := aichat.NewAgentFactory(
		s.LLMService,
		s.AdaptiveRetriever,
		s.Store,
	)
	parrotHandler := aichat.NewParrotHandler(factory, s.LLMService)

	// Configure chat router for auto-routing if intent classifier is enabled
	if s.IntentClassifierConfig != nil && s.IntentClassifierConfig.Enabled {
		chatRouter := aichat.NewChatRouter(s.IntentClassifierConfig)
		parrotHandler.SetChatRouter(chatRouter)
		slog.Info("Chat router enabled",
			"model", s.IntentClassifierConfig.Model,
		)
	}

	return aichat.NewRoutingHandler(parrotHandler)
}

// grpcStreamWrapper wraps the gRPC stream to implement aichat.ChatStream.
type grpcStreamWrapper struct {
	stream v1pb.AIService_ChatServer
}

func (w *grpcStreamWrapper) Send(resp *v1pb.ChatResponse) error {
	return w.stream.Send(resp)
}

func (w *grpcStreamWrapper) Context() context.Context {
	return w.stream.Context()
}

// eventCollectingStream wraps the stream and emits assistant response events.
type eventCollectingStream struct {
	*grpcStreamWrapper
	service        *AIService // Service reference for accessing summarizer
	eventBus       *aichat.EventBus
	userID         int32
	agentType      aichat.AgentType
	conversationID int32
	isTemp         bool
	mu             sync.Mutex
	builder        strings.Builder
}

func (s *eventCollectingStream) Send(resp *v1pb.ChatResponse) error {
	// Collect content from "answer" or "content" events
	if resp.EventType == "answer" || resp.EventType == "content" {
		s.mu.Lock()
		s.builder.WriteString(resp.EventData)
		s.mu.Unlock()
	}

	// When stream is done, emit assistant response event
	if resp.Done {
		s.mu.Lock()
		response := s.builder.String()
		s.mu.Unlock()

		if response != "" {
			s.eventBus.Publish(s.Context(), &aichat.ChatEvent{
				Type:               aichat.EventAssistantResponse,
				UserID:             s.userID,
				AgentType:          s.agentType,
				AssistantResponse:  response,
				ConversationID:     s.conversationID,
				IsTempConversation: s.isTemp,
				Timestamp:          time.Now().Unix(),
			})
		}

		// Check if summarization is needed (async, don't block response)
		// Only summarize for non-temporary conversations
		if !s.isTemp && s.conversationID != 0 {
			go func() {
				summarizer := s.service.getConversationSummarizer()
				if shouldSummarize, count := summarizer.ShouldSummarize(context.Background(), s.conversationID); shouldSummarize {
					slog.Default().Info("Conversation threshold reached, triggering summarization",
						"conversation_id", s.conversationID,
						"message_count", count,
					)
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if err := summarizer.Summarize(ctx, s.conversationID); err != nil {
						slog.Default().Warn("Failed to summarize conversation",
							"conversation_id", s.conversationID,
							"error", err,
						)
					}
				}
			}()
		}
	}

	return s.grpcStreamWrapper.Send(resp)
}
