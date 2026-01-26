package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	localtools "github.com/usememos/memos/plugin/ai/agent/tools"
	"github.com/usememos/memos/server/service/schedule"
)

// SchedulerAgentV2 is the new framework-less schedule agent.
// It uses native LLM tool calling without LangChainGo dependency.
type SchedulerAgentV2 struct {
	agent            *Agent
	llm              ai.LLMService
	scheduleSvc      schedule.Service
	userID           int32
	timezone         string
	timezoneLoc      *time.Location
	intentClassifier *LLMIntentClassifier // LLM-based intent classification
}

// NewSchedulerAgentV2 creates a new framework-less schedule agent.
func NewSchedulerAgentV2(llm ai.LLMService, scheduleSvc schedule.Service, userID int32, userTimezone string) (*SchedulerAgentV2, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLM service is required")
	}
	if scheduleSvc == nil {
		return nil, fmt.Errorf("schedule service is required")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", userID)
	}

	if userTimezone == "" {
		userTimezone = "Asia/Shanghai"
	}

	timezoneLoc, err := time.LoadLocation(userTimezone)
	if err != nil {
		slog.Warn("invalid timezone, using UTC",
			"timezone", userTimezone,
			"user_id", userID,
			"error", err)
		userTimezone = "UTC"
		timezoneLoc = time.UTC
	}

	// Create user ID getter
	userIDGetter := func(ctx context.Context) int32 {
		return userID
	}

	// Create actual tool instances
	queryTool := localtools.NewScheduleQueryTool(scheduleSvc, userIDGetter)
	addTool := localtools.NewScheduleAddTool(scheduleSvc, userIDGetter)
	updateTool := localtools.NewScheduleUpdateTool(scheduleSvc, userIDGetter)
	findFreeTimeTool := localtools.NewFindFreeTimeTool(scheduleSvc, userIDGetter)
	findFreeTimeTool.SetTimezone(userTimezone)

	// Convert to ToolWithSchema using adapter
	tools := []ToolWithSchema{
		wrapToolWithName("schedule_query", queryTool),
		wrapToolWithName("schedule_add", addTool),
		wrapToolWithName("find_free_time", findFreeTimeTool),
		wrapToolWithName("schedule_update", updateTool),
	}

	// Build system prompt
	systemPrompt := buildSystemPromptV2(timezoneLoc)

	// Create the agent
	agent := NewAgent(llm, AgentConfig{
		Name:          "schedule",
		SystemPrompt:  systemPrompt,
		MaxIterations: 10,
	}, tools)

	return &SchedulerAgentV2{
		agent:       agent,
		llm:         llm,
		scheduleSvc: scheduleSvc,
		userID:      userID,
		timezone:    userTimezone,
		timezoneLoc: timezoneLoc,
	}, nil
}

// SetIntentClassifier configures the LLM-based intent classifier.
// When set, the agent will classify user input before execution to optimize
// routing and provide better responses.
func (a *SchedulerAgentV2) SetIntentClassifier(classifier *LLMIntentClassifier) {
	a.intentClassifier = classifier
}

