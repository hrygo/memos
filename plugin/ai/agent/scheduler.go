package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/plugin/ai/agent/tools"
	"github.com/usememos/memos/plugin/ai/timeout"
	"github.com/usememos/memos/server/service/schedule"
)

// Pre-compiled regex for parsing tool calls (using non-greedy matching)
var toolCallRegex = regexp.MustCompile(`TOOL:\s*(\w+)\s+INPUT:\s*(\{.*?\})`)

// SchedulerAgent is a simplified ReAct-style agent for schedule management.
// It uses direct LLM calls with tool execution instead of complex agent frameworks.
type SchedulerAgent struct {
	llm         ai.LLMService
	scheduleSvc schedule.Service
	userID      int32
	timezone    string
	timezoneLoc *time.Location // Cached timezone location
	tools       map[string]*AgentTool

	// Cache management (protected by cacheMutex)
	cacheMutex         sync.RWMutex
	cachedSystemPrompt string    // Cached system prompt with current time
	cachedPromptTime   time.Time // When the cached prompt was generated
	cachedFullPrompt   string    // Cached full prompt (system + tools)

	// Performance monitoring
	cacheHits   int64 // Cache hit counter (atomic)
	cacheMisses int64 // Cache miss counter (atomic)

	// Tool failure tracking
	failureCount map[string]int // Tool failure counts
	failureMutex sync.Mutex     // Protects failureCount map
}

// AgentTool wraps a tool with metadata.
type AgentTool struct {
	Name        string
	Description string
	Execute     func(ctx context.Context, input string) (string, error)
}

// NewSchedulerAgent creates a new schedule agent.
func NewSchedulerAgent(llm ai.LLMService, scheduleSvc schedule.Service, userID int32, userTimezone string) (*SchedulerAgent, error) {
	// Validate inputs
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
		userTimezone = schedule.DefaultTimezone
	}

	// Validate timezone by attempting to load it
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

	// Initialize tools
	queryTool := tools.NewScheduleQueryTool(scheduleSvc, userIDGetter)
	addTool := tools.NewScheduleAddTool(scheduleSvc, userIDGetter)
	updateTool := tools.NewScheduleUpdateTool(scheduleSvc, userIDGetter)
	findFreeTimeTool := tools.NewFindFreeTimeTool(scheduleSvc, userIDGetter)
	findFreeTimeTool.SetTimezone(userTimezone) // Set user timezone

	toolMap := map[string]*AgentTool{
		"schedule_query": {
			Name:        "schedule_query",
			Description: queryTool.Description(),
			Execute: func(ctx context.Context, input string) (string, error) {
				return queryTool.Run(ctx, input)
			},
		},
		"schedule_add": {
			Name:        "schedule_add",
			Description: addTool.Description(),
			Execute: func(ctx context.Context, input string) (string, error) {
				return addTool.Run(ctx, input)
			},
		},
		"schedule_update": {
			Name:        "schedule_update",
			Description: updateTool.Description(),
			Execute: func(ctx context.Context, input string) (string, error) {
				return updateTool.Run(ctx, input)
			},
		},
		"find_free_time": {
			Name:        "find_free_time",
			Description: findFreeTimeTool.Description(),
			Execute: func(ctx context.Context, input string) (string, error) {
				return findFreeTimeTool.Run(ctx, input)
			},
		},
	}

	agent := &SchedulerAgent{
		llm:          llm,
		scheduleSvc:  scheduleSvc,
		userID:       userID,
		timezone:     userTimezone,
		timezoneLoc:  timezoneLoc,
		tools:        toolMap,
		failureCount: make(map[string]int),
	}

	// Initialize caches (build initial full prompt)
	agent.cacheMutex.Lock()
	agent.cachedSystemPrompt = agent.buildSystemPrompt()
	agent.cachedPromptTime = time.Now()
	toolsDesc := agent.buildToolsDescription()
	agent.cachedFullPrompt = agent.cachedSystemPrompt + "\n\nAvailable tools:\n" + toolsDesc
	agent.cacheMutex.Unlock()

	return agent, nil
}

