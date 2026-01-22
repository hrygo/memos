# 意图检测优化设计规格

## 一、概述

**目标**: 优化查询意图检测和停用词过滤逻辑，提高日程查询和创建意图识别的准确性。

**版本**: v1.0
**状态**: 设计中
**优先级**: P2 (体验改进)

---

## 二、背景与问题

### 当前问题

#### 问题 1: 停用词列表过于简单

**当前实现** (`server/queryengine/query_router.go:86-89`):
```go
stopWords: []string{
    "的", "有什么", "查询", "搜索", "查找", "关于", "安排",
    "呢", "吗", "啊", "呀",
    "内容", "笔记", "备忘", "记录",
}
```

**问题**:
- "安排"被当作停用词，但"帮我安排"应该是创建意图
- "有什么"被过滤，可能导致语义丢失
- 停用词列表没有上下文，可能误判

#### 问题 2: 内容词判断简单

**当前实现** (`server/queryengine/query_router.go:434-448`):
```go
scheduleStopWords := []string{"日程", "安排", "事", "计划"}
isScheduleOnly := true
for _, word := range strings.Fields(contentQuery) {
    // 检查是否都是停用词
}
```

**问题**:
- 简单的词袋模型，没有考虑词序
- "今天有什么**会议**安排" → "会议"被保留 ✅
- "今天有什么**安排**" → 被判定为纯日程查询 ✅
- "今天**安排**了什么" → 被判定为纯日程查询，但可能有误

### 影响范围

- 查询意图识别的准确性
- 检索策略的选择
- 用户体验

---

## 三、设计目标

1. **准确性**: 提高意图识别准确率
2. **上下文感知**: 考虑词序和上下文
3. **可扩展性**: 易于添加新的规则和模式
4. **可观测性**: 提供意图识别的调试信息

---

## 四、技术方案

### 4.1 意图分类体系

#### 4.1.1 查询意图类型

```
查询意图
├── 日程查询
│   ├── 纯日程查询 (schedule_bm25_only)
│   │   - 只有时间词，无内容词
│   │   - 例: "今天有什么日程"、"明天的安排"
│   │
│   └── 混合查询 (hybrid_with_time_filter)
│       - 时间词 + 内容词
│       - 例: "今天的会议安排"、"明天关于项目的事"
│
├── 笔记查询
│   ├── 纯笔记查询 (memo_semantic_only)
│   │   - 有笔记关键词，无专有名词
│   │   - 例: "关于React的笔记"、"搜索备忘录"
│   │
│   └── 专有名词查询 (hybrid_bm25_weighted)
│       - 主要是专有名词
│       - 例: "张三的联系方式"、"GitHub配置"
│
└── 通用查询 (hybrid_standard / full_pipeline_with_reranker)
    - 复杂问题，疑问词
    - 例: "如何使用React Hooks"、"为什么服务器会崩溃"
```

#### 4.1.2 创建意图类型

```
创建意图 (日程创建)
├── 明确创建
│   - 关键词: "创建"、"添加"、"新建"、"设置提醒"
│   - 例: "创建一个明天下午3点的会议"
│
├── 委托创建
│   - 关键词: "帮我"、"提醒我"
│   - 例: "帮我安排明天的时间"、"提醒我下午开会"
│
└── 模糊意图 (需要确认)
    - 表达不明确，需要二次确认
    - 例: "明天下午有事" (可能是查询，也可能是想创建)
```

---

### 4.2 停用词优化策略

#### 4.2.1 分类停用词

**功能性停用词** (总是过滤):
```go
functionalStopWords := []string{
    "的", "了", "在", "是", "我", "有", "和",
    "呢", "吗", "啊", "呀", "哦", "吧",
    "查询", "搜索", "查找", "查看", "看看",
}
```

**上下文停用词** (根据上下文决定):
```go
contextualStopWords := map[string][]string{
    "schedule_query": {"安排", "事", "计划", "活动"},
    "schedule_create": {"安排", "设置", "添加"},
    "memo_query": {"笔记", "备忘", "记录", "内容"},
}
```

**保留词** (不过滤):
```go
reservedWords := []string{
    "会议", "项目", "任务", "提醒", "计划",
    "创建", "添加", "新建", "设置",
    // 这些词对意图判断很重要
}
```

#### 4.2.2 上下文感知的停用词过滤

