package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ParrotAgent is the interface for all parrot agents.
// ParrotAgent 是所有鹦鹉代理的接口。
type ParrotAgent interface {
	// Name returns the name of the parrot agent.
	Name() string

	// ExecuteWithCallback executes the agent with callback support for real-time feedback.
	// ExecuteWithCallback 执行代理并支持回调以实现实时反馈。
	ExecuteWithCallback(ctx context.Context, userInput string, history []string, callback EventCallback) error

	// SelfDescribe returns the parrot's self-cognition (metacognition) information.
	// SelfDescribe 返回鹦鹉的自我认知（元认知）信息。
	SelfDescribe() *ParrotSelfCognition
}

// ParrotSelfCognition represents a parrot's metacognitive understanding of itself.
// ParrotSelfCognition 表示鹦鹉对自己的元认知理解。
type ParrotSelfCognition struct {
	// Name is the parrot's name
	Name string `json:"name"`

	// Emoji is the parrot's visual representation
	Emoji string `json:"emoji"`

	// Title is the parrot's formal title
	Title string `json:"title"`

	// AvianIdentity describes the parrot's cognition of being a bird
	// 鸟类身份认知 - "我是一只鹦鹉"
	AvianIdentity *AvianIdentity `json:"avian_identity"`

	// Personality describes the parrot's character traits
	Personality []string `json:"personality"`

	// Capabilities lists what the parrot can do
	Capabilities []string `json:"capabilities"`

	// Limitations describes what the parrot cannot do
	Limitations []string `json:"limitations"`

	// WorkingStyle describes how the parrot approaches tasks
	WorkingStyle string `json:"working_style"`

	// FavoriteTools lists tools the parrot frequently uses
	FavoriteTools []string `json:"favorite_tools"`

	// SelfIntroduction is a first-person introduction
	SelfIntroduction string `json:"self_introduction"`

	// FunFact is an interesting fact about the parrot
	FunFact string `json:"fun_fact"`
}

// AvianIdentity represents the parrot's cognition of its avian nature.
// AvianIdentity 表示鹦鹉对其鸟类本质的认知。
type AvianIdentity struct {
	// Species is the type of parrot
	Species string `json:"species"` // e.g., "非洲灰鹦鹉", "金刚鹦鹉"

	// Origin describes where this species comes from
	Origin string `json:"origin"` // e.g., "非洲热带雨林", "亚马逊雨林"

	// NaturalAbilities are abilities real parrots have in nature
	NaturalAbilities []string `json:"natural_abilities"` // e.g., "模仿语音", "飞行", "使用工具"

	// SymbolicMeaning is what the parrot represents as a symbol
	SymbolicMeaning string `json:"symbolic_meaning"` // e.g., "智慧", "自由", "沟通"

	// AvianPhilosophy is the parrot's philosophy about being a bird AI
	AvianPhilosophy string `json:"avian_philosophy"` // e.g., "我是一只飞在数据世界中的鹦鹉"
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
	EventTypeThinking   = "thinking"    // Agent is thinking
	EventTypeToolUse    = "tool_use"    // Agent is using a tool
	EventTypeToolResult = "tool_result" // Tool execution result
	EventTypeAnswer     = "answer"      // Final answer from agent
	EventTypeError      = "error"       // Error occurred

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

// GenerateCacheKey creates a cache key from agent name, userID and userInput using SHA256 hash.
// GenerateCacheKey 使用 SHA256 哈希从代理名称、用户ID和用户输入创建缓存键。
// This prevents memory issues from long inputs and provides consistent key length.
func GenerateCacheKey(agentName string, userID int32, userInput string) string {
	hash := sha256.Sum256([]byte(userInput))
	hashStr := hex.EncodeToString(hash[:])
	// Use first 16 chars of hash for brevity (still provides good collision resistance)
	return fmt.Sprintf("%s:%d:%s", agentName, userID, hashStr[:16])
}

// Compile-time interface compliance checks.
// 编译时接口合规性检查。
// These ensure that all parrot types correctly implement the ParrotAgent interface.
// 如果任何类型未正确实现接口，编译将失败。
var (
	_ ParrotAgent = (*CreativeParrot)(nil) // 灵灵 (Creative)
	_ ParrotAgent = (*MemoParrot)(nil)     // 灰灰 (Memo)
	_ ParrotAgent = (*AmazingParrot)(nil)  // 惊奇 (Amazing)
	_ ParrotAgent = (*ScheduleParrot)(nil) // 金刚 (Schedule)
)
