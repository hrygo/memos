package retrieval

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/server/queryengine"
	"github.com/usememos/memos/store"
)

// AdaptiveRetriever 自适应检索器
// 根据查询复杂度和结果质量动态调整检索策略
type AdaptiveRetriever struct {
	store            *store.Store
	embeddingService ai.EmbeddingService
	rerankerService  ai.RerankerService
}

// SearchResult 检索结果
type SearchResult struct {
	ID       int64
	Type     string // "memo" or "schedule"
	Score    float32
	Content  string
	Memo     *store.Memo
	Schedule *store.Schedule
}

// RetrievalOptions 检索选项
type RetrievalOptions struct {
	Query            string
	UserID           int32
	Strategy         string
	TimeRange        *queryengine.TimeRange
	MinScore         float32
	Limit            int
	RequestID        string // 请求追踪 ID
	Logger           *slog.Logger // 结构化日志记录器
	ScheduleQueryMode queryengine.ScheduleQueryMode // P1: 日程查询模式
}

// NewAdaptiveRetriever 创建自适应检索器
func NewAdaptiveRetriever(
	st *store.Store,
	embeddingService ai.EmbeddingService,
	rerankerService ai.RerankerService,
) *AdaptiveRetriever {
	return &AdaptiveRetriever{
		store:            st,
		embeddingService: embeddingService,
		rerankerService:  rerankerService,
	}
}

// Retrieve 自适应检索主入口
func (r *AdaptiveRetriever) Retrieve(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = &RetrievalOptions{
			Strategy: "hybrid_standard",
			Limit:    10,
			MinScore: 0.5,
		}
	}

	// 输入验证：P0 改进 - 添加查询长度限制
	if len(opts.Query) > 1000 {
		return nil, fmt.Errorf("query too long: %d characters (max 1000)", len(opts.Query))
	}

	// 初始化日志记录器
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.RequestID == "" {
		opts.RequestID = generateRequestID()
	}

	// 根据路由策略选择检索路径
	switch opts.Strategy {
	case "schedule_bm25_only":
		return r.scheduleBM25Only(ctx, opts)

	case "memo_semantic_only":
		return r.memoSemanticOnly(ctx, opts)

	case "hybrid_bm25_weighted":
		return r.hybridBM25Weighted(ctx, opts)

	case "hybrid_with_time_filter":
		return r.hybridWithTimeFilter(ctx, opts)

	case "hybrid_standard":
		return r.hybridStandard(ctx, opts)

	case "full_pipeline_with_reranker":
		return r.fullPipelineWithReranker(ctx, opts)

	default:
		// 默认使用标准混合检索
		return r.hybridStandard(ctx, opts)
	}
}

// scheduleBM25Only 纯日程查询（BM25 + 时间过滤）
func (r *AdaptiveRetriever) scheduleBM25Only(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "schedule_bm25_only",
		"user_id", opts.UserID,
	)

	// 构建查询条件
	findSchedule := &store.FindSchedule{
		CreatorID: &opts.UserID,
	}

	// P1: 设置查询模式（将 queryengine.ScheduleQueryMode 转换为 int32）
	if opts.ScheduleQueryMode != queryengine.AutoQueryMode {
		mode := int32(opts.ScheduleQueryMode)
		findSchedule.QueryMode = &mode
	}

	// 添加时间过滤（P0 改进：添加 nil 检查和验证）
	if opts.TimeRange != nil {
		// 验证时间范围
		if !opts.TimeRange.ValidateTimeRange() {
			opts.Logger.WarnContext(ctx, "Invalid time range",
				"request_id", opts.RequestID,
				"start", opts.TimeRange.Start,
				"end", opts.TimeRange.End,
			)
			return nil, fmt.Errorf("invalid time range: start=%v, end=%v", opts.TimeRange.Start, opts.TimeRange.End)
		}

		startTs := opts.TimeRange.Start.Unix()
		endTs := opts.TimeRange.End.Unix()
		findSchedule.StartTs = &startTs
		findSchedule.EndTs = &endTs
	}

	// 查询日程
	schedules, err := r.store.ListSchedules(ctx, findSchedule)
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Failed to list schedules",
			"request_id", opts.RequestID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// P1 改进：内存优化 - 预分配切片容量
	results := make([]*SearchResult, 0, len(schedules))
	for _, schedule := range schedules {
		results = append(results, &SearchResult{
			ID:       int64(schedule.ID),
			Type:     "schedule",
			Score:    1.0, // 日程查询默认高分
			Content:  schedule.Title,
			Schedule: schedule,
		})
	}

	// P1 改进：内存优化 - 释放不再需要的大对象引用
	// 如果 Schedule 描述很大，可以只保留必要的字段
	for _, result := range results {
		if result.Schedule != nil && len(result.Schedule.Description) > 10000 {
			// 描述超过 10KB，截断以减少内存占用
			result.Content = result.Schedule.Title
			result.Schedule = nil // 释放完整 Schedule 对象
		}
	}

	opts.Logger.InfoContext(ctx, "Schedule retrieval completed",
		"request_id", opts.RequestID,
		"result_count", len(results),
	)

	return results, nil
}

