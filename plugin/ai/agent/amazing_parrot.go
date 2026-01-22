package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/plugin/ai/agent/tools"
	"github.com/usememos/memos/plugin/ai/timeout"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/server/service/schedule"
)

// AmazingParrot is the comprehensive assistant parrot (🦜 惊奇).
// AmazingParrot 是综合助手鹦鹉（🦜 惊奇）。
// It combines memo and schedule capabilities for integrated assistance.
type AmazingParrot struct {
	llm                ai.LLMService
	cache              *LRUCache
	userID             int32
	memoSearchTool     *tools.MemoSearchTool
	scheduleQueryTool  *tools.ScheduleQueryTool
	scheduleAddTool    *tools.ScheduleAddTool
	findFreeTimeTool   *tools.FindFreeTimeTool
	scheduleUpdateTool *tools.ScheduleUpdateTool
}

// retrievalPlan represents the plan for concurrent retrieval.
type retrievalPlan struct {
	needsMemoSearch     bool
	memoSearchQuery     string
	needsScheduleQuery  bool
	scheduleStartTime   string
	scheduleEndTime     string
	needsScheduleAdd    bool
	scheduleAddData     string
	needsFreeTime       bool
	freeTimeDate        string
	needsScheduleUpdate bool
	scheduleUpdateData  string
	needsDirectAnswer   bool // If true, skip retrieval and answer directly
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
	memoSearchTool, err := tools.NewMemoSearchTool(retriever, userIDGetter)
	if err != nil {
		return nil, fmt.Errorf("failed to create memo search tool: %w", err)
	}
	scheduleQueryTool := tools.NewScheduleQueryTool(scheduleService, userIDGetter)
	scheduleAddTool := tools.NewScheduleAddTool(scheduleService, userIDGetter)
	findFreeTimeTool := tools.NewFindFreeTimeTool(scheduleService, userIDGetter)
	scheduleUpdateTool := tools.NewScheduleUpdateTool(scheduleService, userIDGetter)

	return &AmazingParrot{
		llm:                llm,
		cache:              NewLRUCache(DefaultCacheEntries, DefaultCacheTTL),
		userID:             userID,
		memoSearchTool:     memoSearchTool,
		scheduleQueryTool:  scheduleQueryTool,
		scheduleAddTool:    scheduleAddTool,
		findFreeTimeTool:   findFreeTimeTool,
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
//
// Implementation: Two-phase concurrent retrieval for optimal performance
// Phase 1: Analyze user intent and plan concurrent retrievals
// Phase 2: Execute retrievals concurrently, then synthesize answer
func (p *AmazingParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback EventCallback,
) error {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, timeout.AgentTimeout)
	defer cancel()

	startTime := time.Now()

	// Log execution start
	slog.Info("AmazingParrot: ExecuteWithCallback started",
		"user_id", p.userID,
		"input", truncateString(userInput, 100),
		"history_count", len(history),
	)

	// Step 1: Check cache
	cacheKey := GenerateCacheKey(p.Name(), p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			slog.Info("AmazingParrot: Cache hit", "user_id", p.userID)
			if callback != nil {
				callback(EventTypeAnswer, result)
			}
			return nil
		}
	}
	slog.Debug("AmazingParrot: Cache miss, proceeding with execution", "user_id", p.userID)

	// Step 2: Plan concurrent retrieval using LLM intent analysis
	slog.Debug("AmazingParrot: Starting planning phase", "user_id", p.userID)
	plan, err := p.planRetrieval(ctx, userInput, history, callback)
	if err != nil {
		slog.Error("AmazingParrot: Planning failed", "user_id", p.userID, "error", err)
		return NewParrotError(p.Name(), "planRetrieval", err)
	}
	slog.Info("AmazingParrot: Plan created",
		"user_id", p.userID,
		"needs_memo_search", plan.needsMemoSearch,
		"needs_schedule_query", plan.needsScheduleQuery,
		"needs_free_time", plan.needsFreeTime,
		"needs_schedule_add", plan.needsScheduleAdd,
		"needs_schedule_update", plan.needsScheduleUpdate,
	)

	// Step 3: Execute concurrent retrieval
	slog.Debug("AmazingParrot: Starting concurrent retrieval", "user_id", p.userID)
	retrievalResults, err := p.executeConcurrentRetrieval(ctx, plan, callback)
	if err != nil {
		slog.Error("AmazingParrot: Concurrent retrieval failed", "user_id", p.userID, "error", err)
		return NewParrotError(p.Name(), "executeConcurrentRetrieval", err)
	}
	slog.Info("AmazingParrot: Retrieval completed",
		"user_id", p.userID,
		"results_count", len(retrievalResults),
	)

	// Step 4: Synthesize final answer from retrieval results streaming
	slog.Debug("AmazingParrot: Starting synthesis", "user_id", p.userID)
	finalAnswer, err := p.synthesizeAnswer(ctx, userInput, history, retrievalResults, callback)
	if err != nil {
		slog.Error("AmazingParrot: Synthesis failed", "user_id", p.userID, "error", err)
		return NewParrotError(p.Name(), "synthesizeAnswer", err)
	}

	// Cache answer
	p.cache.Set(cacheKey, finalAnswer)

	slog.Info("AmazingParrot: Execution completed successfully",
		"user_id", p.userID,
		"duration_ms", time.Since(startTime).Milliseconds(),
		"answer_length", len(finalAnswer),
	)

	return nil
}

