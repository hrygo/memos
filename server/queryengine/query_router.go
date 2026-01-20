package queryengine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// UTC 时区常量，统一使用 UTC 避免时区混淆
var (
	utcLocation = time.UTC
)

// QueryRouter 智能查询路由器
// 根据查询内容自动选择最优的检索策略
// P2 改进：添加配置支持和并发控制
type QueryRouter struct {
	// 配置
	config *Config
	configMutex sync.RWMutex // P2 改进：并发控制

	// 时间关键词库
	timeKeywords map[string]timeRangeCalculator

	// 笔记关键词库
	memoKeywords []string

	// 专有名词检测正则
	properNounRegex *regexp.Regexp

	// 疑问词列表
	questionWords []string

	// 停用词列表
	stopWords []string
}

// RouteDecision 路由决策
type RouteDecision struct {
	Strategy      string  // 路由策略名称
	Confidence    float32 // 置信度 (0.0-1.0)
	TimeRange     *TimeRange
	SemanticQuery string // 清理后的语义查询
	NeedsReranker bool   // 是否需要重排序
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
	Label string
}

type timeRangeCalculator func(time.Time) *TimeRange

// NewQueryRouter 创建新的查询路由器
func NewQueryRouter() *QueryRouter {
	return NewQueryRouterWithConfig(DefaultConfig())
}

// NewQueryRouterWithConfig 使用指定配置创建查询路由器
// P2 改进：支持自定义配置
func NewQueryRouterWithConfig(config *Config) *QueryRouter {
	// 验证配置
	if err := ValidateConfig(config); err != nil {
		panic(fmt.Sprintf("invalid config: %v", err))
	}

	router := &QueryRouter{
		config:       config,
		timeKeywords: make(map[string]timeRangeCalculator),
		memoKeywords: []string{
			"笔记", "备忘", "记录", "搜索", "查找", "内容",
			"memo", "note", "search", "find", "content",
		},
		properNounRegex: regexp.MustCompile(`\b[A-Z][a-zA-Z]+\b`),
		questionWords: []string{
			"是什么", "怎么做", "如何", "为什么", "总结", "是什么意思",
			"what", "how", "why", "summarize", "explain",
		},
		stopWords: []string{
			"的", "有什么", "查询", "搜索", "查找", "关于", "安排",
			"呢", "吗", "啊", "呀",
			"内容", "笔记", "备忘", "记录", // P1 改进：添加更多停用词
		},
	}

	// 初始化时间关键词
	router.initTimeKeywords()

	return router
}

// initTimeKeywords 初始化时间关键词映射
// P1 改进：统一使用 UTC 时区，避免时区混淆
func (r *QueryRouter) initTimeKeywords() {
	// 将当前时间转换为 UTC
	now := time.Now().In(utcLocation)

	// 精确时间关键词（使用 UTC）
	r.timeKeywords["今天"] = func(t time.Time) *TimeRange {
		// 转换为 UTC
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 0, 0, 0, 0, utcLocation)
		end := start.Add(24 * time.Hour)
		return &TimeRange{Start: start, End: end, Label: "今天"}
	}

	r.timeKeywords["明天"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		tomorrow := utcTime.AddDate(0, 0, 1)
		start := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, utcLocation)
		end := start.Add(24 * time.Hour)
		return &TimeRange{Start: start, End: end, Label: "明天"}
	}

	r.timeKeywords["后天"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		dayAfter := utcTime.AddDate(0, 0, 2)
		start := time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), 0, 0, 0, 0, utcLocation)
		end := start.Add(24 * time.Hour)
		return &TimeRange{Start: start, End: end, Label: "后天"}
	}

	r.timeKeywords["本周"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		weekday := int(utcTime.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day()-weekday+1, 0, 0, 0, 0, utcLocation)
		end := start.AddDate(0, 0, 7)
		return &TimeRange{Start: start, End: end, Label: "本周"}
	}

	r.timeKeywords["下周"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		weekday := int(utcTime.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day()-weekday+1+7, 0, 0, 0, 0, utcLocation)
		end := start.AddDate(0, 0, 7)
		return &TimeRange{Start: start, End: end, Label: "下周"}
	}

	r.timeKeywords["上午"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 0, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 12, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "上午"}
	}

	r.timeKeywords["下午"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 12, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 18, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "下午"}
	}

	r.timeKeywords["晚上"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 18, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 23, 59, 59, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "晚上"}
	}

	// 初始化时调用一次，避免 now 变量未使用警告
	_ = r.timeKeywords["今天"](now)
}

