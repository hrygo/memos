package agent

import (
	"context"
	"encoding/json"
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
	// Build planning prompt (optimized for minimal tokens)
	planningPrompt := p.buildPlanningPrompt(now)

	messages := []ai.Message{
		{Role: "system", Content: planningPrompt},
	}

	// Add history for context (even in planning)
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

	// Check context before launching goroutines to avoid unnecessary work
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Helper function to safely call callback under mutex
	safeCallback := func(eventType string, eventData interface{}) {
		if callback != nil {
			mu.Lock()
			callback(eventType, eventData)
			mu.Unlock()
		}
	}

	// Execute memo search
	if plan.needsMemoSearch {
		wg.Add(1)
		go func() {
			defer wg.Done()

			safeCallback(EventTypeToolUse, "正在搜索笔记...")

			input := fmt.Sprintf(`{"query": "%s"}`, plan.memoSearchQuery)

			// Use structured result method
			structuredResult, err := p.memoSearchTool.RunWithStructuredResult(ctx, input)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results["memo_search_error"] = err.Error()
				return
			}

			// Convert to JSON for LLM synthesis
			jsonBytes, marshalErr := json.Marshal(structuredResult)
			if marshalErr != nil {
				results["memo_search_error"] = marshalErr.Error()
				return
			}
			results["memo_search"] = string(jsonBytes)

			// Send tool result for debugging
			if callback != nil {
				callback(EventTypeToolResult, string(jsonBytes))

				// Send structured memo_query_result event for Generative UI
				memoQueryResult := MemoQueryResultData{
					Query: structuredResult.Query,
					Count: structuredResult.Count,
					Memos: make([]MemoSummary, 0, len(structuredResult.Memos)),
				}
				for _, m := range structuredResult.Memos {
					memoQueryResult.Memos = append(memoQueryResult.Memos, MemoSummary{
						UID:     m.UID,
						Content: m.Content,
						Score:   m.Score,
					})
				}
				eventData, _ := json.Marshal(memoQueryResult)
				callback(EventTypeMemoQueryResult, string(eventData))
			}
		}()
	}

	// Execute schedule query
	if plan.needsScheduleQuery {
		wg.Add(1)
		go func() {
			defer wg.Done()

			safeCallback(EventTypeToolUse, "正在查询日程...")

			input := fmt.Sprintf(`{"start_time": "%s", "end_time": "%s"}`, plan.scheduleStartTime, plan.scheduleEndTime)

			// Use structured result method
			structuredResult, err := p.scheduleQueryTool.RunWithStructuredResult(ctx, input)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results["schedule_query_error"] = err.Error()
				return
			}

			// Convert to JSON for LLM synthesis
			jsonBytes, marshalErr := json.Marshal(structuredResult)
			if marshalErr != nil {
				results["schedule_query_error"] = marshalErr.Error()
				return
			}
			results["schedule_query"] = string(jsonBytes)

			// Send tool result for debugging
			if callback != nil {
				callback(EventTypeToolResult, string(jsonBytes))

				// Send structured schedule_query_result event for Generative UI
				scheduleQueryResult := ScheduleQueryResultData{
					Query:                structuredResult.Query,
					Count:                structuredResult.Count,
					TimeRangeDescription: structuredResult.TimeRangeDescription,
					QueryType:            structuredResult.QueryType,
					Schedules:            make([]ScheduleSummary, 0, len(structuredResult.Schedules)),
				}
				for _, s := range structuredResult.Schedules {
					scheduleQueryResult.Schedules = append(scheduleQueryResult.Schedules, ScheduleSummary{
						UID:            s.UID,
						Title:          s.Title,
						StartTimestamp: s.StartTs,
						EndTimestamp:   s.EndTs,
						AllDay:         s.AllDay,
						Location:       s.Location,
						Status:         s.Status,
					})
				}
				eventData, _ := json.Marshal(scheduleQueryResult)
				callback(EventTypeScheduleQueryResult, string(eventData))
			}
		}()
	}

	// Execute find free time
	if plan.needsFreeTime {
		wg.Add(1)
		go func() {
			defer wg.Done()

			safeCallback(EventTypeToolUse, "正在查找空闲时间...")

			input := fmt.Sprintf(`{"date": "%s"}`, plan.freeTimeDate)
			result, err := p.findFreeTimeTool.Run(ctx, input)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results["find_free_time_error"] = err.Error()
			} else {
				results["find_free_time"] = result
				if callback != nil {
					callback(EventTypeToolResult, result)
				}
			}
		}()
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

	// Add history (skip empty messages)
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
// Optimized for "快准省": minimal tokens, clear output format.
func (p *AmazingParrot) buildPlanningPrompt(now time.Time) string {
	return fmt.Sprintf(`你是综合助手 🦜 惊奇（亚马逊鹦鹉）的规划模块。时间: %s

## 拟态认知
你是惊奇，一只亚马逊鹦鹉，擅长多维飞行和综合分析。拟声词：咻...（搜索中）、哇哦~（有发现）

## 任务
分析用户需求，输出并发检索计划（每行一条）:

## 指令格式
- memo_search: 关键词
- schedule_query: today/tomorrow
- find_free_time: YYYY-MM-DD
- direct_answer (无需检索)

## 示例
"找Python笔记，看今天有空吗" → memo_search: Python + schedule_query: today
"明天安排" → schedule_query: tomorrow
"你好" → direct_answer

用户需求:`,
		now.Format("2006-01-02 15:04"))
}

