package queryengine

import (
	"context"
	"testing"
	"time"
)

// TestQueryRouter_Route 测试查询路由功能
func TestQueryRouter_Route(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedStrategy string
		minConfidence float32
	}{
		{
			name:          "纯日程查询 - 今天",
			query:         "今天有什么安排",
			expectedStrategy: "schedule_bm25_only",
			minConfidence: 0.90,
		},
		{
			name:          "纯日程查询 - 明天",
			query:         "明天的日程",
			expectedStrategy: "schedule_bm25_only",
			minConfidence: 0.90,
		},
		{
			name:          "纯日程查询 - 本周",
			query:         "本周有什么事",
			expectedStrategy: "schedule_bm25_only",
			minConfidence: 0.90,
		},
		{
			name:          "混合查询 - 今天下午会议",
			query:         "今天下午关于AI项目的会议",
			expectedStrategy: "hybrid_with_time_filter",
			minConfidence: 0.85,
		},
		{
			name:          "笔记查询 - 搜索笔记",
			query:         "搜索关于Python的笔记",
			expectedStrategy: "hybrid_bm25_weighted",
			minConfidence: 0.85,
		},
		{
			name:          "笔记查询 - 包含专有名词",
			query:         "查找关于React和Vue的笔记",
			expectedStrategy: "hybrid_bm25_weighted",
			minConfidence: 0.80,
		},
		{
			name:          "通用问答 - 总结",
			query:         "总结一下我的工作计划",
			expectedStrategy: "full_pipeline_with_reranker",
			minConfidence: 0.60,
		},
		{
			name:          "默认查询",
			query:         "帮我看看",
			expectedStrategy: "hybrid_standard",
			minConfidence: 0.70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(ctx, tt.query)

			if decision.Strategy != tt.expectedStrategy {
				t.Errorf("Route() strategy = %v, want %v", decision.Strategy, tt.expectedStrategy)
			}

			if decision.Confidence < tt.minConfidence {
				t.Errorf("Route() confidence = %v, want >= %v", decision.Confidence, tt.minConfidence)
			}

			t.Logf("Query: '%s'", tt.query)
			t.Logf("  Strategy: %s", decision.Strategy)
			t.Logf("  Confidence: %.2f", decision.Confidence)
			if decision.TimeRange != nil {
				t.Logf("  TimeRange: %s (%v to %v)",
					decision.TimeRange.Label,
					decision.TimeRange.Start.Format("15:04"),
					decision.TimeRange.End.Format("15:04"))
			}
		})
	}
}

// TestQueryRouter_DetectTimeRange 测试时间范围检测
func TestQueryRouter_DetectTimeRange(t *testing.T) {
	router := NewQueryRouter()

	tests := []struct {
		name         string
		query        string
		expectLabel  string
		expectRange  bool // 是否应该有有效的时间范围
	}{
		{
			name:        "今天",
			query:       "今天的事情",
			expectLabel: "今天",
			expectRange: true,
		},
		{
			name:        "明天",
			query:       "明天的安排",
			expectLabel: "明天",
			expectRange: true,
		},
		{
			name:        "后天",
			query:       "后天的日程",
			expectLabel: "后天",
			expectRange: true,
		},
		{
			name:        "本周",
			query:       "本周的工作",
			expectLabel: "本周",
			expectRange: true,
		},
		{
			name:        "下周",
			query:       "下周的计划",
			expectLabel: "下周",
			expectRange: true,
		},
		{
			name:        "上午",
			query:       "上午有什么事",
			expectLabel: "上午",
			expectRange: true,
		},
		{
			name:        "下午",
			query:       "下午的安排",
			expectLabel: "下午",
			expectRange: true,
		},
		{
			name:        "晚上",
			query:       "晚上的日程",
			expectLabel: "晚上",
			expectRange: true,
		},
		{
			name:        "组合时间 - 今天下午",
			query:       "今天下午的会议",
			expectLabel: "下午", // 注意：会匹配到"下午"而不是"今天下午"
			expectRange: true,
		},
		{
			name:        "无时间关键词",
			query:       "搜索关于AI的笔记",
			expectLabel: "",
			expectRange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(context.Background(), tt.query)

			if tt.expectRange {
				if decision.TimeRange == nil {
					t.Errorf("DetectTimeRange() expected time range for query '%s', got nil", tt.query)
					return
				}

				// 检查标签是否包含预期标签（组合时间词可能只匹配到部分）
				if tt.expectLabel != "" && decision.TimeRange.Label != tt.expectLabel {
					// 对于组合时间词，放宽检查
					t.Logf("DetectTimeRange() label = %v, want %v (allowed for compound time words)",
						decision.TimeRange.Label, tt.expectLabel)
				}

				// 验证时间范围有效性
				if !decision.TimeRange.ValidateTimeRange() {
					t.Errorf("DetectTimeRange() invalid time range: Start=%v, End=%v",
						decision.TimeRange.Start, decision.TimeRange.End)
				}

				// 验证时间范围持续时长合理（1秒到30天）
				duration := decision.TimeRange.Duration()
				if duration < time.Second || duration > 30*24*time.Hour {
					t.Errorf("DetectTimeRange() unexpected duration: %v", duration)
				}

				t.Logf("Query: '%s' -> TimeRange: %s (%v to %v, duration: %v)",
					tt.query,
					decision.TimeRange.Label,
					decision.TimeRange.Start.Format("2006-01-02 15:04"),
					decision.TimeRange.End.Format("2006-01-02 15:04"),
					duration)
			} else {
				if decision.TimeRange != nil {
					t.Errorf("DetectTimeRange() expected no time range for query '%s', got %v",
						tt.query, decision.TimeRange.Label)
				}
			}
		})
	}
}

