// Package stats provides simple local usage statistics for personal assistant systems.
// This is a lightweight alternative to enterprise monitoring solutions.
package stats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hrygo/divinesense/store"
)

// Stats represents usage statistics.
type Stats struct {
	Mu sync.RWMutex

	// Memo stats
	TotalMemos      int64
	MemosLastWeek   int64
	MemosLastMonth  int64

	// Schedule stats
	TotalSchedules   int64
	SchedulesThisWeek int64
	SchedulesNextWeek int64

	// Search stats
	TotalSearches    int64
	SearchesToday   int64
	LastSearchTime  time.Time

	// Activity stats
	ActiveDays      int64 // Days with activity in the last 30 days
	LastActivityTime time.Time
	StreakDays      int64 // Current consecutive days with activity

	// AI stats
	TotalAIQueries     int64
	AIQueriesToday     int64
	AIQueriesThisWeek  int64

	// Timestamp
	LastUpdated time.Time
}

// Collector collects and manages usage statistics.
type Collector struct {
	store    *store.Store
	stats    *Stats
	mu       sync.Mutex
	tickStop chan struct{}
}

// NewCollector creates a new statistics collector.
func NewCollector(st *store.Store) *Collector {
	return &Collector{
		store: st,
		stats: &Stats{
			LastUpdated: time.Now(),
		},
		tickStop: make(chan struct{}),
	}
}

// Start begins periodic statistics collection.
// Updates every hour.
func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Initial collection
	c.collect(ctx)

	go func() {
		for {
			select {
			case <-ticker.C:
				c.collect(ctx)
			case <-ctx.Done():
				close(c.tickStop)
				return
			case <-c.tickStop:
				return
			}
		}
	}()
}

// Stop stops the statistics collector.
func (c *Collector) Stop() {
	select {
	case <-c.tickStop:
		// Already closed
	default:
		close(c.tickStop)
	}
}

// GetStats returns a copy of current statistics.
func (c *Collector) GetStats() *Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	return &Stats{
		TotalMemos:        c.stats.TotalMemos,
		MemosLastWeek:     c.stats.MemosLastWeek,
		MemosLastMonth:    c.stats.MemosLastMonth,
		TotalSchedules:    c.stats.TotalSchedules,
		SchedulesThisWeek: c.stats.SchedulesThisWeek,
		SchedulesNextWeek: c.stats.SchedulesNextWeek,
		TotalSearches:     c.stats.TotalSearches,
		SearchesToday:     c.stats.SearchesToday,
		LastSearchTime:    c.stats.LastSearchTime,
		ActiveDays:        c.stats.ActiveDays,
		LastActivityTime:  c.stats.LastActivityTime,
		StreakDays:        c.stats.StreakDays,
		TotalAIQueries:    c.stats.TotalAIQueries,
		AIQueriesToday:    c.stats.AIQueriesToday,
		AIQueriesThisWeek: c.stats.AIQueriesThisWeek,
		LastUpdated:       c.stats.LastUpdated,
	}
}

// collect gathers current statistics from the store.
func (c *Collector) collect(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, 0, -30)
	thisWeekStart := getWeekStart(now)
	nextWeekStart := thisWeekStart.AddDate(0, 0, 7)

	// Collect memo stats
	memos, err := c.store.ListMemos(ctx, &store.FindMemo{})
	if err == nil {
		c.stats.TotalMemos = int64(len(memos))

		// Count memos in last week
		weekCount := int64(0)
		monthCount := int64(0)
		for _, m := range memos {
			created := time.Unix(m.CreatedTs, 0)
			if created.After(weekAgo) || created.Equal(weekAgo) {
				weekCount++
			}
			if created.After(monthAgo) || created.Equal(monthAgo) {
				monthCount++
			}
		}
		c.stats.MemosLastWeek = weekCount
		c.stats.MemosLastMonth = monthCount
	}

	// Collect schedule stats
	schedules, err := c.store.ListSchedules(ctx, &store.FindSchedule{})
	if err == nil {
		c.stats.TotalSchedules = int64(len(schedules))

		thisWeekCount := int64(0)
		nextWeekCount := int64(0)
		for _, s := range schedules {
			start := time.Unix(s.StartTs, 0)
			if (start.After(thisWeekStart) || start.Equal(thisWeekStart)) &&
				start.Before(nextWeekStart) {
				thisWeekCount++
			}
			if start.After(nextWeekStart) || start.Equal(nextWeekStart) {
				nextWeekCount++
			}
		}
		c.stats.SchedulesThisWeek = thisWeekCount
		c.stats.SchedulesNextWeek = nextWeekCount
	}

	// Calculate activity statistics
	// Find the latest activity time across memos and schedules
	lastMemoTime := time.Time{}
	if len(memos) > 0 {
		for _, m := range memos {
			created := time.Unix(m.CreatedTs, 0)
			if created.After(lastMemoTime) {
				lastMemoTime = created
			}
		}
	}

	lastScheduleTime := time.Time{}
	if len(schedules) > 0 {
		for _, s := range schedules {
			created := time.Unix(s.StartTs, 0)
			if created.After(lastScheduleTime) {
				lastScheduleTime = created
			}
		}
	}

	// Set last activity time
	if lastMemoTime.After(lastScheduleTime) {
		c.stats.LastActivityTime = lastMemoTime
	} else if lastScheduleTime.After(time.Time{}) {
		c.stats.LastActivityTime = lastScheduleTime
	}

	// Calculate active days (days with activity in last 30 days)
	activeDaysMap := make(map[string]bool)
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	for _, m := range memos {
		created := time.Unix(m.CreatedTs, 0)
		if created.After(thirtyDaysAgo) || created.Equal(thirtyDaysAgo) {
			dateKey := created.Format("2006-01-02")
			activeDaysMap[dateKey] = true
		}
	}
	for _, s := range schedules {
		created := time.Unix(s.StartTs, 0)
		if created.After(thirtyDaysAgo) || created.Equal(thirtyDaysAgo) {
			dateKey := created.Format("2006-01-02")
			activeDaysMap[dateKey] = true
		}
	}
	c.stats.ActiveDays = int64(len(activeDaysMap))

	// Calculate streak days (consecutive days with activity ending today)
	c.stats.StreakDays = c.calculateStreakDays(ctx, now)

	c.stats.LastUpdated = now
}

