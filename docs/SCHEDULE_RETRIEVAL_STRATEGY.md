# 日程检索策略分析：语义检索 vs 时间检索

## 📊 核心问题

**日程是否适合纯语义检索？**

答案：**不适合**。日程需要 **混合检索策略**（时间 + 语义）。

---

## 🔍 日程 vs 笔记的本质区别

### 笔记（Memo）

| 特性 | 说明 |
|------|------|
| **核心属性** | 内容（文本） |
| **检索方式** | 语义相似度（向量搜索） |
| **典型查询** | "关于AI技术的笔记" |
| **时间维度** | 辅助参考，非核心 |
| **适用检索** | ✅ 纯语义检索 |

**示例**：
```
查询："搜索关于项目架构的笔记"
检索：向量相似度搜索
结果：相关度最高的笔记（不管时间）
```

### 日程（Schedule）

| 特性 | 说明 |
|------|------|
| **核心属性** | 时间（startTs, endTs） |
| **检索方式** | 时间范围 + 内容过滤 |
| **典型查询** | "今天有什么安排"、"明天关于项目的会议" |
| **时间维度** | **核心过滤条件** |
| **适用检索** | ❌ 纯语义检索 → ✅ **混合检索** |

**示例**：
```
查询："今天有什么安排"
预期：今天 00:00 - 23:59 的所有日程（不管内容）
纯语义检索：❌ 可能找到"昨天"的相关日程
混合检索：✅ 时间过滤 + 语义排序
```

---

## 📈 真实场景分析

### 场景 1：纯时间查询

**用户输入**："今天有什么安排"、"明天的事"、"下周会议"

**预期**：
- 时间范围内的 **所有** 日程
- 不关心内容相关度

**检索策略**：
```
✅ 纯 SQL 时间范围查询
SELECT * FROM schedule
WHERE start_ts >= today_start AND end_ts <= today_end

❌ 纯语义检索
→ 可能遗漏不相关的日程
→ 时间判断不准确
```

### 场景 2：纯语义查询

**用户输入**："关于项目A的会议"、"和客户相关的安排"

**预期**：
- 相关的日程（不限时间）
- 按相关度排序

**检索策略**：
```
✅ 语义向量检索
→ 找到标题/描述/地点相关的日程

❌ 纯时间查询
→ 无法判断内容相关度
→ 需要遍历所有时间段
```

### 场景 3：混合查询（最常见）

**用户输入**："今天关于项目A的会议"、"明天和客户相关的安排"

**预期**：
- **今天/明天**的日程（时间过滤）
- **关于项目A/客户**的（语义过滤）
- 按时间排序

**检索策略**：
```
✅ 混合检索
1. 时间过滤：start_ts BETWEEN today AND tomorrow
2. 语义过滤：vector_similarity(query, content) > 0.6
3. 综合排序：时间优先 + 相关度辅助

❌ 纯语义检索
→ 时间范围不准确
❌ 纯时间查询
→ 无法过滤内容
```

---

## 🎯 日程不适用纯语义检索的原因

### 1. 时间语义的复杂性

```
用户："今天的会议"
语义检索：可能匹配到
  - "今天的会议" ✅ 正确
  - "昨天的会议" ❌ 错误（语义相似但时间不对）
  - "关于今天的会议记录" ❌ 错误（笔记而非日程）

SQL 查询：WHERE start_ts >= today_start AND end_ts <= today_end
✅ 准确匹配今天的时间范围
```

### 2. 时间相对性

```
"今天" → 2025-01-21 00:00:00 ~ 2025-01-21 23:59:59
"明天" → 2025-01-22 00:00:00 ~ 2025-01-22 23:59:59
"下周" → 下周一 00:00:00 ~ 下周日 23:59:59

语义检索很难准确理解这些相对时间
```

### 3. 日程的有序性

```
日程必须按时间顺序展示：
09:00 → 10:00 → 14:00

语义检索按相关度排序：
相关度 0.95 → 0.85 → 0.70
时间：14:00 → 09:00 → 10:00 ❌ 顺序混乱
```

### 4. 边界条件

```
跨越两天的日程：
周一 23:00 ~ 周二 02:00

语义检索：
- "周一的安排" → 可能遗漏
- "周二的安排" → 可能遗漏

SQL 查询：
WHERE end_ts >= monday_start AND start_ts <= tuesday_end
✅ 正确匹配
```

---

## ✅ 推荐方案：混合检索策略

### 方案 1：时间过滤 + 语义排序（推荐 ⭐⭐⭐⭐⭐）

**适用场景**：有明确时间关键词（今天、明天、本周）

**流程**：
```
1. 时间范围检测
   - 检测："今天"、"明天"、"本周"等
   - 计算：start_ts, end_ts

2. SQL 时间范围过滤
   SELECT * FROM schedule
   WHERE creator_id = user_id
     AND end_ts >= query_start_ts
     AND start_ts <= query_end_ts
   LIMIT 50

3. 语义相似度排序
   - 对过滤后的日程计算向量相似度
   - 按相似度重排序
   - 或：按时间 + 相似度综合排序

4. 返回 Top 10
```

