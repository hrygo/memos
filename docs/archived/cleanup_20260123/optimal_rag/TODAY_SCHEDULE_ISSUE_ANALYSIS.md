# 🔍 "今日日程"问题深入分析

> **分析日期**：2025-01-21
> **问题描述**：既显示了日程列表，又说"没有日程信息"
> **状态**：✅ 问题已识别，解决方案已提出

---

## 📋 问题描述

### 用户反馈

```
既显示了日程列表
又显示了：
"根据您的笔记记录，目前没有关于今天日程安排的信息。
您的笔记中仅有一条关于"软件进化 集成AI功能"的内容，
没有涉及具体日程或待办事项。"
```

### 现象拆解

1. **前端显示**：✅ **显示了日程列表**
   - 说明检索功能正常
   - 日程数据被正确返回

2. **LLM 回复**：❌ **说"没有日程信息"**
   - 说明 LLM 没有看到日程数据
   - LLM 提到了"软件进化"笔记

3. **矛盾点**：
   - 前端有日程数据
   - LLM 却说没有

---

## 🔍 根本原因分析

### 问题根源：纯日程查询仍然检索了笔记数据

#### 执行流程追踪

```
用户查询: "今日日程"
    ↓
QueryRouter.Route
    ├─ 策略: schedule_bm25_only ✅
    ├─ 时间范围: 今天 00:00-24:00 ✅
    └─ 置信度: 0.95 ✅
    ↓
AdaptiveRetriever.Retrieve
    ├─ 调用: scheduleBM25Only ✅
    ├─ 查询: SELECT * FROM schedule WHERE start_ts >= ... ✅
    ├─ 返回: []*SearchResult (Type="schedule", Schedule=...) ✅
    ↓
    ⚠️ 问题出现！
    scheduleBM25Only 返回了日程数据 ✅
    但 memoResults 为空（应该是这样）✅
    scheduleResults 有日程数据 ✅
    ↓
构建上下文（ai_service_chat.go:245-261）
    ├─ for i, r := range memoResults {  // ⚠️ 只处理 memoResults
    ├─     contextBuilder.WriteString(...)  // ⚠️ 只添加笔记上下文
    ├─ }
    ├─ if r.Memo != nil {
    ├─     sources = append(sources, "memos/...")  // ⚠️ 只添加 memo 到 sources
    ├─ }
    ↓
    ↓
buildOptimizedMessages
    ├─ hasNotes = len(memoResults) > 0  // ⚠️ 可能是 true（如果检索了笔记）
    ├─ hasSchedules = len(scheduleResults) > 0  // ✅ 应该是 true
    ├─
    ├─ if hasNotes {
    ├─     userMsgBuilder.WriteString("### 📝 相关笔记\n")
    ├─     userMsgBuilder.WriteString(memoContext)  // ⚠️ 笔记上下文被添加
    ├─ }
    ├─
    ├─ if hasSchedules {
    ├─     userMsgBuilder.WriteString("### 📅 日程安排\n")
    ├─     for i, r := range scheduleResults {
    ├─         if r.Schedule != nil {  // ⚠️ 关键检查点
    ├─             // 添加日程到上下文
    ├─         }
    ├─     }
    ├─ }
    ↓
LLM 收到提示词
    ├─ 包含: 笔记上下文（如果有）
    ├─ 包含: 日程上下文（如果 r.Schedule != nil）
    └─ 回复: "没有日程信息"
```

### 可能的问题点

#### 问题 1：`r.Schedule` 为 nil ⚠️

**原因**：在 `scheduleBM25Only` 中：

```go
results = append(results, &SearchResult{
    ID:       int64(schedule.ID),
    Type:     "schedule",
    Score:    1.0,
    Content:  schedule.Title,
    Schedule: schedule,  // ✅ 设置了 Schedule
})
```

**但可能的问题**：
- 如果 `schedule` 对象在某些情况下不完整
- 或者 `Schedule` 字段没有正确传递

#### 问题 2：`hasSchedules` 判断错误 ⚠️

**判断逻辑**：
```go
var hasSchedules = len(scheduleResults) > 0
```

**可能的问题**：
- `scheduleResults` 为空
- 或者 `scheduleResults` 中的结果 `Type` 不是 `"schedule"`

#### 问题 3：LLM 优先回复笔记内容 ⚠️

**原因**：提示词顺序问题

```go
if hasNotes {
    userMsgBuilder.WriteString("### 📝 相关笔记\n")
    userMsgBuilder.WriteString(memoContext)
}

if hasSchedules {
    userMsgBuilder.WriteString("### 📅 日程安排\n")
    // ...
}
```

**LLM 可能的行为**：
- 如果同时有笔记和日程上下文
- LLM 可能优先处理笔记
- 或者混淆了笔记和日程的关系

