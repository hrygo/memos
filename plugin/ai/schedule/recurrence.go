package schedule

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RecurrenceRule represents a simplified recurrence rule.
// We use a custom JSON format instead of full RFC 5545 RRULE for simplicity.
type RecurrenceRule struct {
	Type     string `json:"type"`      // "daily", "weekly", "monthly"
	Interval int    `json:"interval"`  // Every N days/weeks/months
	Weekdays []int  `json:"weekdays"`  // Only for type="weekly": [1,2,3,4,5] (Mon-Fri)
	MonthDay int    `json:"month_day"` // Only for type="monthly": day of month (1-31)
}

// ParseRecurrenceRule parses a natural language recurrence pattern.
// Examples:
//   - "每天" → {Type: "daily", Interval: 1}
//   - "每3天" → {Type: "daily", Interval: 3}
//   - "每周一" → {Type: "weekly", Weekdays: [1]}
//   - "每周" → {Type: "weekly", Interval: 1}
//   - "每两周" → {Type: "daily", Interval: 14}
//   - "每月15号" → {Type: "monthly", MonthDay: 15}
func ParseRecurrenceRule(text string) (*RecurrenceRule, error) {
	text = strings.TrimSpace(text)

	// Daily patterns
	if matched, _ := regexp.MatchString(`^(每|每天)(\d+)?天?$`, text); matched {
		rule := &RecurrenceRule{Type: "daily", Interval: 1}
		if parts := regexp.MustCompile(`(\d+)`).FindStringSubmatch(text); len(parts) > 1 {
			if interval := parseInt(parts[1]); interval > 0 {
				rule.Interval = interval
			}
		}
		return rule, nil
	}

	// Weekly patterns
	if matched, _ := regexp.MatchString(`^每(\d+)?(周|星期)(一|二|三|四|五|六|日|天)?$`, text); matched {
		rule := &RecurrenceRule{Type: "weekly", Interval: 1, Weekdays: []int{1, 2, 3, 4, 5}} // Default weekdays

		// Check for specific weekday
		weekdayMap := map[string]int{
			"一": 1, "二": 2, "三": 3, "四": 4, "五": 5,
			"六": 6, "日": 7, "天": 7,
		}
		if dayStr := regexp.MustCompile(`(周|星期)(一|二|三|四|五|六|日|天)`).FindStringSubmatch(text); len(dayStr) > 2 {
			day := dayStr[2]
			if weekdayNum, ok := weekdayMap[day]; ok {
				rule.Weekdays = []int{weekdayNum}
			}
		}

		// Check for interval
		if parts := regexp.MustCompile(`(\d+)`).FindStringSubmatch(text); len(parts) > 1 {
			if interval := parseInt(parts[1]); interval > 0 {
				rule.Interval = interval
			}
		}

		return rule, nil
	}

	// Monthly patterns
	if matched, _ := regexp.MatchString(`^每(月)(\d{1,2})号?$`, text); matched {
		rule := &RecurrenceRule{Type: "monthly", MonthDay: 0, Interval: 1}
		if parts := regexp.MustCompile(`(\d{1,2})`).FindStringSubmatch(text); len(parts) > 1 {
			if day := parseInt(parts[1]); day >= 1 && day <= 31 {
				rule.MonthDay = day
			}
		}
		if rule.MonthDay == 0 {
			return nil, fmt.Errorf("invalid day of month: %s", text)
		}
		return rule, nil
	}

	return nil, fmt.Errorf("unsupported recurrence pattern: %s", text)
}

// GenerateInstances generates all occurrence timestamps within a time range.
// startTs: The start timestamp of the first occurrence
// endTs: The end timestamp for generating instances (0 means no limit)
// Returns a slice of timestamps for each occurrence.
func (r *RecurrenceRule) GenerateInstances(startTs int64, endTs int64) []int64 {
	var instances []int64

	// Safety check
	if startTs <= 0 {
		return instances
	}

	startTime := time.Unix(startTs, 0).In(time.UTC)
	endTime := time.Now().In(time.UTC).Add(365 * 24 * time.Hour) // Default to 1 year limit

	if endTs > 0 {
		endTime = time.Unix(endTs, 0).In(time.UTC)
	}

	switch r.Type {
	case "daily":
		instances = r.generateDailyInstances(startTime, endTime)

	case "weekly":
		instances = r.generateWeeklyInstances(startTime, endTime)

	case "monthly":
		instances = r.generateMonthlyInstances(startTime, endTime)
	}

	return instances
}

