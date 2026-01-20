package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
)

const (
	// Time constants
	DefaultEventDuration   = time.Hour
	TomorrowTimeOffset     = 24 * time.Hour
	DayAfterTomorrowOffset = 48 * time.Hour
	DefaultHour            = 9 * time.Hour
	HalfHour               = 30 * time.Minute
	OneHour                = 60 * time.Minute

	// Validation constants
	MaxInputLength = 500 // characters

	// Time parsing limits
	MaxRemindersCount = 10
)

// Precompiled regex patterns for performance
var (
	// Time patterns
	hourMinutePattern    = regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	chineseHourPattern   = regexp.MustCompile(`(\d{1,2})点`)
	chineseMinutePattern = regexp.MustCompile(`(\d{1,2})分`)

	// Date patterns
	datePattern = regexp.MustCompile(`(\d{1,4})年?(\d{1,2})月(\d{1,2})日?`)

	// Time and date removal patterns for title parsing
	timePatternsToRemove = []*regexp.Regexp{
		regexp.MustCompile(`\d{1,2}:\d{2}`),
		regexp.MustCompile(`\d{1,2}点\d{1,2}分`),
		regexp.MustCompile(`\d{1,2}点半`),
		regexp.MustCompile(`\d{1,2}点`),
		regexp.MustCompile(`\d{1,4}年\d{1,2}月\d{1,2}日`),
		regexp.MustCompile(`\d{1,2}月\d{1,2}日`),
		regexp.MustCompile(`(周|星期|下周|本)[一二三四五六七日天]`),
	}

	// Location patterns
	locationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`地点[:：](.{2,20})`),
		regexp.MustCompile(`在(.{2,20})`),
		regexp.MustCompile(`位于(.{2,20})`),
		regexp.MustCompile(`@(.{2,20})`),
	}

	// Reminder pattern
	reminderPattern = regexp.MustCompile(`提前(\d+)(分钟|小时|天|分|时)`)

	// Description cleanup patterns
	descriptionCleanupPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\d{1,2}:\d{2}`),
		regexp.MustCompile(`提前\d+(分钟|小时|天|分|时)`),
		regexp.MustCompile(`\s+`),
	}
)

// Helper function to safely convert string to int with error handling
func mustAtoi(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		// This should never happen since regex ensures numeric format
		// Log as warning for debugging
		return 0
	}
	return val
}

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
		// Log the error for debugging (use structured logging in production)
		// log.Warn("Invalid timezone, using default", "timezone", timezone, "error", err)
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

	// Try regex parsing first (faster)
	result, err := p.parseWithRegex(ctx, text)
	if err == nil {
		return result, nil
	}

	// Fall back to LLM parsing for complex cases
	return p.parseWithLLM(ctx, text)
}

// parseWithRegex attempts to parse using regex patterns.
func (p *Parser) parseWithRegex(ctx context.Context, text string) (*ParseResult, error) {
	now := time.Now().In(p.location)

	// Default values
	result := &ParseResult{
		Timezone:  p.location.String(),
		StartTs:   now.Add(TomorrowTimeOffset).Unix(), // Default to tomorrow
		EndTs:     now.Add(TomorrowTimeOffset + DefaultEventDuration).Unix(),
		AllDay:    false,
		Reminders: []*v1pb.Reminder{},
	}

	// Parse time
	startTime, endTime, allDay, err := p.parseTime(text, now)
	if err != nil {
		return nil, err
	}
	result.StartTs = startTime
	result.EndTs = endTime
	result.AllDay = allDay

	// Parse title
	result.Title = p.parseTitle(text, allDay)

	// Parse location
	result.Location = p.parseLocation(text)

	// Parse reminders
	result.Reminders = p.parseReminders(text)

	// Extract description (remaining text after removing title, location, and time keywords)
	result.Description = p.parseDescription(text, result.Title, result.Location)

	return result, nil
}

// parseTime parses time expressions from text.
func (p *Parser) parseTime(text string, now time.Time) (startTs, endTs int64, allDay bool, err error) {
	text = strings.ToLower(text)

	// Check for "全天" or "all day" keywords
	if strings.Contains(text, "全天") || strings.Contains(text, "all day") {
		allDay = true
		startTs = now.Unix()
		endTs = now.Add(24 * time.Hour).Unix()
		return
	}

	// Parse relative dates
	switch {
	case strings.Contains(text, "今天"):
		return p.parseTimeForDate(now, text, false)

	case strings.Contains(text, "明天"):
		tomorrow := now.Add(TomorrowTimeOffset)
		return p.parseTimeForDate(tomorrow, text, false)

	case strings.Contains(text, "后天"):
		dayAfterTomorrow := now.Add(DayAfterTomorrowOffset)
		return p.parseTimeForDate(dayAfterTomorrow, text, false)

	case strings.Contains(text, "下周"):
		return p.parseNextWeek(text, now)
	}

	// Parse absolute dates (e.g., "1月20日", "2024年1月20日")
	if match := datePattern.FindStringSubmatch(text); len(match) > 0 {
		year := now.Year()
		month := mustAtoi(match[2])
		day := mustAtoi(match[3])

		if len(match[1]) > 0 {
			year = mustAtoi(match[1])
		}

		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, p.location)
		if date.Before(now) {
			date = date.AddDate(1, 0, 0)
		}

		return p.parseTimeForDate(date, text, false)
	}

	// Parse time of day (e.g., "下午3点", "15:00")
	return p.parseTimeForDate(now, text, false)
}

// parseTimeForDate parses time for a specific date.
func (p *Parser) parseTimeForDate(date time.Time, text string, allDay bool) (startTs, endTs int64, isAllDay bool, err error) {
	isAllDay = allDay

	// Parse hour:minute format (e.g., "15:00", "15:30")
	if match := hourMinutePattern.FindStringSubmatch(text); len(match) > 0 {
		hour := mustAtoi(match[1])
		minute := mustAtoi(match[2])

		startTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, p.location)
		endTime := startTime.Add(DefaultEventDuration)

		return startTime.Unix(), endTime.Unix(), false, nil
	}

	// Parse Chinese time format (e.g., "下午3点", "上午9点半")
	hour, minute := p.parseChineseTime(text)

	if hour >= 0 {
		startTime := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, p.location)
		endTime := startTime.Add(DefaultEventDuration)

		return startTime.Unix(), endTime.Unix(), false, nil
	}

	// Default to 9:00 AM if no time specified
	startTime := time.Date(date.Year(), date.Month(), date.Day(), int(DefaultHour/time.Hour), 0, 0, 0, p.location)
	endTime := startTime.Add(DefaultEventDuration)

	return startTime.Unix(), endTime.Unix(), false, nil
}

// parseChineseTime parses Chinese time expressions (e.g., "下午3点", "上午9点半").
func (p *Parser) parseChineseTime(text string) (hour, minute int) {
	hour = -1
	minute = 0

	// Check for AM/PM
	if strings.Contains(text, "下午") || strings.Contains(text, "pm") {
		if match := chineseHourPattern.FindStringSubmatch(text); len(match) > 0 {
			hour = mustAtoi(match[1])
			if hour < 12 {
				hour += 12
			}
		}

		// Check for "半" (half hour)
		if strings.Contains(text, "半") {
			minute = 30
		}

		// Check for specific minutes
		if match := chineseMinutePattern.FindStringSubmatch(text); len(match) > 0 {
			minute = mustAtoi(match[1])
		}
	} else if strings.Contains(text, "上午") || strings.Contains(text, "am") {
		if match := chineseHourPattern.FindStringSubmatch(text); len(match) > 0 {
			hour = mustAtoi(match[1])
			if hour == 12 {
				hour = 0
			}
		}

		if strings.Contains(text, "半") {
			minute = 30
		}

		if match := chineseMinutePattern.FindStringSubmatch(text); len(match) > 0 {
			minute = mustAtoi(match[1])
		}
	} else {
		// No AM/PM specified, use 24-hour format
		if match := chineseHourPattern.FindStringSubmatch(text); len(match) > 0 {
			hour = mustAtoi(match[1])
			// Keep 24-hour format as-is: 13点 = 13:00, 23点 = 23:00
			// Don't convert hour % 12, that would incorrectly change 13 to 1
		}

		if strings.Contains(text, "半") {
			minute = 30
		}

		if match := chineseMinutePattern.FindStringSubmatch(text); len(match) > 0 {
			minute = mustAtoi(match[1])
		}
	}

	return hour, minute
}

// parseNextWeek parses "next week" expressions.
func (p *Parser) parseNextWeek(text string, now time.Time) (startTs, endTs int64, allDay bool, err error) {
	weekdayMap := map[string]time.Weekday{
		"周一": time.Monday, "星期一": time.Monday, "一": time.Monday,
		"周二": time.Tuesday, "星期二": time.Tuesday, "二": time.Tuesday,
		"周三": time.Wednesday, "星期三": time.Wednesday, "三": time.Wednesday,
		"周四": time.Thursday, "星期四": time.Thursday, "四": time.Thursday,
		"周五": time.Friday, "星期五": time.Friday, "五": time.Friday,
		"周六": time.Saturday, "星期六": time.Saturday, "六": time.Saturday,
		"周日": time.Sunday, "星期日": time.Sunday, "日": time.Sunday,
	}

	// Find the weekday
	for day, weekday := range weekdayMap {
		if strings.Contains(text, day) {
			daysUntil := int(weekday) - int(now.Weekday())
			if daysUntil <= 0 {
				daysUntil += 7
			}

			targetDate := now.AddDate(0, 0, daysUntil)
			return p.parseTimeForDate(targetDate, text, false)
		}
	}

	// Default to next Monday
	daysUntil := int((time.Monday - now.Weekday() + 7) % 7)
	if daysUntil == 0 {
		daysUntil = 7
	}
	targetDate := now.AddDate(0, 0, daysUntil)

	return p.parseTimeForDate(targetDate, text, false)
}

// parseTitle extracts the title from text.
func (p *Parser) parseTitle(text string, allDay bool) string {
	// Common schedule keywords to remove
	keywords := []string{
		"今天", "明天", "后天", "上午", "下午", "晚上", "早上",
		"下周", "本周", "上周", "全天", "all day",
		"点", "分", "时",
		"在", "于", "地点", "位于",
		"提醒", "通知",
	}

	title := text
	for _, keyword := range keywords {
		title = strings.ReplaceAll(title, keyword, "")
	}

	// Remove time and date patterns using precompiled regexes
	for _, pattern := range timePatternsToRemove {
		title = pattern.ReplaceAllString(title, "")
	}

	// Clean up whitespace
	title = strings.TrimSpace(title)
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	// If title is empty, use default
	if title == "" {
		if allDay {
			return "日程"
		}
		return "会议"
	}

	return title
}

// parseLocation extracts location from text.
func (p *Parser) parseLocation(text string) string {
	// Use precompiled location patterns
	for _, pattern := range locationPatterns {
		if match := pattern.FindStringSubmatch(text); len(match) > 1 {
			location := strings.TrimSpace(match[1])
			// Remove common words that aren't locations
			if !strings.ContainsAny(location, "的") && len(location) > 1 {
				return location
			}
		}
	}

	return ""
}

// parseReminders extracts reminder settings from text.
func (p *Parser) parseReminders(text string) []*v1pb.Reminder {
	reminders := []*v1pb.Reminder{}

	// Check for reminder keywords
	if !strings.Contains(text, "提醒") && !strings.Contains(text, "通知") && !strings.Contains(text, "提前") {
		return reminders
	}

	// Parse "提前X分钟/小时/天" or "提前X分/时/天" using precompiled regex
	if match := reminderPattern.FindStringSubmatch(text); len(match) > 2 {
		value := mustAtoi(match[1])
		unit := match[2]

		// Normalize unit using map
		unitMap := map[string]string{
			"分钟": "minutes",
			"分":  "minutes",
			"小时": "hours",
			"时":  "hours",
			"天":  "days",
		}

		if normalizedUnit, ok := unitMap[unit]; ok {
			unit = normalizedUnit
		} else {
			// Default to minutes if unknown unit
			unit = "minutes"
		}

		reminders = append(reminders, &v1pb.Reminder{
			Type:  "before",
			Value: int32(value),
			Unit:  unit,
		})
	}

	// Default reminder if keyword present but no specific time
	if len(reminders) == 0 {
		reminders = append(reminders, &v1pb.Reminder{
			Type:  "before",
			Value: 15,
			Unit:  "minutes",
		})
	}

	return reminders
}

// parseDescription extracts description from remaining text.
func (p *Parser) parseDescription(text, title, location string) string {
	desc := text

	// Remove title
	if title != "" {
		desc = strings.ReplaceAll(desc, title, "")
	}

	// Remove location
	if location != "" {
		desc = strings.ReplaceAll(desc, location, "")
	}

	// Remove time patterns using precompiled regexes
	for _, pattern := range descriptionCleanupPatterns {
		desc = pattern.ReplaceAllString(desc, "")
	}

	// Clean up
	desc = strings.TrimSpace(desc)

	// Only return description if there's meaningful content
	if len(desc) > 2 && !strings.ContainsAny(desc, "的") {
		return desc
	}

	return ""
}

// parseWithLLM uses LLM to parse complex natural language.
func (p *Parser) parseWithLLM(ctx context.Context, text string) (*ParseResult, error) {
	prompt := fmt.Sprintf(`请解析以下日程文本，提取日程信息。

文本：%s

请以JSON格式返回，包含以下字段：
- title: 日程标题
- description: 详细描述（如果没有则为空字符串）
- location: 地点（如果没有则为空字符串）
- start_ts: 开始时间戳（Unix时间戳，秒）
- end_ts: 结束时间戳（Unix时间戳，秒）
- all_day: 是否全天（true/false）
- reminders: 提醒数组，每个提醒包含 type（"before"），value（数字），unit（"minutes"/"hours"/"days"）

当前时区：%s
当前时间：%s

请只返回JSON，不要有其他内容。`, text, p.location.String(), time.Now().In(p.location).Format("2006-01-02 15:04:05"))

	response, err := p.llmService.Chat(ctx, []ai.Message{
		{Role: "system", Content: "你是一个专业的日程解析助手，擅长从自然语言中提取日程信息。"},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM parsing failed: %w", err)
	}

	// Parse JSON response
	var result ParseResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Set timezone
	result.Timezone = p.location.String()

	return &result, nil
}

// ToSchedule converts ParseResult to v1pb.Schedule.
func (r *ParseResult) ToSchedule() *v1pb.Schedule {
	return &v1pb.Schedule{
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
}
