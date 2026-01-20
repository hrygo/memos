package store

import (
	"context"
	"time"
)

// Schedule is the object representing a schedule.
type Schedule struct {
	ID              int32
	UID             string
	CreatorID       int32
	RowStatus       RowStatus
	CreatedTs       int64
	UpdatedTs       int64
	Title           string
	Description     string
	Location        string
	StartTs         int64
	EndTs           *int64
	AllDay          bool
	Timezone        string
	RecurrenceRule  *string
	RecurrenceEndTs *int64
	Reminders       *string
	Payload         *string
}

// FindSchedule is the find condition for schedule.
type FindSchedule struct {
	ID        *int32
	UID       *string
	CreatorID *int32

	// Time range filters
	StartTs *int64
	EndTs   *int64

	// Status filter
	RowStatus *RowStatus

	// Pagination
	Limit  *int
	Offset *int
}

// UpdateSchedule is the update request for schedule.
type UpdateSchedule struct {
	ID              int32
	UID             *string
	CreatedTs       *int64
	UpdatedTs       *int64
	RowStatus       *RowStatus
	Title           *string
	Description     *string
	Location        *string
	StartTs         *int64
	EndTs           *int64
	AllDay          *bool
	Timezone        *string
	RecurrenceRule  *string
	RecurrenceEndTs *int64
	Reminders       *string
	Payload         *string
}

// DeleteSchedule is the delete request for schedule.
type DeleteSchedule struct {
	ID int32
}

// CreateSchedule creates a new schedule.
func (s *Store) CreateSchedule(ctx context.Context, create *Schedule) (*Schedule, error) {
	return s.driver.CreateSchedule(ctx, create)
}

// ListSchedules lists schedules with filter.
func (s *Store) ListSchedules(ctx context.Context, find *FindSchedule) ([]*Schedule, error) {
	return s.driver.ListSchedules(ctx, find)
}

// GetSchedule gets a schedule by uid.
func (s *Store) GetSchedule(ctx context.Context, find *FindSchedule) (*Schedule, error) {
	list, err := s.driver.ListSchedules(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// UpdateSchedule updates a schedule.
func (s *Store) UpdateSchedule(ctx context.Context, update *UpdateSchedule) error {
	return s.driver.UpdateSchedule(ctx, update)
}

// DeleteSchedule deletes a schedule.
func (s *Store) DeleteSchedule(ctx context.Context, delete *DeleteSchedule) error {
	return s.driver.DeleteSchedule(ctx, delete)
}

// ScheduleService is the interface for schedule-related operations.
type ScheduleService interface {
	CreateSchedule(ctx context.Context, create *Schedule) (*Schedule, error)
	ListSchedules(ctx context.Context, find *FindSchedule) ([]*Schedule, error)
	UpdateSchedule(ctx context.Context, update *UpdateSchedule) error
	DeleteSchedule(ctx context.Context, delete *DeleteSchedule) error
}

// Helper functions for schedule time operations

// ParseStartTime parses the schedule start time to time.Time.
func (s *Schedule) ParseStartTime() time.Time {
	return time.Unix(s.StartTs, 0)
}

// ParseEndTime parses the schedule end time to time.Time.
func (s *Schedule) ParseEndTime() *time.Time {
	if s.EndTs == nil {
		return nil
	}
	t := time.Unix(*s.EndTs, 0)
	return &t
}

// IsAllDay returns true if the schedule is an all-day event.
func (s *Schedule) IsAllDay() bool {
	return s.AllDay
}

// GetDuration returns the duration of the schedule.
func (s *Schedule) GetDuration() *time.Duration {
	if s.EndTs == nil {
		return nil
	}
	duration := time.Unix(*s.EndTs, 0).Sub(time.Unix(s.StartTs, 0))
	return &duration
}

// IsActiveAt checks if the schedule is active at the given time.
func (s *Schedule) IsActiveAt(ts int64) bool {
	if ts < s.StartTs {
		return false
	}
	if s.EndTs == nil {
		// All-day events or events without end time
		return s.StartTs <= ts
	}
	return ts <= *s.EndTs
}

// ConflictWith checks if this schedule conflicts with another.
func (s *Schedule) ConflictWith(other *Schedule) bool {
	// Check if time ranges overlap
	sEnd := s.EndTs
	if sEnd == nil {
		sEnd = &s.StartTs
	}
	otherEnd := other.EndTs
	if otherEnd == nil {
		otherEnd = &other.StartTs
	}

	// Two intervals [s1, e1] and [s2, e2] overlap if:
	// s1 <= e2 AND s2 <= e1
	return s.StartTs <= *otherEnd && other.StartTs <= *sEnd
}
