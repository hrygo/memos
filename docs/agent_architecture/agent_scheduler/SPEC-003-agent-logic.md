# SPEC-003: 日程 ReAct 智能体逻辑 (Schedule ReAct Agent Logic)

## 1. 目标 (Goal)
使用 `langchaingo/agents/openai_tools` 构建 `ScheduleAgent`，用于编排推理循环。

## 2. 智能体配置 (Agent Configuration)

### A. System Prompt 策略
Prompt 必须将模型锚定在用户的具体时间上下文中，以防止幻觉。
**模板**: I am a helpful assistant...
**动态上下文**:
*   `Now (UTC)`
*   `Now (Local)` + `Timezone`
*   `Current Weekday` (当前星期几)

### B. 初始化
```go
func NewScheduleAgent(llm llms.Model, service ScheduleService) *Agent {
    tools := []tools.Tool{
        NewScheduleQueryTool(service),
        NewScheduleAddTool(service),
    }
    return agents.NewOpenAIToolsAgent(llm, tools, opts...)
}
```

## 3. 需求 (Requirements)
*   **R1 (循环限制)**: 智能体执行循环必须有最大迭代限制（例如 5 步），以防止无限循环。
*   **R2 (上下文)**: `Execute` 方法必须接受携带用户信息的 `context.Context`。
*   **R3 (错误恢复)**: 如果 LLM 产生了无效的工具参数 JSON，智能体必须捕获此错误并将错误反馈给 LLM 以进行自我修正。

## 4. 关键实施注意事项 (Watchlist)
*   **错误恢复策略 (Error Recovery Strategies)**:
    *   **JSON 修复**: 现实世界的 LLM 输出经常带有尾随逗号或 Markdown 代码块 (```json ... ```)。工具执行层**必须**在报错前执行“模糊解析 (Fuzzy Parsing)”（去除 markdown 标签，修复常见 JSON 错误）。
    *   **反馈循环**: 如果解析完全失败，**不要 Crash**。返回一个专门的观察结果：`System Error: Invalid JSON input. Please strictly align with the Schema and retry.` 这允许智能体在循环的下一步中自我修正。

## 5. 验收标准 (Acceptance Criteria)
*   [ ] 实现了 `plugin/ai/agent/scheduler.go`。
*   [ ] 集成测试：智能体成功回答“下周一我有什么安排？”，包括调用 QueryTool 并总结。
*   [ ] 集成测试：智能体成功处理“明天早上9点定个会”，正确计算日期并调用 AddTool。
