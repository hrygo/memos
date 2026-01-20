package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/internal/util"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

// ScheduleService provides schedule management APIs.
type ScheduleService struct {
	v1pb.UnimplementedScheduleServiceServer

	Store *store.Store
}

// scheduleFromStore converts a store.Schedule to v1pb.Schedule.
func scheduleFromStore(s *store.Schedule) *v1pb.Schedule {
	pb := &v1pb.Schedule{
		Name:       fmt.Sprintf("schedules/%s", s.UID),
		Title:      s.Title,
		StartTs:    s.StartTs,
		AllDay:     s.AllDay,
		Timezone:   s.Timezone,
		CreatedTs:  s.CreatedTs,
		UpdatedTs:  s.UpdatedTs,
		State:      s.RowStatus.String(),
	}

	if s.Description != "" {
		pb.Description = s.Description
	}
	if s.Location != "" {
		pb.Location = s.Location
	}
	if s.EndTs != nil {
		pb.EndTs = *s.EndTs
	}
	if s.RecurrenceRule != nil {
		pb.RecurrenceRule = *s.RecurrenceRule
	}
	if s.RecurrenceEndTs != nil {
		pb.RecurrenceEndTs = *s.RecurrenceEndTs
	}
	if s.CreatorID != 0 {
		pb.Creator = fmt.Sprintf("users/%d", s.CreatorID)
	}

	// Parse reminders from JSON
	if s.Reminders != nil && *s.Reminders != "" && *s.Reminders != "[]" {
		var reminders []map[string]interface{}
		if err := json.Unmarshal([]byte(*s.Reminders), &reminders); err == nil {
			for _, r := range reminders {
				reminder := &v1pb.Reminder{}
				if t, ok := r["type"].(string); ok {
					reminder.Type = t
				}
				if v, ok := r["value"].(float64); ok {
					reminder.Value = int32(v)
				}
				if u, ok := r["unit"].(string); ok {
					reminder.Unit = u
				}
				pb.Reminders = append(pb.Reminders, reminder)
			}
		}
	}

	return pb
}

// scheduleToStore converts a v1pb.Schedule to store.Schedule.
func scheduleToStore(pb *v1pb.Schedule, creatorID int32) (*store.Schedule, error) {
	// Parse UID from name
	uid := strings.TrimPrefix(pb.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	s := &store.Schedule{
		UID:         uid,
		CreatorID:   creatorID,
		Title:       pb.Title,
		StartTs:     pb.StartTs,
		AllDay:      pb.AllDay,
		Timezone:    pb.Timezone,
		RowStatus:   store.RowStatus(pb.State),
		Description: pb.Description,
		Location:    pb.Location,
	}

	if pb.EndTs != 0 {
		s.EndTs = &pb.EndTs
	}
	if pb.RecurrenceRule != "" {
		s.RecurrenceRule = &pb.RecurrenceRule
	}
	if pb.RecurrenceEndTs != 0 {
		s.RecurrenceEndTs = &pb.RecurrenceEndTs
	}

	// Convert reminders to JSON
	var remindersStr string
	if len(pb.Reminders) > 0 {
		reminders := make([]map[string]interface{}, len(pb.Reminders))
		for i, r := range pb.Reminders {
			reminders[i] = map[string]interface{}{
				"type":  r.Type,
				"value": r.Value,
				"unit":  r.Unit,
			}
		}
		remindersJSON, err := json.Marshal(reminders)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal reminders: %v", err)
		}
		remindersStr = string(remindersJSON)
	} else {
		remindersStr = "[]"
	}
	s.Reminders = &remindersStr

	// Set default payload
	payloadStr := "{}"
	s.Payload = &payloadStr

	return s, nil
}

// CreateSchedule creates a new schedule.
func (s *ScheduleService) CreateSchedule(ctx context.Context, req *v1pb.CreateScheduleRequest) (*v1pb.Schedule, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	schedule, err := scheduleToStore(req.Schedule, userID)
	if err != nil {
		return nil, err
	}

	// Generate UID if not provided
	if schedule.UID == "" || schedule.UID == "schedules/" {
		schedule.UID = util.GenUUID()
	}

	created, err := s.Store.CreateSchedule(ctx, schedule)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create schedule: %v", err)
	}

	return scheduleFromStore(created), nil
}

