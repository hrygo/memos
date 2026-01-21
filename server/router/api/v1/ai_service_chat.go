package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/finops"
	"github.com/usememos/memos/server/queryengine"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/store"
)

// Pre-compiled regex patterns for schedule query intent detection
var scheduleQueryPatterns = []struct {
	patterns      []*regexp.Regexp
	intentType    string
	timeRange     string
	calcTimeRange func() (*time.Time, *time.Time)
}{
	{
		// Upcoming schedules (next 7 days)
		patterns: []*regexp.Regexp{
			regexp.MustCompile("近期.*日程"),
			regexp.MustCompile("近期.*安排"),
			regexp.MustCompile("近期的.*日程"),
			regexp.MustCompile("未来.*日程"),
			regexp.MustCompile("接下来.*日程"),
			regexp.MustCompile("最近.*日程"),
			regexp.MustCompile("后面.*日程"),
			regexp.MustCompile("我的.*近期"),
			regexp.MustCompile("我.*近期.*日程"),
			regexp.MustCompile("查看.*近期"),
			regexp.MustCompile("查询.*近期"),
			regexp.MustCompile("有什么安排"),
			regexp.MustCompile("日程查询"),
		},
		intentType: "upcoming",
		timeRange:  "未来7天",
		calcTimeRange: func() (*time.Time, *time.Time) {
			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endOfPeriod := startOfDay.Add(7 * 24 * time.Hour)
			return &startOfDay, &endOfPeriod
		},
	},
	{
		// Today's schedules
		patterns: []*regexp.Regexp{
			regexp.MustCompile("今天.*日程"),
			regexp.MustCompile("今天.*安排"),
			regexp.MustCompile("今天.*事"),
			regexp.MustCompile("今天有什么"),
		},
		intentType: "range",
		timeRange:  "今天",
		calcTimeRange: func() (*time.Time, *time.Time) {
			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endOfDay := startOfDay.Add(24 * time.Hour)
			return &startOfDay, &endOfDay
		},
	},
	{
		// Tomorrow's schedules
		patterns: []*regexp.Regexp{
			regexp.MustCompile("明天.*日程"),
			regexp.MustCompile("明天.*安排"),
			regexp.MustCompile("明天.*事"),
			regexp.MustCompile("明天有什么"),
		},
		intentType: "range",
		timeRange:  "明天",
		calcTimeRange: func() (*time.Time, *time.Time) {
			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			endOfDay := startOfDay.Add(24 * time.Hour)
			return &startOfDay, &endOfDay
		},
	},
	{
		// This week's schedules
		patterns: []*regexp.Regexp{
			regexp.MustCompile("本周.*日程"),
			regexp.MustCompile("这周.*安排"),
			regexp.MustCompile("这周.*事"),
			regexp.MustCompile("本周有什么"),
		},
		intentType: "range",
		timeRange:  "本周",
		calcTimeRange: func() (*time.Time, *time.Time) {
			now := time.Now()
			// Start of week (Monday)
			weekday := now.Weekday()
			if weekday == time.Sunday {
				weekday = 7
			}
			startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-int(weekday)+1, 0, 0, 0, 0, now.Location())
			// End of week (Sunday)
			endOfWeek := startOfWeek.Add(7 * 24 * time.Hour)
			return &startOfWeek, &endOfWeek
		},
	},
	{
		// Next week's schedules
		patterns: []*regexp.Regexp{
			regexp.MustCompile("下周.*日程"),
			regexp.MustCompile("下周.*安排"),
			regexp.MustCompile("下周.*事"),
			regexp.MustCompile("下周有什么"),
		},
		intentType: "range",
		timeRange:  "下周",
		calcTimeRange: func() (*time.Time, *time.Time) {
			now := time.Now()
			// Start of next week (Monday)
			weekday := now.Weekday()
			if weekday == time.Sunday {
				weekday = 7
			}
			startOfNextWeek := time.Date(now.Year(), now.Month(), now.Day()-int(weekday)+1+7, 0, 0, 0, 0, now.Location())
			// End of next week (Sunday)
			endOfNextWeek := startOfNextWeek.Add(7 * 24 * time.Hour)
			return &startOfNextWeek, &endOfNextWeek
		},
	},
}