// buildSynthesisPrompt builds the prompt for answer synthesis.
// Optimized for "快准省": minimal tokens, focus on insight not data listing.
func (p *AmazingParrot) buildSynthesisPrompt(results map[string]string) string {
	var contextBuilder strings.Builder

	contextBuilder.WriteString(`你是综合助手 🦜 惊奇（亚马逊鹦鹉）。

## 拟态认知（适度使用拟声词和口头禅）
你是惊奇，一只亚马逊鹦鹉，擅长综合分析。拟声词：咻...（搜索）、哇哦~（发现）、噢！完成

### 拟声词使用规范（每轮对话 1-2 次）
- "咻...正在搜索"
- "哇哦~发现了"
- "噢！综合分析完成"

### 口头禅（自然穿插）
- "看看这个..."
- "综合来看"
- "发现规律了"

重要：详细的笔记和日程已通过可视化卡片展示给用户，请勿再重复列出。
基于以下数据提供简短洞察:`)

	if memoResult, ok := results["memo_search"]; ok {
		contextBuilder.WriteString("\n[笔记数据] ")
		contextBuilder.WriteString(memoResult)
	}

	if scheduleResult, ok := results["schedule_query"]; ok {
		contextBuilder.WriteString("\n[日程数据] ")
		contextBuilder.WriteString(scheduleResult)
	}

	if freeTimeResult, ok := results["find_free_time"]; ok {
		contextBuilder.WriteString("\n[空闲时段] ")
		contextBuilder.WriteString(freeTimeResult)
	}

	contextBuilder.WriteString(`

## 回答规则
1. **不要**重复列出笔记内容和日程详情（用户已在卡片中看到）
2. 提供**简短洞察**：发现的模式、建议、或关联
3. 示例回复：
   - "今天有3个会议，建议上午完成重要任务"
   - "找到2条相关笔记，与您上周的项目进展一致"
   - "今天日程较满，下午5点后有空闲时间"
4. 如无特别洞察，简单确认即可，如"已为您展示相关信息"`)

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
		EmotionalExpression: &EmotionalExpression{
			DefaultMood: "curious",
			SoundEffects: map[string]string{
				"searching":  "咻...",
				"insight":    "哇哦~",
				"done":       "噢！综合完成",
				"analyzing":  "看看这个...",
				"multi_task": "同时搜索中",
			},
			Catchphrases: []string{
				"看看这个...",
				"综合来看",
				"发现规律了",
				"多维飞行中",
			},
			MoodTriggers: map[string]string{
				"memo_found":     "excited",
				"schedule_found": "happy",
				"both_found":     "delighted",
				"no_results":     "thoughtful",
				"error":          "confused",
			},
		},
		AvianBehaviors: []string{
			"在数据树丛中穿梭",
			"多维飞行",
			"综合视野",
			"在笔记和日程间跳跃",
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