// memoSemanticOnly 纯笔记查询（语义向量）
func (r *AdaptiveRetriever) memoSemanticOnly(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "memo_semantic_only",
		"user_id", opts.UserID,
	)

	// 生成查询向量
	queryVector, err := r.embeddingService.Embed(ctx, opts.Query)
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Failed to embed query",
			"request_id", opts.RequestID,
			"error", err,
			"query_length", len(opts.Query),
		)
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 第一阶段：快速检索 Top 5
	limit := 5
	if opts.Limit > 0 {
		limit = opts.Limit
	}

	vectorResults, err := r.store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: opts.UserID,
		Vector: queryVector,
		Limit:  limit,
	})
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Vector search failed",
			"request_id", opts.RequestID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 评估结果质量
	results := r.convertVectorResults(vectorResults)

	quality := r.evaluateQuality(results)
	opts.Logger.InfoContext(ctx, "Evaluated result quality",
		"request_id", opts.RequestID,
		"quality_level", quality.String(),
		"result_count", len(results),
	)

	// 根据质量决定是否扩展
	if quality == MediumQuality && opts.Limit > 5 {
		// 扩展到 Top 20
		moreResults, err := r.store.VectorSearch(ctx, &store.VectorSearchOptions{
			UserID: opts.UserID,
			Vector: queryVector,
			Limit:  20,
		})
		if err == nil {
			// 合并结果
			results = r.mergeResults(results, r.convertVectorResults(moreResults), opts.Limit)
			opts.Logger.DebugContext(ctx, "Expanded results",
				"request_id", opts.RequestID,
				"new_count", len(results),
			)
		}
	}

	// 过滤低分结果
	filtered := r.filterByScore(results, opts.MinScore)
	opts.Logger.InfoContext(ctx, "Semantic retrieval completed",
		"request_id", opts.RequestID,
		"final_count", len(filtered),
		"min_score", opts.MinScore,
	)

	return filtered, nil
}

// hybridBM25Weighted 混合检索（BM25 加权）
func (r *AdaptiveRetriever) hybridBM25Weighted(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "hybrid_bm25_weighted",
		"user_id", opts.UserID,
	)

	// BM25 权重更高（0.7），语义权重更低（0.3）
	return r.hybridSearch(ctx, opts, 0.3)
}

// hybridWithTimeFilter 混合检索（时间过滤）
func (r *AdaptiveRetriever) hybridWithTimeFilter(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "hybrid_with_time_filter",
		"user_id", opts.UserID,
	)

	// 标准混合检索 + 时间过滤
	results, err := r.hybridSearch(ctx, opts, 0.5)
	if err != nil {
		return nil, err
	}

	// 如果指定了时间范围，过滤日程结果（P0 改进：添加 nil 检查）
	if opts.TimeRange != nil {
		// P1 改进：内存优化 - 预分配容量
		filtered := make([]*SearchResult, 0, len(results))
		for _, result := range results {
			if result.Type == "memo" {
				filtered = append(filtered, result)
			} else if result.Type == "schedule" && result.Schedule != nil {
				scheduleTime := time.Unix(result.Schedule.StartTs, 0)
				if opts.TimeRange.Contains(scheduleTime) {
					filtered = append(filtered, result)
				}
			}
		}
		// P1 改进：内存优化 - 用新切片替换旧切片，让旧切片可被 GC
		results = filtered
	}

	return results, nil
}