**优点**：
- ✅ 准确的时间范围
- ✅ 内容相关度排序
- ✅ 性能好（SQL 过滤快）
- ✅ 符合用户预期

**实现**：
```go
func (s *AIService) querySchedules(ctx context.Context, userID int32, intent *ScheduleQueryIntent) ([]*ScheduleSummary, error) {
    // 1. SQL 时间范围查询
    schedules, err := s.Store.ListSchedules(ctx, &store.FindSchedule{
        CreatorID: &userID,
        StartTs:   &intent.StartTime,
        EndTs:     &intent.EndTime,
    })

    // 2. 如果有语义关键词，进行相似度排序
    if intent.SemanticQuery != "" {
        queryVector, _ := s.EmbeddingService.Embed(ctx, intent.SemanticQuery)

        // 计算相似度
        for _, sched := range schedules {
            similarity := cosineSimilarity(queryVector, sched.Embedding)
            sched.Score = similarity
        }

        // 按相似度排序
        sort.Slice(schedules, func(i, j int) bool {
            return schedules[i].Score > schedules[j].Score
        })
    } else {
        // 按时间排序
        sort.Slice(schedules, func(i, j int) bool {
            return schedules[i].StartTs < schedules[j].StartTs
        })
    }

    return schedules[:min(10, len(schedules))], nil
}
```

### 方案 2：语义检索 + 时间过滤

**适用场景**：纯语义查询（"关于项目A的会议"）

**流程**：
```
1. 语义向量检索
   SELECT * FROM schedule_embedding
   WHERE creator_id = user_id
   ORDER BY embedding <=> query_vector
   LIMIT 50

2. 时间范围过滤（可选）
   - 如果用户指定时间，过滤结果
   - 否则返回所有相关日程

3. 按时间排序
```

**优点**：
- ✅ 找到内容相关的日程
- ✅ 适合无明确时间的查询

**缺点**：
- ❌ 可能遗漏低相似度但相关的日程
- ❌ 向量检索性能较 SQL 慢

### 方案 3：混合评分（高级 ⭐⭐⭐⭐）

**适用场景**：复杂混合查询

**流程**：
```
1. 时间过滤 + 语义检索并行执行
   - 获取时间范围内的日程
   - 获取语义相关的日程

2. 混合评分
   score = α * time_relevance + β * semantic_similarity

   其中：
   - time_relevance: 时间匹配度（0-1）
     - 完全匹配（今天查今天）: 1.0
     - 部分匹配（本周查今天）: 0.5
   - semantic_similarity: 向量相似度（0-1）
   - α, β: 权重系数（根据查询类型调整）

3. 综合排序返回
```

**优点**：
- ✅ 最灵活
- ✅ 平衡时间和语义

**缺点**：
- ⚠️ 复杂度高
- ⚠️ 需要调参

---

## 🏗️ 最终推荐架构

### 三层检索策略

```
┌─────────────────────────────────────────┐
│          用户查询输入                    │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│      意图检测（Intent Detection）         │
├─────────────────────────────────────────┤
│ 1. 时间关键词检测                        │
│    - 今天、明天、本周、下周等            │
│ 2. 语义关键词检测                        │
│    - 关于项目X、和客户等                 │
│ 3. 查询类型分类                          │
│    - 纯时间、纯语义、混合                │
└────────────┬────────────────────────────┘
             │
             ├─→ 纯时间 ────────┐
             │                  │
             ├─→ 纯语义 ────┐   │
             │              │   │
             └─→ 混合 ───┐   │   │
                        │   │   │
                        ▼   ▼   ▼
┌─────────────────────────────────────────┐
│           检索策略选择                    │
├─────────────────────────────────────────┤
│ 纯时间查询：                             │
│   → SQL 时间范围查询                     │
│   → 按时间排序                           │
│                                         │
│ 纯语义查询：                             │
│   → 向量相似度检索                       │
│   → 按相似度排序                         │
│                                         │
│ 混合查询：                               │
│   → SQL 时间过滤 + 语义排序              │
│   → 或：混合评分排序                     │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│         结果返回给 LLM                   │
└─────────────────────────────────────────┘
```

### 实现代码框架

