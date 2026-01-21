package queryengine

import (
	"context"
	"testing"
	"time"
)

// TestQueryRouter_P1_ExplicitYear 测试 P1: 明确年份表达
func TestQueryRouter_P1_ExplicitYear(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedLabel  string
		expectedMode   ScheduleQueryMode
		expectTimeRange bool
	}{
		{
			name:           "YYYY年MM月DD日格式",
			query:          "2025年1月21日的日程",
			expectedLabel:  "2025年1月21日",
			expectedMode:   StrictQueryMode,
			expectTimeRange: true,
		},
		{
			name:           "YYYY-MM-DD格式",
			query:          "2025-01-21有什么安排",
			expectedLabel:  "2025-01-21",
			expectedMode:   StrictQueryMode,
			expectTimeRange: true,
		},
		{
			name:           "YYYY/MM/DD格式",
			query:          "2025/01/21的会议",
			expectedLabel:  "2025/01/21",
			expectedMode:   StrictQueryMode,
			expectTimeRange: true,
		},
		{
			name:           "相对时间-今天",
			query:          "今天的日程",
			expectedLabel:  "今天",
			expectedMode:   StandardQueryMode,
			expectTimeRange: true,
		},
		{
			name:           "相对时间-本周",
			query:          "本周的安排",
			expectedLabel:  "本周",
			expectedMode:   StandardQueryMode,
			expectTimeRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(ctx, tt.query, nil)

			if decision.Strategy != "schedule_bm25_only" && decision.Strategy != "hybrid_with_time_filter" {
				t.Errorf("Expected schedule query strategy, got %v", decision.Strategy)
			}

			if tt.expectTimeRange {
				if decision.TimeRange == nil {
					t.Errorf("Expected time range for query '%s'", tt.query)
					return
				}

				if decision.TimeRange.Label != tt.expectedLabel {
					t.Errorf("TimeRange.Label = %v, want %v", decision.TimeRange.Label, tt.expectedLabel)
				}

				if decision.ScheduleQueryMode != tt.expectedMode {
					t.Errorf("ScheduleQueryMode = %v, want %v", decision.ScheduleQueryMode, tt.expectedMode)
				}

				t.Logf("✓ Query: '%s'", tt.query)
				t.Logf("  Label: %s", decision.TimeRange.Label)
				t.Logf("  Mode: %v", decision.ScheduleQueryMode)
				t.Logf("  Start: %v", decision.TimeRange.Start.Format("2006-01-02"))
				t.Logf("  End: %v", decision.TimeRange.End.Format("2006-01-02"))
			}
		})
	}
}

// TestQueryRouter_P1_FarYearKeywords 测试 P1: 更远的年份关键词
func TestQueryRouter_P1_FarYearKeywords(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()
	now := time.Now().In(utcLocation)

	tests := []struct {
		name           string
		query          string
		expectedLabel  string
		expectedYear   int
		expectTimeRange bool
	}{
		{
			name:           "后年",
			query:          "后年的计划",
			expectedLabel:  "后年",
			expectedYear:   now.Year() + 2,
			expectTimeRange: true,
		},
		{
			name:           "大后年",
			query:          "大后年的目标",
			expectedLabel:  "大后年",
			expectedYear:   now.Year() + 3,
			expectTimeRange: true,
		},
		{
			name:           "前年",
			query:          "前年的数据",
			expectedLabel:  "前年",
			expectedYear:   now.Year() - 2,
			expectTimeRange: true,
		},
		{
			name:           "大前年",
			query:          "大前年的总结",
			expectedLabel:  "大前年",
			expectedYear:   now.Year() - 3,
			expectTimeRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(ctx, tt.query, nil)

			if tt.expectTimeRange {
				if decision.TimeRange == nil {
					t.Errorf("Expected time range for query '%s'", tt.query)
					return
				}

				if decision.TimeRange.Label != tt.expectedLabel {
					t.Errorf("TimeRange.Label = %v, want %v", decision.TimeRange.Label, tt.expectedLabel)
				}

				// 验证年份
				actualYear := decision.TimeRange.Start.Year()
				if actualYear != tt.expectedYear {
					t.Errorf("TimeRange.Start.Year() = %v, want %v", actualYear, tt.expectedYear)
				}

				// 验证持续时间
				duration := decision.TimeRange.End.Sub(decision.TimeRange.Start)
				expectedDuration := 365 * 24 * time.Hour // 约1年
				if duration < expectedDuration-24*time.Hour || duration > expectedDuration+24*time.Hour {
					t.Errorf("Duration = %v, want ~%v", duration, expectedDuration)
				}

				t.Logf("✓ Query: '%s'", tt.query)
				t.Logf("  Label: %s", decision.TimeRange.Label)
				t.Logf("  Year: %d", actualYear)
				t.Logf("  Duration: %v", duration)
			}
		})
	}
}

// TestQueryRouter_P1_QueryModeSelection 测试 P1: 查询模式选择逻辑
func TestQueryRouter_P1_QueryModeSelection(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()

	tests := []struct {
		name     string
		query    string
		expected ScheduleQueryMode
	}{
		{"相对时间-今天", "今天的日程", StandardQueryMode},
		{"相对时间-明天", "明天的安排", StandardQueryMode},
		{"相对时间-本周", "本周的日程", StandardQueryMode},
		{"相对时间-本月", "本月的计划", StandardQueryMode},
		{"相对时间-近期", "近期的安排", StandardQueryMode},
		{"绝对时间-1月21日", "1月21日的会议", StrictQueryMode},
		{"绝对时间-明确年份", "2025年1月21日", StrictQueryMode},
		{"绝对时间-ISO格式", "2025-01-21", StrictQueryMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(ctx, tt.query, nil)

			if decision.TimeRange == nil {
				t.Errorf("Expected time range for query '%s'", tt.query)
				return
			}

			if decision.ScheduleQueryMode != tt.expected {
				t.Errorf("ScheduleQueryMode = %v, want %v for query '%s'", decision.ScheduleQueryMode, tt.expected, tt.query)
			}

			t.Logf("✓ Query: '%s' -> Mode: %v", tt.query, decision.ScheduleQueryMode)
		})
	}
}

// BenchmarkQueryRouter_P1_ExplicitYear 性能测试：明确年份解析
func BenchmarkQueryRouter_P1_ExplicitYear(b *testing.B) {
	router := NewQueryRouter()
	ctx := context.Background()

	queries := []string{
		"2025年1月21日的日程",
		"2025-01-21有什么安排",
		"2025/01/21的会议",
		"今天的日程",
		"本周的安排",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		router.Route(ctx, query, nil)
	}
}

// BenchmarkQueryRouter_P1_FarYear 性能测试：更远年份
func BenchmarkQueryRouter_P1_FarYear(b *testing.B) {
	router := NewQueryRouter()
	ctx := context.Background()

	queries := []string{
		"后年的计划",
		"大后年的目标",
		"前年的数据",
		"大前年的总结",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		router.Route(ctx, query, nil)
	}
}
