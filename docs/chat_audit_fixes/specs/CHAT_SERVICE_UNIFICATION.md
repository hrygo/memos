# Chat 服务统一化设计规格

## 一、概述

**目标**: 统一两个 ChatWithMemos 实现（gRPC 和 Connect RPC），确保前端能获取完整的结构化数据。

**版本**: v1.0
**状态**: 设计中
**优先级**: P0 (紧急)

---

## 二、背景与问题

### 当前问题

1. **两个实现不一致**:
   - `server/router/api/v1/ai_service_chat.go` (gRPC 实现)
   - `server/router/api/v1/connect_handler.go` (Connect RPC 实现)

2. **Connect RPC 缺少关键响应**:
   - 未发送 `Done: true` 标记
   - 未发送 `ScheduleQueryResult` 结构化数据
   - 未发送 `ScheduleCreationIntent` 创建意图

3. **日程时间格式不同**:
   - Connect RPC: 只显示时间 `"15:04"`
   - gRPC: 显示完整日期时间 `"2006-01-02 15:04"`

### 影响范围

- 使用 Connect RPC 的前端无法获取日程查询结果
- 前端无法解析和展示日程列表
- 日程创建意图无法正确传递

---

## 三、设计目标

1. **一致性**: 两个实现返回相同的响应结构
2. **完整性**: 前端能获取所有需要的数据
3. **可维护性**: 提取公共逻辑，减少重复代码
4. **向后兼容**: 不破坏现有 API

---

## 四、技术方案

### 4.1 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                   Common Chat Logic                         │
│  (提取到独立的 helper 包或服务)                              │
│                                                             │
│  - buildChatContext()      构建聊天上下文                    │
│  - formatScheduleTime()    格式化日程时间                    │
│  - buildMessages()         构建消息                          │
│  - parseScheduleIntent()   解析日程意图                      │
│  - buildFinalResponse()    构建最终响应                      │
└──────────────┬──────────────────────────────┬───────────────┘
               │                              │
               ▼                              ▼
┌──────────────────────────┐    ┌──────────────────────────┐
│  gRPC ChatWithMemos      │    │  Connect ChatWithMemos   │
│  (ai_service_chat.go)    │    │  (connect_handler.go)   │
└──────────────────────────┘    └──────────────────────────┘
               │                              │
               └──────────────┬───────────────┘
                              ▼
                    ┌──────────────────┐
                    │   前端 (Web)      │
                    │   统一解析响应    │
                    └──────────────────┘
```

---

### 4.2 公共模块设计

#### 4.2.1 包结构

```
server/router/api/v1/
├── ai_service_chat.go          (gRPC 实现)
├── connect_handler.go          (Connect RPC 实现)
└── chat_common/                (新增公共逻辑)
    ├── context_builder.go      (上下文构建)
    ├── message_builder.go      (消息构建)
    ├── schedule_formatter.go   (日程格式化)
    ├── response_builder.go     (响应构建)
    └── intent_parser.go        (意图解析)
```

#### 4.2.2 核心接口

```go
package chat_common

// ChatContext 聊天上下文
type ChatContext struct {
    UserID           int32
    UserTimezone     *time.Location
    MemoResults      []*retrieval.SearchResult
    ScheduleResults  []*retrieval.SearchResult
    RouteDecision    *queryengine.RouteDecision
    RetrievalDuration time.Duration
}

// ContextBuilder 上下文构建器
type ContextBuilder interface {
    // BuildContext 构建聊天上下文
    BuildContext(ctx context.Context, opts *ContextOptions) (*ChatContext, error)
}

// MessageBuilder 消息构建器
type MessageBuilder interface {
    // BuildMessages 构建发送给 LLM 的消息
    BuildMessages(ctx *ChatContext, userMessage string, history []string) ([]ai.Message, error)
}

// ScheduleFormatter 日程格式化器
type ScheduleFormatter interface {
    // FormatScheduleTime 格式化日程时间
    FormatScheduleTime(schedule *store.Schedule, tz *time.Location) string
    // FormatSchedulesForContext 格式化日程列表用于上下文
    FormatSchedulesForContext(schedules []*retrieval.SearchResult, tz *time.Location) string
}

// ResponseBuilder 响应构建器
type ResponseBuilder interface {
    // BuildFinalResponse 构建最终响应
    BuildFinalResponse(ctx *ChatContext, aiResponse string) *ChatFinalResponse
}