// ChatWithMemos streams a chat response using memos as context.
// 优化版本：使用 Query Routing、Adaptive Retrieval 和 FinOps 监控
func (s *AIService) ChatWithMemos(req *v1pb.ChatWithMemosRequest, stream v1pb.AIService_ChatWithMemosServer) error {
	ctx := stream.Context()

	// Debug: Log every AI chat request
	fmt.Printf("\n======== [ChatWithMemos] NEW REQUEST (Optimized) ========\n")
	fmt.Printf("[ChatWithMemos] User message: '%s'\n", req.Message)
	fmt.Printf("[ChatWithMemos] History items: %d\n", len(req.History))
	fmt.Printf("=========================================================\n\n")

	if !s.IsEnabled() {
		return status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// 1. 获取当前用户
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// 2. 速率限制检查
	userKey := strconv.FormatInt(int64(user.ID), 10)
	if !globalAILimiter.Allow(userKey) {
		return status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded: please wait before making another AI chat request")
	}

	// 3. 参数校验
	if req.Message == "" {
		return status.Errorf(codes.InvalidArgument, "message is required")
	}

	// ============================================================
	// Phase 1: 智能 Query Routing（⭐ 新增）
	// ============================================================
	var routeDecision *queryengine.RouteDecision

	// 解析用户时区
	var userTimezone *time.Location
	if req.UserTimezone != "" {
		var err error
		userTimezone, err = time.LoadLocation(req.UserTimezone)
		if err != nil {
			fmt.Printf("[ChatWithMemos] Invalid timezone %q, using UTC: %v\n", req.UserTimezone, err)
			userTimezone = time.UTC
		}
	} else {
		userTimezone = time.UTC
	}

	if s.QueryRouter != nil {
		routeDecision = s.QueryRouter.Route(ctx, req.Message, userTimezone)
		fmt.Printf("[QueryRouting] Strategy: %s, Confidence: %.2f, Timezone: %v\n",
			routeDecision.Strategy, routeDecision.Confidence, userTimezone)
	} else {
		// 降级：默认策略
		routeDecision = &queryengine.RouteDecision{
			Strategy:      "hybrid_standard",
			Confidence:    0.80,
			SemanticQuery: req.Message,
			NeedsReranker: false,
		}
	}

	// ============================================================
	// Phase 2: Adaptive Retrieval（⭐ 新增）
	// ============================================================
	retrievalStart := time.Now()

	var searchResults []*retrieval.SearchResult
	if s.AdaptiveRetriever != nil {
		// 使用新的自适应检索器
		searchResults, err = s.AdaptiveRetriever.Retrieve(ctx, &retrieval.RetrievalOptions{
			Query:            req.Message,
			UserID:           user.ID,
			Strategy:         routeDecision.Strategy,
			TimeRange:        routeDecision.TimeRange,
			ScheduleQueryMode: routeDecision.ScheduleQueryMode, // P1: 传递查询模式
			MinScore:         0.5,
			Limit:            10,
		})
		if err != nil {
			fmt.Printf("[AdaptiveRetriever] Error: %v, using fallback\n", err)
			// 降级到旧逻辑
			searchResults, err = s.fallbackRetrieval(ctx, user.ID, req.Message)
			if err != nil {
				return status.Errorf(codes.Internal, "retrieval failed: %v", err)
			}
		}
	} else {
		// 降级到旧逻辑
		searchResults, err = s.fallbackRetrieval(ctx, user.ID, req.Message)
		if err != nil {
			return status.Errorf(codes.Internal, "retrieval failed: %v", err)
		}
	}

	retrievalDuration := time.Since(retrievalStart)
	fmt.Printf("[Retrieval] Completed in %dms, found %d results\n",
		retrievalDuration.Milliseconds(), len(searchResults))

	// 分类结果：笔记和日程
	var memoResults []*retrieval.SearchResult
	var scheduleResults []*retrieval.SearchResult
	for _, result := range searchResults {
		switch result.Type {
		case "memo":
			memoResults = append(memoResults, result)
		case "schedule":
			scheduleResults = append(scheduleResults, result)
		}
	}

	// ============================================================
	// Phase 3: 构建上下文和提示词（⭐ 优化）
	// ============================================================
	var contextBuilder strings.Builder
	var sources []string
	totalChars := 0
	maxChars := 3000

	// 添加笔记到上下文
	for i, r := range memoResults {
		content := r.Content
		if totalChars+len(content) > maxChars {
			break
		}

		contextBuilder.WriteString(fmt.Sprintf("### 笔记 %d (相关度: %.0f%%)\n%s\n\n", i+1, r.Score*100, content))
		if r.Memo != nil {
			sources = append(sources, fmt.Sprintf("memos/%s", r.Memo.UID))
		}
		totalChars += len(content)

		if len(sources) >= 5 {
			break
		}
	}

	// 构建优化后的提示词
	var hasNotes = len(memoResults) > 0
	var hasSchedules = len(scheduleResults) > 0

	messages := s.buildOptimizedMessages(req.Message, req.History, contextBuilder.String(),
		scheduleResults, hasNotes, hasSchedules)

	// ============================================================
	// Phase 4: 流式调用 LLM
	// ============================================================
	llmStart := time.Now()

	contentChan, errChan := s.LLMService.ChatStream(ctx, messages)

	// 先发送来源信息
	if err := stream.Send(&v1pb.ChatWithMemosResponse{
		Sources: sources,
	}); err != nil {
		return err
	}

	// 收集完整回复内容
	var fullContent strings.Builder

	// 流式发送内容
	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				contentChan = nil
				if errChan == nil {
					llmDuration := time.Since(llmStart)
					return s.finalizeChatStreamOptimized(stream, fullContent.String(),
						scheduleResults, routeDecision, retrievalDuration, llmDuration)
				}
				continue
			}
			fullContent.WriteString(content)
			if err := stream.Send(&v1pb.ChatWithMemosResponse{
				Content: content,
			}); err != nil {
				return err
			}

		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				if contentChan == nil {
					llmDuration := time.Since(llmStart)
					return s.finalizeChatStreamOptimized(stream, fullContent.String(),
						scheduleResults, routeDecision, retrievalDuration, llmDuration)
				}
				continue
			}
			if err != nil {
				return status.Errorf(codes.Internal, "LLM error: %v", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// fallbackRetrieval 降级检索逻辑（兼容旧版本）
func (s *AIService) fallbackRetrieval(ctx context.Context, userID int32, query string) ([]*retrieval.SearchResult, error) {
	// 简化的向量检索
	queryVector, err := s.EmbeddingService.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	vectorResults, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: userID,
		Vector: queryVector,
		Limit:  20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 转换为 SearchResult
	results := make([]*retrieval.SearchResult, len(vectorResults))
	for i, r := range vectorResults {
		results[i] = &retrieval.SearchResult{
			ID:      int64(r.Memo.ID),
			Type:    "memo",
			Score:   r.Score,
			Content: r.Memo.Content,
			Memo:    r.Memo,
		}
	}

	return results, nil
}

// buildOptimizedMessages 构建优化后的消息（简化版提示词）
func (s *AIService) buildOptimizedMessages(
	userMessage string,
	history []string,
	memoContext string,
	scheduleResults []*retrieval.SearchResult,
	hasNotes, hasSchedules bool,
) []ai.Message {
	// ============================================================
	// System Prompt - 简化版（⭐ 优化）
	// ============================================================
	systemPrompt := `你是 Memos AI 助手，帮助用户管理笔记和日程。

## 回复原则
1. **简洁准确**：基于提供的上下文回答，不编造信息
2. **结构清晰**：使用列表、分段组织内容
3. **完整回复**：
   - 如果有日程，优先列出日程
   - 如果有笔记，补充相关笔记
   - 如果都没有，明确告知

## 日程创建检测（重要）
⚠️ **仅在用户的原始问题明确表示要创建日程时**才添加意图标记：
- 创建意图的明确关键词："帮我创建"、"帮我添加"、"设置提醒"、"新建日程"
- ❌ 以下情况**不是**创建意图：
  - 查询类："有哪些"、"有什么安排"、"今天干什么"、"明天的事要干"
  - 确认类："我明天有安排吗"、"今天有空吗"

仅在检测到创建意图时，在回复最后一行添加：
<<<SCHEDULE_INTENT:{"detected":true,"schedule_description":"自然语言描述"}>>>`

	// 构建消息
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
	}

	// 添加历史对话
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			messages = append(messages, ai.Message{Role: "user", Content: history[i]})
			messages = append(messages, ai.Message{Role: "assistant", Content: history[i+1]})
		}
	}

	// ============================================================
	// User Message - 构建上下文
	// ============================================================
	var userMsgBuilder strings.Builder

	// 添加上下文标题
	if hasNotes || hasSchedules {
		userMsgBuilder.WriteString("## 上下文信息\n\n")
	}

	// 添加笔记上下文
	if hasNotes {
		userMsgBuilder.WriteString("### 📝 相关笔记\n")
		userMsgBuilder.WriteString(memoContext)
		userMsgBuilder.WriteString("\n")
	}

	// 添加日程上下文
	if hasSchedules {
		userMsgBuilder.WriteString("### 📅 日程安排\n")
		for i, r := range scheduleResults {
			if r.Schedule != nil {
				scheduleTime := time.Unix(r.Schedule.StartTs, 0)
				timeStr := scheduleTime.Format("2006-01-02 15:04")
				userMsgBuilder.WriteString(fmt.Sprintf("%d. %s - %s", i+1, timeStr, r.Schedule.Title))
				if r.Schedule.Location != "" {
					userMsgBuilder.WriteString(fmt.Sprintf(" @ %s", r.Schedule.Location))
				}
				userMsgBuilder.WriteString("\n")
			}
		}
		userMsgBuilder.WriteString("\n")
	}

	// 用户问题
	userMsgBuilder.WriteString("## 问题\n")
	userMsgBuilder.WriteString(userMessage)

	messages = append(messages, ai.Message{Role: "user", Content: userMsgBuilder.String()})

	return messages
}

