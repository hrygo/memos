package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/usememos/memos/server/service/schedule"
)

const (
	// DefaultTimezone is used when no timezone is specified
	DefaultTimezone = "Asia/Shanghai"

	// maxTimezoneCacheEntries limits the cache size to prevent unbounded growth.
	// Realistically, there are ~500 IANA timezones, but 100 is more than enough
	// for typical usage while preventing potential DoS via malicious inputs.
	maxTimezoneCacheEntries = 100

	// Audit log field length limits (for sensitive data sanitization)
	maxTitleLengthForLog       = 50
	maxDescriptionLengthForLog = 100
)

// timezoneCache caches parsed timezone locations for performance.
// Uses a simple size-limited cache that resets when full (LRU-lite).
var timezoneCache struct {
	sync.RWMutex
	locations map[string]*time.Location
	hits      int64 // Cache hit counter for metrics
	misses    int64 // Cache miss counter for metrics
}

// init initializes the timezone cache.
func init() {
	timezoneCache.locations = make(map[string]*time.Location)
	// Pre-load common timezone
	if loc, err := time.LoadLocation(DefaultTimezone); err == nil {
		timezoneCache.locations[DefaultTimezone] = loc
	}
}

// getTimezoneLocation gets a timezone location from cache, loading it if necessary.
// Implements size-limited caching to prevent unbounded growth.
func getTimezoneLocation(timezone string) *time.Location {
	// Fast path: read lock for cache hit
	timezoneCache.RLock()
	loc, ok := timezoneCache.locations[timezone]
	timezoneCache.RUnlock()

	if ok {
		return loc
	}

	// Slow path: load and cache with write lock
	timezoneCache.Lock()
	defer timezoneCache.Unlock()

	// Double-check after acquiring write lock
	if loc, ok := timezoneCache.locations[timezone]; ok {
		return loc
	}

	// Enforce cache size limit: if full, reset and keep common timezone
	if len(timezoneCache.locations) >= maxTimezoneCacheEntries {
		slog.Debug("timezone cache full, resetting",
			"current_size", len(timezoneCache.locations),
			"max_size", maxTimezoneCacheEntries,
		)
		// Reset cache, keeping only the default timezone
		timezoneCache.locations = make(map[string]*time.Location)
		if loc, err := time.LoadLocation(DefaultTimezone); err == nil {
			timezoneCache.locations[DefaultTimezone] = loc
		}
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Warn("failed to load timezone, using UTC",
			"timezone", timezone,
			"error", err,
		)
		loc = time.UTC
	}

	timezoneCache.locations[timezone] = loc
	return loc
}

// JSON field name mappings for camelCase to snake_case compatibility.
// Some LLMs generate camelCase (startTime) while we expect snake_case (start_time).
var fieldNameMappings = map[string]string{
	"startTime": "start_time",
	"endTime":   "end_time",
	"allDay":    "all_day",
	"minScore":  "min_score",
}

// normalizeJSONFields converts camelCase keys to snake_case for LLM compatibility.
// This allows the tool to accept both startTime and start_time formats.
func normalizeJSONFields(inputJSON string) string {
	// Parse into a generic map
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &raw); err != nil {
		return inputJSON // Return original if parsing fails
	}

	// Convert keys to snake_case
	normalized := make(map[string]interface{})
	for key, value := range raw {
		newKey := key
		if mapped, ok := fieldNameMappings[key]; ok {
			newKey = mapped
		}
		normalized[newKey] = value
	}

	// Marshal back to JSON
	result, err := json.Marshal(normalized)
	if err != nil {
		return inputJSON // Return original if marshaling fails
	}
	return string(result)
}

// ScheduleQueryTool searches for schedule events within a specific time range.
type ScheduleQueryTool struct {
	service      schedule.Service
	userIDGetter func(ctx context.Context) int32
}