// IntentParser 意图解析器
type IntentParser interface {
    // ParseScheduleIntent 解析日程创建意图
    ParseScheduleIntent(aiResponse string) *v1pb.ScheduleCreationIntent
}

// ContextOptions 上下文构建选项
type ContextOptions struct {
    UserID           int32
    Query            string
    SearchResults    []*retrieval.SearchResult
    RouteDecision    *queryengine.RouteDecision
    RetrievalDuration time.Duration
    UserTimezone     *time.Location
}

// ChatFinalResponse 聊天最终响应
type ChatFinalResponse struct {
    Content                 string
    Sources                 []string
    Done                    bool
    ScheduleQueryResult     *v1pb.ScheduleQueryResult
    ScheduleCreationIntent  *v1pb.ScheduleCreationIntent
}
```

---

### 4.3 响应结构设计

#### 4.3.1 完整的响应结构

```protobuf
message ChatWithMemosResponse {
  // 流式内容
  string content = 1;

  // 来源信息（首次发送）
  repeated string sources = 2;

  // 完成标记（最后发送）
  bool done = 3;

  // 日程查询结果（最后发送）
  ScheduleQueryResult schedule_query_result = 4;

  // 日程创建意图（最后发送）
  ScheduleCreationIntent schedule_creation_intent = 5;
}

message ScheduleQueryResult {
  repeated ScheduleSummary schedules = 1;
}

message ScheduleSummary {
  string uid = 1;
  string title = 2;
  int64 start_ts = 3;
  int64 end_ts = 4;
  bool all_day = 5;
  string location = 6;
  string recurrence_rule = 7;
  string status = 8;
}

message ScheduleCreationIntent {
  bool detected = 1;
  string schedule_description = 2;
}
```

#### 4.3.2 响应发送流程

```
┌─────────────────────────────────────────────────────────────┐
│                      聊天响应流程                             │
└─────────────────────────────────────────────────────────────┘

步骤 1: 发送来源信息
┌─────────────────────────────────────────────────────────────┐
│ Send({                                                     │
│   sources: ["memos/uid1", "schedules/123"]                  │
│ })                                                         │
└─────────────────────────────────────────────────────────────┘

步骤 2: 流式发送内容（多次）
┌─────────────────────────────────────────────────────────────┐
│ Send({ content: "今天" })                                   │
│ Send({ content: "有以下" })                                 │
│ Send({ content: "日程..." })                                │
└─────────────────────────────────────────────────────────────┘

步骤 3: 发送最终响应
┌─────────────────────────────────────────────────────────────┐
│ Send({                                                     │
│   done: true,                                              │
│   schedule_query_result: {                                 │
│     schedules: [                                           │
│       { uid: "schedules/1", title: "团队周会", ... }        │
│     ]                                                       │
│   },                                                       │
│   schedule_creation_intent: nil                            │
│ })                                                         │
└─────────────────────────────────────────────────────────────┘
```

---

### 4.4 日程时间格式化设计

#### 4.4.1 统一的时间格式规范

| 场景 | 格式 | 示例 |
|------|------|------|
| 上下文中的日程时间 | `2006-01-02 15:04` | `2026-01-21 14:00 - 团队周会` |
| 全天日程 | `2006-01-02` | `2026-01-21 - 生日` |
| 响应中的日程时间戳 | Unix 时间戳 | `1737446400` |
| 前端显示 | 由前端根据用户时区格式化 | - |

#### 4.4.2 时间格式化规则

```go
// FormatScheduleTime 格式化单个日程时间
// 规则:
// 1. 全天日程: "2006-01-02"
// 2. 有时间段: "2006-01-02 15:04 - 16:00"
// 3. 无结束时间: "2006-01-02 15:00"
func FormatScheduleTime(schedule *store.Schedule, tz *time.Location) string {
    startTime := time.Unix(schedule.StartTs, 0).In(tz)

    if schedule.AllDay {
        return startTime.Format("2006-01-02")
    }

    if schedule.EndTs != nil {
        endTime := time.Unix(*schedule.EndTs, 0).In(tz)
        return fmt.Sprintf("%s - %s",
            startTime.Format("2006-01-02 15:04"),
            endTime.Format("15:04"))
    }

    return startTime.Format("2006-01-02 15:04")
}

