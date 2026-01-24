package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/usememos/memos/store"
)

// BenchmarkCheckConflicts_SingleSchedule benchmarks conflict detection with a single schedule.
func BenchmarkCheckConflicts_SingleSchedule(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				UID:       "test-uid",
				CreatorID: userID,
				Title:     "Existing Meeting",
				StartTs:   now.Add(2 * time.Hour).Unix(),
				EndTs:     func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
		},
	}
	svc := &service{store: mockStore}

	startTs := now.Add(2 * time.Hour).Unix()
	endTs := now.Add(3 * time.Hour).Unix()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CheckConflicts(ctx, userID, startTs, &endTs, nil)
	}
}

// BenchmarkCheckConflicts_ManySchedules benchmarks conflict detection with 100 schedules.
func BenchmarkCheckConflicts_ManySchedules(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create 100 schedules spread across the day
	schedules := make([]*store.Schedule, 100)
	for i := 0; i < 100; i++ {
		schedules[i] = &store.Schedule{
			ID:        int32(i + 1),
			UID:       "test-uid",
			CreatorID: userID,
			Title:     "Existing Meeting",
			StartTs:   now.Add(time.Duration(i) * 15 * time.Minute).Unix(),
			EndTs:     func() *int64 { ts := now.Add(time.Duration(i)*15*time.Minute + time.Hour).Unix(); return &ts }(),
			RowStatus: store.Normal,
		}
	}

	mockStore := &MockStoreForSchedule{schedules: schedules}
	svc := &service{store: mockStore}

	// Check conflict in the middle of the day
	startTs := now.Add(8 * time.Hour).Unix()
	endTs := now.Add(9 * time.Hour).Unix()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CheckConflicts(ctx, userID, startTs, &endTs, nil)
	}
}

// BenchmarkCreateSchedule_WithoutConflict benchmarks creating a schedule without conflicts.
func BenchmarkCreateSchedule_WithoutConflict(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	mockStore := &MockStoreForSchedule{schedules: []*store.Schedule{}}
	svc := &service{store: mockStore}

	req := &CreateScheduleRequest{
		Title:    "Test Meeting",
		StartTs:  time.Now().Add(2 * time.Hour).Unix(),
		EndTs:    func() *int64 { ts := time.Now().Add(3 * time.Hour).Unix(); return &ts }(),
		Timezone: "Asia/Shanghai",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CreateSchedule(ctx, userID, req)
	}
}

// BenchmarkCreateSchedule_WithRecurring benchmarks creating a recurring schedule.
func BenchmarkCreateSchedule_WithRecurring(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	mockStore := &MockStoreForSchedule{schedules: []*store.Schedule{}}
	svc := &service{store: mockStore}

	// Daily recurrence rule
	dailyRule := `{"type":"daily","interval":1}`

	req := &CreateScheduleRequest{
		Title:          "Daily Standup",
		StartTs:        time.Now().Add(2 * time.Hour).Unix(),
		EndTs:          func() *int64 { ts := time.Now().Add(3 * time.Hour).Unix(); return &ts }(),
		Timezone:       "Asia/Shanghai",
		RecurrenceRule: &dailyRule,
		RecurrenceEndTs: func() *int64 { ts := time.Now().Add(30 * 24 * time.Hour).Unix(); return &ts }(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CreateSchedule(ctx, userID, req)
	}
}

// BenchmarkFindSchedules benchmarks querying schedules within a time range.
func BenchmarkFindSchedules(b *testing.B) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create 50 schedules spread across the week
	schedules := make([]*store.Schedule, 50)
	for i := 0; i < 50; i++ {
		schedules[i] = &store.Schedule{
			ID:        int32(i + 1),
			UID:       "test-uid",
			CreatorID: userID,
			Title:     "Meeting",
			StartTs:   now.Add(time.Duration(i) * 2 * time.Hour).Unix(),
			EndTs:     func() *int64 { ts := now.Add(time.Duration(i)*2*time.Hour + time.Hour).Unix(); return &ts }(),
			RowStatus: store.Normal,
		}
	}

	mockStore := &MockStoreForSchedule{schedules: schedules}
	svc := &service{store: mockStore}

	startTime := now.Add(-24 * time.Hour)
	endTime := now.Add(24 * 7 * time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.FindSchedules(ctx, userID, startTime, endTime)
	}
}
