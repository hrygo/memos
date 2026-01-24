package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/usememos/memos/store"
)

// MockStoreForSchedule is a mock implementation of the Store interface for testing.
type MockStoreForSchedule struct {
	schedules []*store.Schedule
}

func (m *MockStoreForSchedule) CreateSchedule(ctx context.Context, create *store.Schedule) (*store.Schedule, error) {
	m.schedules = append(m.schedules, create)
	return create, nil
}

func (m *MockStoreForSchedule) ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error) {
	result := make([]*store.Schedule, 0)

	for _, sched := range m.schedules {
		// Filter by creator_id
		if find.CreatorID != nil && sched.CreatorID != *find.CreatorID {
			continue
		}

		// Filter by row_status - only apply if explicitly set in find
		if find.RowStatus != nil && sched.RowStatus != *find.RowStatus {
			continue
		}

		// Filter by ID
		if find.ID != nil && sched.ID != *find.ID {
			continue
		}

		// Filter by UID
		if find.UID != nil && sched.UID != *find.UID {
			continue
		}

		// Time range filtering (simplified - just checking basic overlap)
		// Note: FindSchedules doesn't set StartTs/EndTs, it does filtering in-memory
		if find.StartTs != nil || find.EndTs != nil {
			schedEnd := sched.EndTs
			if schedEnd == nil {
				ts := sched.StartTs
				schedEnd = &ts
			}
			// Check if schedule overlaps with query range
			if find.StartTs != nil && *find.StartTs > *schedEnd {
				continue
			}
			if find.EndTs != nil && *find.EndTs < sched.StartTs {
				continue
			}
		}

		result = append(result, sched)
	}

	return result, nil
}

func (m *MockStoreForSchedule) GetSchedule(ctx context.Context, find *store.FindSchedule) (*store.Schedule, error) {
	if len(m.schedules) > 0 {
		return m.schedules[0], nil
	}
	return nil, nil
}

func (m *MockStoreForSchedule) UpdateSchedule(ctx context.Context, update *store.UpdateSchedule) error {
	for _, s := range m.schedules {
		if s.ID == update.ID {
			if update.Title != nil {
				s.Title = *update.Title
			}
			if update.Description != nil {
				s.Description = *update.Description
			}
			if update.StartTs != nil {
				s.StartTs = *update.StartTs
			}
			if update.EndTs != nil {
				s.EndTs = update.EndTs
			}
			if update.Location != nil {
				s.Location = *update.Location
			}
			if update.Timezone != nil {
				s.Timezone = *update.Timezone
			}
			break
		}
	}
	return nil
}

func (m *MockStoreForSchedule) DeleteSchedule(ctx context.Context, delete *store.DeleteSchedule) error {
	return nil
}

// TestFindSchedules tests the FindSchedules method.
func TestFindSchedules(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create mock store with test data
	// Use a fixed time within today to avoid edge cases near midnight
	scheduleStart := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, time.Local)
	scheduleEnd := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, time.Local)

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				UID:       "test-uid-1",
				CreatorID: userID,
				Title:     "Test Meeting",
				StartTs:   scheduleStart.Unix(),
				EndTs:     func() *int64 { ts := scheduleEnd.Unix(); return &ts }(),
				Timezone:  "Asia/Shanghai",
				RowStatus: store.Normal,
			},
		},
	}

	// Create service
	svc := &service{store: mockStore}

	// Test FindSchedules - query for today
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	endTime := startTime.Add(24 * time.Hour)

	instances, err := svc.FindSchedules(ctx, userID, startTime, endTime)
	require.NoError(t, err)
	assert.Greater(t, len(instances), 0, "Should find at least one schedule")

	// Verify non-recurring schedule is found
	found := false
	for _, inst := range instances {
		if inst.Title == "Test Meeting" {
			found = true
			assert.False(t, inst.IsRecurring, "Test Meeting should not be recurring")
		}
	}
	assert.True(t, found, "Should find Test Meeting")
}

// TestCreateSchedule tests the CreateSchedule method.
func TestCreateSchedule(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)

	mockStore := &MockStoreForSchedule{}
	svc := &service{store: mockStore}

	// Test creating a valid schedule
	req := &CreateScheduleRequest{
		Title:       "Test Meeting",
		Description: "Test Description",
		Location:    "Conference Room",
		StartTs:     time.Now().Add(2 * time.Hour).Unix(),
		EndTs:       func() *int64 { ts := time.Now().Add(3 * time.Hour).Unix(); return &ts }(),
		Timezone:    "Asia/Shanghai",
	}

	created, err := svc.CreateSchedule(ctx, userID, req)
	require.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, "Test Meeting", created.Title)
	assert.Equal(t, "Test Description", created.Description)
	assert.Equal(t, "Conference Room", created.Location)
}

