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
	"github.com/usememos/memos/server/service/schedule"
)

const (
	// MaxIterations is the maximum number of reasoning cycles
	MaxIterations = 5
)

// Pre-compiled regex for parsing tool calls (using non-greedy matching)
var toolCallRegex = regexp.MustCompile(`TOOL:\s*(\w+)\s+INPUT:\s*(\{.*?\})`)

// SchedulerAgent is a simplified ReAct-style agent for schedule management.
// It uses direct LLM calls with tool execution instead of complex agent frameworks.
type SchedulerAgent struct {
	llm               ai.LLMService
	scheduleSvc       schedule.Service
	userID            int32
	timezone          string
	timezoneLoc       *time.Location // Cached timezone location
	tools             map[string]*AgentTool

	// Cache management (protected by cacheMutex)
	cacheMutex         sync.RWMutex
	cachedSystemPrompt string   // Cached system prompt with current time
	cachedPromptTime   time.Time // When the cached prompt was generated
	cachedFullPrompt   string   // Cached full prompt (system + tools)

	// Performance monitoring
	cacheHits   int64 // Cache hit counter (atomic)
	cacheMisses int64 // Cache miss counter (atomic)

	// Tool failure tracking
	failureCount map[string]int // Tool failure counts
	failureMutex  sync.Mutex    // Protects failureCount map
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
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
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

	for iteration = 0; iteration < MaxIterations; iteration++ {
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

			// If tool fails 3+ times in a row, abort to avoid wasting resources
			if failCount >= 3 {
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

	if iteration >= MaxIterations {
		return "", fmt.Errorf("agent exceeded maximum iterations (%d), possible infinite loop", MaxIterations)
	}

	// Log execution metrics with cache statistics
	duration := time.Since(startTime)
	cacheHits := atomic.LoadInt64(&a.cacheHits)
	cacheMisses := atomic.LoadInt64(&a.cacheMisses)
	totalCacheOps := cacheHits + cacheMisses
	cacheHitRate := float64(0)
	if totalCacheOps > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCacheOps) * 100
	}

	slog.Info("agent execution completed",
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
func (a *SchedulerAgent) ExecuteWithCallback(ctx context.Context, userInput string, callback func(event string, data string)) (string, error) {
	if strings.TrimSpace(userInput) == "" {
		return "", fmt.Errorf("user input cannot be empty")
	}

	// Add timeout protection for the entire agent execution
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	startTime := time.Now()

	// Get full system prompt (cached with thread-safe refresh)
	fullPrompt := a.getFullSystemPrompt()

	messages := []ai.Message{
		ai.SystemPrompt(fullPrompt),
		ai.UserMessage(userInput),
	}

	var iteration int
	var finalResponse string

	for iteration = 0; iteration < MaxIterations; iteration++ {
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
			if callback != nil {
				callback("answer", finalResponse)
			}
			break
		}

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

		toolResult, err := tool.Execute(ctx, toolInput)
		if err != nil {
			// Check for repeated failures
			a.failureMutex.Lock()
			a.failureCount[toolCall]++
			failCount := a.failureCount[toolCall]
			a.failureMutex.Unlock()

			// If tool fails 3+ times in a row, abort to avoid wasting resources
			if failCount >= 3 {
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

		// Notify tool result
		if callback != nil {
			callback("tool_result", toolResult)
		}

		// Add tool result to conversation
		messages = append(messages, ai.AssistantMessage(response), ai.UserMessage(fmt.Sprintf("Tool result: %s", toolResult)))
	}

	if iteration >= MaxIterations {
		return "", fmt.Errorf("agent exceeded maximum iterations (%d), possible infinite loop", MaxIterations)
	}

	// Log execution metrics with cache statistics
	duration := time.Since(startTime)
	cacheHits := atomic.LoadInt64(&a.cacheHits)
	cacheMisses := atomic.LoadInt64(&a.cacheMisses)
	totalCacheOps := cacheHits + cacheMisses
	cacheHitRate := float64(0)
	if totalCacheOps > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCacheOps) * 100
	}

	slog.Info("agent execution completed",
		"user_id", a.userID,
		"iterations", iteration+1,
		"duration_ms", duration.Milliseconds(),
		"had_callback", callback != nil,
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
	nowUTC := now.UTC()

	// Use cached timezone location (validated in NewSchedulerAgent)
	nowLocal := now.In(a.timezoneLoc)

	return fmt.Sprintf(`You are an efficient schedule assistant for Memos.

Current Time Context:
- UTC: %s
- Local: %s (%s)
- Weekday: %s

## Your Role (DIRECT & EFFICIENT)
Help users manage schedules QUICKLY with MINIMAL back-and-forth.
Supports creating, updating, and finding schedules.

## CRITICAL RULES (FinOps-Optimized):

1. **BE DIRECT, NOT INQUISITIVE**:
   - Extract date/time and title from user input
   - Default duration: 1 hour (NEVER ask "how long")
   - If time is unclear, make a REASONABLE assumption (prefer evening over ambiguous times)
   - DO NOT ask clarifying questions unless absolutely necessary

2. **UNDERSTAND USER INTENT**:
   - "创建日程"/"新建日程"/"安排" → Use schedule_add
   - "更新日程"/"修改日程"/"改" → Use schedule_update
   - "查询日程"/"查看日程"/"有没有空" → Use schedule_query or find_free_time

3. **AUTO-RESOLVE CONFLICTS**:
   - Check for conflicts using schedule_query tool
   - If conflict exists, use find_free_time tool to find the nearest available slot
   - Create the schedule at the free time automatically
   - Inform user: "Found a conflict, scheduled you at [new time] instead"

4. **DEFAULT VALUES (NEVER ASK USER)**:
   - Duration: 1 hour (3600 seconds) - ALWAYS assume this unless explicitly specified
   - All-day: false (unless explicitly requested)
   - Timezone: user's timezone (automatically detected)

5. **WORKFLOW - CREATE**:
   a. Parse user input to extract: date, time, title
   b. Check for conflicts at requested time
   c. If no conflict: create schedule directly with 1-hour duration
   d. If conflict: find free time and create there
   e. Return: "Successfully created: [title] at [time]"

6. **WORKFLOW - UPDATE**:
   a. Parse user input: "update/modify [date] [new time or title]"
   b. If ID provided: update directly by ID
   c. If date provided: find schedule(s) on that date
     - 1 schedule: update it automatically
     - Multiple schedules: list them and ask for ID
     - No schedule: inform user and suggest creating
   d. Keep original duration if not specified
   e. Return: "Successfully updated: [title] at [new time]"

7. **EXAMPLES**:
   Create:
   - Input: "明天下午3点开会" → Create: "明天15:00-16:00开会" (默认1小时)
   - Input: "后天21点买鲜花" → Create: "后天21:00-22:00买鲜花" (默认1小时)
   - Input: "周三开会" → Create: "下周三15:00-16:00开会" (默认1小时，15:00为合理时间)

   Update:
   - Input: "把明天的会议改到下午4点" → Update: 明天15:00的会议改到16:00-17:00
   - Input: "更新日程：明天下午3点开会" → Find tomorrow's schedule, update to 15:00-16:00
   - Input: "后天21点买鲜花" → Create: "后天21:00-22:00买鲜花"

8. **AVOID (NEVER DO THESE)**:
   - ❌ Don't ask: "需要多久？" → Use 1 hour default
   - ❌ Don't ask: "是下午还是晚上？" → Assume evening if ambiguous
   - ❌ Don't ask: "确定吗？" → Just execute
   - ❌ Don't ask: "哪个会议？" → If only one on that day, update it

Tool Usage Format:
TOOL: tool_name
INPUT: {"key": "value"}

Available Tools:
- schedule_query: Check for existing schedules
- schedule_add: Create new schedule (1 hour default duration)
- schedule_update: Update existing schedule (by ID or date)
- find_free_time: Find available 1-hour slots (8 AM - 10 PM inclusive)

BE CONCISE: Your goal is to MANAGE schedules, not have conversations.`,
		nowUTC.Format("2006-01-02T15:04:05Z"),
		nowLocal.Format("2006-01-02T15:04:05"),
		a.timezone,
		nowLocal.Weekday().String(),
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
