package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/proto/gen/api/v1/apiv1connect"
	"github.com/usememos/memos/server/queryengine"
	"github.com/usememos/memos/server/retrieval"
	"github.com/usememos/memos/server/timezone"
	"github.com/usememos/memos/store"
)

// ConnectServiceHandler wraps APIV1Service to implement Connect handler interfaces.
// It adapts the existing gRPC service implementations to work with Connect's
// request/response wrapper types.
//
// This wrapper pattern allows us to:
// - Reuse existing gRPC service implementations
// - Support both native gRPC and Connect protocols
// - Maintain a single source of truth for business logic.
type ConnectServiceHandler struct {
	*APIV1Service
}

// NewConnectServiceHandler creates a new Connect service handler.
func NewConnectServiceHandler(svc *APIV1Service) *ConnectServiceHandler {
	return &ConnectServiceHandler{APIV1Service: svc}
}

// RegisterConnectHandlers registers all Connect service handlers on the given mux.
func (s *ConnectServiceHandler) RegisterConnectHandlers(mux *http.ServeMux, opts ...connect.HandlerOption) {
	// Register all service handlers
	handlers := []struct {
		path    string
		handler http.Handler
	}{
		wrap(apiv1connect.NewInstanceServiceHandler(s, opts...)),
		wrap(apiv1connect.NewAuthServiceHandler(s, opts...)),
		wrap(apiv1connect.NewUserServiceHandler(s, opts...)),
		wrap(apiv1connect.NewMemoServiceHandler(s, opts...)),
		wrap(apiv1connect.NewAttachmentServiceHandler(s, opts...)),
		wrap(apiv1connect.NewShortcutServiceHandler(s, opts...)),
		wrap(apiv1connect.NewActivityServiceHandler(s, opts...)),
		wrap(apiv1connect.NewIdentityProviderServiceHandler(s, opts...)),
	}

	if s.AIService != nil {
		handlers = append(handlers, wrap(apiv1connect.NewAIServiceHandler(s, opts...)))
	}

	// Register Schedule service handlers
	handlers = append(handlers, wrap(apiv1connect.NewScheduleServiceHandler(s, opts...)))

	for _, h := range handlers {
		mux.Handle(h.path, h.handler)
	}
}

// wrap converts (path, handler) return value to a struct for cleaner iteration.
func wrap(path string, handler http.Handler) struct {
	path    string
	handler http.Handler
} {
	return struct {
		path    string
		handler http.Handler
	}{path, handler}
}