// Execute runs the agent with the given user input.
func (a *SchedulerAgent) Execute(ctx context.Context, userInput string) (string, error) {
	if strings.TrimSpace(userInput) == "" {
		return "", fmt.Errorf("user input cannot be empty")
	}

	// Add timeout protection for the entire agent execution
	ctx, cancel := context.WithTimeout(ctx, timeout.AgentTimeout)
	defer cancel()

	startTime := time.Now()

	// Get full system prompt (cached with thread-safe refresh)
	fullPrompt := a.getFullSystemPrompt()

	// ReAct loop
	messages := []ai.Message{
		ai.SystemPrompt(fullPrompt),
		ai.UserMessage(userInput),
	}

	var iteration int
	var finalResponse string

	// Track recent tool calls to detect loops (tool name + input hash)
	type toolCallKey struct {
		name      string
		inputHash string
	}
	recentToolCalls := make([]toolCallKey, 0, 5)

	for iteration = 0; iteration < timeout.MaxIterations; iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("agent execution cancelled: %w", ctx.Err())
		default:
		}

		// Get LLM response
		response, err := a.llm.Chat(ctx, messages)
		if err != nil {
			return "", fmt.Errorf("LLM chat failed (iteration %d): %w", iteration+1, err)
		}

		// Check if LLM wants to use a tool
		toolCall, toolInput, err := a.parseToolCall(response)
		if err != nil {
			// No tool call, this is the final answer
			finalResponse = response
			break
		}

		// Detect loops: check if we've seen this exact tool call before
		callKey := toolCallKey{name: toolCall, inputHash: toolInput}
		repeatedCount := 0
		for _, prevCall := range recentToolCalls {
			if prevCall == callKey {
				repeatedCount++
			}
		}
		if repeatedCount > 0 {
			slog.Warn("detected repeated tool call, forcing completion",
				"user_id", a.userID,
				"tool", toolCall,
				"repeat_count", repeatedCount,
				"iteration", iteration+1,
			)
			// Return a synthesized response instead of continuing the loop
			finalResponse = fmt.Sprintf("I've completed your request. The %s tool has been executed multiple times, which suggests the operation was successful.", toolCall)
			break
		}

		// Track this tool call (keep last timeout.MaxRecentToolCalls)
		recentToolCalls = append(recentToolCalls, callKey)
		if len(recentToolCalls) > timeout.MaxRecentToolCalls {
			recentToolCalls = recentToolCalls[1:]
		}

		// Execute tool
		tool, ok := a.tools[toolCall]
		if !ok {
			errorMsg := fmt.Sprintf("Unknown tool: %s. Available tools: %s", toolCall, a.getToolNames())
			messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
			continue
		}

		toolResult, err := tool.Execute(ctx, toolInput)
		if err != nil {
			// Check for repeated failures
			a.failureMutex.Lock()
			a.failureCount[toolCall]++
			failCount := a.failureCount[toolCall]
			a.failureMutex.Unlock()

			// Log tool failure for monitoring
			slog.Warn("tool execution failed",
				"user_id", a.userID,
				"tool", toolCall,
				"failure_count", failCount,
				"error", err,
				"input", truncateString(toolInput, timeout.MaxTruncateLength),
			)

			// If tool fails 3+ times in a row, abort to avoid wasting resources
			if failCount >= timeout.MaxToolFailures {
				return "", fmt.Errorf("tool %s failed repeatedly (%d times): %w", toolCall, failCount, err)
			}

			errorMsg := fmt.Sprintf("Tool %s failed: %v", toolCall, err)
			messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
			continue
		}

		// Reset failure count on success
		a.failureMutex.Lock()
		a.failureCount[toolCall] = 0
		a.failureMutex.Unlock()

		// Add tool result to conversation
		messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(fmt.Sprintf("Tool result: %s", toolResult)))
	}

	if iteration >= timeout.MaxIterations {
		return "", fmt.Errorf("agent exceeded maximum iterations (%d), possible infinite loop", timeout.MaxIterations)
	}

	// Log execution metrics
	duration := time.Since(startTime)
	cacheHits := atomic.LoadInt64(&a.cacheHits)
	cacheMisses := atomic.LoadInt64(&a.cacheMisses)
	cacheHitRate := float64(cacheHits) / float64(cacheHits+cacheMisses+1) * 100

	slog.Debug("agent execution completed",
		"user_id", a.userID,
		"iterations", iteration+1,
		"duration_ms", duration.Milliseconds(),
		"cache_hits", cacheHits,
		"cache_misses", cacheMisses,
		"cache_hit_rate", fmt.Sprintf("%.2f%%", cacheHitRate),
	)

	return finalResponse, nil
}

