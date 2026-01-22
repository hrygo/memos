// Package suggestion provides intelligent schedule time suggestions.
package suggestion

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/usememos/memos/store"
)

// Analyzer analyzes schedule patterns and provides time suggestions.
type Analyzer struct {
	store *store.Store
}

// NewAnalyzer creates a new schedule analyzer.
func NewAnalyzer(st *store.Store) *Analyzer {
	return &Analyzer{
		store: st,
	}
}

// Suggestion represents a time slot suggestion with confidence.
type Suggestion struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Confidence  float32   `json:"confidence"`
	Reason      string    `json:"reason"`
	IsAvailable bool      `json:"is_available"`
	ConflictCount int    `json:"conflict_count"`
}

// SuggestionOptions holds options for generating suggestions.
type SuggestionOptions struct {
	UserID        int32
	StartDate     time.Time
	EndDate       time.Time
	Duration      time.Duration // Desired duration
	PreferredTime []string      // Preferred time slots (e.g., "morning", "afternoon")
	ExcludeWeekend bool
}

// GenerateSuggestions generates time slot suggestions based on:
// 1. Historical patterns (when user usually schedules)
// 2. Current schedule load
// 3. Preferred time windows
func (a *Analyzer) GenerateSuggestions(ctx context.Context, opts *SuggestionOptions) ([]*Suggestion, error) {
	// Default to 7 days if no range specified
	if opts.StartDate.IsZero() {
		opts.StartDate = time.Now().Truncate(24 * time.Hour)
	}
	if opts.EndDate.IsZero() {
		opts.EndDate = opts.StartDate.AddDate(0, 0, 7)
	}
	if opts.Duration == 0 {
		opts.Duration = time.Hour // Default 1 hour
	}

	// Fetch user's schedules in the range
	schedules, err := a.store.ListSchedules(ctx, &store.FindSchedule{
		CreatorID: &opts.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// Filter schedules by date range
	filtered := a.filterSchedulesByRange(schedules, opts.StartDate, opts.EndDate)

	// Analyze patterns
	patterns := a.analyzePatterns(filtered)

	// Generate suggestions
	suggestions := a.generateTimeSlots(opts, filtered, patterns)

	// Score and rank suggestions
	a.scoreSuggestions(suggestions, patterns)

	// Sort by confidence
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	// Return top 10 suggestions
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return suggestions, nil
}

// SchedulePattern represents learned scheduling patterns.
type SchedulePattern struct {
	HourOfDay      map[int]int    // Hour -> count
	DayOfWeek      map[time.Weekday]int
	AverageDuration time.Duration
	PreferredDays  []time.Weekday
	BusyHours      []string
	LastScheduled  time.Time
}

func (a *Analyzer) analyzePatterns(schedules []*store.Schedule) *SchedulePattern {
	pattern := &SchedulePattern{
		HourOfDay: make(map[int]int),
		DayOfWeek: make(map[time.Weekday]int),
	}

	if len(schedules) == 0 {
		return pattern
	}

	var totalDuration time.Duration
	for _, s := range schedules {
		start := time.Unix(s.StartTs, 0)
		hour := start.Hour()
		pattern.HourOfDay[hour]++
		pattern.DayOfWeek[start.Weekday()]++

		if s.EndTs != nil && *s.EndTs > 0 {
			duration := time.Duration(*s.EndTs-s.StartTs) * time.Second
			totalDuration += duration
		}

		if start.After(pattern.LastScheduled) {
			pattern.LastScheduled = start
		}
	}

	if len(schedules) > 0 {
		pattern.AverageDuration = totalDuration / time.Duration(len(schedules))
	}

	// Find preferred days
	for day, count := range pattern.DayOfWeek {
		if count >= 2 { // At least 2 schedules
			pattern.PreferredDays = append(pattern.PreferredDays, day)
		}
	}

	// Find busy hours (3+ schedules)
	for hour, count := range pattern.HourOfDay {
		if count >= 3 {
			pattern.BusyHours = append(pattern.BusyHours,
				fmt.Sprintf("%02d:00", hour))
		}
	}

	return pattern
}

func (a *Analyzer) filterSchedulesByRange(schedules []*store.Schedule, start, end time.Time) []*store.Schedule {
	filtered := make([]*store.Schedule, 0)
	for _, s := range schedules {
		scheduleTime := time.Unix(s.StartTs, 0)
		if (scheduleTime.After(start) || scheduleTime.Equal(start)) &&
			scheduleTime.Before(end) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (a *Analyzer) generateTimeSlots(opts *SuggestionOptions, schedules []*store.Schedule, patterns *SchedulePattern) []*Suggestion {
	suggestions := make([]*Suggestion, 0)

	// Generate hourly slots from 8 AM to 8 PM
	for day := opts.StartDate; day.Before(opts.EndDate); day = day.AddDate(0, 0, 1) {
		// Skip weekends if requested
		if opts.ExcludeWeekend && (day.Weekday() == time.Saturday || day.Weekday() == time.Sunday) {
			continue
		}

		// Check if this day is a preferred day
		isPreferredDay := false
		for _, pd := range patterns.PreferredDays {
			if day.Weekday() == pd {
				isPreferredDay = true
				break
			}
		}

		// Generate slots for this day
		for hour := 8; hour <= 20; hour++ {
			slotStart := time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, time.Local)
			slotEnd := slotStart.Add(opts.Duration)

			// Check for conflicts
			conflicts := a.countConflicts(schedules, slotStart, slotEnd)

			// Determine if preferred time
			isPreferredTime := a.isPreferredTime(hour, opts.PreferredTime)

			suggestion := &Suggestion{
				StartTime:      slotStart,
				EndTime:        slotEnd,
				IsAvailable:    conflicts == 0,
				ConflictCount:  conflicts,
			}

			// Generate reason
			if conflicts == 0 {
				suggestion.Reason = "No conflicts"
			} else {
				suggestion.Reason = fmt.Sprintf("%d conflict(s)", conflicts)
			}

			if isPreferredDay {
				suggestion.Reason += ", preferred day"
			}
			if isPreferredTime {
				suggestion.Reason += ", preferred time"
			}

			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

func (a *Analyzer) countConflicts(schedules []*store.Schedule, start, end time.Time) int {
	count := 0
	for _, s := range schedules {
		scheduleStart := time.Unix(s.StartTs, 0)
		var scheduleEnd time.Time
		if s.EndTs != nil && *s.EndTs > 0 {
			scheduleEnd = time.Unix(*s.EndTs, 0)
		} else {
			scheduleEnd = scheduleStart.Add(time.Hour) // Assume 1 hour
		}

		// Check for overlap
		if (start.Before(scheduleEnd) || start.Equal(scheduleEnd)) &&
			(end.After(scheduleStart) || end.Equal(scheduleStart)) {
			count++
		}
	}
	return count
}

func (a *Analyzer) isPreferredTime(hour int, preferred []string) bool {
	if len(preferred) == 0 {
		return false
	}

	for _, p := range preferred {
		switch p {
		case "morning":
			if hour >= 6 && hour < 12 {
				return true
			}
		case "afternoon":
			if hour >= 12 && hour < 18 {
				return true
			}
		case "evening":
			if hour >= 18 && hour < 22 {
				return true
			}
		default:
			// Try to parse as hour
			var h int
			fmt.Sscanf(p, "%d", &h)
			if h == hour {
				return true
			}
		}
	}
	return false
}

func (a *Analyzer) scoreSuggestions(suggestions []*Suggestion, patterns *SchedulePattern) {
	// Find max conflicts for normalization
	maxConflicts := 0
	for _, s := range suggestions {
		if s.ConflictCount > maxConflicts {
			maxConflicts = s.ConflictCount
		}
	}

	for _, s := range suggestions {
		score := float32(0.5) // Base score

		// Prefer available slots
		if s.IsAvailable {
			score += 0.3
		}

		// Penalize conflicts
		if maxConflicts > 0 {
			score -= float32(s.ConflictCount) / float32(maxConflicts) * 0.3
		}

		// Bonus for preferred times (morning/afternoon)
		hour := s.StartTime.Hour()
		if hour >= 9 && hour <= 11 {
			score += 0.1 // Morning bonus
		} else if hour >= 14 && hour <= 16 {
			score += 0.1 // Afternoon bonus
		}

		// Bonus for weekdays
		if s.StartTime.Weekday() >= time.Monday && s.StartTime.Weekday() <= time.Friday {
			score += 0.05
		}

		// Apply historical pattern bonus
		if patterns.HourOfDay[hour] >= 2 {
			score += 0.1
		}

		s.Confidence = float32(math.Min(1.0, math.Max(0.0, float64(score))))
	}
}

// GetOptimalTime returns the best time slot for a given duration within a range.
func (a *Analyzer) GetOptimalTime(ctx context.Context, userID int32, start, end time.Time, duration time.Duration) (*Suggestion, error) {
	suggestions, err := a.GenerateSuggestions(ctx, &SuggestionOptions{
		UserID:    userID,
		StartDate: start,
		EndDate:   end,
		Duration:  duration,
	})
	if err != nil {
		return nil, err
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("no suggestions available")
	}

	// Return the highest confidence suggestion that's available
	for _, s := range suggestions {
		if s.IsAvailable {
			return s, nil
		}
	}

	// If no available slot, return the one with least conflicts
	return suggestions[0], nil
}