// NewScheduleQueryTool creates a new schedule query tool.
func NewScheduleQueryTool(service schedule.Service, userIDGetter func(ctx context.Context) int32) *ScheduleQueryTool {
	return &ScheduleQueryTool{
		service:      service,
		userIDGetter: userIDGetter,
	}
}

// Name returns the tool name.
func (t *ScheduleQueryTool) Name() string {
	return "schedule_query"
}

// Description returns the tool description for the LLM.
func (t *ScheduleQueryTool) Description() string {
	return `[MUST USE FIRST] Query existing schedules BEFORE creating any new schedule.

IMPORTANT USAGE RULE:
- ALWAYS call this tool BEFORE schedule_add to check for conflicts
- Call this first when user asks about their schedule

Input: {"start_time": "ISO8601", "end_time": "ISO8601"}
Example: {"start_time": "2026-01-25T00:00:00+08:00", "end_time": "2026-01-26T00:00:00+08:00"}

Returns: List of existing schedules or "No schedules found"`
}

// InputType returns the expected input type schema.
func (t *ScheduleQueryTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "ISO8601 time string (e.g., 2026-01-01T09:00:00Z)",
			},
			"end_time": map[string]interface{}{
				"type":        "string",
				"description": "ISO8601 time string",
			},
		},
		"required": []string{"start_time", "end_time"},
	}
}

// Run executes the tool.
func (t *ScheduleQueryTool) Run(ctx context.Context, inputJSON string) (string, error) {
	// Normalize JSON field names (camelCase -> snake_case) for LLM compatibility
	normalizedJSON := normalizeJSONFields(inputJSON)

	// Parse input
	var input struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	if err := json.Unmarshal([]byte(normalizedJSON), &input); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate input
	if input.StartTime == "" {
		return "", fmt.Errorf("start_time is required")
	}
	if input.EndTime == "" {
		return "", fmt.Errorf("end_time is required")
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		return "", fmt.Errorf("invalid start_time format: %w. Please use ISO8601 format (e.g., 2026-01-21T09:00:00Z)", err)
	}

	endTime, err := time.Parse(time.RFC3339, input.EndTime)
	if err != nil {
		return "", fmt.Errorf("invalid end_time format: %w. Please use ISO8601 format (e.g., 2026-01-21T09:00:00Z)", err)
	}

	if endTime.Before(startTime) {
		return "", fmt.Errorf("end_time must be after start_time")
	}

	// Get user ID from context
	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return "", fmt.Errorf("unauthorized: no user ID in context")
	}

	// Query schedules
	instances, err := t.service.FindSchedules(ctx, userID, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("failed to query schedules: %w", err)
	}

	// Format results for LLM
	if len(instances) == 0 {
		return "No schedules found in the specified time range.", nil
	}

	// Build response with user-friendly formatting using strings.Builder for efficiency
	var result strings.Builder
	result.Grow(256) // Pre-allocate capacity
	result.WriteString(fmt.Sprintf("Found %d schedule(s):\n", len(instances)))

	for i, inst := range instances {
		startTimeFormatted := formatTime(inst.StartTs, inst.Timezone)
		var endTimeFormatted string
		if inst.EndTs != nil {
			endTimeFormatted = formatTime(*inst.EndTs, inst.Timezone)
		} else {
			endTimeFormatted = "no end time"
		}

		result.WriteString(fmt.Sprintf("%d. %s (%s - %s)", i+1, inst.Title, startTimeFormatted, endTimeFormatted))

		if inst.Location != "" {
			result.WriteString(fmt.Sprintf(" at %s", inst.Location))
		}

		if inst.IsRecurring {
			result.WriteString(" [recurring]")
		}

		result.WriteByte('\n')
	}

	return result.String(), nil
}

// Validate runs before execution to check input validity.
func (t *ScheduleQueryTool) Validate(ctx context.Context, inputJSON string) error {
	var input struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return err
	}

	if input.StartTime == "" || input.EndTime == "" {
		return fmt.Errorf("both start_time and end_time are required")
	}

	return nil
}

