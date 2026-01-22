# SPEC-004: 前端交互与事件信令 (Frontend Interaction & Event Signaling)

## 1. 目标 (Goal)
在智能体推理过程中为用户提供即时反馈，并在发生变更时自动刷新 UI。

## 2. API 变更 (API Changes)
### 流式响应格式 (Stream Response Format)
聊天 API (`/api/v1/chat/stream`) 需要在文本块之外发出不同的事件类型。
*   `event: thinking`: "正在查询您的日历..." (显示为加载圈或状态文本)。
*   `event: tool_use`: "发现了 3 个日程。"
*   `event: answer`: "我已经帮您安排好了会议。"
*   `event: schedule_updated`: (信号，通知前端重新获取日历数据)。

## 3. 前端处理 (Frontend Handling)
**文件**: `web/src/components/AIChat/ChatWindow.tsx` (或类似文件)
*   **监听器**: 定制的 fetch/EventSource 客户端需要监听 `schedule_updated`。
*   **动作**: 收到 `schedule_updated` 时，分发全局事件或调用 `queryClient.invalidateQueries(['schedules'])`。

## 4. 需求 (Requirements)
*   **R1**: 当智能体思考时（可能延迟 10秒+），聊天 UI **不能阻塞**。必须有中间状态显示。
*   **R2**: 如果通过聊天添加了日程，侧边栏/主区域的日历视图必须在**不刷新页面**的情况下更新。

## 5. 验收标准 (Acceptance Criteria)
*   [ ] 后端 Chat Service 支持发送自定义 SSE 事件。
*   [ ] 前端正确渲染 "Thinking..." 状态。
*   [ ] 当智能体完成添加后，前端自动刷新日程列表。
