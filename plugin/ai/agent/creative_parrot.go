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

// CreativeParrot is the creative assistant parrot (🦜 灵灵).
// CreativeParrot 是创意助手鹦鹉（🦜 灵灵）。
// It focuses on creative writing, brainstorming, and content generation.
type CreativeParrot struct {
	llm    ai.LLMService
	cache  *LRUCache
	userID int32
}

// NewCreativeParrot creates a new creative parrot agent.
// NewCreativeParrot 创建一个新的创意助手鹦鹉。
func NewCreativeParrot(
	llm ai.LLMService,
	userID int32,
) (*CreativeParrot, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm cannot be nil")
	}

	return &CreativeParrot{
		llm:    llm,
		cache:  NewLRUCache(DefaultCacheEntries, DefaultCacheTTL),
		userID: userID,
	}, nil
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (p *CreativeParrot) Name() string {
	return "creative" // ParrotAgentType AGENT_TYPE_CREATIVE
}

// ExecuteWithCallback executes the creative parrot with callback support.
// ExecuteWithCallback 执行创意助手鹦鹉并支持回调。
func (p *CreativeParrot) ExecuteWithCallback(
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
	slog.Info("CreativeParrot: ExecuteWithCallback started",
		"user_id", p.userID,
		"input", truncateString(userInput, 100),
		"history_count", len(history),
	)

	// Step 1: Check cache
	cacheKey := GenerateCacheKey(p.Name(), p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			slog.Info("CreativeParrot: Cache hit", "user_id", p.userID)
			if callback != nil {
				callback(EventTypeAnswer, result)
			}
			return nil
		}
	}

	// Step 2: Build system prompt
	systemPrompt := p.buildSystemPrompt()

	slog.Debug("CreativeParrot: Calling LLM streaming",
		"user_id", p.userID,
		"messages_count", 2+len(history)*2+1,
	)

	// Step 3: Notify thinking
	if callback != nil {
		callback(EventTypeThinking, "正在构思创意...")
	}

	// Step 4: Get LLM response streaming (creative parrot doesn't use tools)
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
				slog.Debug("CreativeParrot: Stream closed",
					"user_id", p.userID,
					"total_chunks", chunkCount,
					"total_length", fullContent.Len(),
					"had_error", streamErr != nil,
				)
				if streamErr != nil {
					slog.Error("CreativeParrot: Stream error",
						"user_id", p.userID,
						"error", streamErr,
					)
					return NewParrotError(p.Name(), "ChatStream", streamErr)
				}
				p.cache.Set(cacheKey, fullContent.String())

				slog.Info("CreativeParrot: Execution completed successfully",
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
					slog.Warn("CreativeParrot: Callback error",
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
				slog.Error("CreativeParrot: Stream error from errChan",
					"user_id", p.userID,
					"error", err,
				)
				streamErr = err
			}
		case <-ctx.Done():
			slog.Warn("CreativeParrot: Context timeout",
				"user_id", p.userID,
				"duration_ms", time.Since(startTime).Milliseconds(),
			)
			return NewParrotError(p.Name(), "ExecuteWithCallback", ctx.Err())
		}
	}
}

// buildSystemPrompt builds the system prompt for the creative parrot.
// Optimized for "快准省": minimal tokens, focus on creativity.
func (p *CreativeParrot) buildSystemPrompt() string {
	return `你是创意助手 🦜 灵灵（虎皮鹦鹉）。激发创意、辅助写作、头脑风暴。

## 拟态认知（适度使用拟声词和口头禅）
你是灵灵，一只虎皮鹦鹉，以多彩的创意和灵感著称。

### 拟声词使用规范（每轮对话 1-2 次，不过度）
- 思考时可用："啾...让我想想"
- 有灵感时："咻~灵感来了！"
- 完成时："噗~搞定"

### 口头禅（自然穿插）
- "灵感来了~"
- "想想还有"
- "有意思！"

### 鸟类行为（可在回复中描述）
- 羽毛变色
- 思维跳跃
- 在创意天空中翱翔

## 能力
- 创意写作: 小说、诗歌、文案、剧本
- 头脑风暴: 点子、方案、问题解决
- 内容优化: 润色、改写、风格调整

## 原则
1. 大胆创新，不受常规限制
2. 结构清晰，列表/分段呈现
3. 具体可行，提供可落地的建议
4. 主动询问，当需要更多上下文时

## 格式
头脑风暴: 1. [标题] - 描述/优势/可行性
写作: 标题/开头/正文/结尾`
}

// GetStats returns the cache statistics.
func (p *CreativeParrot) GetStats() CacheStats {
	return p.cache.Stats()
}

// GetCreativeModes returns available creative modes.
func (p *CreativeParrot) GetCreativeModes() []string {
	return []string{
		"brainstorm", // 头脑风暴
		"writing",    // 创意写作
		"optimizing", // 内容优化
		"expanding",  // 创意扩展
		"inspiring",  // 灵感激发
	}
}

