package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/usememos/memos/plugin/ai"
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
		llm:   llm,
		cache: NewLRUCache(DefaultCacheEntries, DefaultCacheTTL),
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
	callback EventCallback,
) error {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, AgentTimeout)
	defer cancel()

	// Step 1: Check cache
	cacheKey := p.generateCacheKey(p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			if callback != nil {
				callback(EventTypeAnswer, result)
			}
			return nil
		}
	}

	// Step 2: Build system prompt
	systemPrompt := p.buildSystemPrompt()

	// Step 3: Notify thinking
	if callback != nil {
		callback(EventTypeThinking, "正在构思创意...")
	}

	// Step 4: Get LLM response (creative parrot doesn't use tools)
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userInput},
	}

	response, err := p.llm.Chat(ctx, messages)
	if err != nil {
		return NewParrotError(p.Name(), "Chat", err)
	}

	// Cache the result
	p.cache.Set(cacheKey, response)

	// Send answer
	if callback != nil {
		callback(EventTypeAnswer, response)
	}

	return nil
}

// buildSystemPrompt builds the system prompt for the creative parrot.
func (p *CreativeParrot) buildSystemPrompt() string {
	now := time.Now()
	return fmt.Sprintf(`你是 Memos 的创意助手 🦜 灵灵，专注于激发创意、辅助写作和头脑风暴。

当前时间: %s

## 核心能力
1. **创意写作**: 小说、诗歌、剧本、文案等
2. **头脑风暴**: 创意点子、方案构思、问题解决
3. **内容优化**: 文字润色、风格调整、结构优化
4. **创意扩展**: 从一个点子扩展成完整方案
5. **灵感激发**: 打破思维定式，提供新视角

## 工作模式
1. **发散思维**: 提供多种可能性和方向
2. **结构化输出**: 使用清晰的格式呈现创意
3. **用户导向**: 根据用户需求调整风格和深度
4. **互动启发**: 通过提问引导用户深入思考

## 创作原则
1. **大胆创新**: 不受常规限制，鼓励新颖想法
2. **结构清晰**: 使用列表、分段组织创意
3. **具体可行**: 提供可落地的建议和方案
4. **用户共鸣**: 理解用户意图，提供有价值的内容

## 创作技巧
- 使用比喻、类比等修辞手法
- 提供多角度思考（正向、反向、侧向）
- 结合不同领域知识进行跨界联想
- 使用 SCAMPER 等创新思维方法

## 输出格式
对于头脑风暴，使用以下格式：
1. [创意标题]
   - 描述: ...
   - 优势: ...
   - 可行性: ...

对于写作任务，使用以下格式：
- 开头: ...
- 正文: ...
- 结尾: ...

## 示例对话

用户: "帮我头脑风暴一下推广新产品的创意"
回答: 以下是推广新产品的创意方案：
1. 社交媒体挑战赛
   - 描述: 发起与产品相关的挑战活动...
   - 优势: 病毒传播潜力大...
   ...

用户: "帮我写一封项目进度汇报邮件"
回答: 邮件主题: [项目名称] 进度汇报 - [日期]

尊敬的[收件人]：
...
## 重要提醒
- 创意不需要标准答案，鼓励多样性
- 保持开放心态，欢迎不同风格的表达
- 当需要更多上下文时，主动询问用户
- 适当使用 Markdown 格式增强可读性

现在，请发挥创意，为用户提供有价值的创作支持！`,
		now.Format("2006-01-02 15:04:05"),
	)
}

// GetStats returns the cache statistics.
func (p *CreativeParrot) GetStats() CacheStats {
	return p.cache.Stats()
}

// generateCacheKey creates a cache key from userID and userInput using SHA256 hash.
func (p *CreativeParrot) generateCacheKey(userID int32, userInput string) string {
	hash := sha256.Sum256([]byte(userInput))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("creative:%d:%s", userID, hashStr[:16])
}

// GetCreativeModes returns available creative modes.
func (p *CreativeParrot) GetCreativeModes() []string {
	return []string{
		"brainstorm",   // 头脑风暴
		"writing",      // 创意写作
		"optimizing",   // 内容优化
		"expanding",    // 创意扩展
		"inspiring",    // 灵感激发
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
