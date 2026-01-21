# P1 阶段实施计划 - 日程查询优化

**开始日期**: 2026-01-21
**预计完成**: 1周
**状态**: 实施中

---

## 一、当前状态

✅ **已完成**:
- P0 阶段：时区统一化
- P0 阶段：Chat 服务统一化
- 测试修复：所有测试 100% 通过 (219/219)

📊 **测试通过率**:
- server/queryengine: 52/52 (100%)
- server/retrieval: 40/40 (100%)
- server/router/api/v1: 125/125 (100%)
- server/timezone: 11/11 (100%)

---

## 二、P1 阶段目标

### 核心任务

1. **日程查询模式**
   - 实现标准模式和严格模式
   - 自动模式选择逻辑
   - API 扩展（可选参数）

2. **明确年份支持**
   - 支持"2025年1月21日"格式
   - 支持 YYYY-MM-DD, YYYY/MM/DD 格式
   - 优化年份推断算法

3. **更多时间表达**
   - 支持更多相对年份表达（后年、大后年等）
   - 改进年份推断逻辑

---

## 三、详细实施步骤

### Step 1: API 扩展 (1天)

#### 1.1 Proto API 更新

**文件**: `proto/api/v1/ai_service.proto`

```protobuf
message ChatWithMemosRequest {
  string message = 1;
  repeated string history = 2;
  string user_timezone = 3;

  // 新增：日程查询模式
  ScheduleQueryMode schedule_query_mode = 4;  // 可选，默认为 AUTO
}

enum ScheduleQueryMode {
  AUTO = 0;       // 自动选择（默认）
  STANDARD = 1;   // 标准模式：返回范围内有任何部分的日程
  STRICT = 2;     // 严格模式：只返回完全在范围内的日程
}
```

**操作**:
```bash
# 1. 修改 proto 文件
# 2. 重新生成 Go 代码
make generate
```

#### 1.2 RouteDecision 扩展

**文件**: `server/queryengine/query_router.go`

```go
type RouteDecision struct {
    // 现有字段
    Strategy   string
    Confidence float32
    TimeRange  *TimeRange
    SemanticQuery string

    // 新增字段
    ScheduleQueryMode ScheduleQueryMode // 日程查询模式
}

type ScheduleQueryMode int32

const (
    AutoQueryMode      ScheduleQueryMode = 0  // 自动选择
    StandardQueryMode  ScheduleQueryMode = 1  // 标准模式
    StrictQueryMode    ScheduleQueryMode = 2  // 严格模式
)
```

---

### Step 2: 模式选择逻辑 (1天)

#### 2.1 自动模式选择算法

**文件**: `server/queryengine/query_router.go`

```go
// determineScheduleQueryMode 确定日程查询模式
func (r *QueryRouter) determineScheduleQueryMode(
    query string,
    userMode ScheduleQueryMode,
    timeRange *TimeRange,
) ScheduleQueryMode {
    // 1. 用户明确指定 → 使用用户选择
    if userMode != AutoQueryMode {
        return userMode
    }

    // 2. 自动选择规则
    if timeRange == nil {
        return StandardQueryMode // 默认标准模式
    }

    // 规则：
    // - 相对时间（今天、明天、本周）→ 标准模式
    // - 绝对时间（1月21日、2025-01-21）→ 严格模式

    // 检查是否为相对时间
    relativeTimeKeywords := []string{
        "今天", "明天", "后天", "昨天",
        "本周", "下周", "上周",
        "本月", "下月", "上月",
        "今年", "明年", "去年",
        "近期", "最近",
    }

    for _, keyword := range relativeTimeKeywords {
        if strings.Contains(timeRange.Label, keyword) {
            return StandardQueryMode // 相对时间用标准模式
        }
    }

    // 绝对时间用严格模式
    return StrictQueryMode
}
```

#### 2.2 集成到 Route 方法

```go
func (r *QueryRouter) Route(_ context.Context, query string, userTimezone *time.Location) *RouteDecision {
    // ... 现有逻辑 ...

    decision := &RouteDecision{
        Strategy:   strategy,
        Confidence: confidence,
        TimeRange:  timeRange,
        SemanticQuery: contentQuery,
    }

    // 新增：确定日程查询模式
    decision.ScheduleQueryMode = r.determineScheduleQueryMode(
        query,
        AutoQueryMode, // TODO: 从请求参数获取
        timeRange,
    )

    return decision
}
```

---

### Step 3: 明确年份支持 (2天)

#### 3.1 扩展日期解析

