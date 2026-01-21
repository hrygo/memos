# SPEC-001: 日程服务接口抽象 (Schedule Service Interface Abstraction)

## 1. 目标 (Goal)
将“日程管理”的核心业务逻辑从 gRPC/HTTP 处理层 (`server/service/schedule_service.go`) 中解耦。
这使得后续的 `ScheduleAgent` 工具能够直接调用业务逻辑（进程内调用），而无需进行内部网络回调或重复代码。

## 2. 方法 (Approach)
提取一个明确的 Go 接口 `ScheduleCoreService`，封装所有日程操作。

## 3. 接口定义 (Interface Definition)
**位置**: `server/service/schedule/interface.go` (重构目标)

```go
type Service interface {
    // FindSchedules 返回开始和结束时间之间的日程
    FindSchedules(ctx context.Context, userID int32, start, end time.Time) ([]*store.Schedule, error)
    
    // CreateSchedule 创建一个新的日程（包含校验逻辑）
    CreateSchedule(ctx context.Context, userID int32, create *CreateScheduleRequest) (*store.Schedule, error)
    
    // UpdateSchedule 更新现有的日程
    UpdateSchedule(ctx context.Context, userID int32, id int32, update *UpdateScheduleRequest) (*store.Schedule, error)
    
    // DeleteSchedule 删除日程
    DeleteSchedule(ctx context.Context, userID int32, id int32) error
}
```

## 4. 需求 (Requirements)
*   **R1**: 服务方法签名必须使用领域对象（或 Store 对象），而不是 Proto 对象，以避免与 API 层紧密耦合。
*   **R2**: 输入校验（时间范围检查、最大长度等）必须从 Handler 层下沉到此 Service 层。
*   **R3**: 权限检查（userID 是否拥有此日程？）必须在此层强制执行。
*   **R4 (关键点)**: **周期性日程展开逻辑 (Recurrence Expansion Logic)** (当前位于 `server/router/api/v1/schedule_service.go` 的 274-360 行) **必须**移动到 `FindSchedules` 中。智能体需要看到具体的会议实例（例如：“下周一9点的会议”），而不仅仅是抽象的重复规则。

## 5. 验收标准 (Acceptance Criteria)
*   [ ] 在独立包（可能是 `server/service/schedule`）中定义了新的 `Service` 接口。
*   [ ] 现有的 gRPC Handler (`server/service/schedule_service.go`) 被重构为调用此接口。
*   [ ] 为 `Service` 实现添加了单元测试，以验证独立于 HTTP 上下文的校验逻辑。