// ExecuteWithCallback runs the agent with callback support for real-time feedback.
func (a *SchedulerAgent) ExecuteWithCallback(ctx context.Context, userInput string, history []string, callback func(event string, data string)) (string, error) {
	if strings.TrimSpace(userInput) == "" {
		return "", fmt.Errorf("user input cannot be empty")
	}

	// Add timeout protection for the entire agent execution
	ctx, cancel := context.WithTimeout(ctx, timeout.AgentTimeout)
	defer cancel()

	startTime := time.Now()

	// Log execution start
	truncatedInput := truncateString(userInput, 100)
	slog.Info("SchedulerAgent: ExecuteWithCallback started",
		"user_id", a.userID,
		"timezone", a.timezone,
		"input", truncatedInput,
		"history_count", len(history),
	)

	// Get full system prompt (cached with thread-safe refresh)
	fullPrompt := a.getFullSystemPrompt()

	messages := []ai.Message{
		ai.SystemPrompt(fullPrompt),
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

	// Add current user input
	messages = append(messages, ai.Message{Role: "user", Content: userInput})

	var iteration int
	var finalResponse string

	// Track recent tool calls to detect loops (tool name + input hash)
	type toolCallKey struct {
		name      string
		inputHash string
	}
	recentToolCalls := make([]toolCallKey, 0, 5)

	for iteration = 0; iteration < timeout.MaxIterations; iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			errorMsg := fmt.Sprintf("execution timeout: %v", ctx.Err())
			if callback != nil {
				callback("error", errorMsg)
			}
			return "", fmt.Errorf("agent execution cancelled: %w", ctx.Err())
		default:
		}

		// Notify thinking
		if callback != nil {
			callback("thinking", "Agent is thinking...")
		}

		// Log LLM call
		slog.Debug("SchedulerAgent: Calling LLM",
			"iteration", iteration+1,
			"messages_count", len(messages),
		)

		// Get LLM response
		response, err := a.llm.Chat(ctx, messages)
		if err != nil {
			slog.Error("SchedulerAgent: LLM chat failed",
				"iteration", iteration+1,
				"error", err,
			)
			return "", fmt.Errorf("LLM chat failed (iteration %d): %w", iteration+1, err)
		}

		slog.Debug("SchedulerAgent: LLM response received",
			"iteration", iteration+1,
			"response_length", len(response),
			"response_preview", truncateString(response, 200),
		)

		// Check if LLM wants to use a tool
		toolCall, toolInput, parseErr := a.parseToolCall(response)
		if parseErr != nil {
			// No tool call, this is the final answer.
			// Optimize: Perform final answer with streaming for better UX.
			// Note: We use the same message history including this turn's prompt.
			finalResponse = response

			// Notify streaming start
			if callback != nil {
				// Clear "thinking" or "tool_use" status if needed and start answer
				// (Frontend handles this via onContent)
			}

			// Note: We use the context with AgentTimeout
			contentChan, errChan := a.llm.ChatStream(ctx, messages)
			var fullContent strings.Builder

			for {
				select {
				case chunk, ok := <-contentChan:
					if !ok {
						// Stream closed
						if callback != nil {
							// For scheduler, we might want to send the full accumulated response
							// but here we send chunks as they come.
						}
						return fullContent.String(), nil
					}
					fullContent.WriteString(chunk)
					if callback != nil {
						callback("answer", chunk)
					}
				case err := <-errChan:
					if err != nil {
						if callback != nil {
							callback("error", fmt.Sprintf("Streaming error: %v", err))
						}
						return fullContent.String(), err
					}
				case <-ctx.Done():
					return fullContent.String(), ctx.Err()
				}
			}
		}

		// Log tool call
		slog.Info("SchedulerAgent: Tool call parsed",
			"iteration", iteration+1,
			"tool", toolCall,
			"input", truncateString(toolInput, 200),
		)

		// Notify tool use
		if callback != nil {
			var action string
			switch toolCall {
			case "schedule_query":
				action = "Querying your calendar..."
			case "schedule_add":
				action = "Creating a new schedule..."
			default:
				action = fmt.Sprintf("Using tool: %s", toolCall)
			}
			callback("tool_use", action)
		}

		// Detect loops: check if we've seen this exact tool call before executing
		callKey := toolCallKey{name: toolCall, inputHash: toolInput}
		repeatedCount := 0
		for _, prevCall := range recentToolCalls {
			if prevCall == callKey {
				repeatedCount++
			}
		}
		if repeatedCount > 0 {
			slog.Warn("detected repeated tool call in ExecuteWithCallback, forcing completion",
				"user_id", a.userID,
				"tool", toolCall,
				"repeat_count", repeatedCount,
				"iteration", iteration+1,
			)
			// Return the last tool result as the final answer instead of executing again
			// Find the last assistant message and extract tool result from it
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Role == "assistant" && strings.Contains(messages[i].Content, "Tool result:") {
					// Extract tool result more precisely
					parts := strings.SplitN(messages[i].Content, "Tool result:", 2)
					if len(parts) == 2 {
						finalResponse = strings.TrimSpace(parts[1])
						if callback != nil {
							callback("answer", finalResponse)
						}
					}
					break
				}
			}
			if finalResponse == "" {
				// Fallback: synthesized message
				finalResponse = fmt.Sprintf("I've completed your request. The %s tool was executed successfully.", toolCall)
				if callback != nil {
					callback("answer", finalResponse)
				}
			}
			break
		}

		// Track this tool call (keep last timeout.MaxRecentToolCalls)
		recentToolCalls = append(recentToolCalls, callKey)
		if len(recentToolCalls) > timeout.MaxRecentToolCalls {
			recentToolCalls = recentToolCalls[1:]
		}

		// Execute tool
		tool, ok := a.tools[toolCall]
		if !ok {
			errorMsg := fmt.Sprintf("Unknown tool: %s. Available tools: %s", toolCall, a.getToolNames())
			messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
			if callback != nil {
				callback("error", errorMsg)
			}
			continue
		}

		slog.Debug("SchedulerAgent: Executing tool",
			"iteration", iteration+1,
			"tool", toolCall,
			"input", truncateString(toolInput, 200),
		)

		toolResult, err := tool.Execute(ctx, toolInput)
		if err != nil {
			slog.Warn("SchedulerAgent: Tool execution failed",
				"iteration", iteration+1,
				"tool", toolCall,
				"error", err,
			)
			// Check for repeated failures
			a.failureMutex.Lock()
			a.failureCount[toolCall]++
			failCount := a.failureCount[toolCall]
			a.failureMutex.Unlock()

			// Log tool failure for monitoring
			slog.Warn("tool execution failed",
				"user_id", a.userID,
				"tool", toolCall,
				"failure_count", failCount,
				"error", err,
				"input", truncateString(toolInput, timeout.MaxTruncateLength),
			)

			// If tool fails 3+ times in a row, abort to avoid wasting resources
			if failCount >= timeout.MaxToolFailures {
				errorMsg := fmt.Sprintf("tool %s failed repeatedly (%d times): %v", toolCall, failCount, err)
				if callback != nil {
					callback("error", errorMsg)
				}
				return "", fmt.Errorf("tool %s failed repeatedly (%d times): %w", toolCall, failCount, err)
			}

			errorMsg := fmt.Sprintf("Tool %s failed: %v", toolCall, err)
			messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
			if callback != nil {
				callback("error", errorMsg)
			}
			continue
		}

		// Reset failure count on success
		a.failureMutex.Lock()
		a.failureCount[toolCall] = 0
		a.failureMutex.Unlock()

		slog.Info("SchedulerAgent: Tool executed successfully",
			"iteration", iteration+1,
			"tool", toolCall,
			"result_length", len(toolResult),
			"result_preview", truncateString(toolResult, 150),
		)

		// Notify tool result
		if callback != nil {
			callback("tool_result", toolResult)
		}

		// Add tool result to conversation
		messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(fmt.Sprintf("Tool result: %s", toolResult)))
	}

	if iteration >= timeout.MaxIterations {
		slog.Error("SchedulerAgent: Max iterations exceeded",
			"user_id", a.userID,
			"max_iterations", timeout.MaxIterations,
		)
		return "", fmt.Errorf("agent exceeded maximum iterations (%d), possible infinite loop", timeout.MaxIterations)
	}

	// Log execution metrics
	duration := time.Since(startTime)
	cacheHits := atomic.LoadInt64(&a.cacheHits)
	cacheMisses := atomic.LoadInt64(&a.cacheMisses)
	cacheHitRate := float64(cacheHits) / float64(cacheHits+cacheMisses+1) * 100

	slog.Info("SchedulerAgent: Execution completed",
		"user_id", a.userID,
		"iterations", iteration+1,
		"duration_ms", duration.Milliseconds(),
		"cache_hits", cacheHits,
		"cache_misses", cacheMisses,
		"cache_hit_rate", fmt.Sprintf("%.2f%%", cacheHitRate),
	)

	return finalResponse, nil
}