// RecordSearch records a search action.
func (c *Collector) RecordSearch() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.TotalSearches++
	c.stats.SearchesToday++
	c.stats.LastSearchTime = time.Now()
}

// RecordAIQuery records an AI query action.
func (c *Collector) RecordAIQuery() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.TotalAIQueries++
	c.stats.AIQueriesToday++
	c.stats.AIQueriesThisWeek++
}

// GetSummary returns a human-readable summary.
func (s *Stats) GetSummary() string {
	return fmt.Sprintf(
		`📊 使用统计 (更新于: %s)

📝 笔记
  总计: %d 条
  最近一周: %d 条
  最近一月: %d 条

📅 日程
  总计: %d 个
  本周: %d 个
  下周: %d 个

🔍 搜索
  总计: %d 次
  今日: %d 次

🤖 AI 查询
  总计: %d 次
  今日: %d 次
  本周: %d 次

📈 活跃度
  活跃天数 (30天): %d 天
  连续天数: %d 天
  最后活动: %s`,
		s.LastUpdated.Format("2006-01-02 15:04"),
		s.TotalMemos,
		s.MemosLastWeek,
		s.MemosLastMonth,
		s.TotalSchedules,
		s.SchedulesThisWeek,
		s.SchedulesNextWeek,
		s.TotalSearches,
		s.SearchesToday,
		s.TotalAIQueries,
		s.AIQueriesToday,
		s.AIQueriesThisWeek,
		s.ActiveDays,
		s.StreakDays,
		formatLastActivity(s.LastActivityTime),
	)
}

func formatLastActivity(t time.Time) string {
	if t.IsZero() {
		return "无"
	}
	duration := time.Since(t)
	if duration < time.Hour {
		return "刚刚"
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(duration.Hours()))
	}
	if duration < 7*24*time.Hour {
		return fmt.Sprintf("%d天前", int(duration.Hours()/24))
	}
	return t.Format("2006-01-02")
}

func getWeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.Truncate(24 * time.Hour).AddDate(0, 0, -weekday+1)
}

// calculateStreakDays calculates the current streak of consecutive days with activity.
// A day is considered active if there's at least one memo or schedule created.
func (c *Collector) calculateStreakDays(ctx context.Context, now time.Time) int64 {
	streak := int64(0)

	// Check each day going backwards from today
	for i := 0; i < 365; i++ { // Check up to a year
		checkDate := now.AddDate(0, 0, -i)
		dayStart := checkDate.Truncate(24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)

		// Check if there's any memo or schedule on this day
		hasActivity := false

		// Check memos
		memos, err := c.store.ListMemos(ctx, &store.FindMemo{})
		if err == nil {
			for _, m := range memos {
				created := time.Unix(m.CreatedTs, 0)
				if (created.After(dayStart) || created.Equal(dayStart)) && created.Before(dayEnd) {
					hasActivity = true
					break
				}
			}
		}

		// Check schedules if no memo found
		if !hasActivity {
			schedules, err := c.store.ListSchedules(ctx, &store.FindSchedule{})
			if err == nil {
				for _, s := range schedules {
					created := time.Unix(s.StartTs, 0)
					if (created.After(dayStart) || created.Equal(dayStart)) && created.Before(dayEnd) {
						hasActivity = true
						break
					}
				}
			}
		}

		if !hasActivity {
			break
		}

		streak++
	}

	return streak
}
