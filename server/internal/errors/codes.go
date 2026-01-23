package errors

import (
	"fmt"
)

// ErrorCode represents a specific error type for AI operations.
type ErrorCode string

const (
	// ErrCodeUnauthorized indicates authentication failure.
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrCodeRateLimitExceeded indicates rate limit has been exceeded.
	ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	// ErrCodeInvalidArgument indicates invalid input parameters.
	ErrCodeInvalidArgument ErrorCode = "INVALID_ARGUMENT"
	// ErrCodeServiceUnavailable indicates the service is not available.
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	// ErrCodeAgentExecutionFailed indicates agent execution failure.
	ErrCodeAgentExecutionFailed ErrorCode = "AGENT_EXECUTION_FAILED"
	// ErrCodeAgentNotFound indicates the requested agent type does not exist.
	ErrCodeAgentNotFound ErrorCode = "AGENT_NOT_FOUND"
	// ErrCodeLLMUnavailable indicates the LLM service is not available.
	ErrCodeLLMUnavailable ErrorCode = "LLM_UNAVAILABLE"
	// ErrCodeContextCanceled indicates the operation was canceled.
	ErrCodeContextCanceled ErrorCode = "CONTEXT_CANCELED"
	// ErrCodeTimeout indicates the operation timed out.
	ErrCodeTimeout ErrorCode = "TIMEOUT"
)

// AIError represents a structured error for AI operations.
type AIError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *AIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause.
func (e *AIError) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error.
func (e *AIError) WithContext(key string, value interface{}) *AIError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithContextMap adds multiple context values to the error.
func (e *AIError) WithContextMap(ctx map[string]interface{}) *AIError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	for k, v := range ctx {
		e.Context[k] = v
	}
	return e
}

// GetCode returns the error code.
func (e *AIError) GetCode() ErrorCode {
	return e.Code
}

// Convenience constructors for common error types.

// Unauthorized creates an unauthorized error.
func Unauthorized(msg string) *AIError {
	return &AIError{Code: ErrCodeUnauthorized, Message: msg}
}

// RateLimitExceeded creates a rate limit exceeded error.
func RateLimitExceeded(msg string) *AIError {
	return &AIError{Code: ErrCodeRateLimitExceeded, Message: msg}
}

// InvalidArgument creates an invalid argument error.
func InvalidArgument(msg string) *AIError {
	return &AIError{Code: ErrCodeInvalidArgument, Message: msg}
}

// ServiceUnavailable creates a service unavailable error.
func ServiceUnavailable(msg string) *AIError {
	return &AIError{Code: ErrCodeServiceUnavailable, Message: msg}
}

// AgentExecutionFailed creates an agent execution failed error.
func AgentExecutionFailed(msg string, cause error) *AIError {
	return &AIError{Code: ErrCodeAgentExecutionFailed, Message: msg, Cause: cause}
}

// AgentNotFound creates an agent not found error.
func AgentNotFound(agentType string) *AIError {
	return &AIError{
		Code:    ErrCodeAgentNotFound,
		Message: fmt.Sprintf("agent type not found: %s", agentType),
	}
}

// LLMUnavailable creates an LLM unavailable error.
func LLMUnavailable(msg string) *AIError {
	return &AIError{Code: ErrCodeLLMUnavailable, Message: msg}
}

// ContextCanceled creates a context canceled error.
func ContextCanceled(cause error) *AIError {
	return &AIError{Code: ErrCodeContextCanceled, Message: "operation canceled", Cause: cause}
}

// Timeout creates a timeout error.
func Timeout(msg string) *AIError {
	return &AIError{Code: ErrCodeTimeout, Message: msg}
}

// Wrap wraps an existing error with additional context.
func Wrap(cause error, code ErrorCode, msg string) *AIError {
	return &AIError{Code: code, Message: msg, Cause: cause}
}

// IsCode checks if an error is of a specific code.
func IsCode(err error, code ErrorCode) bool {
	if aiErr, ok := err.(*AIError); ok {
		return aiErr.Code == code
	}
	return false
}

// GetCodeFromError extracts the error code from any error.
// Returns the provided default code if the error is not an AIError.
func GetCodeFromError(err error, defaultCode ErrorCode) ErrorCode {
	if aiErr, ok := err.(*AIError); ok {
		return aiErr.Code
	}
	return defaultCode
}