```go
// FilterStopWordsWithContext 上下文感知的停用词过滤
func FilterStopWordsWithContext(query string, intentHint string) string {
    // 1. 总是过滤功能性停用词
    filtered := query
    for _, word := range functionalStopWords {
        filtered = strings.ReplaceAll(filtered, word, " ")
    }

    // 2. 根据意图提示过滤上下文停用词
    if words, ok := contextualStopWords[intentHint]; ok {
        for _, word := range words {
            filtered = strings.ReplaceAll(filtered, word, " ")
        }
    }

    // 3. 保留保留词
    // 保留词不被过滤

    return strings.TrimSpace(filtered)
}
```

---

### 4.3 内容词提取优化

#### 4.3.1 词性标注（简化版）

**关键词分类**:
```go
wordCategories := map[string]WordCategory{
    // 时间词
    "今天": TIME_WORD, "明天": TIME_WORD, "本周": TIME_WORD,

    // 动作词
    "创建": ACTION_WORD_CREATE, "添加": ACTION_WORD_CREATE,
    "安排": ACTION_WORD_AMBIGUOUS,  // 可能是创建，也可能是查询
    "查询": ACTION_WORD_QUERY, "搜索": ACTION_WORD_QUERY,

    // 内容词
    "会议": CONTENT_WORD, "项目": CONTENT_WORD,
    "任务": CONTENT_WORD, "计划": CONTENT_WORD,

    // 实体词（专有名词）
    // ... 需要通过 NER 识别
}

type WordCategory int
const (
    UNKNOWN WordCategory = iota
    TIME_WORD
    ACTION_WORD_CREATE
    ACTION_WORD_QUERY
    ACTION_WORD_AMBIGUOUS
    CONTENT_WORD
    ENTITY_WORD
)
```

#### 4.3.2 句法模式识别

**查询模式**:
```go
queryPatterns := []struct{
    pattern string
    intent  string
}{
    // 时间 + 查询动词
    {`今天.*查询|查询.*今天`, "schedule_query"},
    {`明天.*有什么|有什么.*明天`, "schedule_query"},

    // 创建动词 + 时间 + 内容
    {`创建|添加|新建.*时间`, "schedule_create"},
    {`帮我.*安排|提醒我`, "schedule_create"},

    // 笔记关键词 + 内容
    {`笔记|备忘|记录.*内容`, "memo_query"},
}
```

#### 4.3.3 内容提取算法

```go
// ExtractContentWithSemantics 语义化内容提取
func ExtractContentWithSemantics(query string) *ContentExtraction {
    extraction := &ContentExtraction{
        OriginalQuery: query,
        TimeWords:     []string{},
        ActionWords:   []string{},
        ContentWords:  []string{},
        Entities:      []string{},
    }

    // 1. 分词（简化：使用空格分词，实际应使用中文分词）
    words := strings.Fields(query)

    // 2. 词性标注
    for _, word := range words {
        if category, ok := wordCategories[word]; ok {
            switch category {
            case TIME_WORD:
                extraction.TimeWords = append(extraction.TimeWords, word)
            case ACTION_WORD_CREATE, ACTION_WORD_QUERY:
                extraction.ActionWords = append(extraction.ActionWords, word)
            case CONTENT_WORD:
                extraction.ContentWords = append(extraction.ContentWords, word)
            }
        } else if isEntity(word) {
            extraction.Entities = append(extraction.Entities, word)
        }
    }

    // 3. 判断主要意图
    extraction.PrimaryIntent = inferPrimaryIntent(extraction)

    return extraction
}

// ContentExtraction 内容提取结果
type ContentExtraction struct {
    OriginalQuery string
    TimeWords     []string
    ActionWords   []string
    ContentWords  []string
    Entities      []string
    PrimaryIntent string
    Confidence    float32
}
```

---

### 4.4 意图判断优化

#### 4.4.1 意图判断决策树

