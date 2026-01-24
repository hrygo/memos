package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/usememos/memos/store"
)

// BenchmarkConflictResolver_Resolve_NoConflict benchmarks conflict resolution with no conflicts.
func BenchmarkConflictResolver_Resolve_NoConflict(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now().Truncate(time.Hour)

	// Empty mock store - no existing schedules
	mockStore := &MockStoreForSchedule{schedules: []*store.Schedule{}}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	startTime := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	duration := time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.Resolve(ctx, userID, startTime, time.Time{}, duration)
		require.NoError(b, err)
	}
}

// BenchmarkConflictResolver_Resolve_WithConflict benchmarks conflict resolution with conflicts.
func BenchmarkConflictResolver_Resolve_WithConflict(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now().Truncate(time.Hour)

	// Create 50 schedules to simulate a busy day
	var schedules []*store.Schedule
	for hour := 8; hour < 22; hour++ {
		start := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.Local)
		end := time.Date(now.Year(), now.Month(), now.Day(), hour+1, 0, 0, 0, time.Local)
		schedules = append(schedules, &store.Schedule{
			ID:        int32(hour),
			UID:       "busy-" + string(rune('0'+hour)),
			CreatorID: userID,
			Title:     "Busy Slot",
			StartTs:   start.Unix(),
			EndTs:     func() *int64 { ts := end.Unix(); return &ts }(),
			RowStatus: store.Normal,
		})
	}

	mockStore := &MockStoreForSchedule{schedules: schedules}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Request a busy slot
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	duration := time.Hour

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.Resolve(ctx, userID, startTime, time.Time{}, duration)
		require.NoError(b, err)
	}
}

// BenchmarkConflictResolver_FindAllFreeSlots benchmarks finding all free slots.
func BenchmarkConflictResolver_FindAllFreeSlots(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()
	testDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Create schedules occupying 9:00-10:00 and 14:00-15:00
	busy1Start := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local)
	busy1End := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	busy2Start := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, time.Local)
	busy2End := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, time.Local)

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				CreatorID: userID,
				Title:     "Morning Meeting",
				StartTs:   busy1Start.Unix(),
				EndTs:     func() *int64 { ts := busy1End.Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
			{
				ID:        2,
				CreatorID: userID,
				Title:     "Afternoon Meeting",
				StartTs:   busy2Start.Unix(),
				EndTs:     func() *int64 { ts := busy2End.Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
		},
	}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.FindAllFreeSlots(ctx, userID, testDate, time.Hour)
		require.NoError(b, err)
	}
}

// BenchmarkConflictResolver_FindAllFreeSlots_FullyBooked benchmarks with fully booked day.
func BenchmarkConflictResolver_FindAllFreeSlots_FullyBooked(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()
	testDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Create schedules that occupy the entire day (8:00-22:00)
	var schedules []*store.Schedule
	for hour := 8; hour < 22; hour++ {
		start := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.Local)
		end := time.Date(now.Year(), now.Month(), now.Day(), hour+1, 0, 0, 0, time.Local)
		schedules = append(schedules, &store.Schedule{
			ID:        int32(hour),
			CreatorID: userID,
			Title:     "Busy Slot",
			StartTs:   start.Unix(),
			EndTs:     func() *int64 { ts := end.Unix(); return &ts }(),
			RowStatus: store.Normal,
		})
	}

	mockStore := &MockStoreForSchedule{schedules: schedules}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.FindAllFreeSlots(ctx, userID, testDate, time.Hour)
		require.NoError(b, err)
	}
}

// BenchmarkConflictResolver_CalculateScore benchmarks the scoring function.
func BenchmarkConflictResolver_CalculateScore(b *testing.B) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local)
	resolver := &ConflictResolver{}

	slot := TimeSlot{
		Start: time.Date(2026, 1, 15, 9, 0, 0, 0, time.Local),
		End:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resolver.calculateScore(now, slot)
	}
}

