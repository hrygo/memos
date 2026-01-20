package queryengine

import (
	"context"
	"testing"
)

// TestQueryRouter_TodayIssue 测试"今天日程"vs"今天有什么安排"的差异
func TestQueryRouter_TodayIssue(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()

	tests := []struct {
		name string
		query string
	}{
		{
			name:  "今天日程",
			query: "今天日程",
		},
		{
			name:  "今天有什么安排",
			query: "今天有什么安排",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(ctx, tt.query)

			// 打印详细信息
			t.Logf("Query: '%s'", tt.query)
			t.Logf("  Strategy: %s", decision.Strategy)
			t.Logf("  Confidence: %.2f", decision.Confidence)
			if decision.TimeRange != nil {
				t.Logf("  TimeRange: %s", decision.TimeRange.Label)
			} else {
				t.Logf("  TimeRange: nil")
			}
			t.Logf("  SemanticQuery: '%s'", decision.SemanticQuery)

			// 验证策略
			if decision.Strategy != "schedule_bm25_only" {
				t.Errorf("Expected schedule_bm25_only, got %s", decision.Strategy)
			}

			// 验证时间范围存在
			if decision.TimeRange == nil {
				t.Errorf("Expected time range, got nil")
			}
		})
	}
}
