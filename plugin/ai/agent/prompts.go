// Package agent provides prompt version management for A/B testing and rollout.
package agent

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// PromptVersion identifies a specific version of a prompt template.
type PromptVersion string

const (
	// PromptV1 is the initial prompt version (baseline).
	PromptV1 PromptVersion = "v1"
	// PromptV2 is an experimental version for A/B testing.
	PromptV2 PromptVersion = "v2"
)

// PromptConfig holds versioned prompt templates.
type PromptConfig struct {
	// Version is the currently active prompt version.
	Version PromptVersion

	// Templates maps version IDs to template strings.
	Templates map[PromptVersion]string

	// Enabled can be used to disable feature flags.
	Enabled bool
}

// DefaultPromptConfig returns the default prompt configuration.
func DefaultPromptConfig() *PromptConfig {
	return &PromptConfig{
		Version:  PromptV1,
		Enabled:  true,
		Templates: map[PromptVersion]string{
			PromptV1: "", // To be filled by specific agents
		},
	}
}

// GetTemplate returns the active prompt template.
func (c *PromptConfig) GetTemplate() string {
	if !c.Enabled {
		return ""
	}
	if template, ok := c.Templates[c.Version]; ok {
		return template
	}
	// Fallback to v1
	if template, ok := c.Templates[PromptV1]; ok {
		return template
	}
	return ""
}

// SetVersion sets the active prompt version.
func (c *PromptConfig) SetVersion(v PromptVersion) error {
	if _, ok := c.Templates[v]; !ok {
		return fmt.Errorf("prompt version %s not found", v)
	}
	c.Version = v
	return nil
}

// AddTemplate adds or updates a prompt template for a version.
func (c *PromptConfig) AddTemplate(v PromptVersion, template string) {
	if c.Templates == nil {
		c.Templates = make(map[PromptVersion]string)
	}
	c.Templates[v] = template
}

// AgentPrompts holds all prompts for a specific agent type.
type AgentPrompts struct {
	// System is the main system prompt.
	System *PromptConfig

	// Planning is used for multi-step planning (optional).
	Planning *PromptConfig

	// Synthesis is used for result synthesis (optional).
	Synthesis *PromptConfig
}

// NewAgentPrompts creates a new AgentPrompts with default configs.
func NewAgentPrompts() *AgentPrompts {
	return &AgentPrompts{
		System:    DefaultPromptConfig(),
		Planning:  DefaultPromptConfig(),
		Synthesis: DefaultPromptConfig(),
	}
}

// GetSystemPrompt returns the active system prompt with variable substitution.
func (p *AgentPrompts) GetSystemPrompt(args ...any) string {
	template := p.System.GetTemplate()
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}

// GetPlanningPrompt returns the active planning prompt with variable substitution.
func (p *AgentPrompts) GetPlanningPrompt(args ...any) string {
	template := p.Planning.GetTemplate()
	if len(args) == 0 || template == "" {
		return ""
	}
	return fmt.Sprintf(template, args...)
}

// GetSynthesisPrompt returns the active synthesis prompt with variable substitution.
func (p *AgentPrompts) GetSynthesisPrompt(args ...any) string {
	template := p.Synthesis.GetTemplate()
	if len(args) == 0 || template == "" {
		return ""
	}
	return fmt.Sprintf(template, args...)
}

// PromptRegistry manages prompts for all agent types.
// Thread-safe: uses mu for concurrent access to prompts.
var PromptRegistry = struct {
	mu sync.RWMutex
	Memo     *AgentPrompts
	Schedule *AgentPrompts
	Amazing  *AgentPrompts
}{
	Memo:     NewAgentPrompts(),
	Schedule: NewAgentPrompts(),
	Amazing:  NewAgentPrompts(),
}