// FormatScheduleForContext 格式化日程用于 LLM 上下文
// 格式: "1. 2026-01-21 14:00 - 团队周会 @ 会议室A"
func FormatScheduleForContext(schedule *store.Schedule, index int, tz *time.Location) string {
    timeStr := FormatScheduleTime(schedule, tz)
    result := fmt.Sprintf("%d. %s - %s", index+1, timeStr, schedule.Title)

    if schedule.Location != "" {
        result += fmt.Sprintf(" @ %s", schedule.Location)
    }

    return result
}
```

---

### 4.5 两个实现的修改

#### 4.5.1 ai_service_chat.go 修改

**目标**: 使用公共逻辑，减少重复代码

**修改点**:

1. 导入公共包:
```go
import "github.com/usememos/memos/server/router/api/v1/chat_common"
```

2. 使用公共的上下文构建器:
```go
contextBuilder := chat_common.NewContextBuilder(s.Store, s.AdaptiveRetriever)
chatCtx, err := contextBuilder.BuildContext(ctx, &chat_common.ContextOptions{
    UserID: user.ID,
    Query: req.Message,
    SearchResults: searchResults,
    RouteDecision: routeDecision,
    UserTimezone: userTimezone,
})
```

3. 使用公共的响应构建器:
```go
responseBuilder := chat_common.NewResponseBuilder()
finalResponse := responseBuilder.BuildFinalResponse(chatCtx, fullContent.String())
```

#### 4.5.2 connect_handler.go 修改

**目标**: 补充缺失的响应逻辑

**修改点**:

1. **流结束时的处理** (关键修改):

```go
// 当前代码 (问题):
for {
    select {
    case content, ok := <-contentChan:
        if !ok {
            return nil  // ❌ 直接返回，没有发送最终响应
        }
        // ...
    }
}

// 修改后:
for {
    select {
    case content, ok := <-contentChan:
        if !ok {
            contentChan = nil
            if errChan == nil {
                // ✅ 构建并发送最终响应
                return s.sendFinalResponse(stream, chatCtx, fullContent.String())
            }
            continue
        }
        // ...
    }
}

// sendFinalResponse 发送最终响应
func (s *ConnectServiceHandler) sendFinalResponse(
    stream *connect.ServerStream[v1pb.ChatWithMemosResponse],
    chatCtx *chat_common.ChatContext,
    aiResponse string,
) error {
    // 使用公共响应构建器
    responseBuilder := chat_common.NewResponseBuilder()
    finalResponse := responseBuilder.BuildFinalResponse(chatCtx, aiResponse)

    // 发送最终响应
    return stream.Send(&v1pb.ChatWithMemosResponse{
        Done: finalResponse.Done,
        ScheduleQueryResult: finalResponse.ScheduleQueryResult,
        ScheduleCreationIntent: finalResponse.ScheduleCreationIntent,
    })
}
```

2. **导入公共包** (同 ai_service_chat.go)

3. **使用公共的日程格式化**:
```go
// 修改前:
timeStr := scheduleTime.Format("15:04")  // ❌ 只显示时间

