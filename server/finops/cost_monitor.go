package finops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CostMonitor 成本监控器，用于追踪 AI 查询的成本和性能
type CostMonitor struct {
	db     *sql.DB
	logger *slog.Logger
	// 内存缓存，用于快速获取策略统计
	statsCache map[string]*StrategyStats
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
	lastUpdate time.Time
}

// QueryCostRecord 查询成本记录
type QueryCostRecord struct {
	Timestamp     time.Time
	UserID        int32
	Query         string
	Strategy      string

	// 成本细分（单位：美元）
	VectorCost   float64
	RerankerCost float64
	LLMCost      float64
	TotalCost    float64

	// 性能指标
	LatencyMs int64

	// 结果指标
	ResultCount    int
	UserSatisfied  float32 // 0-1，NULL 表示未评分
}

// StrategyStats 策略统计
type StrategyStats struct {
	Strategy     string
	QueryCount   int64
	Cost         float64
	AvgLatency   float64
	AvgResults   float64
	LastUpdated  time.Time
}

// CostReport 成本报告
type CostReport struct {
	Period     string
	TotalCost  float64
	ByStrategy map[string]*StrategyStats
	TopCosts   []QueryCostRecord
}

// NewCostMonitor 创建新的成本监控器
func NewCostMonitor(db *sql.DB) *CostMonitor {
	return &CostMonitor{
		db:         db,
		logger:     slog.Default(),
		statsCache: make(map[string]*StrategyStats),
		cacheTTL:   5 * time.Minute,
		lastUpdate: time.Time{},
	}
}