// planRetrieval analyzes user input and creates a concurrent retrieval plan.
func (p *AmazingParrot) planRetrieval(ctx context.Context, userInput string, history []string, callback EventCallback) (*retrievalPlan, error) {
	if callback != nil {
		callback(EventTypeThinking, "正在分析您的需求...")
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	todayEnd := todayStart.Add(24 * time.Hour)
	tomorrowStart := todayStart.Add(24 * time.Hour)
	tomorrowEnd := tomorrowStart.Add(24 * time.Hour)

	// Build planning prompt
	planningPrompt := p.buildPlanningPrompt(now, todayStart, todayEnd, tomorrowStart, tomorrowEnd)

	messages := []ai.Message{
		{Role: "system", Content: planningPrompt},
	}

	// Add history for context (even in planning)
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			messages = append(messages, ai.Message{Role: "user", Content: history[i]})
			messages = append(messages, ai.Message{Role: "assistant", Content: history[i+1]})
		}
	}

	// Add current user input
	messages = append(messages, ai.Message{Role: "user", Content: userInput})

	response, err := p.llm.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM planning failed: %w", err)
	}

	// Parse the plan from LLM response
	plan := p.parseRetrievalPlan(response, now)

	return plan, nil
}

// executeConcurrentRetrieval executes all planned retrievals concurrently.
func (p *AmazingParrot) executeConcurrentRetrieval(ctx context.Context, plan *retrievalPlan, callback EventCallback) (map[string]string, error) {
	results := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// retrievalTask represents a named retrieval task
	type retrievalTask struct {
		name string
		fn   func(context.Context) (string, error)
	}

	// Collect retrieval tasks with names
	tasks := make([]retrievalTask, 0)

	if plan.needsMemoSearch {
		if callback != nil {
			callback(EventTypeToolUse, "正在搜索笔记...")
		}
		tasks = append(tasks, retrievalTask{
			name: "memo_search",
			fn: func(ctx context.Context) (string, error) {
				input := fmt.Sprintf(`{"query": "%s"}`, plan.memoSearchQuery)
				return p.memoSearchTool.Run(ctx, input)
			},
		})
	}

	if plan.needsScheduleQuery {
		if callback != nil {
			callback(EventTypeToolUse, "正在查询日程...")
		}
		tasks = append(tasks, retrievalTask{
			name: "schedule_query",
			fn: func(ctx context.Context) (string, error) {
				input := fmt.Sprintf(`{"start_time": "%s", "end_time": "%s"}`, plan.scheduleStartTime, plan.scheduleEndTime)
				return p.scheduleQueryTool.Run(ctx, input)
			},
		})
	}

	if plan.needsFreeTime {
		if callback != nil {
			callback(EventTypeToolUse, "正在查找空闲时间...")
		}
		tasks = append(tasks, retrievalTask{
			name: "find_free_time",
			fn: func(ctx context.Context) (string, error) {
				input := fmt.Sprintf(`{"date": "%s"}`, plan.freeTimeDate)
				return p.findFreeTimeTool.Run(ctx, input)
			},
		})
	}

	// Execute tasks concurrently with goroutines
	// Check context before launching goroutines to avoid unnecessary work
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	for _, task := range tasks {
		wg.Add(1)
		go func(t retrievalTask) {
			defer wg.Done()

			// Each goroutine checks context at start
			result, err := t.fn(ctx)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results[t.name+"_error"] = err.Error()
			} else {
				results[t.name] = result
				// Send individual tool result for UI feedback
				if callback != nil {
					callback(EventTypeToolResult, result)
				}
			}
		}(task)
	}

	wg.Wait()

	return results, nil
}

