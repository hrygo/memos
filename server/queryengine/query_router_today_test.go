package queryengine

import (
	"context"
	"testing"
	"time"
)

// TestQueryRouter_TodaySynonyms 测试"今日"等同义词（举一反三优化）
func TestQueryRouter_TodaySynonyms(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()
	now := time.Now().In(utcLocation)

	tests := []struct {
		name             string
		query            string
		expectedLabel    string
		expectedStrategy string // 添加预期策略字段
	}{
		{
			name:             "今日日程",
			query:            "今日日程",
			expectedLabel:    "今天", // 同义词，返回原始标签
			expectedStrategy: "schedule_bm25_only", // 纯时间查询
		},
		{
			name:             "明日安排",
			query:            "明日安排",
			expectedLabel:    "明天",
			expectedStrategy: "schedule_bm25_only", // 纯时间查询
		},
		{
			name:             "后日计划",
			query:            "后日计划",
			expectedLabel:    "后天",
			expectedStrategy: "schedule_bm25_only", // "计划"是停用词，视为纯时间查询
		},
		{
			name:             "昨日总结",
			query:            "昨日总结",
			expectedLabel:    "昨天",
			expectedStrategy: "hybrid_with_time_filter", // 包含"总结"内容词
		},
		{
			name:             "前日回顾",
			query:            "前日回顾",
			expectedLabel:    "前天",
			expectedStrategy: "hybrid_with_time_filter", // 包含"回顾"内容词
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(ctx, tt.query)

			// 验证策略
			if decision.Strategy != tt.expectedStrategy {
				t.Errorf("Route() strategy = %v, want %v", decision.Strategy, tt.expectedStrategy)
			}

			// 验证置信度
			if decision.Confidence < 0.90 {
				t.Errorf("Route() confidence = %v, want >= 0.90", decision.Confidence)
			}

			// 验证时间范围存在
			if decision.TimeRange == nil {
				t.Errorf("Route() expected time range for query '%s', got nil", tt.query)
				return
			}

			// 验证标签
			if decision.TimeRange.Label != tt.expectedLabel {
				t.Errorf("Route() timeRange.Label = %v, want %v", decision.TimeRange.Label, tt.expectedLabel)
			}

			// 验证时间范围是今天（对于"今日"）
			if tt.query == "今日日程" {
				expectedStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, utcLocation)
				if decision.TimeRange.Start != expectedStart {
					t.Errorf("Route() TimeRange.Start = %v, want %v", decision.TimeRange.Start, expectedStart)
				}

				expectedEnd := expectedStart.Add(24 * time.Hour)
				if decision.TimeRange.End != expectedEnd {
					t.Errorf("Route() TimeRange.End = %v, want %v", decision.TimeRange.End, expectedEnd)
				}
			}

			// 验证时间范围有效性
			if !decision.TimeRange.ValidateTimeRange() {
				t.Errorf("Route() invalid time range: Start=%v, End=%v",
					decision.TimeRange.Start, decision.TimeRange.End)
			}

			// 记录详细信息
			t.Logf("✓ Query: '%s'", tt.query)
			t.Logf("  Strategy: %s", decision.Strategy)
			t.Logf("  Confidence: %.2f", decision.Confidence)
			t.Logf("  TimeRange: %s (%v to %v)",
				decision.TimeRange.Label,
				decision.TimeRange.Start.Format("15:04"),
				decision.TimeRange.End.Format("15:04"))
			t.Logf("  Duration: %v", decision.TimeRange.Duration())
		})
	}
}

// TestQueryRouter_TodayPerformance 性能测试：对比"今日"vs"今天"
func TestQueryRouter_TodayPerformance(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()

	queries := []string{
		"今天有什么安排",
		"今日日程",
		"明天的计划",
		"明日安排",
	}

	// 预热
	for _, query := range queries {
		router.Route(ctx, query)
	}

	// 性能测试：1000次路由
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		for _, query := range queries {
			router.Route(ctx, query)
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations*len(queries))

	t.Logf("Performance: %d routes in %v", iterations*len(queries), duration)
	t.Logf("Average time per route: %v", avgDuration)

	// 目标：平均路由时间 < 10μs
	if avgDuration > 10*time.Microsecond {
		t.Errorf("Route() too slow: %v, want < 10μs", avgDuration)
	}

	t.Logf("✓ Performance target met: %v < 10μs", avgDuration)
}
