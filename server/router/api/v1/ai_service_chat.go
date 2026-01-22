package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/plugin/ai"
	agentpkg "github.com/usememos/memos/plugin/ai/agent"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/finops"
	"github.com/usememos/memos/server/queryengine"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/server/service/schedule"
	"github.com/usememos/memos/store"
)

// Constants for AI chat configuration
const (
	// MaxContextLength is the maximum length of context to include in LLM prompt
	MaxContextLength = 3000

	// MaxAgentIterations is the maximum number of iterations for agent ReAct loop
	MaxAgentIterations = 5

	// StreamTimeout is the timeout for streaming responses
	StreamTimeout = 60 * time.Second

	// AsyncRecordTimeout is the timeout for async cost recording
	// A simple INSERT should complete in <100ms normally.
	// Using 500ms provides 5x buffer for abnormal conditions (high load, network latency)
	// If it takes longer than 500ms, there's likely a systemic issue that should be investigated.
	AsyncRecordTimeout = 500 * time.Millisecond

	// DefaultAgentSystemPrompt is the system prompt for the default agent
	DefaultAgentSystemPrompt = "你是 Memos AI 助手。"
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

	if !s.IsEnabled() {
		return status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Log request info with structured logging
	slog.Debug("ChatWithMemos new request",
		"message", req.Message,
		"history_count", len(req.History),
		"agent_type", req.AgentType.String(),
	)

	// ============================================================
	// 鹦鹉路由（Milestone 1 - NEW）
	// ============================================================
	// 检查是否需要路由到鹦鹉代理
	if req.AgentType != v1pb.AgentType_AGENT_TYPE_DEFAULT {
		return s.chatWithParrot(ctx, req, stream)
	}

	// 原有逻辑继续...

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
			slog.Warn("Invalid timezone, using UTC", "timezone", req.UserTimezone, "error", err)
			userTimezone = time.UTC
		}
	} else {
		userTimezone = time.UTC
	}

	if s.QueryRouter != nil {
		routeDecision = s.QueryRouter.Route(ctx, req.Message, userTimezone)
		slog.Debug("Query routing decision",
			"strategy", routeDecision.Strategy,
			"confidence", routeDecision.Confidence,
			"timezone", userTimezone,
		)
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
			Query:             req.Message,
			UserID:            user.ID,
			Strategy:          routeDecision.Strategy,
			TimeRange:         routeDecision.TimeRange,
			ScheduleQueryMode: routeDecision.ScheduleQueryMode, // P1: 传递查询模式
			MinScore:          0.5,
			Limit:             10,
		})
		if err != nil {
			slog.Warn("AdaptiveRetriever error, using fallback", "error", err)
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
	slog.Debug("Retrieval completed",
		"duration_ms", retrievalDuration.Milliseconds(),
		"results_count", len(searchResults),
	)

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
	maxChars := MaxContextLength

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
	// 构建消息
	messages := []ai.Message{
		{Role: "system", Content: DefaultAgentSystemPrompt},
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

	slog.Debug("ChatWithMemos completed",
		"retrieval_ms", retrievalDuration.Milliseconds(),
		"llm_ms", llmDuration.Milliseconds(),
		"total_ms", totalDuration.Milliseconds(),
		"strategy", routeDecision.Strategy,
	)

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

		// 异步记录成本（避免阻塞响应，使用独立 goroutine 和超时控制）
		go func() {
			// 使用带超时的 context，防止 goroutine 泄漏
			ctx, cancel := context.WithTimeout(context.Background(), AsyncRecordTimeout)
			defer cancel()

			// 重试机制：最多重试 2 次
			maxRetries := 2
			var err error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					// 指数退避
					time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
				}
				err = s.CostMonitor.Record(ctx, record)
				if err == nil {
					return
				}
			}

			// 所有重试都失败后记录警告
			slog.Warn("Failed to record cost after retries",
				"error", err,
				"user_id", user.ID,
				"strategy", routeDecision.Strategy,
				"retries", maxRetries,
			)
		}()
	}

	// 解析日程创建意图
	scheduleIntent := ParseScheduleIntentFromAIResponse(aiResponse)

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

