package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

	// Register ScheduleAgent service handlers if available
	if s.ScheduleAgentService != nil {
		handlers = append(handlers, wrap(apiv1connect.NewScheduleAgentServiceHandler(s, opts...)))
	}

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
			slog.Warn("Invalid timezone, using UTC", "timezone", req.Msg.UserTimezone, "error", err)
			userTimezone = time.UTC
		}
	} else {
		userTimezone = time.UTC
	}

	if s.AIService.QueryRouter != nil {
		routeDecision = s.AIService.QueryRouter.Route(ctx, req.Msg.Message, userTimezone)
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
			slog.Warn("AdaptiveRetriever error, using fallback", "error", err)
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

	slog.Debug("Retrieval completed", "results_count", len(searchResults))

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

// ChatWithScheduleAgent streams a chat response using the schedule agent.
func (s *ConnectServiceHandler) ChatWithScheduleAgent(ctx context.Context, req *connect.Request[v1pb.ChatWithMemosRequest], stream *connect.ServerStream[v1pb.ChatWithMemosResponse]) error {
	if s.ScheduleAgentService == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("schedule agent service is not available"))
	}

	// Create schedule agent request
	agentReq := &v1pb.ScheduleAgentChatRequest{
		Message:      req.Msg.Message,
		UserTimezone: req.Msg.UserTimezone,
	}

	// Create a stream adapter that wraps the Connect stream
	grpcStream := &chatStreamToScheduleAgentAdapter{
		connectStream: stream,
		ctx:           ctx,
	}

	// Call the schedule agent's streaming implementation
	return s.ScheduleAgentService.ChatStream(agentReq, grpcStream)
}

// ChatWithMemosIntegrated integrates both RAG and schedule agent.
// For now, this is an alias to ChatWithMemos (RAG only).
func (s *ConnectServiceHandler) ChatWithMemosIntegrated(ctx context.Context, req *connect.Request[v1pb.ChatWithMemosRequest], stream *connect.ServerStream[v1pb.ChatWithMemosResponse]) error {
	// TODO: Implement true integration with schedule agent
	// For now, just use the existing ChatWithMemos implementation
	return s.ChatWithMemos(ctx, req, stream)
}

// chatStreamToScheduleAgentAdapter adapts Connect ChatWithMemosResponse stream to ScheduleAgentStreamResponse
type chatStreamToScheduleAgentAdapter struct {
	connectStream *connect.ServerStream[v1pb.ChatWithMemosResponse]
	ctx           context.Context
}

func (a *chatStreamToScheduleAgentAdapter) Context() context.Context {
	return a.ctx
}

func (a *chatStreamToScheduleAgentAdapter) Send(resp *v1pb.ScheduleAgentStreamResponse) error {
	// Convert ScheduleAgentStreamResponse to ChatWithMemosResponse
	chatResp := &v1pb.ChatWithMemosResponse{
		Content: resp.Content,
		Done:    resp.Done,
		// Sources field doesn't exist in ScheduleAgentStreamResponse
		// The agent response in Content field should contain all necessary information
	}
	return a.connectStream.Send(chatResp)
}

func (a *chatStreamToScheduleAgentAdapter) SendMsg(m any) error {
	if resp, ok := m.(*v1pb.ScheduleAgentStreamResponse); ok {
		return a.Send(resp)
	}
	return fmt.Errorf("invalid message type: %T", m)
}

func (a *chatStreamToScheduleAgentAdapter) RecvMsg(m any) error {
	return fmt.Errorf("RecvMsg not supported for server streaming")
}

func (a *chatStreamToScheduleAgentAdapter) SetHeader(md metadata.MD) error {
	return nil
}

func (a *chatStreamToScheduleAgentAdapter) SendHeader(md metadata.MD) error {
	return nil
}

func (a *chatStreamToScheduleAgentAdapter) SetTrailer(md metadata.MD) {
	// Connect doesn't support gRPC metadata trailers
}

// ScheduleAgentService wrappers for Connect

