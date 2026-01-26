package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// IntentResult represents the LLM classification result.
type IntentResult struct {
	Intent     TaskIntent `json:"intent"`
	Confidence float64    `json:"confidence"`
	Reasoning  string     `json:"reasoning,omitempty"`
}

// LLMIntentClassifier uses a lightweight LLM for intent classification.
// This provides better accuracy than rule-based matching, especially for
// nuanced natural language inputs.
type LLMIntentClassifier struct {
	client *openai.Client
	model  string

	// Fallback rule-based classifier for when LLM fails
	fallback *IntentClassifier
}

// LLMIntentConfig holds configuration for the LLM intent classifier.
type LLMIntentConfig struct {
	APIKey  string
	BaseURL string
	Model   string // Recommended: Qwen/Qwen2.5-7B-Instruct
}

// NewLLMIntentClassifier creates a new LLM-based intent classifier.
// Uses a lightweight model optimized for fast classification.
func NewLLMIntentClassifier(cfg LLMIntentConfig) *LLMIntentClassifier {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	model := cfg.Model
	if model == "" {
		// Default to a fast, cost-effective model for classification
		model = "Qwen/Qwen2.5-7B-Instruct"
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = baseURL

	return &LLMIntentClassifier{
		client:   openai.NewClientWithConfig(clientConfig),
		model:    model,
		fallback: NewIntentClassifier(),
	}
}

// Classify determines the intent of the user input using LLM.
func (ic *LLMIntentClassifier) Classify(ctx context.Context, input string) (TaskIntent, error) {
	result, err := ic.ClassifyWithDetails(ctx, input)
	if err != nil {
		slog.Warn("LLM intent classification failed, using fallback",
			"error", err,
			"input", truncateForLog(input, 50))
		return ic.fallback.Classify(input), nil
	}
	return result.Intent, nil
}

// ClassifyWithDetails returns the full classification result including confidence.
func (ic *LLMIntentClassifier) ClassifyWithDetails(ctx context.Context, input string) (*IntentResult, error) {
	// Set timeout for classification (should be fast)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	prompt := ic.buildPrompt(input)

	req := openai.ChatCompletionRequest{
		Model:       ic.model,
		MaxTokens:   50, // Strict schema ensures minimal output
		Temperature: 0,  // Deterministic output
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: intentSystemPromptStrict,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "intent_classification",
				Strict: true,
				Schema: intentJSONSchema,
			},
		},
	}

	start := time.Now()
	resp, err := ic.client.CreateChatCompletion(ctx, req)
	latency := time.Since(start)

	if err != nil {
		slog.Error("LLM intent classification request failed",
			"error", err,
			"latency_ms", latency.Milliseconds())
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	content := resp.Choices[0].Message.Content
	result, err := ic.parseResponse(content)
	if err != nil {
		slog.Warn("Failed to parse LLM response",
			"content", content,
			"error", err)
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	slog.Debug("LLM intent classification completed",
		"input", truncateForLog(input, 30),
		"intent", result.Intent,
		"confidence", result.Confidence,
		"latency_ms", latency.Milliseconds(),
		"tokens", resp.Usage.TotalTokens)

	return result, nil
}

// buildPrompt constructs the classification prompt.
func (ic *LLMIntentClassifier) buildPrompt(input string) string {
	return fmt.Sprintf("用户输入: %s", input)
}

// parseResponse parses the LLM JSON response.
func (ic *LLMIntentClassifier) parseResponse(content string) (*IntentResult, error) {
	// Try to extract JSON from response
	content = strings.TrimSpace(content)

	// Handle potential markdown code blocks
	if strings.HasPrefix(content, "```") {
		re := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```")
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			content = matches[1]
		}
	}

	var raw struct {
		Intent     string  `json:"intent"`
		Confidence float64 `json:"confidence"`
		Reasoning  string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Map string to TaskIntent
	intent := ic.mapIntent(raw.Intent)

	return &IntentResult{
		Intent:     intent,
		Confidence: raw.Confidence,
		Reasoning:  raw.Reasoning,
	}, nil
}

// mapIntent converts string intent to TaskIntent enum.
func (ic *LLMIntentClassifier) mapIntent(s string) TaskIntent {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "simple_create", "create", "add":
		return IntentSimpleCreate
	case "simple_query", "query", "list", "search":
		return IntentSimpleQuery
	case "simple_update", "update", "modify", "change":
		return IntentSimpleUpdate
	case "batch_create", "batch", "recurring":
		return IntentBatchCreate
	case "conflict_resolve", "conflict":
		return IntentConflictResolve
	case "multi_query", "multi":
		return IntentMultiQuery
	default:
		slog.Warn("Unknown intent from LLM, defaulting to simple_create",
			"raw_intent", s)
		return IntentSimpleCreate
	}
}

// ShouldUsePlanExecute returns true if the intent should use Plan-Execute mode.
func (ic *LLMIntentClassifier) ShouldUsePlanExecute(intent TaskIntent) bool {
	switch intent {
	case IntentBatchCreate, IntentMultiQuery:
		return true
	default:
		return false
	}
}

// ClassifyAndRoute is a convenience method that classifies and returns the execution mode.
func (ic *LLMIntentClassifier) ClassifyAndRoute(ctx context.Context, input string) (TaskIntent, bool, error) {
	intent, err := ic.Classify(ctx, input)
	if err != nil {
		return IntentSimpleCreate, false, err
	}
	usePlanExecute := ic.ShouldUsePlanExecute(intent)
	return intent, usePlanExecute, nil
}

// truncateForLog truncates a string for logging purposes.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// intentSystemPromptStrict is a minimal prompt for strict JSON schema mode.
// The schema enforces the output format, so we only need classification rules.
const intentSystemPromptStrict = `日程意图分类器。根据用户输入判断意图：

simple_create: 创建单个日程 (有时间+事件)
simple_query: 查询日程/空闲 (问句)
simple_update: 修改/删除日程
batch_create: 重复日程 (每天/每周/工作日)
conflict_resolve: 处理冲突
multi_query: 综合分析

默认: simple_create`

// intentJSONSchema defines the strict output schema for intent classification.
// Using enum to constrain intent values and prevent hallucination.
var intentJSONSchema = &jsonSchema{
	Type: "object",
	Properties: map[string]*jsonSchema{
		"intent": {
			Type: "string",
			Enum: []string{
				"simple_create",
				"simple_query",
				"simple_update",
				"batch_create",
				"conflict_resolve",
				"multi_query",
			},
			Description: "The classified intent type",
		},
		"confidence": {
			Type:        "number",
			Description: "Confidence score between 0 and 1",
		},
	},
	Required:             []string{"intent", "confidence"},
	AdditionalProperties: false,
}

// jsonSchema implements json.Marshaler for OpenAI's JSON Schema format.
type jsonSchema struct {
	Type                 string                 `json:"type"`
	Properties           map[string]*jsonSchema `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Enum                 []string               `json:"enum,omitempty"`
	Description          string                 `json:"description,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

func (s *jsonSchema) MarshalJSON() ([]byte, error) {
	type alias jsonSchema
	return json.Marshal((*alias)(s))
}