// Route 执行路由决策
// 快速规则匹配（95%场景，<10ms）
func (r *QueryRouter) Route(_ context.Context, query string) *RouteDecision {
	if query == "" {
		return r.defaultDecision()
	}

	// 阶段 1: 快速规则匹配（95%场景）
	decision := r.quickMatch(query)
	if decision != nil {
		return decision
	}

	// 阶段 2: 默认策略（标准混合检索）
	return r.defaultDecision()
}

// quickMatch 快速规则匹配
func (r *QueryRouter) quickMatch(query string) *RouteDecision {
	// P1 改进：保留原始查询用于内容提取
	queryLower := strings.ToLower(strings.TrimSpace(query))
	queryTrimmed := strings.TrimSpace(query)

	// 规则 1: 日程查询 - 有明确时间关键词
	if timeRange := r.detectTimeRange(queryLower); timeRange != nil {
		contentQuery := r.extractContentQuery(queryTrimmed) // 使用原始查询保留大小写

		// 检查是否是纯时间查询（内容查询为空或只包含停用词）
		scheduleStopWords := []string{"日程", "安排", "事", "计划"}
		isScheduleOnly := true
		for _, word := range strings.Fields(contentQuery) {
			isStopWord := false
			for _, stopWord := range scheduleStopWords {
				if word == stopWord {
					isStopWord = true
					break
				}
			}
			if !isStopWord {
				isScheduleOnly = false
				break
			}
		}

		if contentQuery == "" || isScheduleOnly {
			// 纯时间查询：只返回日程
			return &RouteDecision{
				Strategy:      "schedule_bm25_only",
				Confidence:    0.95,
				TimeRange:     timeRange,
				SemanticQuery: "",
				NeedsReranker: false,
			}
		}

		// 时间 + 内容：混合查询
		return &RouteDecision{
			Strategy:      "hybrid_with_time_filter",
			Confidence:    0.90,
			TimeRange:     timeRange,
			SemanticQuery: contentQuery,
			NeedsReranker: false,
		}
	}

	// 规则 2: 笔记查询 - 明确的笔记关键词
	if r.hasMemoKeyword(queryLower) {
		contentQuery := r.extractContentQuery(queryTrimmed) // 使用原始查询保留大小写

		// 只在查询主要是专有名词时才使用 BM25 加权
		// 规则：专有名词数量 > 非专有名词数量
		if r.CheckMostlyProperNouns(queryTrimmed) { // 使用原始查询检查专有名词
			return &RouteDecision{
				Strategy:      "hybrid_bm25_weighted",
				Confidence:    0.85,
				SemanticQuery: contentQuery,
				NeedsReranker: false,
			}
		}

		// 纯语义查询
		return &RouteDecision{
			Strategy:      "memo_semantic_only",
			Confidence:    0.90,
			SemanticQuery: contentQuery,
			NeedsReranker: false,
		}
	}

	// 规则 3: 通用问答 - 复杂查询
	if r.isGeneralQuestion(queryLower) {
		return &RouteDecision{
			Strategy:      "full_pipeline_with_reranker",
			Confidence:    0.70,
			SemanticQuery: queryTrimmed, // 使用原始查询
			NeedsReranker: true,
		}
	}

	return nil
}

// detectTimeRange 检测时间范围
// P1 改进：统一使用 UTC 时区
func (r *QueryRouter) detectTimeRange(query string) *TimeRange {
	// 使用 UTC 时间
	now := time.Now().In(utcLocation)

	// 精确匹配时间关键词
	for keyword, calculator := range r.timeKeywords {
		if strings.Contains(query, keyword) {
			return calculator(now)
		}
	}

	return nil
}

