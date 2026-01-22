package agent

import (
	"context"
	"fmt"
)

// ParrotAgent is the interface for all parrot agents.
// ParrotAgent 是所有鹦鹉代理的接口。
type ParrotAgent interface {
	// Name returns the name of the parrot agent.
	Name() string

	// ExecuteWithCallback executes the agent with callback support for real-time feedback.
	// ExecuteWithCallback 执行代理并支持回调以实现实时反馈。
	ExecuteWithCallback(ctx context.Context, userInput string, callback EventCallback) error
}

// EventCallback is the callback function type for agent events.
// EventCallback 是代理事件的回调函数类型。
//
// The callback receives:
//   - eventType: The type of event (e.g., "thinking", "tool_use", "tool_result", "answer", "error")
//   - eventData: The event data (can be a struct, string, or nil)
//
// 返回错误将中止代理执行。
type EventCallback func(eventType string, eventData interface{}) error

// Common event types
// 常用事件类型
const (
	EventTypeThinking   = "thinking"   // Agent is thinking
	EventTypeToolUse    = "tool_use"   // Agent is using a tool
	EventTypeToolResult = "tool_result" // Tool execution result
	EventTypeAnswer     = "answer"     // Final answer from agent
	EventTypeError      = "error"      // Error occurred

	// Memo-specific events
	EventTypeMemoQueryResult = "memo_query_result" // Memo search results

	// Schedule-specific events
	EventTypeScheduleQueryResult = "schedule_query_result" // Schedule query results
	EventTypeScheduleUpdated     = "schedule_updated"      // Schedule created/updated
)

// MemoQueryResultData represents the result of a memo search.
// MemoQueryResultData 表示笔记搜索的结果。
type MemoQueryResultData struct {
	Memos []MemoSummary `json:"memos"`
	Query string        `json:"query"`
	Count int           `json:"count"`
}

// MemoSummary represents a simplified memo for query results.
// MemoSummary 表示查询结果中的简化笔记。
type MemoSummary struct {
	UID     string  `json:"uid"`
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

// ScheduleQueryResultData represents the result of a schedule query.
// ScheduleQueryResultData 表示日程查询的结果。
type ScheduleQueryResultData struct {
	Schedules            []ScheduleSummary `json:"schedules"`
	Query                string            `json:"query"`
	Count                int               `json:"count"`
	TimeRangeDescription string            `json:"time_range_description"`
	QueryType            string            `json:"query_type"` // e.g., "upcoming", "range", "filter"
}

// ScheduleSummary represents a simplified schedule for query results.
// ScheduleSummary 表示查询结果中的简化日程。
type ScheduleSummary struct {
	UID            string `json:"uid"`
	Title          string `json:"title"`
	StartTimestamp int64  `json:"start_ts"`
	EndTimestamp   int64  `json:"end_ts"`
	AllDay         bool   `json:"all_day"`
	Location       string `json:"location,omitempty"`
	Status         string `json:"status"`
}

// ParrotStream is the interface for streaming responses to the client.
// ParrotStream 是向客户端流式传输响应的接口。
type ParrotStream interface {
	// Send sends an event to the client.
	// Send 向客户端发送一个事件。
	Send(eventType string, eventData interface{}) error

	// Close closes the stream.
	// Close 关闭流。
	Close() error
}

// ParrotStreamAdapter adapts Connect RPC server stream to ParrotStream interface.
// ParrotStreamAdapter 将 Connect RPC 服务端流适配到 ParrotStream 接口。
type ParrotStreamAdapter struct {
	// The actual stream implementation will be provided by the caller
	// 实际的流实现将由调用者提供
	sendFunc func(eventType string, eventData interface{}) error
}

// NewParrotStreamAdapter creates a new ParrotStreamAdapter.
// NewParrotStreamAdapter 创建一个新的 ParrotStreamAdapter。
func NewParrotStreamAdapter(sendFunc func(eventType string, eventData interface{}) error) *ParrotStreamAdapter {
	return &ParrotStreamAdapter{
		sendFunc: sendFunc,
	}
}

// Send sends an event through the adapter.
// Send 通过适配器发送事件。
func (a *ParrotStreamAdapter) Send(eventType string, eventData interface{}) error {
	if a.sendFunc == nil {
		return fmt.Errorf("send function not set")
	}
	return a.sendFunc(eventType, eventData)
}

// Close is a no-op for the adapter (the caller manages stream lifecycle).
// Close 对适配器来说是无操作（调用者管理流的生命周期）。
func (a *ParrotStreamAdapter) Close() error {
	return nil
}

// ParrotError represents an error from a parrot agent.
// ParrotError 表示来自鹦鹉代理的错误。
type ParrotError struct {
	AgentName string // Name of the agent that produced the error
	Operation string // Operation being performed when error occurred
	Err       error  // Underlying error
}

// Error implements the error interface.
// Error 实现错误接口。
func (e *ParrotError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("parrot %s: %s failed: %v", e.AgentName, e.Operation, e.Err)
}

// Unwrap returns the underlying error.
// Unwrap 返回底层错误。
func (e *ParrotError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewParrotError creates a new ParrotError.
// NewParrotError 创建一个新的 ParrotError。
func NewParrotError(agentName, operation string, err error) *ParrotError {
	return &ParrotError{
		AgentName: agentName,
		Operation: operation,
		Err:       err,
	}
}