// chatWithParrot handles chat requests routed to parrot agents.
// chatWithParrot 处理路由到鹦鹉代理的聊天请求。
func (s *AIService) chatWithParrot(
	ctx context.Context,
	req *v1pb.ChatWithMemosRequest,
	stream v1pb.AIService_ChatWithMemosServer,
) error {
	// Check if LLM service is initialized (required for all Agents)
	if !s.IsLLMEnabled() {
		slog.Warn("LLM service not available for Agent chat",
			"agent_type", req.AgentType.String(),
			"user_message", req.Message,
		)
		return status.Errorf(codes.Unavailable, "LLM service is not available. Please check your AI configuration and ensure LLM provider is set correctly.")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Rate limiting check
	userKey := strconv.FormatInt(int64(user.ID), 10)
	if !globalAILimiter.Allow(userKey) {
		return status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded: please wait before making another AI chat request")
	}

	// Get agent type
	agentType := req.AgentType
	agentTypeStr := agentType.String()
	slog.Info("ChatWithParrot: Starting agent execution",
		"agent_type", agentTypeStr,
		"user_id", user.ID,
		"message_length", len(req.Message),
		"history_count", len(req.History),
	)

	// Get user timezone from request
	userTimezone := req.UserTimezone
	if userTimezone == "" {
		userTimezone = "Asia/Shanghai" // Default timezone
	}

	// Use mutex to ensure thread-safety for stream.Send in concurrent agent execution (e.g. AmazingParrot)
	var streamMu sync.Mutex

	// Create stream adapter
	streamAdapter := agentpkg.NewParrotStreamAdapter(func(eventType string, eventData interface{}) error {
		// Convert event data to string for streaming
		var dataStr string
		switch v := eventData.(type) {
		case string:
			dataStr = v
		case error:
			dataStr = v.Error()
		default:
			// Try to convert to JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				dataStr = fmt.Sprintf("%v", v)
			} else {
				dataStr = string(jsonBytes)
			}
		}

		// Thread-safe send
		streamMu.Lock()
		defer streamMu.Unlock()

		// Send event through stream
		return stream.Send(&v1pb.ChatWithMemosResponse{
			EventType: eventType,
			EventData: dataStr,
		})
	})

	// Create appropriate agent based on type
	var parrotAgent agentpkg.ParrotAgent
	scheduleSvc := schedule.NewService(s.Store)

	switch agentType {
	case v1pb.AgentType_AGENT_TYPE_MEMO:
		// Memo Parrot (灰灰)
		parrotAgent, err = agentpkg.NewMemoParrot(
			s.AdaptiveRetriever,
			s.LLMService,
			user.ID,
		)
	case v1pb.AgentType_AGENT_TYPE_SCHEDULE:
		// Schedule Parrot (金刚) - wrap existing SchedulerAgent
		schedulerAgent, agentErr := agentpkg.NewSchedulerAgent(
			s.LLMService,
			scheduleSvc,
			user.ID,
			userTimezone,
		)
		if agentErr != nil {
			return status.Errorf(codes.Internal, "failed to create scheduler agent: %v", agentErr)
		}
		parrotAgent, err = agentpkg.NewScheduleParrot(schedulerAgent)
	case v1pb.AgentType_AGENT_TYPE_AMAZING:
		// Amazing Parrot (惊奇) - comprehensive assistant
		parrotAgent, err = agentpkg.NewAmazingParrot(
			s.LLMService,
			s.AdaptiveRetriever,
			scheduleSvc,
			user.ID,
		)
	case v1pb.AgentType_AGENT_TYPE_CREATIVE:
		// Creative Parrot (灵灵) - creative writing assistant
		parrotAgent, err = agentpkg.NewCreativeParrot(
			s.LLMService,
			user.ID,
		)
	default:
		// For DEFAULT or unknown types, fall back to standard RAG chat
		return s.chatWithStandardRAG(ctx, req, stream, user)
	}

	if err != nil {
		slog.Error("Failed to create parrot agent",
			"error", err,
			"agent_type", agentTypeStr,
			"llm_available", s.LLMService != nil,
			"retriever_available", s.AdaptiveRetriever != nil,
		)
		return status.Errorf(codes.Internal, "failed to create agent: %v", err)
	}

	slog.Info("ChatWithParrot: Agent created successfully",
		"agent_type", agentTypeStr,
		"agent_name", parrotAgent.Name(),
	)

	// Create callback wrapper
	callback := func(eventType string, eventData interface{}) error {
		return streamAdapter.Send(eventType, eventData)
	}

	// Execute agent
	slog.Info("ChatWithParrot: Executing agent", "agent_type", agentTypeStr)
	if err := parrotAgent.ExecuteWithCallback(ctx, req.Message, req.History, callback); err != nil {
		slog.Error("Parrot agent execution failed",
			"error", err,
			"agent_type", agentTypeStr,
			"agent_name", parrotAgent.Name(),
		)
		return status.Errorf(codes.Internal, "agent execution failed: %v", err)
	}

	slog.Info("ChatWithParrot: Agent execution completed", "agent_type", agentTypeStr)

	// Send done marker
	streamMu.Lock()
	defer streamMu.Unlock()
	if err := stream.Send(&v1pb.ChatWithMemosResponse{
		Done: true,
	}); err != nil {
		return err
	}

	return nil
}

