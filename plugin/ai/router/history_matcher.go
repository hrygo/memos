// Package router provides the LLM routing service.
package router

import (
	"context"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai/memory"
)

// HistoryMatcher implements Layer 2 history-based intent matching.
// Target: ~10ms latency, handle 20%+ of requests that pass Layer 1.
type HistoryMatcher struct {
	memoryService       memory.MemoryService
	similarityThreshold float32
	maxHistoryLookup    int
}

// NewHistoryMatcher creates a new history matcher.
func NewHistoryMatcher(ms memory.MemoryService) *HistoryMatcher {
	return &HistoryMatcher{
		memoryService:       ms,
		similarityThreshold: 0.8,
		maxHistoryLookup:    10,
	}
}

// HistoryMatchResult contains the result of history matching.
type HistoryMatchResult struct {
	Intent     Intent
	Confidence float32
	SourceID   int64 // ID of the matched episode
	Matched    bool
}

// Match attempts to classify intent by finding similar historical patterns.
// Returns matched=true if a similar pattern was found with confidence >= threshold.
func (m *HistoryMatcher) Match(ctx context.Context, userID int32, input string) (*HistoryMatchResult, error) {
	if m.memoryService == nil {
		return &HistoryMatchResult{Matched: false}, nil
	}

	// Search for similar episodes
	episodes, err := m.memoryService.SearchEpisodes(ctx, userID, input, m.maxHistoryLookup)
	if err != nil {
		return nil, err
	}

	if len(episodes) == 0 {
		return &HistoryMatchResult{Matched: false}, nil
	}

	// Find best matching episode
	var bestMatch *memory.EpisodicMemory
	var bestSimilarity float32

	for i := range episodes {
		ep := &episodes[i]
		similarity := m.calculateSimilarity(input, ep.UserInput)

		// Only consider successful outcomes
		if ep.Outcome == "success" && similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = ep
		}
	}

	// Check if similarity meets threshold
	if bestMatch == nil || bestSimilarity < m.similarityThreshold {
		return &HistoryMatchResult{Matched: false}, nil
	}

	// Map agent type to intent
	intent := m.agentTypeToIntent(bestMatch.AgentType, input)

	return &HistoryMatchResult{
		Intent:     intent,
		Confidence: bestSimilarity,
		SourceID:   bestMatch.ID,
		Matched:    true,
	}, nil
}

// calculateSimilarity calculates similarity score between two strings.
// Uses character-level bigrams for Chinese text, which works better than word-based
// tokenization without a proper segmentation library.
// Reference: Character n-gram is effective for short text similarity in CJK languages.
func (m *HistoryMatcher) calculateSimilarity(a, b string) float32 {
	bigramsA := m.extractBigrams(a)
	bigramsB := m.extractBigrams(b)

	if len(bigramsA) == 0 || len(bigramsB) == 0 {
		return 0
	}

	// Calculate Jaccard similarity on bigram sets
	intersection := 0
	for bg := range bigramsA {
		if bigramsB[bg] {
			intersection++
		}
	}

	// Union = |A| + |B| - |A ∩ B|
	union := len(bigramsA) + len(bigramsB) - intersection
	if union == 0 {
		return 0
	}

	return float32(intersection) / float32(union)
}

// extractBigrams extracts character-level bigrams from input.
// This approach works well for Chinese text without requiring a segmentation library.
func (m *HistoryMatcher) extractBigrams(input string) map[string]bool {
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	// Remove common punctuation
	for _, r := range []string{" ", ",", "。", "，", "？", "?", "！", "!", "、"} {
		input = strings.ReplaceAll(input, r, "")
	}

	runes := []rune(input)
	bigrams := make(map[string]bool)

	// Generate character bigrams
	for i := 0; i < len(runes)-1; i++ {
		bigram := string(runes[i : i+2])
		bigrams[bigram] = true
	}

	// Also add individual characters for short inputs
	if len(runes) <= 4 {
		for _, r := range runes {
			bigrams[string(r)] = true
		}
	}

	return bigrams
}

// agentTypeToIntent maps agent type from episode to current intent.
func (m *HistoryMatcher) agentTypeToIntent(agentType, input string) Intent {
	switch agentType {
	case "schedule":
		// Further classify based on input
		if containsAny(input, []string{"查看", "有什么", "哪些"}) {
			return IntentScheduleQuery
		}
		if containsAny(input, []string{"修改", "更新", "取消"}) {
			return IntentScheduleUpdate
		}
		return IntentScheduleCreate
	case "memo":
		if containsAny(input, []string{"搜索", "查找", "找"}) {
			return IntentMemoSearch
		}
		return IntentMemoCreate
	case "amazing":
		return IntentAmazing
	default:
		return IntentUnknown
	}
}

// SaveDecision saves a routing decision to memory for future matching.
func (m *HistoryMatcher) SaveDecision(ctx context.Context, userID int32, input string, intent Intent, success bool) error {
	if m.memoryService == nil {
		return nil
	}

	outcome := "failure"
	if success {
		outcome = "success"
	}

	episode := memory.EpisodicMemory{
		UserID:     userID,
		Timestamp:  time.Now(),
		AgentType:  m.intentToAgentType(intent),
		UserInput:  input,
		Outcome:    outcome,
		Summary:    "routing_decision:" + string(intent),
		Importance: 0.5,
	}

	return m.memoryService.SaveEpisode(ctx, episode)
}

// intentToAgentType maps intent to agent type for storage.
func (m *HistoryMatcher) intentToAgentType(intent Intent) string {
	switch intent {
	case IntentScheduleCreate, IntentScheduleQuery, IntentScheduleUpdate, IntentBatchSchedule:
		return "schedule"
	case IntentMemoSearch, IntentMemoCreate:
		return "memo"
	case IntentAmazing:
		return "amazing"
	default:
		return "unknown"
	}
}