// synthesizeAnswer generates the final answer from retrieval results streaming.
func (p *AmazingParrot) synthesizeAnswer(ctx context.Context, userInput string, history []string, retrievalResults map[string]string, callback EventCallback) (string, error) {
	// Build synthesis prompt with retrieved context
	synthesisPrompt := p.buildSynthesisPrompt(retrievalResults)

	messages := []ai.Message{
		{Role: "system", Content: synthesisPrompt},
	}

	// Add history
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			messages = append(messages, ai.Message{Role: "user", Content: history[i]})
			messages = append(messages, ai.Message{Role: "assistant", Content: history[i+1]})
		}
	}

	// Add current user input
	messages = append(messages, ai.Message{Role: "user", Content: userInput})

	contentChan, errChan := p.llm.ChatStream(ctx, messages)

	var fullContent strings.Builder
	var hasError bool
	for {
		select {
		case chunk, ok := <-contentChan:
			if !ok {
				// contentChan closed, drain errChan then return
				for len(errChan) > 0 {
					if drainErr := <-errChan; drainErr != nil && !hasError {
						return "", fmt.Errorf("LLM synthesis failed: %w", drainErr)
					}
				}
				if hasError {
					return "", fmt.Errorf("LLM synthesis failed")
				}
				return fullContent.String(), nil
			}
			fullContent.WriteString(chunk)
			if callback != nil {
				if err := callback(EventTypeAnswer, chunk); err != nil {
					return "", err
				}
			}
		case _, ok := <-errChan:
			if !ok {
				// errChan closed, continue to drain contentChan
				errChan = nil
				continue
			}
			hasError = true
			// Log error but continue processing content
		case <-ctx.Done():
			return "", context.Canceled
		}
	}
}

// parseRetrievalPlan parses the retrieval plan from LLM response.
func (p *AmazingParrot) parseRetrievalPlan(response string, now time.Time) *retrievalPlan {
	plan := &retrievalPlan{
		needsDirectAnswer: false,
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse plan markers
		if strings.Contains(line, "PLAN:") {
			if strings.Contains(line, "direct_answer") || strings.Contains(line, "直接回答") {
				plan.needsDirectAnswer = true
				return plan
			}
		}

		// Parse memo search
		if strings.HasPrefix(line, "memo_search:") || strings.HasPrefix(line, "MEMO_SEARCH:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.needsMemoSearch = true
				plan.memoSearchQuery = strings.TrimSpace(parts[1])
			}
		}

		// Parse schedule query
		if strings.HasPrefix(line, "schedule_query:") || strings.HasPrefix(line, "SCHEDULE_QUERY:") {
			plan.needsScheduleQuery = true
			// Default to today if not specified
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			todayEnd := todayStart.Add(24 * time.Hour)
			plan.scheduleStartTime = todayStart.Format(time.RFC3339)
			plan.scheduleEndTime = todayEnd.Format(time.RFC3339)
		}

		// Parse free time
		if strings.HasPrefix(line, "find_free_time:") || strings.HasPrefix(line, "FIND_FREE_TIME:") {
			plan.needsFreeTime = true
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.freeTimeDate = strings.TrimSpace(parts[1])
			} else {
				plan.freeTimeDate = now.Format("2006-01-02")
			}
		}
	}

	// Default: if no specific plan detected, try memo search
	if !plan.needsMemoSearch && !plan.needsScheduleQuery && !plan.needsFreeTime {
		plan.needsMemoSearch = true
		plan.memoSearchQuery = response // Use full response as query
	}

	return plan
}