**文件**: `server/queryengine/query_router.go`

```go
// detectTimeRangeWithTimezone 增强版
func (r *QueryRouter) detectTimeRangeWithTimezone(query string, userTimezone *time.Location) *TimeRange {
    if userTimezone == nil {
        userTimezone = utcLocation
    }
    now := time.Now().In(userTimezone)

    // ============================================================
    // 0. 明确年份日期（新增：P1 优化）
    // ============================================================

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

    // ... 继续现有的相对时间匹配逻辑 ...
}
```

#### 3.2 改进年份推断

```go
// inferYear 启发式推断年份
func inferYear(month, day int, now time.Time, userTimezone *time.Location) int {
    currentYear := now.Year()

    // 1. 尝试当年
    candidateDate := time.Date(currentYear, time.Month(month), day, 0, 0, 0, 0, userTimezone)

    // 如果日期在未来（含今天），使用当年
    if !candidateDate.Before(now) {
        return currentYear
    }

    // 2. 日期在过去，判断是否应该使用明年
    daysSince := int(now.Sub(candidateDate).Hours() / 24)

    // 规则：如果在最近3个月内（90天），可能是在查询明年的循环计划
    if daysSince <= 90 {
        return currentYear + 1 // 使用明年
    }

    // 3. 超过3个月，仍使用当年（历史查询）
    return currentYear
}
```

---

### Step 4: 更多时间表达 (1天)

#### 4.1 扩展时间关键词

**文件**: `server/queryengine/query_router.go`

```go
func (r *QueryRouter) initTimeKeywords() {
    // ... 现有关键词 ...

    // ============================================================
    // 新增：更远的年份关键词
    // ============================================================

    // 后年（当前年份 + 2）
    r.timeKeywords["后年"] = func(t time.Time) *TimeRange {
        utcTime := t.In(utcLocation)
        targetYear := utcTime.Year() + 2
        start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, utcLocation)
        end := time.Date(targetYear+1, 1, 1, 0, 0, 0, 0, utcLocation)
        return &TimeRange{Start: start, End: end, Label: "后年"}
    }

    // 大后年（当前年份 + 3）
    r.timeKeywords["大后年"] = func(t time.Time) *TimeRange {
        utcTime := t.In(utcLocation)
        targetYear := utcTime.Year() + 3
        start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, utcLocation)
        end := time.Date(targetYear+1, 1, 1, 0, 0, 0, 0, utcLocation)
        return &TimeRange{Start: start, End: end, Label: "大后年"}
    }

    // 前年（当前年份 - 2）
    r.timeKeywords["前年"] = func(t time.Time) *TimeRange {
        utcTime := t.In(utcLocation)
        targetYear := utcTime.Year() - 2
        start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, utcLocation)
        end := time.Date(targetYear+1, 1, 1, 0, 0, 0, 0, utcLocation)
        return &TimeRange{Start: start, End: end, Label: "前年"}
    }

    // ... 同义词映射 ...
    r.timeKeywords["大前年"] = r.timeKeywords["前年"]
}
```

---

### Step 5: 查询逻辑集成 (1天)

#### 5.1 修改 RetrievalOptions

**文件**: `server/retrieval/adaptive_retrieval.go`

```go
type RetrievalOptions struct {
    // 现有字段
    Strategy    string
    UserID      int32
    Query       string
    Limit       int
    MinScore    float32
    TimeRange   *queryengine.TimeRange

    // 新增字段
    ScheduleQueryMode queryengine.ScheduleQueryMode // 日程查询模式
}
```

#### 5.2 应用查询模式

**文件**: `server/store/db/postgres/schedule.go`

```go
func (s *ScheduleStore) ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error) {
    // ... 现有逻辑 ...

    // 根据 ScheduleQueryMode 选择 WHERE 条件
    if find.TimeRange != nil {
        switch find.ScheduleQueryMode {
        case queryengine.StrictQueryMode:
            // 严格模式：只返回完全在范围内的日程
            where, args = append(where, "schedule.start_ts >= ? AND (schedule.end_ts <= ? OR schedule.end_ts IS NULL)"), find.TimeRange.Start, find.TimeRange.End
        case queryengine.StandardQueryMode, queryengine.AutoQueryMode, default:
            // 标准模式：返回范围内有任何部分的日程
            where, args = append(where, "(schedule.end_ts >= ? OR schedule.end_ts IS NULL) AND schedule.start_ts <= ?"), find.TimeRange.Start, find.TimeRange.End
        }
    }

    // ...
}
```

---

### Step 6: 测试 (1天)