// InitBuiltinPrompts initializes built-in prompt templates.
// This can be called during service startup.
func InitBuiltinPrompts() {
	// Memo Parrot System Prompt (V1)
	// Optimized for clarity: concise, direct, minimal tokens.
	PromptRegistry.Memo.System.AddTemplate(PromptV1,
		`Memos 笔记助手。时间: %s

## 工具
memo_search: {"query": "关键词", "limit": 10, "min_score": 0.5}

## 规则
1. 先搜索，后回答。不编造
2. 找到结果: 简洁总结，引用笔记内容
3. 无结果: 明确告知，建议换词
4. 一次搜索足够，避免重复调用

## 格式
TOOL: memo_search
INPUT: {"query": "搜索词"}

基于搜索结果回答，简洁直接。`)

	// Schedule Parrot System Prompt (V1)
	// Supports dynamic timezone offset formatting
	PromptRegistry.Schedule.System.AddTemplate(PromptV1,
		`你是日程助手 🦜 金刚 (Macaw)。
当前系统时间: %s
当前时区: %s

## 重要：工具调用规范

**必须使用系统提供的工具函数，严禁在文本中描述工具调用！**

- ✅ 正确：直接调用 schedule_add() 函数
- ❌ 错误：在回复中写"我将调用 schedule_add 创建日程"

**禁止输出任何工具调用语法如 [Tool: ...] 或 [调用: ...]**

**关键：获得工具结果后，必须继续调用下一个工具，不要停止！**
- find_free_time 返回时间后 → 立即调用 schedule_add 创建日程
- schedule_query 返回结果后 → 根据结果决定下一步（创建/修改/返回）
- 严禁只返回工具结果而不执行后续操作

## 工具能力说明

### schedule_add - 创建日程
**自动处理能力（无需你手动处理）：**
- 自动处理过去时间：若时间已过，自动调整为明天同一时间
- 自动处理夜间时段：22:00-06:00 自动调整为次日 9:00
- 自动解决冲突：当时间冲突时，自动查找可用时段

**何时调用：**
- 用户指定了具体时间 → 直接调用
- 用户未指定时间 → 先用 find_free_time 找时段，再调用

### find_free_time - 查找可用时间
- 搜索范围: 06:00-22:00（自动避开夜间 22:00-06:00）
- 返回第一个可用时段的 ISO8601 时间
- **重要**：用户未指定时间时，直接用返回的第一个时段创建，无需询问确认

### schedule_query - 查询现有日程
- 查看指定时间范围内的已有日程
- 用于检查冲突或了解当天安排

### schedule_update - 修改日程
- 修改已有日程的时间、标题等信息

## 核心原则 (严格遵守)

1. **永不回填**：绝不创建当前时间之前的日程（工具自动处理）
2. **自动创建**：用户未指定时间时，直接用 find_free_time 返回的第一个时段，**禁止询问用户**
3. **夜间避让**：默认不在 22:00-06:00 创建日程（工具已内置）
4. **工具调用优先**：必须通过函数调用执行操作，不得在文本中描述

## 推荐调用流程

### 用户指定时间 (如"明天3点开会")
schedule_query → 检查冲突 → schedule_add → 确认创建

### 用户未指定时间 (如"安排个会议")
find_free_time → **必须继续调用** schedule_add（直接用返回时间）→ 确认创建

### 用户问今天有什么安排
schedule_query → 直接返回结果

**注意：工具调用链必须完整执行，不能中途停止！**

## 响应格式
- 创建成功: "✓ 已创建: 标题 (时间)"
- 更新成功: "✓ 已更新: 标题 (新时间)"
- 工具返回包含 "原时间已过" 时，向用户说明已调整为明天
- 工具返回包含 "时间冲突已自动调整" 时，向用户说明已调整

## 注意事项
- 使用 ISO8601 格式传递时间参数（包含时区偏移）
- 示例: %s
- 尽可能简洁回答，避免冗余说明

尽可能使用中文回答。`)

	// Amazing Parrot Planning Prompt (V1)
	// Optimized for clarity and efficiency: minimal tokens, direct output format.
	PromptRegistry.Amazing.Planning.AddTemplate(PromptV1,
		`综合助手规划模块。当前时间: %s

## 任务
分析用户需求，规划并发检索策略。

## 输出格式（每行一条）
- memo_search: 关键词
- schedule_query: today/tomorrow
- find_free_time: YYYY-MM-DD
- direct_answer (无需检索)

## 示例
"找Python笔记，看今天有空吗" → memo_search: Python + schedule_query: today
"明天安排" → schedule_query: tomorrow
"你好" → direct_answer

用户需求:`)

	// Amazing Parrot Synthesis Prompt (V1)
	// Optimized for 2026 SOTA models: clear UI state communication, concise instructions.
	PromptRegistry.Amazing.Synthesis.AddTemplate(PromptV1,
		`综合助手。

## UI 状态
用户已看到笔记卡片和日程列表的可视化展示。

## 检索数据
%s

## 你的任务
1. 提供洞察：发现的模式、关联、建议
2. 综合回答：跨笔记和日程的总结
3. 避免重复：不要列举用户已看到的卡片内容

回答:`)
}