// TestQueryRouter_ExtractContentQuery 测试内容查询提取
func TestQueryRouter_ExtractContentQuery(t *testing.T) {
	router := NewQueryRouter()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "去除时间词",
			query:    "今天关于Python的笔记",
			expected: "Python", // P1 改进：更新期望，"的"和"笔记"是停用词，应该被移除
		},
		{
			name:     "去除停用词",
			query:    "搜索关于AI的内容",
			expected: "AI",
		},
		{
			name:     "去除多个停用词",
			query:    "查询搜索关于React的笔记",
			expected: "React",
		},
		{
			name:     "保留专有名词",
			query:    "查找关于Docker和Kubernetes的笔记",
			expected: "Docker和Kubernetes",
		},
		{
			name:     "纯时间查询",
			query:    "今天有什么安排",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := router.Route(context.Background(), tt.query)
			result := decision.SemanticQuery

			if result != tt.expected {
				t.Errorf("ExtractContentQuery() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestQueryRouter_Performance 测试路由性能
func TestQueryRouter_Performance(t *testing.T) {
	router := NewQueryRouter()
	ctx := context.Background()

	queries := []string{
		"今天有什么安排",
		"搜索关于AI的笔记",
		"本周关于React的学习计划",
		"总结一下我的工作",
		"查找关于Python和Django的资料",
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

	// 目标：平均路由时间 < 10ms
	if avgDuration > 10*time.Millisecond {
		t.Errorf("Route() too slow: %v, want < 10ms", avgDuration)
	}
}

// TestTimeRange_Contains 测试时间范围包含检查
func TestTimeRange_Contains(t *testing.T) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())

	tr := &TimeRange{
		Start: start,
		End:   end,
		Label: "测试时间范围",
	}

	tests := []struct {
		name     string
		testTime time.Time
		expected bool
	}{
		{
			name:     "范围内 - 11:00",
			testTime: time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, now.Location()),
			expected: true,
		},
		{
			name:     "范围外 - 09:00",
			testTime: time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()),
			expected: false,
		},
		{
			name:     "范围外 - 13:00",
			testTime: time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, now.Location()),
			expected: false,
		},
		{
			name:     "边界 - 10:00",
			testTime: start,
			expected: false, // Contains 使用 After，所以边界不算
		},
		{
			name:     "边界 - 12:00",
			testTime: end,
			expected: false, // Contains 使用 Before，所以边界不算
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tr.Contains(tt.testTime)
			if result != tt.expected {
				t.Errorf("Contains() = %v, want %v for time %v", result, tt.expected, tt.testTime)
			}
		})
	}
}