// generateDailyInstances generates instances for daily recurrence.
func (r *RecurrenceRule) generateDailyInstances(start, end time.Time) []int64 {
	var instances []int64
	current := start

	// Prevent infinite loops - limit to 10 years or 1000 instances
	maxInstances := 1000
	count := 0

	for current.Before(end) || current.Equal(end) {
		if count >= maxInstances {
			break
		}

		instances = append(instances, current.Unix())
		current = current.AddDate(0, 0, r.Interval)
		count++
	}

	return instances
}

// generateWeeklyInstances generates instances for weekly recurrence.
func (r *RecurrenceRule) generateWeeklyInstances(start, end time.Time) []int64 {
	var instances []int64
	current := start

	maxInstances := 520 // ~10 years
	count := 0

	for current.Before(end) || current.Equal(end) {
		if count >= maxInstances {
			break
		}

		// Check if current day matches target weekdays
		if r.matchesWeekday(current) {
			instances = append(instances, current.Unix())
		}

		// Move to next week
		current = current.AddDate(0, 0, 7*r.Interval)
		count++
	}

	return instances
}

// generateMonthlyInstances generates instances for monthly recurrence.
func (r *RecurrenceRule) generateMonthlyInstances(start, end time.Time) []int64 {
	var instances []int64
	// Iterate from 1st of start month to avoid skipping months when adding days to 31st
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)

	maxInstances := 120 // ~10 years
	count := 0

	for current.Before(end) || current.Equal(end) {
		if count >= maxInstances {
			break
		}

		// Find the target day in this month
		targetDay := r.MonthDay
		if targetDay > 28 {
			// Adjust for months with fewer days
			lastDay := getLastDayOfMonth(current.Year(), current.Month())
			if targetDay > lastDay {
				targetDay = lastDay
			}
		}

		// Create date for target day, preserving the original timezone
		instanceTime := time.Date(current.Year(), current.Month(), targetDay, 0, 0, 0, 0, start.Location())

		// Only add if it's the same or after start time
		if instanceTime.Equal(start) || instanceTime.After(start) {
			instances = append(instances, instanceTime.Unix())
		}

		// Move to next month
		current = current.AddDate(0, r.Interval, 0)
		count++
	}

	return instances
}

// matchesWeekday checks if the given time matches the target weekdays.
func (r *RecurrenceRule) matchesWeekday(t time.Time) bool {
	weekdays := r.Weekdays
	if len(weekdays) == 0 {
		// Default to all weekdays (Mon-Fri)
		weekdays = []int{1, 2, 3, 4, 5}
	}

	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Convert Sunday from 0 to 7
	}

	for _, target := range weekdays {
		if target == weekday {
			return true
		}
	}
	return false
}

// ToJSON converts the recurrence rule to JSON string.
func (r *RecurrenceRule) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ParseRecurrenceRuleFromJSON parses a recurrence rule from JSON string.
func ParseRecurrenceRuleFromJSON(jsonStr string) (*RecurrenceRule, error) {
	var rule RecurrenceRule
	if err := json.Unmarshal([]byte(jsonStr), &rule); err != nil {
		return nil, err
	}
	// Normalize type to lowercase to handle LLM variations (Daily, DAILY, etc.)
	rule.Type = strings.ToLower(rule.Type)
	return &rule, nil
}

// getLastDayOfMonth returns the last day of the month.
func getLastDayOfMonth(year int, month time.Month) int {
	// First day of next month minus 1 day
	firstOfMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	return firstOfMonth.AddDate(0, 0, -1).Day()
}

// parseInt parses an integer from string (for recurrence interval/day).
func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	return val
}
