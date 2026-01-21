package queryengine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestQueryRouter_P1_Integration 测试 P1: 查询模式集成
// 验证从 RouteDecision 到 RetrievalOptions 的模式传递
func TestQueryRouter_P1_Integration(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()

	tests := []struct {
		name             string
		query            string
		expectedMode     ScheduleQueryMode
		hasTimeRange     bool
	}{
		{
			name:         "相对时间-今天",
			query:        "今天的日程",
			expectedMode: StandardQueryMode,
			hasTimeRange: true,
		},
		{
			name:         "相对时间-本周",
			query:        "本周的安排",
			expectedMode: StandardQueryMode,
			hasTimeRange: true,
		},
		{
			name:         "绝对时间-1月21日",
			query:        "1月21日的会议",
			expectedMode: StrictQueryMode,
			hasTimeRange: true,
		},
		{
			name:         "明确年份-2025年1月21日",
			query:        "2025年1月21日的日程",
			expectedMode: StrictQueryMode,
			hasTimeRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(ctx, tt.query, nil)

			// 验证模式选择正确
			assert.Equal(t, tt.expectedMode, decision.ScheduleQueryMode,
				"ScheduleQueryMode mismatch for query '%s'", tt.query)

			if tt.hasTimeRange {
				assert.NotNil(t, decision.TimeRange,
					"Expected time range for query '%s'", tt.query)

				// 验证时间范围有效
				assert.True(t, decision.TimeRange.ValidateTimeRange(),
					"Time range validation failed for query '%s'", tt.query)
			}

			t.Logf("✓ Query: '%s'", tt.query)
			t.Logf("  Mode: %v", decision.ScheduleQueryMode)
			if decision.TimeRange != nil {
				t.Logf("  TimeRange: %s (%v to %v)",
					decision.TimeRange.Label,
					decision.TimeRange.Start.Format("15:04"),
					decision.TimeRange.End.Format("15:04"))
			}
		})
	}
}

// TestQueryRouter_P1_ModeMapping 测试 P1: 模式到 int32 的映射
// 验证 ScheduleQueryMode 可以正确转换为 store.FindSchedule 的 QueryMode 字段
func TestQueryRouter_P1_ModeMapping(t *testing.T) {
	tests := []struct {
		name   string
		mode   ScheduleQueryMode
		expect int32
	}{
		{"AUTO maps to 0", AutoQueryMode, 0},
		{"STANDARD maps to 1", StandardQueryMode, 1},
		{"STRICT maps to 2", StrictQueryMode, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证枚举值映射
			actual := int32(tt.mode)
			assert.Equal(t, tt.expect, actual,
				"ScheduleQueryMode %v should map to %d", tt.mode, tt.expect)
		})
	}
}

// BenchmarkQueryRouter_P1_CompleteRoute P1 完整路由性能测试
// 包含模式选择和明确年份解析的性能
func BenchmarkQueryRouter_P1_CompleteRoute(b *testing.B) {
	router := NewQueryRouter()
	ctx := context.Background()

	queries := []string{
		"今天的日程",
		"本周的安排",
		"1月21日的会议",
		"2025年1月21日的日程",
		"后年的计划",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		router.Route(ctx, query, nil)
	}
}

// TestP1_FeatureComplete P1 功能完整性检查
func TestP1_FeatureComplete(t *testing.T) {
	// 这个测试验证 P1 的所有核心功能是否已实现

	t.Run("ScheduleQueryMode exists", func(t *testing.T) {
		// 验证类型定义存在
		assert.NotNil(t, AutoQueryMode, "AutoQueryMode should be defined")
		assert.NotNil(t, StandardQueryMode, "StandardQueryMode should be defined")
		assert.NotNil(t, StrictQueryMode, "StrictQueryMode should be defined")

		// 验证值
		assert.Equal(t, ScheduleQueryMode(0), AutoQueryMode)
		assert.Equal(t, ScheduleQueryMode(1), StandardQueryMode)
		assert.Equal(t, ScheduleQueryMode(2), StrictQueryMode)
	})

	t.Run("RouteDecision has ScheduleQueryMode", func(t *testing.T) {
		router := NewQueryRouter()
		ctx := context.Background()

		decision := router.Route(ctx, "今天的日程", nil)
		assert.NotNil(t, decision.ScheduleQueryMode, "RouteDecision should have ScheduleQueryMode")
	})

	t.Run("Explicit year parsing works", func(t *testing.T) {
		router := NewQueryRouter()
		ctx := context.Background()

		// 测试 2025年1月21日 格式
		decision := router.Route(ctx, "2025年1月21日的日程", nil)
		assert.NotNil(t, decision.TimeRange, "Should parse explicit year")
		assert.Equal(t, "2025年1月21日", decision.TimeRange.Label)
	})

	t.Run("Far year keywords work", func(t *testing.T) {
		router := NewQueryRouter()
		ctx := context.Background()
		now := time.Now().In(time.UTC)
		expectedYear := now.Year() + 2

		// 测试"后年"
		decision := router.Route(ctx, "后年的计划", nil)
		assert.NotNil(t, decision.TimeRange, "Should parse '后年'")
		assert.Equal(t, "后年", decision.TimeRange.Label)
		assert.Equal(t, expectedYear, decision.TimeRange.Start.Year())
	})

	t.Run("Mode selection logic works", func(t *testing.T) {
		router := NewQueryRouter()
		ctx := context.Background()

		// 相对时间 → Standard
		decision1 := router.Route(ctx, "今天的日程", nil)
		assert.Equal(t, StandardQueryMode, decision1.ScheduleQueryMode)

		// 绝对时间 → Strict
		decision2 := router.Route(ctx, "1月21日的会议", nil)
		assert.Equal(t, StrictQueryMode, decision2.ScheduleQueryMode)
	})
}