// ScheduleSummary represents a simplified schedule for query results.
type ScheduleSummary struct {
	UID      string `json:"uid"`
	Title    string `json:"title"`
	StartTs  int64  `json:"start_ts"`
	EndTs    int64  `json:"end_ts"`
	AllDay   bool   `json:"all_day"`
	Location string `json:"location,omitempty"`
	Status   string `json:"status"`
}

// ScheduleQueryToolResult represents the structured result of schedule query.
type ScheduleQueryToolResult struct {
	Schedules            []ScheduleSummary `json:"schedules"`
	Query                string            `json:"query"`
	Count                int               `json:"count"`
	TimeRangeDescription string            `json:"time_range_description"`
	QueryType            string            `json:"query_type"`
}

// RunWithStructuredResult executes the tool and returns a structured result.
// RunWithStructuredResult 执行工具并返回结构化结果。
func (t *ScheduleQueryTool) RunWithStructuredResult(ctx context.Context, inputJSON string) (*ScheduleQueryToolResult, error) {
	// Normalize JSON field names (camelCase -> snake_case) for LLM compatibility
	normalizedJSON := normalizeJSONFields(inputJSON)

	// Parse input
	var input struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	if err := json.Unmarshal([]byte(normalizedJSON), &input); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate input
	if input.StartTime == "" {
		return nil, fmt.Errorf("start_time is required")
	}
	if input.EndTime == "" {
		return nil, fmt.Errorf("end_time is required")
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format: %w. Please use ISO8601 format (e.g., 2026-01-21T09:00:00Z)", err)
	}

	endTime, err := time.Parse(time.RFC3339, input.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format: %w. Please use ISO8601 format (e.g., 2026-01-21T09:00:00Z)", err)
	}

	if endTime.Before(startTime) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	// Get user ID from context
	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("unauthorized: no user ID in context")
	}

	// Query schedules
	instances, err := t.service.FindSchedules(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}

	// Convert to ScheduleSummary
	schedules := make([]ScheduleSummary, 0, len(instances))
	for _, inst := range instances {
		var endTs int64
		if inst.EndTs != nil {
			endTs = *inst.EndTs
		}
		schedules = append(schedules, ScheduleSummary{
			UID:      inst.UID,
			Title:    inst.Title,
			StartTs:  inst.StartTs,
			EndTs:    endTs,
			AllDay:   inst.AllDay,
			Location: inst.Location,
			Status:   "ACTIVE", // Default status
		})
	}

	// Determine time range description
	timeRangeDescription := fmt.Sprintf("%s to %s",
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"))

	return &ScheduleQueryToolResult{
		Schedules:            schedules,
		Query:                fmt.Sprintf("%s - %s", input.StartTime, input.EndTime),
		Count:                len(schedules),
		TimeRangeDescription: timeRangeDescription,
		QueryType:            "range",
	}, nil
}

// ScheduleAddTool creates a new schedule event.
type ScheduleAddTool struct {
	service          schedule.Service
	userIDGetter     func(ctx context.Context) int32
	conflictResolver *schedule.ConflictResolver
}

// NewScheduleAddTool creates a new schedule add tool.
func NewScheduleAddTool(service schedule.Service, userIDGetter func(ctx context.Context) int32) *ScheduleAddTool {
	return &ScheduleAddTool{
		service:          service,
		userIDGetter:     userIDGetter,
		conflictResolver: schedule.NewConflictResolver(service),
	}
}

// Name returns the tool name.
func (t *ScheduleAddTool) Name() string {
	return "schedule_add"
}

// Description returns the tool description for the LLM.
func (t *ScheduleAddTool) Description() string {
	return `Create a NEW schedule AFTER checking for conflicts with schedule_query.

REQUIRED WORKFLOW:
1. First: Call schedule_query to check availability
2. Only if no conflict: Call schedule_add
3. If conflict exists: Call find_free_time instead

Input: {"title": "event name", "start_time": "ISO8601", "end_time": "ISO8601"}
Example: {"title": "Team Meeting", "start_time": "2026-01-25T15:00:00+08:00", "end_time": "2026-01-25T16:00:00+08:00"}

Note: end_time can be omitted for 1-hour default duration. Auto-adjusts conflicts if enabled.`
}

