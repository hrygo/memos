package v1

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lithammer/shortuuid/v4"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/router/api/v1/ai"
	"github.com/usememos/memos/store"
)

// Chat streams a chat response with AI agents.
// This is the main entry point for AI chat requests.
// Routes to appropriate handler based on agent_type:
// - DEFAULT: Direct LLM chat (no RAG)
// - MEMO: Chat with memo context (RAG)
// - SCHEDULE: Schedule management agent
// - AMAZING: Comprehensive assistant
// - CREATIVE: Creative assistant
func (s *AIService) Chat(req *v1pb.ChatWithMemosRequest, stream v1pb.AIService_ChatServer) error {
	ctx := stream.Context()

	if !s.IsEnabled() {
		return status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Check if LLM service is available (required for all chat)
	if !s.IsLLMEnabled() {
		return status.Errorf(codes.Unavailable, "LLM service is not available")
	}

	// Get authenticated user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Rate limiting check
	userKey := strconv.FormatInt(int64(user.ID), 10)
	if !globalAILimiter.Allow(userKey) {
		return status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}

	// Convert request to internal format
	chatReq := ai.ToChatRequest(req)
	chatReq.UserID = user.ID

	// Normalize timezone: use provided timezone or default
	if chatReq.Timezone == "" || !ai.IsValidTimezone(chatReq.Timezone) {
		chatReq.Timezone = ai.GetDefaultTimezone()
	}

	// Create handler and process request
	handler := s.createChatHandler()
	wrappedStream := &grpcStreamWrapper{stream: stream}

	// Persist cutting line if message is a command to clear context
	// Note: Modern frontend sends '---' or similar to clear context
	if req.Message == "---" && chatReq.ConversationID != 0 {
		_, _ = s.Store.CreateAIMessage(ctx, &store.AIMessage{
			UID:            shortuuid.New(),
			ConversationID: chatReq.ConversationID,
			Type:           store.AIMessageTypeSeparator,
			Role:           store.AIMessageRoleSystem,
			Content:        "Context cleared",
			Metadata:       "{}",
			CreatedTs:      time.Now().Unix(),
		})
	}

	if err := handler.Handle(ctx, chatReq, wrappedStream); err != nil {
		return ai.HandleError(err)
	}

	return nil
}

// createChatHandler creates the appropriate chat handler based on configuration.
func (s *AIService) createChatHandler() ai.Handler {
	// Create agent factory
	factory := ai.NewAgentFactory(
		s.LLMService,
		s.AdaptiveRetriever,
		s.Store,
	)

	// Create individual handlers
	directHandler := ai.NewDirectLLMHandler(s.LLMService)
	parrotHandler := ai.NewParrotHandler(factory, s.LLMService)

	// Create routing handler
	return ai.NewRoutingHandler(directHandler, parrotHandler)
}

// grpcStreamWrapper wraps the gRPC stream to implement ai.ChatStream.
type grpcStreamWrapper struct {
	stream v1pb.AIService_ChatServer
}

// Send sends a response through the gRPC stream.
func (w *grpcStreamWrapper) Send(resp *v1pb.ChatWithMemosResponse) error {
	return w.stream.Send(resp)
}

// Context returns the stream's context.
func (w *grpcStreamWrapper) Context() context.Context {
	return w.stream.Context()
}
