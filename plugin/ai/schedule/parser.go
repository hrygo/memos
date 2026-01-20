package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
)

const (
	// Validation constants
	MaxInputLength = 500 // characters
)

// Parser handles natural language parsing for schedules.
type Parser struct {
	llmService ai.LLMService
	location   *time.Location
}

// NewParser creates a new schedule parser.
func NewParser(llmService ai.LLMService, timezone string) (*Parser, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Default to Asia/Shanghai if timezone is invalid
		loc, _ = time.LoadLocation("Asia/Shanghai")
	}

	return &Parser{
		llmService: llmService,
		location:   loc,
	}, nil
}

// ParseResult represents the parsed schedule information.
type ParseResult struct {
	Title       string
	Description string
	Location    string
	StartTs     int64
	EndTs       int64
	AllDay      bool
	Timezone    string
	Reminders   []*v1pb.Reminder
	Recurrence  *RecurrenceRule
}

// Parse parses natural language text and returns schedule information.
func (p *Parser) Parse(ctx context.Context, text string) (*ParseResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Validate input length
	if len(text) > MaxInputLength {
		return nil, fmt.Errorf("input too long: maximum %d characters, got %d", MaxInputLength, len(text))
	}

	// Use LLM parsing directly
	return p.parseWithLLM(ctx, text)
}

// llmScheduleResponse is the intermediate JSON structure for LLM output
type llmScheduleResponse struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Location    string           `json:"location"`
	StartTime   string           `json:"start_time"` // Format: YYYY-MM-DD HH:mm:ss
	EndTime     string           `json:"end_time"`   // Format: YYYY-MM-DD HH:mm:ss
	AllDay      bool             `json:"all_day"`
	Reminders   []*v1pb.Reminder `json:"reminders"`
	Recurrence  *RecurrenceRule  `json:"recurrence"`
}

// parseWithLLM uses LLM to parse complex natural language.
func (p *Parser) parseWithLLM(ctx context.Context, text string) (*ParseResult, error) {
	now := time.Now().In(p.location)
	nowUTC := now.UTC()

	systemPrompt := fmt.Sprintf(`You are an intelligent schedule parser. Your goal is to extract schedule details from user input into a strict JSON format.

Current Time (UTC): %s
User Timezone: %s

IMPORTANT RULES:
1. Always return start_time and end_time in UTC timezone
2. Format: YYYY-MM-DD HH:mm:ss (no timezone suffix)
3. Calculate times in UTC, accounting for the user's timezone

Output Schema (JSON Only):
{
  "title": "Clean title without time/date keywords",
  "description": "Details, or empty string",
  "location": "Location name, or empty string",
  "start_time": "YYYY-MM-DD HH:mm:ss UTC",
  "end_time": "YYYY-MM-DD HH:mm:ss UTC",
  "all_day": boolean,
  "reminders": [{"type": "before", "value": int, "unit": "minutes|hours|days"}],
  "recurrence": {"type": "daily|weekly|monthly", "interval": int, "weekdays": [int], "month_day": int} or null
}

Rules:
1. Calculate absolute 'start_time' and 'end_time' relative to Current Time (in UTC).
2. If duration is not specified, default to 1 hour (end_time = start_time + 1h).
3. If only date is mentioned (no specific time), set 'all_day': true, and use 00:00:00 for times (in UTC).
4. Extract reminders if mentioned (e.g., "10 mins before").
5. Remove time, date, and location words from the 'title'.
6. Extract recurrence patterns:
   - "每天"/"daily" → {"type": "daily", "interval": 1}
   - "每周"/"weekly" → {"type": "weekly", "interval": 1}
   - "每周一"/"每周三" → {"type": "weekly", "weekdays": [1]/[3]}
   - "每月15号" → {"type": "monthly", "month_day": 15}
   - Weekdays: Monday=1, Tuesday=2, ..., Sunday=7
`, nowUTC.Format("2006-01-02 15:04:05"), p.location.String())

	userPrompt := fmt.Sprintf("User Input: %s", text)

	response, err := p.llmService.Chat(ctx, []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM parsing failed: %w", err)
	}

	// Clean code blocks if present
	jsonStr := strings.TrimSpace(response)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")

	// Parse JSON response
	var llmResp llmScheduleResponse
	if err := json.Unmarshal([]byte(jsonStr), &llmResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w. Response: %s", err, response)
	}

	// Convert string times to int64
	var startTs, endTs int64

	// Helper to parse time as UTC
	parseTime := func(timeStr string) (int64, error) {
		// Remove possible UTC suffix
		timeStr = strings.TrimSuffix(timeStr, " UTC")
		timeStr = strings.TrimSpace(timeStr)

		// Parse as UTC
		t, err := time.Parse("2006-01-02 15:04:05", timeStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse time %q: %w", timeStr, err)
		}
		return t.Unix(), nil
	}

	if llmResp.StartTime != "" {
		if ts, err := parseTime(llmResp.StartTime); err == nil {
			startTs = ts
		}
	}
	if llmResp.EndTime != "" {
		if ts, err := parseTime(llmResp.EndTime); err == nil {
			endTs = ts
		}
	}

	// Validate time is not too far in the past (more than 24 hours)
	if startTs < nowUTC.Add(-24*time.Hour).Unix() {
		return nil, fmt.Errorf("parsed start time is too far in the past: %d", startTs)
	}

	// Validate end time is not before start time
	if endTs > 0 && endTs < startTs {
		return nil, fmt.Errorf("end time %d is before start time %d", endTs, startTs)
	}

	// Fallback if parsing failed (should rely on LLM correctness though)
	if startTs == 0 {
		startTs = nowUTC.Add(time.Hour).Unix()
	}
	if endTs == 0 || endTs < startTs {
		endTs = startTs + 3600
	}

	return &ParseResult{
		Title:       llmResp.Title,
		Description: llmResp.Description,
		Location:    llmResp.Location,
		StartTs:     startTs,
		EndTs:       endTs,
		AllDay:      llmResp.AllDay,
		Timezone:    p.location.String(),
		Reminders:   llmResp.Reminders,
		Recurrence:  llmResp.Recurrence,
	}, nil

}

// ToSchedule converts ParseResult to v1pb.Schedule.
func (r *ParseResult) ToSchedule() *v1pb.Schedule {
	schedule := &v1pb.Schedule{
		Title:       r.Title,
		Description: r.Description,
		Location:    r.Location,
		StartTs:     r.StartTs,
		EndTs:       r.EndTs,
		AllDay:      r.AllDay,
		Timezone:    r.Timezone,
		Reminders:   r.Reminders,
		State:       "NORMAL",
	}

	// Convert recurrence rule to JSON string
	if r.Recurrence != nil {
		recurrenceJSON, err := r.Recurrence.ToJSON()
		if err != nil {
			// Failed to serialize recurrence rule - create schedule without it
			// This shouldn't happen in practice as RecurrenceRule is simple JSON
			// User can manually edit the schedule later to add recurrence
		} else {
			schedule.RecurrenceRule = recurrenceJSON
		}
	}

	return schedule
}