// InputType returns the expected input type schema.
func (t *ScheduleAddTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Event title",
			},
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "ISO8601 time string (e.g., 2026-01-21T09:00:00Z)",
			},
			"end_time": map[string]interface{}{
				"type":        "string",
				"description": "ISO8601 time string (optional for all-day events)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Event description (optional)",
			},
			"location": map[string]interface{}{
				"type":        "string",
				"description": "Event location (optional)",
			},
			"all_day": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether this is an all-day event (default: false)",
			},
		},
		"required": []string{"title", "start_time"},
	}
}

// Run executes the tool.
func (t *ScheduleAddTool) Run(ctx context.Context, inputJSON string) (string, error) {
	// Normalize JSON field names (camelCase -> snake_case) for LLM compatibility
	normalizedJSON := normalizeJSONFields(inputJSON)

	// Parse input
	var input struct {
		Title       string `json:"title"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time,omitempty"`
		Description string `json:"description,omitempty"`
		Location    string `json:"location,omitempty"`
		AllDay      bool   `json:"all_day,omitempty"`
	}

	if err := json.Unmarshal([]byte(normalizedJSON), &input); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate required fields
	if input.Title == "" {
		return "", fmt.Errorf("title is required")
	}
	if input.StartTime == "" {
		return "", fmt.Errorf("start_time is required")
	}

	// Parse start time
	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		return "", fmt.Errorf("invalid start_time format: %w. Please use ISO8601 format", err)
	}

	// Parse end time if provided, otherwise default to 1 hour
	var endTime *int64
	if input.EndTime != "" {
		end, err := time.Parse(time.RFC3339, input.EndTime)
		if err != nil {
			return "", fmt.Errorf("invalid end_time format: %w. Please use ISO8601 format", err)
		}
		endTs := end.Unix()
		endTime = &endTs
	} else {
		// Default duration: 1 hour (3600 seconds)
		defaultEndTs := startTime.Unix() + 3600
		endTime = &defaultEndTs
	}

	// Get user ID from context
	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return "", fmt.Errorf("unauthorized: no user ID in context")
	}

	// Create schedule request
	createReq := &schedule.CreateScheduleRequest{
		Title:       input.Title,
		Description: input.Description,
		Location:    input.Location,
		StartTs:     startTime.Unix(),
		EndTs:       endTime,
		AllDay:      input.AllDay,
		Timezone:    DefaultTimezone,
	}

	// Create schedule
	created, err := t.service.CreateSchedule(ctx, userID, createReq)

	// Auto-resolve conflicts: if creation fails due to conflict, find next available slot
	if errors.Is(err, schedule.ErrScheduleConflict) {
		slog.Info("schedule_add: conflict detected, finding available slot",
			"user_id", userID,
			"requested_start", startTime.Unix(),
		)

		// Calculate duration
		durationSec := int64(3600)
		if endTime != nil {
			durationSec = *endTime - startTime.Unix()
		}
		duration := time.Duration(durationSec) * time.Second

		// Use ConflictResolver to find best alternative
		resolution, resErr := t.conflictResolver.Resolve(ctx, userID, startTime, time.Time{}, duration)
		if resErr != nil {
			return "", fmt.Errorf("failed to create schedule: %w (and failed to find alternatives: %v)", err, resErr)
		}

		// If auto-resolved slot available, retry with that time
		if resolution.AutoResolved != nil {
			newStartTs := resolution.AutoResolved.Start.Unix()
			newEndTs := resolution.AutoResolved.End.Unix()
			createReq.StartTs = newStartTs
			createReq.EndTs = &newEndTs

			slog.Info("schedule_add: retrying with auto-resolved time",
				"user_id", userID,
				"original_start", startTime.Unix(),
				"new_start", newStartTs,
			)

			// Retry creation with new time
			created, err = t.service.CreateSchedule(ctx, userID, createReq)
			if err != nil {
				// Still failed, provide alternatives in error
				return t.formatConflictError(resolution, err)
			}
		} else {
			// No auto-resolved slot available, provide alternatives
			return t.formatConflictError(resolution, err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to create schedule: %w", err)
	}

	// Audit log for schedule creation
	slog.Info("schedule created",
		"user_id", userID,
		"schedule_id", created.ID,
		"title", sanitizeString(created.Title, maxTitleLengthForLog),
		"description", sanitizeString(created.Description, maxDescriptionLengthForLog),
		"start_ts", created.StartTs,
		"end_ts", created.EndTs,
		"has_end_time", created.EndTs != nil,
		"location", created.Location,
		"all_day", created.AllDay,
		"timezone", created.Timezone,
		"timestamp", time.Now().Unix(),
	)

	// Format response
	startTimeFormatted := formatTime(created.StartTs, created.Timezone)
	var endTimeFormatted string
	if created.EndTs != nil {
		endTimeFormatted = formatTime(*created.EndTs, created.Timezone)
	}

	// Check if time was auto-adjusted (original start time != actual start time)
	wasAdjusted := created.StartTs != startTime.Unix()

	result := fmt.Sprintf("✓ 已创建: %s (%s", created.Title, startTimeFormatted)
	if endTimeFormatted != "" {
		result += fmt.Sprintf(" - %s", endTimeFormatted)
	}
	result += ")"

	if wasAdjusted {
		result += " [时间冲突已自动调整]"
	}

	if created.Location != "" {
		result += fmt.Sprintf(" @ %s", created.Location)
	}

	return result, nil
}

// formatConflictError formats a conflict error with available alternative slots.
// Returns structured error for i18n support in the frontend.
func (t *ScheduleAddTool) formatConflictError(resolution *schedule.ConflictResolution, originalErr error) (string, error) {
	if len(resolution.Alternatives) == 0 {
		return "", fmt.Errorf("failed to create schedule: %w (no alternative slots available)", originalErr)
	}

	// Convert to structured alternatives (top 3)
	maxSlots := 3
	if len(resolution.Alternatives) < maxSlots {
		maxSlots = len(resolution.Alternatives)
	}

	alternatives := make([]schedule.TimeSlotAlternative, 0, maxSlots)
	for i := 0; i < maxSlots; i++ {
		alt := resolution.Alternatives[i]
		alternatives = append(alternatives, schedule.TimeSlotAlternative{
			StartTs: alt.Start.Unix(),
			EndTs:   alt.End.Unix(),
			Reason:  alt.Reason,
		})
	}

	// Wrap in structured error for frontend i18n
	conflictErr := schedule.NewConflictError(
		alternatives,
		len(resolution.Conflicts),
		resolution.OriginalStart.Unix(),
	)

	// Return with base error for compatibility
	return "", fmt.Errorf("%w: %s", schedule.ErrScheduleConflict, conflictErr.Error())
}

// Validate runs before execution to check input validity.
func (t *ScheduleAddTool) Validate(ctx context.Context, inputJSON string) error {
	var input struct {
		Title     string `json:"title"`
		StartTime string `json:"start_time"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return err
	}

	if input.Title == "" {
		return fmt.Errorf("title is required")
	}
	if input.StartTime == "" {
		return fmt.Errorf("start_time is required")
	}

	return nil
}

// Helper function to format timestamp for display in user's timezone.
// Uses cached timezone locations for better performance.
func formatTime(ts int64, timezone string) string {
	t := time.Unix(ts, 0)
	loc := getTimezoneLocation(timezone)
	return t.In(loc).Format("2006-01-02 15:04 MST")
}

// sanitizeString sanitizes sensitive data for audit logging.
// It limits the length and adds a suffix if truncated.
func sanitizeString(s string, maxLen int) string {
	if s == "" {
		return ""
	}
	runes := []rune(s) // Use runes to properly handle multi-byte characters
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "...[truncated]"
}

// FindFreeTimeTool finds available time slots for scheduling.
type FindFreeTimeTool struct {
	service      schedule.Service
	userIDGetter func(ctx context.Context) int32
	timezone     string
}

// NewFindFreeTimeTool creates a new find free time tool.
func NewFindFreeTimeTool(service schedule.Service, userIDGetter func(ctx context.Context) int32) *FindFreeTimeTool {
	return &FindFreeTimeTool{
		service:      service,
		userIDGetter: userIDGetter,
		timezone:     DefaultTimezone,
	}
}

// Name returns the tool name.
func (t *FindFreeTimeTool) Name() string {
	return "find_free_time"
}

// SetTimezone sets the user's timezone for date parsing.
func (t *FindFreeTimeTool) SetTimezone(timezone string) {
	if timezone != "" {
		t.timezone = timezone
	}
}

// Description returns the tool description for the LLM.
func (t *FindFreeTimeTool) Description() string {
	return `Find available 1-hour time slots when schedule_query detects conflicts.

WHEN TO USE:
- After schedule_query returns conflicting schedules
- User asks "when am I free" or similar

Input: {"date": "YYYY-MM-DD"}
Example: {"date": "2026-01-25"}

Returns: ISO8601 start time of available slot (you must add 1 hour for end time)
Error: Returns error message if no slots available (all 8AM-10PM occupied)

IMPORTANT: The returned time is the START only. End time = start + 1 hour.`
}

// InputType returns the expected input type schema.
func (t *FindFreeTimeTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"date": map[string]interface{}{
				"type":    "string",
				"format":  "date",
				"example": "2026-01-22",
			},
		},
		"required": []string{"date"},
	}
}

// Run executes the find free time tool.
func (t *FindFreeTimeTool) Run(ctx context.Context, inputJSON string) (string, error) {
	var input struct {
		Date string `json:"date"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if input.Date == "" {
		return "", fmt.Errorf("date is required")
	}

	userID := t.userIDGetter(ctx)

	// Load user's timezone for proper date parsing (cached)
	loc := getTimezoneLocation(t.timezone)

	// Parse the input date in user's timezone (e.g., "2026-01-22")
	date, err := time.ParseInLocation("2006-01-02", input.Date, loc)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	// Set time to start of day and end of day in user's timezone
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, loc)

	// Find schedules for the entire day
	schedules, err := t.service.FindSchedules(ctx, userID, startOfDay, endOfDay)
	if err != nil {
		return "", fmt.Errorf("failed to query schedules: %w", err)
	}

	// Define default duration: 1 hour (3600 seconds)
	const defaultDuration = int64(3600)

	// Find free slots (checking each hour from 8:00 to 22:00 inclusive)
	// hourStart=8 (8 AM), hourEnd=22 (10 PM)
	// We check hour <= hourEnd to include the 22:00-23:00 slot
	const hourStart = 8 // 8 AM
	const hourEnd = 22  // 10 PM (last slot starts at 22:00)

	// Check each hour slot
	for hour := hourStart; hour <= hourEnd; hour++ {
		// Create slot start and end times in user's timezone
		slotStart := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, loc)
		slotEnd := slotStart.Add(time.Duration(defaultDuration))

		// Skip if slot is beyond day end
		if slotEnd.After(endOfDay) {
			break
		}

		// Check for conflicts in this time slot
		hasConflict := false
		for _, existing := range schedules {
			// Calculate existing schedule's end time
			var existingEndTs int64
			if existing.EndTs != nil && *existing.EndTs > 0 {
				existingEndTs = *existing.EndTs
			} else {
				// No end time specified, assume 1 hour duration
				existingEndTs = existing.StartTs + defaultDuration
			}

			// Check overlap: (StartA < EndB) && (EndA > StartB)
			slotStartTs := slotStart.Unix()
			slotEndTs := slotEnd.Unix()

			if (slotStartTs < existingEndTs) && (slotEndTs > existing.StartTs) {
				hasConflict = true
				break
			}
		}

		if !hasConflict {
			// Found a free slot! Return in ISO8601 format
			return slotStart.Format(time.RFC3339), nil
		}
	}

	return "", fmt.Errorf("no available time slots on %s (all slots from 8 AM to 10 PM are occupied)", input.Date)
}