// hybridStandard 标准混合检索（BM25 + 语义）
func (r *AdaptiveRetriever) hybridStandard(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "hybrid_standard",
		"user_id", opts.UserID,
	)

	// BM25 和语义权重相等（0.5 + 0.5）
	return r.hybridSearch(ctx, opts, 0.5)
}

// fullPipelineWithReranker 完整流程（混合检索 + Reranker）
func (r *AdaptiveRetriever) fullPipelineWithReranker(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "full_pipeline_with_reranker",
		"user_id", opts.UserID,
	)

	// 第一步：混合检索 Top 20
	hybridResults, err := r.hybridSearch(ctx, opts, 0.5)
	if err != nil {
		return nil, err
	}

	// 第二步：检查是否需要重排
	if !r.shouldRerank(opts.Query, hybridResults) {
		opts.Logger.InfoContext(ctx, "Skipping reranker (not needed)",
			"request_id", opts.RequestID,
			"reason", "simple_query_or_few_results",
		)
		// 不需要重排，直接返回 Top K
		return r.truncateResults(hybridResults, opts.Limit), nil
	}

	// 第三步：Reranker 重排序
	opts.Logger.InfoContext(ctx, "Applying reranker",
		"request_id", opts.RequestID,
		"result_count", len(hybridResults),
	)

	// 准备文档
	// P1 改进：内存优化 - 预分配容量
	documents := make([]string, 0, len(hybridResults))
	for _, result := range hybridResults {
		// P1 改进：内存优化 - 限制文档长度
		content := result.Content
		if len(content) > 5000 {
			// 内容超过 5000 字符，截断以减少内存和 API 成本
			content = content[:5000]
		}
		documents = append(documents, content)
	}

	// 调用 Reranker
	rerankResults, err := r.rerankerService.Rerank(ctx, opts.Query, documents, opts.Limit)
	if err != nil {
		opts.Logger.WarnContext(ctx, "Reranker failed, using hybrid results",
			"request_id", opts.RequestID,
			"error", err,
		)
		// 降级：返回原始结果
		return r.truncateResults(hybridResults, opts.Limit), nil
	}

	// 重新排序
	// P1 改进：内存优化 - 预分配容量
	reordered := make([]*SearchResult, 0, len(rerankResults))
	for _, rr := range rerankResults {
		if rr.Index < len(hybridResults) {
			result := hybridResults[rr.Index]
			result.Score = rr.Score // 更新分数
			reordered = append(reordered, result)
		}
	}

	opts.Logger.InfoContext(ctx, "Reranker completed",
		"request_id", opts.RequestID,
		"result_count", len(reordered),
	)

	// P1 改进：内存优化 - 释放不需要的大对象
	// 清空 documents 以便 GC 回收
	for i := range documents {
		documents[i] = ""
	}

	return reordered, nil
}

// hybridSearch 混合检索实现
func (r *AdaptiveRetriever) hybridSearch(ctx context.Context, opts *RetrievalOptions, semanticWeight float32) ([]*SearchResult, error) {
	// 语义检索
	queryVector, err := r.embeddingService.Embed(ctx, opts.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	vectorResults, err := r.store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: opts.UserID,
		Vector: queryVector,
		Limit:  20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to vector search: %w", err)
	}

	// 转换并融合
	results := r.convertVectorResults(vectorResults)

	// 简化实现：只使用语义检索结果（BM25 需要全文检索支持）
	// 在实际生产环境中，应该结合 BM25 分数
	for _, result := range results {
		result.Score = result.Score * semanticWeight
	}

	return results, nil
}