#### 问题 4：纯日程查询不应该有笔记上下文 ⭐

**根本问题**：

在 `schedule_bm25_only` 策略下：
- 应该**只**检索日程数据
- 不应该**检索笔记数据

**但当前的实现**：
- `scheduleBM25Only` 只查询日程 ✅
- 但 `memoResults` 可能有数据（如果其他逻辑添加了）

**或者更严重的问题**：
- 如果"今日日程"被路由到 `hybrid_standard` 而不是 `schedule_bm25_only`
- 那就会同时检索笔记和日程
- 导致 LLM 混淆

---

## ✅ 解决方案

### 方案 1：确保纯日程查询不返回笔记数据（推荐）

**修改 `scheduleBM25Only` 函数**：

```go
func (r *AdaptiveRetriever) scheduleBM25Only(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
    // ... 查询日程 ...

    // 构建结果：只包含日程，不包含笔记
    results := make([]*SearchResult, 0, len(schedules))
    for _, schedule := range schedules {
        results = append(results, &SearchResult{
            ID:       int64(schedule.ID),
            Type:     "schedule",
            Score:    1.0,
            Content:  schedule.Title,
            Schedule: schedule,
            // ⚠️ 关键：不设置 Memo 字段，确保不会与笔记混淆
        })
    }

    // ✅ 确保没有笔记数据
    return results, nil
}
```

**验证**：在 `ai_service_chat.go` 中：

```go
// 分类后立即验证
if len(scheduleResults) > 0 && len(memoResults) > 0 {
    // 记录错误：纯日程查询不应该返回笔记
    fmt.Printf("[WARNING] 纯日程查询返回了笔记数据！\n")
    fmt.Printf("[DEBUG] 策略: %s\n", routeDecision.Strategy)
    fmt.Printf("[DEBUG] Memo 结果数: %d, Schedule 结果数: %d\n",
        len(memoResults), len(scheduleResults))
}
```

### 方案 2：修改提示词构建逻辑

**优化 `buildOptimizedMessages` 函数**：

```go
// 纯日程查询：不添加笔记上下文
if routeDecision.Strategy == "schedule_bm25_only" {
    // 只添加日程上下文，忽略笔记
    if hasSchedules {
        userMsgBuilder.WriteString("### 📅 今日日程安排\n")
        // ... 添加日程 ...
    }
} else {
    // 混合查询：同时添加笔记和日程
    if hasNotes {
        userMsgBuilder.WriteString("### 📝 相关笔记\n")
        userMsgBuilder.WriteString(memoContext)
        userMsgBuilder.WriteString("\n")
    }

    if hasSchedules {
        userMsgBuilder.WriteString("### 📅 相关日程\n")
        // ... 添加日程 ...
    }
}
```

### 方案 3：明确提示纯日程查询

**修改系统提示词**：

```go
systemPrompt := `你是 Memos AI 助手，帮助用户管理笔记和日程。

## 重要：纯日程查询识别
当用户查询今日、明日、本周等时间范围的日程时（如"今日日程"、"明天安排"）：
1. **只回复日程信息**，不提笔记
2. 如果检索到日程，直接列出时间和标题
3. 如果没有日程，明确告知"今日暂无安排"

## 回复原则
1. **简洁准确**：基于提供的上下文回答，不编造信息
2. **结构清晰**：使用列表、分段组织内容
3. **自然对话**：像真人助手一样友好、直接

## 日程创建检测
当用户想创建日程时（关键词："创建"、"提醒"、"安排"、"添加"），在回复最后一行添加：
<<<SCHEDULE_INTENT:{"detected":true,"schedule_description":"自然语言描述"}>>>
```

### 方案 4：添加路由策略验证

**在 `ChatWithMemos` 开始时验证**：

```go
// 获取路由决策
routeDecision := s.QueryRouter.Route(ctx, req.Message)
fmt.Printf("[DEBUG] 路由决策: %s (置信度: %.2f)\n",
    routeDecision.Strategy, routeDecision.Confidence)

// 验证策略合理性
if routeDecision.Strategy == "schedule_bm25_only" {
    // 验证没有笔记数据
    if len(searchResults) > 0 {
        memoCount := 0
        for _, r := range searchResults {
            if r.Type == "memo" {
                memoCount++
            }
        }
        if memoCount > 0 {
            fmt.Printf("[ERROR] 策略矛盾！schedule_bm25_only 返回了笔记数据\n")
            fmt.Printf("[DEBUG] Memo 数量: %d, Schedule 数量: %d\n",
                memoCount, len(searchResults)-memoCount)
        }
    }
}
```

---

## 🧪 验证步骤

### 步骤 1：添加调试日志

在以下关键位置添加日志：

1. **检索结果分类后**
```go
fmt.Printf("[DEBUG] Memo: %d, Schedule: %d\n",
    len(memoResults), len(scheduleResults))