```
                    ┌─────────────┐
                    │  用户查询    │
                    └──────┬──────┘
                           │
                ┌──────────┴──────────┐
                │ 是否有创建动词？    │
                └──────────┬──────────┘
                           │
              ┌────────────┴────────────┐
              │ 是                       │ 否
              ▼                          ▼
    ┌─────────────────┐       ┌─────────────────┐
    │ 创建意图        │       │ 是否有时间词？  │
    │ (高置信度)      │       └────────┬────────┘
    └─────────────────┘                │
                              ┌────────┴────────┐
                              │ 是               │ 否
                              ▼                  ▼
                    ┌─────────────┐     ┌─────────────┐
                    │ 日程查询     │     │ 是否有笔记   │
                    │             │     │ 关键词？    │
                    └──────┬──────┘     └──────┬──────┘
                           │                   │
                ┌──────────┴─────────┐         │
                │ 是否有内容词？      │         │
                └──────────┬─────────┘         │
                           │                   │
              ┌────────────┴────────────┐      │
              │ 是                       │ 否   │
              ▼                          ▼      ▼
    ┌─────────────────┐     ┌────────────┐ ┌────────┐
    │ 混合查询         │     │ 纯日程查询 │ │笔记查询│
    │ (hybrid+filter) │     │ (bm25_only)│ │        │
    └─────────────────┘     └────────────┘ └────────┘
```

#### 4.4.2 意图判断规则

**规则 1: 创建意图优先**
```go
if hasCreateKeyword(query) {
    return "schedule_create", 0.95  // 高置信度
}
```

**规则 2: 时间词驱动**
```go
if hasTimeKeyword(query) {
    if hasContentWord(query) {
        return "hybrid_with_time_filter", 0.90
    } else {
        return "schedule_bm25_only", 0.95
    }
}
```

**规则 3: 笔记关键词驱动**
```go
if hasMemoKeyword(query) {
    if isMostlyProperNouns(query) {
        return "hybrid_bm25_weighted", 0.85
    } else {
        return "memo_semantic_only", 0.90
    }
}
```

**规则 4: 疑问词驱动**
```go
if hasQuestionWord(query) {
    return "full_pipeline_with_reranker", 0.70
}
```

**规则 5: 默认策略**
```go
return "hybrid_standard", 0.80
```

---

### 4.5 可观测性设计

#### 4.5.1 意图识别结果

```go
// IntentDetectionResult 意图识别结果
type IntentDetectionResult struct {
    // 基础信息
    Query          string
    DetectedIntent string
    Confidence     float32

    // 分析结果
    TimeWords      []string
    ActionWords    []string
    ContentWords   []string
    Entities       []string

    // 决策路径
    DecisionPath   []string  // 记录判断过程

    // 推荐策略
    RecommendedStrategy string

    // 调试信息
    DebugInfo      map[string]interface{}
}
```

#### 4.5.2 日志输出

```go
// 日志格式
[IntentDetection] Query: "今天有什么会议安排？"
[IntentDetection] TimeWords: [今天]
[IntentDetection] ContentWords: [会议]
[IntentDetection] ActionWords: []
[IntentDetection] DecisionPath: [hasTimeKeyword=true, hasContentWord=true]
[IntentDetection] DetectedIntent: schedule_query
[IntentDetection] Confidence: 0.90
[IntentDetection] RecommendedStrategy: hybrid_with_time_filter
```

---

## 五、关键场景设计

### 5.1 场景 1: 纯日程查询

**用户查询**: "今天有什么日程？"

**分析过程**:

1. 分词: ["今天", "有什么", "日程"]
2. 词性标注:
   - "今天" → TIME_WORD
   - "有什么" → 功能性停用词
   - "日程" → 上下文停用词（schedule_query）
3. 内容提取:
   - TimeWords: ["今天"]
   - ContentWords: []
4. 意图判断:
   - 有时间词 → 日程查询
   - 无内容词 → 纯日程查询
5. 推荐: `schedule_bm25_only`

### 5.2 场景 2: 混合查询

**用户查询**: "今天关于项目的会议安排"

**分析过程**:

1. 分词: ["今天", "关于", "项目", "的", "会议", "安排"]
2. 词性标注:
   - "今天" → TIME_WORD
   - "项目", "会议" → CONTENT_WORD
   - "安排" → ACTION_WORD_AMBIGUOUS
3. 内容提取:
   - TimeWords: ["今天"]
   - ContentWords: ["项目", "会议"]
4. 意图判断:
   - 有时间词 + 有内容词 → 混合查询
5. 推荐: `hybrid_with_time_filter`

### 5.3 场景 3: 创建意图

**用户查询**: "帮我创建明天下午3点的会议"

**分析过程**:

1. 分词: ["帮我", "创建", "明天", "下午", "3点", "的", "会议"]
2. 词性标注:
   - "创建" → ACTION_WORD_CREATE
   - "明天", "下午" → TIME_WORD
   - "会议" → CONTENT_WORD