// Chat handles non-streaming schedule agent chat requests.
func (s *ConnectServiceHandler) Chat(ctx context.Context, req *connect.Request[v1pb.ScheduleAgentChatRequest]) (*connect.Response[v1pb.ScheduleAgentChatResponse], error) {
	if s.ScheduleAgentService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("schedule agent service is not available"))
	}

	resp, err := s.ScheduleAgentService.Chat(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// ChatStream handles streaming schedule agent chat requests.
func (s *ConnectServiceHandler) ChatStream(ctx context.Context, req *connect.Request[v1pb.ScheduleAgentChatRequest], stream *connect.ServerStream[v1pb.ScheduleAgentStreamResponse]) error {
	if s.ScheduleAgentService == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("schedule agent service is not available"))
	}

	// Convert Connect stream to gRPC stream interface
	grpcStream := &scheduleAgentStreamAdapter{
		connectStream: stream,
		ctx:           ctx,
	}

	// Call the gRPC streaming implementation
	return s.ScheduleAgentService.ChatStream(req.Msg, grpcStream)
}

// scheduleAgentStreamAdapter adapts Connect ServerStream to gRPC ScheduleAgentService_ChatStreamServer
type scheduleAgentStreamAdapter struct {
	connectStream *connect.ServerStream[v1pb.ScheduleAgentStreamResponse]
	ctx           context.Context
}

func (a *scheduleAgentStreamAdapter) Context() context.Context {
	return a.ctx
}

func (a *scheduleAgentStreamAdapter) Send(resp *v1pb.ScheduleAgentStreamResponse) error {
	return a.connectStream.Send(resp)
}

func (a *scheduleAgentStreamAdapter) SendMsg(m any) error {
	if resp, ok := m.(*v1pb.ScheduleAgentStreamResponse); ok {
		return a.connectStream.Send(resp)
	}
	return fmt.Errorf("invalid message type: %T", m)
}

func (a *scheduleAgentStreamAdapter) RecvMsg(m any) error {
	// Server-side streaming doesn't receive messages from client after initial request
	return fmt.Errorf("RecvMsg not supported for server streaming")
}

func (a *scheduleAgentStreamAdapter) SetHeader(md metadata.MD) error {
	// Connect doesn't support gRPC metadata headers
	return nil
}

func (a *scheduleAgentStreamAdapter) SendHeader(md metadata.MD) error {
	// Connect doesn't support gRPC metadata headers
	return nil
}

func (a *scheduleAgentStreamAdapter) SetTrailer(md metadata.MD) {
	// Connect doesn't support gRPC metadata trailers
}

// GetParrotSelfCognition returns the metacognitive information of a parrot agent.
func (s *ConnectServiceHandler) GetParrotSelfCognition(ctx context.Context, req *connect.Request[v1pb.GetParrotSelfCognitionRequest]) (*connect.Response[v1pb.GetParrotSelfCognitionResponse], error) {
	agentType := req.Msg.GetAgentType()
	selfCognition := getParrotSelfCognition(agentType)

	return connect.NewResponse(&v1pb.GetParrotSelfCognitionResponse{
		SelfCognition: selfCognition,
	}), nil
}

// ListParrots returns all available parrot agents with their metacognitive information.
func (s *ConnectServiceHandler) ListParrots(ctx context.Context, req *connect.Request[v1pb.ListParrotsRequest]) (*connect.Response[v1pb.ListParrotsResponse], error) {
	// Return all available parrot types
	agentTypes := []v1pb.AgentType{
		v1pb.AgentType_AGENT_TYPE_DEFAULT,
		v1pb.AgentType_AGENT_TYPE_MEMO,
		v1pb.AgentType_AGENT_TYPE_SCHEDULE,
		v1pb.AgentType_AGENT_TYPE_AMAZING,
		v1pb.AgentType_AGENT_TYPE_CREATIVE,
	}

	parrots := make([]*v1pb.ParrotInfo, 0, len(agentTypes))
	for _, agentType := range agentTypes {
		parrots = append(parrots, &v1pb.ParrotInfo{
			AgentType:     agentType,
			Name:          getParrotNameByAgentType(agentType),
			SelfCognition: getParrotSelfCognition(agentType),
		})
	}

	return connect.NewResponse(&v1pb.ListParrotsResponse{
		Parrots: parrots,
	}), nil
}