// BenchmarkConflictResolver_ScoreAlternatives benchmarks scoring and sorting alternatives.
func BenchmarkConflictResolver_ScoreAlternatives(b *testing.B) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local)
	resolver := &ConflictResolver{}

	// Create 50 alternative time slots
	alternatives := make([]TimeSlot, 50)
	for i := 0; i < 50; i++ {
		hour := 8 + (i % 14) // 8:00 to 21:00
		dayOffset := i / 14
		alternatives[i] = TimeSlot{
			Start: time.Date(2026, 1, 15+dayOffset, hour, 0, 0, 0, time.Local),
			End:   time.Date(2026, 1, 15+dayOffset, hour+1, 0, 0, 0, time.Local),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resolver.scoreAlternatives(now, alternatives)
	}
}

// BenchmarkCheckRecurringConflicts benchmarks recurring conflict detection.
func BenchmarkCheckRecurringConflicts(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create some conflicting schedules
	var schedules []*store.Schedule
	for day := 0; day < 7; day++ {
		start := time.Date(now.Year(), now.Month(), now.Day()+day, 10, 0, 0, 0, time.Local)
		end := time.Date(now.Year(), now.Month(), now.Day()+day, 11, 0, 0, 0, time.Local)
		schedules = append(schedules, &store.Schedule{
			ID:        int32(day),
			CreatorID: userID,
			Title:     "Daily Meeting",
			StartTs:   start.Unix(),
			EndTs:     func() *int64 { ts := end.Unix(); return &ts }(),
			RowStatus: store.Normal,
		})
	}

	mockStore := &MockStoreForSchedule{schedules: schedules}
	svc := &service{store: mockStore}

	// Create a recurring schedule request
	rule := `{"frequency": "DAILY", "count": 30}`
	startTs := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local).Unix()
	endTs := startTs + 3600

	createReq := &CreateScheduleRequest{
		Title:          "Recurring Event",
		StartTs:        startTs,
		EndTs:          &endTs,
		RecurrenceRule: &rule,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.checkRecurringConflicts(ctx, userID, createReq)
	}
}

// BenchmarkBuildConflictIndex benchmarks the conflict index building.
func BenchmarkBuildConflictIndex(b *testing.B) {
	// Create 100 schedules
	var schedules []*store.Schedule
	for i := 0; i < 100; i++ {
		hour := 8 + (i % 14)
		day := i / 14
		start := time.Date(2026, 1, 1+day, hour, 0, 0, 0, time.Local)
		end := time.Date(2026, 1, 1+day, hour+1, 0, 0, 0, time.Local)
		schedules = append(schedules, &store.Schedule{
			ID:        int32(i),
			CreatorID: 1,
			Title:     "Schedule",
			StartTs:   start.Unix(),
			EndTs:     func() *int64 { ts := end.Unix(); return &ts }(),
			RowStatus: store.Normal,
		})
	}

	svc := &service{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.buildConflictIndex(schedules)
	}
}

// BenchmarkHasConflictInIndex benchmarks the conflict lookup in index.
func BenchmarkHasConflictInIndex(b *testing.B) {
	// Create 100 schedules and build index
	var schedules []*store.Schedule
	for i := 0; i < 100; i++ {
		hour := 8 + (i % 14)
		day := i / 14
		start := time.Date(2026, 1, 1+day, hour, 0, 0, 0, time.Local)
		end := time.Date(2026, 1, 1+day, hour+1, 0, 0, 0, time.Local)
		schedules = append(schedules, &store.Schedule{
			ID:        int32(i),
			CreatorID: 1,
			Title:     "Schedule",
			StartTs:   start.Unix(),
			EndTs:     func() *int64 { ts := end.Unix(); return &ts }(),
			RowStatus: store.Normal,
		})
	}

	svc := &service{}
	index := svc.buildConflictIndex(schedules)

	startTs := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local).Unix()
	endTs := startTs + 3600

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.hasConflictInIndex(index, startTs, endTs)
	}
}