// getFullSystemPrompt returns the full system prompt (system prompt + tools description).
// Uses thread-safe caching with time-aware refresh (1 minute expiry).
// Implements double-checked locking for performance in high-concurrency scenarios.
func (a *SchedulerAgent) getFullSystemPrompt() string {
	// Fast path: read lock check
	a.cacheMutex.RLock()
	cached := a.cachedFullPrompt
	cachedTime := a.cachedPromptTime
	a.cacheMutex.RUnlock()

	// Check if cache is valid
	if cached != "" && time.Since(cachedTime) <= time.Minute {
		// Cache hit - increment counter atomically
		atomic.AddInt64(&a.cacheHits, 1)
		return cached
	}

	// Slow path: need to refresh cache
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	// Double-check: another goroutine might have refreshed while we waited
	if a.cachedFullPrompt != "" && time.Since(a.cachedPromptTime) <= time.Minute {
		atomic.AddInt64(&a.cacheHits, 1)
		return a.cachedFullPrompt
	}

	// Cache miss - rebuild full prompt
	atomic.AddInt64(&a.cacheMisses, 1)

	// Build system prompt with current time
	a.cachedSystemPrompt = a.buildSystemPrompt()
	a.cachedPromptTime = time.Now()

	// Build tools description (only needed on first init or if tools change)
	toolsDesc := a.buildToolsDescription()

	// Combine into full prompt
	a.cachedFullPrompt = a.cachedSystemPrompt + "\n\nAvailable tools:\n" + toolsDesc

	return a.cachedFullPrompt
}

