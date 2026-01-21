# SPEC-002: 日程智能体工具集 (Schedule Agent Tools)

## 1. 目标 (Goal)
实现标准的 `langchaingo` 工具，封装 `ScheduleCoreService` (源自 SPEC-001)，允许 AI 智能体安全地与数据库交互。

## 2. 工具定义 (Tool Definitions)

### A. `ScheduleQueryTool`
*   **名称**: `schedule_query`
*   **描述**: “搜索特定时间范围内的日程事件。输入必须是 ISO8601 格式的时间。”
*   **输入 Schema (Struct)**:
    ```go
    type ScheduleQueryInput struct {
        StartTime string `json:"start_time" description:"ISO8601 时间字符串 (例如 2026-01-01T09:00:00Z)"`
        EndTime   string `json:"end_time"   description:"ISO8601 时间字符串"`
    }
    ```
*   **逻辑**:
    1.  将 `StartTime`/`EndTime` 解析为 `time.Time`。
    2.  调用 `service.FindSchedules(ctx, userID, start, end)`。
    3.  将结果格式化为紧凑的 JSON 字符串 (ID, Title, Start, End)。
*   **校验**: `EndTime` 必须晚于 `StartTime`。

### B. `ScheduleAddTool`
*   **名称**: `schedule_add`
*   **描述**: “创建一个新的日程事件。**仅在**核实了可用性或者用户明确忽略冲突后才使用此工具。”
*   **输入 Schema (Struct)**:
    ```go
    type ScheduleAddInput struct {
        Title       string `json:"title"`
        StartTime   string `json:"start_time"`
        EndTime     string `json:"end_time"`
        Description string `json:"description,omitempty"`
    }
    ```
*   **逻辑**:
    1.  解析时间。
    2.  调用 `service.CreateSchedule(...)`。
    3.  返回包含新 Schedule ID 的成功消息。

## 3. 需求 (Requirements)
*   **R1**: 工具必须使用上下文注入来获取当前的 `UserID`。
*   **R2**: 工具必须优雅地处理 `time.ParseError`，并向 LLM 返回描述性的错误字符串。
*   **R3**: 输出应针对 Token 使用量进行优化（例如：不要返回完整的原始 DB 行，只返回相关字段）。

## 4. 关键实施注意事项 (Watchlist)
*   **时区处理 (Timezone Handling)**:
    *   数据库以 UTC 或 Unix 时间戳存储时间。
    *   LLM 以“用户本地时间”思考（例如：“周一三点”）。
    *   **要求**: `ScheduleQueryTool` **必须**在将结果返回给 Agent 之前，将 DB 时间戳转换为用户的本地时区（格式化字符串）。这可以防止 LLM 需要进行脑内 UTC 转换，从而减少幻觉。
    *   **输入**: 同样，来自 Agent 的输入很可能是本地时间 ISO8601（如果 System Prompt 指示了）或 UTC。工具必须通过 Prompt 严格执行一种格式，或健壮地处理两者。

## 5. 验收标准 (Acceptance Criteria)
*   [ ] 创建了 `plugin/ai/agent/tools/scheduler.go`，包含结构体和 `Call` 方法。
*   [ ] 单元测试 Mock 了 `Service` 接口，并验证工具能否正确解析 JSON 输入并调用服务方法。
*   [ ] 错误字符串清晰，且对 LLM 具有指导意义。
