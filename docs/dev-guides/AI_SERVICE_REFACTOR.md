# AI 服务重构计划 v2

> **状态**: ✅ 已完成 (2026-01-24)
>
> **变更**: `ChatWithMemos` 已重命名为 `Chat`，废弃的 RPC 方法 (`ChatWithScheduleAgent`, `ChatWithMemosIntegrated`) 已删除
>
> **相关提交**: feat-ai-chat-session 分支

## 🎯 核心变更

### 问题分析
1. **命名问题**：`ChatWithMemos` 暗示使用 Memos（RAG 检索），但 DEFAULT 模式是纯 LLM 对话
2. **架构问题**：DEFAULT 和 CREATIVE 模式不需要鹦鹉框架，却被强制走复杂的路由逻辑
3. **语义混乱**：Memo = 鹦鹉灰灰，DEFAULT 模式不应与 Memos 绑定

### 重构方案

```
当前结构：
ChatWithMemos()
├── DEFAULT    → 直连 LLM (但名字暗示 Memos)
└── 其他 Parrot → 鹦鹉框架

目标结构：
Chat()           ← 新增：纯 LLM 对话（DEFAULT 使用）
├── simple LLM 对话
└── ChatWithAgents() ← 重命名：鹦鹉框架入口
    ├── MEMO       → 灰灰 (RAG)
    ├── SCHEDULE   → 金刚 (日程工具)
    ├── AMAZING    → 惊奇 (组合)
    └── CREATIVE   → 灵灵 (创意 LLM)
```

---

## 📁 文件结构

```
server/
├── router/api/v1/
│   ├── ai_service.go           # 主服务定义
│   ├── ai_chat.go              # 新增：纯 Chat 实现
│   ├── ai_agents.go            # 新增：鹦鹉路由入口
│   └── ai_service_chat.go     # 删除：合并到 ai_chat.go
│
├── internal/
│   ├── errors/
│   │   └── codes.go           # 统一错误码
│   └── observability/
│       ├── logger.go          # 结构化日志
│       └── metrics.go         # 指标采集
│
└── middleware/
    └── ai_validation.go       # AI 专用验证
```

---

## 🔧 API 设计

### 1. 新增：纯 Chat API（默认模式）

```go
// ai_chat.go

// Chat 纯 LLM 对话，不使用任何 Agent
// 这是 DEFAULT 模式使用的接口
func (s *AIService) Chat(
    ctx context.Context,
    req *v1pb.ChatRequest,
    stream v1pb.AIService_ChatServer,
) error {
    // 1. 验证和限流
    // 2. 构建 messages (system + history + user)
    // 3. 流式调用 LLM
    // 4. 发送响应
}
```

**请求/响应定义**：
```protobuf
// 新增纯 Chat 请求
message ChatRequest {
    string message = 1;
    repeated string history = 2;  // 可选的历史对话
    string system_prompt = 3;     // 可选的自定义系统提示词
}

// 复用现有的 ChatWithMemosResponse
message ChatResponse {
    string content = 1;      // 流式内容块
    bool done = 2;           // 完成标记
}
```

### 2. 重命名：ChatWithAgents（鹦鹉框架入口）

```go
// ai_agents.go

// ChatWithAgents 通过鹦鹉框架处理复杂任务
// MEMO、SCHEDULE、AMAZING、CREATIVE 都走这个入口
func (s *AIService) ChatWithAgents(
    ctx context.Context,
    req *v1pb.ChatWithAgentsRequest,
    stream v1pb.AIService_ChatWithAgentsServer,
) error {
    // 1. 路由到对应 Parrot Agent
    // 2. 执行 Agent (带工具调用)
    // 3. 流式返回事件
}
```

**请求定义**：
```protobuf
message ChatWithAgentsRequest {
    string message = 1;
    repeated string history = 2;
    AgentType agent_type = 3;      // MEMO, SCHEDULE, AMAZING, CREATIVE
    string user_timezone = 4;
}

message ChatWithAgentsResponse {
    string event_type = 1;      // thinking, tool_use, tool_result, answer
    string event_data = 2;      // JSON 或纯文本
    bool done = 3;
}
```

### 3. 彻底删除 ChatWithMemos

**删除策略**：
- ❌ 不保留向后兼容适配器
- ❌ 不保留 `ChatWithMemos` 方法
- ✅ 直接删除，前端必须同步迁移

**原因**：
- `ChatWithMemos` 语义混乱（Memo = 鹦鹉灰灰，但 DEFAULT 不走 Memos）
- 新 API 命名更清晰，无需适配器层

**前端迁移**：
```typescript
// 旧代码 - 删除
aiServiceClient.chatWithMemos(request)

// 新代码 - DEFAULT 模式
aiServiceClient.chat(request)

// 新代码 - Parrot 模式
aiServiceClient.chatWithAgents(request)
```

## 🔄 迁移路径

### 阶段 1：删除旧接口，新增新接口

```
第1步：新增 Chat() 和 ChatWithAgents() gRPC 方法
第2步：删除 ChatWithMemos() 方法
第3步：更新 Proto 定义
第4步：前端同步迁移
```

### 阶段 2：前端同步迁移