// ScheduleUpdateTool updates an existing schedule event.
type ScheduleUpdateTool struct {
	service      schedule.Service
	userIDGetter func(ctx context.Context) int32
}

// NewScheduleUpdateTool creates a new schedule update tool.
func NewScheduleUpdateTool(service schedule.Service, userIDGetter func(ctx context.Context) int32) *ScheduleUpdateTool {
	return &ScheduleUpdateTool{
		service:      service,
		userIDGetter: userIDGetter,
	}
}

// Name returns the tool name.
func (t *ScheduleUpdateTool) Name() string {
	return "schedule_update"
}

// Description returns the tool description for the LLM.
func (t *ScheduleUpdateTool) Description() string {
	return `Update an existing schedule event.
Can update schedule by ID or find matching schedule by date/title.
All times must be in ISO8601 format (e.g., "2026-01-21T09:00:00Z").
If duration is not specified, keeps original duration or defaults to 1 hour.`
}

// InputType returns the expected input type schema.
func (t *ScheduleUpdateTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "integer",
				"description": "Schedule ID to update (optional if date/title provided)",
			},
			"date": map[string]interface{}{
				"type":        "string",
				"description": "Date to find matching schedule (YYYY-MM-DD format, used if ID not provided)",
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "New event title (optional, keeps original if not specified)",
			},
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "New start time in ISO8601 format (optional)",
			},
			"end_time": map[string]interface{}{
				"type":        "string",
				"description": "New end time in ISO8601 format (optional)",
			},
			"location": map[string]interface{}{
				"type":        "string",
				"description": "Event location (optional)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Event description (optional)",
			},
		},
	}
}

