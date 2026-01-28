// Package timezone provides timezone utilities for the DivineSense application.
//
// This package handles timezone conversions, parsing, and formatting
// to ensure consistent time handling across the application.
package timezone

import (
	"fmt"
	"time"
)

// Default location constants
var (
	// UTC is the coordinated universal time timezone
	UTC = time.UTC

	// Local is the local timezone
	Local = time.Local
)

// ParseTimezone parses an IANA timezone identifier (e.g., "Asia/Shanghai").
// If the timezone is invalid, returns UTC and an error.
func ParseTimezone(tz string) (*time.Location, error) {
	if tz == "" || tz == "UTC" {
		return UTC, nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return UTC, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}

	return loc, nil
}

// MustParseTimezone parses a timezone or panics if invalid.
// Use this for constants that are known to be valid at compile time.
func MustParseTimezone(tz string) *time.Location {
	loc, err := ParseTimezone(tz)
	if err != nil {
		panic(err)
	}
	return loc
}

// GetDefaultTimezone returns the default timezone (UTC).
func GetDefaultTimezone() *time.Location {
	return UTC
}

// IsValidTimezone checks if a timezone identifier is valid.
func IsValidTimezone(tz string) bool {
	if tz == "" || tz == "UTC" {
		return true
	}

	_, err := time.LoadLocation(tz)
	return err == nil
}

// ToUserTimezone converts a Unix timestamp to the user's timezone.
func ToUserTimezone(ts int64, tz *time.Location) time.Time {
	if tz == nil {
		tz = UTC
	}
	return time.Unix(ts, 0).In(tz)
}

// ToUTCTimestamp converts a time in the user's timezone to a Unix timestamp (UTC).
func ToUTCTimestamp(t time.Time) int64 {
	return t.Unix()
}

// FormatTimeWithTimezone formats a Unix timestamp as a string in the given timezone.
// The format should be a valid Go time format string (e.g., "2006-01-02 15:04").
func FormatTimeWithTimezone(ts int64, tz *time.Location, format string) string {
	if tz == nil {
		tz = UTC
	}
	return time.Unix(ts, 0).In(tz).Format(format)
}

// FormatScheduleTime formats a schedule's time for display.
// Rules:
//   - All-day event: "2006-01-02"
//   - With end time: "2006-01-02 15:04 - 16:00"
//   - No end time: "2006-01-02 15:00"
func FormatScheduleTime(startTs int64, endTs *int64, allDay bool, tz *time.Location) string {
	if tz == nil {
		tz = UTC
	}
	startTime := time.Unix(startTs, 0).In(tz)

	if allDay {
		return startTime.Format("2006-01-02")
	}

	if endTs != nil {
		endTime := time.Unix(*endTs, 0).In(tz)
		return fmt.Sprintf("%s - %s",
			startTime.Format("2006-01-02 15:04"),
			endTime.Format("15:04"))
	}

	return startTime.Format("2006-01-02 15:04")
}

// FormatScheduleForContext formats a schedule for LLM context.
// Format: "1. 2026-01-21 14:00 - 16:00 - Team Meeting @ Room A"
func FormatScheduleForContext(startTs int64, endTs *int64, title string, location string, allDay bool, index int, tz *time.Location) string {
	timeStr := FormatScheduleTime(startTs, endTs, allDay, tz)
	result := fmt.Sprintf("%d. %s - %s", index+1, timeStr, title)

	if location != "" {
		result += fmt.Sprintf(" @ %s", location)
	}

	return result
}

// StartOfDay returns the start of the day (00:00:00) in the given timezone.
func StartOfDay(t time.Time, tz *time.Location) time.Time {
	if tz == nil {
		tz = UTC
	}
	return time.Date(t.In(tz).Year(), t.In(tz).Month(), t.In(tz).Day(), 0, 0, 0, 0, tz)
}

// EndOfDay returns the end of the day (23:59:59.999999999) in the given timezone.
func EndOfDay(t time.Time, tz *time.Location) time.Time {
	if tz == nil {
		tz = UTC
	}
	return time.Date(t.In(tz).Year(), t.In(tz).Month(), t.In(tz).Day(), 23, 59, 59, 999999999, tz)
}

// NowInTimezone returns the current time in the given timezone.
func NowInTimezone(tz *time.Location) time.Time {
	if tz == nil {
		tz = UTC
	}
	return time.Now().In(tz)
}

// Common timezone constants
const (
	// TimezoneUTC is the UTC timezone identifier
	TimezoneUTC = "UTC"

	// TimezoneAsiaShanghai is the China Standard Time timezone
	TimezoneAsiaShanghai = "Asia/Shanghai"

	// TimezoneAmericaNewYork is the Eastern Time timezone
	TimezoneAmericaNewYork = "America/New_York"

	// TimezoneAmericaLosAngeles is the Pacific Time timezone
	TimezoneAmericaLosAngeles = "America/Los_Angeles"

	// TimezoneEuropeLondon is the GMT/BST timezone
	TimezoneEuropeLondon = "Europe/London"

	// TimezoneEuropeParis is the CET/CEST timezone
	TimezoneEuropeParis = "Europe/Paris"

	// TimezoneAsiaTokyo is the Japan Standard Time timezone
	TimezoneAsiaTokyo = "Asia/Tokyo"

	// TimezoneAustraliaSydney is the AEST/AEDT timezone
	TimezoneAustraliaSydney = "Australia/Sydney"
)

// Common timezone locations (pre-loaded for performance)
var (
	// LocationAsiaShanghai is the pre-loaded Asia/Shanghai location
	LocationAsiaShanghai = MustParseTimezone(TimezoneAsiaShanghai)

	// LocationAmericaNewYork is the pre-loaded America/New_York location
	LocationAmericaNewYork = MustParseTimezone(TimezoneAmericaNewYork)

	// LocationAmericaLosAngeles is the pre-loaded America/Los_Angeles location
	LocationAmericaLosAngeles = MustParseTimezone(TimezoneAmericaLosAngeles)

	// LocationEuropeLondon is the pre-loaded Europe/London location
	LocationEuropeLondon = MustParseTimezone(TimezoneEuropeLondon)

	// LocationEuropeParis is the pre-loaded Europe/Paris location
	LocationEuropeParis = MustParseTimezone(TimezoneEuropeParis)

	// LocationAsiaTokyo is the pre-loaded Asia/Tokyo location
	LocationAsiaTokyo = MustParseTimezone(TimezoneAsiaTokyo)

	// LocationAustraliaSydney is the pre-loaded Australia/Sydney location
	LocationAustraliaSydney = MustParseTimezone(TimezoneAustraliaSydney)
)