// wrapTool converts a tool with Run() and Description() methods to ToolWithSchema.
// It handles tools that also have InputType() for JSON Schema.
func wrapTool(tool interface{}) ToolWithSchema {
	// Try to get Run method
	var runFunc func(ctx context.Context, input string) (string, error)
	var description string
	var params map[string]interface{}

	switch t := tool.(type) {
	case interface {
		Run(ctx context.Context, input string) (string, error)
	}:
		runFunc = t.Run
	case interface {
		Call(ctx context.Context, input string) (string, error)
	}:
		runFunc = t.Call
	}

	// Get description
	if d, ok := tool.(interface{ Description() string }); ok {
		description = d.Description()
	}

	// Get input type/schema
	if i, ok := tool.(interface{ InputType() map[string]interface{} }); ok {
		params = i.InputType()
	}
	if i, ok := tool.(interface{ Parameters() map[string]interface{} }); ok {
		params = i.Parameters()
	}

	// Fallback params
	if params == nil {
		params = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// Create tool with zero name; will be set by wrapToolWithName
	return &NativeTool{
		name:        "", // Set by wrapToolWithName
		description: description,
		execute:     runFunc,
		params:      params,
	}
}

// wrapToolWithName is a helper that also sets the tool name.
func wrapToolWithName(name string, tool interface{}) ToolWithSchema {
	wrapped := wrapTool(tool)
	// Set the tool name to the provided value
	if nt, ok := wrapped.(*NativeTool); ok {
		nt.name = name
	}
	return wrapped
}

// Execute runs the agent with the given user input.
func (a *SchedulerAgentV2) Execute(ctx context.Context, userInput string) (string, error) {
	return a.ExecuteWithCallback(ctx, userInput, nil, nil)
}

// ExecuteWithCallback runs the agent with state-aware context and callback support.
func (a *SchedulerAgentV2) ExecuteWithCallback(ctx context.Context, userInput string, conversationCtx *ConversationContext, callback func(event string, data string)) (string, error) {
	// Intent classification (if classifier is configured)
	var intent TaskIntent = IntentSimpleCreate // default
	if a.intentClassifier != nil {
		classifiedIntent, err := a.intentClassifier.Classify(ctx, userInput)
		if err != nil {
			slog.Warn("intent classification failed, using default",
				"error", err,
				"input", truncateForLog(userInput, 30))
		} else {
			intent = classifiedIntent
			slog.Debug("intent classified",
				"intent", intent,
				"input", truncateForLog(userInput, 30))

			// Notify frontend about classified intent
			if callback != nil {
				callback("intent_classified", string(intent))
			}
		}
	}

	// If there's conversation context, prepend it to the input
	fullInput := userInput
	if conversationCtx != nil {
		historyPrompt := conversationCtx.ToHistoryPrompt()
		if historyPrompt != "" {
			fullInput = historyPrompt + "\nCurrent Request: " + userInput
		}
	}

	// Add intent hint to help the agent
	if intent != IntentSimpleCreate {
		fullInput = fmt.Sprintf("[意图: %s]\n%s", a.intentToHint(intent), fullInput)
	}

	// Wrap the callback to inject UI events
	uiCallback := a.wrapUICallback(callback)

	// Run the agent
	// TODO: For IntentBatchCreate, use Plan-Execute mode instead of ReAct
	return a.agent.RunWithCallback(ctx, fullInput, uiCallback)
}

// intentToHint converts intent to a hint string for the LLM.
func (a *SchedulerAgentV2) intentToHint(intent TaskIntent) string {
	switch intent {
	case IntentSimpleCreate:
		return "创建单个日程"
	case IntentSimpleQuery:
		return "查询日程或空闲时间"
	case IntentSimpleUpdate:
		return "修改或删除日程"
	case IntentBatchCreate:
		return "批量创建重复日程"
	case IntentConflictResolve:
		return "处理日程冲突"
	case IntentMultiQuery:
		return "综合查询"
	default:
		return "通用日程操作"
	}
}

// wrapUICallback wraps the original callback to inject UI events based on tool usage.
// This enables generative UI by emitting structured UI events when tools are called.
func (a *SchedulerAgentV2) wrapUICallback(originalCallback func(event string, data string)) func(event string, data string) {
	var pendingSchedule *UIScheduleSuggestionData

	return func(event string, data string) {
		if originalCallback != nil {
			originalCallback(event, data)
		}

		if event == "tool_use" && strings.HasPrefix(data, "schedule_add:") {
			if scheduleData := a.parseScheduleAddInput(data); scheduleData != nil {
				pendingSchedule = scheduleData
				a.emitUIEvent(originalCallback, EventTypeUIScheduleSuggestion, scheduleData)
			}
		}

		if event == "tool_result" && pendingSchedule != nil {
			if a.isConflictResult(data) {
				conflictData := a.buildConflictResolutionData(pendingSchedule)
				if conflictData != nil {
					a.emitUIEvent(originalCallback, EventTypeUIConflictResolution, conflictData)
				}
			}
			pendingSchedule = nil
		}
	}
}

// parseScheduleAddInput parses the schedule_add tool input to extract schedule data.
func (a *SchedulerAgentV2) parseScheduleAddInput(toolData string) *UIScheduleSuggestionData {
	// Format: "schedule_add:{JSON}"
	if !strings.HasPrefix(toolData, "schedule_add:") {
		return nil
	}

	jsonPart := strings.TrimPrefix(toolData, "schedule_add:")

	var input struct {
		Title       string `json:"title"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time,omitempty"`
		Description string `json:"description,omitempty"`
		Location    string `json:"location,omitempty"`
		AllDay      bool   `json:"all_day,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonPart), &input); err != nil {
		slog.Debug("failed to parse schedule_add input", "error", err)
		return nil
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		slog.Debug("failed to parse start_time", "error", err)
		return nil
	}

	var endTs int64
	if input.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, input.EndTime)
		if err != nil {
			// Default to 1 hour
			endTs = startTime.Unix() + 3600
		} else {
			endTs = endTime.Unix()
		}
	} else {
		endTs = startTime.Unix() + 3600
	}

	return &UIScheduleSuggestionData{
		Title:       input.Title,
		StartTs:     startTime.Unix(),
		EndTs:       endTs,
		Location:    input.Location,
		Description: input.Description,
		AllDay:      input.AllDay,
		Confidence:  0.9,
		Reason:      "根据您的输入解析",
	}
}

// isConflictResult checks if a tool result indicates a schedule conflict.
func (a *SchedulerAgentV2) isConflictResult(result string) bool {
	lowerResult := strings.ToLower(result)
	return strings.Contains(lowerResult, "conflict") ||
		strings.Contains(lowerResult, "冲突") ||
		strings.Contains(lowerResult, "occupied") ||
		strings.Contains(lowerResult, "已占用")
}

// buildConflictResolutionData builds conflict resolution UI data.
func (a *SchedulerAgentV2) buildConflictResolutionData(pending *UIScheduleSuggestionData) *UIConflictResolutionData {
	return &UIConflictResolutionData{
		NewSchedule:    *pending,
		Actions:        []string{"reschedule", "override", "cancel"},
		SuggestedSlots: []UITimeSlotData{},
	}
}

// emitUIEvent emits a UI event by marshaling the data and calling the callback.
func (a *SchedulerAgentV2) emitUIEvent(callback func(event string, data string), eventType string, data interface{}) {
	if callback == nil {
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Warn("failed to marshal UI event data",
			"event_type", eventType,
			"error", err)
		return
	}

	// Emit the UI event
	callback(eventType, string(jsonData))

	slog.Debug("emitted UI event",
		"event_type", eventType,
		"data", truncateString(string(jsonData), 200))
}

// buildSystemPromptV2 builds the system prompt for the schedule agent.
func buildSystemPromptV2(timezoneLoc *time.Location) string {
	nowLocal := time.Now().In(timezoneLoc)
	tzOffset := nowLocal.Format("-07:00")

	return fmt.Sprintf(`你是日程助手 🦜 金刚 (Macaw)。
当前系统时间: %s (%s)

## 核心原则
1. **先查后建**: 创建日程前必须先用 schedule_query 检查该时段是否有冲突
2. **冲突必处理**: 发现冲突时必须调用 find_free_time 查找可用时间
3. **默认1小时**: 用户未指定时长时，默认为1小时
4. **时间推断**: 若时间在当前之前，默认视为明天

## 工具调用最佳实践
根据任务类型选择最优调用链：

### 简单创建 (如"明天3点开会")
1. schedule_query → 检查冲突
2. schedule_add → 创建日程
⚡ 共2步，最高效

### 有冲突时
1. schedule_query → 发现冲突
2. find_free_time → 查找空闲时间
3. schedule_add → 创建日程
⚡ 共3步

### 修改日程 (如"把会议改到4点")
1. schedule_query → 找到目标日程
2. schedule_update → 更新时间

### 查询日程 (如"今天有什么安排")
1. schedule_query → 直接返回结果
⚡ 仅1步

## 响应格式
- 创建成功后，回复格式: "✓ 已创建: [标题] ([时间])"
- 更新成功后，回复格式: "✓ 已更新: [标题] ([新时间])"
- 如有冲突，先说明冲突，再给出建议时间

## 注意事项
- 使用 ISO8601 格式传递时间参数 (如 2026-01-27T15:00:00%s)
- 所有日期时间都应基于用户时区 (%s)
- 尽可能简洁回答，避免冗余说明

尽可能使用中文回答。`,
		nowLocal.Format("2006-01-02 15:04"),
		tzOffset,
		tzOffset,
		timezoneLoc.String(),
	)
}