// TestCreateScheduleValidation tests validation in CreateSchedule.
func TestCreateScheduleValidation(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)

	mockStore := &MockStoreForSchedule{}
	svc := &service{store: mockStore}

	tests := []struct {
		name    string
		req     *CreateScheduleRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing title",
			req: &CreateScheduleRequest{
				StartTs: time.Now().Add(2 * time.Hour).Unix(),
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "end time before start time",
			req: &CreateScheduleRequest{
				Title:   "Test",
				StartTs: time.Now().Add(3 * time.Hour).Unix(),
				EndTs:   func() *int64 { ts := time.Now().Add(2 * time.Hour).Unix(); return &ts }(),
			},
			wantErr: true,
			errMsg:  "end_ts must be greater than or equal to start_ts",
		},
		{
			name: "invalid start time",
			req: &CreateScheduleRequest{
				Title:   "Test",
				StartTs: -1,
			},
			wantErr: true,
			errMsg:  "start_ts must be a positive timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateSchedule(ctx, userID, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckConflicts tests the CheckConflicts method.
func TestCheckConflicts(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create a schedule that conflicts
	conflictingSchedule := &store.Schedule{
		ID:        1,
		UID:       "conflicting-uid",
		CreatorID: userID,
		Title:     "Existing Meeting",
		StartTs:   now.Add(2 * time.Hour).Unix(),
		EndTs:     func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
		RowStatus: store.Normal,
	}

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{conflictingSchedule},
	}
	svc := &service{store: mockStore}

	// Check for conflicts - should find the existing schedule
	conflicts, err := svc.CheckConflicts(ctx, userID, now.Add(2*time.Hour).Unix(), func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(), nil)
	require.NoError(t, err)
	assert.Len(t, conflicts, 1, "Should find one conflict")
	assert.Equal(t, "Existing Meeting", conflicts[0].Title)
}

// TestUpdateSchedule tests the UpdateSchedule method.
func TestUpdateSchedule(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)

	existingSchedule := &store.Schedule{
		ID:        1,
		UID:       "test-uid",
		CreatorID: userID,
		Title:     "Original Title",
		StartTs:   time.Now().Add(2 * time.Hour).Unix(),
		EndTs:     func() *int64 { ts := time.Now().Add(3 * time.Hour).Unix(); return &ts }(),
		RowStatus: store.Normal,
	}

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{existingSchedule},
	}
	svc := &service{store: mockStore}

	// Update title
	newTitle := "Updated Title"
	updateReq := &UpdateScheduleRequest{
		Title: &newTitle,
	}

	updated, err := svc.UpdateSchedule(ctx, userID, 1, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)
}

// TestDeleteSchedule tests the DeleteSchedule method.
func TestDeleteSchedule(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)

	existingSchedule := &store.Schedule{
		ID:        1,
		UID:       "test-uid",
		CreatorID: userID,
		Title:     "To Be Deleted",
		StartTs:   time.Now().Add(2 * time.Hour).Unix(),
		RowStatus: store.Normal,
	}

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{existingSchedule},
	}
	svc := &service{store: mockStore}

	// Delete schedule
	err := svc.DeleteSchedule(ctx, userID, 1)
	require.NoError(t, err)
}

// TestCheckConflicts_WithArchivedSchedules tests that archived schedules don't trigger conflicts.
func TestCheckConflicts_WithArchivedSchedules(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create an archived schedule that overlaps
	archivedSchedule := &store.Schedule{
		ID:        1,
		UID:       "archived-uid",
		CreatorID: userID,
		Title:     "Archived Meeting",
		StartTs:   now.Add(2 * time.Hour).Unix(),
		EndTs:     func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
		RowStatus: store.Archived,
	}

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{archivedSchedule},
	}
	svc := &service{store: mockStore}

	// Check for conflicts - should NOT find the archived schedule
	conflicts, err := svc.CheckConflicts(ctx, userID, now.Add(2*time.Hour).Unix(), func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(), nil)
	require.NoError(t, err)
	assert.Len(t, conflicts, 0, "Archived schedules should not trigger conflicts")
}

// TestCheckConflicts_ActiveScheduleOnly tests that active schedules do trigger conflicts.
func TestCheckConflicts_ActiveScheduleOnly(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create an active schedule that overlaps
	activeSchedule := &store.Schedule{
		ID:        1,
		UID:       "active-uid",
		CreatorID: userID,
		Title:     "Active Meeting",
		StartTs:   now.Add(2 * time.Hour).Unix(),
		EndTs:     func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
		RowStatus: store.Normal,
	}

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{activeSchedule},
	}
	svc := &service{store: mockStore}

	// Check for conflicts - SHOULD find the active schedule
	conflicts, err := svc.CheckConflicts(ctx, userID, now.Add(2*time.Hour).Unix(), func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(), nil)
	require.NoError(t, err)
	assert.Len(t, conflicts, 1, "Active schedules should trigger conflicts")
	assert.Equal(t, "Active Meeting", conflicts[0].Title)
}

