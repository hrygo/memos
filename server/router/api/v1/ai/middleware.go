package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/server/internal/errors"
	"github.com/usememos/memos/server/internal/observability"
	"github.com/usememos/memos/server/middleware"
)

// ChatRequest represents a chat request.
type ChatRequest struct {
	Message        string
	History        []string
	AgentType      AgentType
	UserID         int32
	Timezone       string
	ConversationID int32
}

// ChatStream represents the streaming response interface.
type ChatStream interface {
	Send(*v1pb.ChatWithMemosResponse) error
	Context() context.Context
}

// Handler is the interface for handling chat requests.
type Handler interface {
	Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error
}

// Middleware is a function that wraps a handler.
type Middleware func(Handler) Handler

// Chain chains multiple middlewares together.
func Chain(h Handler, middlewares ...Middleware) Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// ValidationMiddleware validates incoming requests.
type ValidationMiddleware struct{}

// NewValidationMiddleware creates a new validation middleware.
func NewValidationMiddleware() Middleware {
	return func(next Handler) Handler {
		return &validationHandler{next: next}
	}
}

type validationHandler struct {
	next Handler
}

func (h *validationHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	if req.Message == "" {
		return status.Error(codes.InvalidArgument, "message is required")
	}
	return h.next.Handle(ctx, req, stream)
}

// AuthMiddleware authenticates the user.
type AuthMiddleware struct {
	store  Store
	logger *observability.RequestContext
}

// Store is the minimal interface needed for authentication.
type Store interface {
	GetUser(ctx context.Context, find *FindUser) (*User, error)
}

// FindUser represents user search criteria.
type FindUser struct {
	ID *int32
}

// User represents a user.
type User struct {
	ID       int32
	Username string
}

// NewAuthMiddleware creates a new auth middleware.
func NewAuthMiddleware(store Store, logger *observability.RequestContext) Middleware {
	return func(next Handler) Handler {
		return &authHandler{
			store:  store,
			logger: logger,
			next:   next,
		}
	}
}

type authHandler struct {
	store  Store
	logger *observability.RequestContext
	next   Handler
}

func (h *authHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	// Get user ID from context (set by authentication layer)
	userID, err := getUserID(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, "unauthorized")
	}

	req.UserID = userID
	return h.next.Handle(ctx, req, stream)
}

// RateLimitMiddleware applies rate limiting.
type RateLimitMiddleware struct {
	limiter *middleware.RateLimiter
	logger  *observability.RequestContext
}

// NewRateLimitMiddleware creates a new rate limit middleware.
func NewRateLimitMiddleware(limiter *middleware.RateLimiter, logger *observability.RequestContext) Middleware {
	return func(next Handler) Handler {
		return &rateLimitHandler{
			limiter: limiter,
			logger:  logger,
			next:    next,
		}
	}
}

type rateLimitHandler struct {
	limiter *middleware.RateLimiter
	logger  *observability.RequestContext
	next    Handler
}

func (h *rateLimitHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	userKey := strconv.FormatInt(int64(req.UserID), 10)
	if !h.limiter.Allow(userKey) {
		return status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	return h.next.Handle(ctx, req, stream)
}

// TruncateString truncates a string for logging.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// getUserID extracts the user ID from context using the auth layer.
func getUserID(ctx context.Context) (int32, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return 0, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// ToAIError converts a gRPC status error to an AIError.
func ToAIError(err error) *errors.AIError {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return errors.Wrap(err, errors.ErrCodeServiceUnavailable, "unknown error")
	}

	switch st.Code() {
	case codes.Unauthenticated:
		return errors.Unauthorized(st.Message())
	case codes.ResourceExhausted:
		return errors.RateLimitExceeded(st.Message())
	case codes.InvalidArgument:
		return errors.InvalidArgument(st.Message())
	case codes.Unavailable:
		return errors.ServiceUnavailable(st.Message())
	case codes.DeadlineExceeded:
		return errors.Timeout(st.Message())
	case codes.Canceled:
		return errors.ContextCanceled(err)
	default:
		return errors.Wrap(err, errors.ErrCodeAgentExecutionFailed, st.Message())
	}
}

// FromAIError converts an AIError to a gRPC status error.
func FromAIError(err *errors.AIError) error {
	if err == nil {
		return nil
	}

	switch err.Code {
	case errors.ErrCodeUnauthorized:
		return status.Error(codes.Unauthenticated, err.Message)
	case errors.ErrCodeRateLimitExceeded:
		return status.Error(codes.ResourceExhausted, err.Message)
	case errors.ErrCodeInvalidArgument:
		return status.Error(codes.InvalidArgument, err.Message)
	case errors.ErrCodeServiceUnavailable, errors.ErrCodeLLMUnavailable:
		return status.Error(codes.Unavailable, err.Message)
	case errors.ErrCodeTimeout:
		return status.Error(codes.DeadlineExceeded, err.Message)
	case errors.ErrCodeContextCanceled:
		return status.Error(codes.Canceled, err.Message)
	default:
		return status.Error(codes.Internal, err.Message)
	}
}

// BuildChatMessages constructs chat messages from user input and history.
func BuildChatMessages(message string, history []string, systemPrompt string) []ai.Message {
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
	}

	// Add history (skip empty messages to avoid LLM API errors)
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			userMsg := history[i]
			assistantMsg := history[i+1]
			// Only add non-empty messages
			if userMsg != "" && assistantMsg != "" {
				messages = append(messages, ai.Message{Role: "user", Content: userMsg})
				messages = append(messages, ai.Message{Role: "assistant", Content: assistantMsg})
			}
		}
	}

	// Add current user message
	messages = append(messages, ai.Message{Role: "user", Content: message})

	return messages
}

// StreamLLMResponse streams LLM responses to the client.
func StreamLLMResponse(
	ctx context.Context,
	llm ai.LLMService,
	messages []ai.Message,
	stream ChatStream,
	logger *observability.RequestContext,
) error {
	llmStart := time.Now()
	contentChan, errChan := llm.ChatStream(ctx, messages)

	// Use strings.Builder for efficient string concatenation
	var fullContent strings.Builder
	var totalChunks int

	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				contentChan = nil
				if errChan == nil {
					logger.Debug("LLM stream completed",
						slog.Int64(observability.LogFieldDuration, time.Since(llmStart).Milliseconds()),
						slog.Int("output_length", fullContent.Len()),
						slog.Int("total_chunks", totalChunks),
					)
					// Send done marker
					return stream.Send(&v1pb.ChatWithMemosResponse{Done: true})
				}
				continue
			}
			totalChunks++
			fullContent.WriteString(content)
			if err := stream.Send(&v1pb.ChatWithMemosResponse{Content: content}); err != nil {
				return err
			}

		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				if contentChan == nil {
					logger.Debug("LLM stream completed",
						slog.Int64(observability.LogFieldDuration, time.Since(llmStart).Milliseconds()),
						slog.Int("output_length", fullContent.Len()),
						slog.Int("total_chunks", totalChunks),
					)
					return stream.Send(&v1pb.ChatWithMemosResponse{Done: true})
				}
				continue
			}
			if err != nil {
				return status.Error(codes.Internal, fmt.Sprintf("LLM error: %v", err))
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
