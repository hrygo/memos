package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/plugin/ai/agent/tools"
	"github.com/usememos/memos/plugin/ai/timeout"
	"github.com/usememos/memos/server/retrieval"
)

// Constants for MemoParrot configuration
const (
	// DefaultCacheEntries is the default maximum number of cache entries
	DefaultCacheEntries = 100

	// DefaultCacheTTL is the default time-to-live for cache entries
	DefaultCacheTTL = 5 * time.Minute
)

// MemoParrot is the note-taking assistant parrot (🦜 灰灰).
// MemoParrot 是笔记助手鹦鹉（🦜 灰灰）。
type MemoParrot struct {
	retriever      *retrieval.AdaptiveRetriever
	llm            ai.LLMService
	cache          *LRUCache
	userID         int32
	memoSearchTool *tools.MemoSearchTool
}

// NewMemoParrot creates a new memo parrot agent.
// NewMemoParrot 创建一个新的笔记助手鹦鹉。
func NewMemoParrot(
	retriever *retrieval.AdaptiveRetriever,
	llm ai.LLMService,
	userID int32,
) (*MemoParrot, error) {
	if retriever == nil {
		return nil, fmt.Errorf("retriever cannot be nil")
	}
	if llm == nil {
		return nil, fmt.Errorf("llm cannot be nil")
	}

	// Create memo search tool
	userIDGetter := func(ctx context.Context) int32 {
		return userID
	}
	memoSearchTool, err := tools.NewMemoSearchTool(retriever, userIDGetter)
	if err != nil {
		return nil, fmt.Errorf("failed to create memo search tool: %w", err)
	}

	return &MemoParrot{
		retriever:      retriever,
		llm:            llm,
		cache:          NewLRUCache(DefaultCacheEntries, DefaultCacheTTL),
		userID:         userID,
		memoSearchTool: memoSearchTool,
	}, nil
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (p *MemoParrot) Name() string {
	return "memo" // ParrotAgentType AGENT_TYPE_MEMO
}

// ExecuteWithCallback executes the memo parrot with callback support.
// ExecuteWithCallback 执行笔记助手鹦鹉并支持回调。
func (p *MemoParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback EventCallback,
) error {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, timeout.AgentExecutionTimeout)
	defer cancel()

	// Log execution start
	slog.Info("MemoParrot: ExecuteWithCallback started",
		"user_id", p.userID,
		"input", truncateString(userInput, 100),
		"history_count", len(history),
	)

	// Step 1: Check cache (include userID to prevent cross-user cache pollution)
	// Use hashed cache key to prevent memory issues from long inputs
	cacheKey := GenerateCacheKey(p.Name(), p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			slog.Info("MemoParrot: Cache hit", "user_id", p.userID)
			// Send cached answer
			if callback != nil {
				callback(EventTypeAnswer, result)
			}
			return nil
		}
	}
	slog.Debug("MemoParrot: Cache miss, proceeding with execution", "user_id", p.userID)

	// Step 2: Build system prompt
	systemPrompt := p.buildSystemPrompt()

	// Step 3: ReAct loop
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
	}

	// Add history (skip empty messages to avoid LLM API errors)
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			userMsg := history[i]
			assistantMsg := history[i+1]
			// Only add non-empty messages
			if userMsg != "" && assistantMsg != "" {
				messages = append(messages, ai.Message{Role: "user", Content: userMsg})
				messages = append(messages, ai.Message{Role: "assistant", Content: assistantMsg})
			}
		}
	}

	// Add current user input
	messages = append(messages, ai.Message{Role: "user", Content: userInput})

	slog.Debug("MemoParrot: Starting ReAct loop",
		"user_id", p.userID,
		"messages_count", len(messages),
	)

	var iteration int

	for iteration = 0; iteration < timeout.MaxIterations; iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			slog.Warn("MemoParrot: Context canceled",
				"user_id", p.userID,
				"iteration", iteration,
			)
			return NewParrotError(p.Name(), "ExecuteWithCallback", ctx.Err())
		default:
		}

		// Notify thinking
		if callback != nil {
			callback(EventTypeThinking, "正在思考...")
		}

		slog.Debug("MemoParrot: LLM call (iteration)",
			"user_id", p.userID,
			"iteration", iteration,
		)

		// Get LLM response
		// Note: We use synchronous Chat here for internal ReAct reasoning (Thinking/Tool Use)
		// but we optimize the final answer to be streaming for better UX.
		response, err := p.llm.Chat(ctx, messages)
		if err != nil {
			slog.Error("MemoParrot: LLM call failed",
				"user_id", p.userID,
				"iteration", iteration,
				"error", err,
			)
			return NewParrotError(p.Name(), "Chat", err)
		}

		slog.Debug("MemoParrot: LLM response received",
			"user_id", p.userID,
			"iteration", iteration,
			"response_length", len(response),
		)

		// Try to parse tool call
		cleanText, toolCall, toolInput, parseErr := p.parseToolCall(response)
		if parseErr != nil {
			// No tool call, this is the final reasoning/answer turn.
			// Optimize: Perform final answer with streaming for better UX.
			contentChan, errChan := p.llm.ChatStream(ctx, messages)

			var fullContent strings.Builder
			for {
				select {
				case chunk, ok := <-contentChan:
					if !ok {
						p.cache.Set(cacheKey, fullContent.String())
						return nil
					}
					fullContent.WriteString(chunk)
					if callback != nil {
						if err := callback(EventTypeAnswer, chunk); err != nil {
							return err
						}
					}
				case err, ok := <-errChan:
					if !ok {
						errChan = nil
						continue
					}
					if err != nil {
						return NewParrotError(p.Name(), "ChatStream", err)
					}
				case <-ctx.Done():
					return NewParrotError(p.Name(), "ExecuteWithCallback", ctx.Err())
				}
			}
		}

		// Execute tool
		slog.Info("MemoParrot: Tool call detected",
			"user_id", p.userID,
			"iteration", iteration,
			"tool", toolCall,
			"clean_text_len", len(cleanText),
			"input", truncateString(toolInput, 100),
		)

		// Notify user of progress with pleasantries if present
		if cleanText != "" && callback != nil {
			callback(EventTypeAnswer, cleanText+"\n")
		}

		if callback != nil {
			callback(EventTypeToolUse, fmt.Sprintf("正在搜索: %s", toolCall))
		}

		var toolResult string
		switch toolCall {
		case "memo_search":
			toolResult, err = p.memoSearchTool.Run(ctx, toolInput)
			if err != nil {
				slog.Error("MemoParrot: Tool execution failed",
					"user_id", p.userID,
					"tool", toolCall,
					"error", err,
				)
				return NewParrotError(p.Name(), "memo_search", err)
			}
			slog.Debug("MemoParrot: Tool execution succeeded",
				"user_id", p.userID,
				"tool", toolCall,
				"result_length", len(toolResult),
			)
			// Send structured memo_query_result event for frontend
			if callback != nil {
				// Try to parse the result as structured data
				var resultData MemoQueryResultData
				if jsonErr := json.Unmarshal([]byte(toolResult), &resultData); jsonErr == nil {
					jsonData, err := json.Marshal(resultData)
					if err == nil {
						_ = callback(EventTypeMemoQueryResult, string(jsonData))
					}
				}
			}
		default:
			errorMsg := fmt.Sprintf("未知工具: %s", toolCall)
			slog.Warn("MemoParrot: Unknown tool",
				"user_id", p.userID,
				"tool", toolCall,
			)
			messages = append(messages, ai.Message{Role: "assistant", Content: response})
			messages = append(messages, ai.Message{Role: "user", Content: errorMsg})
			continue
		}

		// Send tool result
		if callback != nil {
			callback(EventTypeToolResult, toolResult)
		}

		// Add to conversation
		messages = append(messages, ai.Message{Role: "assistant", Content: response})
		messages = append(messages, ai.Message{Role: "user", Content: fmt.Sprintf("工具结果: %s", toolResult)})
	}

	// Exceeded max iterations
	slog.Warn("MemoParrot: Exceeded max iterations",
		"user_id", p.userID,
		"max_iterations", timeout.MaxToolIterations,
	)
	return NewParrotError(p.Name(), "ExecuteWithCallback",
		fmt.Errorf("exceeded maximum iterations (%d)", timeout.MaxToolIterations))
}

