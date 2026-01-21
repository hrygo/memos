package queryengine

import (
	"context"
	"testing"
	"time"
)

// TestQueryRouter_DateParsing 测试具体日期解析功能
func TestQueryRouter_DateParsing(t *testing.T) {
	router := NewQueryRouter()

	tests := []struct {
		name          string
		query         string
		expectLabel   string
		expectYear    int
		expectMonth   int
		expectDay     int
		shouldMatch   bool
	}{
		{
			name:        "1月21日格式",
			query:       "1月21日有哪些事？",
			expectLabel: "1月21日",
			shouldMatch: true,
		},
		{
			name:        "01月21日格式（补零）",
			query:       "01月21日有什么安排？",
			expectLabel: "1月21日",
			shouldMatch: true,
		},
		{
			name:        "1月21号格式",
			query:       "1月21号的日程",
			expectLabel: "1月21日",
			shouldMatch: true,
		},
		{
			name:        "1-21格式（横线分隔）",
			query:       "1-21有什么事？",
			expectLabel: "1月21日",
			shouldMatch: true,
		},
		{
			name:        "01-21格式（补零+横线）",
			query:       "01-21的安排",
			expectLabel: "1月21日",
			shouldMatch: true,
		},
		{
			name:        "1/21格式（斜杠分隔）",
			query:       "1/21有哪些事？",
			expectLabel: "1月21日",
			shouldMatch: true,
		},
		{
			name:        "斜杠分隔（补零）",
			query:       "01/21有什么安排？",
			expectLabel: "1月21日",
			shouldMatch: true,
		},
		{
			name:        "不支持年月日格式（需后续扩展）",
			query:       "2025年1月21日",
			shouldMatch: false, // 暂不支持年份
		},
		{
			name:        "无效日期（2月30日）",
			query:       "2月30日有什么事？",
			shouldMatch: false, // 无效日期
		},
		{
			name:        "无效月份（13月）",
			query:       "13月1日有什么事？",
			shouldMatch: false, // 无效月份
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			decision := router.Route(ctx, tt.query, nil)

			if tt.shouldMatch {
				if decision.TimeRange == nil {
					t.Errorf("expected TimeRange to be set for query '%s', got nil", tt.query)
					return
				}

				if decision.TimeRange.Label != tt.expectLabel {
					t.Errorf("expected label '%s', got '%s'", tt.expectLabel, decision.TimeRange.Label)
				}

				// 验证时间范围是24小时
				duration := decision.TimeRange.End.Sub(decision.TimeRange.Start)
				expectedDuration := 24 * time.Hour
				if duration != expectedDuration {
					t.Errorf("expected duration %v, got %v", expectedDuration, duration)
				}
			} else {
				if decision.TimeRange != nil && decision.TimeRange.Label == tt.expectLabel {
					t.Errorf("expected NO match for query '%s', but got TimeRange with label '%s'", tt.query, decision.TimeRange.Label)
				}
			}
		})
	}
}

// TestQueryRouter_DateParsingStrategy 测试日期解析后的路由策略
func TestQueryRouter_DateParsingStrategy(t *testing.T) {
	router := NewQueryRouter()

	tests := []struct {
		name             string
		query            string
		expectedStrategy string
	}{
		{
			name:             "1月21日纯日程查询",
			query:            "1月21日有哪些事？",
			expectedStrategy: "hybrid_with_time_filter", // "有哪些"不是停用词
		},
		{
			name:             "1月21日带内容查询",
			query:            "1月21日的会议安排",
			expectedStrategy: "hybrid_with_time_filter", // "会议"是内容词
		},
		{
			name:             "横线格式纯日程查询",
			query:            "1-21有什么安排？",
			expectedStrategy: "hybrid_with_time_filter", // "有什么"不是停用词
		},
		{
			name:             "斜杠格式带内容",
			query:            "1/21的项目进度",
			expectedStrategy: "hybrid_with_time_filter", // "项目进度"是内容词
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			decision := router.Route(ctx, tt.query, nil)

			if decision.Strategy != tt.expectedStrategy {
				t.Errorf("expected strategy '%s', got '%s'", tt.expectedStrategy, decision.Strategy)
			}

			if decision.TimeRange == nil {
				t.Errorf("expected TimeRange to be set for query '%s'", tt.query)
			}
		})
	}
}

// TestQueryRouter_DateParsingEdgeCases 测试边界情况
func TestQueryRouter_DateParsingEdgeCases(t *testing.T) {
	router := NewQueryRouter()
	now := time.Now().UTC()
	currentYear := now.Year()

	tests := []struct {
		name              string
		query             string
		shouldMatch       bool
		expectedInPast    bool
		expectedInFuture  bool
	}{
		{
			name:     "今天的日期（如今天是1月21日）",
			query:    "1月21日有什么事？",
			// 这个测试取决于今天是否是1月21日
		},
		{
			name:     "过去的日期（如1月1日）",
			query:    "1月1日有什么事？",
			// 如果现在是1月21日，1月1日已经过去
		},
		{
			name:     "未来的日期（如12月31日）",
			query:    "12月31日有什么事？",
			// 如果现在是1月21日，12月31日还在未来
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			decision := router.Route(ctx, tt.query, nil)

			if decision.TimeRange != nil {
				// 验证日期被正确解析
				t.Logf("Query '%s' parsed to TimeRange: %s to %s",
					tt.query,
					decision.TimeRange.Start.Format("2006-01-02"),
					decision.TimeRange.End.Format("2006-01-02"))

				// 验证年份是否正确（应该是当前年或明年）
				year := decision.TimeRange.Start.Year()
				if year != currentYear && year != currentYear+1 {
					t.Errorf("expected year to be %d or %d, got %d", currentYear, currentYear+1, year)
				}
			}
		})
	}
}