#### 6.1 单元测试

**文件**: `server/queryengine/query_router_p1_test.go` (新建)

```go
func TestQueryRouter_ExplicitYear(t *testing.T) {
    router := NewQueryRouter()
    ctx := context.Background()

    tests := []struct {
        name     string
        query    string
        expected string
    }{
        {"YYYY年MM月DD日", "2025年1月21日的日程", "2025年1月21日"},
        {"YYYY-MM-DD", "2025-01-21有什么安排", "2025-01-21"},
        {"YYYY/MM/DD", "2025/01/21的会议", "2025/01/21"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            decision := router.Route(ctx, tt.query, nil)
            if decision.TimeRange == nil {
                t.Errorf("Expected time range for query '%s'", tt.query)
                return
            }
            if decision.TimeRange.Label != tt.expected {
                t.Errorf("Label = %v, want %v", decision.TimeRange.Label, tt.expected)
            }
        })
    }
}

func TestQueryRouter_FarYearKeywords(t *testing.T) {
    router := NewQueryRouter()
    ctx := context.Background()

    tests := []struct {
        name     string
        query    string
        expected string
    }{
        {"后年", "后年的计划", "后年"},
        {"大后年", "大后年的目标", "大后年"},
        {"前年", "前年的数据", "前年"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            decision := router.Route(ctx, tt.query, nil)
            if decision.TimeRange == nil {
                t.Errorf("Expected time range for query '%s'", tt.query)
                return
            }
            if decision.TimeRange.Label != tt.expected {
                t.Errorf("Label = %v, want %v", decision.TimeRange.Label, tt.expected)
            }
        })
    }
}

func TestQueryRouter_QueryModeSelection(t *testing.T) {
    router := NewQueryRouter()
    ctx := context.Background()

    tests := []struct {
        name     string
        query    string
        expected ScheduleQueryMode
    }{
        {"相对时间 - 今天", "今天的日程", StandardQueryMode},
        {"相对时间 - 本周", "本周的安排", StandardQueryMode},
        {"绝对时间 - 1月21日", "1月21日的会议", StrictQueryMode},
        {"明确年份", "2025年1月21日", StrictQueryMode},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            decision := router.Route(ctx, tt.query, nil)
            if decision.ScheduleQueryMode != tt.expected {
                t.Errorf("Mode = %v, want %v", decision.ScheduleQueryMode, tt.expected)
            }
        })
    }
}
```

#### 6.2 集成测试

**文件**: `server/router/api/v1/p1_integration_test.go` (新建)

测试完整的查询流程，验证：
- 标准模式返回跨天日程
- 严格模式不返回跨天日程
- 明确年份正确解析

---

### Step 7: 文档更新 (0.5天)

更新以下文档：
- P1 实施报告
- API 文档
- 用户指南

---

## 四、验收标准

| 标准 | 要求 | 验证方法 |
|------|------|----------|
| 功能完整性 | 所有新功能实现 | 代码审查 |
| 测试覆盖 | ≥90% | go test -cover |
| 性能影响 | <5% | Benchmark 对比 |
| 向后兼容 | 现有功能不受影响 | 回归测试 |
| 文档完整 | 所有变更都有文档 | 文档审查 |

---

## 五、风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| API 变更破坏兼容性 | 高 | 新字段设为可选，默认 AUTO |
| 性能退化 | 中 | 性能基准测试，优化热点 |
| 测试覆盖不足 | 中 | 增加 P1 专项测试 |
| 年份推断错误 | 中 | 启发式算法 + 用户确认机制 |

---

## 六、时间线

| 阶段 | 任务 | 预计时间 | 负责人 |
|------|------|----------|--------|
| Step 1 | API 扩展 | 1天 | - |
| Step 2 | 模式选择 | 1天 | - |
| Step 3 | 明确年份 | 2天 | - |
| Step 4 | 更多表达 | 1天 | - |
| Step 5 | 查询集成 | 1天 | - |
| Step 6 | 测试 | 1天 | - |
| Step 7 | 文档 | 0.5天 | - |
| **总计** | | **7.5天** | |

---

## 七、下一步行动

**立即开始**:
1. 创建分支 `feature/p1-schedule-query-optimization`
2. 更新 Proto API (`make generate`)
3. 实现模式选择逻辑

**本周完成**:
- Step 1-3: API 扩展和明确年份支持

**下周完成**:
- Step 4-7: 更多表达、集成、测试和文档

---

**文档版本**: v1.0
**创建时间**: 2026-01-21
**状态**: 准备实施
