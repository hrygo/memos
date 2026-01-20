package main

import (
	"fmt"
	"time"
)

// 分析 1 月 21 日日程查询问题
func main() {
	// 假设当前时间：2026-01-21
	currentDate := time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC)
	fmt.Printf("分析日期: %s\n\n", currentDate.Format("2006-01-02 15:04:05 MST"))

	// 前端查询参数（useSchedulesOptimized）
	anchorDate := currentDate
	startOfRange := time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day()-15, 0, 0, 0, 0, time.UTC)
	endOfRange := time.Date(anchorDate.Year(), anchorDate.Month(), anchorDate.Day()+15, 23, 59, 59, 999999999, time.UTC)

	startTs := startOfRange.Unix()
	endTs := endOfRange.Unix()

	fmt.Printf("前端查询范围:\n")
	fmt.Printf("  开始日期: %s\n", startOfRange.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  startTs: %d\n", startTs)
	fmt.Printf("  结束日期: %s\n", endOfRange.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  endTs:   %d\n\n", endTs)

	// 测试场景：1 月 21 10:00-11:00 的日程
	scheduleStart := time.Date(2026, 1, 21, 10, 0, 0, 0, time.UTC)
	scheduleEnd := time.Date(2026, 1, 21, 11, 0, 0, 0, time.UTC)

	fmt.Printf("\n测试日程: 1 月 21 日 10:00-11:00\n")
	fmt.Printf("  开始: %s\n", scheduleStart.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  start_ts: %d\n", scheduleStart.Unix())
	fmt.Printf("  结束: %s\n", scheduleEnd.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  end_ts:   %d\n\n", scheduleEnd.Unix())

	// 检查查询条件
	fmt.Printf("数据库查询条件 (PostgreSQL):\n")
	fmt.Printf("  WHERE (end_ts >= %d OR end_ts IS NULL)\n", startTs)
	fmt.Printf("    AND start_ts <= %d\n", endTs)

	endTsCondition := "(end_ts >= ? OR end_ts IS NULL)"
	startTsCondition := "start_ts <= ?"

	fmt.Printf("\n验证查询条件:\n")
	fmt.Printf("  %s: %v >= %v = %v\n", endTsCondition, scheduleEnd.Unix(), startTs, scheduleEnd.Unix() >= startTs)
	fmt.Printf("  %s: %v <= %v = %v\n", startTsCondition, scheduleStart.Unix(), endTs, scheduleStart.Unix() <= endTs)

	fmt.Printf("\n结论: 该日程应该被查询到 ✓\n")

	// 额外测试场景
	fmt.Printf("\n其他测试场景:\n")

	scenarios := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected bool
	}{
		{
			name:     "1 月 21 日全天日程",
			start:    time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 1, 21, 23, 59, 59, 999999999, time.UTC),
			expected: true,
		},
		{
			name:     "跨天日程 (1月20-1月22)",
			start:    time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "1 月 6 日（查询范围外）",
			start:    time.Date(2026, 1, 6, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 1, 6, 11, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "2 月 6 日（查询范围外）",
			start:    time.Date(2026, 2, 6, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2026, 2, 6, 11, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, scenario := range scenarios {
		matches := (scenario.end.Unix() >= startTs) && (scenario.start.Unix() <= endTs)
		status := "✓"
		if matches != scenario.expected {
			status = "✗ BUG!"
		}
		fmt.Printf("  %s: %s %v\n", status, scenario.name, matches)
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("请检查后端日志中的 [DEBUG] 输出:\n")
	fmt.Printf("1. [DEBUG] ListSchedules called with find.StartTs=X, find.EndTs=Y\n")
	fmt.Printf("2. [DEBUG] Added start_ts/end_ts conditions\n")
	fmt.Printf("3. [DEBUG] ListSchedules returning N schedules\n")
	fmt.Printf("4. [DEBUG]   [0] 标题: start_ts=X, end_ts=Y\n")
	fmt.Printf("========================================\n")
}
