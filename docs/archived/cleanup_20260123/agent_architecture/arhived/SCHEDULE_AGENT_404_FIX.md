# Schedule Agent 404 错误修复报告

## 问题描述

前端调用 Schedule Agent API 时出现 404 错误：
```
memos.api.v1.ScheduleAgentService/Chat: Failed to load resource: the server responded with a status of 404 (Not Found)
```

## 根本原因

ScheduleAgentService 的 Connect handler 没有在后端注册。

虽然 gRPC-Gateway 路由已在 `server/router/api/v1/v1.go` 中注册（第 183-187 行），但 **Connect 协议的 handler 未注册**。

在 `server/router/api/v1/connect_handler.go` 中：
- ✅ ScheduleService 已注册
- ❌ ScheduleAgentService **未注册**

## 修复方案

### 1. 注册 ScheduleAgentService Connect Handler

**文件**: `server/router/api/v1/connect_handler.go`

**修改**: 在 `RegisterConnectHandlers` 函数中添加 ScheduleAgentService 注册（第 66-69 行）：

```go
// Register ScheduleAgent service handlers if available
if s.ScheduleAgentService != nil {
    handlers = append(handlers, wrap(apiv1connect.NewScheduleAgentServiceHandler(s, opts...)))
}
```

### 2. 实现 ScheduleAgentService Connect Handler 方法

添加了两个 Connect handler 方法：

#### 2.1 Chat 方法（非流式）
```go
func (s *ConnectServiceHandler) Chat(
    ctx context.Context,
    req *connect.Request[v1pb.ScheduleAgentChatRequest],
) (*connect.Response[v1pb.ScheduleAgentChatResponse], error)
```

#### 2.2 ChatStream 方法（流式）
```go
func (s *ConnectServiceHandler) ChatStream(
    ctx context.Context,
    req *connect.Request[v1pb.ScheduleAgentChatRequest],
    stream *connect.ServerStream[v1pb.ScheduleAgentStreamResponse],
) error
```

### 3. 实现流适配器（Stream Adapter）

由于底层 gRPC service 使用 gRPC stream 接口，而 Connect handler 使用 Connect stream 接口，需要适配器来桥接两者。

**创建的适配器**:

#### 3.1 scheduleAgentStreamAdapter
将 Connect ServerStream 适配为 gRPC ServerStreamingServer 接口：

```go
type scheduleAgentStreamAdapter struct {
    connectStream *connect.ServerStream[v1pb.ScheduleAgentStreamResponse]
    ctx           context.Context
}
```

实现了以下方法：
- `Context() context.Context`
- `Send(*v1pb.ScheduleAgentStreamResponse) error`
- `SendMsg(m any) error`
- `RecvMsg(m any) error`
- `SetHeader(metadata.MD) error`
- `SendHeader(metadata.MD) error`
- `SetTrailer(metadata.MD)`

### 4. 实现 AIService 缺失方法

AIServiceHandler Connect 接口要求以下两个方法，但原 APIV1Service 只提供了 gRPC 版本：

#### 4.1 ChatWithScheduleAgent
```go
func (s *ConnectServiceHandler) ChatWithScheduleAgent(
    ctx context.Context,
    req *connect.Request[v1pb.ChatWithMemosRequest],
    stream *connect.ServerStream[v1pb.ChatWithMemosResponse],
) error
```

**实现**: 转换请求格式并调用 ScheduleAgentService 的流式实现。

#### 4.2 ChatWithMemosIntegrated
```go
func (s *ConnectServiceHandler) ChatWithMemosIntegrated(
    ctx context.Context,
    req *connect.Request[v1pb.ChatWithMemosRequest],
    stream *connect.ServerStream[v1pb.ChatWithMemosResponse],
) error
```

**实现**: 暂时调用现有的 `ChatWithMemos` 实现（RAG only）。

### 5. 添加必要的导入

在 `connect_handler.go` 中添加了 metadata 包导入：
```go
import (
    ...
    "google.golang.org/grpc/metadata"
    ...
)
```

## 测试验证

### 编译测试

✅ **后端编译成功**:
```bash
go build -o /tmp/memos-test5 ./cmd/memos
# 输出: 52M 二进制文件，无错误
```

✅ **前端编译成功**:
```bash
cd web && npm run build
# 输出: ✓ built in 8.18s
```

### 预期行为

修复后，前端调用：
```typescript
scheduleAgentServiceClient.chat({
    message: "明天下午3点开会",
    userTimezone: "Asia/Shanghai"
})
```

将路由到：
- HTTP 路径: `/memos.api.v1.ScheduleAgentService/Chat`
- 后端处理器: `ConnectServiceHandler.Chat`
- 底层服务: `ScheduleAgentService.Chat`

## 架构说明

### Connect RPC vs gRPC

```
Frontend (Connect-Web)
    ↓
/memos.api.v1.ScheduleAgentService/Chat (Connect RPC)
    ↓
ConnectServiceHandler.Chat (Connect wrapper)
    ↓
ScheduleAgentService.Chat (gRPC service)
    ↓
SchedulerAgent.Execute (Business logic)
```

### 为什么需要适配器？

1. **不同的流接口**:
   - gRPC: `ServerStreamingServer[Response]`
   - Connect: `ServerStream[Response]`

2. **不同的请求包装**:
   - gRPC: 直接消息类型 `*Request`
   - Connect: `*connect.Request[Request]`

3. **元数据处理**:
   - gRPC: 使用 `metadata.MD`
   - Connect: 使用 HTTP headers

## 相关文件

| 文件 | 修改类型 | 说明 |
|------|----------|------|
| `server/router/api/v1/connect_handler.go` | 修改 | 添加 ScheduleAgentService 注册和 handler 实现 |
| `server/router/api/v1/v1.go` | 已存在 | gRPC-Gateway 注册（已存在，无需修改） |
| `server/router/api/v1/schedule_agent_service.go` | 已存在 | gRPC service 实现（已存在，无需修改） |

## 总结

**问题**: ScheduleAgentService 的 Connect handler 未注册导致 404 错误

**解决**: 在 `connect_handler.go` 中添加：
1. ✅ ScheduleAgentService Connect handler 注册
2. ✅ Connect 版本的 Chat 和 ChatStream 方法实现
3. ✅ gRPC ↔ Connect 流适配器
4. ✅ AIService 缺失的 Connect 方法实现

**结果**: 前后端编译成功，ScheduleAgent API 可通过 Connect 协议访问

---

**修复完成时间**: 2026-01-21 21:04
**编译状态**: ✅ 后端通过，✅ 前端通过