// Run executes the tool.
func (t *ScheduleUpdateTool) Run(ctx context.Context, inputJSON string) (string, error) {
	// Normalize JSON field names (camelCase -> snake_case) for LLM compatibility
	normalizedJSON := normalizeJSONFields(inputJSON)

	// Parse input
	var input struct {
		ID          int32  `json:"id,omitempty"`
		Date        string `json:"date,omitempty"`
		Title       string `json:"title,omitempty"`
		StartTime   string `json:"start_time,omitempty"`
		EndTime     string `json:"end_time,omitempty"`
		Description string `json:"description,omitempty"`
		Location    string `json:"location,omitempty"`
	}

	if err := json.Unmarshal([]byte(normalizedJSON), &input); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Get user ID from context
	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return "", fmt.Errorf("unauthorized: no user ID in context")
	}

	// Determine which schedule to update
	var scheduleID int32
	var targetSchedule *schedule.ScheduleInstance

	if input.ID > 0 {
		// Direct update by ID
		scheduleID = input.ID
	} else if input.Date != "" {
		// Find schedule by date (using cached timezone)
		loc := getTimezoneLocation(DefaultTimezone)

		date, err := time.ParseInLocation("2006-01-02", input.Date, loc)
		if err != nil {
			return "", fmt.Errorf("invalid date format: %w", err)
		}

		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
		endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, loc)

		schedules, err := t.service.FindSchedules(ctx, userID, startOfDay, endOfDay)
		if err != nil {
			return "", fmt.Errorf("failed to find schedules: %w", err)
		}

		if len(schedules) == 0 {
			return "", fmt.Errorf("no schedule found on %s", input.Date)
		}

		if len(schedules) > 1 {
			// Multiple schedules found - return list for user to choose
			var result strings.Builder
			result.WriteString(fmt.Sprintf("Found %d schedules on %s. Please specify ID to update:\n", len(schedules), input.Date))
			for _, s := range schedules {
				result.WriteString(fmt.Sprintf("- ID %d: %s (%s)\n", s.ID, s.Title, formatTime(s.StartTs, s.Timezone)))
			}
			return result.String(), nil
		}

		// Found exactly one schedule
		targetSchedule = schedules[0]
		scheduleID = targetSchedule.ID
	} else {
		return "", fmt.Errorf("either 'id' or 'date' must be provided to identify the schedule")
	}

	// Build update request
	updateReq := &schedule.UpdateScheduleRequest{}

	// Set fields to update (only update provided fields)
	if input.Title != "" {
		updateReq.Title = &input.Title
	}
	if input.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, input.StartTime)
		if err != nil {
			return "", fmt.Errorf("invalid start_time format: %w", err)
		}
		startTs := startTime.Unix()
		updateReq.StartTs = &startTs

		// Handle end time
		if input.EndTime != "" {
			endTime, err := time.Parse(time.RFC3339, input.EndTime)
			if err != nil {
				return "", fmt.Errorf("invalid end_time format: %w", err)
			}
			endTs := endTime.Unix()
			updateReq.EndTs = &endTs
		} else if targetSchedule != nil && targetSchedule.EndTs != nil {
			// Keep original duration
			originalDuration := *targetSchedule.EndTs - targetSchedule.StartTs
			newEndTs := startTs + originalDuration
			updateReq.EndTs = &newEndTs
		} else {
			// Default to 1 hour
			defaultEndTs := startTs + 3600
			updateReq.EndTs = &defaultEndTs
		}
	}
	if input.Description != "" {
		updateReq.Description = &input.Description
	}
	if input.Location != "" {
		updateReq.Location = &input.Location
	}

	// Update schedule
	updated, err := t.service.UpdateSchedule(ctx, userID, scheduleID, updateReq)
	if err != nil {
		return "", fmt.Errorf("failed to update schedule: %w", err)
	}

	// Build changed fields list for audit tracking
	var changedFields []string
	if updateReq.Title != nil {
		changedFields = append(changedFields, "title")
	}
	if updateReq.Description != nil {
		changedFields = append(changedFields, "description")
	}
	if updateReq.StartTs != nil {
		changedFields = append(changedFields, "start_ts")
	}
	if updateReq.EndTs != nil {
		changedFields = append(changedFields, "end_ts")
	}
	if updateReq.Location != nil {
		changedFields = append(changedFields, "location")
	}
	if updateReq.AllDay != nil {
		changedFields = append(changedFields, "all_day")
	}

	// Audit log for schedule update with change tracking
	slog.Info("schedule updated",
		"user_id", userID,
		"schedule_id", updated.ID,
		"title", sanitizeString(updated.Title, maxTitleLengthForLog),
		"description", sanitizeString(updated.Description, maxDescriptionLengthForLog),
		"start_ts", updated.StartTs,
		"end_ts", updated.EndTs,
		"has_end_time", updated.EndTs != nil,
		"location", updated.Location,
		"all_day", updated.AllDay,
		"timezone", updated.Timezone,
		"changed_fields", strings.Join(changedFields, ","),
		"timestamp", time.Now().Unix(),
	)

	// Format response
	startTimeFormatted := formatTime(updated.StartTs, updated.Timezone)
	var endTimeFormatted string
	if updated.EndTs != nil {
		endTimeFormatted = formatTime(*updated.EndTs, updated.Timezone)
	}

	result := fmt.Sprintf("Successfully updated schedule (ID %d): %s (%s", updated.ID, updated.Title, startTimeFormatted)
	if endTimeFormatted != "" {
		result += fmt.Sprintf(" - %s", endTimeFormatted)
	}
	result += ")"

	if updated.Location != "" {
		result += fmt.Sprintf(" at %s", updated.Location)
	}

	return result, nil
}
