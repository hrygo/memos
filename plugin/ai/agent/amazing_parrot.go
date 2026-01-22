package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/plugin/ai/agent/tools"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/server/service/schedule"
)

// AmazingParrot is the comprehensive assistant parrot (🦜 惊奇).
// AmazingParrot 是综合助手鹦鹉（🦜 惊奇）。
// It combines memo and schedule capabilities for integrated assistance.
type AmazingParrot struct {
	llm              ai.LLMService
	cache            *LRUCache
	userID           int32
	memoSearchTool   *tools.MemoSearchTool
	scheduleQueryTool   *tools.ScheduleQueryTool
	scheduleAddTool     *tools.ScheduleAddTool
	findFreeTimeTool    *tools.FindFreeTimeTool
	scheduleUpdateTool  *tools.ScheduleUpdateTool
}

// NewAmazingParrot creates a new amazing parrot agent.
// NewAmazingParrot 创建一个新的综合助手鹦鹉。
func NewAmazingParrot(
	llm ai.LLMService,
	retriever *retrieval.AdaptiveRetriever,
	scheduleService schedule.Service,
	userID int32,
) (*AmazingParrot, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm cannot be nil")
	}
	if retriever == nil {
		return nil, fmt.Errorf("retriever cannot be nil")
	}
	if scheduleService == nil {
		return nil, fmt.Errorf("scheduleService cannot be nil")
	}

	// Create user ID getter
	userIDGetter := func(ctx context.Context) int32 {
		return userID
	}

	// Initialize tools
	memoSearchTool := tools.NewMemoSearchTool(retriever, userIDGetter)
	scheduleQueryTool := tools.NewScheduleQueryTool(scheduleService, userIDGetter)
	scheduleAddTool := tools.NewScheduleAddTool(scheduleService, userIDGetter)
	findFreeTimeTool := tools.NewFindFreeTimeTool(scheduleService, userIDGetter)
	scheduleUpdateTool := tools.NewScheduleUpdateTool(scheduleService, userIDGetter)

	return &AmazingParrot{
		llm:              llm,
		cache:            NewLRUCache(DefaultCacheEntries, DefaultCacheTTL),
		userID:           userID,
		memoSearchTool:   memoSearchTool,
		scheduleQueryTool: scheduleQueryTool,
		scheduleAddTool:   scheduleAddTool,
		findFreeTimeTool:  findFreeTimeTool,
		scheduleUpdateTool: scheduleUpdateTool,
	}, nil
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (p *AmazingParrot) Name() string {
	return "amazing" // ParrotAgentType AGENT_TYPE_AMAZING
}

// ExecuteWithCallback executes the amazing parrot with callback support.
// ExecuteWithCallback 执行综合助手鹦鹉并支持回调。
func (p *AmazingParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	callback EventCallback,
) error {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, AgentTimeout)
	defer cancel()

	// Step 1: Check cache
	cacheKey := p.generateCacheKey(p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			if callback != nil {
				callback(EventTypeAnswer, result)
			}
			return nil
		}
	}

	// Step 2: Build system prompt
	systemPrompt := p.buildSystemPrompt()

	// Step 3: ReAct loop
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userInput},
	}

	var iteration int

	for iteration = 0; iteration < MaxToolIterations; iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return NewParrotError(p.Name(), "ExecuteWithCallback", ctx.Err())
		default:
		}

		// Notify thinking
		if callback != nil {
			callback(EventTypeThinking, "正在思考...")
		}

		// Get LLM response
		response, err := p.llm.Chat(ctx, messages)
		if err != nil {
			return NewParrotError(p.Name(), "Chat", err)
		}

		// Try to parse tool call
		toolCall, toolInput, err := p.parseToolCall(response)
		if err != nil {
			// No tool call, this is the final answer
			p.cache.Set(cacheKey, response)

			if callback != nil {
				callback(EventTypeAnswer, response)
			}
			return nil
		}

		// Execute tool
		if callback != nil {
			callback(EventTypeToolUse, fmt.Sprintf("正在使用工具: %s", toolCall))
		}

		var toolResult string
		var toolErr error

		switch toolCall {
		case "memo_search":
			toolResult, toolErr = p.memoSearchTool.Run(ctx, toolInput)
		case "schedule_query":
			toolResult, toolErr = p.scheduleQueryTool.Run(ctx, toolInput)
		case "schedule_add":
			toolResult, toolErr = p.scheduleAddTool.Run(ctx, toolInput)
		case "find_free_time":
			toolResult, toolErr = p.findFreeTimeTool.Run(ctx, toolInput)
		case "schedule_update":
			toolResult, toolErr = p.scheduleUpdateTool.Run(ctx, toolInput)
		default:
			errorMsg := fmt.Sprintf("未知工具: %s，可用工具: memo_search, schedule_query, schedule_add, find_free_time, schedule_update", toolCall)
			messages = append(messages, ai.Message{Role: "assistant", Content: response})
			messages = append(messages, ai.Message{Role: "user", Content: errorMsg})
			continue
		}

		if toolErr != nil {
			// Tool execution failed
			errorMsg := fmt.Sprintf("工具执行失败 (%s): %v", toolCall, toolErr)
			messages = append(messages, ai.Message{Role: "assistant", Content: response})
			messages = append(messages, ai.Message{Role: "user", Content: errorMsg})
			continue
		}

		// Send tool result
		if callback != nil {
			callback(EventTypeToolResult, toolResult)
		}

		// Add to conversation
		messages = append(messages, ai.Message{Role: "assistant", Content: response})
		messages = append(messages, ai.Message{Role: "user", Content: fmt.Sprintf("工具结果: %s", toolResult)})
	}

	// Exceeded max iterations
	return NewParrotError(p.Name(), "ExecuteWithCallback",
		fmt.Errorf("exceeded maximum iterations (%d)", MaxToolIterations))
}

