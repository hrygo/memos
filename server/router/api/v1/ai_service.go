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
			regexp.MustCompile("近期日程"),
			regexp.MustCompile("近期的日程"),
			regexp.MustCompile("未来.*日程"),
			regexp.MustCompile("接下来.*日程"),
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
	var scheduleContext string

	if scheduleQueryIntent.Detected {
		// 查询日程
		result, err := s.querySchedules(ctx, user.ID, scheduleQueryIntent)
		if err != nil {
			// 日程查询失败，记录错误并在上下文中告知 AI
			fmt.Printf("[ScheduleQuery] Failed to query schedules: %v\n", err)
			scheduleContext = fmt.Sprintf("[日程查询失败: %v]", err)
		} else {
			scheduleQueryResult = result
			// 格式化日程信息用于 AI 上下文
			scheduleContext = s.formatSchedulesForContext(result.Schedules)
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

	// 5. 构建 Prompt
	var systemPrompt string
	if len(sources) == 0 {
		// 没有任何笔记时，明确告知
		systemPrompt = `你是一个基于用户个人笔记和日程的AI助手。

## 功能
1. 笔记查询：基于用户笔记回答问题
2. 日程查询：查询用户近期日程安排
3. 日程创建：帮助用户创建新日程

## 日程意图检测（重要）
如果检测到用户想创建日程/提醒（用户说"帮我创建"、"提醒我"、"安排"等），在回复的最后独立成行添加：
<<<SCHEDULE_INTENT:{"detected":true,"description":"简短描述，如：明天下午2点开会"}>>>

如果用户只是询问日程但没有创建意图，不要添加此标记。

## 回答要求
- 使用中文，简洁准确
- 如果用户询问日程，基于下方"用户日程"部分回答
- 如果"用户日程"部分显示"日程查询失败"，请告知用户查询失败并建议稍后重试
- 如果没有日程或笔记，友好告知用户`
	} else {
		systemPrompt = `你是一个基于用户个人笔记和日程的AI助手。

## 功能
1. 笔记查询：基于用户笔记回答问题
2. 日程查询：查询用户近期日程安排
3. 日程创建：帮助用户创建新日程

## 日程意图检测（重要）
如果检测到用户想创建日程/提醒（用户说"帮我创建"、"提醒我"、"安排"等），在回复的最后独立成行添加：
<<<SCHEDULE_INTENT:{"detected":true,"description":"简短描述，如：明天下午2点开会"}>>>

如果用户只是询问日程但没有创建意图，不要添加此标记。

## 回答要求
- 优先基于笔记和日程回答
- 如果用户询问日程，基于下方"用户日程"部分回答
- 如果"用户日程"部分显示"日程查询失败"，请告知用户查询失败并建议稍后重试
- 使用中文，简洁准确
- 不要编造信息`
	}
	messages := []ai.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// 添加历史对话
	for i := 0; i < len(req.History)-1; i += 2 {
		if i+1 < len(req.History) {
			messages = append(messages, ai.Message{Role: "user", Content: req.History[i]})
			messages = append(messages, ai.Message{Role: "assistant", Content: req.History[i+1]})
		}
	}

	// 添加当前问题
	userMessageBuilder := &strings.Builder{}
	userMessageBuilder.WriteString("## 相关笔记\n")
	userMessageBuilder.WriteString(contextBuilder.String())
	userMessageBuilder.WriteString("\n")

	// 添加日程上下文（如果有）
	if scheduleContext != "" {
		userMessageBuilder.WriteString("## 用户日程\n")
		userMessageBuilder.WriteString(scheduleContext)
		userMessageBuilder.WriteString("\n")
	}

	userMessageBuilder.WriteString("## 用户问题\n")
	userMessageBuilder.WriteString(req.Message)

	userMessage := userMessageBuilder.String()
	messages = append(messages, ai.Message{Role: "user", Content: userMessage})

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
// This replaces the additional LLM call approach for better performance
// Marker format: <<<SCHEDULE_INTENT:{"detected":true,"description":"..."}>>>
func (s *AIService) parseScheduleIntentFromAIResponse(aiResponse string) *v1pb.ScheduleCreationIntent {
	// 查找意图标记：使用更独特的 <<<SCHEDULE_INTENT: 格式避免误判
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

	// 清理 JSON 字符串：移除换行符和制表符
	cleanJSON := strings.ReplaceAll(jsonStr, "\n", " ")
	cleanJSON = strings.ReplaceAll(cleanJSON, "\t", " ")
	cleanJSON = strings.TrimSpace(cleanJSON)

	// 解析 JSON
	type IntentJSON struct {
		Detected   bool   `json:"detected"`
		Description string `json:"description"`
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

	// 验证描述不为空
	if strings.TrimSpace(intentJSON.Description) == "" {
		fmt.Printf("[ScheduleIntent] Intent detected but description is empty\n")
		return nil
	}

	fmt.Printf("[ScheduleIntent] Detected from AI response: %s\n", intentJSON.Description)

	return &v1pb.ScheduleCreationIntent{
		Detected:            true,
		ScheduleDescription: intentJSON.Description,
		Reasoning:           "Detected from AI response marker",
	}
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
		return "无"
	}

	var builder strings.Builder
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

		builder.WriteString(fmt.Sprintf("%d. %s: %s%s%s\n", i+1, timeStr, sched.Title, location, recurrence))
	}

	return builder.String()
}
