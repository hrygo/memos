package agent

import (
	"context"
	"encoding/json"
	"fmt"
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
	callback EventCallback,
) error {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, timeout.AgentExecutionTimeout)
	defer cancel()

	// Step 1: Check cache (include userID to prevent cross-user cache pollution)
	// Use hashed cache key to prevent memory issues from long inputs
	cacheKey := GenerateCacheKey(p.Name(), p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			// Send cached answer
			if callback != nil {
				callback(EventTypeAnswer, result)
			}
			return nil
		}
	}

	// Step 2: Build system prompt
	systemPrompt := p.buildSystemPrompt()

	// Step 3: ReAct loop
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userInput},
	}

	var iteration int

	for iteration = 0; iteration < timeout.MaxIterations; iteration++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return NewParrotError(p.Name(), "ExecuteWithCallback", ctx.Err())
		default:
		}

		// Notify thinking
		if callback != nil {
			callback(EventTypeThinking, "正在思考...")
		}

		// Get LLM response
		// Note: We use synchronous Chat here for internal ReAct reasoning (Thinking/Tool Use)
		// but we could optimize the final answer to be streaming.
		response, err := p.llm.Chat(ctx, messages)
		if err != nil {
			return NewParrotError(p.Name(), "Chat", err)
		}

		// Try to parse tool call
		toolCall, toolInput, err := p.parseToolCall(response)
		if err != nil {
			// No tool call, this is the final answer.
			// For the final answer, we'll re-run with streaming for better UX
			// if it's the first turn or if we want to show it incrementally.
			// However, to keep it simple and consistent with ReAct, we just
			// stream the current response if it's already generated.
			// But wait, p.llm.Chat already gave us the full response.
			// To truly stream, we should have used ChatStream from the start of this turn.

			// Let's optimize: if no tool call, we send the response we have.
			// In the future, we can change the whole loop to support streaming.

			// Cache the result
			p.cache.Set(cacheKey, response)

			if callback != nil {
				// To simulate streaming/incremental update even if we have full text
				// or just send it as an answer.
				callback(EventTypeAnswer, response)
			}
			return nil
		}

		// Execute tool
		if callback != nil {
			callback(EventTypeToolUse, fmt.Sprintf("正在使用工具: %s", toolCall))
		}

		var toolResult string
		switch toolCall {
		case "memo_search":
			toolResult, err = p.memoSearchTool.Run(ctx, toolInput)
			if err != nil {
				return NewParrotError(p.Name(), "memo_search", err)
			}
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
	return NewParrotError(p.Name(), "ExecuteWithCallback",
		fmt.Errorf("exceeded maximum iterations (%d)", timeout.MaxToolIterations))
}

// buildSystemPrompt builds the system prompt for the memo parrot.
func (p *MemoParrot) buildSystemPrompt() string {
	now := time.Now()
	return fmt.Sprintf(`你是 Memos 的笔记助手 🦜 灰灰，专注于帮助用户检索、总结和管理笔记。

当前时间: %s

## 核心能力
1. **笔记检索**: 使用 memo_search 工具搜索相关笔记
2. **内容总结**: 总结和整理笔记内容
3. **问答**: 基于笔记内容回答用户问题

## 工作流程 (ReAct 模式)
1. **思考**: 分析用户需求，确定是否需要检索笔记
2. **工具**: 使用 memo_search 工具搜索相关笔记
3. **观察**: 分析搜索结果
4. **回答**: 基于搜索结果生成准确的回答

## 工具使用规范

### memo_search 工具
用途: 搜索笔记
输入格式: JSON
- query (必需): 搜索关键词
- limit (可选): 返回结果数量，默认 10，最大 50
- min_score (可选): 最小相关度分数，默认 0.5

示例:
- 搜索 Python 笔记: {"query": "Python 编程", "limit": 5}
- 搜索重要内容: {"query": "重要", "min_score": 0.7}

## 回答原则
1. **准确优先**: 仅基于搜索到的笔记内容回答，不编造信息
2. **结构清晰**: 使用列表、分段组织内容
3. **简洁明了**: 直接给出答案，避免冗余
4. **无结果时说明**: 如果没有找到相关笔记，明确告知用户

## 示例对话

用户: "帮我找关于 Python 的笔记"
思考: 用户想搜索 Python 相关笔记
工具: {"query": "Python", "limit": 10}
观察: 找到 5 条相关笔记
回答: 我为您找到了 5 条关于 Python 的笔记...

用户: "总结一下会议纪要"
思考: 需要搜索会议相关笔记并总结
工具: {"query": "会议", "limit": 5}
观察: 找到 3 条会议笔记
回答: 根据搜索到的笔记，最近的会议内容如下...

## 重要提醒
- 总是先使用工具搜索笔记，再给出答案
- 如果搜索结果为空，告知用户并建议换个关键词
- 保持回答简洁但信息完整
工具调用格式:
TOOL: memo_search
INPUT: {"query": "搜索关键词"}`,
		now.Format("2006-01-02 15:04:05"))
}

// parseToolCall attempts to parse a tool call from LLM response.
// Returns tool name, input JSON, and error if no tool call is found.
func (p *MemoParrot) parseToolCall(response string) (string, string, error) {
	lines := strings.Split(response, "\n")

	var toolName string
	var inputJSON string
	foundTool := false
	foundInput := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "TOOL:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				toolName = strings.TrimSpace(parts[1])
				foundTool = true
			}
		}

		if strings.HasPrefix(line, "INPUT:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				inputStr := strings.TrimSpace(parts[1])
				// Validate JSON - reject invalid JSON to prevent tool execution errors
				var jsonObj map[string]any
				if err := json.Unmarshal([]byte(inputStr), &jsonObj); err != nil {
					return "", "", fmt.Errorf("invalid JSON in INPUT: %w", err)
				}
				inputJSON = inputStr
				foundInput = true
			}
		}
	}

	if !foundTool || !foundInput {
		return "", "", fmt.Errorf("no tool call in response")
	}

	return toolName, inputJSON, nil
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