// 修改后:
formatter := chat_common.NewScheduleFormatter()
timeStr := formatter.FormatScheduleTime(schedule, userTimezone)  // ✅ 显示完整日期时间
```

---

## 五、关键场景设计

### 5.1 场景 1: 纯日程查询

**用户查询**: "今天有哪些日程？"

**执行流程**:

1. 前端发送请求到 Connect RPC
2. connect_handler.go 处理请求
3. QueryRouter 检测为日程查询
4. AdaptiveRetriever 检索日程
5. 构建响应并发送:
   - 先发送 sources: `["schedules/1", "schedules/2"]`
   - 流式发送 LLM 回复内容
   - 最后发送最终响应:
     ```json
     {
       "done": true,
       "schedule_query_result": {
         "schedules": [
           {
             "uid": "schedules/1",
             "title": "团队周会",
             "start_ts": 1737446400,
             "end_ts": 1737450000,
             "location": "会议室A"
           }
         ]
       }
     }
     ```
6. 前端接收并展示日程列表

### 5.2 场景 2: 日程创建意图

**用户查询**: "帮我创建一个明天下午3点的会议"

**执行流程**:

1. 前端发送请求
2. 检索、构建上下文（同场景 1）
3. LLM 生成回复并添加意图标记:
   ```
   好的，我来帮您创建明天下午3点的会议。

   <<<SCHEDULE_INTENT:{"detected":true,"schedule_description":"明天下午3点的会议"}>>>
   ```
4. 解析意图并构建最终响应:
   ```json
   {
     "done": true,
     "schedule_creation_intent": {
       "detected": true,
       "schedule_description": "明天下午3点的会议"
     }
   }
   ```
5. 前端检测到意图，显示日程创建表单

### 5.3 场景 3: 混合查询

**用户查询**: "关于项目进度有什么笔记或安排？"

**执行流程**:

1. 检索笔记和日程
2. 构建混合上下文
3. LLM 生成回复
4. 发送最终响应，包含:
   - sources: `["memos/uid1", "schedules/1"]`
   - schedule_query_result: 相关日程
   - (可能) schedule_creation_intent

---

## 六、测试用例

### 6.1 单元测试

| 测试用例 | 描述 | 预期结果 |
|---------|------|----------|
| 日程时间格式化 | 全天日程 | "2006-01-21" |
| 日程时间格式化 | 有结束时间 | "2006-01-21 14:00 - 15:00" |
| 日程时间格式化 | 无结束时间 | "2006-01-21 14:00" |
| 日程时间格式化 | 带地点 | "2006-01-21 14:00 - 会议 @ 会议室" |
| 上下文构建 | 纯日程查询 | 只包含日程上下文 |
| 上下文构建 | 混合查询 | 包含笔记和日程上下文 |
| 意图解析 | 有意图标记 | 返回意图对象 |
| 意图解析 | 无意图标记 | 返回 nil |
| 响应构建 | 纯日程查询 | 包含 ScheduleQueryResult |
| 响应构建 | 有创建意图 | 包含 ScheduleCreationIntent |

### 6.2 集成测试

| 测试用例 | 描述 | 预期结果 |
|---------|------|----------|
| Connect RPC 响应完整性 | 验证发送了所有响应 | 包含 Done、ScheduleQueryResult |
| gRPC 响应完整性 | 验证响应结构 | 与 Connect RPC 相同 |
| 两个实现一致性 | 相同输入 | 相同输出 |
| 前端解析 | 解析响应数据 | 成功提取日程列表 |

### 6.3 E2E 测试

| 测试用例 | 描述 | 预期结果 |
|---------|------|----------|
| 用户查询今天的日程 | 完整流程 | 前端显示日程列表 |
| 用户创建日程 | 完整流程 | 前端显示创建表单 |
| 混合查询 | 完整流程 | 前端显示笔记和日程 |

---

## 七、迁移计划

### 7.1 阶段 1: 创建公共模块

1. 创建 `chat_common` 包
2. 实现公共接口
3. 编写单元测试

### 7.2 阶段 2: 迁移 gRPC 实现

1. 修改 `ai_service_chat.go` 使用公共模块
2. 验证功能不变
3. 运行现有测试

### 7.3 阶段 3: 迁移 Connect RPC 实现

1. 修改 `connect_handler.go` 使用公共模块
2. 补充缺失的响应逻辑
3. 验证与 gRPC 实现一致

### 7.4 阶段 4: 测试和验证

1. 运行所有测试
2. E2E 测试
3. 性能测试

---

## 八、验收标准

### 8.1 功能验收

- [ ] 两个实现返回相同的响应结构
- [ ] Connect RPC 发送完整的最终响应
- [ ] 前端能正确解析和展示日程
- [ ] 日程创建意图正确传递

### 8.2 一致性验收

- [ ] 相同输入产生相同输出
- [ ] 日程时间格式统一
- [ ] 消息构建逻辑一致

### 8.3 测试验收

- [ ] 单元测试覆盖率 ≥ 80%
- [ ] 所有测试用例通过
- [ ] 零 P0/P1 bug

### 8.4 性能验收

- [ ] 响应时间不受影响（< 5% 增加）
- [ ] 内存使用无明显增长

---

## 九、风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| 公共模块引入 bug | 充分的单元测试，逐步迁移 |
| Connect RPC 响应格式变化 | 保持向后兼容，添加字段而非修改 |
| 前端未准备就绪 | 与前端团队同步，提供清晰的 API 文档 |

---

## 十、相关文档

- [实施方案](../IMPLEMENTATION_PLAN.md)
- [时区统一化](./TIMEZONE_UNIFICATION.md)
- [日程查询优化](./SCHEDULE_QUERY_OPTIMIZATION.md)