// Helper function to get parrot self-cognition by agent type
func getParrotSelfCognition(agentType v1pb.AgentType) *v1pb.ParrotSelfCognition {
	switch agentType {
	case v1pb.AgentType_AGENT_TYPE_MEMO:
		return &v1pb.ParrotSelfCognition{
			Name:             "memo",
			Emoji:            "🦜",
			Title:            "灰灰 - 笔记助手鹦鹉",
			Personality:      []string{"专注", "善于总结", "记忆力强"},
			Capabilities:     []string{"memo_search", "memo_summary", "memo_analysis"},
			Limitations:      []string{"不能直接修改笔记", "不能访问外部信息"},
			WorkingStyle:     "先理解问题，检索相关笔记，然后综合分析给出答案",
			FavoriteTools:    []string{"semantic_search", "memo_query"},
			SelfIntroduction: "我是灰灰，您的笔记助手。我擅长在您的笔记中搜索信息、总结内容和发现关联。",
			FunFact:          "我是一只非洲灰鹦鹉，以记忆力和智慧著称",
		}
	case v1pb.AgentType_AGENT_TYPE_SCHEDULE:
		return &v1pb.ParrotSelfCognition{
			Name:             "schedule",
			Emoji:            "📅",
			Title:            "金刚 - 日程助手鹦鹉",
			Personality:      []string{"守时", "条理清晰", "注重计划"},
			Capabilities:     []string{"schedule_query", "schedule_create", "schedule_manage"},
			Limitations:      []string{"不能代替您做决定", "不能访问外部日历"},
			WorkingStyle:     "分析时间需求，查询现有日程，帮助安排和提醒",
			FavoriteTools:    []string{"schedule_list", "schedule_create", "conflict_check"},
			SelfIntroduction: "我是金刚，您的日程助手。我帮您管理时间、安排日程、避免冲突。",
			FunFact:          "我是一只蓝黄金刚鹦鹉，以守时和可靠著称",
		}
	case v1pb.AgentType_AGENT_TYPE_AMAZING:
		return &v1pb.ParrotSelfCognition{
			Name:             "amazing",
			Emoji:            "⭐",
			Title:            "惊奇 - 综合助手鹦鹉",
			Personality:      []string{"全能", "灵活", "善于整合"},
			Capabilities:     []string{"memo_search", "schedule_query", "integrated_analysis"},
			Limitations:      []string{"复杂任务可能需要专门助手"},
			WorkingStyle:     "综合分析笔记和日程，提供全面的视角和建议",
			FavoriteTools:    []string{"memo_search", "schedule_query", "combined_analysis"},
			SelfIntroduction: "我是惊奇，您的综合助手。我能同时查看您的笔记和日程，给您完整的信息。",
			FunFact:          "我是一只亚马逊鹦鹉，以多才多艺著称",
		}
	case v1pb.AgentType_AGENT_TYPE_CREATIVE:
		return &v1pb.ParrotSelfCognition{
			Name:             "creative",
			Emoji:            "💡",
			Title:            "灵灵 - 创意助手鹦鹉",
			Personality:      []string{"创意", "活泼", "善于表达"},
			Capabilities:     []string{"creative_writing", "brainstorm", "text_improvement"},
			Limitations:      []string{"创意建议需要您的判断", "不能保证所有想法都适用"},
			WorkingStyle:     "激发创意思维，提供多种表达方式，帮助完善文字",
			FavoriteTools:    []string{"idea_generation", "text_polish", "style_transform"},
			SelfIntroduction: "我是灵灵，您的创意伙伴。我帮您头脑风暴、改进文字、激发灵感。",
			FunFact:          "我是一只虎皮鹦鹉，以活泼和创造力著称",
		}
	default:
		return &v1pb.ParrotSelfCognition{
			Name:             "default",
			Emoji:            "🤖",
			Title:            "默认助手",
			Personality:      []string{"通用", "友好", "乐于助人"},
			Capabilities:     []string{"memo_search", "memo_summary", "general_qa"},
			Limitations:      []string{"通用能力，专业任务建议使用专门助手"},
			WorkingStyle:     "理解问题，搜索相关信息，提供帮助",
			FavoriteTools:    []string{"search", "analyze"},
			SelfIntroduction: "我是您的 AI 助手，随时准备帮助您。",
			FunFact:          "我是默认助手，什么都会一点",
		}
	}
}

// Helper function to get parrot name by agent type
func getParrotNameByAgentType(agentType v1pb.AgentType) string {
	switch agentType {
	case v1pb.AgentType_AGENT_TYPE_MEMO:
		return "灰灰"
	case v1pb.AgentType_AGENT_TYPE_SCHEDULE:
		return "金刚"
	case v1pb.AgentType_AGENT_TYPE_AMAZING:
		return "惊奇"
	case v1pb.AgentType_AGENT_TYPE_CREATIVE:
		return "灵灵"
	default:
		return "默认助手"
	}
}