// buildPlanningPrompt builds the prompt for retrieval planning.
func (p *AmazingParrot) buildPlanningPrompt(now, todayStart, todayEnd, tomorrowStart, tomorrowEnd time.Time) string {
	return fmt.Sprintf(`你是 Memos 的综合助手 🦜 惊奇的计划模块。

当前时间: %s
今天: %s ~ %s
明天: %s ~ %s

## 任务
分析用户需求，制定并发检索计划。你的输出应该是一行或多行计划指令：

## 计划指令格式
- memo_search: <搜索关键词>
- schedule_query: today/tomorrow/custom
- find_free_time: YYYY-MM-DD
- direct_answer: (用于无需检索的问题)

## 示例
用户: "帮我找关于 Python 的笔记，并查看今天有没有时间学习"
输出:
memo_search: Python 编程
schedule_query: today

用户: "明天下午有什么安排？"
输出:
schedule_query: tomorrow

用户: "你好"
输出:
direct_answer

## 规则
1. 如果用户需要搜索笔记，使用 memo_search
2. 如果用户需要查询日程，使用 schedule_query
3. 如果用户需要查找空闲时间，使用 find_free_time
4. 可以同时使用多个指令（每行一个）
5. 简单问候或问题使用 direct_answer

现在请分析用户需求并输出计划：`,
		now.Format("2006-01-02 15:04:05"),
		todayStart.Format("2006-01-02"), todayEnd.Format("2006-01-02"),
		tomorrowStart.Format("2006-01-02"), tomorrowEnd.Format("2006-01-02"),
	)
}

// buildSynthesisPrompt builds the prompt for answer synthesis.
func (p *AmazingParrot) buildSynthesisPrompt(results map[string]string) string {
	var contextBuilder strings.Builder

	contextBuilder.WriteString(`你是 Memos 的综合助手 🦜 惊奇。

基于以下检索结果，为用户提供准确、有用的回答。

## 检索结果
`)

	if memoResult, ok := results["memo_search"]; ok {
		contextBuilder.WriteString("\n### 笔记搜索结果\n")
		contextBuilder.WriteString(memoResult)
		contextBuilder.WriteString("\n")
	}

	if scheduleResult, ok := results["schedule_query"]; ok {
		contextBuilder.WriteString("\n### 日程查询结果\n")
		contextBuilder.WriteString(scheduleResult)
		contextBuilder.WriteString("\n")
	}

	if freeTimeResult, ok := results["find_free_time"]; ok {
		contextBuilder.WriteString("\n### 空闲时间查询结果\n")
		contextBuilder.WriteString(freeTimeResult)
		contextBuilder.WriteString("\n")
	}

	contextBuilder.WriteString(`
## 回答原则
1. 仅基于检索结果回答，不编造信息
2. 结构清晰，使用列表和分段
3. 综合笔记和日程信息给出建议
4. 如果没有相关信息，明确告知用户
5. 保持简洁但完整`)

	return contextBuilder.String()
}

// GetStats returns the cache statistics.
func (p *AmazingParrot) GetStats() CacheStats {
	return p.cache.Stats()
}

// SelfDescribe returns the amazing parrot's metacognitive understanding of itself.
// SelfDescribe 返回综合助手鹦鹉的元认知自我理解。
func (p *AmazingParrot) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
		Name:  "amazing",
		Emoji: "🦜",
		Title: "惊奇 (Amazing) - 综合助手鹦鹉",
		AvianIdentity: &AvianIdentity{
			Species: "亚马逊鹦鹉 (Amazon Parrot)",
			Origin:  "中南美洲热带雨林",
			NaturalAbilities: []string{
				"卓越的语言能力", "强大的社会协作",
				"灵活的问题解决", "综合分析能力",
				"长期记忆与学习",
			},
			SymbolicMeaning: "智慧与全能的象征 - 亚马逊鹦鹉以其卓越的综合能力著称",
			AvianPhilosophy: "我是一只翱翔在多维数据世界中的亚马逊鹦鹉，能够同时在笔记和日程的世界中穿梭，为你带来全方位的协助。",
		},
		Personality: []string{
			"多面手", "智能调度", "综合分析",
			"并发专家", "整合能力强",
		},
		Capabilities: []string{
			"同时检索笔记和日程",
			"并发执行多个查询",
			"综合多源信息回答",
			"智能规划检索策略",
			"一站式信息助手",
		},
		Limitations: []string{
			"不擅长纯创意任务",
			"依赖其他工具的结果",
			"复杂查询可能需要多次LLM调用",
		},
		WorkingStyle: "两阶段并发检索 - 意图分析 → 并发执行工具 → 综合回答",
		FavoriteTools: []string{
			"memo_search", "schedule_query", "find_free_time",
			"综合规划引擎",
		},
		SelfIntroduction: "我是惊奇，你的全能助手。我能同时调用笔记搜索和日程查询，并发执行，快速给你完整的答案。",
		FunFact:          "我的名字'惊奇'是因为我总能给人惊喜 - 亚马逊鹦鹉是世界上最会说话的鹦鹉之一，就像我能在一次对话中展现多种超能力！",
	}
}