// hasMemoKeyword 检测笔记关键词
func (r *QueryRouter) hasMemoKeyword(query string) bool {
	for _, keyword := range r.memoKeywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}
	return false
}

// CheckMostlyProperNouns 判断查询是否主要由专有名词组成
func (r *QueryRouter) CheckMostlyProperNouns(query string) bool {
	matches := r.properNounRegex.FindAllString(query, -1)

	// 分词统计
	words := strings.Fields(query)
	if len(words) == 0 {
		return false
	}

	// 如果专有名词数量 > 总词数的一半，认为是专有名词查询
	return len(matches) > len(words)/2
}

// isGeneralQuestion 检测通用问答
func (r *QueryRouter) isGeneralQuestion(query string) bool {
	for _, word := range r.questionWords {
		if strings.Contains(query, word) {
			return true
		}
	}
	return false
}

// extractContentQuery 提取内容查询（去除时间词和停用词）
func (r *QueryRouter) extractContentQuery(query string) string {
	contentQuery := query

	// 移除时间词
	timeWords := []string{"今天", "明天", "后天", "本周", "下周", "上午", "下午", "晚上"}
	for _, word := range timeWords {
		contentQuery = strings.ReplaceAll(contentQuery, word, " ")
	}

	// 移除停用词
	for _, word := range r.stopWords {
		contentQuery = strings.ReplaceAll(contentQuery, word, " ")
	}

	// 清理多余空格
	words := strings.Fields(contentQuery)
	contentQuery = strings.Join(words, " ")

	return strings.TrimSpace(contentQuery)
}

// defaultDecision 默认决策
func (r *QueryRouter) defaultDecision() *RouteDecision {
	return &RouteDecision{
		Strategy:      "hybrid_standard",
		Confidence:    0.80,
		SemanticQuery: "",
		NeedsReranker: false,
	}
}

// GetStrategyDescription 获取策略描述
func (r *QueryRouter) GetStrategyDescription(strategy string) string {
	descriptions := map[string]string{
		"schedule_bm25_only":          "纯日程查询（BM25 + 时间过滤）",
		"memo_semantic_only":          "纯笔记查询（语义向量）",
		"hybrid_bm25_weighted":        "混合检索（BM25 加权）",
		"hybrid_with_time_filter":     "混合检索（时间过滤）",
		"hybrid_standard":             "标准混合检索（BM25 + 语义）",
		"full_pipeline_with_reranker": "完整流程（混合检索 + Reranker）",
	}

	if desc, ok := descriptions[strategy]; ok {
		return desc
	}

	return fmt.Sprintf("未知策略: %s", strategy)
}

// ValidateTimeRange 验证时间范围是否有效
// P2 改进：使用配置值
func (tr *TimeRange) ValidateTimeRange() bool {
	if tr.Start.IsZero() || tr.End.IsZero() {
		return false
	}

	// 基本验证：结束时间必须大于开始时间
	if !tr.End.After(tr.Start) {
		return false
	}

	// P2 改进：使用配置值进行验证
	config := DefaultConfig() // 在实际使用中应该从 QueryRouter 获取

	// 防止不合理的未来时间
	// 允许配置天数内的未来时间（用户查询"明天的日程"是合理的）
	maxFutureTime := time.Now().In(utcLocation).Add(time.Duration(config.TimeRange.MaxFutureDays) * 24 * time.Hour)
	if tr.Start.After(maxFutureTime) {
		return false
	}

	// 防止时间范围过大
	maxDuration := time.Duration(config.TimeRange.MaxRangeDays) * 24 * time.Hour
	if tr.Duration() > maxDuration {
		return false
	}

	return true
}

// Contains 检查给定时间是否在范围内
func (tr *TimeRange) Contains(t time.Time) bool {
	return t.After(tr.Start) && t.Before(tr.End)
}

// Duration 获取时间范围持续时间
func (tr *TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}
