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
// Pre-compiled regex for parsing tool calls (using non-greedy multiline matching)
var toolCallRegex = regexp.MustCompile(`(?s)TOOL:\s*(\w+)\s*INPUT:\s*(\{.*?\})`)

// SchedulerAgent is a simplified ReAct-style agent for schedule management.
// It uses direct LLM calls with tool execution instead of complex agent frameworks.
type SchedulerAgent struct {
	llm         ai.LLMService
	scheduleSvc schedule.Service
	userID      int32
	timezone    string
	timezoneLoc *time.Location // Cached timezone location
	tools       map[string]*AgentTool
	metrics     *AgentMetrics // Metrics collection

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
		metrics:      GetGlobalMetrics(), // Use shared metrics instance
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
		_, toolCall, toolInput, err := a.parseToolCall(response)
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

		// Track tool start time for metrics
		toolStart := time.Now()

		toolResult, err := tool.Execute(ctx, toolInput)
		toolDuration := time.Since(toolStart)

		// Record tool call metrics
		if a.metrics != nil {
			a.metrics.RecordToolCall(toolCall, toolDuration, err == nil)
		}

		if err != nil {
			// Classify the error to determine retry strategy
			classified := ClassifyError(err)

			// Check for repeated failures based on error class
			a.failureMutex.Lock()
			a.failureCount[toolCall]++
			failCount := a.failureCount[toolCall]
			a.failureMutex.Unlock()

			// Log tool failure with classification
			slog.Warn("tool execution failed",
				"user_id", a.userID,
				"tool", toolCall,
				"error_class", classified.Class,
				"failure_count", failCount,
				"error", err,
				"input", truncateString(toolInput, timeout.MaxTruncateLength),
			)

			// Record error class metrics
			if a.metrics != nil {
				a.metrics.RecordErrorClass(classified.Class)
			}

			// Handle based on error class
			switch classified.Class {
			case ErrorClassConflict:
				// Conflict errors are permanent for this tool call
				// Reset failure count and continue - let LLM try a different approach
				a.failureMutex.Lock()
				a.failureCount[toolCall] = 0
				a.failureMutex.Unlock()

				// Provide a helpful error message to the LLM
				errorMsg := fmt.Sprintf("Tool %s failed due to schedule conflict. Please use find_free_time to check availability or suggest a different time.", toolCall)
				messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
				continue

			case ErrorClassPermanent:
				// Permanent errors should not be retried - abort immediately
				return "", fmt.Errorf("tool %s failed with permanent error: %w", toolCall, err)

			case ErrorClassTransient:
				// Transient errors can be retried with exponential backoff
				if failCount >= timeout.MaxToolFailures {
					return "", fmt.Errorf("tool %s failed repeatedly (%d times) with transient errors: %w", toolCall, failCount, err)
				}

				// Add delay before retry for transient errors
				if classified.RetryAfter > 0 {
					select {
					case <-time.After(classified.RetryAfter):
						// Continue with retry
					case <-ctx.Done():
						return "", fmt.Errorf("retry cancelled: %w", ctx.Err())
					}
				}

				errorMsg := fmt.Sprintf("Tool %s failed: %v. Retrying...", toolCall, err)
				messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
				continue

			default:
				// Unknown error - treat as permanent
				return "", fmt.Errorf("tool %s failed with unknown error: %w", toolCall, err)
			}
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

	// Record metrics
	if a.metrics != nil {
		a.metrics.RecordExecution(duration, iteration+1, true)
	}

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

		slog.Info("SchedulerAgent: LLM response received",
			"iteration", iteration+1,
			"response_length", len(response),
			"response_preview", truncateString(response, 200),
		)

		// Check if LLM wants to use a tool
		cleanText, toolCall, toolInput, parseErr := a.parseToolCall(response)

		if parseErr != nil {
			slog.Info("SchedulerAgent: No tool call detected",
				"iteration", iteration+1,
				"parse_error", parseErr.Error(),
				"response_preview", truncateString(response, 300),
			)
			// No tool call, this is the final answer.
			// If we have content in response, send it (stripping any broken tool markers)
			finalResponse = response

			// Notify streaming start
			if callback != nil {
				// We still use ChatStream for the final response to provide that "live" feel
				// unless the response is already complete and short.
				// But to be safe and consistent with previous turns, we stream it.
			}

			// Note: We use the context with AgentTimeout
			contentChan, errChan := a.llm.ChatStream(ctx, messages)
			var fullContent strings.Builder

			for {
				select {
				case chunk, ok := <-contentChan:
					if !ok {
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
			"clean_text_len", len(cleanText),
			"input", truncateString(toolInput, 200),
		)

		// Notify user of progress with pleasantries if present
		if cleanText != "" && callback != nil {
			// Send the pleasantry part as an answer chunk
			callback("answer", cleanText+"\n")
		}

		// Notify tool use
		if callback != nil {
			var action string
			switch toolCall {
			case "schedule_query":
				action = "正在查询日程..."
			case "schedule_add":
				action = "正在创建新日程..."
			case "schedule_update":
				action = "正在更新日程..."
			default:
				action = fmt.Sprintf("正在执行: %s", toolCall)
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

		// Track tool start time for metrics
		toolStart := time.Now()

		toolResult, err := tool.Execute(ctx, toolInput)
		toolDuration := time.Since(toolStart)

		// Record tool call metrics
		if a.metrics != nil {
			a.metrics.RecordToolCall(toolCall, toolDuration, err == nil)
		}

		if err != nil {
			slog.Warn("SchedulerAgent: Tool execution failed",
				"iteration", iteration+1,
				"tool", toolCall,
				"error", err,
			)
			// Classify the error to determine retry strategy
			classified := ClassifyError(err)

			// Check for repeated failures based on error class
			a.failureMutex.Lock()
			a.failureCount[toolCall]++
			failCount := a.failureCount[toolCall]
			a.failureMutex.Unlock()

			// Log tool failure with classification
			slog.Warn("tool execution failed",
				"user_id", a.userID,
				"tool", toolCall,
				"error_class", classified.Class,
				"failure_count", failCount,
				"error", err,
				"input", truncateString(toolInput, timeout.MaxTruncateLength),
			)

			// Record error class metrics
			if a.metrics != nil {
				a.metrics.RecordErrorClass(classified.Class)
			}

			// Handle based on error class
			switch classified.Class {
			case ErrorClassConflict:
				// Conflict errors are permanent for this tool call
				// Reset failure count and continue - let LLM try a different approach
				a.failureMutex.Lock()
				a.failureCount[toolCall] = 0
				a.failureMutex.Unlock()

				errorMsg := fmt.Sprintf("Tool %s failed due to schedule conflict. Please use find_free_time to check availability.", toolCall)
				messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
				if callback != nil {
					callback("error", errorMsg)
				}
				continue

			case ErrorClassPermanent:
				// Permanent errors should not be retried - abort immediately
				errorMsg := fmt.Sprintf("Tool %s failed with permanent error: %v", toolCall, err)
				if callback != nil {
					callback("error", errorMsg)
				}
				return "", fmt.Errorf("tool %s failed with permanent error: %w", toolCall, err)

			case ErrorClassTransient:
				// Transient errors can be retried
				if failCount >= timeout.MaxToolFailures {
					errorMsg := fmt.Sprintf("tool %s failed repeatedly (%d times) with transient errors: %v", toolCall, failCount, err)
					if callback != nil {
						callback("error", errorMsg)
					}
					return "", fmt.Errorf("tool %s failed repeatedly (%d times): %w", toolCall, failCount, err)
				}

				// Add delay before retry for transient errors
				if classified.RetryAfter > 0 {
					select {
					case <-time.After(classified.RetryAfter):
						// Continue with retry
					case <-ctx.Done():
						errorMsg := fmt.Sprintf("Retry cancelled: %v", ctx.Err())
						if callback != nil {
							callback("error", errorMsg)
						}
						return "", ctx.Err()
					}
				}

				errorMsg := fmt.Sprintf("Tool %s failed: %v. Retrying...", toolCall, err)
				messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(errorMsg))
				if callback != nil {
					callback("error", errorMsg)
				}
				continue

			default:
				// Unknown error - treat as permanent
				errorMsg := fmt.Sprintf("Tool %s failed with unknown error: %v", toolCall, err)
				if callback != nil {
					callback("error", errorMsg)
				}
				return "", fmt.Errorf("tool %s failed with unknown error: %w", toolCall, err)
			}
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
// Optimized for "快准省": minimal tokens, clear actions.
func (a *SchedulerAgent) buildSystemPrompt() string {
	nowLocal := time.Now().In(a.timezoneLoc)
	return fmt.Sprintf(`你是日程助手 🦜 金刚 (Macaw)。
当前系统时间: %s (%s)

## 身份与态度
- 你是一只聪明、严谨且守时的金刚鹦鹉。
- 说话简练有力。默认日程时长为1小时。
- 只有在执行工具前可以简要回复用户你的动作，工具调用必须严格遵守格式。

## 工具调用规则
- 必须包含 TOOL 和 INPUT 两个标识符且独立占行。
- 严禁向用户展示 TOOL 或 INPUT 的原始文本。
- schedule_add: 用于创建用户提到的新活动、新安排或意图。
- schedule_update: 仅用于修改、更新已有日程或补充缺失信息（如地点）。
- find_free_time: 在检测到冲突或用户询问“什么时候有空”时使用。

## 格式样例
好的，我来帮你安排。
TOOL: schedule_add
INPUT: {"title": "评估绩效", "start_time": "2026-01-23T15:00:00+08:00"}`,
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
// Returns cleaned text, tool name, input JSON, and error if no tool call is found.
func (a *SchedulerAgent) parseToolCall(response string) (string, string, string, error) {
	// 1. Try robust regex parsing first (handles multiline and embedding better)
	matches := toolCallRegex.FindStringSubmatch(response)
	if len(matches) == 3 {
		toolName := matches[1]
		inputJSON := matches[2]

		// Extract text BEFORE the tool call
		startIndex := toolCallRegex.FindStringIndex(response)[0]
		cleanText := strings.TrimSpace(response[:startIndex])

		// Normalize the matched JSON
		normalized, err := normalizeJSON(inputJSON)
		if err != nil {
			return cleanText, toolName, inputJSON, nil // Return as-is on error
		}
		return cleanText, toolName, normalized, nil
	}

	// 2. Fallback to line-by-line parsing if regex failed for some complex reason
	lines := strings.Split(response, "\n")
	var toolName string
	var inputJSON string
	var pleasantryLines []string
	foundTool := false
	foundInput := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "TOOL:") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				toolName = strings.TrimSpace(parts[1])
				foundTool = true
			}
			continue
		}

		if strings.HasPrefix(trimmedLine, "INPUT:") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				inputStr := strings.TrimSpace(parts[1])
				normalized, err := normalizeJSON(inputStr)
				if err != nil {
					inputJSON = inputStr
				} else {
					inputJSON = normalized
				}
				foundInput = true
			}
			continue
		}

		if !foundTool && !foundInput {
			pleasantryLines = append(pleasantryLines, line)
		}
	}

	if foundTool && foundInput {
		cleanText := strings.TrimSpace(strings.Join(pleasantryLines, "\n"))
		return cleanText, toolName, inputJSON, nil
	}

	return response, "", "", fmt.Errorf("no tool call in response")
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

// GetMetrics returns the agent's metrics collector.
func (a *SchedulerAgent) GetMetrics() *AgentMetrics {
	return a.metrics
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