// convertGRPCError converts gRPC status errors to Connect errors.
// This preserves the error code semantics between the two protocols.
func convertGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return connect.NewError(grpcCodeToConnectCode(st.Code()), err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// grpcCodeToConnectCode converts gRPC status codes to Connect error codes.
// gRPC and Connect use the same error code semantics, so this is a direct cast.
// See: https://connectrpc.com/docs/protocol/#error-codes
func grpcCodeToConnectCode(code codes.Code) connect.Code {
	return connect.Code(code)
}

// AIService wrappers for Connect

func (s *ConnectServiceHandler) SuggestTags(ctx context.Context, req *connect.Request[v1pb.SuggestTagsRequest]) (*connect.Response[v1pb.SuggestTagsResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.SuggestTags(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) SemanticSearch(ctx context.Context, req *connect.Request[v1pb.SemanticSearchRequest]) (*connect.Response[v1pb.SemanticSearchResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.SemanticSearch(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetRelatedMemos(ctx context.Context, req *connect.Request[v1pb.GetRelatedMemosRequest]) (*connect.Response[v1pb.GetRelatedMemosResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.GetRelatedMemos(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ChatWithMemos(ctx context.Context, req *connect.Request[v1pb.ChatWithMemosRequest], stream *connect.ServerStream[v1pb.ChatWithMemosResponse]) error {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}

	// 1. 获取当前用户
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthorized"))
	}
	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthorized"))
	}

	// 2. 参数校验
	if req.Msg.Message == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message is required"))
	}

	// ============================================================
	// Phase 1: 智能 Query Routing（⭐ 新增）
	// ============================================================
	var routeDecision *queryengine.RouteDecision

	// 解析用户时区
	var userTimezone *time.Location
	if req.Msg.UserTimezone != "" {
		var err error
		userTimezone, err = time.LoadLocation(req.Msg.UserTimezone)
		if err != nil {
			fmt.Printf("[ChatWithMemos] Invalid timezone %q, using UTC: %v\n", req.Msg.UserTimezone, err)
			userTimezone = time.UTC
		}
	} else {
		userTimezone = time.UTC
	}

	if s.AIService.QueryRouter != nil {
		routeDecision = s.AIService.QueryRouter.Route(ctx, req.Msg.Message, userTimezone)
		fmt.Printf("[QueryRouting] Strategy: %s, Confidence: %.2f, Timezone: %v\n",
			routeDecision.Strategy, routeDecision.Confidence, userTimezone)
	} else {
		// 降级：默认策略
		routeDecision = &queryengine.RouteDecision{
			Strategy:      "hybrid_standard",
			Confidence:    0.80,
			SemanticQuery: req.Msg.Message,
			NeedsReranker: false,
		}
	}

	// ============================================================
	// Phase 2: Adaptive Retrieval（⭐ 新增）
	// ============================================================
	var searchResults []*retrieval.SearchResult
	if s.AIService.AdaptiveRetriever != nil {
		// 使用新的自适应检索器
		searchResults, err = s.AIService.AdaptiveRetriever.Retrieve(ctx, &retrieval.RetrievalOptions{
			Query:     req.Msg.Message,
			UserID:    user.ID,
			Strategy:  routeDecision.Strategy,
			TimeRange: routeDecision.TimeRange,
			MinScore:  0.5,
			Limit:     10,
		})
		if err != nil {
			fmt.Printf("[AdaptiveRetriever] Error: %v, using fallback\n", err)
			// 降级到旧逻辑
			searchResults, err = s.fallbackRetrieval(ctx, user.ID, req.Msg.Message)
			if err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("retrieval failed: %v", err))
			}
		}
	} else {
		// 降级到旧逻辑
		searchResults, err = s.fallbackRetrieval(ctx, user.ID, req.Msg.Message)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("retrieval failed: %v", err))
		}
	}

	fmt.Printf("[Retrieval] Found %d results\n", len(searchResults))

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
	// Phase 3: 构建上下文（⭐ 支持日程）
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

	// ⭐ 新增：添加日程到上下文
	if len(scheduleResults) > 0 {
		contextBuilder.WriteString("### 📅 日程安排\n")
		for i, r := range scheduleResults {
			if r.Schedule != nil {
				// 使用 timezone 包格式化日程时间（完整日期时间）
				timeStr := timezone.FormatScheduleTime(
					r.Schedule.StartTs,
					r.Schedule.EndTs,
					r.Schedule.AllDay,
					userTimezone,
				)
				contextBuilder.WriteString(fmt.Sprintf("%d. %s - %s", i+1, timeStr, r.Schedule.Title))
				if r.Schedule.Location != "" {
					contextBuilder.WriteString(fmt.Sprintf(" @ %s", r.Schedule.Location))
				}
				contextBuilder.WriteString("\n")
				// ⭐ 添加日程到 sources
				sources = append(sources, fmt.Sprintf("schedules/%d", r.Schedule.ID))
			}
		}
		contextBuilder.WriteString("\n")
	}

	// ============================================================
	// Phase 4: 构建提示词（⭐ 优化）
	// ============================================================
	var hasNotes = len(memoResults) > 0
	var hasSchedules = len(scheduleResults) > 0

	messages := s.buildOptimizedMessagesForConnect(
		req.Msg.Message,
		req.Msg.History,
		contextBuilder.String(),
		scheduleResults,
		hasNotes,
		hasSchedules,
	)

	// ============================================================
	// Phase 5: 流式调用 LLM
	// ============================================================
	contentChan, errChan := s.AIService.LLMService.ChatStream(ctx, messages)

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
					// 流结束，发送最终响应
					return s.sendFinalResponse(stream, fullContent.String(), scheduleResults)
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
					// 流结束，发送最终响应
					return s.sendFinalResponse(stream, fullContent.String(), scheduleResults)
				}
				continue
			}
			if err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("LLM error: %v", err))
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// sendFinalResponse 发送最终响应（包含 Done、ScheduleQueryResult 等）
func (s *ConnectServiceHandler) sendFinalResponse(
	stream *connect.ServerStream[v1pb.ChatWithMemosResponse],
	aiResponse string,
	scheduleResults []*retrieval.SearchResult,
) error {
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

// parseScheduleIntentFromAIResponse 从 AI 响应中解析日程创建意图
// 复用 ai_service_chat.go 中的逻辑
func (s *ConnectServiceHandler) parseScheduleIntentFromAIResponse(aiResponse string) *v1pb.ScheduleCreationIntent {
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

// fallbackRetrieval 降级检索逻辑（兼容旧版本）
func (s *ConnectServiceHandler) fallbackRetrieval(ctx context.Context, userID int32, query string) ([]*retrieval.SearchResult, error) {
	// 简化的向量检索
	queryVector, err := s.AIService.EmbeddingService.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	vectorResults, err := s.AIService.Store.VectorSearch(ctx, &store.VectorSearchOptions{
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

// buildOptimizedMessagesForConnect 构建优化后的消息（支持日程）
func (s *ConnectServiceHandler) buildOptimizedMessagesForConnect(
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

## 日程查询
当用户查询时间范围的日程时（如"今天"、"本周"）：
1. **优先回复日程信息**
2. 格式：时间 - 标题 (@地点)
3. 如果没有日程，明确告知"暂无日程"

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

	// ⭐ 添加日程上下文
	if hasSchedules {
		userMsgBuilder.WriteString("### 📅 日程安排\n")
		for i, r := range scheduleResults {
			if r.Schedule != nil {
				scheduleTime := time.Unix(r.Schedule.StartTs, 0)
				timeStr := scheduleTime.Format("15:04")
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

// ScheduleService wrappers for Connect

func (s *ConnectServiceHandler) CreateSchedule(ctx context.Context, req *connect.Request[v1pb.CreateScheduleRequest]) (*connect.Response[v1pb.Schedule], error) {
	resp, err := s.ScheduleService.CreateSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListSchedules(ctx context.Context, req *connect.Request[v1pb.ListSchedulesRequest]) (*connect.Response[v1pb.ListSchedulesResponse], error) {
	resp, err := s.ScheduleService.ListSchedules(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetSchedule(ctx context.Context, req *connect.Request[v1pb.GetScheduleRequest]) (*connect.Response[v1pb.Schedule], error) {
	resp, err := s.ScheduleService.GetSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateSchedule(ctx context.Context, req *connect.Request[v1pb.UpdateScheduleRequest]) (*connect.Response[v1pb.Schedule], error) {
	resp, err := s.ScheduleService.UpdateSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteSchedule(ctx context.Context, req *connect.Request[v1pb.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.ScheduleService.DeleteSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CheckConflict(ctx context.Context, req *connect.Request[v1pb.CheckConflictRequest]) (*connect.Response[v1pb.CheckConflictResponse], error) {
	resp, err := s.ScheduleService.CheckConflict(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ParseAndCreateSchedule(ctx context.Context, req *connect.Request[v1pb.ParseAndCreateScheduleRequest]) (*connect.Response[v1pb.ParseAndCreateScheduleResponse], error) {
	resp, err := s.ScheduleService.ParseAndCreateSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}