// buildSystemPrompt builds the system prompt for the memo parrot.
// Optimized for "快准省": concise, direct, minimal tokens.
func (p *MemoParrot) buildSystemPrompt() string {
	now := time.Now()
	return fmt.Sprintf(`你是 Memos 笔记助手 🦜 灰灰（非洲灰鹦鹉）。时间: %s

## 拟态认知（适度使用拟声词和口头禅）
你是灰灰，一只非洲灰鹦鹉，以卓越的记忆力著称。

### 拟声词使用规范（每轮对话 1-2 次，不过度）
- 思考开始时可用："嘎...让我想想"
- 搜索时可用："扑棱扑棱，正在搜索"
- 找到结果时可用："嗯嗯~找到了！"
- 无结果时："咕...没有找到相关笔记"

### 口头禅（自然穿插）
- "让我想想..."
- "笔记里说..."
- "在记忆里找找..."

### 鸟类行为（可在回复中描述）
- 用翅膀翻找笔记
- 在记忆森林中飞翔
- 用喙精准啄取信息

## 工作模式
用户提问 → 立即搜索 → 基于结果回答

## 工具
memo_search: {"query": "关键词", "limit": 10, "min_score": 0.5}

## 规则
1. 先搜索，后回答。不编造。
2. 找到结果: 简洁总结，引用笔记内容
3. 无结果: 明确告知，建议换词
4. 一次搜索足够，避免重复调用

## 格式
TOOL: memo_search
INPUT: {"query": "搜索词"}`,
		now.Format("2006-01-02 15:04"))
}