// finalizeChatStreamOptimized 发送最终响应（优化版，包含性能指标）
func (s *AIService) finalizeChatStreamOptimized(
	stream v1pb.AIService_ChatWithMemosServer,
	aiResponse string,
	scheduleResults []*retrieval.SearchResult,
	routeDecision *queryengine.RouteDecision,
	retrievalDuration, llmDuration time.Duration,
) error {
	totalDuration := retrievalDuration + llmDuration

	fmt.Printf("[ChatWithMemos] Completed - Retrieval: %dms, LLM: %dms, Total: %dms, Strategy: %s\n",
		retrievalDuration.Milliseconds(), llmDuration.Milliseconds(),
		totalDuration.Milliseconds(), routeDecision.Strategy)

	// ============================================================
	// Phase 5: FinOps 监控记录（⭐ 新增）
	// ============================================================
	ctx := stream.Context()
	user, err := getCurrentUser(ctx, s.Store)
	if err == nil && s.CostMonitor != nil {
		// 估算成本
		vectorCost := finops.EstimateEmbeddingCost(len(aiResponse))
		llmCost := finops.EstimateLLMCost(len(aiResponse)*2, len(aiResponse)) // 粗略估算
		// totalCost is calculated internally by CreateQueryCostRecord or not needed here

		// 创建成本记录
		record := finops.CreateQueryCostRecord(
			user.ID,
			"", // query（从上下文获取，这里简化为空）
			routeDecision.Strategy,
			vectorCost,
			0, // rerankerCost（如果使用了）
			llmCost,
			totalDuration.Milliseconds(),
			len(scheduleResults),
		)

		// 异步记录成本（避免阻塞响应）
		go func() {
			if err := s.CostMonitor.Record(context.Background(), record); err != nil {
				fmt.Printf("[FinOps] Failed to record cost: %v\n", err)
			}
		}()
	}

	// 解析日程创建意图
	scheduleIntent := s.parseScheduleIntentFromAIResponse(aiResponse)

	// 构建最终响应
	response := &v1pb.ChatWithMemosResponse{
		Done: true,
	}

	// 添加日程创建意图
	if scheduleIntent != nil {
		response.ScheduleCreationIntent = scheduleIntent
	}

	// 添加日程查询结果
	if len(scheduleResults) > 0 {
		scheduleSummaries := make([]*v1pb.ScheduleSummary, 0, len(scheduleResults))
		for _, r := range scheduleResults {
			if r.Schedule != nil {
				summary := &v1pb.ScheduleSummary{
					Uid:      fmt.Sprintf("schedules/%d", r.Schedule.ID),
					Title:    r.Schedule.Title,
					StartTs:  r.Schedule.StartTs,
					AllDay:   r.Schedule.AllDay,
					Location: r.Schedule.Location,
				}

				// 处理可选字段
				if r.Schedule.EndTs != nil {
					summary.EndTs = *r.Schedule.EndTs
				}
				if r.Schedule.RecurrenceRule != nil {
					summary.RecurrenceRule = *r.Schedule.RecurrenceRule
				}
				// 使用 RowStatus 作为 Status
				summary.Status = r.Schedule.RowStatus.String()

				scheduleSummaries = append(scheduleSummaries, summary)
			}
		}
		response.ScheduleQueryResult = &v1pb.ScheduleQueryResult{
			Schedules: scheduleSummaries,
		}
	}

	return stream.Send(response)
}