// buildSystemPrompt builds the system prompt for the amazing parrot.
func (p *AmazingParrot) buildSystemPrompt() string {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	todayEnd := todayStart.Add(24 * time.Hour)
	tomorrowSame := todayStart.Add(24 * time.Hour)

	return fmt.Sprintf(`你是 Memos 的综合助手 🦜 惊奇，专注于帮助用户管理笔记和日程。

当前时间: %s

## 核心能力
1. **笔记管理**: 使用 memo_search 工具搜索笔记
2. **日程查询**: 使用 schedule_query 工具查询日程安排
3. **日程创建**: 使用 schedule_add 工具创建新日程
4. **空闲时间**: 使用 find_free_time 工具查找可用时间
5. **日程更新**: 使用 schedule_update 工具修改已有日程

## 工作流程 (ReAct 模式)
1. **思考**: 分析用户需求，确定需要使用哪些工具
2. **工具**: 按需调用一个或多个工具
3. **观察**: 分析工具结果
4. **回答**: 基于工具结果生成综合回答

## 工具使用规范

### memo_search - 笔记搜索
用途: 搜索笔记内容
输入格式: JSON
- query (必需): 搜索关键词
- limit (可选): 返回结果数量，默认 10，最大 50
- min_score (可选): 最小相关度分数，默认 0.5

示例:
- 搜索编程笔记: {"query": "编程", "limit": 10}
- 搜索重要内容: {"query": "重要", "min_score": 0.7}

### schedule_query - 日程查询
用途: 查询指定时间范围内的日程
输入格式: JSON
- start_time (必需): ISO8601 格式开始时间
- end_time (必需): ISO8601 格式结束时间

当前时间示例:
- 今天开始: %s
- 今天结束: %s
- 明天此时: %s

### schedule_add - 创建日程
用途: 创建新的日程事件
输入格式: JSON
- title (必需): 日程标题
- start_time (必需): ISO8601 格式开始时间
- end_time (可选): ISO8601 格式结束时间
- location (可选): 地点
- description (可选): 描述
- all_day (可选): 是否全天事件，默认 false

示例:
- 创建会议: {"title": "团队会议", "start_time": "2026-01-24T09:00:00Z", "location": "会议室A"}

### find_free_time - 查找空闲时间
用途: 查找指定日期的可用 1 小时时间段（8:00-22:00）
输入格式: JSON
- date (必需): 日期，格式 YYYY-MM-DD

示例:
- 查找明天空闲: {"date": "2026-01-24"}

### schedule_update - 更新日程
用途: 更新已有日程
输入格式: JSON
- id (可选): 日程 ID（如果不提供则用 date 查找）
- date (可选): 日期用于查找日程
- title (可选): 新标题
- start_time (可选): 新开始时间
- end_time (可选): 新结束时间
- location (可选): 地点
- description (可选): 描述

示例:
- 通过日期更新: {"date": "2026-01-24", "title": "新标题"}

## 回答原则
1. **准确优先**: 仅基于工具结果回答，不编造信息
2. **结构清晰**: 使用列表、分段组织内容
3. **简洁明了**: 直接给出答案，避免冗余
4. **综合分析**: 当涉及笔记和日程时，综合给出建议

## 示例对话

用户: "帮我找关于 Python 的笔记，并查看今天有没有时间学习"
思考: 需要搜索 Python 笔记，并查询今天日程
工具1: {"query": "Python", "limit": 5}
观察1: 找到 5 条 Python 笔记
工具2: {"start_time": "2026-01-24T00:00:00Z", "end_time": "2026-01-25T00:00:00Z"}
观察2: 今天有 3 个日程
回答: 为您找到 5 条 Python 笔记... 今天日程较满，建议晚上 8 点后学习...

用户: "明天下午有什么安排？"
思考: 查询明天的日程
工具: {"start_time": "2026-01-24T00:00:00Z", "end_time": "2026-01-25T00:00:00Z"}
回答: 明天下午有以下安排...

## 重要提醒
- 使用工具前确保输入参数格式正确
- ISO8601 时间格式: 2026-01-24T09:00:00Z
- 综合笔记和日程信息时，给出有价值的建议
- 如果找不到相关信息，明确告知用户

工具调用格式:
TOOL: <工具名>
INPUT: <JSON输入>`,
		now.Format("2006-01-02 15:04:05"),
		todayStart.Format(time.RFC3339),
		todayEnd.Format(time.RFC3339),
		tomorrowSame.Format(time.RFC3339),
	)
}

// parseToolCall attempts to parse a tool call from LLM response.
func (p *AmazingParrot) parseToolCall(response string) (string, string, error) {
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
				// Validate JSON
				var jsonObj map[string]any
				if err := json.Unmarshal([]byte(inputStr), &jsonObj); err != nil {
					return "", "", fmt.Errorf("invalid JSON in INPUT: %w", err)
				}
				inputJSON = inputStr
				foundInput = true
			}
		}
	}

	if !foundTool || !foundInput {
		return "", "", fmt.Errorf("no tool call in response")
	}

	return toolName, inputJSON, nil
}

// GetStats returns the cache statistics.
func (p *AmazingParrot) GetStats() CacheStats {
	return p.cache.Stats()
}

// generateCacheKey creates a cache key from userID and userInput using SHA256 hash.
func (p *AmazingParrot) generateCacheKey(userID int32, userInput string) string {
	hash := sha256.Sum256([]byte(userInput))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("amazing:%d:%s", userID, hashStr[:16])
}
