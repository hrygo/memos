package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	// LogFieldRequestID is the field name for request ID.
	LogFieldRequestID = "request_id"
	// LogFieldUserID is the field name for user ID.
	LogFieldUserID = "user_id"
	// LogFieldAgentType is the field name for agent type.
	LogFieldAgentType = "agent_type"
	// LogFieldDuration is the field name for duration in milliseconds.
	LogFieldDuration = "duration_ms"
	// LogFieldMessageLen is the field name for message length.
	LogFieldMessageLen = "message_length"
	// LogFieldErrorCode is the field name for error code.
	LogFieldErrorCode = "error_code"
	// LogFieldEventType is the field name for event type.
	LogFieldEventType = "event_type"
	// LogFieldIteration is the field name for iteration count.
	LogFieldIteration = "iteration"
)

// RequestContext represents the context for a single request with structured logging.
type RequestContext struct {
	RequestID string
	UserID    int32
	AgentType string
	StartTime time.Time
	Logger    *slog.Logger
}

// NewRequestContext creates a new request context with a generated request ID.
func NewRequestContext(logger *slog.Logger, agentType string, userID int32) *RequestContext {
	return &RequestContext{
		RequestID: generateRequestID(),
		UserID:    userID,
		AgentType: agentType,
		StartTime: time.Now(),
		Logger:    logger,
	}
}

// NewRequestContextWithID creates a new request context with a specific request ID.
func NewRequestContextWithID(logger *slog.Logger, requestID, agentType string, userID int32) *RequestContext {
	return &RequestContext{
		RequestID: requestID,
		UserID:    userID,
		AgentType: agentType,
		StartTime: time.Now(),
		Logger:    logger,
	}
}

// WithFields returns a new logger with additional fields.
func (r *RequestContext) WithFields(attrs ...slog.Attr) *slog.Logger {
	base := r.baseAttrs()
	result := make([]any, 0, len(base)+len(attrs))
	for _, attr := range base {
		result = append(result, attr)
	}
	for _, attr := range attrs {
		result = append(result, attr)
	}
	return r.Logger.With(result...)
}

// Info logs an info message.
func (r *RequestContext) Info(msg string, attrs ...slog.Attr) {
	combined := r.baseAttrsAppended(attrs...)
	r.Logger.LogAttrs(context.Background(), slog.LevelInfo, msg, combined...)
}

// Debug logs a debug message.
func (r *RequestContext) Debug(msg string, attrs ...slog.Attr) {
	combined := r.baseAttrsAppended(attrs...)
	r.Logger.LogAttrs(context.Background(), slog.LevelDebug, msg, combined...)
}

// Warn logs a warning message.
func (r *RequestContext) Warn(msg string, attrs ...slog.Attr) {
	combined := r.baseAttrsAppended(attrs...)
	r.Logger.LogAttrs(context.Background(), slog.LevelWarn, msg, combined...)
}

// Error logs an error message with the error.
func (r *RequestContext) Error(msg string, err error, attrs ...slog.Attr) {
	allAttrs := append(attrs, slog.String("error", err.Error()))
	combined := r.baseAttrsAppended(allAttrs...)
	r.Logger.LogAttrs(context.Background(), slog.LevelError, msg, combined...)
}

// Duration returns the elapsed time since the request started.
func (r *RequestContext) Duration() time.Duration {
	return time.Since(r.StartTime)
}

// DurationMs returns the elapsed time in milliseconds.
func (r *RequestContext) DurationMs() int64 {
	return r.Duration().Milliseconds()
}

// baseAttrs returns the base attributes.
func (r *RequestContext) baseAttrs() []slog.Attr {
	return []slog.Attr{
		slog.String(LogFieldRequestID, r.RequestID),
		slog.Int64(LogFieldUserID, int64(r.UserID)),
		slog.String(LogFieldAgentType, r.AgentType),
	}
}

// baseAttrsAppended combines the base attributes with additional attributes.
func (r *RequestContext) baseAttrsAppended(attrs ...slog.Attr) []slog.Attr {
	base := r.baseAttrs()
	return append(base, attrs...)
}

// generateRequestID generates a unique request ID using full UUID.
func generateRequestID() string {
	return uuid.New().String()
}

// LogCtx is a helper to get request context from context.Context.
// This allows logging functions to accept context and extract request info.
type ctxKey struct{}

// WithRequestContext adds the request context to the context.
func WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, reqCtx)
}

// FromContext extracts the request context from the context.
func FromContext(ctx context.Context) (*RequestContext, bool) {
	reqCtx, ok := ctx.Value(ctxKey{}).(*RequestContext)
	return reqCtx, ok
}

// MustFromContext extracts the request context from the context or panics.
func MustFromContext(ctx context.Context) *RequestContext {
	reqCtx, ok := FromContext(ctx)
	if !ok {
		panic("request context not found in context")
	}
	return reqCtx
}