```go
// ai_service.go

// ScheduleQueryIntent 日程查询意图
type ScheduleQueryIntent struct {
    Detected       bool
    QueryType      string  // "time_only", "semantic_only", "mixed"
    TimeRange      string  // "today", "tomorrow", "week", etc.
    StartTime      *time.Time
    EndTime        *time.Time
    SemanticQuery  string  // 语义关键词（如有）
}

// detectScheduleQueryIntent 检测日程查询意图
func (s *AIService) detectScheduleQueryIntent(message string) *ScheduleQueryIntent {
    // 1. 检测时间关键词
    hasTimeKeyword, timeRange := s.detectTimeKeywords(message)

    // 2. 检测语义关键词
    semanticKeywords := s.extractSemanticKeywords(message)

    // 3. 判断查询类型
    if hasTimeKeyword && semanticKeywords == "" {
        return &ScheduleQueryIntent{
            Detected:      true,
            QueryType:     "time_only",
            TimeRange:     timeRange,
            StartTime:     calculateTimeRange(timeRange).Start,
            EndTime:       calculateTimeRange(timeRange).End,
        }
    } else if !hasTimeKeyword && semanticKeywords != "" {
        return &ScheduleQueryIntent{
            Detected:      true,
            QueryType:     "semantic_only",
            SemanticQuery: semanticKeywords,
        }
    } else if hasTimeKeyword && semanticKeywords != "" {
        return &ScheduleQueryIntent{
            Detected:      true,
            QueryType:     "mixed",
            TimeRange:     timeRange,
            StartTime:     calculateTimeRange(timeRange).Start,
            EndTime:       calculateTimeRange(timeRange).End,
            SemanticQuery: semanticKeywords,
        }
    }

    return &ScheduleQueryIntent{Detected: false}
}

// querySchedules 查询日程
func (s *AIService) querySchedules(ctx context.Context, userID int32, intent *ScheduleQueryIntent) (*v1pb.ScheduleQueryResult, error) {
    var schedules []*store.Schedule
    var err error

    switch intent.QueryType {
    case "time_only":
        // 纯时间查询：SQL 时间范围查询
        schedules, err = s.Store.ListSchedules(ctx, &store.FindSchedule{
            CreatorID: &userID,
            StartTs:   intent.StartTime,
            EndTs:     intent.EndTime,
        })

        // 按时间排序
        sortSchedulesByTime(schedules)

    case "semantic_only":
        // 纯语义查询：向量检索
        schedules, err = s.vectorSearchSchedules(ctx, userID, intent.SemanticQuery)

        // 按相似度排序
        sortSchedulesBySimilarity(schedules)

    case "mixed":
        // 混合查询：时间过滤 + 语义排序
        schedules, err = s.Store.ListSchedules(ctx, &store.FindSchedule{
            CreatorID: &userID,
            StartTs:   intent.StartTime,
            EndTs:     intent.EndTime,
        })

        // 语义排序
        if intent.SemanticQuery != "" {
            s.rerankSchedulesBySemantic(ctx, schedules, intent.SemanticQuery)
        }

        // 按时间排序（同一天内按时间，跨天按日期）
        sortSchedulesByTime(schedules)
    }

    // 限制结果数量
    if len(schedules) > 10 {
        schedules = schedules[:10]
    }

    return s.buildScheduleQueryResult(intent, schedules)
}

// rerankSchedulesBySemantic 使用语义相关度重排序
func (s *AIService) rerankSchedulesBySemantic(ctx context.Context, schedules []*store.Schedule, query string) {
    queryVector, err := s.EmbeddingService.Embed(ctx, query)
    if err != nil {
        return
    }

    // 计算每个日程的相似度
    for _, sched := range schedules {
        // 获取日程的向量
        embedding, err := s.Store.GetScheduleEmbedding(ctx, sched.ID)
        if err != nil {
            continue
        }

        // 计算余弦相似度
        similarity := cosineSimilarity(queryVector, embedding.Embedding)
        sched.Score = similarity
    }

    // 按相似度排序
    sort.Slice(schedules, func(i, j int) bool {
        return schedules[i].Score > schedules[j].Score
    })
}
```

---

## 📊 性能对比

| 检索策略 | 时间范围准确度 | 内容相关度 | 性能 | 复杂度 | 推荐度 |
|---------|---------------|-----------|------|--------|--------|
| 纯 SQL 时间查询 | ⭐⭐⭐⭐⭐ | ⭐ | ⭐⭐⭐⭐⭐ | ⭐ | ⭐⭐⭐ |
| 纯语义向量检索 | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐ |
| 时间过滤 + 语义排序 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 混合评分排序 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

---

## 🎯 总结与建议

### ✅ 推荐：时间过滤 + 语义排序

**适用场景**：90% 的日程查询场景

**理由**：
1. ✅ 时间范围准确（SQL 查询）
2. ✅ 内容相关度排序（语义向量）
3. ✅ 性能好（SQL 过滤快）
4. ✅ 实现简单

### ⚠️ 不推荐：纯语义检索

**原因**：
1. ❌ 时间语义理解不准确
2. ❌ 可能遗漏日程
3. ❌ 排序不符合时间顺序
4. ❌ 性能较慢

### 🚀 进阶：混合评分（可选）

**适用场景**：对准确度要求极高的场景

**需要**：
- 更复杂的实现
- 参数调优
- 性能优化

---

## 📝 实施建议

### 第一阶段：基础实现
- ✅ 时间过滤 + 语义排序
- ✅ 覆盖 90% 场景

### 第二阶段：优化增强
- ✅ 混合评分（可选）
- ✅ 缓存优化
- ✅ 性能调优

### 第三阶段：智能演进
- ✅ 用户反馈学习
- ✅ 自适应权重调整
- ✅ A/B 测试验证

---

## 🔗 相关文档

- [统一 RAG 架构设计](./UNIFIED_RAG_ARCHITECTURE.md)
- [日程向量化实现](./SCHEDULE_EMBEDDING_IMPLEMENTATION.md)（待补充）