// Record 记录查询成本
func (m *CostMonitor) Record(ctx context.Context, record *QueryCostRecord) error {
	if record == nil {
		return fmt.Errorf("record cannot be nil")
	}

	// 参数验证（P0 改进：增强输入验证）
	if record.UserID <= 0 {
		m.logger.WarnContext(ctx, "Invalid user ID in cost record",
			"user_id", record.UserID,
		)
		return fmt.Errorf("invalid user ID")
	}
	if record.Strategy == "" {
		m.logger.WarnContext(ctx, "Empty strategy in cost record",
			"user_id", record.UserID,
		)
		return fmt.Errorf("strategy cannot be empty")
	}
	if record.TotalCost < 0 {
		m.logger.WarnContext(ctx, "Negative total cost in cost record",
			"user_id", record.UserID,
			"total_cost", record.TotalCost,
		)
		return fmt.Errorf("total cost cannot be negative")
	}
	if record.LatencyMs < 0 {
		m.logger.WarnContext(ctx, "Negative latency in cost record",
			"user_id", record.UserID,
			"latency_ms", record.LatencyMs,
		)
		return fmt.Errorf("latency cannot be negative")
	}

	_, err := m.db.ExecContext(ctx, `
		INSERT INTO query_cost_log (
			timestamp, user_id, query, strategy,
			vector_cost, reranker_cost, llm_cost, total_cost,
			latency_ms, result_count, user_satisfied
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		record.Timestamp,
		record.UserID,
		record.Query,
		record.Strategy,
		record.VectorCost,
		record.RerankerCost,
		record.LLMCost,
		record.TotalCost,
		record.LatencyMs,
		record.ResultCount,
		record.UserSatisfied,
	)

	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to record query cost",
			"user_id", record.UserID,
			"strategy", record.Strategy,
			"error", err,
		)
		return err
	}

	m.logger.DebugContext(ctx, "Recorded query cost",
		"user_id", record.UserID,
		"strategy", record.Strategy,
		"total_cost", record.TotalCost,
		"latency_ms", record.LatencyMs,
	)

	return nil
}

// GetCostReport 获取成本报告
func (m *CostMonitor) GetCostReport(ctx context.Context, period string) (*CostReport, error) {
	startTime, err := m.getPeriodStartTime(period)
	if err != nil {
		return nil, err
	}

	// 查询总成本
	var totalCost float64
	err = m.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM query_cost_log
		WHERE timestamp >= $1
	`, startTime).Scan(&totalCost)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// 按策略分组统计
	rows, err := m.db.QueryContext(ctx, `
		SELECT
			strategy,
			COUNT(*) as query_count,
			COALESCE(SUM(total_cost), 0) as cost,
			COALESCE(AVG(latency_ms), 0) as avg_latency,
			COALESCE(AVG(result_count), 0) as avg_results
		FROM query_cost_log
		WHERE timestamp >= $1
		GROUP BY strategy
		ORDER BY cost DESC
	`, startTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byStrategy := make(map[string]*StrategyStats)
	for rows.Next() {
		var stats StrategyStats
		err := rows.Scan(&stats.Strategy, &stats.QueryCount, &stats.Cost, &stats.AvgLatency, &stats.AvgResults)
		if err != nil {
			continue
		}
		stats.LastUpdated = time.Now()
		byStrategy[stats.Strategy] = &stats

		// 更新缓存
		m.cacheMutex.Lock()
		m.statsCache[stats.Strategy] = &stats
		m.cacheMutex.Unlock()
	}

	// 查询成本最高的查询
	topCostRows, err := m.db.QueryContext(ctx, `
		SELECT
			timestamp, user_id, query, strategy,
			vector_cost, reranker_cost, llm_cost, total_cost,
			latency_ms, result_count
		FROM query_cost_log
		WHERE timestamp >= $1
		ORDER BY total_cost DESC
		LIMIT 10
	`, startTime)

	if err != nil {
		return nil, err
	}
	defer topCostRows.Close()

	topCosts := make([]QueryCostRecord, 0)
	for topCostRows.Next() {
		var record QueryCostRecord
		err := topCostRows.Scan(
			&record.Timestamp,
			&record.UserID,
			&record.Query,
			&record.Strategy,
			&record.VectorCost,
			&record.RerankerCost,
			&record.LLMCost,
			&record.TotalCost,
			&record.LatencyMs,
			&record.ResultCount,
		)
		if err != nil {
			continue
		}
		topCosts = append(topCosts, record)
	}

	m.lastUpdate = time.Now()

	return &CostReport{
		Period:     period,
		TotalCost:  totalCost,
		ByStrategy: byStrategy,
		TopCosts:   topCosts,
	}, nil
}

// getPeriodStartTime 根据周期获取开始时间
func (m *CostMonitor) getPeriodStartTime(period string) (time.Time, error) {
	now := time.Now()

	switch period {
	case "daily", "today":
		return now.AddDate(0, 0, -1), nil
	case "weekly", "this_week":
		return now.AddDate(0, 0, -7), nil
	case "monthly", "this_month":
		return now.AddDate(0, -1, 0), nil
	default:
		return now.AddDate(0, 0, -1), nil
	}
}

// getStrategyStats 从缓存或数据库获取策略统计
func (m *CostMonitor) getStrategyStats(strategy string) *StrategyStats {
	// 检查缓存是否有效
	m.cacheMutex.RLock()
	if time.Since(m.lastUpdate) < m.cacheTTL {
		if stats, ok := m.statsCache[strategy]; ok {
			m.cacheMutex.RUnlock()
			return stats
		}
	}
	m.cacheMutex.RUnlock()

	// 缓存过期，从数据库查询（异步更新缓存）
	go m.updateCache()

	return nil
}

// updateCache 更新缓存
func (m *CostMonitor) updateCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.GetCostReport(ctx, "daily")
	if err != nil {
		// 记录错误但不影响主流程
		m.logger.ErrorContext(ctx, "Failed to update cost cache",
			"error", err,
		)
	}
}

