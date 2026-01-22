package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/server/finops"
	"github.com/usememos/memos/store"
)

// SummaryService provides AI-powered summarization features.
type SummaryService struct {
	store     *store.Store
	llm       ai.LLMService
	costMonitor *finops.CostMonitor
}

// NewSummaryService creates a new summary service.
func NewSummaryService(st *store.Store, llm ai.LLMService, cm *finops.CostMonitor) *SummaryService {
	return &SummaryService{
		store:      st,
		llm:        llm,
		costMonitor: cm,
	}
}

// SummarizeMemosOptions holds options for memo summarization.
type SummarizeMemosOptions struct {
	UserID      int32
	TimeRange   *TimeRange
	Query       string // Optional filter query
	MaxMemos    int    // Maximum number of memos to summarize
	MaxTokens   int    // Maximum tokens in summary
	Language    string // "zh" or "en"
	Style       string // "brief", "detailed", "bullet_points"
}

// TimeRange represents a time range for filtering.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Contains checks if a time is within the range.
func (tr *TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && !t.After(tr.End)
}

// SummarizeMemosRequest is the request for memo summarization.
type SummarizeMemosRequest struct {
	Options *SummarizeMemosOptions
}

// SummarizeMemosResponse is the response for memo summarization.
type SummarizeMemosResponse struct {
	Summary         string   `json:"summary"`
	MemoCount       int      `json:"memo_count"`
	TopTopics       []string `json:"top_topics,omitempty"`
	TotalChars      int      `json:"total_chars"`
	TimeRange       string   `json:"time_range"`
	ProcessingTime  int64    `json:"processing_time_ms"`
	EstimatedCost   float64  `json:"estimated_cost_usd"`
}