```typescript
// 旧代码 - 删除
aiServiceClient.chatWithMemos(request)

// 新代码 - DEFAULT 模式
aiServiceClient.chat(request)

// 新代码 - Parrot 模式
aiServiceClient.chatWithAgents(request)
```

### 迁移对照表

| 旧 AgentType | 新接口 | 说明 |
|-------------|--------|------|
| `AGENT_TYPE_DEFAULT` | `Chat()` | 纯 LLM 对话 |
| `AGENT_TYPE_MEMO` | `ChatWithAgents()` + `MEMO` | 灰灰 + RAG |
| `AGENT_TYPE_SCHEDULE` | `ChatWithAgents()` + `SCHEDULE` | 金刚 + 日程工具 |
| `AGENT_TYPE_AMAZING` | `ChatWithAgents()` + `AMAZING` | 惊奇 + 组合 |
| `AGENT_TYPE_CREATIVE` | `ChatWithAgents()` + `CREATIVE` | 灵灵 + 创意 LLM |

---

## 📝 接口对比

| API | 用途 | RAG | 工具调用 | 复杂度 |
|-----|------|-----|---------|--------|
| `Chat()` | 纯 LLM 对话 | ❌ | ❌ | 低 |
| `ChatWithAgents()` | 鹦鹉框架 | ✅ | ✅ | 高 |

**语义清晰度：**
- `Chat()` → 简单的 AI 对话
- `ChatWithAgents()` → 通过鹦鹉 Agents 的增强对话

---

## 🎯 重构优先级

### P0 (必须)
1. ✅ 创建 `Chat()` 接口 - 纯 LLM 对话
2. ✅ 创建 `ChatWithAgents()` 接口 - 鹦鹉框架入口
3. ✅ 更新 Proto 定义

### P1 (重要)
4. ✅ 统一错误处理
5. ✅ 可观测性增强
6. ✅ 前端适配

### P2 (优化)
7. ⏳ 单元测试迁移
8. ⏳ 文档更新
9. ⏳ 删除废弃代码

---

## 🔧 日志格式规范

### 统一日志格式（最佳实践）

```
[LEVEL] filename:line_number [component] message key=value key=value ...
```

**示例：**
```
[INFO] ai_chat.go:45 [AI] Chat request received user_id=123 agent_type=DEFAULT message_length=10
[INFO] ai_chat.go:78 [AI] LLM stream started duration_ms=123
[ERROR] ai_agents.go:56 [AI] Agent execution failed agent_type=MEMO error="timeout"
```

### 日志级别使用

| 级别 | 使用场景 | 示例 |
|------|----------|------|
| DEBUG | 详细调试信息 | 函数入口/出口、中间状态 |
| INFO | 正常业务流程 | 请求开始/完成、Agent 创建 |
| WARN | 可恢复的异常 | 重试、降级、配置警告 |
| ERROR | 错误需要关注 | Agent 失败、LLM 错误 |

### 关键日志字段

```go
const (
    LogFieldComponent   = "component"   // 组件名（AI、Agent、LLM）
    LogFieldUserID      = "user_id"     // 用户 ID
    LogFieldAgentType   = "agent_type"  // Agent 类型
    LogFieldRequestID   = "request_id"  // 请求 ID
    LogFieldDuration    = "duration_ms" // 耗时（毫秒）
    LogFieldMessageLen  = "msg_length"  // 消息长度
    LogFieldErrorCode   = "error_code"  // 错误码
    LogFieldToolName    = "tool_name"   // 工具名称
    LogFieldChunkCount   = "chunks"      // 流式块数
)
```

### 日志代码示例

```go
// ai_chat.go
func (s *AIService) Chat(...) error {
    slog.Info("Chat request started",
        slog.String(LogFieldComponent, "AI"),
        slog.Int64(LogFieldUserID, userID),
        slog.String(LogFieldAgentType, "DEFAULT"),
        slog.Int(LogFieldMessageLen, len(req.Message)),
    )

    start := time.Now()
    // ... 业务逻辑 ...

    slog.Info("Chat request completed",
        slog.String(LogFieldComponent, "AI"),
        slog.Int64(LogFieldUserID, userID),
        slog.String(LogFieldAgentType, "DEFAULT"),
        slog.Int(LogFieldDuration, time.Since(start).Milliseconds()),
        slog.Int(LogFieldChunkCount, chunkCount),
    )
}
```

```go
// ai_agents.go
func (s *AIService) ChatWithAgents(...) error {
    slog.Info("Agent execution started",
        slog.String(LogFieldComponent, "Agent"),
        slog.String(LogFieldAgentType, agentTypeStr),
        slog.Int64(LogFieldUserID, userID),
    )

    // ... Agent 执行 ...

    if err != nil {
        slog.Error("Agent execution failed",
            slog.String(LogFieldComponent, "Agent"),
            slog.String(LogFieldAgentType, agentTypeStr),
            slog.String(LogFieldErrorCode, "EXECUTION_FAILED"),
            slog.String("error", err.Error()),
        )
    }
}
```

---

## 🧪 验证清单

- [ ] Proto 编译通过
- [ ] 新接口测试通过
- [ ] 向后兼容测试通过
- [ ] 前端功能正常
- [ ] 日志格式符合规范（含文件名:行号）
- [ ] 指标采集正常
