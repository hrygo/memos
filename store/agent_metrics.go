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

// PromptVersionMetrics represents metrics for a specific prompt version in an A/B experiment.
// This enables comparison of prompt performance across versions.
type PromptVersionMetrics struct {
	ID               int64
	HourBucket       time.Time
	AgentType        string
	PromptVersion    string // e.g., "v1", "v2"
	RequestCount     int64
	SuccessCount     int64
	AvgLatencyMs     int64
	UserSatisfaction float32 // Optional: derived from user feedback
}

// UpsertPromptVersionMetrics specifies the data for upserting prompt version metrics.
type UpsertPromptVersionMetrics struct {
	HourBucket    time.Time
	AgentType     string
	PromptVersion string
	RequestCount  int64
	SuccessCount  int64
	AvgLatencyMs  int64
}

// FindPromptVersionMetrics specifies the conditions for finding prompt version metrics.
type FindPromptVersionMetrics struct {
	AgentType     *string
	PromptVersion *string
	StartTime     *time.Time
	EndTime       *time.Time
	Limit         int
}

// PromptExperimentSummary represents aggregated metrics for an A/B experiment.
// Used for comparing control vs treatment performance.
type PromptExperimentSummary struct {
	AgentType              string
	ControlVersion         string
	TreatmentVersion       string
	ControlRequests        int64
	TreatmentRequests      int64
	ControlSuccessRate     float64
	TreatmentSuccessRate   float64
	ControlAvgLatencyMs    int64
	TreatmentAvgLatencyMs  int64
	ImprovementRate        float64 // Percentage improvement of treatment over control
	IsStatisticallySignificant bool
	Recommendation         string // "keep_control", "rollout_treatment", "inconclusive"
}
