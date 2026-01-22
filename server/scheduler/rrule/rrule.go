// Package rrule provides RRULE (Recurrence Rule) parsing and generation.
// Supports iCalendar RFC 5545 recurrence rules.
package rrule

import (
	"fmt"
	"strings"
	"time"
)

// Frequency represents the recurrence frequency.
type Frequency string

const (
	Secondly Frequency = "SECONDLY"
	Minutely Frequency = "MINUTELY"
	Hourly   Frequency = "HOURLY"
	Daily    Frequency = "DAILY"
	Weekly   Frequency = "WEEKLY"
	Monthly  Frequency = "MONTHLY"
	Yearly   Frequency = "YEARLY"
)

// Weekday represents the day of week for recurrence.
type Weekday string

const (
	Sunday    Weekday = "SU"
	Monday    Weekday = "MO"
	Tuesday   Weekday = "TU"
	Wednesday Weekday = "WE"
	Thursday  Weekday = "TH"
	Friday    Weekday = "FR"
	Saturday  Weekday = "SA"
)

// Rule represents a parsed recurrence rule.
type Rule struct {
	Frequency  Frequency // FREQ
	Interval   int       // INTERVAL (default 1)
	Count      int       // COUNT (number of occurrences)
	Until      time.Time // UNTIL (end date)
	BySecond   []int     // BYSECOND
	ByMinute   []int     // BYMINUTE
	ByHour     []int     // BYHOUR
	ByDay      []Weekday // BYDAY
	ByMonthDay []int     // BYMONTHDAY
	ByYearDay  []int     // BYYEARDAY
	ByWeekNo   []int     // BYWEEKNO
	ByMonth    []int     // BYMONTH
	BySetPos   []int     // BYSETPOS
	Wkst       Weekday   // WKST (week start)
}

// Parser parses RRULE strings.
type Parser struct{}

// NewParser creates a new RRULE parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses an RRULE string into a Rule struct.
// Example: "FREQ=WEEKLY;BYDAY=MO,WE,FR;COUNT=10"
func (p *Parser) Parse(rrule string) (*Rule, error) {
	rule := &Rule{
		Interval: 1, // Default interval
	}

	if rrule == "" {
		return rule, nil
	}

	parts := strings.Split(rrule, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "FREQ":
			rule.Frequency = Frequency(value)
		case "INTERVAL":
			fmt.Sscanf(value, "%d", &rule.Interval)
		case "COUNT":
			fmt.Sscanf(value, "%d", &rule.Count)
		case "UNTIL":
			// Parse RFC 5545 date-time format: YYYYMMDDTHHmmssZ
			layout := "20060102T150405Z"
			t, err := time.Parse(layout, value)
			if err == nil {
				rule.Until = t
			}
		case "BYDAY":
			rule.ByDay = parseByDay(value)
		case "BYMONTHDAY":
			rule.ByMonthDay = parseIntList(value)
		case "BYMONTH":
			rule.ByMonth = parseIntList(value)
		case "BYHOUR":
			rule.ByHour = parseIntList(value)
		case "BYMINUTE":
			rule.ByMinute = parseIntList(value)
		case "BYSECOND":
			rule.BySecond = parseIntList(value)
		case "WKST":
			rule.Wkst = Weekday(value)
		}
	}

	// Validate
	if rule.Frequency == "" {
		return nil, fmt.Errorf("missing required FREQ in RRULE")
	}
	if rule.Interval < 1 {
		rule.Interval = 1
	}

	return rule, nil
}

func parseByDay(value string) []Weekday {
	parts := strings.Split(value, ",")
	days := make([]Weekday, 0, len(parts))
	for _, part := range parts {
		day := Weekday(strings.TrimSpace(part))
		if day != "" {
			days = append(days, day)
		}
	}
	return days
}

func parseIntList(value string) []int {
	parts := strings.Split(value, ",")
	nums := make([]int, 0, len(parts))
	for _, part := range parts {
		var num int
		fmt.Sscanf(strings.TrimSpace(part), "%d", &num)
		nums = append(nums, num)
	}
	return nums
}

// Generator generates occurrences from a recurrence rule.
type Generator struct {
	rule    *Rule
	start   time.Time // Start date of the recurrence
	timezone *time.Location
}

