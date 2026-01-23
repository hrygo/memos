// Package schedule provides schedule management functionality including creation,
// querying, updating, and deleting schedules with recurring event support.
//
// Key features:
//   - Recurring schedule expansion using RRule
//   - Conflict detection and prevention
//   - Timezone-aware time handling
//
// The service layer abstracts business logic from the store layer and provides
// a clean interface for upper layers.
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/usememos/memos/internal/util"
	aischedule "github.com/usememos/memos/plugin/ai/schedule"
	"github.com/usememos/memos/store"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
)

const (
	// DefaultConflictCheckWindow is the default time window for conflict checking
	DefaultConflictCheckWindow = 1 * time.Hour
)

// Schedule-specific errors that can be checked with errors.Is.
var (
	// ErrScheduleConflict is returned when a schedule conflicts with existing schedules.
	ErrScheduleConflict = fmt.Errorf("schedule conflicts detected")
)

type service struct {
	store Store
}

// Store is the interface for store operations needed by the schedule service.
type Store interface {
	CreateSchedule(ctx context.Context, create *store.Schedule) (*store.Schedule, error)
	ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error)
	GetSchedule(ctx context.Context, find *store.FindSchedule) (*store.Schedule, error)
	UpdateSchedule(ctx context.Context, update *store.UpdateSchedule) error
	DeleteSchedule(ctx context.Context, delete *store.DeleteSchedule) error
}

// NewService creates a new schedule service.
func NewService(store *store.Store) Service {
	return &service{store: store}
}

// FindSchedules returns schedules between start and end time, with recurring schedules expanded.
func (s *service) FindSchedules(ctx context.Context, userID int32, start, end time.Time) ([]*ScheduleInstance, error) {
	// Convert to timestamps
	startTs := start.Unix()
	endTs := end.Unix()

	// Find schedule templates from database
	find := &store.FindSchedule{
		CreatorID: &userID,
		// For recurring schedules, we need to query without time constraints
		// to get the schedule templates, then expand instances
	}

	list, err := s.store.ListSchedules(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// Expand recurring schedules and convert to instances
	var instances []*ScheduleInstance
	maxTotalInstances := MaxInstances // Use package constant
	truncated := false

	for _, sched := range list {
		// Check total instance limit before processing each schedule
		if len(instances) >= maxTotalInstances {
			truncated = true
			break
		}

		// If this is a recurring schedule, expand it
		if sched.RecurrenceRule != nil && *sched.RecurrenceRule != "" {
			// Parse recurrence rule
			rule, err := aischedule.ParseRecurrenceRuleFromJSON(*sched.RecurrenceRule)
			if err != nil {
				// If parsing fails, just return the base schedule
				instances = append(instances, s.convertToInstance(sched, false, ""))
				continue
			}

			// Generate instances starting from the schedule's start time
			// This ensures we get the correct sequence from the first occurrence
			instanceTimestamps := rule.GenerateInstances(sched.StartTs, endTs)

			// For each instance, create a schedule with adjusted time
			for _, instanceTs := range instanceTimestamps {
				// Check if we've hit the total instance limit
				if len(instances) >= maxTotalInstances {
					truncated = true
					break
				}

				// Only add instances within the query window
				if instanceTs < startTs || instanceTs > endTs {
					continue
				}

				instance := s.convertToInstance(sched, true, sched.UID)
				instance.StartTs = instanceTs

				// Calculate end time for this instance
				if sched.EndTs != nil && sched.StartTs > 0 {
					duration := *sched.EndTs - sched.StartTs
					endTsValue := instanceTs + duration
					instance.EndTs = &endTsValue
				}

				instances = append(instances, instance)

				// Break if we've hit the limit
				if len(instances) >= maxTotalInstances {
					truncated = true
					break
				}
			}
		} else {
			// Non-recurring schedule, add as-is if within time range
			// Check if schedule overlaps with query window
			scheduleEnd := sched.EndTs
			if scheduleEnd == nil {
				scheduleEnd = &sched.StartTs
			}

			// Check overlap: query window [startTs, endTs] vs schedule [sched.StartTs, *scheduleEnd]
			overlaps := startTs <= *scheduleEnd && endTs >= sched.StartTs

			if overlaps {
				instances = append(instances, s.convertToInstance(sched, false, ""))
			}
		}
	}

	// Log warning if truncated
	if truncated {
		slog.Warn("schedule instance expansion truncated",
			"count", len(instances),
			"limit", maxTotalInstances,
			"user_id", userID)
	}

	return instances, nil
}

// CreateSchedule creates a new schedule with validation and conflict checking.
func (s *service) CreateSchedule(ctx context.Context, userID int32, create *CreateScheduleRequest) (*store.Schedule, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		slog.Debug("schedule create operation",
			"user_id", userID,
			"title", create.Title,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Validate required fields
	if create.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if create.StartTs <= 0 {
		return nil, fmt.Errorf("start_ts must be a positive timestamp")
	}
	if create.EndTs != nil && *create.EndTs < create.StartTs {
		return nil, fmt.Errorf("end_ts must be greater than or equal to start_ts")
	}

	// Set default timezone if not provided
	timezone := create.Timezone
	if timezone == "" {
		timezone = DefaultTimezone
	}

	// Marshal reminders
	var remindersStr string
	if len(create.Reminders) > 0 {
		reminders := make([]*v1pb.Reminder, len(create.Reminders))
		for i, r := range create.Reminders {
			reminders[i] = &v1pb.Reminder{
				Type:  r.Type,
				Value: r.Value,
				Unit:  r.Unit,
			}
		}
		var err error
		remindersStr, err = aischedule.MarshalReminders(reminders)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal reminders: %w", err)
		}
	} else {
		remindersStr = "[]"
	}

	// Create schedule object
	sched := &store.Schedule{
		UID:            util.GenUUID(),
		CreatorID:      userID,
		Title:          create.Title,
		Description:    create.Description,
		Location:       create.Location,
		StartTs:        create.StartTs,
		EndTs:          create.EndTs,
		AllDay:         create.AllDay,
		Timezone:       timezone,
		RecurrenceRule: create.RecurrenceRule,
		RecurrenceEndTs: create.RecurrenceEndTs,
		Reminders:      &remindersStr,
		RowStatus:      store.Normal,
	}

	// Set default payload
	payloadStr := "{}"
	sched.Payload = &payloadStr

	// Check for conflicts before creating
	conflicts, err := s.CheckConflicts(ctx, userID, create.StartTs, create.EndTs, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrScheduleConflict, buildConflictError(conflicts))
	}

	// Create schedule in database
	created, err := s.store.CreateSchedule(ctx, sched)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return created, nil
}