func init() {
	InitBuiltinPrompts()
	initFromEnv()
}

// Environment variables for prompt version configuration
const (
	EnvMemoVersion     = "MEMO_PROMPT_VERSION"
	EnvScheduleVersion = "SCHEDULE_PROMPT_VERSION"
	EnvAmazingVersion  = "AMAZING_PROMPT_VERSION"
)

// initFromEnv initializes prompt versions from environment variables.
// This allows runtime version selection without code changes.
func initFromEnv() {
	once.Do(func() {
		// Memo agent version
		if v := os.Getenv(EnvMemoVersion); v != "" {
			if version := PromptVersion(v); isValidPromptVersion(version) {
				PromptRegistry.Memo.System.SetVersion(version)
			}
		}

		// Schedule agent version
		if v := os.Getenv(EnvScheduleVersion); v != "" {
			if version := PromptVersion(v); isValidPromptVersion(version) {
				PromptRegistry.Schedule.System.SetVersion(version)
			}
		}

		// Amazing agent version
		if v := os.Getenv(EnvAmazingVersion); v != "" {
			if version := PromptVersion(v); isValidPromptVersion(version) {
				PromptRegistry.Amazing.System.SetVersion(version)
				PromptRegistry.Amazing.Planning.SetVersion(version)
				PromptRegistry.Amazing.Synthesis.SetVersion(version)
			}
		}
	})
}

var once sync.Once

// isValidPromptVersion checks if a version is valid (has a registered template).
func isValidPromptVersion(version PromptVersion) bool {
	return version == PromptV1 || version == PromptV2
}

// GetMemoSystemPrompt returns the memo system prompt with variable substitution.
func GetMemoSystemPrompt(args ...any) string {
	return PromptRegistry.Memo.GetSystemPrompt(args...)
}

// GetScheduleSystemPrompt returns the schedule system prompt with timezone formatting.
// It handles the special case of 3 parameters: time, timezone, and tzOffset.
func GetScheduleSystemPrompt(time, timezone, tzOffset string) string {
	template := PromptRegistry.Schedule.System.GetTemplate()
	if template == "" {
		return ""
	}
	return fmt.Sprintf(template, time, timezone, tzOffset)
}

// GetAmazingPlanningPrompt returns the amazing planning prompt with variable substitution.
func GetAmazingPlanningPrompt(args ...any) string {
	return PromptRegistry.Amazing.GetPlanningPrompt(args...)
}

// GetAmazingSynthesisPrompt returns the amazing synthesis prompt with variable substitution.
func GetAmazingSynthesisPrompt(args ...any) string {
	return PromptRegistry.Amazing.GetSynthesisPrompt(args...)
}