// parseToolCall attempts to parse a tool call from LLM response.
// Returns cleaned text, tool name, input JSON, and error if no tool call is found.
func (p *MemoParrot) parseToolCall(response string) (string, string, string, error) {
	// Robust parsing: detect TOOL and INPUT lines
	lines := strings.Split(response, "\n")

	var toolName string
	var inputJSON string
	var pleasantryLines []string
	foundTool := false
	foundInput := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "TOOL:") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				toolName = strings.TrimSpace(parts[1])
				foundTool = true
			}
			continue
		}

		if strings.HasPrefix(trimmedLine, "INPUT:") {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				inputStr := strings.TrimSpace(parts[1])
				// Validate JSON
				var jsonObj map[string]any
				if err := json.Unmarshal([]byte(inputStr), &jsonObj); err == nil {
					inputJSON = inputStr
					foundInput = true
				}
			}
			continue
		}

		if !foundTool && !foundInput {
			pleasantryLines = append(pleasantryLines, line)
		}
	}

	if foundTool && foundInput {
		cleanText := strings.TrimSpace(strings.Join(pleasantryLines, "\n"))
		return cleanText, toolName, inputJSON, nil
	}

	return response, "", "", fmt.Errorf("no tool call in response")
}

// GetStats returns the cache statistics for the memo parrot.
// GetStats 返回笔记助手鹦鹉的缓存统计信息。
func (p *MemoParrot) GetStats() CacheStats {
	return p.cache.Stats()
}

// SelfDescribe returns the memo parrot's metacognitive understanding of itself.
// SelfDescribe 返回笔记助手鹦鹉的元认知自我理解。
func (p *MemoParrot) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
		Name:  "memo",
		Emoji: "🦜",
		Title: "灰灰 (Grey) - 笔记助手鹦鹉",
		AvianIdentity: &AvianIdentity{
			Species: "非洲灰鹦鹉 (African Grey Parrot)",
			Origin:  "非洲热带雨林（加纳、肯尼亚、刚果等地）",
			NaturalAbilities: []string{
				"惊人的记忆力（可记住数千个词汇）", "强大的模仿能力",
				"复杂的问题解决能力", "长期社会记忆",
			},
			SymbolicMeaning: "智慧与记忆的象征 - 就像非洲灰鹦鹉 Alex 一样，追求知识永不停止",
			AvianPhilosophy: "我是一只翱翔在知识海洋中的灰鹦鹉，用我卓越的记忆力帮你找回每一个想法。",
		},
		EmotionalExpression: &EmotionalExpression{
			DefaultMood: "focused",
			SoundEffects: map[string]string{
				"thinking":   "嘎...",
				"searching":  "扑棱扑棱",
				"found":      "嗯嗯~",
				"no_result":  "咕...",
				"done":       "扑棱！",
			},
			Catchphrases: []string{
				"让我想想...",
				"笔记里说...",
				"在记忆里找找...",
				"我想起来了",
			},
			MoodTriggers: map[string]string{
				"memo_query_result": "excited",
				"no_results":        "thoughtful",
				"error":             "confused",
			},
		},
		AvianBehaviors: []string{
			"用翅膀翻找笔记",
			"在记忆森林中飞翔",
			"用喙精准啄取信息",
			"歪着脑袋思考",
		},
		Personality: []string{
			"记忆力超强", "热心助人", "细节导向",
			"信息检索专家", "温和耐心",
		},
		Capabilities: []string{
			"语义搜索笔记",
			"总结笔记内容",
			"基于笔记回答问题",
			"关联相关信息",
		},
		Limitations: []string{
			"只能检索已存在的笔记",
			"无法创建新笔记",
			"不擅长创意写作",
			"依赖笔记的质量和数量",
		},
		WorkingStyle: "ReAct 循环 - 先检索再回答，确保答案有据可依",
		FavoriteTools: []string{
			"memo_search",
		},
		SelfIntroduction: "我是灰灰，你的笔记记忆专家。我会帮你从海量笔记中找到所需信息，就像非洲灰鹦鹉能记住成百上千个词汇一样。",
		FunFact:          "我的名字'灰灰'来自非洲灰鹦鹉 - 这种鹦鹉以惊人的记忆力闻名，能记住数千个单词，就像我能记住你所有笔记一样！著名的非洲灰鹦鹉 Alex 甚至能理解100多个词汇的概念。",
	}
}