// NewGenerator creates a new occurrence generator.
func NewGenerator(rule *Rule, start time.Time, timezone *time.Location) *Generator {
	if timezone == nil {
		timezone = time.UTC
	}
	return &Generator{
		rule:     rule,
		start:    start.In(timezone),
		timezone: timezone,
	}
}

// All generates all occurrences up to the limit.
// If COUNT is specified in the rule, it generates exactly that many.
// If UNTIL is specified, it generates occurrences until that date.
// Otherwise, it generates up to maxOccurrences.
func (g *Generator) All(maxOccurrences int) []time.Time {
	var occurrences []time.Time

	// Determine the limit
	limit := maxOccurrences
	if g.rule.Count > 0 {
		limit = g.rule.Count
	}

	current := g.start
	count := 0

	for count < limit {
		// Check UNTIL
		if !g.rule.Until.IsZero() && current.After(g.rule.Until) {
			break
		}

		// Check COUNT
		if g.rule.Count > 0 && count >= g.rule.Count {
			break
		}

		occurrences = append(occurrences, current)
		count++

		// Generate next occurrence
		next := g.next(current)
		if next.IsZero() || next.Equal(current) {
			break
		}
		current = next
	}

	return occurrences
}

// Between generates occurrences between start and end (inclusive).
func (g *Generator) Between(start, end time.Time) []time.Time {
	var occurrences []time.Time

	current := g.start
	for current.Before(end) || current.Equal(end) {
		if (current.After(start) || current.Equal(start)) &&
			(current.Before(end) || current.Equal(end)) {
			occurrences = append(occurrences, current)
		}

		// Check UNTIL
		if !g.rule.Until.IsZero() && current.After(g.rule.Until) {
			break
		}

		next := g.next(current)
		if next.IsZero() || next.Equal(current) {
			break
		}
		current = next
	}

	return occurrences
}

// next calculates the next occurrence after the given time.
func (g *Generator) next(current time.Time) time.Time {
	interval := g.rule.Interval
	if interval < 1 {
		interval = 1
	}

	switch g.rule.Frequency {
	case Daily:
		return current.AddDate(0, 0, interval)

	case Weekly:
		return g.nextWeekly(current)

	case Monthly:
		return g.nextMonthly(current)

	case Yearly:
		return current.AddDate(interval, 0, 0)

	case Hourly:
		return current.Add(time.Duration(interval) * time.Hour)

	case Minutely:
		return current.Add(time.Duration(interval) * time.Minute)

	case Secondly:
		return current.Add(time.Duration(interval) * time.Second)

	default:
		return current.AddDate(0, 0, interval)
	}
}

// nextWeekly calculates the next occurrence for weekly frequency.
func (g *Generator) nextWeekly(current time.Time) time.Time {
	// If BYDAY is specified, find the next matching day
	if len(g.rule.ByDay) > 0 {
		return g.nextWeeklyByDay(current)
	}

	// Otherwise, just add interval weeks
	interval := g.rule.Interval
	if interval < 1 {
		interval = 1
	}
	return current.AddDate(0, 0, interval*7)
}

// nextWeeklyByDay calculates the next occurrence for weekly with BYDAY.
func (g *Generator) nextWeeklyByDay(current time.Time) time.Time {
	// Get the weekday mapping
	weekdayMap := map[time.Weekday]Weekday{
		time.Sunday:    Sunday,
		time.Monday:    Monday,
		time.Tuesday:   Tuesday,
		time.Wednesday: Wednesday,
		time.Thursday:  Thursday,
		time.Friday:    Friday,
		time.Saturday:  Saturday,
	}

	currentWeekday := current.Weekday()
	currentWeekdayStr := weekdayMap[currentWeekday]

	// Find the position of current weekday in BYDAY
	currentPos := -1
	for i, day := range g.rule.ByDay {
		if day == currentWeekdayStr {
			currentPos = i
			break
		}
	}

	// If today is one of the BYDAY and not the last one, find next BYDAY in this week
	if currentPos >= 0 && currentPos < len(g.rule.ByDay)-1 {
		// Move to the next BYDAY in the same week
		nextDay := g.rule.ByDay[currentPos+1]
		nextWeekdayValue := weekdayValue(nextDay)
		currentWeekdayValue := int(currentWeekday)
		if nextWeekdayValue > currentWeekdayValue {
			return current.AddDate(0, 0, nextWeekdayValue-currentWeekdayValue)
		}
		// Wrap around to next week
		return current.AddDate(0, 0, 7-(currentWeekdayValue-nextWeekdayValue))
	}

	// Move to the first BYDAY of the next week
	daysUntilMonday := (7 - int(currentWeekday)) % 7
	firstDay := g.rule.ByDay[0]
	firstDayValue := weekdayValue(firstDay)
	daysToAdd := daysUntilMonday + firstDayValue
	return current.AddDate(0, 0, daysToAdd)
}