// chatWithStandardRAG handles standard RAG-based chat (fallback for DEFAULT agent type).
func (s *AIService) chatWithStandardRAG(
	ctx context.Context,
	req *v1pb.ChatWithMemosRequest,
	stream v1pb.AIService_ChatWithMemosServer,
	user *store.User,
) error {
	// Get user timezone from request
	userTimezone := req.UserTimezone
	if userTimezone == "" {
		userTimezone = "Asia/Shanghai"
	}

	// Parse timezone
	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		slog.Warn("Invalid timezone, using default", "timezone", userTimezone, "error", err)
		loc = time.FixedZone("UTC", 0)
	}

	// Create context for standard RAG chat
	ragCtx := &chatContext{
		userID:       user.ID,
		username:     user.Username,
		userEmail:    user.Email,
		userTimezone: userTimezone,
		messageCount: 0,
	}

	// Use query router to determine query type
	decision := s.QueryRouter.Route(ctx, req.Message, loc)
	queryType := decision.Strategy

	// Execute the appropriate query strategy
	results, err := s.executeRetrieval(ctx, req.Message, ragCtx.userID, queryType)
	if err != nil {
		slog.Error("Failed to execute retrieval", "error", err)
		return err
	}

	// Build prompt and stream response
	return s.streamChatResponse(ctx, req, stream, ragCtx, results, &queryengine.RouteDecision{Strategy: queryType})
}

// chatContext holds the context for a chat session.
type chatContext struct {
	userID       int32
	username     string
	userEmail    string
	userTimezone string
	messageCount int
}

// executeRetrieval executes the retrieval strategy based on query type.
func (s *AIService) executeRetrieval(
	ctx context.Context,
	query string,
	userID int32,
	queryType string,
) ([]*retrieval.SearchResult, error) {
	opts := &retrieval.RetrievalOptions{
		Query:    query,
		UserID:   userID,
		Strategy: queryType,
		Limit:    10,
		MinScore: 0.5,
	}

	return s.AdaptiveRetriever.Retrieve(ctx, opts)
}

// streamChatResponse streams the chat response based on retrieval results.
func (s *AIService) streamChatResponse(
	ctx context.Context,
	req *v1pb.ChatWithMemosRequest,
	stream v1pb.AIService_ChatWithMemosServer,
	_ *chatContext,
	results []*retrieval.SearchResult,
	_ *queryengine.RouteDecision,
) error {
	// Build context from retrieval results
	var context strings.Builder
	if len(results) > 0 {
		context.WriteString("相关笔记:\n")
		for i, r := range results {
			context.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Content))
		}
	}

	// Build system prompt - 使用统一的默认 Agent prompt
	systemPrompt := DefaultAgentSystemPrompt

	// Build messages for LLM
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context.String(), req.Message)},
	}

	// Use LLM to generate response
	response, err := s.LLMService.Chat(ctx, messages)
	if err != nil {
		return err
	}

	// Send content in chunks
	chunkSize := 100
	for i := 0; i < len(response); i += chunkSize {
		end := i + chunkSize
		if end > len(response) {
			end = len(response)
		}
		chunk := response[i:end]

		if err := stream.Send(&v1pb.ChatWithMemosResponse{
			Content: chunk,
		}); err != nil {
			return err
		}
	}

	// Send done marker
	return stream.Send(&v1pb.ChatWithMemosResponse{
		Done: true,
	})
}
