package queryengine

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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
	config      *Config
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

// ScheduleQueryMode 日程查询模式
type ScheduleQueryMode int32

const (
	AutoQueryMode     ScheduleQueryMode = 0 // 自动选择
	StandardQueryMode ScheduleQueryMode = 1 // 标准模式：返回范围内有任何部分的日程
	StrictQueryMode   ScheduleQueryMode = 2 // 严格模式：只返回完全在范围内的日程
)

// RouteDecision 路由决策
type RouteDecision struct {
	Strategy          string  // 路由策略名称
	Confidence        float32 // 置信度 (0.0-1.0)
	TimeRange         *TimeRange
	SemanticQuery     string            // 清理后的语义查询
	NeedsReranker     bool              // 是否需要重排序
	ScheduleQueryMode ScheduleQueryMode // 日程查询模式（P1 新增）
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
// 举一反三优化：系统性扩展时间关键词库，覆盖所有常见时间表达
func (r *QueryRouter) initTimeKeywords() {
	// 将当前时间转换为 UTC
	now := time.Now().In(utcLocation)

	// ============================================================
	// 1. 精确日期关键词
	// ============================================================
	r.timeKeywords["今天"] = func(t time.Time) *TimeRange {
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

	// 举一反三：添加过去的日期
	r.timeKeywords["昨天"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		yesterday := utcTime.AddDate(0, 0, -1)
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, utcLocation)
		end := start.Add(24 * time.Hour)
		return &TimeRange{Start: start, End: end, Label: "昨天"}
	}

	r.timeKeywords["前天"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		dayBefore := utcTime.AddDate(0, 0, -2)
		start := time.Date(dayBefore.Year(), dayBefore.Month(), dayBefore.Day(), 0, 0, 0, 0, utcLocation)
		end := start.Add(24 * time.Hour)
		return &TimeRange{Start: start, End: end, Label: "前天"}
	}

	// 举一反三：添加更远的日期
	r.timeKeywords["大后天"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		dayAfter := utcTime.AddDate(0, 0, 3)
		start := time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), 0, 0, 0, 0, utcLocation)
		end := start.Add(24 * time.Hour)
		return &TimeRange{Start: start, End: end, Label: "大后天"}
	}

	// P1 新增：更远的年份关键词
	r.timeKeywords["后年"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		targetYear := utcTime.Year() + 2
		start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, utcLocation)
		end := time.Date(targetYear+1, 1, 1, 0, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "后年"}
	}

	r.timeKeywords["大后年"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		targetYear := utcTime.Year() + 3
		start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, utcLocation)
		end := time.Date(targetYear+1, 1, 1, 0, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "大后年"}
	}

	r.timeKeywords["前年"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		targetYear := utcTime.Year() - 2
		start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, utcLocation)
		end := time.Date(targetYear+1, 1, 1, 0, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "前年"}
	}

	r.timeKeywords["大前年"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		targetYear := utcTime.Year() - 3
		start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, utcLocation)
		end := time.Date(targetYear+1, 1, 1, 0, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "大前年"}
	}

	// ============================================================
	// 同义词映射（举一反三优化：覆盖文言表达）
	// ============================================================
	// 精确日期同义词
	r.timeKeywords["今日"] = r.timeKeywords["今天"]
	r.timeKeywords["明日"] = r.timeKeywords["明天"]
	r.timeKeywords["后日"] = r.timeKeywords["后天"]
	r.timeKeywords["昨日"] = r.timeKeywords["昨天"]
	r.timeKeywords["前日"] = r.timeKeywords["前天"]

	// ============================================================
	// 2. 周相关关键词
	// ============================================================
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

	// 举一反三：添加上周
	r.timeKeywords["上周"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		weekday := int(utcTime.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day()-weekday+1-7, 0, 0, 0, 0, utcLocation)
		end := start.AddDate(0, 0, 7)
		return &TimeRange{Start: start, End: end, Label: "上周"}
	}

	// 举一反三：添加这周（同义词）
	r.timeKeywords["这周"] = r.timeKeywords["本周"]

	// ============================================================
	// 3. 星期关键词
	// ============================================================
	weekdayMap := map[string]time.Weekday{
		"周一":  time.Monday,
		"周二":  time.Tuesday,
		"周三":  time.Wednesday,
		"周四":  time.Thursday,
		"周五":  time.Friday,
		"周六":  time.Saturday,
		"周日":  time.Sunday,
		"星期一": time.Monday,
		"星期二": time.Tuesday,
		"星期三": time.Wednesday,
		"星期四": time.Thursday,
		"星期五": time.Friday,
		"星期六": time.Saturday,
		"星期日": time.Sunday,
	}

	for name, targetWeekday := range weekdayMap {
		r.timeKeywords[name] = func(targetWD time.Weekday) func(time.Time) *TimeRange {
			return func(t time.Time) *TimeRange {
				utcTime := t.In(utcLocation)
				currentWeekday := int(utcTime.Weekday())
				if currentWeekday == 0 {
					currentWeekday = 7
				}
				targetWDInt := int(targetWD)
				if targetWD == 0 {
					targetWDInt = 7
				}

				daysUntil := (targetWDInt - currentWeekday + 7) % 7
				if daysUntil == 0 && utcTime.Hour() > 12 {
					// 如果今天已经是目标星期且已过中午，查询下周的
					daysUntil = 7
				}

				targetDate := utcTime.AddDate(0, 0, daysUntil)
				start := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, utcLocation)
				end := start.Add(24 * time.Hour)
				return &TimeRange{Start: start, End: end, Label: name}
			}
		}(targetWeekday)
	}

	// ============================================================
	// 4. 时段关键词
	// ============================================================
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

	// 举一反三：添加更多时段
	r.timeKeywords["早上"] = r.timeKeywords["上午"]
	r.timeKeywords["中午"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 11, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 13, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "中午"}
	}
	r.timeKeywords["凌晨"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 0, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 6, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "凌晨"}
	}

	// ============================================================
	// 5. 模糊时间关键词（举一反三优化重点）
	// ============================================================
	// 5.1 短期模糊时间（7天内）
	r.timeKeywords["近期"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 0, 0, 0, 0, utcLocation)
		end := start.AddDate(0, 0, 7) // 近期 = 7天
		return &TimeRange{Start: start, End: end, Label: "近期"}
	}

	r.timeKeywords["最近"] = r.timeKeywords["近期"]
	r.timeKeywords["这几天"] = r.timeKeywords["近期"]

	// 5.2 中期模糊时间（30天内）
	r.timeKeywords["这个月"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), 1, 0, 0, 0, 0, utcLocation)
		end := start.AddDate(0, 1, 0)
		return &TimeRange{Start: start, End: end, Label: "这个月"}
	}

	r.timeKeywords["本月"] = r.timeKeywords["这个月"]
	r.timeKeywords["这月"] = r.timeKeywords["这个月"]
	r.timeKeywords["月内"] = r.timeKeywords["这个月"]

	// 5.3 未来月份
	r.timeKeywords["下个月"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), 1, 0, 0, 0, 0, utcLocation).AddDate(0, 1, 0)
		end := start.AddDate(0, 1, 0)
		return &TimeRange{Start: start, End: end, Label: "下个月"}
	}

	r.timeKeywords["下月"] = r.timeKeywords["下个月"]

	// 5.4 过去月份
	r.timeKeywords["上个月"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), utcTime.Month(), 1, 0, 0, 0, 0, utcLocation).AddDate(0, -1, 0)
		end := start.AddDate(0, 1, 0)
		return &TimeRange{Start: start, End: end, Label: "上个月"}
	}

	r.timeKeywords["上月"] = r.timeKeywords["上个月"]

	// ============================================================
	// 6. 年份关键词
	// ============================================================
	r.timeKeywords["今年"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year(), 1, 1, 0, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year()+1, 1, 1, 0, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "今年"}
	}

	r.timeKeywords["明年"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year()+1, 1, 1, 0, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year()+2, 1, 1, 0, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "明年"}
	}

	r.timeKeywords["去年"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start := time.Date(utcTime.Year()-1, 1, 1, 0, 0, 0, 0, utcLocation)
		end := time.Date(utcTime.Year(), 1, 1, 0, 0, 0, 0, utcLocation)
		return &TimeRange{Start: start, End: end, Label: "去年"}
	}

	// ============================================================
	// 7. 季度关键词
	// ============================================================
	// 获取季度的辅助函数
	getQuarterRange := func(year int, quarter int) (time.Time, time.Time) {
		startMonth := time.Month((quarter-1)*3 + 1)
		start := time.Date(year, startMonth, 1, 0, 0, 0, 0, utcLocation)
		end := start.AddDate(0, 3, 0)
		return start, end
	}

	r.timeKeywords["一季度"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start, end := getQuarterRange(utcTime.Year(), 1)
		return &TimeRange{Start: start, End: end, Label: "一季度"}
	}
	r.timeKeywords["第一季度"] = r.timeKeywords["一季度"]

	r.timeKeywords["二季度"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start, end := getQuarterRange(utcTime.Year(), 2)
		return &TimeRange{Start: start, End: end, Label: "二季度"}
	}
	r.timeKeywords["第二季度"] = r.timeKeywords["二季度"]

	r.timeKeywords["三季度"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start, end := getQuarterRange(utcTime.Year(), 3)
		return &TimeRange{Start: start, End: end, Label: "三季度"}
	}
	r.timeKeywords["第三季度"] = r.timeKeywords["三季度"]

	r.timeKeywords["四季度"] = func(t time.Time) *TimeRange {
		utcTime := t.In(utcLocation)
		start, end := getQuarterRange(utcTime.Year(), 4)
		return &TimeRange{Start: start, End: end, Label: "四季度"}
	}
	r.timeKeywords["第四季度"] = r.timeKeywords["四季度"]

	// 初始化时调用一次，避免 now 变量未使用警告
	_ = r.timeKeywords["今天"](now)
}

