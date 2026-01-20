package finops

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCostMonitor_NewCostMonitor 测试创建成本监控器
func TestCostMonitor_NewCostMonitor(t *testing.T) {
	// 注意：这需要一个真实的数据库连接，实际测试中应该使用 mock
	// 这里只是示例结构

	db, err := sql.Open("postgres", "host=localhost port=25432 user=memos password=memos dbname=memos sslmode=disable")
	if err != nil {
		t.Skip("需要数据库连接")
		return
	}
	defer db.Close()

	monitor := NewCostMonitor(db)

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.db)
	assert.NotNil(t, monitor.statsCache)
	assert.Equal(t, 5*time.Minute, monitor.cacheTTL)
}

// TestCostMonitor_CreateQueryCostRecord 测试创建成本记录
func TestCostMonitor_CreateQueryCostRecord(t *testing.T) {
	record := CreateQueryCostRecord(
		1,                    // userID
		"今天的日程",          // query
		"schedule_bm25_only", // strategy
		0.001,                // vectorCost
		0.0,                  // rerankerCost
		0.002,                // llmCost
		150,                  // latencyMs
		3,                    // resultCount
	)

	assert.NotNil(t, record)
	assert.Equal(t, int32(1), record.UserID)
	assert.Equal(t, "schedule_bm25_only", record.Strategy)
	assert.Equal(t, 0.001, record.VectorCost)
	assert.Equal(t, 0.002, record.LLMCost)
	assert.Equal(t, 0.003, record.TotalCost)
	assert.Equal(t, int64(150), record.LatencyMs)
	assert.Equal(t, 3, record.ResultCount)
	assert.False(t, record.Timestamp.IsZero())
}

// TestCostMonitor_CalculateTotalCost 测试成本计算
func TestCostMonitor_CalculateTotalCost(t *testing.T) {
	tests := []struct {
		name         string
		vectorCost   float64
		rerankerCost float64
		llmCost      float64
		expected     float64
	}{
		{
			name:         "全部为零",
			vectorCost:   0,
			rerankerCost: 0,
			llmCost:      0,
			expected:     0,
		},
		{
			name:         "只有向量成本",
			vectorCost:   0.001,
			rerankerCost: 0,
			llmCost:      0,
			expected:     0.001,
		},
		{
			name:         "只有 Reranker 成本",
			vectorCost:   0,
			rerankerCost: 0.005,
			llmCost:      0,
			expected:     0.005,
		},
		{
			name:         "只有 LLM 成本",
			vectorCost:   0,
			rerankerCost: 0,
			llmCost:      0.01,
			expected:     0.01,
		},
		{
			name:         "全部成本",
			vectorCost:   0.001,
			rerankerCost: 0.005,
			llmCost:      0.01,
			expected:     0.016,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTotalCost(tt.vectorCost, tt.rerankerCost, tt.llmCost)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCostMonitor_EstimateEmbeddingCost 测试 Embedding 成本估算
func TestCostMonitor_EstimateEmbeddingCost(t *testing.T) {
	tests := []struct {
		name         string
		textLength   int
		minCost      float64
		maxCost      float64
	}{
		{
			name:    "短文本",
			textLength: 10,
			minCost: 0,
			maxCost: 0.00001,
		},
		{
			name:    "中等文本",
			textLength: 100,
			minCost: 0,
			maxCost: 0.0001,
		},
		{
			name:    "长文本",
			textLength: 1000,
			minCost: 0.00001,
			maxCost: 0.001,
		},
		{
			name:    "超长文本",
			textLength: 10000,
			minCost: 0.0001,
			maxCost: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateEmbeddingCost(tt.textLength)
			assert.GreaterOrEqual(t, cost, tt.minCost)
			assert.LessOrEqual(t, cost, tt.maxCost)
		})
	}
}

// TestCostMonitor_EstimateRerankerCost 测试 Reranker 成本估算
func TestCostMonitor_EstimateRerankerCost(t *testing.T) {
	tests := []struct {
		name          string
		queryLength   int
		docCount      int
		avgDocLength  int
		minCost       float64
		maxCost       float64
	}{
		{
			name:    "少量短文档",
			queryLength: 10,
			docCount: 5,
			avgDocLength: 50,
			minCost: 0,
			maxCost: 0.0001,
		},
		{
			name:    "中等数量文档",
			queryLength: 20,
			docCount: 10,
			avgDocLength: 100,
			minCost: 0,
			maxCost: 0.0005,
		},
		{
			name:    "大量长文档",
			queryLength: 50,
			docCount: 20,
			avgDocLength: 200,
			minCost: 0.0001,
			maxCost: 0.002,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateRerankerCost(tt.queryLength, tt.docCount, tt.avgDocLength)
			assert.GreaterOrEqual(t, cost, tt.minCost)
			assert.LessOrEqual(t, cost, tt.maxCost)
		})
	}
}

// TestCostMonitor_EstimateLLMCost 测试 LLM 成本估算
func TestCostMonitor_EstimateLLMCost(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int
		outputTokens int
		minCost      float64
		maxCost      float64
	}{
		{
			name:    "短对话",
			inputTokens: 100,
			outputTokens: 50,
			minCost: 0,
			maxCost: 0.0001,
		},
		{
			name:    "中等对话",
			inputTokens: 500,
			outputTokens: 200,
			minCost: 0.00001,
			maxCost: 0.0005,
		},
		{
			name:    "长对话",
			inputTokens: 2000,
			outputTokens: 1000,
			minCost: 0.0001,
			maxCost: 0.002,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := EstimateLLMCost(tt.inputTokens, tt.outputTokens)
			assert.GreaterOrEqual(t, cost, tt.minCost)
			assert.LessOrEqual(t, cost, tt.maxCost)
		})
	}
}