// UpdateSchedule updates an existing schedule.
func (s *service) UpdateSchedule(ctx context.Context, userID int32, id int32, update *UpdateScheduleRequest) (*store.Schedule, error) {
	// Get existing schedule to verify ownership
	find := &store.FindSchedule{
		ID:        &id,
		CreatorID: &userID,
	}
	existing, err := s.store.GetSchedule(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("schedule not found")
	}

	// Build update request
	storeUpdate := &store.UpdateSchedule{ID: id}

	if update.Title != nil {
		storeUpdate.Title = update.Title
	}
	if update.Description != nil {
		storeUpdate.Description = update.Description
	}
	if update.Location != nil {
		storeUpdate.Location = update.Location
	}
	if update.StartTs != nil {
		storeUpdate.StartTs = update.StartTs
	}
	if update.EndTs != nil {
		storeUpdate.EndTs = update.EndTs
	}
	if update.AllDay != nil {
		storeUpdate.AllDay = update.AllDay
	}
	if update.Timezone != nil {
		storeUpdate.Timezone = update.Timezone
	}
	if update.RecurrenceRule != nil {
		storeUpdate.RecurrenceRule = update.RecurrenceRule
	}
	if update.RecurrenceEndTs != nil {
		storeUpdate.RecurrenceEndTs = update.RecurrenceEndTs
	}
	if update.RowStatus != nil {
		storeUpdate.RowStatus = update.RowStatus
	}
	if update.Reminders != nil {
		reminders := make([]*v1pb.Reminder, len(update.Reminders))
		for i, r := range update.Reminders {
			reminders[i] = &v1pb.Reminder{
				Type:  r.Type,
				Value: r.Value,
				Unit:  r.Unit,
			}
		}
		remindersStr, err := aischedule.MarshalReminders(reminders)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal reminders: %w", err)
		}
		storeUpdate.Reminders = &remindersStr
	}

	// Determine new time values for conflict checking
	newStartTs := existing.StartTs
	newEndTs := existing.EndTs

	if update.StartTs != nil {
		newStartTs = *update.StartTs
	}
	if update.EndTs != nil {
		newEndTs = update.EndTs
	}

	// Check for conflicts (excluding the current schedule itself)
	excludeIDs := []int32{id}
	conflicts, err := s.CheckConflicts(ctx, userID, newStartTs, newEndTs, excludeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrScheduleConflict, buildConflictError(conflicts))
	}

	// Update schedule in database
	if err := s.store.UpdateSchedule(ctx, storeUpdate); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	// Fetch updated schedule
	updated, err := s.store.GetSchedule(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated schedule: %w", err)
	}

	return updated, nil
}