// parseScheduleIntentFromAIResponse parses schedule intent from AI's response text
// Marker format: <<<SCHEDULE_INTENT:{"detected":true,"schedule_description":"..."}>>>
func (s *AIService) parseScheduleIntentFromAIResponse(aiResponse string) *v1pb.ScheduleCreationIntent {
	// 查找意图标记：使用独特的 <<<SCHEDULE_INTENT: 格式避免误判
	const intentMarker = "<<<SCHEDULE_INTENT:"

	startIdx := strings.Index(aiResponse, intentMarker)
	if startIdx == -1 {
		// 没有意图标记，用户没有创建日程的意图
		return nil
	}

	// 提取 JSON 部分
	startIdx += len(intentMarker)

	// 查找结束标记 >>>（使用 LastIndex 避免描述中的 >>> 截断）
	endIdx := strings.LastIndex(aiResponse[startIdx:], ">>>")
	if endIdx == -1 {
		fmt.Printf("[ScheduleIntent] Found marker but missing closing '>>>'\n")
		return nil
	}

	jsonStr := strings.TrimSpace(aiResponse[startIdx : startIdx+endIdx])

	// 清理 JSON 字符串：移除换行符和制表符，但保留空格（description 中可能包含空格）
	cleanJSON := strings.ReplaceAll(jsonStr, "\n", "")
	cleanJSON = strings.ReplaceAll(cleanJSON, "\t", "")
	cleanJSON = strings.TrimSpace(cleanJSON)

	// 解析 JSON
	type IntentJSON struct {
		Detected            bool   `json:"detected"`
		ScheduleDescription string `json:"schedule_description"` // 正确的字段名
		Description         string `json:"description"`          // 兼容旧字段名
	}

	var intentJSON IntentJSON
	if err := json.Unmarshal([]byte(cleanJSON), &intentJSON); err != nil {
		fmt.Printf("[ScheduleIntent] Failed to parse intent JSON: %v, original: %s, cleaned: %s\n", err, jsonStr, cleanJSON)
		return nil
	}

	// 检查是否检测到意图
	if !intentJSON.Detected {
		return nil
	}

	// 获取描述（优先使用正确的字段名，兼容旧字段名）
	description := intentJSON.ScheduleDescription
	if description == "" {
		description = intentJSON.Description // 兼容旧格式
	}

	// 验证描述不为空
	if strings.TrimSpace(description) == "" {
		fmt.Printf("[ScheduleIntent] Intent detected but description is empty\n")
		return nil
	}

	// 构建返回对象
	intent := &v1pb.ScheduleCreationIntent{
		Detected:            true,
		ScheduleDescription: description,
	}

	// 记录成功解析
	fmt.Printf("[ScheduleIntent] Successfully parsed intent: description='%s'\n", description)

	return intent
}

