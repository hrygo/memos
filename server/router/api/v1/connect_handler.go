package v1

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

	// 3. Semantic Search (Top 5, Score > 0.5)
	queryVector, err := s.AIService.EmbeddingService.Embed(ctx, req.Msg.Message)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to embed query: %v", err))
	}

	results, err := s.AIService.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  5,
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to search: %v", err))
	}

	// 4. Build Context (max chars: 3000)
	var contextBuilder strings.Builder
	var sources []string
	totalChars := 0
	maxChars := 3000

	for i, r := range results {
		if r.Score < 0.5 { // Ignore low relevance
			continue
		}

		content := r.Memo.Content
		if totalChars+len(content) > maxChars {
			break // Stop adding context
		}

		contextBuilder.WriteString(fmt.Sprintf("### 笔记 %d\n%s\n\n", i+1, content))
		sources = append(sources, fmt.Sprintf("memos/%s", r.Memo.UID))
		totalChars += len(content)
	}

	// 5. Build Prompt
	messages := []ai.Message{
		{
			Role:    "system",
			Content: "你是一个基于用户个人笔记的AI助手。请根据以下笔记内容回答问题。如果笔记中没有相关信息，请明确告知用户。回答时使用中文，保持简洁准确。",
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
