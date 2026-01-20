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
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/server/middleware"
	"github.com/usememos/memos/store"
)

// Global AI rate limiter
var globalAILimiter = middleware.NewRateLimiter()

// Pre-compiled regex patterns for schedule query intent detection
var scheduleQueryPatterns = []struct {
	patterns   []*regexp.Regexp
	intentType string
	timeRange  string
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


// AIService provides AI-powered features for memo management.
type AIService struct {
	v1pb.UnimplementedAIServiceServer

	Store *store.Store

	EmbeddingService ai.EmbeddingService
	RerankerService  ai.RerankerService
	LLMService       ai.LLMService
}

// IsEnabled returns whether AI features are enabled.
func (s *AIService) IsEnabled() bool {
	return s.EmbeddingService != nil
}

// getCurrentUser gets the authenticated user from context.
func getCurrentUser(ctx context.Context, st *store.Store) (*store.User, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("user not found in context")
	}
	user, err := st.GetUser(ctx, &store.FindUser{
		ID: &userID,
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user %d not found", userID)
	}
	return user, nil
}

// SemanticSearch performs semantic search on memos.
func (s *AIService) SemanticSearch(ctx context.Context, req *v1pb.SemanticSearchRequest) (*v1pb.SemanticSearchResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate parameters
	if req.Query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "query is required")
	}

	// Add input length validation
	const (
		maxQueryLength = 1000
		minQueryLength = 2
	)

	if len(req.Query) > maxQueryLength {
		return nil, status.Errorf(codes.InvalidArgument,
			"query too long: maximum %d characters, got %d", maxQueryLength, len(req.Query))
	}

	// Trim and check minimum length
	trimmedQuery := strings.TrimSpace(req.Query)
	if len(trimmedQuery) < minQueryLength {
		return nil, status.Errorf(codes.InvalidArgument,
			"query too short: minimum %d characters after trimming", minQueryLength)
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Vectorize the query
	queryVector, err := s.EmbeddingService.Embed(ctx, req.Query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to embed query: %v", err)
	}

	// Vector search (Top 10, optimized for 2C2G)
	results, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  10,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search: %v", err)
	}

	// Filter low relevance results (Threshold: 0.5)
	var filteredResults []*store.MemoWithScore
	for _, r := range results {
		if r.Score >= 0.5 {
			filteredResults = append(filteredResults, r)
		}
	}
	results = filteredResults

	if len(results) == 0 {
		return &v1pb.SemanticSearchResponse{Results: []*v1pb.SearchResult{}}, nil
	}

	// Re-rank (optional)
	if s.RerankerService.IsEnabled() && len(results) > limit {
		documents := make([]string, len(results))
		for i, r := range results {
			documents[i] = r.Memo.Content
		}

		rerankResults, err := s.RerankerService.Rerank(ctx, req.Query, documents, limit)
		if err == nil {
			// Reorder based on rerank results
			reordered := make([]*store.MemoWithScore, len(rerankResults))
			for i, rr := range rerankResults {
				reordered[i] = results[rr.Index]
				reordered[i].Score = rr.Score
			}
			results = reordered
		}
	}

	// Truncate results
	if len(results) > limit {
		results = results[:limit]
	}

	// Build response
	response := &v1pb.SemanticSearchResponse{
		Results: make([]*v1pb.SearchResult, len(results)),
	}

	for i, r := range results {
		snippet := r.Memo.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		response.Results[i] = &v1pb.SearchResult{
			Name:    fmt.Sprintf("memos/%s", r.Memo.UID),
			Snippet: snippet,
			Score:   r.Score,
		}
	}

	return response, nil
}