// nextMonthly calculates the next occurrence for monthly frequency.
func (g *Generator) nextMonthly(current time.Time) time.Time {
	interval := g.rule.Interval
	if interval < 1 {
		interval = 1
	}

	// If BYMONTHDAY is specified
	if len(g.rule.ByMonthDay) > 0 {
		dayOfMonth := current.Day()
		currentPos := -1
		for i, day := range g.rule.ByMonthDay {
			if day == dayOfMonth {
				currentPos = i
				break
			}
		}

		if currentPos >= 0 && currentPos < len(g.rule.ByMonthDay)-1 {
			// Next day in same month
			nextDay := g.rule.ByMonthDay[currentPos+1]
			if nextDay <= daysInMonth(current.Year(), int(current.Month())) {
				return time.Date(current.Year(), current.Month(), nextDay,
					current.Hour(), current.Minute(), current.Second(), 0, g.timezone)
			}
		}
	}

	// Move to next interval month
	nextMonth := current.AddDate(0, interval, -current.Day()+1)
	targetDay := current.Day()
	if len(g.rule.ByMonthDay) > 0 {
		targetDay = g.rule.ByMonthDay[0]
	}

	maxDay := daysInMonth(nextMonth.Year(), int(nextMonth.Month()))
	if targetDay > maxDay {
		targetDay = maxDay
	}

	return time.Date(nextMonth.Year(), nextMonth.Month(), targetDay,
		current.Hour(), current.Minute(), current.Second(), 0, g.timezone)
}

func weekdayValue(day Weekday) int {
	switch day {
	case Sunday:
		return 0
	case Monday:
		return 1
	case Tuesday:
		return 2
	case Wednesday:
		return 3
	case Thursday:
		return 4
	case Friday:
		return 5
	case Saturday:
		return 6
	}
	return 0
}

func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	}
	return 30
}

func isLeapYear(year int) bool {
	if year%4 != 0 {
		return false
	}
	if year%100 != 0 {
		return true
	}
	return year%400 == 0
}

// String returns the RRULE string representation.
func (r *Rule) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("FREQ=%s", r.Frequency))

	if r.Interval > 1 {
		parts = append(parts, fmt.Sprintf("INTERVAL=%d", r.Interval))
	}

	if r.Count > 0 {
		parts = append(parts, fmt.Sprintf("COUNT=%d", r.Count))
	}

	if !r.Until.IsZero() {
		parts = append(parts, fmt.Sprintf("UNTIL=%s", r.Until.Format("20060102T150405Z")))
	}

	if len(r.ByDay) > 0 {
		dayStrs := make([]string, len(r.ByDay))
		for i, day := range r.ByDay {
			dayStrs[i] = string(day)
		}
		parts = append(parts, fmt.Sprintf("BYDAY=%s", strings.Join(dayStrs, ",")))
	}

	if len(r.ByMonthDay) > 0 {
		parts = append(parts, fmt.Sprintf("BYMONTHDAY=%s", intListToString(r.ByMonthDay)))
	}

	if len(r.ByMonth) > 0 {
		parts = append(parts, fmt.Sprintf("BYMONTH=%s", intListToString(r.ByMonth)))
	}

	if r.Wkst != "" {
		parts = append(parts, fmt.Sprintf("WKST=%s", r.Wkst))
	}

	return strings.Join(parts, ";")
}

func intListToString(nums []int) string {
	strs := make([]string, len(nums))
	for i, num := range nums {
		strs[i] = fmt.Sprintf("%d", num)
	}
	return strings.Join(strs, ",")
}