// detectScheduleQueryIntent detects whether user wants to query schedules.
// Uses pre-compiled regex patterns for performance and reliability.
func (s *AIService) detectScheduleQueryIntent(message string) *ScheduleQueryIntent {
	// Normalize message for matching
	normalizedMessage := strings.ToLower(strings.TrimSpace(message))

	// Try to match patterns using pre-compiled regex
	for _, qp := range scheduleQueryPatterns {
		for _, pattern := range qp.patterns {
			if pattern.MatchString(normalizedMessage) {
				startTime, endTime := qp.calcTimeRange()
				return &ScheduleQueryIntent{
					Detected:  true,
					QueryType: qp.intentType,
					TimeRange: qp.timeRange,
					StartTime: startTime,
					EndTime:   endTime,
				}
			}
		}
	}

	// No schedule query intent detected
	return &ScheduleQueryIntent{Detected: false}
}

// ScheduleQueryIntent represents the detected intent for schedule query.
type ScheduleQueryIntent struct {
	Detected  bool
	QueryType string // "upcoming", "range", "filter"
	TimeRange string // "7d", "today", "tomorrow", "week"
	StartTime *time.Time
	EndTime   *time.Time
}

// formatSchedulesForContext formats schedules for AI context.
func (s *AIService) formatSchedulesForContext(schedules []*v1pb.ScheduleSummary) string {
	if len(schedules) == 0 {
		return "共找到 0 个日程安排（暂无日程）"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "共找到 %d 个日程安排（按时间排序）：\n\n", len(schedules))

	for i, sched := range schedules {
		startTime := time.Unix(sched.StartTs, 0)
		timeStr := startTime.Format("2006-01-02 15:04")
		if sched.AllDay {
			timeStr = startTime.Format("2006-01-02") + " (全天)"
		}

		location := ""
		if sched.Location != "" {
			location = fmt.Sprintf(" @ %s", sched.Location)
		}

		recurrence := ""
		if sched.RecurrenceRule != "" {
			recurrence = " [重复]"
		}

		fmt.Fprintf(&builder, "%d. %s: %s%s%s\n", i+1, timeStr, sched.Title, location, recurrence)
	}

	return builder.String()
}