// SuggestTags suggests tags for memo content.
func (s *AIService) SuggestTags(ctx context.Context, req *v1pb.SuggestTagsRequest) (*v1pb.SuggestTagsResponse, error) {
	if s.LLMService == nil {
		return nil, status.Errorf(codes.Unavailable, "LLM features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if req.Content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content is required")
	}

	// Validate and set limit
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	// Get existing tags as reference
	existingTags, err := s.getExistingTags(ctx, user.ID)
	if err != nil {
		// Non-critical error, continue with empty tags
		existingTags = []string{}
	}

	// Build prompt
	prompt := fmt.Sprintf(`请为以下内容推荐 %d 个合适的标签。

## 内容
%s

## 已有标签（参考）
%s

## 要求
1. 每个标签不超过10个字符
2. 标签要准确反映内容主题
3. 优先使用已有标签列表中的标签
4. 只返回标签列表，每行一个，不要其他内容
`, limit, req.Content, strings.Join(existingTags, ", "))

	messages := []ai.Message{
		{Role: "user", Content: prompt},
	}

	// Call LLM
	response, err := s.LLMService.Chat(ctx, messages)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate tags: %v", err)
	}

	// Parse results
	tags := parseTagsFromLLM(response, limit)

	return &v1pb.SuggestTagsResponse{Tags: tags}, nil
}