// TestCostMonitor_GetPeriodStartTime 测试周期开始时间计算
func TestCostMonitor_GetPeriodStartTime(t *testing.T) {
	monitor := &CostMonitor{}

	tests := []struct {
		name         string
		period       string
		expectedDiff time.Duration
	}{
		{
			name:    "每日",
			period:  "daily",
			expectedDiff: 24 * time.Hour,
		},
		{
			name:    "每周",
			period:  "weekly",
			expectedDiff: 7 * 24 * time.Hour,
		},
		{
			name:    "每月",
			period:  "monthly",
			expectedDiff: 30 * 24 * time.Hour,
		},
		{
			name:    "今天",
			period:  "today",
			expectedDiff: 24 * time.Hour,
		},
		{
			name:    "默认（未知周期）",
			period:  "unknown",
			expectedDiff: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, err := monitor.getPeriodStartTime(tt.period)
			require.NoError(t, err)

			elapsed := time.Since(startTime)
			// 允许 1 小时的误差
			assert.InDelta(t, tt.expectedDiff, elapsed, float64(time.Hour))
		})
	}
}

// TestCostRecord_Validation 测试成本记录验证
func TestCostRecord_Validation(t *testing.T) {
	tests := []struct {
		name      string
		record    *QueryCostRecord
		expectErr bool
	}{
		{
			name: "有效记录",
			record: &QueryCostRecord{
				UserID:        1,
				Query:         "测试查询",
				Strategy:      "hybrid_standard",
				VectorCost:    0.001,
				RerankerCost:  0.0,
				LLMCost:       0.002,
				TotalCost:     0.003,
				LatencyMs:     150,
				ResultCount:   5,
			},
			expectErr: false,
		},
		{
			name: "无效的用户ID",
			record: &QueryCostRecord{
				UserID:        0,
				Query:         "测试查询",
				Strategy:      "hybrid_standard",
				VectorCost:    0.001,
				TotalCost:     0.001,
				LatencyMs:     150,
				ResultCount:   5,
			},
			expectErr: true,
		},
		{
			name: "空策略",
			record: &QueryCostRecord{
				UserID:        1,
				Query:         "测试查询",
				Strategy:      "",
				VectorCost:    0.001,
				TotalCost:     0.001,
				LatencyMs:     150,
				ResultCount:   5,
			},
			expectErr: true,
		},
		{
			name: "负成本",
			record: &QueryCostRecord{
				UserID:        1,
				Query:         "测试查询",
				Strategy:      "hybrid_standard",
				VectorCost:    -0.001,
				TotalCost:     -0.001,
				LatencyMs:     150,
				ResultCount:   5,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCostRecord(tt.record)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// validateCostRecord 验证成本记录
func validateCostRecord(record *QueryCostRecord) error {
	if record.UserID <= 0 {
		return assert.AnError
	}
	if record.Strategy == "" {
		return assert.AnError
	}
	if record.TotalCost < 0 {
		return assert.AnError
	}
	return nil
}

// BenchmarkCostMonitor_CalculateTotalCost 性能基准测试
func BenchmarkCostMonitor_CalculateTotalCost(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateTotalCost(0.001, 0.005, 0.01)
	}
}

// BenchmarkCostMonitor_EstimateEmbeddingCost 性能基准测试
func BenchmarkCostMonitor_EstimateEmbeddingCost(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateEmbeddingCost(1000)
	}
}
