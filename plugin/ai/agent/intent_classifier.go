package agent

import (
	"regexp"
	"strings"
)

// TaskIntent represents the type of schedule task.
type TaskIntent string

const (
	// IntentSimpleCreate is for simple single schedule creation.
	IntentSimpleCreate TaskIntent = "simple_create"

	// IntentSimpleQuery is for simple schedule queries.
	IntentSimpleQuery TaskIntent = "simple_query"

	// IntentSimpleUpdate is for simple schedule modifications.
	IntentSimpleUpdate TaskIntent = "simple_update"

	// IntentBatchCreate is for batch schedule creation (e.g., "每天", "每周").
	IntentBatchCreate TaskIntent = "batch_create"

	// IntentConflictResolve is for handling schedule conflicts.
	IntentConflictResolve TaskIntent = "conflict_resolve"

	// IntentMultiQuery is for queries that span multiple domains.
	IntentMultiQuery TaskIntent = "multi_query"
)

// IntentClassifier classifies user input into task intents.
// This is used to route requests to the appropriate execution mode:
// - Simple tasks (IntentSimpleCreate, IntentSimpleQuery, IntentSimpleUpdate) → ReAct mode
// - Batch tasks (IntentBatchCreate) → Plan-Execute mode
type IntentClassifier struct {
	// Batch keywords that trigger Plan-Execute mode
	batchKeywords []string

	// Query keywords that indicate a search/list operation
	queryKeywords []string

	// Update keywords that indicate a modification
	updateKeywords []string

	// Compiled regex patterns for batch detection
	batchPatterns []*regexp.Regexp
}

// NewIntentClassifier creates a new IntentClassifier with default patterns.
func NewIntentClassifier() *IntentClassifier {
	ic := &IntentClassifier{
		// Keywords that suggest batch operations
		batchKeywords: []string{
			// Chinese
			"每天", "每日", "每周", "每月", "每年",
			"这周每天", "下周每天", "本周每天",
			"连续", "批量", "所有", "全部",
			"周一到周五", "工作日",
			// English
			"every day", "daily", "every week", "weekly",
			"every month", "monthly", "batch",
			"all days", "weekdays",
		},

		// Keywords that suggest queries
		queryKeywords: []string{
			// Chinese
			"有什么", "有哪些", "什么安排", "什么日程",
			"查看", "查询", "列出", "显示",
			"今天", "明天", "后天", "这周", "下周", "本周",
			"有空吗", "空闲", "忙吗",
			// English
			"what", "list", "show", "display", "query",
			"schedule", "schedules", "free", "busy", "available",
		},

		// Keywords that suggest updates
		updateKeywords: []string{
			// Chinese
			"改到", "改成", "修改", "调整", "更新",
			"推迟", "提前", "延后", "取消", "删除",
			"移到", "换到",
			// English
			"change", "modify", "update", "reschedule",
			"postpone", "cancel", "delete", "move",
		},
	}

	// Compile batch patterns
	ic.batchPatterns = []*regexp.Regexp{
		// "每[天|周|月]...做..." pattern
		regexp.MustCompile(`每[天日周月年]`),
		// "从...到..." range pattern
		regexp.MustCompile(`从.+到.+[每|都]`),
		// "周一至周五" pattern
		regexp.MustCompile(`周[一二三四五六日]至?到?周[一二三四五六日]`),
		// "下周所有" pattern
		regexp.MustCompile(`[这下本]周(所有|每[天日])`),
	}

	return ic
}

// Classify determines the intent of the user input.
func (ic *IntentClassifier) Classify(input string) TaskIntent {
	lowerInput := strings.ToLower(input)

	// Check for batch patterns first (highest priority)
	if ic.isBatchIntent(input, lowerInput) {
		return IntentBatchCreate
	}

	// Check for update intent
	if ic.isUpdateIntent(input, lowerInput) {
		return IntentSimpleUpdate
	}

	// Check for query intent
	if ic.isQueryIntent(input, lowerInput) {
		return IntentSimpleQuery
	}

	// Default to simple create
	return IntentSimpleCreate
}

// isBatchIntent checks if the input suggests a batch operation.
func (ic *IntentClassifier) isBatchIntent(input, lowerInput string) bool {
	// Check regex patterns
	for _, pattern := range ic.batchPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// Check keywords
	for _, keyword := range ic.batchKeywords {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// isQueryIntent checks if the input suggests a query operation.
func (ic *IntentClassifier) isQueryIntent(input, lowerInput string) bool {
	for _, keyword := range ic.queryKeywords {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			// Make sure it's not also a create request
			// e.g., "今天下午3点开会" is create, not query
			if !ic.hasTimeAndAction(input) {
				return true
			}
		}
	}

	// Questions are usually queries
	if strings.Contains(input, "?") || strings.Contains(input, "？") {
		return true
	}

	return false
}

// isUpdateIntent checks if the input suggests an update operation.
func (ic *IntentClassifier) isUpdateIntent(input, lowerInput string) bool {
	for _, keyword := range ic.updateKeywords {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// hasTimeAndAction checks if the input has both a time reference and an action.
// This helps distinguish "今天有什么" (query) from "今天3点开会" (create).
func (ic *IntentClassifier) hasTimeAndAction(input string) bool {
	// Simple heuristic: if it has a specific time (hour/minute), it's likely a create
	timePatterns := []*regexp.Regexp{
		regexp.MustCompile(`\d{1,2}[点时:]`),
		regexp.MustCompile(`上午|下午|早上|晚上|中午`),
		regexp.MustCompile(`\d{1,2}:\d{2}`),
	}

	for _, pattern := range timePatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	return false
}

// ShouldUsePlanExecute returns true if the intent should use Plan-Execute mode.
func (ic *IntentClassifier) ShouldUsePlanExecute(intent TaskIntent) bool {
	switch intent {
	case IntentBatchCreate, IntentMultiQuery:
		return true
	default:
		return false
	}
}

// ClassifyAndRoute is a convenience method that classifies and returns the execution mode.
func (ic *IntentClassifier) ClassifyAndRoute(input string) (TaskIntent, bool) {
	intent := ic.Classify(input)
	usePlanExecute := ic.ShouldUsePlanExecute(intent)
	return intent, usePlanExecute
}