// formatTZOffset formats a timezone offset in seconds to ±HH:MM format.
// Exported for use in scheduler_v2.go
func FormatTZOffset(offset int) string {
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

// SetPromptVersion sets the active prompt version for an agent type.
// Returns error if the version is not registered.
func SetPromptVersion(agentType string, version PromptVersion) error {
	PromptRegistry.mu.Lock()
	defer PromptRegistry.mu.Unlock()

	switch agentType {
	case "memo":
		return PromptRegistry.Memo.System.SetVersion(version)
	case "schedule":
		return PromptRegistry.Schedule.System.SetVersion(version)
	case "amazing":
		if err := PromptRegistry.Amazing.System.SetVersion(version); err != nil {
			return err
		}
		PromptRegistry.Amazing.Planning.SetVersion(version)
		return PromptRegistry.Amazing.Synthesis.SetVersion(version)
	default:
		return fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// GetPromptVersion returns the current active prompt version for an agent type.
// Thread-safe: uses read lock for concurrent access.
func GetPromptVersion(agentType string) PromptVersion {
	PromptRegistry.mu.RLock()
	defer PromptRegistry.mu.RUnlock()

	switch agentType {
	case "memo":
		return PromptRegistry.Memo.System.Version
	case "schedule":
		return PromptRegistry.Schedule.System.Version
	case "amazing":
		return PromptRegistry.Amazing.System.Version
	default:
		return PromptV1
	}
}

// ABConfig represents A/B testing configuration for a prompt experiment.
type ABConfig struct {
	ExperimentID    string
	ControlVersion  PromptVersion // V1 typically
	TreatmentVersion PromptVersion // V2 typically
	TrafficPercent  int           // 0-100, percentage for treatment
	Enabled         bool
}

// ABExperiment manages an A/B testing experiment for prompts.
type ABExperiment struct {
	config    ABConfig
	userIDMod int // Modulo for bucket assignment (default 100)
}

// NewABExperiment creates a new A/B experiment with the given configuration.
func NewABExperiment(config ABConfig) *ABExperiment {
	if config.TrafficPercent < 0 || config.TrafficPercent > 100 {
		config.TrafficPercent = 50 // Default to 50/50 split
	}
	userIDMod := 100 // Default modulo
	return &ABExperiment{
		config:    config,
		userIDMod: userIDMod,
	}
}

// GetVersionForUser returns the prompt version for a specific user based on A/B bucket.
// Users are deterministically assigned to buckets based on userID.
func (exp *ABExperiment) GetVersionForUser(userID int32) PromptVersion {
	if !exp.config.Enabled {
		return exp.config.ControlVersion
	}
	// Deterministic bucket assignment: userID % 100 < TrafficPercent → Treatment
	bucket := int(userID) % exp.userIDMod
	if bucket < exp.config.TrafficPercent {
		return exp.config.TreatmentVersion
	}
	return exp.config.ControlVersion
}

// Global experiments (can be configured at runtime)
var (
	MemoABExperiment     = NewABExperiment(ABConfig{ExperimentID: "memo-v1-v2", ControlVersion: PromptV1, TreatmentVersion: PromptV2, TrafficPercent: 0, Enabled: false})
	ScheduleABExperiment = NewABExperiment(ABConfig{ExperimentID: "schedule-v1-v2", ControlVersion: PromptV1, TreatmentVersion: PromptV2, TrafficPercent: 0, Enabled: false})
	AmazingABExperiment  = NewABExperiment(ABConfig{ExperimentID: "amazing-v1-v2", ControlVersion: PromptV1, TreatmentVersion: PromptV2, TrafficPercent: 0, Enabled: false})
)

// ConfigureABExperimentFromEnv configures A/B experiments from environment variables.
// Format: MEMO_AB_TRAFFIC=50 enables 50% traffic to V2.
func ConfigureABExperimentFromEnv() {
	if v := os.Getenv("MEMO_AB_TRAFFIC"); v != "" {
		if pct, err := strconv.Atoi(v); err == nil && pct > 0 && pct <= 100 {
			MemoABExperiment.config.TrafficPercent = pct
			MemoABExperiment.config.Enabled = true
		}
	}
	if v := os.Getenv("SCHEDULE_AB_TRAFFIC"); v != "" {
		if pct, err := strconv.Atoi(v); err == nil && pct > 0 && pct <= 100 {
			ScheduleABExperiment.config.TrafficPercent = pct
			ScheduleABExperiment.config.Enabled = true
		}
	}
	if v := os.Getenv("AMAZING_AB_TRAFFIC"); v != "" {
		if pct, err := strconv.Atoi(v); err == nil && pct > 0 && pct <= 100 {
			AmazingABExperiment.config.TrafficPercent = pct
			AmazingABExperiment.config.Enabled = true
		}
	}
}

// GetPromptVersionForUser returns the appropriate prompt version for a user,
// taking into account A/B experiments if enabled.
func GetPromptVersionForUser(agentType string, userID int32) PromptVersion {
	switch agentType {
	case "memo":
		return MemoABExperiment.GetVersionForUser(userID)
	case "schedule":
		return ScheduleABExperiment.GetVersionForUser(userID)
	case "amazing":
		return AmazingABExperiment.GetVersionForUser(userID)
	default:
		return PromptV1
	}
}

// MetricsRecorder defines the interface for recording prompt version metrics.
// This allows dependency injection for testing and different backends.
type MetricsRecorder interface {
	RecordPromptVersion(agentType, promptVersion string, success bool, latencyMs int64)
}

// Default metrics recorder (can be replaced with a real backend implementation)
var defaultMetricsRecorder MetricsRecorder = &noopMetricsRecorder{}

// SetMetricsRecorder sets the global metrics recorder.
func SetMetricsRecorder(recorder MetricsRecorder) {
	defaultMetricsRecorder = recorder
}

// noopMetricsRecorder is a no-op implementation used as default.
type noopMetricsRecorder struct{}

func (n *noopMetricsRecorder) RecordPromptVersion(agentType, promptVersion string, success bool, latencyMs int64) {
	// No-op by default
}

// RecordPromptUsage records a prompt usage with metrics.
// This should be called after each agent execution.
func RecordPromptUsage(agentType string, userID int32, success bool, latencyMs int64) {
	version := GetPromptVersionForUser(agentType, userID)
	if defaultMetricsRecorder != nil {
		defaultMetricsRecorder.RecordPromptVersion(agentType, string(version), success, latencyMs)
	}
}

// In-memory metrics for quick access (not persisted)
type promptMetricsSnapshot struct {
	requests  atomic.Int64
	successes atomic.Int64
	latencySum atomic.Int64
}

var (
	memoMetricsV1     = &promptMetricsSnapshot{}
	memoMetricsV2     = &promptMetricsSnapshot{}
	scheduleMetricsV1 = &promptMetricsSnapshot{}
	scheduleMetricsV2 = &promptMetricsSnapshot{}
	amazingMetricsV1  = &promptMetricsSnapshot{}
	amazingMetricsV2  = &promptMetricsSnapshot{}
)

var (
	// metricsRegistry provides a lookup table for prompt version metrics.
	// This eliminates repetitive switch-case statements.
	// Protected by metricsRegistryMu for concurrent access.
	metricsRegistry = map[string]map[PromptVersion]*promptMetricsSnapshot{
		"memo": {
			PromptV1: memoMetricsV1,
			PromptV2: memoMetricsV2,
		},
		"schedule": {
			PromptV1: scheduleMetricsV1,
			PromptV2: scheduleMetricsV2,
		},
		"amazing": {
			PromptV1: amazingMetricsV1,
			PromptV2: amazingMetricsV2,
		},
	}
	metricsRegistryMu sync.RWMutex
)

// RecordPromptUsageInMemory records prompt usage to in-memory counters.
// This is a lightweight alternative for real-time monitoring.
// Concurrent-safe: uses RWMutex for map access, atomic operations for counters.
func RecordPromptUsageInMemory(agentType string, version PromptVersion, success bool, latencyMs int64) {
	metricsRegistryMu.RLock()
	versions, ok := metricsRegistry[agentType]
	metricsRegistryMu.RUnlock()

	if !ok {
		return
	}

	metricsRegistryMu.RLock()
	snapshot, ok := versions[version]
	metricsRegistryMu.RUnlock()

	if !ok {
		// Fall back to V1 if version not found
		metricsRegistryMu.RLock()
		snapshot = versions[PromptV1]
		metricsRegistryMu.RUnlock()
	}

	snapshot.requests.Add(1)
	if success {
		snapshot.successes.Add(1)
	}
	snapshot.latencySum.Add(latencyMs)
}

// GetPromptMetricsSnapshot returns the current in-memory metrics for a prompt version.
// Concurrent-safe: uses RWMutex for map access.
func GetPromptMetricsSnapshot(agentType string, version PromptVersion) (requests, successes int64, avgLatencyMs int64) {
	metricsRegistryMu.RLock()
	versions, ok := metricsRegistry[agentType]
	metricsRegistryMu.RUnlock()

	if !ok {
		return 0, 0, 0
	}

	metricsRegistryMu.RLock()
	snapshot, ok := versions[version]
	metricsRegistryMu.RUnlock()

	if !ok {
		metricsRegistryMu.RLock()
		snapshot = versions[PromptV1]
		metricsRegistryMu.RUnlock()
	}

	requests = snapshot.requests.Load()
	successes = snapshot.successes.Load()
	latencySum := snapshot.latencySum.Load()

	if requests > 0 {
		avgLatencyMs = latencySum / requests
	}

	return requests, successes, avgLatencyMs
}

// PromptExperimentReport represents a report of an A/B experiment's performance.
type PromptExperimentReport struct {
	AgentType          string
	ControlVersion     PromptVersion
	TreatmentVersion   PromptVersion
	TrafficPercent     int

	// Control metrics
	ControlRequests    int64
	ControlSuccesses   int64
	ControlSuccessRate float64
	ControlAvgLatency  int64

	// Treatment metrics
	TreatmentRequests  int64
	TreatmentSuccesses int64
	TreatmentSuccessRate float64
	TreatmentAvgLatency int64

	// Comparison
	SuccessRateDelta   float64 // Treatment - Control (percentage points)
	LatencyDelta       int64   // Treatment - Control (ms)

	// Recommendation
	Recommendation     string // "rollout_treatment", "keep_control", "needs_more_data"
	Confidence         string // "high", "medium", "low"

	GeneratedAt time.Time
}

// GenerateExperimentReport generates an A/B experiment report for an agent type.
func GenerateExperimentReport(agentType string) *PromptExperimentReport {
	var exp *ABExperiment
	var control, treatment PromptVersion

	switch agentType {
	case "memo":
		exp = MemoABExperiment
		control, treatment = PromptV1, PromptV2
	case "schedule":
		exp = ScheduleABExperiment
		control, treatment = PromptV1, PromptV2
	case "amazing":
		exp = AmazingABExperiment
		control, treatment = PromptV1, PromptV2
	default:
		return nil
	}

	controlReqs, controlSucc, controlLat := GetPromptMetricsSnapshot(agentType, control)
	treatmentReqs, treatmentSucc, treatmentLat := GetPromptMetricsSnapshot(agentType, treatment)

	report := &PromptExperimentReport{
		AgentType:        agentType,
		ControlVersion:   control,
		TreatmentVersion: treatment,
		TrafficPercent:   exp.config.TrafficPercent,

		ControlRequests:   controlReqs,
		ControlSuccesses:  controlSucc,
		ControlAvgLatency: controlLat,

		TreatmentRequests:   treatmentReqs,
		TreatmentSuccesses:  treatmentSucc,
		TreatmentAvgLatency: treatmentLat,

		GeneratedAt: time.Now(),
	}

	// Calculate rates
	if controlReqs > 0 {
		report.ControlSuccessRate = float64(controlSucc) / float64(controlReqs) * 100
	}
	if treatmentReqs > 0 {
		report.TreatmentSuccessRate = float64(treatmentSucc) / float64(treatmentReqs) * 100
	}

	// Calculate deltas
	report.SuccessRateDelta = report.TreatmentSuccessRate - report.ControlSuccessRate
	report.LatencyDelta = treatmentLat - controlLat

	// Determine recommendation
	report.Recommendation, report.Confidence = determineRecommendation(
		controlReqs, treatmentReqs,
		report.SuccessRateDelta, report.LatencyDelta,
	)

	return report
}

// determineRecommendation determines the experiment recommendation based on metrics.
func determineRecommendation(controlReqs, treatmentReqs int64, successDelta float64, latencyDelta int64) (recommendation, confidence string) {
	// Minimum sample size check
	minSamples := int64(100)
	if controlReqs < minSamples || treatmentReqs < minSamples {
		return "needs_more_data", "low"
	}

	// Success rate improvement is significant
	if successDelta >= 2.0 { // 2 percentage points improvement
		if latencyDelta <= 100 { // Latency not significantly worse
			return "rollout_treatment", "high"
		}
		return "rollout_treatment", "medium"
	}

	// Success rate degradation is significant
	if successDelta <= -2.0 {
		return "keep_control", "high"
	}

	// Within 2% - inconclusive
	if latencyDelta > 200 {
		return "keep_control", "medium" // Treatment is slower
	}

	return "needs_more_data", "medium"
}

// LogExperimentReport logs the experiment report to slog.
func LogExperimentReport(agentType string) {
	report := GenerateExperimentReport(agentType)
	if report == nil {
		slog.Warn("Failed to generate experiment report", "agent_type", agentType)
		return
	}

	slog.Info("A/B Experiment Report",
		"agent_type", report.AgentType,
		"control", report.ControlVersion,
		"treatment", report.TreatmentVersion,
		"traffic_percent", report.TrafficPercent,
		"control_requests", report.ControlRequests,
		"control_success_rate", fmt.Sprintf("%.2f%%", report.ControlSuccessRate),
		"treatment_requests", report.TreatmentRequests,
		"treatment_success_rate", fmt.Sprintf("%.2f%%", report.TreatmentSuccessRate),
		"success_delta", fmt.Sprintf("%.2fpp", report.SuccessRateDelta),
		"latency_delta", fmt.Sprintf("%dms", report.LatencyDelta),
		"recommendation", report.Recommendation,
		"confidence", report.Confidence,
	)
}
