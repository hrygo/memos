package ai

import "time"

const (
	// DefaultTimezone is the default timezone used when no user timezone is specified.
	// Uses Asia/Shanghai as it's a commonly used timezone for Chinese users.
	DefaultTimezone = "Asia/Shanghai"

	// DefaultTimezoneLocation is the cached time.Location for the default timezone.
	// This avoids repeated calls to time.LoadLocation.
	// It is initialized in package init.
)

var defaultTimezoneLocation *time.Location

func init() {
	var err error
	defaultTimezoneLocation, err = time.LoadLocation(DefaultTimezone)
	if err != nil {
		// Fallback to UTC if timezone loading fails
		defaultTimezoneLocation = time.UTC
	}
}

// GetDefaultTimezone returns the default timezone string.
func GetDefaultTimezone() string {
	return DefaultTimezone
}

// GetDefaultTimezoneLocation returns the cached time.Location for the default timezone.
func GetDefaultTimezoneLocation() *time.Location {
	return defaultTimezoneLocation
}

// IsValidTimezone checks if a timezone string is valid by attempting to load it.
func IsValidTimezone(tz string) bool {
	if tz == "" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

// NormalizeTimezone returns a valid timezone string.
// If the input is empty or invalid, returns the default timezone.
func NormalizeTimezone(tz string) string {
	if tz == "" || !IsValidTimezone(tz) {
		return DefaultTimezone
	}
	return tz
}