// Route executes routing decision with user timezone support.
// Fast rule matching (95% scenarios, <10ms).
// If userTimezone is nil, defaults to UTC.
func (r *QueryRouter) Route(_ context.Context, query string, userTimezone *time.Location) *RouteDecision {
	if query == "" {
		return r.defaultDecision()
	}

	// 阶段 1: 快速规则匹配（95%场景）
	decision := r.quickMatchWithTimezone(query, userTimezone)
	if decision != nil {
		return decision
	}

	// 阶段 2: 默认策略（标准混合检索）
	return r.defaultDecision()
}

// quickMatchWithTimezone 快速规则匹配（带时区支持）
func (r *QueryRouter) quickMatchWithTimezone(query string, userTimezone *time.Location) *RouteDecision {
	// P1 改进：保留原始查询用于内容提取
	queryLower := strings.ToLower(strings.TrimSpace(query))
	queryTrimmed := strings.TrimSpace(query)

	// 规则 1: 日程查询 - 有明确时间关键词
	if timeRange := r.detectTimeRangeWithTimezone(queryLower, userTimezone); timeRange != nil {
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
				Strategy:          "schedule_bm25_only",
				Confidence:        0.95,
				TimeRange:         timeRange,
				SemanticQuery:     "",
				NeedsReranker:     false,
				ScheduleQueryMode: r.determineScheduleQueryMode(queryTrimmed, timeRange),
			}
		}

		// 时间 + 内容：混合查询
		return &RouteDecision{
			Strategy:          "hybrid_with_time_filter",
			Confidence:        0.90,
			TimeRange:         timeRange,
			SemanticQuery:     contentQuery,
			NeedsReranker:     false,
			ScheduleQueryMode: r.determineScheduleQueryMode(queryTrimmed, timeRange),
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
// P2 改进：支持具体日期解析（"1月21日"、"1-21"等）
func (r *QueryRouter) detectTimeRange(query string) *TimeRange {
	// 使用 UTC 时间
	now := time.Now().In(utcLocation)

	// ============================================================
	// 1. 精确匹配时间关键词（相对时间）
	// ============================================================
	// 修复：优先匹配最长关键词，避免"大后天"匹配到"后天"
	var matchedKeyword string
	var matchedCalculator timeRangeCalculator
	for keyword, calculator := range r.timeKeywords {
		if strings.Contains(query, keyword) {
			// 选择最长的匹配关键词
			if len(keyword) > len(matchedKeyword) {
				matchedKeyword = keyword
				matchedCalculator = calculator
			}
		}
	}
	if matchedCalculator != nil {
		return matchedCalculator(now)
	}

	// ============================================================
	// 2. 解析具体日期（新增：P0 紧急修复）
	// ============================================================
	// 支持的格式：
	// - "1月21日"、"01月21日"、"1月21号"
	// - "1-21"、"01-21"、"1/21"、"01/21"
	// - "1-21"、"1-21日"

	// 匹配 "1月21日" 或 "1月21号" 或 "01月21日"
	monthDayRegex := regexp.MustCompile(`(\d{1,2})月(\d{1,2})[日号]`)
	if matches := monthDayRegex.FindStringSubmatch(query); len(matches) >= 3 {
		month, err1 := strconv.Atoi(matches[1])
		day, err2 := strconv.Atoi(matches[2])
		if err1 == nil && err2 == nil && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			// 构造日期（当年）
			year := now.Year()
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, utcLocation)
			end := start.Add(24 * time.Hour)

			// 如果日期在过去，尝试明年
			if end.Before(now) && !start.After(now) {
				// 检查是否是去年的今天（比如12月31日，现在是1月1日）
				// 或者是今年的未来日期
				if start.AddDate(0, 0, 1).Before(now) {
					// 日期在过去，使用明年
					start = time.Date(year+1, time.Month(month), day, 0, 0, 0, 0, utcLocation)
					end = start.Add(24 * time.Hour)
				}
			}

			label := fmt.Sprintf("%d月%d日", month, day)
			return &TimeRange{Start: start, End: end, Label: label}
		}
	}

	// 匹配 "1-21"、"01-21"、"1/21"、"01/21"
	slashDayRegex := regexp.MustCompile(`(\d{1,2})[-/](\d{1,2})`)
	if matches := slashDayRegex.FindStringSubmatch(query); len(matches) >= 3 {
		month, err1 := strconv.Atoi(matches[1])
		day, err2 := strconv.Atoi(matches[2])
		if err1 == nil && err2 == nil && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			// 构造日期（当年）
			year := now.Year()
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, utcLocation)
			end := start.Add(24 * time.Hour)

			// 如果日期在过去，尝试明年
			if end.Before(now) && !start.After(now) {
				if start.AddDate(0, 0, 1).Before(now) {
					start = time.Date(year+1, time.Month(month), day, 0, 0, 0, 0, utcLocation)
					end = start.Add(24 * time.Hour)
				}
			}

			label := fmt.Sprintf("%d月%d日", month, day)
			return &TimeRange{Start: start, End: end, Label: label}
		}
	}

	return nil
}