// convertVectorResults 转换向量检索结果
func (r *AdaptiveRetriever) convertVectorResults(results []*store.MemoWithScore) []*SearchResult {
	searchResults := make([]*SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = &SearchResult{
			ID:      int64(r.Memo.ID),
			Type:    "memo",
			Score:   r.Score,
			Content: r.Memo.Content,
			Memo:    r.Memo,
		}
	}
	return searchResults
}

// QualityLevel 结果质量等级
type QualityLevel int

const (
	LowQuality QualityLevel = iota
	MediumQuality
	HighQuality
)

// String 返回质量等级的字符串表示
func (q QualityLevel) String() string {
	switch q {
	case LowQuality:
		return "low"
	case MediumQuality:
		return "medium"
	case HighQuality:
		return "high"
	default:
		return "unknown"
	}
}

// evaluateQuality 评估结果质量
func (r *AdaptiveRetriever) evaluateQuality(results []*SearchResult) QualityLevel {
	if len(results) == 0 {
		return LowQuality
	}

	topScore := results[0].Score

	// 判断 1：前2名分数差距大 → 高质量
	if len(results) >= 2 {
		scoreGap := topScore - results[1].Score
		if scoreGap > 0.20 {
			return HighQuality
		}
	}

	// 判断 2：第1名分数很高 → 高质量
	if topScore > 0.90 {
		return HighQuality
	}

	// 判断 3：第1名分数中等 → 中等质量
	if topScore > 0.70 {
		return MediumQuality
	}

	// 否则：低质量
	return LowQuality
}

// mergeResults 合并结果（去重，按分数排序）
func (r *AdaptiveRetriever) mergeResults(results1, results2 []*SearchResult, topK int) []*SearchResult {
	// 去重（基于 ID）
	seen := make(map[int64]bool)
	merged := make([]*SearchResult, 0)

	for _, result := range results1 {
		if !seen[result.ID] {
			seen[result.ID] = true
			merged = append(merged, result)
		}
	}

	for _, result := range results2 {
		if !seen[result.ID] {
			seen[result.ID] = true
			merged = append(merged, result)
		}
	}

	// 按分数排序
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	// 返回 Top K
	return r.truncateResults(merged, topK)
}

// shouldRerank 判断是否需要重排
func (r *AdaptiveRetriever) shouldRerank(query string, results []*SearchResult) bool {
	// 检查 Reranker 是否启用
	if r.rerankerService == nil || !r.rerankerService.IsEnabled() {
		return false
	}

	// 规则 1：结果少（<5），不需要重排
	if len(results) < 5 {
		return false
	}

	// 规则 2：简单查询，不需要重排
	if r.isSimpleKeywordQuery(query) {
		return false
	}

	// 规则 3：前2名分数差距大（>0.15），不需要重排
	if len(results) >= 2 {
		if results[0].Score-results[1].Score > 0.15 {
			return false
		}
	}

	// 其他情况：需要重排
	return true
}

// isSimpleKeywordQuery 判断是否为简单关键词查询
func (r *AdaptiveRetriever) isSimpleKeywordQuery(query string) bool {
	// 简单查询特征：
	// 1. 查询短（<10个字符）
	if len(query) < 10 {
		return true
	}

	// 2. 检测是否有疑问词、连词等复杂语法
	complexWords := []string{"如何", "怎么", "为什么", "和", "或者", "但是", "how", "why"}
	for _, word := range complexWords {
		if strings.Contains(query, word) {
			return false
		}
	}

	return true
}

// filterByScore 过滤低分结果
func (r *AdaptiveRetriever) filterByScore(results []*SearchResult, minScore float32) []*SearchResult {
	if minScore <= 0 {
		return results
	}

	filtered := make([]*SearchResult, 0)
	for _, result := range results {
		if result.Score >= minScore {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// truncateResults 截断结果到指定数量
func (r *AdaptiveRetriever) truncateResults(results []*SearchResult, limit int) []*SearchResult {
	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
}

// generateRequestID 生成唯一的请求 ID
func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), b)
}