// ListSchedules lists schedules with filters.
func (s *ScheduleService) ListSchedules(ctx context.Context, req *v1pb.ListSchedulesRequest) (*v1pb.ListSchedulesResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	find := &store.FindSchedule{
		Limit: pointerOf(100), // Default limit
	}

	// Parse creator from name
	if req.Creator != "" {
		creatorID := strings.TrimPrefix(req.Creator, "users/")
		if creatorID == "" {
			return nil, status.Errorf(codes.InvalidArgument, "invalid creator format")
		}
		id := parseInt32(creatorID)
		find.CreatorID = &id
	} else {
		// Default to current user
		find.CreatorID = &userID
	}

	if req.StartTs != 0 {
		find.StartTs = &req.StartTs
	}
	if req.EndTs != 0 {
		find.EndTs = &req.EndTs
	}
	if req.State != "" {
		rowStatus := store.RowStatus(req.State)
		find.RowStatus = &rowStatus
	}
	if req.PageSize != 0 {
		limit := int(req.PageSize)
		find.Limit = &limit
	}

	list, err := s.Store.ListSchedules(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list schedules: %v", err)
	}

	response := &v1pb.ListSchedulesResponse{
		Schedules: make([]*v1pb.Schedule, len(list)),
	}
	for i, s := range list {
		response.Schedules[i] = scheduleFromStore(s)
	}

	return response, nil
}

// GetSchedule gets a schedule by name.
func (s *ScheduleService) GetSchedule(ctx context.Context, req *v1pb.GetScheduleRequest) (*v1pb.Schedule, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	uid := strings.TrimPrefix(req.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	find := &store.FindSchedule{
		UID:       &uid,
		CreatorID: &userID,
	}

	schedule, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get schedule: %v", err)
	}
	if schedule == nil {
		return nil, status.Errorf(codes.NotFound, "schedule not found")
	}

	return scheduleFromStore(schedule), nil
}

// UpdateSchedule updates a schedule.
func (s *ScheduleService) UpdateSchedule(ctx context.Context, req *v1pb.UpdateScheduleRequest) (*v1pb.Schedule, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	uid := strings.TrimPrefix(req.Schedule.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	// Get existing schedule
	find := &store.FindSchedule{
		UID:       &uid,
		CreatorID: &userID,
	}
	existing, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get schedule: %v", err)
	}
	if existing == nil {
		return nil, status.Errorf(codes.NotFound, "schedule not found")
	}

	// Build update request
	update := &store.UpdateSchedule{
		ID: existing.ID,
	}

	if req.UpdateMask != nil {
		for _, path := range req.UpdateMask.Paths {
			switch path {
			case "title":
				update.Title = &req.Schedule.Title
			case "description":
				update.Description = &req.Schedule.Description
			case "location":
				update.Location = &req.Schedule.Location
			case "start_ts":
				update.StartTs = &req.Schedule.StartTs
			case "end_ts":
				if req.Schedule.EndTs != 0 {
					update.EndTs = &req.Schedule.EndTs
				}
			case "all_day":
				update.AllDay = &req.Schedule.AllDay
			case "timezone":
				update.Timezone = &req.Schedule.Timezone
			case "recurrence_rule":
				update.RecurrenceRule = &req.Schedule.RecurrenceRule
			case "recurrence_end_ts":
				if req.Schedule.RecurrenceEndTs != 0 {
					update.RecurrenceEndTs = &req.Schedule.RecurrenceEndTs
				}
			case "state":
				rowStatus := store.RowStatus(req.Schedule.State)
				update.RowStatus = &rowStatus
			case "reminders":
				// Convert reminders to JSON
				if len(req.Schedule.Reminders) > 0 {
					reminders := make([]map[string]interface{}, len(req.Schedule.Reminders))
					for i, r := range req.Schedule.Reminders {
						reminders[i] = map[string]interface{}{
							"type":  r.Type,
							"value": r.Value,
							"unit":  r.Unit,
						}
					}
					remindersJSON, err := json.Marshal(reminders)
					if err != nil {
						return nil, status.Errorf(codes.Internal, "failed to marshal reminders: %v", err)
					}
					remindersStr := string(remindersJSON)
					update.Reminders = &remindersStr
				}
			}
		}
	} else {
		// If no UpdateMask provided, update all non-zero/non-empty fields
		if req.Schedule.Title != "" {
			update.Title = &req.Schedule.Title
		}
		if req.Schedule.Description != "" {
			update.Description = &req.Schedule.Description
		}
		if req.Schedule.Location != "" {
			update.Location = &req.Schedule.Location
		}
		if req.Schedule.StartTs != 0 {
			update.StartTs = &req.Schedule.StartTs
		}
		if req.Schedule.EndTs != 0 {
			update.EndTs = &req.Schedule.EndTs
		}
		// Always update boolean if provided
		update.AllDay = &req.Schedule.AllDay
		if req.Schedule.Timezone != "" {
			update.Timezone = &req.Schedule.Timezone
		}
		if req.Schedule.RecurrenceRule != "" {
			update.RecurrenceRule = &req.Schedule.RecurrenceRule
		}
		if req.Schedule.RecurrenceEndTs != 0 {
			update.RecurrenceEndTs = &req.Schedule.RecurrenceEndTs
		}
		if req.Schedule.State != "" {
			rowStatus := store.RowStatus(req.Schedule.State)
			update.RowStatus = &rowStatus
		}
		// Convert reminders to JSON if provided
		if len(req.Schedule.Reminders) > 0 {
			reminders := make([]map[string]interface{}, len(req.Schedule.Reminders))
			for i, r := range req.Schedule.Reminders {
				reminders[i] = map[string]interface{}{
					"type":  r.Type,
					"value": r.Value,
					"unit":  r.Unit,
				}
			}
			remindersJSON, err := json.Marshal(reminders)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to marshal reminders: %v", err)
			}
			remindersStr := string(remindersJSON)
			update.Reminders = &remindersStr
		}
	}

	if err := s.Store.UpdateSchedule(ctx, update); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update schedule: %v", err)
	}

	// Fetch updated schedule
	updated, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated schedule: %v", err)
	}

	return scheduleFromStore(updated), nil
}