// getExistingTags retrieves all tags from user's memos.
func (s *AIService) getExistingTags(ctx context.Context, userID int32) ([]string, error) {
	memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &userID,
	})
	if err != nil {
		return nil, err
	}

	tagSet := make(map[string]bool)
	for _, memo := range memos {
		if memo.Payload != nil {
			for _, tag := range memo.Payload.Tags {
				tagSet[tag] = true
			}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags, nil
}

// parseTagsFromLLM parses tags from LLM response.
func parseTagsFromLLM(response string, limit int) []string {
	lines := strings.Split(response, "\n")
	var tags []string

	for _, line := range lines {
		tag := strings.TrimSpace(line)
		tag = strings.TrimPrefix(tag, "-")
		tag = strings.TrimPrefix(tag, "#")
		tag = strings.TrimSpace(tag)

		if tag != "" && len(tag) <= 20 {
			tags = append(tags, tag)
			if len(tags) >= limit {
				break
			}
		}
	}

	return tags
}

// ChatWithMemos streams a chat response using memos as context.
func (s *AIService) ChatWithMemos(req *v1pb.ChatWithMemosRequest, stream v1pb.AIService_ChatWithMemosServer) error {
	ctx := stream.Context()

	// Debug: Log every AI chat request
	fmt.Printf("\n======== [ChatWithMemos] NEW REQUEST ========\n")
	fmt.Printf("[ChatWithMemos] User message: '%s'\n", req.Message)
	fmt.Printf("[ChatWithMemos] History items: %d\n", len(req.History))
	fmt.Printf("=============================================\n\n")

	if !s.IsEnabled() {
		return status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// 1. 获取当前用户
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// 1.5. 速率限制检查
	userKey := strconv.FormatInt(int64(user.ID), 10)
	if !globalAILimiter.Allow(userKey) {
		return status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded: please wait before making another AI chat request")
	}

	// 2. 参数校验
	if req.Message == "" {
		return status.Errorf(codes.InvalidArgument, "message is required")
	}

	// 3. 两阶段检索：初步回捞 + Reranker 重排序
	// Stage 1: 向量搜索初步回捞 (阈值 0.6，Top 20)
	queryVector, err := s.EmbeddingService.Embed(ctx, req.Message)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to embed query: %v", err)
	}

	results, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  20, // 初步回捞更多候选
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to search: %v", err)
	}

	// 4. 过滤低相关性结果 (阈值 0.6)
	var filteredResults []*store.MemoWithScore
	minScoreThreshold := float32(0.6)
	for _, r := range results {
		if r.Score >= minScoreThreshold {
			filteredResults = append(filteredResults, r)
		}
	}

	// Stage 2: Reranker 重排序提升精度
	if len(filteredResults) > 1 && s.RerankerService != nil && s.RerankerService.IsEnabled() {
		documents := make([]string, len(filteredResults))
		for i, r := range filteredResults {
			documents[i] = r.Memo.Content
		}

		rerankResults, err := s.RerankerService.Rerank(ctx, req.Message, documents, 5)
		if err == nil && len(rerankResults) > 0 {
			// 按重排序结果重新排列
			reordered := make([]*store.MemoWithScore, 0, len(rerankResults))
			for _, rr := range rerankResults {
				if rr.Index < len(filteredResults) {
					// 更新分数为 reranker 分数
					filteredResults[rr.Index].Score = rr.Score
					reordered = append(reordered, filteredResults[rr.Index])
				}
			}
			filteredResults = reordered
		}
	}

	// 4.5 检测日程查询意图并查询日程
	scheduleQueryIntent := s.detectScheduleQueryIntent(req.Message)
	var scheduleQueryResult *v1pb.ScheduleQueryResult

	fmt.Printf("[DEBUG] detectScheduleQueryIntent: message='%s', detected=%v\n",
		req.Message, scheduleQueryIntent.Detected)

	if scheduleQueryIntent.Detected {
		// 查询日程
		result, err := s.querySchedules(ctx, user.ID, scheduleQueryIntent)
		if err != nil {
			// 日程查询失败，记录错误
			fmt.Printf("[ScheduleQuery] Failed to query schedules: %v\n", err)
		} else {
			scheduleQueryResult = result
			fmt.Printf("[ScheduleQuery] Detected '%s' query, found %d schedules\n",
				scheduleQueryIntent.TimeRange, len(result.Schedules))
		}
	}

	// 5. 构建上下文 (最大字符数: 3000)
	var contextBuilder strings.Builder
	var sources []string
	totalChars := 0
	maxChars := 3000

	for i, r := range filteredResults {
		content := r.Memo.Content
		if totalChars+len(content) > maxChars {
			break
		}

		contextBuilder.WriteString(fmt.Sprintf("### 笔记 %d (相关度: %.0f%%)\n%s\n\n", i+1, r.Score*100, content))
		sources = append(sources, fmt.Sprintf("memos/%s", r.Memo.UID))
		totalChars += len(content)

		if len(sources) >= 5 {
			break // 最多使用 5 条笔记
		}
	}

	// 5.1 回退逻辑：如果没有匹配的笔记，使用所有搜索结果
	if len(sources) == 0 && len(results) > 0 {
		// 使用所有搜索结果（即使相关度低），因为用户可能在问通用问题
		for i, r := range results {
			content := r.Memo.Content
			if totalChars+len(content) > maxChars {
				break
			}
			contextBuilder.WriteString(fmt.Sprintf("### 笔记 %d\n%s\n\n", i+1, content))
			sources = append(sources, fmt.Sprintf("memos/%s", r.Memo.UID))
			totalChars += len(content)
			if len(sources) >= 5 {
				break
			}
		}
	}

	// 5. 构建 Prompt - 统一RAG架构
	// 核心思想：笔记和日程作为统一的RAG源，AI智能识别问题类型并组织回复

	var hasNotes = len(sources) > 0
	var hasSchedules = scheduleQueryResult != nil && len(scheduleQueryResult.Schedules) > 0

	// 构建智能回复的 system prompt
	var promptBuilder strings.Builder
	promptBuilder.WriteString(`你是一个基于用户个人笔记和日程的AI助手。

## 【核心】智能识别与回复策略

你将接收两类RAG数据：
1. 笔记数据（Memo）：用户记录的备忘录、文档、笔记
2. 日程数据（Schedule）：用户的日程安排、时间计划

### 问题类型识别（必须首先判断）

请分析用户问题属于以下哪种类型：

**类型1：纯日程查询**
特征：用户询问"近期日程"、"今天安排"、"明天有什么"等时间相关的问题
回复要求：
- 必须直接引用"【日程数据】"部分的内容
- 严格按照时间顺序列出日程
- 明确说明日程数量
- 格式："为您找到 N 个日程安排：1. [时间] [标题]..."

**类型2：纯笔记查询**
特征：用户询问备忘录内容、查找信息、"搜索..."等
回复要求：
- 基于"【笔记数据】"部分回答
- 引用相关的笔记内容
- 总结关键信息

**类型3：混合查询**
特征：用户同时涉及笔记和日程，如"我最近的工作安排和相关记录"
回复要求：
- 分别组织日程和笔记信息
- 使用清晰的分隔结构
- 先日程后笔记（或根据问题重点调整）

### 绝对禁止的错误（CRITICAL）

❌ **永远不要说**：
- "没有备忘录"（当有日程数据时）
- "没有日程"（当日程数据显示N>0时）
- "还没有任何数据"（当任何一个数据源有内容时）

✅ **正确的空数据处理**：
- 日程为空："暂无日程安排"
- 笔记为空："暂无相关笔记"

### 日程查询的正确回复示例

示例1 - 有日程：
用户："我近期的日程"
【日程数据】：共找到 3 个日程安排...
正确回复："为您找到 3 个近期的日程安排：
1. 2025-01-22 14:00: 开会 @ 会议室A
2. 2025-01-23 09:00: 产品评审
3. 2025-01-24 15:00: 客户会议"

示例2 - 无日程：
用户："我近期的日程"
【日程数据】：共找到 0 个日程安排
正确回复："暂无日程安排。需要帮您创建新的日程吗？"

示例3 - 混合场景：
用户："我最近的工作安排"
【日程数据】：共找到 2 个日程...
【笔记数据】：### 笔记1 (相关度: 95%) 工作计划...
正确回复："关于您最近的工作安排：

**日程安排**（2个）：
1. 2025-01-22 14:00: 团队会议
2. 2025-01-23 10:00: 项目评审

**相关笔记**：
- 工作计划笔记提到..."

## 日程创建检测（结构化输出）

当用户想创建日程时（关键词："创建"、"提醒"、"安排"、"添加"），必须在回复最后一行添加：

<<<SCHEDULE_INTENT:{"detected":true,"schedule_description":"自然语言描述"}>>>

注意：
- detected 必须为 true
- schedule_description 不能为空
- 字段名必须准确：schedule_description（不是 description）
- 查询日程时不要添加此标记

## 回答要求

1. **准确性**：只基于提供的RAG数据回答，不编造
2. **简洁性**：使用中文，简洁明了
3. **结构化**：使用列表、分隔线等组织信息
4. **真实性**：日程数据是准确的数据库查询结果，必须完全信任`)

	// 根据数据情况动态调整提示
	if !hasNotes && !hasSchedules {
		promptBuilder.WriteString("\n\n【当前状态】用户笔记和日程均为空，请引导用户创建内容。")
	} else if hasNotes && !hasSchedules {
		promptBuilder.WriteString("\n\n【当前状态】有笔记数据，无日程数据。")
	} else if !hasNotes && hasSchedules {
		promptBuilder.WriteString("\n\n【⚠️ 重要】有日程数据，无笔记数据！用户询问日程时，必须基于【日程数据】回答，禁止说'笔记中没有'。")
	} else {
		promptBuilder.WriteString("\n\n【当前状态】同时有笔记和日程数据，根据问题类型智能组织回复。")
	}

	messages := []ai.Message{
		{
			Role:    "system",
			Content: promptBuilder.String(),
		},
	}

	// 添加历史对话
	for i := 0; i < len(req.History)-1; i += 2 {
		if i+1 < len(req.History) {
			messages = append(messages, ai.Message{Role: "user", Content: req.History[i]})
			messages = append(messages, ai.Message{Role: "assistant", Content: req.History[i+1]})
		}
	}

	// 添加当前问题 - 统一的RAG上下文格式
	userMessageBuilder := &strings.Builder{}

	// 统一的RAG数据源格式
	userMessageBuilder.WriteString("============================================\n")
	userMessageBuilder.WriteString("【RAG数据源】笔记和日程的统一查询结果\n")
	userMessageBuilder.WriteString("============================================\n\n")

	// 策略调整：当有日程数据且无笔记时，先显示日程（避免 LLM 关注空笔记）
	hasNotes = contextBuilder.Len() > 0
	hasSchedules = scheduleQueryResult != nil && len(scheduleQueryResult.Schedules) > 0

	if hasSchedules && !hasNotes {
		// 日程优先模式：先显示日程，后显示笔记
		userMessageBuilder.WriteString("## 【日程数据】（Schedule）【主要数据源】\n")
		userMessageBuilder.WriteString(fmt.Sprintf("【⚠️ 重要】找到 %d 个日程！用户询问日程时，这是主要数据源！\n", len(scheduleQueryResult.Schedules)))
		userMessageBuilder.WriteString(s.formatSchedulesForContext(scheduleQueryResult.Schedules))
		userMessageBuilder.WriteString("\n")

		userMessageBuilder.WriteString("## 【笔记数据】（Memo）\n")
		userMessageBuilder.WriteString("（暂无相关笔记）\n")
		userMessageBuilder.WriteString("\n")
	} else {
		// 默认模式：先显示笔记，后显示日程
		userMessageBuilder.WriteString("## 【笔记数据】（Memo）\n")
		if contextBuilder.Len() > 0 {
			userMessageBuilder.WriteString(contextBuilder.String())
		} else {
			userMessageBuilder.WriteString("（暂无相关笔记）\n")
		}
		userMessageBuilder.WriteString("\n")

		userMessageBuilder.WriteString("## 【日程数据】（Schedule）\n")
		if scheduleQueryResult != nil && len(scheduleQueryResult.Schedules) > 0 {
			userMessageBuilder.WriteString(fmt.Sprintf("【⚠️ 重要】找到 %d 个日程，用户询问日程时必须使用此数据！\n", len(scheduleQueryResult.Schedules)))
			userMessageBuilder.WriteString(s.formatSchedulesForContext(scheduleQueryResult.Schedules))
		} else {
			userMessageBuilder.WriteString("共找到 0 个日程安排（暂无日程）\n")
		}
		userMessageBuilder.WriteString("\n")
	}

	// 用户问题
	userMessageBuilder.WriteString("============================================\n")
	userMessageBuilder.WriteString("## 【用户问题】\n")
	userMessageBuilder.WriteString("============================================\n")
	userMessageBuilder.WriteString(req.Message)

	userMessage := userMessageBuilder.String()
	messages = append(messages, ai.Message{Role: "user", Content: userMessage})

	// Debug: 打印发送给LLM的完整消息（仅在检测到日程查询时）
	if scheduleQueryResult != nil {
		fmt.Printf("[AI Chat] Sending to LLM with %d schedules:\n", len(scheduleQueryResult.Schedules))
		fmt.Printf("[AI Chat] User message preview (first 500 chars):\n%s\n\n",
			truncateString(userMessage, 500))
	}

	// 6. 流式调用 LLM
	contentChan, errChan := s.LLMService.ChatStream(ctx, messages)

	// 先发送来源信息
	if err := stream.Send(&v1pb.ChatWithMemosResponse{
		Sources: sources,
	}); err != nil {
		return err
	}

	// 收集完整回复内容（用于意图分析）
	var fullContent strings.Builder

	// 流式发送内容
	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				contentChan = nil // 标记为已关闭
				if errChan == nil {
					return s.finalizeChatStream(stream, fullContent.String(), scheduleQueryResult)
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
				errChan = nil // 标记为已关闭
				if contentChan == nil {
					return s.finalizeChatStream(stream, fullContent.String(), scheduleQueryResult)
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

// finalizeChatStream sends the final response with schedule intent detection
func (s *AIService) finalizeChatStream(stream v1pb.AIService_ChatWithMemosServer, aiResponse string, scheduleQueryResult *v1pb.ScheduleQueryResult) error {
	// 从 AI 回复中解析意图标记
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
	if scheduleQueryResult != nil {
		response.ScheduleQueryResult = scheduleQueryResult
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

	// 清理 JSON 字符串：移除所有空白字符（换行符、制表符、多余空格）
	cleanJSON := strings.ReplaceAll(jsonStr, "\n", "")
	cleanJSON = strings.ReplaceAll(cleanJSON, "\t", "")
	cleanJSON = strings.ReplaceAll(cleanJSON, " ", "")
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

// GetRelatedMemos finds memos related to a specific memo.
func (s *AIService) GetRelatedMemos(ctx context.Context, req *v1pb.GetRelatedMemosRequest) (*v1pb.GetRelatedMemosResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}

	// Parse memo UID from name (format: "memos/{uid}")
	var memoUID string
	if _, err := fmt.Sscanf(req.Name, "memos/%s", &memoUID); err != nil || memoUID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid name format, expected 'memos/{uid}'")
	}

	// Find the memo
	memo, err := s.Store.GetMemo(ctx, &store.FindMemo{
		UID:       &memoUID,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get memo: %v", err)
	}
	if memo == nil {
		return nil, status.Errorf(codes.NotFound, "memo not found")
	}

	// Get embedding for the memo
	embedding, err := s.Store.GetMemoEmbedding(ctx, memo.ID, "BAAI/bge-m3")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get memo embedding: %v", err)
	}

	var vector []float32
	if embedding == nil {
		// Generate embedding on-the-fly if not available
		vector, err = s.EmbeddingService.Embed(ctx, memo.Content)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate embedding: %v", err)
		}
		// Save the embedding for future use
		_, _ = s.Store.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
			MemoID:    memo.ID,
			Embedding: vector,
			Model:     "BAAI/bge-m3",
		})
	} else {
		vector = embedding.Embedding
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 5
	}

	// Vector search
	results, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: vector,
		Limit:  limit + 1, // +1 to exclude the original memo
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search: %v", err)
	}

	// Filter out the original memo and build response
	response := &v1pb.GetRelatedMemosResponse{
		Memos: []*v1pb.SearchResult{},
	}
	for _, r := range results {
		if r.Memo.ID == memo.ID {
			continue
		}
		snippet := r.Memo.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		response.Memos = append(response.Memos, &v1pb.SearchResult{
			Name:    fmt.Sprintf("memos/%s", r.Memo.UID),
			Snippet: snippet,
			Score:   r.Score,
		})
		if len(response.Memos) >= limit {
			break
		}
	}

	return response, nil
}