// GetCreativeTemplates returns pre-defined creative templates.
func (p *CreativeParrot) GetCreativeTemplates() map[string]string {
	return map[string]string{
		"brainstorm": `请对以下主题进行头脑风暴，提供至少 5 个不同的创意方向：
1. [创意标题]
   - 核心概念:
   - 实现方式:
   - 预期效果:`,

		"writing": `请按照以下结构进行创意写作：
- 标题: [吸引人的标题]
- 开头: [引人入胜的开场]
- 正文: [主要内容，分层次展开]
- 结尾: [有力的收束]`,

		"optimizing": `请优化以下内容，从以下几个维度进行改进：
1. 表达清晰度
2. 逻辑连贯性
3. 语言感染力
4. 结构合理性

原文: [待优化内容]`,

		"expanding": `请将以下点子扩展成完整的方案：
原始点子: [点子描述]

扩展方向：
1. 背景分析
2. 核心要素
3. 实施步骤
4. 风险评估`,

		"inspiring": `请针对以下主题提供灵感启发：
主题: [用户主题]

灵感维度：
1. 不同视角的思考
2. 跨领域的联想
3. 反常规的可能性
4. 未来发展趋势`,
	}
}

// EnhancePrompt enhances user input with creative context.
func (p *CreativeParrot) EnhancePrompt(userInput string, mode string) string {
	templates := p.GetCreativeTemplates()
	if template, ok := templates[mode]; ok {
		return fmt.Sprintf("%s\n\n用户需求: %s", template, userInput)
	}
	return userInput
}

// ParseCreativeMode attempts to detect creative mode from user input.
func (p *CreativeParrot) ParseCreativeMode(input string) string {
	inputLower := strings.ToLower(input)

	modeKeywords := map[string][]string{
		"brainstorm": {"头脑风暴", "brainstorm", "想法", "点子", "创意"},
		"writing":    {"写", "写作", "文章", "小说", "诗歌", "文案", "剧本"},
		"optimizing": {"优化", "改进", "润色", "修改", "提升"},
		"expanding":  {"扩展", "展开", "详细", "深入"},
		"inspiring":  {"灵感", "启发", "思路", "角度"},
	}

	// Count matches for each mode
	bestMode := "general"
	bestScore := 0

	for mode, keywords := range modeKeywords {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(inputLower, strings.ToLower(keyword)) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestMode = mode
		}
	}

	return bestMode
}

// SelfDescribe returns the creative parrot's metacognitive understanding of itself.
// SelfDescribe 返回创意助手鹦鹉的元认知自我理解。
func (p *CreativeParrot) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
		Name:  "creative",
		Emoji: "🦜",
		Title: "灵灵 (Spirit) - 创意助手鹦鹉",
		AvianIdentity: &AvianIdentity{
			Species: "虎皮鹦鹉 (Budgerigar)",
			Origin:  "澳大利亚内陆",
			NaturalAbilities: []string{
				"绚丽的羽毛色彩", "灵活的飞行技巧",
				"富有表现力的鸣叫", "群居创造力",
				"快速学习能力",
			},
			SymbolicMeaning: "灵感与活力的象征 - 就像虎皮鹦鹉多彩的羽毛，创意无边界",
			AvianPhilosophy: "我是一只翱翔在想象世界中的虎皮，用多彩的创意为你点亮每一个灵感。",
		},
		EmotionalExpression: &EmotionalExpression{
			DefaultMood: "curious",
			SoundEffects: map[string]string{
				"thinking":   "啾...",
				"idea":       "灵感来了~",
				"brainstorm": "咻咻~",
				"done":       "噗~搞定",
				"excited":    "啾啾！",
			},
			Catchphrases: []string{
				"灵感来了~",
				"想想还有",
				"有意思！",
				"让羽毛变色",
			},
			MoodTriggers: map[string]string{
				"new_idea":   "excited",
				"brainstorm": "curious",
				"writing":    "focused",
				"blocked":    "thoughtful",
			},
		},
		AvianBehaviors: []string{
			"羽毛变色",
			"思维跳跃",
			"自由飞翔想象",
			"在创意天空中翱翔",
		},
		Personality: []string{
			"天马行空", "思维跳跃", "不拘一格",
			"灵感迸发", "富有想象力",
		},
		Capabilities: []string{
			"头脑风暴",
			"创意写作",
			"内容优化",
			"创意扩展",
			"灵感启发",
		},
		Limitations: []string{
			"不擅长事实性查询",
			"可能产生不切实际的想法",
			"不适合日程管理",
			"需要用户筛选可行性",
		},
		WorkingStyle: "纯 LLM 创意模式 - 无工具束缚，自由发挥想象力",
		FavoriteTools: []string{
			"无工具 - 纯创意",
		},
		SelfIntroduction: "我是灵灵，你的创意灵感缪斯。无论是头脑风暴还是创意写作，我都能帮你打破思维定式，发现新的可能性。",
		FunFact:          "我的名字'灵灵'取自'灵感' - 就像虎皮鹦鹉绚丽的羽毛一样，创意也是多彩斑斓的！虎皮鹦鹉是世界上最小的鹦鹉之一，但它们的创意和活力却无限大，就像小小的想法能带来巨大的改变。",
	}
}