// buildSystemPrompt creates the system prompt with current time context.
func (a *SchedulerAgent) buildSystemPrompt() string {
	now := time.Now()
	nowLocal := now.In(a.timezoneLoc)

	return fmt.Sprintf(`你是 Memos 日程助手。当前: %s (%s)

## 工具调用格式
TOOL: tool_name
INPUT: {"field": "value"}

## 可用工具
- schedule_add: 创建日程 (默认1小时)
- schedule_update: 更新日程 (按ID或日期)
- schedule_query: 查询日程
- find_free_time: 查找空闲时段 (8:00-22:00)

## 字段格式 (重要!)
- 时间: ISO8601格式，如 "2026-01-23T15:00:00+08:00"
- 字段名: 使用 snake_case (start_time, end_time, all_day)
- 时长: 默认3600秒(1小时)，end_time = start_time + 3600

## 冲突解决
当 schedule_add 返回 "schedule conflicts detected" 时:
1. 调用 find_free_time: {"date": "YYYY-MM-DD"}
2. 使用返回的空闲时间重新调用 schedule_add
3. 一次成功后立即停止，不要重复调用

## 停止条件
- 创建/更新成功后立即停止，不要再验证
- 查询结果后直接反馈给用户，不要重复查询

## 快捷指令
"明天3点开会" → schedule_add
"把明天的会议改到4点" → schedule_update
"明天有空吗" → find_free_time

目标：快速完成，减少对话轮次。`,
		nowLocal.Format("2006-01-02 15:04"),
		a.timezone,
	)
}