// determineScheduleQueryMode 确定日程查询模式（P1 新增）
// 自动选择规则：
// - 相对时间（今天、明天、本周）→ 标准模式
// - 绝对时间（1月21日、2025-01-21）→ 严格模式
func (r *QueryRouter) determineScheduleQueryMode(query string, timeRange *TimeRange) ScheduleQueryMode {
	if timeRange == nil {
		return StandardQueryMode // 默认标准模式
	}

	// 检查是否为相对时间关键词
	relativeTimeKeywords := []string{
		"今天", "明天", "后天", "大后天", "昨天", "前天",
		"今日", "明日", "后日", "昨日", "前日",
		"本周", "这周", "下周", "上周",
		"本月", "这月", "这个月", "下月", "上月", "月内",
		"今年", "明年", "去年",
		"近期", "最近", "这几天", "近几天",
		"上午", "下午", "晚上", "中午", "凌晨", "早上",
		"周一", "周二", "周三", "周四", "周五", "周六", "周日",
		"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日",
		"一季度", "二季度", "三季度", "四季度",
		"第一季度", "第二季度", "第三季度", "第四季度",
	}

	label := timeRange.Label
	for _, keyword := range relativeTimeKeywords {
		if strings.Contains(label, keyword) {
			return StandardQueryMode // 相对时间用标准模式
		}
	}

	// 绝对时间用严格模式
	return StrictQueryMode
}

