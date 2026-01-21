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
	return m.schedules, nil
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
	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				UID:       "test-uid-1",
				CreatorID: userID,
				Title:     "Test Meeting",
				StartTs:   now.Add(2 * time.Hour).Unix(),
				EndTs:     func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
				Timezone:  "Asia/Shanghai",
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
	}

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{existingSchedule},
	}
	svc := &service{store: mockStore}

	// Delete schedule
	err := svc.DeleteSchedule(ctx, userID, 1)
	require.NoError(t, err)
}
