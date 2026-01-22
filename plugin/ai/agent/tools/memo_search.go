package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/usememos/memos/plugin/ai/timeout"
	"github.com/usememos/memos/server/retrieval"
)

const (
	// Default search limit for memo search results.
	defaultSearchLimit = 10

	// Maximum search limit to prevent excessive results.
	maxSearchLimit = 50

	// Default minimum relevance score for search results.
	defaultMinScore = 0.5
)

// MemoSearchTool searches for memos using semantic and keyword search.
// MemoSearchTool 使用语义和关键词搜索来查找笔记。
type MemoSearchTool struct {
	retriever    *retrieval.AdaptiveRetriever
	userIDGetter func(ctx context.Context) int32
}

// NewMemoSearchTool creates a new memo search tool.
// NewMemoSearchTool 创建一个新的笔记搜索工具。
func NewMemoSearchTool(
	retriever *retrieval.AdaptiveRetriever,
	userIDGetter func(ctx context.Context) int32,
) (*MemoSearchTool, error) {
	if retriever == nil {
		return nil, fmt.Errorf("retriever cannot be nil")
	}
	if userIDGetter == nil {
		return nil, fmt.Errorf("userIDGetter cannot be nil")
	}

	return &MemoSearchTool{
		retriever:    retriever,
		userIDGetter: userIDGetter,
	}, nil
}

// Name returns the name of the tool.
// Name 返回工具名称。
func (t *MemoSearchTool) Name() string {
	return "memo_search"
}

// Description returns a description of what the tool does.
// Description 返回工具描述。
func (t *MemoSearchTool) Description() string {
	return `Searches for memos using semantic and keyword search.

Input format: JSON with 'query' field (required), 'limit' (optional, default 10), 'min_score' (optional, default 0.5).

Examples:
- {"query": "Python programming"}
- {"query": "meeting notes", "limit": 5}
- {"query": "important", "limit": 10, "min_score": 0.7}

The tool returns relevant memos with their content and relevance scores.`
}

// MemoSearchInput represents the input for memo search.
// MemoSearchInput 表示笔记搜索的输入。
type MemoSearchInput struct {
	Query     string  `json:"query"`              // Search query (required)
	Limit     int     `json:"limit,omitempty"`    // Maximum number of results (default: 10)
	MinScore  float32 `json:"min_score,omitempty"` // Minimum relevance score (default: 0.5)
	Strategy  string  `json:"strategy,omitempty"`  // Retrieval strategy (optional)
}

// Run executes the memo search tool.
// Run 执行笔记搜索工具。
func (t *MemoSearchTool) Run(ctx context.Context, input string) (string, error) {
	// Add timeout protection for search operation
	ctx, cancel := context.WithTimeout(ctx, timeout.ToolExecutionTimeout)
	defer cancel()

	// Parse input
	var searchInput MemoSearchInput
	if err := json.Unmarshal([]byte(input), &searchInput); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate query
	if strings.TrimSpace(searchInput.Query) == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	// Set defaults
	if searchInput.Limit <= 0 {
		searchInput.Limit = defaultSearchLimit
	}
	if searchInput.Limit > maxSearchLimit {
		searchInput.Limit = maxSearchLimit
	}
	if searchInput.MinScore <= 0 {
		searchInput.MinScore = defaultMinScore
	}

	// Get user ID
	userID := t.userIDGetter(ctx)

	// Set strategy (use memo_semantic_only for focused memo search)
	strategy := searchInput.Strategy
	if strategy == "" {
		strategy = "memo_semantic_only"
	}

	// Execute search
	opts := &retrieval.RetrievalOptions{
		Query:    searchInput.Query,
		UserID:   userID,
		Strategy: strategy,
		Limit:    searchInput.Limit,
		MinScore: searchInput.MinScore,
	}

	results, err := t.retriever.Retrieve(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	// Filter only memo results (exclude schedules)
	var memoResults []*retrieval.SearchResult
	for _, result := range results {
		if result.Type == "memo" {
			memoResults = append(memoResults, result)
		}
	}

	// Format results
	if len(memoResults) == 0 {
		return fmt.Sprintf("No memos found matching query: %s", searchInput.Query), nil
	}

	// Build response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("Found %d memo(s) matching query: %s\n\n", len(memoResults), searchInput.Query))

	for i, result := range memoResults {
		response.WriteString(fmt.Sprintf("%d. [Score: %.2f] %s\n", i+1, result.Score, result.Content))

		// Add memo UID if available
		if result.Memo != nil && result.Memo.UID != "" {
			response.WriteString(fmt.Sprintf("   UID: %s\n", result.Memo.UID))
		}

		response.WriteString("\n")
	}

	return response.String(), nil
}

// MemoSummary represents a simplified memo for query results.
type MemoSummary struct {
	UID     string  `json:"uid"`
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

// MemoSearchToolResult represents the structured result of memo search.
// MemoSearchToolResult 表示笔记搜索的结构化结果。
type MemoSearchToolResult struct {
	Query string         `json:"query"`
	Memos []MemoSummary  `json:"memos"`
	Count int            `json:"count"`
}

// RunWithStructuredResult executes the tool and returns a structured result.
// RunWithStructuredResult 执行工具并返回结构化结果。
func (t *MemoSearchTool) RunWithStructuredResult(ctx context.Context, input string) (*MemoSearchToolResult, error) {
	// Add timeout protection for search operation
	ctx, cancel := context.WithTimeout(ctx, timeout.ToolExecutionTimeout)
	defer cancel()

	// Parse input
	var searchInput MemoSearchInput
	if err := json.Unmarshal([]byte(input), &searchInput); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate query
	if strings.TrimSpace(searchInput.Query) == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Set defaults
	if searchInput.Limit <= 0 {
		searchInput.Limit = 10
	}
	if searchInput.Limit > 50 {
		searchInput.Limit = 50
	}
	if searchInput.MinScore <= 0 {
		searchInput.MinScore = 0.5
	}

	// Get user ID
	userID := t.userIDGetter(ctx)

	// Set strategy
	strategy := searchInput.Strategy
	if strategy == "" {
		strategy = "memo_semantic_only"
	}

	// Execute search
	opts := &retrieval.RetrievalOptions{
		Query:    searchInput.Query,
		UserID:   userID,
		Strategy: strategy,
		Limit:    searchInput.Limit,
		MinScore: searchInput.MinScore,
	}

	results, err := t.retriever.Retrieve(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Filter only memo results and convert to summaries
	var memos []MemoSummary
	for _, result := range results {
		if result.Type == "memo" && result.Memo != nil {
			memos = append(memos, MemoSummary{
				UID:     result.Memo.UID,
				Content: result.Content,
				Score:   result.Score,
			})
		}
	}

	return &MemoSearchToolResult{
		Query: searchInput.Query,
		Memos: memos,
		Count: len(memos),
	}, nil
}