// OptimizeStrategy 根据成本效益优化策略
func (m *CostMonitor) OptimizeStrategy(query string, currentStrategy string) string {
	stats := m.getStrategyStats(currentStrategy)
	if stats == nil {
		return currentStrategy
	}

	// 规则 1：高频查询且成本低，继续使用
	if stats.QueryCount > 1000 && stats.Cost < 0.01 {
		return currentStrategy
	}

	// 规则 2：高频查询且成本高，降级策略
	if stats.QueryCount > 1000 && stats.Cost > 0.05 {
		return m.downgradeStrategy(currentStrategy)
	}

	// 规则 3：低频查询且成本高，考虑缓存
	if stats.QueryCount < 100 && stats.Cost > 0.05 {
		return "cached"
	}

	return currentStrategy
}

// downgradeStrategy 降级策略
func (m *CostMonitor) downgradeStrategy(strategy string) string {
	downgradeMap := map[string]string{
		"full_pipeline_with_reranker": "hybrid_standard",
		"hybrid_standard":              "memo_semantic_only",
		"hybrid_bm25_weighted":         "schedule_bm25_only",
	}

	if downgrade, ok := downgradeMap[strategy]; ok {
		return downgrade
	}

	return strategy
}

// CalculateTotalCost 计算总成本
func CalculateTotalCost(vectorCost, rerankerCost, llmCost float64) float64 {
	return vectorCost + rerankerCost + llmCost
}

// EstimateEmbeddingCost 估算 Embedding 成本（假设使用 SiliconFlow BGE-M3）
// 价格：$0.0001 / 1M tokens（约 0.000001 / 10K tokens）
func EstimateEmbeddingCost(textLength int) float64 {
	// 估算 token 数：中文约 2 字符 = 1 token，英文约 4 字符 = 1 token
	estimatedTokens := float64(textLength) / 3.0

	// SiliconFlow BGE-M3 价格：$0.0001 / 1M tokens
	costPerToken := 0.0001 / 1000000.0

	return estimatedTokens * costPerToken
}

// EstimateRerankerCost 估算 Reranker 成本（假设使用 SiliconFlow BGE Reranker）
// 价格：$0.0001 / 1K tokens
func EstimateRerankerCost(queryLength int, docCount int, avgDocLength int) float64 {
	// 估算总 token 数
	queryTokens := float64(queryLength) / 3.0
	docTokens := float64(docCount * avgDocLength) / 3.0
	totalTokens := queryTokens + docTokens

	// SiliconFlow BGE Reranker 价格：$0.0001 / 1K tokens
	costPer1KTokens := 0.0001

	return (totalTokens / 1000.0) * costPer1KTokens
}

// EstimateLLMCost 估算 LLM 成本（假设使用 DeepSeek Chat）
// 价格：输入 $0.14 / 1M tokens，输出 $0.28 / 1M tokens
func EstimateLLMCost(inputTokens, outputTokens int) float64 {
	// DeepSeek 价格
	inputCostPerToken := 0.14 / 1000000.0
	outputCostPerToken := 0.28 / 1000000.0

	inputCost := float64(inputTokens) * inputCostPerToken
	outputCost := float64(outputTokens) * outputCostPerToken

	return inputCost + outputCost
}

// CreateQueryCostRecord 创建查询成本记录（辅助函数）
func CreateQueryCostRecord(
	userID int32,
	query string,
	strategy string,
	vectorCost float64,
	rerankerCost float64,
	llmCost float64,
	latencyMs int64,
	resultCount int,
) *QueryCostRecord {
	return &QueryCostRecord{
		Timestamp:     time.Now(),
		UserID:        userID,
		Query:         query,
		Strategy:      strategy,
		VectorCost:    vectorCost,
		RerankerCost:  rerankerCost,
		LLMCost:       llmCost,
		TotalCost:     CalculateTotalCost(vectorCost, rerankerCost, llmCost),
		LatencyMs:     latencyMs,
		ResultCount:   resultCount,
		UserSatisfied: 0, // 初始为 0，等待用户反馈
	}
}
