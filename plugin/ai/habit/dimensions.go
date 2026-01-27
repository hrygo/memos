// Package habit provides user habit learning and analysis for AI agents.
package habit

import (
	"time"
)

// TimeHabits represents learned time-related habits.
type TimeHabits struct {
	// ActiveHours are the most active hours of the day (0-23)
	ActiveHours []int `json:"active_hours"`
	// PreferredTimes are specific times the user prefers (e.g., "09:00", "14:00")
	PreferredTimes []string `json:"preferred_times"`
	// ReminderLeadMin is the preferred reminder lead time in minutes
	ReminderLeadMin int `json:"reminder_lead_min"`
	// WeekdayPattern indicates if user is primarily active on weekdays
	WeekdayPattern bool `json:"weekday_pattern"`
}

// ScheduleHabits represents learned schedule-related habits.
type ScheduleHabits struct {
	// DefaultDuration is the typical meeting/event duration in minutes
	DefaultDuration int `json:"default_duration"`
	// PreferredSlots are preferred time slots (e.g., "morning", "afternoon")
	PreferredSlots []string `json:"preferred_slots"`
	// FrequentLocations are commonly used locations
	FrequentLocations []string `json:"frequent_locations"`
	// TitlePatterns are common title patterns/keywords
	TitlePatterns []string `json:"title_patterns"`
}

// SearchHabits represents learned search-related habits.
type SearchHabits struct {
	// FrequentKeywords are commonly used search keywords
	FrequentKeywords []string `json:"frequent_keywords"`
	// SearchMode is the preferred search mode ("exact" or "fuzzy")
	SearchMode string `json:"search_mode"`
	// ResultPreference indicates preferred result type
	ResultPreference string `json:"result_preference"`
}

// UserHabits aggregates all learned habits for a user.
type UserHabits struct {
	UserID    int32           `json:"user_id"`
	Time      *TimeHabits     `json:"time"`
	Schedule  *ScheduleHabits `json:"schedule"`
	Search    *SearchHabits   `json:"search"`
	UpdatedAt time.Time       `json:"updated_at"`
	Version   int             `json:"version"` // For optimistic locking
}

// DefaultTimeHabits returns sensible defaults for time habits.
func DefaultTimeHabits() *TimeHabits {
	return &TimeHabits{
		ActiveHours:     []int{9, 10, 14, 15, 16},
		PreferredTimes:  []string{"09:00", "14:00"},
		ReminderLeadMin: 15,
		WeekdayPattern:  true,
	}
}

// DefaultScheduleHabits returns sensible defaults for schedule habits.
func DefaultScheduleHabits() *ScheduleHabits {
	return &ScheduleHabits{
		DefaultDuration:   60,
		PreferredSlots:    []string{"morning", "afternoon"},
		FrequentLocations: []string{},
		TitlePatterns:     []string{},
	}
}

// DefaultSearchHabits returns sensible defaults for search habits.
func DefaultSearchHabits() *SearchHabits {
	return &SearchHabits{
		FrequentKeywords: []string{},
		SearchMode:       "fuzzy",
		ResultPreference: "",
	}
}

// DefaultUserHabits returns a UserHabits with all default values.
func DefaultUserHabits(userID int32) *UserHabits {
	return &UserHabits{
		UserID:    userID,
		Time:      DefaultTimeHabits(),
		Schedule:  DefaultScheduleHabits(),
		Search:    DefaultSearchHabits(),
		UpdatedAt: time.Now(),
		Version:   0,
	}
}

// AnalysisConfig holds configuration for habit analysis.
type AnalysisConfig struct {
	// LookbackDays is how many days of history to analyze
	LookbackDays int `json:"lookback_days"`
	// MinSamples is the minimum number of samples required for analysis
	MinSamples int `json:"min_samples"`
	// PeakMultiplier is the threshold for identifying peak hours (vs average)
	PeakMultiplier float64 `json:"peak_multiplier"`
	// MinKeywordFrequency is the minimum frequency for a keyword to be considered
	MinKeywordFrequency int `json:"min_keyword_frequency"`
}

// DefaultAnalysisConfig returns the default analysis configuration.
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		LookbackDays:        30,
		MinSamples:          10,
		PeakMultiplier:      1.5,
		MinKeywordFrequency: 3,
	}
}