// DeleteSchedule deletes a schedule by ID.
func (s *service) DeleteSchedule(ctx context.Context, userID int32, id int32) error {
	// Get existing schedule to verify ownership
	find := &store.FindSchedule{
		ID:        &id,
		CreatorID: &userID,
	}
	existing, err := s.store.GetSchedule(ctx, find)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("schedule not found")
	}

	// Delete schedule
	if err := s.store.DeleteSchedule(ctx, &store.DeleteSchedule{ID: id}); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

// CheckConflicts checks for schedule conflicts within a time range.
func (s *service) CheckConflicts(ctx context.Context, userID int32, startTs int64, endTs *int64, excludeIDs []int32) ([]*store.Schedule, error) {
	// Determine end time for conflict check
	checkEndTs := startTs
	if endTs != nil && *endTs > startTs {
		checkEndTs = *endTs
	} else {
		// Default to 1 hour from start if not specified
		checkEndTs = startTs + int64(DefaultConflictCheckWindow.Seconds())
	}

	// Find schedules that might conflict within the time window
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   &startTs,
		EndTs:     &checkEndTs,
	}

	list, err := s.store.ListSchedules(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// Filter out excluded schedules and check for actual conflicts
	var conflicts []*store.Schedule
	excludeSet := make(map[int32]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	for _, sched := range list {
		if !excludeSet[sched.ID] {
			// Check if time ranges actually overlap
			// 区间约定 Convention: [start, end) 左闭右开
			// Two intervals [s1, e1) and [s2, e2) overlap if: s1 < e2 AND s2 < e1
			scheduleEnd := sched.EndTs
			if scheduleEnd == nil {
				// For schedules without end time, treat as a point event at start_ts
				scheduleEnd = &sched.StartTs
			}

			// Check overlap: query window [startTs, checkEndTs) vs schedule [sched.StartTs, *scheduleEnd)
			// Using [start, end) convention: overlap when new.start < existing.end AND new.end > existing.start
			if startTs < *scheduleEnd && checkEndTs > sched.StartTs {
				conflicts = append(conflicts, sched)
			}
		}
	}

	return conflicts, nil
}

// Helper functions

// convertToInstance converts a store.Schedule to a ScheduleInstance.
func (s *service) convertToInstance(sched *store.Schedule, isRecurring bool, parentUID string) *ScheduleInstance {
	return &ScheduleInstance{
		ID:          sched.ID,
		UID:         sched.UID,
		Title:       sched.Title,
		Description: sched.Description,
		Location:    sched.Location,
		StartTs:     sched.StartTs,
		EndTs:       sched.EndTs,
		AllDay:      sched.AllDay,
		Timezone:    sched.Timezone,
		IsRecurring: isRecurring,
		ParentUID:   parentUID,
	}
}

// buildConflictError builds a human-readable error message for schedule conflicts.
func buildConflictError(conflicts []*store.Schedule) string {
	if len(conflicts) == 0 {
		return ""
	}

	if len(conflicts) == 1 {
		c := conflicts[0]
		return fmt.Sprintf("conflicts with existing schedule \"%s\" (from %s to %s)",
			c.Title,
			formatTimestamp(c.StartTs, c.Timezone),
			formatEndTs(c.EndTs, c.Timezone))
	}

	var titles []string
	for _, c := range conflicts {
		titles = append(titles, fmt.Sprintf("\"%s\"", c.Title))
	}

	return fmt.Sprintf("conflicts with %d existing schedules: %s",
		len(conflicts),
		strings.Join(titles, ", "))
}

// formatEndTs formats an end timestamp for display in a human-readable format.
func formatEndTs(endTs *int64, timezone string) string {
	if endTs == nil {
		return "no end time"
	}
	return formatTimestamp(*endTs, timezone)
}

// formatTimestamp formats a Unix timestamp for display in the given timezone.
func formatTimestamp(ts int64, timezone string) string {
	t := time.Unix(ts, 0)

	// Parse timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		slog.Warn("invalid timezone, using UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}

	return t.In(loc).Format("2006-01-02 15:04")
}
