package v1

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/proto/gen/api/v1/apiv1connect"
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

	// 1. Get current user
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthorized"))
	}
	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthorized"))
	}

	// 2. Validate parameters
	if req.Msg.Message == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message is required"))
	}

	// 3. 两阶段检索：初步回捞 + Reranker 重排序
	// Stage 1: 向量搜索初步回捞 (阈值 0.6，Top 20)
	queryVector, err := s.AIService.EmbeddingService.Embed(ctx, req.Msg.Message)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to embed query: %v", err))
	}

	results, err := s.AIService.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  20, // 初步回捞更多候选
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to search: %v", err))
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
	if len(filteredResults) > 1 && s.AIService.RerankerService != nil && s.AIService.RerankerService.IsEnabled() {
		documents := make([]string, len(filteredResults))
		for i, r := range filteredResults {
			documents[i] = r.Memo.Content
		}

		rerankResults, err := s.AIService.RerankerService.Rerank(ctx, req.Msg.Message, documents, 5)
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

	// 5. Build Prompt
	var systemPrompt string
	if len(sources) == 0 {
		// 没有任何笔记时，明确告知
		systemPrompt = "你是一个基于用户个人笔记的AI助手。当前用户没有任何备忘录，请友好地告知用户这一情况，并建议他们先创建一些备忘录。"
	} else {
		systemPrompt = "你是一个基于用户个人笔记的AI助手。请根据以下笔记内容回答问题。你必须严格基于提供的笔记内容回答，不要编造或假设任何笔记中没有的信息。如果笔记中没有相关信息，请明确告知用户。回答时使用中文，保持简洁准确。"
	}
	messages := []ai.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Add history
	for i := 0; i < len(req.Msg.History)-1; i += 2 {
		if i+1 < len(req.Msg.History) {
			messages = append(messages, ai.Message{Role: "user", Content: req.Msg.History[i]})
			messages = append(messages, ai.Message{Role: "assistant", Content: req.Msg.History[i+1]})
		}
	}

	// Add current message
	userMessage := fmt.Sprintf("## 相关笔记\n%s\n## 用户问题\n%s", contextBuilder.String(), req.Msg.Message)
	messages = append(messages, ai.Message{Role: "user", Content: userMessage})

	// 6. Stream LLM Response
	contentChan, errChan := s.AIService.LLMService.ChatStream(ctx, messages)

	// Send sources first
	if err := stream.Send(&v1pb.ChatWithMemosResponse{
		Sources: sources,
	}); err != nil {
		return err
	}

	// Stream content
	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				contentChan = nil // Closed
				if errChan == nil {
					return nil // Done
				}
				continue
			}
			if err := stream.Send(&v1pb.ChatWithMemosResponse{
				Content: content,
			}); err != nil {
				return err
			}

		case err, ok := <-errChan:
			if !ok {
				errChan = nil // Closed
				if contentChan == nil {
					return nil // Done
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
