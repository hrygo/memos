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
	agent       *Agent
	llm         ai.LLMService
	scheduleSvc schedule.Service
	userID      int32
	timezone    string
	timezoneLoc *time.Location
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

// toolWithWrapper is a helper that implements ToolWithSchema for existing tools.
type toolWithWrapper struct {
	name        string
	description string
	runFunc     func(ctx context.Context, input string) (string, error)
	params      map[string]interface{}
}

func (t *toolWithWrapper) Name() string        { return t.name }
func (t *toolWithWrapper) Description() string { return t.description }
func (t *toolWithWrapper) Parameters() map[string]interface{} {
	if t.params == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}
	return t.params
}
func (t *toolWithWrapper) Run(ctx context.Context, input string) (string, error) {
	return t.runFunc(ctx, input)
}

// Execute runs the agent with the given user input.
func (a *SchedulerAgentV2) Execute(ctx context.Context, userInput string) (string, error) {
	return a.ExecuteWithCallback(ctx, userInput, nil, nil)
}

// ExecuteWithCallback runs the agent with state-aware context and callback support.
func (a *SchedulerAgentV2) ExecuteWithCallback(ctx context.Context, userInput string, conversationCtx *ConversationContext, callback func(event string, data string)) (string, error) {
	// If there's conversation context, prepend it to the input
	fullInput := userInput
	if conversationCtx != nil {
		historyPrompt := conversationCtx.ToHistoryPrompt()
		if historyPrompt != "" {
			fullInput = historyPrompt + "\nCurrent Request: " + userInput
		}
	}

	// Wrap the callback to inject UI events
	uiCallback := a.wrapUICallback(callback)

	// Run the agent
	return a.agent.RunWithCallback(ctx, fullInput, uiCallback)
}

// wrapUICallback wraps the original callback to inject UI events based on tool usage.
// This enables generative UI by emitting structured UI events when tools are called.
func (a *SchedulerAgentV2) wrapUICallback(originalCallback func(event string, data string)) func(event string, data string) {
	// Track pending schedule data for UI events
	var pendingSchedule *UIScheduleSuggestionData
	var lastQueryResult *localtools.ScheduleQueryToolResult

	return func(event string, data string) {
		// Always forward to original callback
		if originalCallback != nil {
			originalCallback(event, data)
		}

		// Process tool_use events to extract schedule data
		if event == "tool_use" {
			if strings.HasPrefix(data, "schedule_add:") {
				// Extract schedule_add input and emit UI suggestion event
				scheduleData := a.parseScheduleAddInput(data)
				if scheduleData != nil {
					pendingSchedule = scheduleData
					// Emit UI schedule suggestion event
					a.emitUIEvent(originalCallback, EventTypeUIScheduleSuggestion, scheduleData)
				}
			} else if strings.HasPrefix(data, "schedule_query:") {
				// Store query result for potential conflict resolution
				queryResult := a.parseScheduleQueryResult(data)
				if queryResult != nil {
					lastQueryResult = queryResult
				}
			}
		}

		// Process tool_result events to detect conflicts
		if event == "tool_result" && pendingSchedule != nil {
			// Check if result indicates conflict
			if a.isConflictResult(data) {
				// Emit conflict resolution UI event
				conflictData := a.buildConflictResolutionData(pendingSchedule, lastQueryResult)
				if conflictData != nil {
					a.emitUIEvent(originalCallback, EventTypeUIConflictResolution, conflictData)
				}
			}
			// Clear pending schedule after processing result
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

// parseScheduleQueryResult parses schedule_query tool input.
func (a *SchedulerAgentV2) parseScheduleQueryResult(toolData string) *localtools.ScheduleQueryToolResult {
	// This would parse the query result if needed for conflict detection
	// For now, we keep it simple and return nil
	return nil
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
func (a *SchedulerAgentV2) buildConflictResolutionData(pending *UIScheduleSuggestionData, queryResult *localtools.ScheduleQueryToolResult) *UIConflictResolutionData {
	return &UIConflictResolutionData{
		NewSchedule:   *pending,
		Actions:       []string{"reschedule", "override", "cancel"},
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

核心原则:
1. 先查后建: 创建日程前建议先检查冲突。
2. 冲突必处理: 发现冲突必须查找可用时间。
3. 默认1小时: 用户未指定时长时，默认为1小时。
4. 时间推断: 若用户输入的时间在当前时间之前，默认视为明天。

任务:
使用提供的工具(Tools)来管理用户的日程。
如果需要执行操作，请直接调用相应的函数。

尽可能使用中文回答。`,
		nowLocal.Format("2006-01-02 15:04"),
		tzOffset,
	)
}
