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
		Type:              aichat.EventConversationStart,
		UserID:            user.ID,
		AgentType:         chatReq.AgentType,
		ConversationID:    chatReq.ConversationID,
		IsTempConversation: chatReq.IsTempConversation,
		Timestamp:         time.Now().Unix(),
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
			Type:              aichat.EventSeparator,
			UserID:            user.ID,
			AgentType:         chatReq.AgentType,
			SeparatorContent:  "Context cleared",
			ConversationID:    chatReq.ConversationID,
			IsTempConversation: chatReq.IsTempConversation,
			Timestamp:         time.Now().Unix(),
		})
		return stream.Send(&v1pb.ChatResponse{Done: true})
	}

	// Emit user message event
	eventBus.Publish(ctx, &aichat.ChatEvent{
		Type:              aichat.EventUserMessage,
		UserID:            user.ID,
		AgentType:         chatReq.AgentType,
		UserMessage:       req.Message,
		ConversationID:    chatReq.ConversationID,
		IsTempConversation: chatReq.IsTempConversation,
		Timestamp:         time.Now().Unix(),
	})

	// Create handler and process request
	handler := s.createChatHandler()

	// Wrap stream to collect assistant response
	collectingStream := &eventCollectingStream{
		grpcStreamWrapper: &grpcStreamWrapper{stream: stream},
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
				Type:              aichat.EventAssistantResponse,
				UserID:            s.userID,
				AgentType:         s.agentType,
				AssistantResponse: response,
				ConversationID:    s.conversationID,
				IsTempConversation: s.isTemp,
				Timestamp:         time.Now().Unix(),
			})
		}
	}

	return s.grpcStreamWrapper.Send(resp)
}