// TestCheckConflicts_ExcludeIDs tests that excluded schedules are ignored.
func TestCheckConflicts_ExcludeIDs(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	schedule1 := &store.Schedule{
		ID:        1,
		UID:       "uid-1",
		CreatorID: userID,
		Title:     "Meeting 1",
		StartTs:   now.Add(2 * time.Hour).Unix(),
		EndTs:     func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
		RowStatus: store.Normal,
	}

	schedule2 := &store.Schedule{
		ID:        2,
		UID:       "uid-2",
		CreatorID: userID,
		Title:     "Meeting 2",
		StartTs:   now.Add(2 * time.Hour).Unix(),
		EndTs:     func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
		RowStatus: store.Normal,
	}

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{schedule1, schedule2},
	}
	svc := &service{store: mockStore}

	// Check for conflicts excluding schedule1 - should only find schedule2
	conflicts, err := svc.CheckConflicts(ctx, userID, now.Add(2*time.Hour).Unix(), func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(), []int32{1})
	require.NoError(t, err)
	assert.Len(t, conflicts, 1, "Only non-excluded schedules should trigger conflicts")
	assert.Equal(t, "Meeting 2", conflicts[0].Title)
}

// TestCheckConflicts_IntervalConvention tests the [start, end) interval convention.
func TestCheckConflicts_IntervalConvention(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	tests := []struct {
		name           string
		existingStart  time.Time
		existingEnd    time.Time
		newStart       time.Time
		newEnd         time.Time
		expectConflict bool
	}{
		{
			name:           "exact overlap",
			existingStart:  now.Add(2 * time.Hour),
			existingEnd:    now.Add(3 * time.Hour),
			newStart:       now.Add(2 * time.Hour),
			newEnd:         now.Add(3 * time.Hour),
			expectConflict: true,
		},
		{
			name:           "partial overlap - new starts during existing",
			existingStart:  now.Add(2 * time.Hour),
			existingEnd:    now.Add(3 * time.Hour),
			newStart:       now.Add(2 * time.Hour + 30*time.Minute),
			newEnd:         now.Add(3 * time.Hour + 30*time.Minute),
			expectConflict: true,
		},
		{
			name:           "partial overlap - new ends during existing",
			existingStart:  now.Add(2 * time.Hour + 30*time.Minute),
			existingEnd:    now.Add(4 * time.Hour),
			newStart:       now.Add(2 * time.Hour),
			newEnd:         now.Add(3 * time.Hour),
			expectConflict: true,
		},
		{
			name:           "adjacent - new ends when existing starts (no conflict)",
			existingStart:  now.Add(3 * time.Hour),
			existingEnd:    now.Add(4 * time.Hour),
			newStart:       now.Add(2 * time.Hour),
			newEnd:         now.Add(3 * time.Hour),
			expectConflict: false, // [2,3) and [3,4) don't overlap
		},
		{
			name:           "adjacent - new starts when existing ends (no conflict)",
			existingStart:  now.Add(2 * time.Hour),
			existingEnd:    now.Add(3 * time.Hour),
			newStart:       now.Add(3 * time.Hour),
			newEnd:         now.Add(4 * time.Hour),
			expectConflict: false, // [2,3) and [3,4) don't overlap
		},
		{
			name:           "no overlap - new before existing",
			existingStart:  now.Add(3 * time.Hour),
			existingEnd:    now.Add(4 * time.Hour),
			newStart:       now.Add(1 * time.Hour),
			newEnd:         now.Add(2 * time.Hour),
			expectConflict: false,
		},
		{
			name:           "no overlap - new after existing",
			existingStart:  now.Add(1 * time.Hour),
			existingEnd:    now.Add(2 * time.Hour),
			newStart:       now.Add(3 * time.Hour),
			newEnd:         now.Add(4 * time.Hour),
			expectConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existingSchedule := &store.Schedule{
				ID:        1,
				UID:       "existing-uid",
				CreatorID: userID,
				Title:     "Existing Meeting",
				StartTs:   tt.existingStart.Unix(),
				EndTs:     func() *int64 { ts := tt.existingEnd.Unix(); return &ts }(),
				RowStatus: store.Normal,
			}

			mockStore := &MockStoreForSchedule{
				schedules: []*store.Schedule{existingSchedule},
			}
			svc := &service{store: mockStore}

			conflicts, err := svc.CheckConflicts(ctx, userID, tt.newStart.Unix(), func() *int64 { ts := tt.newEnd.Unix(); return &ts }(), nil)
			require.NoError(t, err)

			if tt.expectConflict {
				assert.Len(t, conflicts, 1, "Expected conflict for: %s", tt.name)
			} else {
				assert.Len(t, conflicts, 0, "Expected no conflict for: %s", tt.name)
			}
		})
	}
}