// buildToolsDescription builds a description of available tools.
func (a *SchedulerAgent) buildToolsDescription() string {
	// Calculate accurate capacity: sum of name + description + formatting overhead
	estimatedSize := 0
	for _, tool := range a.tools {
		estimatedSize += len(tool.Name) + len(tool.Description) + 4 // +4 for "- " + ": " + "\n"
	}
	estimatedSize += 100 // Extra buffer for safety

	var desc strings.Builder
	desc.Grow(estimatedSize)

	for _, tool := range a.tools {
		desc.WriteString("- ")
		desc.WriteString(tool.Name)
		desc.WriteString(": ")
		desc.WriteString(tool.Description)
		desc.WriteByte('\n')
	}
	return desc.String()
}

// parseToolCall attempts to parse a tool call from LLM response.
// Returns tool name, input JSON, and error if no tool call is found.
func (a *SchedulerAgent) parseToolCall(response string) (string, string, error) {
	// Try to parse tool call format: "TOOL: tool_name\nINPUT: {json}"
	lines := strings.Split(response, "\n")

	var toolName string
	var inputJSON string
	foundTool := false
	foundInput := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "TOOL:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				toolName = strings.TrimSpace(parts[1])
				foundTool = true
			}
		}

		if strings.HasPrefix(line, "INPUT:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				inputStr := strings.TrimSpace(parts[1])
				// Validate and normalize JSON
				normalized, err := normalizeJSON(inputStr)
				if err != nil {
					// If JSON is invalid, use as-is (best effort)
					inputJSON = inputStr
				} else {
					inputJSON = normalized
				}
				foundInput = true
			}
		}
	}

	if !foundTool || !foundInput {
		// Try alternative format with JSON in same line using pre-compiled regex
		matches := toolCallRegex.FindStringSubmatch(response)
		if len(matches) == 3 {
			// Normalize the matched JSON
			normalized, err := normalizeJSON(matches[2])
			if err != nil {
				return matches[1], matches[2], nil // Return as-is on error
			}
			return matches[1], normalized, nil
		}

		// No tool call found
		return "", "", fmt.Errorf("no tool call in response")
	}

	return toolName, inputJSON, nil
}

// normalizeJSON validates and normalizes a JSON string.
// It parses the JSON and re-encodes it to ensure consistent formatting.
func normalizeJSON(jsonStr string) (string, error) {
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	normalized, err := json.Marshal(jsonObj)
	if err != nil {
		return "", fmt.Errorf("failed to encode JSON: %w", err)
	}

	return string(normalized), nil
}

// getToolNames returns a comma-separated list of tool names.
func (a *SchedulerAgent) getToolNames() string {
	names := make([]string, 0, len(a.tools))
	for name := range a.tools {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// truncateString truncates a string to a maximum length for logging.
func truncateString(s string, maxLen int) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
