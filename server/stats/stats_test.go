package stats

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/usememos/memos/store/test"
)

func TestCollector_Collect(t *testing.T) {
	ctx := context.Background()
	ts := test.NewTestingStore(ctx, t)
	defer ts.Close()

	collector := NewCollector(ts)
	collector.collect(ctx)

	stats := collector.GetStats()

	// Stats should be initialized
	if stats.LastUpdated.IsZero() {
		t.Error("LastUpdated should be set")
	}
}

func TestStats_GetSummary(t *testing.T) {
	stats := &Stats{
		TotalMemos:        100,
		MemosLastWeek:     10,
		MemosLastMonth:    30,
		TotalSchedules:    20,
		SchedulesThisWeek: 3,
		SchedulesNextWeek: 2,
		TotalSearches:     500,
		SearchesToday:     15,
		TotalAIQueries:    50,
		AIQueriesToday:    5,
		AIQueriesThisWeek: 20,
		ActiveDays:        25,
		StreakDays:        7,
		LastActivityTime:  time.Now(),
		LastUpdated:       time.Now(),
	}

	summary := stats.GetSummary()

	// Check that summary contains key information
	if len(summary) == 0 {
		t.Error("Summary should not be empty")
	}

	// Check for key sections
	sections := []string{
		"📝 笔记", "📅 日程", "🔍 搜索",
		"🤖 AI 查询", "📈 活跃度",
	}

	for _, section := range sections {
		if !strings.Contains(summary, section) {
			t.Errorf("Summary should contain section: %s", section)
		}
	}
}

func TestCollector_RecordSearch(t *testing.T) {
	ctx := context.Background()
	ts := test.NewTestingStore(ctx, t)
	defer ts.Close()

	collector := NewCollector(ts)

	initialStats := collector.GetStats()
	if initialStats.TotalSearches != 0 {
		t.Errorf("Initial TotalSearches should be 0, got %d", initialStats.TotalSearches)
	}

	collector.RecordSearch()
	collector.RecordSearch()

	stats := collector.GetStats()
	if stats.TotalSearches != 2 {
		t.Errorf("TotalSearches should be 2 after recording 2 searches, got %d", stats.TotalSearches)
	}
	if stats.SearchesToday != 2 {
		t.Errorf("SearchesToday should be 2, got %d", stats.SearchesToday)
	}
}

func TestCollector_RecordAIQuery(t *testing.T) {
	ctx := context.Background()
	ts := test.NewTestingStore(ctx, t)
	defer ts.Close()

	collector := NewCollector(ts)

	collector.RecordAIQuery()
	collector.RecordAIQuery()
	collector.RecordAIQuery()

	stats := collector.GetStats()
	if stats.TotalAIQueries != 3 {
		t.Errorf("TotalAIQueries should be 3, got %d", stats.TotalAIQueries)
	}
	if stats.AIQueriesToday != 3 {
		t.Errorf("AIQueriesToday should be 3, got %d", stats.AIQueriesToday)
	}
	if stats.AIQueriesThisWeek != 3 {
		t.Errorf("AIQueriesThisWeek should be 3, got %d", stats.AIQueriesThisWeek)
	}
}