// ScheduleQueryIntent represents the detected intent for schedule query.
type ScheduleQueryIntent struct {
	Detected  bool
	QueryType string   // "upcoming", "range", "filter"
	TimeRange string   // "7d", "today", "tomorrow", "week"
	StartTime *time.Time
	EndTime   *time.Time
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

// querySchedules queries schedules based on the detected intent.
func (s *AIService) querySchedules(ctx context.Context, userID int32, intent *ScheduleQueryIntent) (*v1pb.ScheduleQueryResult, error) {
	// Default to 7 days starting from today 00:00:00 if no time range specified
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endTime := startTime.Add(7 * 24 * time.Hour)

	if intent.StartTime != nil {
		startTime = *intent.StartTime
	}
	if intent.EndTime != nil {
		endTime = *intent.EndTime
	}

	// Convert time to timestamps for query
	startTs := startTime.Unix()
	endTs := endTime.Unix()

	// Debug logging
	fmt.Printf("[DEBUG] querySchedules: userID=%d, intent.TimeRange=%s\n", userID, intent.TimeRange)
	fmt.Printf("[DEBUG] querySchedules: startTime=%s -> startTs=%d\n", startTime.Format("2006-01-02 15:04:05 MST"), startTs)
	fmt.Printf("[DEBUG] querySchedules: endTime=%s -> endTs=%d\n", endTime.Format("2006-01-02 15:04:05 MST"), endTs)

	// Query schedules from store
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   &startTs,
		EndTs:     &endTs,
	}

	schedules, err := s.Store.ListSchedules(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	fmt.Printf("[DEBUG] querySchedules: found %d schedules\n", len(schedules))

	// Convert to proto format
	result := &v1pb.ScheduleQueryResult{
		Detected:             true,
		Schedules:            make([]*v1pb.ScheduleSummary, 0, len(schedules)),
		TimeRangeDescription: intent.TimeRange,
		QueryType:            intent.QueryType,
	}

	// Limit results to avoid overwhelming the user
	maxResults := 10
	if len(schedules) > maxResults {
		schedules = schedules[:maxResults]
	}

	for _, sched := range schedules {
		// Convert end_ts (can be nil for all-day events)
		var endTsValue int64
		if sched.EndTs != nil {
			endTsValue = *sched.EndTs
		}

		// Map RowStatus to status string
		status := "ACTIVE"
		if sched.RowStatus == store.Archived {
			status = "CANCELLED"
		}

		// Handle RecurrenceRule (can be nil)
		var recurrenceRule string
		if sched.RecurrenceRule != nil {
			recurrenceRule = *sched.RecurrenceRule
		}

		result.Schedules = append(result.Schedules, &v1pb.ScheduleSummary{
			Uid:            sched.UID,
			Title:          sched.Title,
			StartTs:        sched.StartTs,
			EndTs:          endTsValue,
			AllDay:         sched.AllDay,
			Location:       sched.Location,
			RecurrenceRule:  recurrenceRule,
			Status:         status,
		})
	}

	return result, nil
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

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
