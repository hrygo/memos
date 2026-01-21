package queryengine

import (
	"context"
	"testing"
	"time"
)

// BenchmarkQueryRouter_Route 基准测试：路由性能
// P2 改进：建立性能基准
func BenchmarkQueryRouter_Route(b *testing.B) {
	router := NewQueryRouter()
	ctx := context.Background()

	queries := []string{
		"今天有什么安排",
		"搜索关于AI的笔记",
		"本周关于React的学习计划",
		"总结一下我的工作",
		"查找关于Python和Django的资料",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		router.Route(ctx, query, nil)
	}
}

// BenchmarkQueryRouter_Route_Parallel 并发路由性能测试
func BenchmarkQueryRouter_Route_Parallel(b *testing.B) {
	router := NewQueryRouter()
	ctx := context.Background()

	queries := []string{
		"今天有什么安排",
		"搜索关于AI的笔记",
		"本周关于React的学习计划",
		"总结一下我的工作",
		"查找关于Python和Django的资料",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			query := queries[i%len(queries)]
			router.Route(ctx, query, nil)
			i++
		}
	})
}

// BenchmarkQueryRouter_DetectTimeRange 时间检测性能
func BenchmarkQueryRouter_DetectTimeRange(b *testing.B) {
	router := NewQueryRouter()

	queries := []string{
		"今天的事情",
		"明天的安排",
		"后天的日程",
		"本周的工作",
		"下周的计划",
		"上午有什么事",
		"下午的安排",
		"晚上的日程",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		router.detectTimeRange(query)
	}
}

// BenchmarkQueryRouter_ExtractContentQuery 内容提取性能
func BenchmarkQueryRouter_ExtractContentQuery(b *testing.B) {
	router := NewQueryRouter()

	queries := []string{
		"今天关于Python的笔记",
		"搜索关于AI的内容",
		"查询搜索关于React的笔记",
		"查找关于Docker和Kubernetes的笔记",
		"今天关于Vue的学习笔记",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		router.extractContentQuery(query)
	}
}

// BenchmarkQueryRouter_CheckMostlyProperNouns 专有名词检测性能
func BenchmarkQueryRouter_CheckMostlyProperNouns(b *testing.B) {
	router := NewQueryRouter()

	queries := []string{
		"Python和Django",
		"React和Vue.js",
		"Docker和Kubernetes",
		"AI和Machine Learning",
		"JavaScript和TypeScript",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		router.CheckMostlyProperNouns(query)
	}
}

// BenchmarkTimeRange_ValidateTimeRange 时间范围验证性能
func BenchmarkTimeRange_ValidateTimeRange(b *testing.B) {
	now := time.Now()

	tr := &TimeRange{
		Start: now,
		End:   now.Add(24 * time.Hour),
		Label: "今天",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.ValidateTimeRange()
	}
}

// BenchmarkQueryRouter_ConcurrentConfig 并发配置读写性能
func BenchmarkQueryRouter_ConcurrentConfig(b *testing.B) {
	config := DefaultConfig()
	router := NewQueryRouterWithConfig(config)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 80% 读操作，20% 写操作
			if i%5 == 0 {
				newConfig := DefaultConfig()
				newConfig.QueryLimits.MaxQueryLength = 2000
				router.ApplyConfig(newConfig)
			} else {
				_ = router.GetConfig()
			}
			i++
		}
	})
}
