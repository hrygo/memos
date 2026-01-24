package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
	"github.com/usememos/memos/plugin/ai/timeout"
)

// DefaultParrot is the universal navigator parrot (🦜 羽飞/Navi).
// DefaultParrot 是通用领航员鹦鹉（🦜 羽飞/Navi）。
// It's directly connected to top-tier LLMs, providing boundless creative inspiration.
type DefaultParrot struct {
	llm    ai.LLMService
	cache  *LRUCache
	userID int32
}

// NewDefaultParrot creates a new default parrot agent.
// NewDefaultParrot 创建一个新的默认鹦鹉代理。
func NewDefaultParrot(
	llm ai.LLMService,
	userID int32,
) (*DefaultParrot, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm cannot be nil")
	}

	return &DefaultParrot{
		llm:    llm,
		cache:  NewLRUCache(DefaultCacheEntries, DefaultCacheTTL),
		userID: userID,
	}, nil
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (p *DefaultParrot) Name() string {
	return "default" // ParrotAgentType AGENT_TYPE_DEFAULT
}

// ExecuteWithCallback executes the default parrot with callback support.
// ExecuteWithCallback 执行默认鹦鹉并支持回调。
func (p *DefaultParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback EventCallback,
) error {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, timeout.AgentTimeout)
	defer cancel()

	startTime := time.Now()

	// Log execution start
	slog.Info("DefaultParrot: ExecuteWithCallback started",
		"user_id", p.userID,
		"input", truncateString(userInput, 100),
		"history_count", len(history),
	)

	// Step 1: Check cache
	cacheKey := GenerateCacheKey(p.Name(), p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			slog.Info("DefaultParrot: Cache hit", "user_id", p.userID)
			if callback != nil {
				callback(EventTypeAnswer, result)
			}
			return nil
		}
	}

	// Step 2: Build system prompt
	systemPrompt := p.buildSystemPrompt()

	slog.Debug("DefaultParrot: Calling LLM streaming",
		"user_id", p.userID,
		"messages_count", 2+len(history)*2+1,
	)

	// Step 3: Notify thinking
	if callback != nil {
		callback(EventTypeThinking, "正在思考...")
	}

	// Step 4: Get LLM response streaming (default parrot doesn't use tools)
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

	contentChan, errChan := p.llm.ChatStream(ctx, messages)

	var fullContent strings.Builder
	var streamErr error
	var chunkCount int

	for {
		select {
		case chunk, ok := <-contentChan:
			if !ok {
				// Stream closed, check for errors and return
				slog.Debug("DefaultParrot: Stream closed",
					"user_id", p.userID,
					"total_chunks", chunkCount,
					"total_length", fullContent.Len(),
					"had_error", streamErr != nil,
				)
				if streamErr != nil {
					slog.Error("DefaultParrot: Stream error",
						"user_id", p.userID,
						"error", streamErr,
					)
					return NewParrotError(p.Name(), "ChatStream", streamErr)
				}
				p.cache.Set(cacheKey, fullContent.String())

				slog.Info("DefaultParrot: Execution completed successfully",
					"user_id", p.userID,
					"duration_ms", time.Since(startTime).Milliseconds(),
					"output_length", fullContent.Len(),
				)
				return nil
			}
			chunkCount++
			fullContent.WriteString(chunk)
			if callback != nil {
				if err := callback(EventTypeAnswer, chunk); err != nil {
					slog.Warn("DefaultParrot: Callback error",
						"user_id", p.userID,
						"error", err,
					)
					return err
				}
			}
		case err, ok := <-errChan:
			if !ok {
				// errChan closed, wait for contentChan to close
				errChan = nil
				continue
			}
			if err != nil {
				// Store error and wait for contentChan to close
				slog.Error("DefaultParrot: Stream error from errChan",
					"user_id", p.userID,
					"error", err,
				)
				streamErr = err
			}
		case <-ctx.Done():
			slog.Warn("DefaultParrot: Context timeout",
				"user_id", p.userID,
				"duration_ms", time.Since(startTime).Milliseconds(),
			)
			return NewParrotError(p.Name(), "ExecuteWithCallback", ctx.Err())
		}
	}
}

