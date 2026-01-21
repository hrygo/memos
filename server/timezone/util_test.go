package timezone

import (
	"testing"
	"time"
)

func TestParseTimezone(t *testing.T) {
	tests := []struct {
		name    string
		tz      string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "UTC",
			tz:      "UTC",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "empty string defaults to UTC",
			tz:      "",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "Asia/Shanghai",
			tz:      "Asia/Shanghai",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "America/New_York",
			tz:      "America/New_York",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "invalid timezone",
			tz:      "Invalid/Timezone",
			wantNil: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := ParseTimezone(tt.tz)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimezone() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (loc == nil) != tt.wantNil {
				t.Errorf("ParseTimezone() location = %v, wantNil %v", loc, tt.wantNil)
			}
		})
	}
}

func TestIsValidTimezone(t *testing.T) {
	tests := []struct {
		name string
		tz   string
		want bool
	}{
		{"UTC", "UTC", true},
		{"empty", "", true},
		{"Asia/Shanghai", "Asia/Shanghai", true},
		{"America/New_York", "America/New_York", true},
		{"invalid", "Invalid/Timezone", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTimezone(tt.tz); got != tt.want {
				t.Errorf("IsValidTimezone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToUserTimezone(t *testing.T) {
	// 2025-01-21 00:00:00 UTC
	ts := int64(1737417600)

	tests := []struct {
		name      string
		ts        int64
		timezone  string
		wantHour  int
		wantDay   int
	}{
		{
			name:     "UTC timezone",
			ts:       ts,
			timezone: "UTC",
			wantHour: 0,
			wantDay:  21,
		},
		{
			name:     "Asia/Shanghai (UTC+8)",
			ts:       ts,
			timezone: "Asia/Shanghai",
			wantHour: 8,
			wantDay:  21,
		},
		{
			name:     "America/New_York (UTC-5)",
			ts:       ts,
			timezone: "America/New_York",
			wantHour: 19,
			wantDay:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _ := ParseTimezone(tt.timezone)
			got := ToUserTimezone(tt.ts, loc)
			if got.Hour() != tt.wantHour {
				t.Errorf("ToUserTimezone() hour = %v, want %v", got.Hour(), tt.wantHour)
			}
			if got.Day() != tt.wantDay {
				t.Errorf("ToUserTimezone() day = %v, want %v", got.Day(), tt.wantDay)
			}
		})
	}
}

func TestFormatScheduleTime(t *testing.T) {
	// 2025-01-21 14:00:00 UTC
	startTs := int64(1737468000)

	tests := []struct {
		name     string
		startTs  int64
		endTs    *int64
		allDay   bool
		tz       string
		wantContains string
	}{
		{
			name:    "all-day event",
			startTs: startTs,
			endTs:   nil,
			allDay:  true,
			tz:      "UTC",
			wantContains: "2025-01-21",
		},
		{
			name:     "event with end time",
			startTs:  startTs,
			endTs:    func() *int64 { t := int64(1737471600); return &t }(), // 15:00
			allDay:   false,
			tz:       "UTC",
			wantContains: "14:00 - 15:00",
		},
		{
			name:     "event without end time",
			startTs:  startTs,
			endTs:    nil,
			allDay:   false,
			tz:       "UTC",
			wantContains: "14:00",
		},
		{
			name:     "Asia/Shanghai timezone",
			startTs:  startTs,
			endTs:    func() *int64 { t := int64(1737471600); return &t }(),
			allDay:   false,
			tz:       "Asia/Shanghai",
			wantContains: "22:00 - 23:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _ := ParseTimezone(tt.tz)
			got := FormatScheduleTime(tt.startTs, tt.endTs, tt.allDay, loc)
			if !contains(got, tt.wantContains) {
				t.Errorf("FormatScheduleTime() = %v, want to contain %v", got, tt.wantContains)
			}
		})
	}
}

func TestFormatScheduleForContext(t *testing.T) {
	startTs := int64(1737468000) // 2025-01-21 14:00:00 UTC
	endTs := int64(1737471600)   // 2025-01-21 15:00:00 UTC

	loc, _ := ParseTimezone("UTC")
	got := FormatScheduleForContext(startTs, &endTs, "Team Meeting", "Room A", false, 0, loc)

	want := "1. 2025-01-21 14:00 - 15:00 - Team Meeting @ Room A"
	if got != want {
		t.Errorf("FormatScheduleForContext() = %v, want %v", got, want)
	}
}

func TestStartOfDay(t *testing.T) {
	// 2025-01-21 14:30:00 UTC
	testTime := time.Date(2025, 1, 21, 14, 30, 0, 0, time.UTC)

	loc, _ := ParseTimezone("Asia/Shanghai")
	got := StartOfDay(testTime, loc)

	// Should be 2025-01-21 00:00:00 Asia/Shanghai
	// which is 2025-01-20 16:00:00 UTC
	want := time.Date(2025, 1, 20, 16, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("StartOfDay() = %v, want %v", got, want)
	}
}

func TestEndOfDay(t *testing.T) {
	// 2025-01-21 14:30:00 UTC
	testTime := time.Date(2025, 1, 21, 14, 30, 0, 0, time.UTC)

	loc, _ := ParseTimezone("Asia/Shanghai")
	got := EndOfDay(testTime, loc)

	// Should be 2025-01-21 23:59:59.999999999 Asia/Shanghai
	// which is 2025-01-21 15:59:59.999999999 UTC
	// But the result is in Shanghai timezone, so hour should be 23
	expectedHour := 23
	if got.Hour() != expectedHour {
		t.Errorf("EndOfDay() hour = %v, want %v", got.Hour(), expectedHour)
	}

	// Verify it's in the correct timezone
	if got.Location() != loc {
		t.Errorf("EndOfDay() location = %v, want %v", got.Location(), loc)
	}

	// Verify the date is correct
	if got.Day() != 21 {
		t.Errorf("EndOfDay() day = %v, want %v", got.Day(), 21)
	}
}

func TestNowInTimezone(t *testing.T) {
	loc, _ := ParseTimezone("Asia/Shanghai")
	got := NowInTimezone(loc)

	// Check that the timezone is correctly set
	if got.Location() != loc {
		t.Errorf("NowInTimezone() location = %v, want %v", got.Location(), loc)
	}
}

func TestCommonTimezoneConstants(t *testing.T) {
	// Test that pre-loaded locations are valid
	locations := []*time.Location{
		LocationAsiaShanghai,
		LocationAmericaNewYork,
		LocationAmericaLosAngeles,
		LocationEuropeLondon,
		LocationEuropeParis,
		LocationAsiaTokyo,
		LocationAustraliaSydney,
	}

	for _, loc := range locations {
		if loc == nil {
			t.Errorf("Pre-loaded location is nil")
		}
		now := time.Now().In(loc)
		if now.Location() != loc {
			t.Errorf("Time location mismatch")
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