// SummarizeMemos generates a summary of memos within the given criteria.
func (s *SummaryService) SummarizeMemos(ctx context.Context, req *SummarizeMemosRequest) (*SummarizeMemosResponse, error) {
	startTime := time.Now()

	// Set defaults
	if req.Options.MaxMemos <= 0 {
		req.Options.MaxMemos = 50
	}
	if req.Options.MaxTokens <= 0 {
		req.Options.MaxTokens = 500
	}
	if req.Options.Language == "" {
		req.Options.Language = "zh"
	}
	if req.Options.Style == "" {
		req.Options.Style = "brief"
	}

	// Fetch memos
	findMemo := &store.FindMemo{
		CreatorID: &req.Options.UserID,
		Limit:     &req.Options.MaxMemos,
		OrderByUpdatedTs: true, // Get most recent first
	}

	memos, err := s.store.ListMemos(ctx, findMemo)
	if err != nil {
		return nil, fmt.Errorf("failed to list memos: %w", err)
	}

	// Filter by time range if specified
	filteredMemos := make([]*store.Memo, 0, len(memos))
	for _, memo := range memos {
		if req.Options.TimeRange != nil {
			memoTime := time.Unix(memo.CreatedTs, 0)
			if !req.Options.TimeRange.Contains(memoTime) {
				continue
			}
		}
		filteredMemos = append(filteredMemos, memo)
	}

	memos = filteredMemos

	if len(memos) == 0 {
		return &SummarizeMemosResponse{
			Summary:        "没有找到符合条件的笔记。",
			MemoCount:      0,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Build content for summarization
	content := s.buildMemoContent(memos, req.Options.MaxTokens*4) // Allow 4x input for summary

	// Generate summary
	prompt := s.buildMemoSummaryPrompt(content, req.Options)
	summary, err := s.generateSummary(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Extract topics
	topics := s.extractTopics(memos)

	// Build response
	timeRangeStr := "全部时间"
	if req.Options.TimeRange != nil {
		timeRangeStr = fmt.Sprintf("%s 至 %s",
			req.Options.TimeRange.Start.Format("2006-01-02"),
			req.Options.TimeRange.End.Format("2006-01-02"))
	}

	totalChars := 0
	for _, m := range memos {
		totalChars += len(m.Content)
	}

	// Estimate cost (rough estimate for DeepSeek)
	estimatedCost := s.estimateCost(len(content)+len(summary))

	return &SummarizeMemosResponse{
		Summary:        summary,
		MemoCount:      len(memos),
		TopTopics:      topics,
		TotalChars:     totalChars,
		TimeRange:      timeRangeStr,
		ProcessingTime: time.Since(startTime).Milliseconds(),
		EstimatedCost:  estimatedCost,
	}, nil
}

// SummarizeSchedulesOptions holds options for schedule summarization.
type SummarizeSchedulesOptions struct {
	UserID      int32
	TimeRange   *TimeRange
	MaxSchedules int
	Language    string
	Style       string
}

// SummarizeSchedulesRequest is the request for schedule summarization.
type SummarizeSchedulesRequest struct {
	Options *SummarizeSchedulesOptions
}

// SummarizeSchedulesResponse is the response for schedule summarization.
type SummarizeSchedulesResponse struct {
	Summary          string   `json:"summary"`
	ScheduleCount    int      `json:"schedule_count"`
	BusyPeriods      []string `json:"busy_periods,omitempty"`
	FreePeriods      []string `json:"free_periods,omitempty"`
	ConflictCount    int      `json:"conflict_count"`
	TimeRange        string   `json:"time_range"`
	ProcessingTime   int64    `json:"processing_time_ms"`
}

// SummarizeSchedules generates a summary of schedules within the given criteria.
func (s *SummaryService) SummarizeSchedules(ctx context.Context, req *SummarizeSchedulesRequest) (*SummarizeSchedulesResponse, error) {
	startTime := time.Now()

	// Set defaults
	if req.Options.MaxSchedules <= 0 {
		req.Options.MaxSchedules = 100
	}
	if req.Options.Language == "" {
		req.Options.Language = "zh"
	}

	// Fetch schedules
	findSchedule := &store.FindSchedule{
		CreatorID: &req.Options.UserID,
		Limit:     &req.Options.MaxSchedules,
	}

	// Add time range filter
	if req.Options.TimeRange != nil {
		startTs := req.Options.TimeRange.Start.Unix()
		endTs := req.Options.TimeRange.End.Unix()
		findSchedule.StartTs = &startTs
		findSchedule.EndTs = &endTs
	}

	schedules, err := s.store.ListSchedules(ctx, findSchedule)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	if len(schedules) == 0 {
		return &SummarizeSchedulesResponse{
			Summary:        "没有找到符合条件的日程。",
			ScheduleCount:  0,
			ProcessingTime: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Analyze schedules
	busyPeriods, freePeriods := s.analyzeScheduleLoad(schedules, req.Options.TimeRange)

	// Detect conflicts
	conflictCount := s.detectConflicts(schedules)

	// Build content for summarization
	content := s.buildScheduleContent(schedules)

	// Generate summary
	prompt := s.buildScheduleSummaryPrompt(content, req.Options)
	summary, err := s.generateSummary(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Build response
	timeRangeStr := "全部时间"
	if req.Options.TimeRange != nil {
		timeRangeStr = fmt.Sprintf("%s 至 %s",
			req.Options.TimeRange.Start.Format("2006-01-02"),
			req.Options.TimeRange.End.Format("2006-01-02"))
	}

	return &SummarizeSchedulesResponse{
		Summary:       summary,
		ScheduleCount: len(schedules),
		BusyPeriods:   busyPeriods,
		FreePeriods:   freePeriods,
		ConflictCount: conflictCount,
		TimeRange:     timeRangeStr,
		ProcessingTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// buildMemoContent builds content from memos for summarization.
func (s *SummaryService) buildMemoContent(memos []*store.Memo, maxChars int) string {
	var sb strings.Builder
	totalChars := 0

	for i, memo := range memos {
		if totalChars >= maxChars {
			sb.WriteString(fmt.Sprintf("\n... (还有 %d 条笔记未包含)", len(memos)-i))
			break
		}

		// Format: [日期] 标题\n内容\n
		timestamp := time.Unix(memo.CreatedTs, 0).Format("2006-01-02 15:04")
		content := strings.TrimSpace(memo.Content)

		// Truncate long memos
		if len(content) > 500 {
			content = content[:500] + "..."
		}

		sb.WriteString(fmt.Sprintf("[%s]\n%s\n\n", timestamp, content))
		totalChars += len(content) + 30 // Approximate overhead
	}

	return sb.String()
}

// buildMemoSummaryPrompt builds the prompt for memo summarization.
func (s *SummaryService) buildMemoSummaryPrompt(content string, opts *SummarizeMemosOptions) string {
	language := "中文"
	if opts.Language == "en" {
		language = "英文"
	}

	style := "简要"
	if opts.Style == "detailed" {
		style = "详细"
	} else if opts.Style == "bullet_points" {
		style = "要点列表"
	}

	prompt := fmt.Sprintf(`你是一个专业的笔记管理助手。请根据以下笔记内容生成%s%s总结：

**要求**：
1. 使用%s输出
2. 突出主要主题和关键信息
3. 提取 3-5 个核心话题标签
4. %s风格，控制在200字以内

**笔记内容**：
%s

**请直接输出总结，不要包含其他解释文字。`, style, language, language, style, content)

	return prompt
}

// buildScheduleContent builds content from schedules for summarization.
func (s *SummaryService) buildScheduleContent(schedules []*store.Schedule) string {
	var sb strings.Builder

	for _, schedule := range schedules {
		timestamp := time.Unix(schedule.StartTs, 0).Format("2006-01-02 15:04")
		sb.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, schedule.Title))
		if schedule.Description != "" {
			sb.WriteString(fmt.Sprintf("  描述: %s\n", schedule.Description))
		}
		if schedule.Location != "" {
			sb.WriteString(fmt.Sprintf("  地点: %s\n", schedule.Location))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildScheduleSummaryPrompt builds the prompt for schedule summarization.
func (s *SummaryService) buildScheduleSummaryPrompt(content string, opts *SummarizeSchedulesOptions) string {
	language := "中文"
	if opts.Language == "en" {
		language = "英文"
	}

	prompt := fmt.Sprintf(`你是一个专业的时间管理助理。请根据以下日程信息生成%s总结：

**要求**：
1. 使用%s输出
2. 统计日程数量和总时长
3. 识别最忙的时间段
4. 提供时间管理建议
5. 控制在200字以内

**日程内容**：
%s

**请直接输出总结，不要包含其他解释文字。`, language, language, content)

	return prompt
}

// generateSummary generates a summary using the LLM service.
func (s *SummaryService) generateSummary(ctx context.Context, prompt string) (string, error) {
	messages := []ai.Message{
		{Role: "system", Content: "你是一个专业的助理，擅长总结和提取关键信息。"},
		{Role: "user", Content: prompt},
	}

	response, err := s.llm.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	// Clean up response
	summary := strings.TrimSpace(response)
	summary = strings.TrimPrefix(summary, "```")
	summary = strings.TrimSuffix(summary, "```")
	summary = strings.TrimSpace(summary)

	return summary, nil
}

// extractTopics extracts topics from memos using simple keyword extraction.
func (s *SummaryService) extractTopics(memos []*store.Memo) []string {
	// Simple implementation: extract common keywords
	// In production, use proper keyword extraction or topic modeling
	wordCount := make(map[string]int)

	stopWords := map[string]bool{
		"的": true, "了": true, "是": true, "在": true, "和": true,
		"有": true, "我": true, "你": true, "他": true, "她": true,
		"it": true, "is": true, "the": true, "and": true, "to": true,
		"a": true, "an": true, "in": true, "on": true, "at": true,
	}

	for _, memo := range memos {
		words := strings.Fields(memo.Content)
		for _, word := range words {
			word = strings.Trim(word, ".,!?;:()[]{}\"'，。！？；：（）【】「」")
			if len(word) < 2 || stopWords[word] {
				continue
			}
			wordCount[word]++
		}
	}

	// Get top 5 topics
	type topic struct {
		word  string
		count int
	}
	topics := make([]topic, 0)
	for word, count := range wordCount {
		if count >= 2 { // At least 2 occurrences
			topics = append(topics, topic{word, count})
		}
	}

	// Sort by count
	for i := 0; i < len(topics); i++ {
		for j := i + 1; j < len(topics); j++ {
			if topics[j].count > topics[i].count {
				topics[i], topics[j] = topics[j], topics[i]
			}
		}
	}

	// Take top 5
	result := make([]string, 0, 5)
	for i := 0; i < len(topics) && i < 5; i++ {
		result = append(result, topics[i].word)
	}

	return result
}

// analyzeScheduleLoad analyzes schedule load and identifies busy/free periods.
func (s *SummaryService) analyzeScheduleLoad(schedules []*store.Schedule, timeRange *TimeRange) (busyPeriods, freePeriods []string) {
	// Simple implementation: identify consecutive schedules
	// In production, use proper time range analysis

	// Group schedules by date
	schedulesByDate := make(map[string][]*store.Schedule)
	for _, schedule := range schedules {
		date := time.Unix(schedule.StartTs, 0).Format("2006-01-02")
		schedulesByDate[date] = append(schedulesByDate[date], schedule)
	}

	// Find busy days (3+ schedules)
	for date, daySchedules := range schedulesByDate {
		if len(daySchedules) >= 3 {
			busyPeriods = append(busyPeriods, fmt.Sprintf("%s (%d个日程)", date, len(daySchedules)))
		}
	}

	// Find free days (no schedules)
	if timeRange != nil {
		current := timeRange.Start
		for current.Before(timeRange.End) {
			date := current.Format("2006-01-02")
			if schedulesByDate[date] == nil {
				freePeriods = append(freePeriods, date)
			}
			current = current.AddDate(1, 0, 0)
		}
	}

	return busyPeriods, freePeriods
}

// detectConflicts detects schedule conflicts (overlapping time ranges).
func (s *SummaryService) detectConflicts(schedules []*store.Schedule) int {
	conflicts := 0

	for i, s1 := range schedules {
		for _, s2 := range schedules[i+1:] {
			// Check if schedules overlap
			if s1.EndTs != nil && *s1.EndTs > 0 {
				// s1 has end time
				if s2.EndTs != nil && *s2.EndTs > 0 {
					// Both have end time
					if !(*s1.EndTs <= s2.StartTs || *s2.EndTs <= s1.StartTs) {
						conflicts++
					}
				}
			} else {
				// At least one has no end time, check if start times are close (within 1 hour)
				if abs(s1.StartTs-s2.StartTs) < 3600 {
					conflicts++
				}
			}
		}
	}

	return conflicts / 2 // Each conflict counted twice
}

// estimateCost estimates the API cost for a request.
func (s *SummaryService) estimateCost(totalChars int) float64 {
	// Rough estimate for DeepSeek: ~$0.14 per million tokens
	tokens := float64(totalChars) / 2
	cost := (tokens / 1_000_000) * 0.14
	return cost
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