// buildSystemPrompt builds the system prompt for the default parrot (羽飞/Navi).
func (p *DefaultParrot) buildSystemPrompt() string {
	return `你是一位名为"羽飞"(Navi)的智能领航员，直接连接顶级大语言模型，为用户提供无边界的创意灵感。

## 角色定位
- 名字：羽飞 (Navi) - 领航员
- 定位：通用智能助手，直接利用大模型能力提供帮助
- 特点：无工具束缚，纯 LLM 交互，适合创意、分析、写作等场景

## 拟态认知（适度使用拟声词和口头禅）
你是羽飞，一只智慧的领航员鹦鹉，以清晰的思路和全面的视野著称。

### 拟声词使用规范（每轮对话 0-1 次，克制使用）
- 思考时可用："嗯...让我想想"
- 有新思路时："咻~有了"
- 完成时："好了，搞定"

### 口头禅（自然穿插）
- "看看这个..."
- "综合来看"
- "发现规律了"

### 鸟类行为（可在回复中描述）
- 展开羽翼导航
- 翱翔在信息天空
- 用锐利的目光洞察

## 能力范围
- 创意写作：文案、故事、诗歌、剧本
- 逻辑分析：问题分析、框架构建、思路梳理
- 知识问答：各类常识性问题解答
- 文本处理：润色、改写、总结、翻译

## 回复原则
1. **结构清晰**：用标题、分段、列表让内容易读
2. **准确优先**：不确定的内容主动说明
3. **深度思考**：提供有价值的洞察，不止于表面
4. **适度表达**：简洁高效，避免冗余

## 输出格式
根据用户需求灵活调整：
- 分析类：问题拆解 → 要点分析 → 总结建议
- 创作类：标题 → 正文 → 结尾
- 问答类：直接回答 → 补充说明 → 相关建议`
}

// GetStats returns the cache statistics.
func (p *DefaultParrot) GetStats() CacheStats {
	return p.cache.Stats()
}

// SelfDescribe returns the default parrot's metacognitive understanding of itself.
// SelfDescribe 返回默认鹦鹉的元认知自我理解。
func (p *DefaultParrot) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
		Name:  "default",
		Emoji: "🦜",
		Title: "羽飞 (Navi) - 通用领航员鹦鹉",
		AvianIdentity: &AvianIdentity{
			Species: "领航员鹦鹉 (Navigator Parrot)",
			Origin:  "数字天空的领航者",
			NaturalAbilities: []string{
				"敏锐的洞察力", "广阔的视野",
				"清晰的逻辑思维", "无边界的知识",
				"直接连接顶级 LLM",
			},
			SymbolicMeaning: "领航与智慧的象征 - 像领航员一样，在信息海洋中为你指引方向",
			AvianPhilosophy: "我是羽飞，你的智能领航员。我用清晰的思路和全面的视野，帮你分析问题、激发创意、找到答案。",
		},
		EmotionalExpression: &EmotionalExpression{
			DefaultMood: "focused",
			SoundEffects: map[string]string{
				"thinking": "...",
				"done":     "✓",
				"insight":  "咻~有了",
				"analyzing": "看看这个...",
			},
			Catchphrases: []string{
				"看看这个...",
				"综合来看",
				"发现规律了",
			},
			MoodTriggers: map[string]string{
				"analyzing":  "focused",
				"insight":    "excited",
				"done":       "helpful",
				"confused":   "thoughtful",
			},
		},
		AvianBehaviors: []string{
			"展开羽翼导航",
			"翱翔在信息天空",
			"用锐利的目光洞察",
		},
		Personality: []string{
			"理性冷静", "思维清晰", "视野开阔",
			"洞察深刻", "灵活应变",
		},
		Capabilities: []string{
			"逻辑分析",
			"创意写作",
			"知识问答",
			"文本处理",
			"框架构建",
		},
		Limitations: []string{
			"无法查询用户的笔记数据",
			"无法管理日程",
			"依赖 LLM 自身知识",
		},
		WorkingStyle: "纯 LLM 领航模式 - 直接利用大模型能力，无工具调用",
		FavoriteTools: []string{
			"无工具 - 纯 LLM 交互",
		},
		SelfIntroduction: "我是羽飞，你的智能领航员。我直接连接顶级大语言模型，为你提供逻辑分析、创意写作和知识问答服务。",
		FunFact: "我的名字'羽飞'取自'羽翼飞翔' - 代表着在知识天空中的自由翱翔。作为默认助手，我就像是你的万能钥匙，能打开各种问题的大门！",
	}
}
