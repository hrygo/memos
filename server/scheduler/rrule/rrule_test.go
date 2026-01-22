package rrule

import (
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		rrule   string
		want    *Rule
		wantErr bool
	}{
		{
			name:  "simple weekly",
			rrule: "FREQ=WEEKLY",
			want: &Rule{
				Frequency: Weekly,
				Interval:  1,
			},
		},
		{
			name:  "weekly with interval",
			rrule: "FREQ=WEEKLY;INTERVAL=2",
			want: &Rule{
				Frequency: Weekly,
				Interval:  2,
			},
		},
		{
			name:  "weekly with days",
			rrule: "FREQ=WEEKLY;BYDAY=MO,WE,FR",
			want: &Rule{
				Frequency: Weekly,
				Interval:  1,
				ByDay:     []Weekday{Monday, Wednesday, Friday},
			},
		},
		{
			name:  "daily with count",
			rrule: "FREQ=DAILY;COUNT=10",
			want: &Rule{
				Frequency: Daily,
				Interval:  1,
				Count:     10,
			},
		},
		{
			name:  "monthly by month day",
			rrule: "FREQ=MONTHLY;BYMONTHDAY=15",
			want: &Rule{
				Frequency:  Monthly,
				Interval:   1,
				ByMonthDay: []int{15},
			},
		},
		{
			name:  "empty string",
			rrule: "",
			want: &Rule{
				Interval: 1,
			},
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.rrule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareRules(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareRules(a, b *Rule) bool {
	if a.Frequency != b.Frequency || a.Interval != b.Interval || a.Count != b.Count {
		return false
	}
	if len(a.ByDay) != len(b.ByDay) {
		return false
	}
	for i := range a.ByDay {
		if a.ByDay[i] != b.ByDay[i] {
			return false
		}
	}
	if len(a.ByMonthDay) != len(b.ByMonthDay) {
		return false
	}
	for i := range a.ByMonthDay {
		if a.ByMonthDay[i] != b.ByMonthDay[i] {
			return false
		}
	}
	return true
}

func TestGenerator_All(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)

	tests := []struct {
		name         string
		rrule        string
		start        string
		maxGen       int
		wantCount    int
		wantFirstDay string
	}{
		{
			name:      "daily for 5 days",
			rrule:     "FREQ=DAILY;COUNT=5",
			start:     "2024-01-15T10:00:00+08:00",
			maxGen:    10,
			wantCount: 5,
			wantFirstDay: "2024-01-15T10:00:00+08:00",
		},
		{
			name:      "weekly mon/wed/fri",
			rrule:     "FREQ=WEEKLY;BYDAY=MO,WE,FR;COUNT=6",
			start:     "2024-01-15T10:00:00+08:00", // Monday
			maxGen:    10,
			wantCount: 6,
			wantFirstDay: "2024-01-15T10:00:00+08:00",
		},
		{
			name:      "monthly on 15th",
			rrule:     "FREQ=MONTHLY;BYMONTHDAY=15;COUNT=3",
			start:     "2024-01-15T10:00:00+08:00",
			maxGen:    10,
			wantCount: 3,
			wantFirstDay: "2024-01-15T10:00:00+08:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			rule, err := parser.Parse(tt.rrule)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			start, err := time.Parse(time.RFC3339, tt.start)
			if err != nil {
				t.Fatalf("Parse start time error = %v", err)
			}

			gen := NewGenerator(rule, start, loc)
			occurrences := gen.All(tt.maxGen)

			if len(occurrences) != tt.wantCount {
				t.Errorf("All() count = %d, want %d", len(occurrences), tt.wantCount)
			}

			if len(occurrences) > 0 {
				firstDay := occurrences[0].Format(time.RFC3339)
				if firstDay != tt.wantFirstDay {
					t.Errorf("All() first = %s, want %s", firstDay, tt.wantFirstDay)
				}
			}
		})
	}
}

func TestRule_String(t *testing.T) {
	tests := []struct {
		name  string
		rule  *Rule
		want  string
	}{
		{
			name: "simple weekly",
			rule: &Rule{
				Frequency: Weekly,
				Interval:  1,
			},
			want: "FREQ=WEEKLY",
		},
		{
			name: "weekly with days",
			rule: &Rule{
				Frequency: Weekly,
				Interval:  1,
				ByDay:     []Weekday{Monday, Wednesday, Friday},
			},
			want: "FREQ=WEEKLY;BYDAY=MO,WE,FR",
		},
		{
			name: "daily with count",
			rule: &Rule{
				Frequency: Daily,
				Interval:  1,
				Count:     10,
			},
			want: "FREQ=DAILY;COUNT=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rule.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Common RRULE examples for schedules
func ExampleParser() {
	parser := NewParser()

	// Every Monday, Wednesday, Friday
	rule, _ := parser.Parse("FREQ=WEEKLY;BYDAY=MO,WE,FR")
	_ = rule

	// Every day at 10am
	rule, _ = parser.Parse("FREQ=DAILY;BYHOUR=10")
	_ = rule

	// 15th of every month
	rule, _ = parser.Parse("FREQ=MONTHLY;BYMONTHDAY=15")
	_ = rule

	// Every year on January 1st
	rule, _ = parser.Parse("FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1")
	_ = rule
}
