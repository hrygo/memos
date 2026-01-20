package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

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

	// 添加历史对话
	for i := 0; i < len(req.History)-1; i += 2 {
		if i+1 < len(req.History) {
			messages = append(messages, ai.Message{Role: "user", Content: req.History[i]})
			messages = append(messages, ai.Message{Role: "assistant", Content: req.History[i+1]})
		}
	}

	// 添加当前问题
	userMessage := fmt.Sprintf("## 相关笔记\n%s\n## 用户问题\n%s", contextBuilder.String(), req.Message)
	messages = append(messages, ai.Message{Role: "user", Content: userMessage})

	// 6. 流式调用 LLM
	contentChan, errChan := s.LLMService.ChatStream(ctx, messages)

	// 先发送来源信息
	if err := stream.Send(&v1pb.ChatWithMemosResponse{
		Sources: sources,
	}); err != nil {
		return err
	}

	// 流式发送内容
	for {
		select {
		case content, ok := <-contentChan:
			if !ok {
				contentChan = nil // 标记为已关闭
				if errChan == nil {
					return stream.Send(&v1pb.ChatWithMemosResponse{Done: true})
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
				errChan = nil // 标记为已关闭
				if contentChan == nil {
					return stream.Send(&v1pb.ChatWithMemosResponse{Done: true})
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