// DeleteSchedule deletes a schedule.
func (s *ScheduleService) DeleteSchedule(ctx context.Context, req *v1pb.DeleteScheduleRequest) (*emptypb.Empty, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	uid := strings.TrimPrefix(req.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	// Get existing schedule to verify ownership
	find := &store.FindSchedule{
		UID:       &uid,
		CreatorID: &userID,
	}
	existing, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get schedule: %v", err)
	}
	if existing == nil {
		return nil, status.Errorf(codes.NotFound, "schedule not found")
	}

	if err := s.Store.DeleteSchedule(ctx, &store.DeleteSchedule{ID: existing.ID}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete schedule: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// CheckConflict checks for schedule conflicts.
func (s *ScheduleService) CheckConflict(ctx context.Context, req *v1pb.CheckConflictRequest) (*v1pb.CheckConflictResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Find schedules that might conflict
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   &req.StartTs,
	}

	endTs := req.EndTs
	if endTs == 0 {
		// Default to 1 hour from start
		endTs = req.StartTs + 3600
	}
	find.EndTs = &endTs

	list, err := s.Store.ListSchedules(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check conflicts: %v", err)
	}

	// Filter out excluded schedules
	var conflicts []*store.Schedule
	excludeSet := make(map[string]bool)
	for _, name := range req.ExcludeNames {
		excludeSet[name] = true
	}

	for _, s := range list {
		name := fmt.Sprintf("schedules/%s", s.UID)
		if !excludeSet[name] {
			// Check if time ranges overlap
			sEnd := s.EndTs
			if sEnd == nil {
				sEnd = &s.StartTs
			}
			if req.StartTs <= *sEnd && endTs >= s.StartTs {
				conflicts = append(conflicts, s)
			}
		}
	}

	response := &v1pb.CheckConflictResponse{
		Conflicts: make([]*v1pb.Schedule, len(conflicts)),
	}
	for i, c := range conflicts {
		response.Conflicts[i] = scheduleFromStore(c)
	}

	return response, nil
}

// ParseAndCreateSchedule parses natural language and creates a schedule.
func (s *ScheduleService) ParseAndCreateSchedule(ctx context.Context, req *v1pb.ParseAndCreateScheduleRequest) (*v1pb.ParseAndCreateScheduleResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// TODO: Implement natural language parsing
	// For now, return Unimplemented
	return nil, status.Errorf(codes.Unimplemented, "natural language parsing not yet implemented")
}

// Helper functions

func pointerOf[T any](v T) *T {
	return &v
}

func parseInt32(s string) int32 {
	var i int32
	fmt.Sscanf(s, "%d", &i)
	return i
}
