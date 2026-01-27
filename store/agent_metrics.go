package store

import "time"

// AgentMetrics represents hourly aggregated metrics for an agent type.
type AgentMetrics struct {
	ID           int64
	HourBucket   time.Time
	AgentType    string
	RequestCount int64
	SuccessCount int64
	LatencySumMs int64
	LatencyP50Ms int32
	LatencyP95Ms int32
	Errors       string // JSON: {"error_type": count}
}

// ToolMetrics represents hourly aggregated metrics for a tool.
type ToolMetrics struct {
	ID           int64
	HourBucket   time.Time
	ToolName     string
	CallCount    int64
	SuccessCount int64
	LatencySumMs int64
}

// UpsertAgentMetrics specifies the data for upserting agent metrics.
type UpsertAgentMetrics struct {
	HourBucket   time.Time
	AgentType    string
	RequestCount int64
	SuccessCount int64
	LatencySumMs int64
	LatencyP50Ms int32
	LatencyP95Ms int32
	Errors       string
}

// UpsertToolMetrics specifies the data for upserting tool metrics.
type UpsertToolMetrics struct {
	HourBucket   time.Time
	ToolName     string
	CallCount    int64
	SuccessCount int64
	LatencySumMs int64
}

// FindAgentMetrics specifies the conditions for finding agent metrics.
type FindAgentMetrics struct {
	AgentType *string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
}

// FindToolMetrics specifies the conditions for finding tool metrics.
type FindToolMetrics struct {
	ToolName  *string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
}

// DeleteAgentMetrics specifies the conditions for deleting agent metrics.
type DeleteAgentMetrics struct {
	BeforeTime *time.Time // Delete records older than this time
}

// DeleteToolMetrics specifies the conditions for deleting tool metrics.
type DeleteToolMetrics struct {
	BeforeTime *time.Time // Delete records older than this time
}