```

2. **提示词构建前**
```go
fmt.Printf("[DEBUG] hasNotes: %v, hasSchedules: %v\n", hasNotes, hasSchedules)
```

3. **日程上下文构建**
```go
for i, r := range scheduleResults {
    fmt.Printf("[DEBUG] Schedule[%d]: Schedule=%v\n", i, r.Schedule != nil)
}
```

### 步骤 2：运行测试

```bash
# 启动服务
make start

# 发送测试请求
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"message":"今天日程","history":[]}'

# 查看日志
make logs backend | grep -A 30 "今日日程"
```

### 步骤 3：检查输出

**预期的日志输出**：

```
[QueryRouting] Strategy: schedule_bm25_only, Confidence: 0.95
[DEBUG] Memo: 0, Schedule: 3  ← ✅ 没有笔记数据
[DEBUG] hasNotes: false, hasSchedules: true  ← ✅ 纯日程查询
[DEBUG] Schedule[0]: Schedule=true  ← ✅ Schedule 字段存在
```

**错误情况的日志输出**：

```
[QueryRouting] Strategy: schedule_bm25_only, Confidence: 0.95
[DEBUG] Memo: 2, Schedule: 3  ← ❌ 有笔记数据（矛盾！）
[DEBUG] hasNotes: true, hasSchedules: true
[DEBUG] Schedule[0]: Schedule=true
```

---

## 📋 问题诊断清单

请使用以下清单诊断问题：

- [ ] **1. 路由策略验证**
  - 检查 `[QueryRouting] Strategy` 日志
  - 应该是 `schedule_bm25_only`
  - 如果是 `hybrid_standard`，说明路由失败

- [ ] **2. 检索结果分类**
  - 检查 `[DEBUG] Memo: X, Schedule: Y` 日志
  - 对于"今日日程"，Memo 应该为 0
  - Schedule 应该 > 0

- [ ] **3. Schedule 字段验证**
  - 检查 `[DEBUG] Schedule[0]: Schedule=true/false` 日志
  - 应该都是 `true`
  - 如果是 `false`，说明 `Schedule` 字段为 nil

- [ ] **4. 提示词内容验证**
  - 检查 `[日程上下文字符串]` 日志
  - 应该包含日程信息
  - 不应该包含笔记信息

- [ ] **5. LLM 输入验证**
  - 检查发送给 LLM 的完整提示词
  - 确认日程上下文被正确包含

---

## 🎯 预期结果

### 正确的执行流程

```
用户: "今日日程"
    ↓
路由: schedule_bm25_only ✅
    ↓
检索: 只返回日程数据
    ├─ Memo: 0 条 ✅
    └─ Schedule: 3 条 ✅
    ↓
提示词构建:
    ├─ hasNotes: false ✅
    ├─ hasSchedules: true ✅
    ├─ 不添加笔记上下文 ✅
    └─ 添加日程上下文 ✅
    ↓
LLM 输入:
    """
    ## 上下文信息

    ### 📅 日程安排
    1. 2026-01-20 10:00 - 团队周会
    2. 2026-01-20 14:00 - 项目评审
    3. 2026-01-20 16:00 - 代码审查

    ## 问题
    今日日程
    """
    ↓
LLM 输出: "您今天的日程安排如下：..."
```

---

## 📝 总结

### 核心问题

1. **纯日程查询可能检索了笔记数据**
   - `schedule_bm25_only` 应该只返回日程
   - 但可能因为其他逻辑添加了笔记

2. **LLM 看到了笔记上下文**
   - 提示词构建时添加了 `memoContext`
   - 即使 `memoResults` 为空，`memoContext` 可能有内容

3. **LLM 优先回复笔记内容**
   - 提示词顺序问题
   - 或者 LLM 误解了查询意图

### 解决方案优先级

| 方案 | 优先级 | 难度 | 效果 |
|------|--------|------|------|
| 方案 1：确保纯日程不返回笔记 | P0 | 中 | ⭐⭐⭐⭐⭐ 根本解决 |
| 方案 2：修改提示词构建逻辑 | P0 | 低 | ⭐⭐⭐⭐⭐ 立即见效 |
| 方案 3：明确系统提示词 | P1 | 低 | ⭐⭐⭐⭐ 辅助 |
| 方案 4：添加验证和调试日志 | P2 | 低 | ⭐⭐⭐⭐ 诊断 |

### 下一步行动

**立即可执行**：
1. 添加调试日志验证上述假设
2. 根据日志输出确定具体问题
3. 应用对应的解决方案

**推荐方案**：
- 先应用方案 2（修改提示词构建逻辑）
- 同时添加方案 4（调试日志）
- 根据日志决定是否需要应用方案 1

---

**文档版本**：v1.0
**最后更新**：2025-01-21
**维护者**：Claude & Memos Team