// detectTimeRangeWithTimezone 检测时间范围（带时区支持）
// P2 改进：使用用户时区而非 UTC
func (r *QueryRouter) detectTimeRangeWithTimezone(query string, userTimezone *time.Location) *TimeRange {
	// 使用用户时区，如果为 nil 则使用 UTC
	if userTimezone == nil {
		userTimezone = utcLocation
	}
	now := time.Now().In(userTimezone)

	// ============================================================
	// 1. 精确匹配时间关键词（相对时间）
	// ============================================================
	// 修复：优先匹配最长关键词，避免"大后天"匹配到"后天"
	var matchedKeyword string
	var matchedCalculator timeRangeCalculator
	for keyword, calculator := range r.timeKeywords {
		if strings.Contains(query, keyword) {
			// 选择最长的匹配关键词
			if len(keyword) > len(matchedKeyword) {
				matchedKeyword = keyword
				matchedCalculator = calculator
			}
		}
	}
	if matchedCalculator != nil {
		// calculator 仍然使用 UTC（因为 timeKeywords 是用 UTC 初始化的）
		// 但我们在调用时会传入用户时区的 "now"
		userNow := time.Now().In(userTimezone)
		return matchedCalculator(userNow)
	}

	// ============================================================
	// 2. 明确年份日期（新增：P1 优化）
	// ============================================================
	// 支持的格式：
	// - "YYYY年MM月DD日" 或 "YYYY年M月D日" 或 "YYYY年MM月DD号"
	// - "YYYY-MM-DD" 或 "YYYY-M-D"
	// - "YYYY/MM/DD" 或 "YYYY/M/D"

	// 格式 1: "YYYY年MM月DD日" 或 "YYYY年M月D日"
	yearMonthDayRegex := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})[日号]`)
	if matches := yearMonthDayRegex.FindStringSubmatch(query); len(matches) >= 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])

		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, userTimezone)
			end := start.Add(24 * time.Hour)

			label := fmt.Sprintf("%d年%d月%d日", year, month, day)
			return &TimeRange{Start: start, End: end, Label: label}
		}
	}

	// 格式 2: "YYYY-MM-DD" 或 "YYYY-M-D"
	isoDateRegex := regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)
	if matches := isoDateRegex.FindStringSubmatch(query); len(matches) >= 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])

		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, userTimezone)
			end := start.Add(24 * time.Hour)

			label := fmt.Sprintf("%d-%02d-%02d", year, month, day)
			return &TimeRange{Start: start, End: end, Label: label}
		}
	}

	// 格式 3: "YYYY/MM/DD" 或 "YYYY/M/D"
	slashDateRegex := regexp.MustCompile(`(\d{4})/(\d{1,2})/(\d{1,2})`)
	if matches := slashDateRegex.FindStringSubmatch(query); len(matches) >= 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])

		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, userTimezone)
			end := start.Add(24 * time.Hour)

			label := fmt.Sprintf("%d/%02d/%02d", year, month, day)
			return &TimeRange{Start: start, End: end, Label: label}
		}
	}

	// ============================================================
	// 3. 解析具体日期（P0 紧急修复）
	// ============================================================
	// 支持的格式：
	// - "1月21日"、"01月21日"、"1月21号"
	// - "1-21"、"01-21"、"1/21"、"01/21"
	// - "1-21"、"1-21日"

	// 匹配 "1月21日" 或 "1月21号" 或 "01月21日"
	monthDayRegex := regexp.MustCompile(`(\d{1,2})月(\d{1,2})[日号]`)
	if matches := monthDayRegex.FindStringSubmatch(query); len(matches) >= 3 {
		month, err1 := strconv.Atoi(matches[1])
		day, err2 := strconv.Atoi(matches[2])
		if err1 == nil && err2 == nil && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			// 构造日期（当年）- 使用用户时区
			year := now.Year()
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, userTimezone)
			end := start.Add(24 * time.Hour)

			// 如果日期在过去，尝试明年
			if end.Before(now) && !start.After(now) {
				if start.AddDate(0, 0, 1).Before(now) {
					// 日期在过去，使用明年
					start = time.Date(year+1, time.Month(month), day, 0, 0, 0, 0, userTimezone)
					end = start.Add(24 * time.Hour)
				}
			}

			label := fmt.Sprintf("%d月%d日", month, day)
			return &TimeRange{Start: start, End: end, Label: label}
		}
	}

	// 匹配 "1-21"、"01-21"、"1/21"、"01/21"
	slashDayRegex := regexp.MustCompile(`(\d{1,2})[-/](\d{1,2})`)
	if matches := slashDayRegex.FindStringSubmatch(query); len(matches) >= 3 {
		month, err1 := strconv.Atoi(matches[1])
		day, err2 := strconv.Atoi(matches[2])
		if err1 == nil && err2 == nil && month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			// 构造日期（当年）- 使用用户时区
			year := now.Year()
			start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, userTimezone)
			end := start.Add(24 * time.Hour)

			// 如果日期在过去，尝试明年
			if end.Before(now) && !start.After(now) {
				if start.AddDate(0, 0, 1).Before(now) {
					start = time.Date(year+1, time.Month(month), day, 0, 0, 0, 0, userTimezone)
					end = start.Add(24 * time.Hour)
				}
			}

			label := fmt.Sprintf("%d月%d日", month, day)
			return &TimeRange{Start: start, End: end, Label: label}
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
// 举一反三优化：扩展时间词列表，覆盖所有新增的时间关键词
func (r *QueryRouter) extractContentQuery(query string) string {
	contentQuery := query

	// 移除时间词（全面覆盖所有时间关键词）
	timeWords := []string{
		// 精确日期
		"今天", "明天", "后天", "昨天", "前天", "大后天",
		"今日", "明日", "后日", "昨日", "前日", "大后日",
		// 周相关
		"本周", "下周", "上周", "这周",
		// 星期
		"周一", "周二", "周三", "周四", "周五", "周六", "周日",
		"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日",
		// 时段
		"上午", "下午", "晚上", "早上", "中午", "凌晨",
		// 模糊时间
		"近期", "最近", "这几天",
		"这个月", "本月", "这月", "月内",
		"下个月", "下月", "上个月", "上月",
		// 年份
		"今年", "明年", "去年",
		// 季度
		"一季度", "第一季度", "二季度", "第二季度",
		"三季度", "第三季度", "四季度", "第四季度",
	}

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
	return tr.Duration() <= maxDuration
}

// Contains 检查给定时间是否在范围内
func (tr *TimeRange) Contains(t time.Time) bool {
	return t.After(tr.Start) && t.Before(tr.End)
}

// Duration 获取时间范围持续时间
func (tr *TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}