3. 内容提取:
   - TimeWords: ["明天", "下午"]
   - ContentWords: ["会议"]
   - ActionWords: ["创建"]
4. 意图判断:
   - 有创建动词 → 创建意图
5. 推荐: 添加 `<<<SCHEDULE_INTENT:...>>>` 标记

### 5.4 场景 4: 模糊意图

**用户查询**: "明天下午有事"

**分析过程**:

1. 分词: ["明天", "下午", "有", "事"]
2. 词性标注:
   - "明天", "下午" → TIME_WORD
   - "有" → 功能性停用词
   - "事" → 上下文停用词
3. 内容提取:
   - TimeWords: ["明天", "下午"]
   - ContentWords: []
4. 意图判断:
   - 有时间词，无内容词，无创建动词
   - 可能是查询，也可能是想创建
5. 推荐: `schedule_bm25_only`（查询优先）
6. **提示**: 可以在回复中询问"是否需要创建日程？"

---

## 六、测试用例

### 6.1 意图识别测试

| 测试用例 | 查询 | 预期意图 | 置信度 |
|---------|------|----------|--------|
| 纯日程查询 | "今天有什么日程？" | schedule_query | ≥ 0.9 |
| 混合查询 | "今天的会议安排" | schedule_query | ≥ 0.85 |
| 创建意图 | "创建明天的会议" | schedule_create | ≥ 0.95 |
| 委托创建 | "帮我安排明天的时间" | schedule_create | ≥ 0.85 |
| 笔记查询 | "关于React的笔记" | memo_query | ≥ 0.9 |
| 专有名词 | "张三的联系方式" | entity_query | ≥ 0.85 |
| 通用查询 | "如何使用React" | general_query | ≥ 0.7 |
| 模糊意图 | "明天有事" | schedule_query | ≥ 0.7 |

### 6.2 停用词过滤测试

| 测试用例 | 输入 | 过滤后 | 预期 |
|---------|------|--------|------|
| 功能停用词 | "今天的安排" | "安排" | 保留关键词 |
| 上下文停用词 | "查询今天的日程" | "" | 全部过滤 |
| 保留词 | "创建会议" | "创建 会议" | 保留 |
| 混合 | "今天的会议安排" | "会议" | 只保留内容词 |

### 6.3 内容提取测试

| 测试用例 | 输入 | 时间词 | 内容词 | 动作词 |
|---------|------|--------|--------|--------|
| 简单查询 | "今天有什么" | ["今天"] | [] | [] |
| 混合查询 | "今天的会议" | ["今天"] | ["会议"] | [] |
| 创建意图 | "创建明天会议" | ["明天"] | ["会议"] | ["创建"] |
| 复杂 | "明天下午关于项目的会议" | ["明天", "下午"] | ["项目", "会议"] | [] |

---

## 七、实施计划

### 7.1 阶段 1: 停用词优化

1. 分类停用词（功能性、上下文性、保留词）
2. 实现上下文感知的停用词过滤
3. 编写测试

### 7.2 阶段 2: 内容提取优化

1. 实现简化的词性标注
2. 实现语义化内容提取
3. 编写测试

### 7.3 阶段 3: 意图判断优化

1. 实现决策树逻辑
2. 添加可观测性支持
3. 编写测试

### 7.4 阶段 4: 集成和验证

1. 集成到 QueryRouter
2. A/B 测试对比
3. 性能优化

---

## 八、验收标准

### 8.1 功能验收

- [ ] 意图识别准确率 ≥ 90%
- [ ] 创建意图识别准确率 ≥ 95%
- [ ] 停用词过滤准确率 ≥ 85%

### 8.2 性能验收

- [ ] 意图识别时间 < 5ms
- [ ] 不影响整体响应时间（< 2% 增加）

### 8.3 测试验收

- [ ] 单元测试覆盖率 ≥ 85%
- [ ] 所有测试用例通过
- [ ] 零 P0/P1 bug

---

## 九、风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 词性标注不准确 | 使用简化版本，逐步优化 |
| 意图误判 | 添加置信度阈值，低置信度时使用默认策略 |
| 性能退化 | 优化规则匹配，使用缓存 |
| 规则过于复杂 | 保持规则简单，可解释 |

---

## 十、相关文档

- [实施方案](../IMPLEMENTATION_PLAN.md)
- [Chat 服务统一化](./CHAT_SERVICE_UNIFICATION.md)
- [日程查询优化](./SCHEDULE_QUERY_OPTIMIZATION.md)
