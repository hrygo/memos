# 使用 Schedule 查询

<cite>
**本文档引用的文件**
- [plugin/ai/schedule/helpers.go](file://plugin/ai/schedule/helpers.go)
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go)
- [plugin/ai/schedule/recurrence.go](file://plugin/ai/schedule/recurrence.go)
- [plugin/ai/schedule/timezone_validator.go](file://plugin/ai/schedule/timezone_validator.go)
- [plugin/ai/agent/tools/scheduler.go](file://plugin/ai/agent/tools/scheduler.go)
- [server/router/api/v1/schedule_service.go](file://server/router/api/v1/schedule_service.go)
- [server/service/schedule/service.go](file://server/service/schedule/service.go)
- [proto/api/v1/schedule_service.proto](file://proto/api/v1/schedule_service.proto)
- [web/src/hooks/useScheduleQueries.ts](file://web/src/hooks/useScheduleQueries.ts)
- [web/src/pages/Schedule.tsx](file://web/src/pages/Schedule.tsx)
- [web/src/components/AIChat/ScheduleCalendar.tsx](file://web/src/components/AIChat/ScheduleCalendar.tsx)
- [plugin/ai/schedule/helpers_test.go](file://plugin/ai/schedule/helpers_test.go)
- [plugin/ai/schedule/recurrence_test.go](file://plugin/ai/schedule/recurrence_test.go)
- [plugin/ai/agent/tools/scheduler_test.go](file://plugin/ai/agent/tools/scheduler_test.go)
</cite>

## 目录
1. [简介](#简介)
2. [项目结构](#项目结构)
3. [核心组件](#核心组件)
4. [架构概览](#架构概览)
5. [详细组件分析](#详细组件分析)
6. [依赖关系分析](#依赖关系分析)
7. [性能考虑](#性能考虑)
8. [故障排除指南](#故障排除指南)
9. [结论](#结论)

## 简介

本文档详细介绍了 Memos 项目中的 Schedule 查询系统，这是一个基于自然语言处理的日程管理系统。该系统允许用户通过自然语言描述创建、查询和管理日程事件，支持重复性日程、提醒功能和时区处理。

系统的核心特性包括：
- 自然语言解析和转换为结构化日程数据
- 重复性日程规则的生成和管理
- 冲突检测和预防机制
- 前端优化的查询和显示功能
- 时区感知的时间处理

## 项目结构

```mermaid
graph TB
subgraph "前端层"
Web[React 前端]
Hooks[查询钩子]
Components[UI 组件]
end
subgraph "服务层"
API[API 服务]
Service[业务逻辑服务]
Tools[智能工具]
end
subgraph "插件层"
Parser[日程解析器]
Recurrence[重复规则]
Helpers[辅助函数]
TZ[时区验证器]
end
subgraph "存储层"
Database[(数据库)]
Store[(存储接口)]
end
Web --> API
API --> Service
Service --> Tools
Tools --> Parser
Parser --> Recurrence
Parser --> TZ
Service --> Store
Store --> Database
```

**图表来源**
- [web/src/pages/Schedule.tsx](file://web/src/pages/Schedule.tsx#L1-L196)
- [server/router/api/v1/schedule_service.go](file://server/router/api/v1/schedule_service.go#L1-L826)
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L1-L378)

**章节来源**
- [web/src/pages/Schedule.tsx](file://web/src/pages/Schedule.tsx#L1-L196)
- [server/router/api/v1/schedule_service.go](file://server/router/api/v1/schedule_service.go#L1-L826)
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L1-L378)

## 核心组件

### 日程解析器 (Schedule Parser)

日程解析器是系统的核心组件，负责将自然语言转换为结构化的日程数据。它包含以下关键功能：

- **自然语言处理**: 使用 LLM 服务解析复杂的日程描述
- **时间计算**: 处理相对日期和时间计算
- **重复规则提取**: 从文本中识别和解析重复模式
- **提醒设置**: 提取和解析提醒配置

### 重复规则引擎 (Recurrence Engine)

重复规则引擎提供灵活的日程重复功能：

- **三种重复类型**: 每日、每周、每月
- **间隔控制**: 支持自定义间隔（如每3天、每2周）
- **工作日过滤**: 支持特定工作日的重复
- **实例生成**: 动态生成重复事件的时间戳

### 时区验证器 (Timezone Validator)

时区验证器确保日程在不同时区下的正确性：

- **夏令时处理**: 处理 DST 转换中的边界情况
- **无效时间检测**: 识别和修正不存在的时间
- **模糊时间处理**: 处理重复出现的时间

### 智能工具 (AI Tools)

智能工具集成为用户提供自动化日程管理能力：

- **schedule_query**: 查询现有日程以避免冲突
- **schedule_add**: 创建新日程并处理冲突
- **find_free_time**: 查找可用时间段

**章节来源**
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L1-L378)
- [plugin/ai/schedule/recurrence.go](file://plugin/ai/schedule/recurrence.go#L1-L557)
- [plugin/ai/schedule/timezone_validator.go](file://plugin/ai/schedule/timezone_validator.go#L1-L247)
- [plugin/ai/agent/tools/scheduler.go](file://plugin/ai/agent/tools/scheduler.go#L1-L800)

## 架构概览

```mermaid
sequenceDiagram
participant User as 用户
participant Frontend as 前端界面
participant API as API 服务
participant Service as 业务服务
participant Tools as 智能工具
participant Parser as 解析器
participant Store as 存储层
User->>Frontend : 输入自然语言日程
Frontend->>API : ParseAndCreateSchedule 请求
API->>Service : 验证和处理
Service->>Tools : schedule_query 检查冲突
Tools->>Service : 返回现有日程
Service->>Tools : schedule_add 创建日程
Tools->>Parser : 解析自然语言
Parser->>Parser : 生成重复规则
Parser->>Store : 保存日程数据
Store-->>API : 返回创建结果
API-->>Frontend : 显示日程
```

**图表来源**
- [server/router/api/v1/schedule_service.go](file://server/router/api/v1/schedule_service.go#L654-L723)
- [plugin/ai/agent/tools/scheduler.go](file://plugin/ai/agent/tools/scheduler.go#L132-L266)
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L62-L76)

## 详细组件分析

### 日程解析器实现

日程解析器采用模块化设计，每个组件都有明确的职责：

```mermaid
classDiagram
class Parser {
+LLMService llmService
+Location location
+TimezoneValidator validator
+Parse(ctx, text) ParseResult
+parseWithLLM(ctx, text) ParseResult
}
class ParseResult {
+string Title
+string Description
+string Location
+int64 StartTs
+int64 EndTs
+bool AllDay
+string Timezone
+[]Reminder Reminders
+RecurrenceRule Recurrence
+ToSchedule() Schedule
}
class RecurrenceRule {
+RecurrenceType Type
+int Interval
+[]int Weekdays
+int MonthDay
+Validate() error
+GenerateInstances(startTs, endTs) []int64
+ToJSON() string
}
Parser --> ParseResult
ParseResult --> RecurrenceRule
```

**图表来源**
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L22-L60)
- [plugin/ai/schedule/recurrence.go](file://plugin/ai/schedule/recurrence.go#L42-L47)

#### 时间处理算法

日程解析器实现了复杂的时间处理逻辑：

```mermaid
flowchart TD
Start([开始解析]) --> ValidateInput["验证输入参数"]
ValidateInput --> ParseLLM["调用 LLM 解析"]
ParseLLM --> ParseTime["解析时间字符串"]
ParseTime --> ValidateTime["验证时间有效性"]
ValidateTime --> CheckDST["检查夏令时转换"]
CheckDST --> GenerateResult["生成解析结果"]
GenerateResult --> End([结束])
ValidateTime --> TimeTooPast{"时间是否过旧?"}
TimeTooPast --> |是| Error["返回错误"]
TimeTooPast --> |否| CheckDST
CheckDST --> DSTWarnings{"是否有警告?"}
DSTWarnings --> |是| LogWarning["记录警告"]
DSTWarnings --> |否| GenerateResult
LogWarning --> GenerateResult
```

**图表来源**
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L91-L348)
- [plugin/ai/schedule/timezone_validator.go](file://plugin/ai/schedule/timezone_validator.go#L110-L129)

**章节来源**
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L1-L378)
- [plugin/ai/schedule/timezone_validator.go](file://plugin/ai/schedule/timezone_validator.go#L1-L247)

### 重复规则引擎

重复规则引擎提供了强大的日程重复功能：

```mermaid
classDiagram
class RecurrenceRule {
+RecurrenceType Type
+int Interval
+[]int Weekdays
+int MonthDay
+Validate() error
+GenerateInstances(startTs, endTs) []int64
+Iterator(startTs) RecurrenceIterator
+ToJSON() string
+ParseRecurrenceRuleFromJSON(json) RecurrenceRule
}
class RecurrenceIterator {
+RecurrenceRule rule
+int64 startTs
+[]int64 cache
+int64 cacheEndTs
+bool exhausted
+GetUntil(endTs) []int64
+Next() int64
+CountInRange(startTs, endTs) int
+Reset() void
}
RecurrenceRule --> RecurrenceIterator
```

**图表来源**
- [plugin/ai/schedule/recurrence.go](file://plugin/ai/schedule/recurrence.go#L42-L557)

#### 实例生成算法

重复规则引擎使用高效的算法生成日程实例：

```mermaid
flowchart TD
Start([生成实例]) --> CheckStart{"开始时间有效?"}
CheckStart --> |否| ReturnEmpty["返回空数组"]
CheckStart --> |是| SetEnd["设置结束时间"]
SetEnd --> ChooseType{"选择重复类型"}
ChooseType --> Daily["每日重复"]
ChooseType --> Weekly["每周重复"]
ChooseType --> Monthly["每月重复"]
Daily --> DailyLoop["循环添加实例"]
Weekly --> WeeklyLoop["查找匹配工作日"]
Monthly --> MonthLoop["计算目标日期"]
DailyLoop --> CheckLimit{"检查实例限制"}
WeeklyLoop --> CheckLimit
MonthLoop --> CheckLimit
CheckLimit --> |达到限制| ReturnResult["返回结果"]
CheckLimit --> |未达到| AddInstance["添加实例"]
AddInstance --> DailyLoop
AddInstance --> WeeklyLoop
AddInstance --> MonthLoop
```

**图表来源**
- [plugin/ai/schedule/recurrence.go](file://plugin/ai/schedule/recurrence.go#L151-L282)

**章节来源**
- [plugin/ai/schedule/recurrence.go](file://plugin/ai/schedule/recurrence.go#L1-L557)

### 前端查询系统

前端提供了优化的查询和显示功能：

```mermaid
sequenceDiagram
participant User as 用户
participant Hook as 查询钩子
participant API as API 服务
participant Cache as 缓存系统
participant UI as UI 组件
User->>Hook : 设置查询范围
Hook->>Cache : 检查缓存
Cache-->>Hook : 返回缓存数据或空
Hook->>API : 发送查询请求
API-->>Hook : 返回日程数据
Hook->>Cache : 更新缓存
Hook->>UI : 渲染日程列表
UI->>User : 显示日历视图
```

**图表来源**
- [web/src/hooks/useScheduleQueries.ts](file://web/src/hooks/useScheduleQueries.ts#L243-L265)
- [web/src/pages/Schedule.tsx](file://web/src/pages/Schedule.tsx#L33-L35)

#### 查询优化策略

前端查询系统采用了多种优化策略：

- **时间范围优化**: 默认查询当前日期前后15天的数据
- **缓存管理**: 合理的缓存策略避免不必要的网络请求
- **分页处理**: 支持大数据量的分页查询
- **实时更新**: 自动刷新最新的日程数据

**章节来源**
- [web/src/hooks/useScheduleQueries.ts](file://web/src/hooks/useScheduleQueries.ts#L1-L593)
- [web/src/pages/Schedule.tsx](file://web/src/pages/Schedule.tsx#L1-L196)

## 依赖关系分析

```mermaid
graph TB
subgraph "外部依赖"
LLM[LLM 服务]
Proto[Protobuf 定义]
React[React Query]
end
subgraph "内部模块"
Parser[日程解析器]
Tools[智能工具]
Service[业务服务]
API[API 服务]
Store[存储层]
end
subgraph "测试模块"
ParserTest[解析器测试]
RecurrenceTest[重复规则测试]
ToolTest[工具测试]
end
LLM --> Parser
Proto --> API
React --> Tools
Parser --> Service
Tools --> Service
Service --> API
API --> Store
ParserTest --> Parser
RecurrenceTest --> Parser
ToolTest --> Tools
```

**图表来源**
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L12-L14)
- [proto/api/v1/schedule_service.proto](file://proto/api/v1/schedule_service.proto#L1-L166)

### 关键依赖关系

系统的关键依赖关系包括：

1. **LLM 服务集成**: 日程解析依赖于外部 LLM 服务进行自然语言处理
2. **Protobuf 协议**: 前后端通信使用标准化的 Protobuf 接口
3. **React Query 缓存**: 前端查询状态管理
4. **PostgreSQL 存储**: 数据持久化层

**章节来源**
- [plugin/ai/schedule/parser.go](file://plugin/ai/schedule/parser.go#L1-L378)
- [proto/api/v1/schedule_service.proto](file://proto/api/v1/schedule_service.proto#L1-L166)

## 性能考虑

### 查询性能优化

系统在多个层面进行了性能优化：

1. **前端缓存策略**: 使用 React Query 的智能缓存避免重复请求
2. **实例数量限制**: 限制重复日程实例的数量防止内存溢出
3. **批量查询**: 支持批量查询减少网络往返次数
4. **懒加载迭代器**: 使用迭代器模式处理大量重复实例

### 内存管理

```mermaid
flowchart TD
Start([开始处理]) --> CheckMemory["检查可用内存"]
CheckMemory --> HasMemory{"内存充足?"}
HasMemory --> |是| ProcessData["处理数据"]
HasMemory --> |否| Cleanup["清理缓存"]
Cleanup --> CheckMemory
ProcessData --> Optimize["优化数据结构"]
Optimize --> End([完成])
```

**图表来源**
- [plugin/ai/schedule/recurrence.go](file://plugin/ai/schedule/recurrence.go#L341-L363)

## 故障排除指南

### 常见问题及解决方案

#### 日程冲突检测

当创建日程时遇到冲突，系统会返回详细的冲突信息：

1. **冲突检测失败**: 检查时间范围和时区设置
2. **重复冲突**: 使用 `checkRecurringConflicts` 方法检查重复日程
3. **时区问题**: 确保所有时间都转换为 UTC 格式

#### 解析器错误

解析自然语言时可能遇到的问题：

1. **输入格式错误**: 确保输入符合预期格式
2. **LLM 服务不可用**: 检查 LLM 服务连接状态
3. **时间解析失败**: 验证时间字符串的格式

#### 前端查询问题

前端查询可能出现的问题：

1. **数据不同步**: 使用 `invalidateQueries` 刷新缓存
2. **时间显示错误**: 检查时区转换逻辑
3. **性能问题**: 调整查询范围和缓存策略

**章节来源**
- [plugin/ai/schedule/helpers_test.go](file://plugin/ai/schedule/helpers_test.go#L1-L181)
- [plugin/ai/schedule/recurrence_test.go](file://plugin/ai/schedule/recurrence_test.go#L1-L373)
- [plugin/ai/agent/tools/scheduler_test.go](file://plugin/ai/agent/tools/scheduler_test.go#L1-L273)

## 结论

Memos 的 Schedule 查询系统是一个功能完整、设计合理的日程管理系统。它通过以下特点实现了优秀的用户体验：

1. **自然语言处理**: 允许用户用自然语言描述日程
2. **智能冲突检测**: 自动检测和解决日程冲突
3. **灵活的重复规则**: 支持复杂的重复模式
4. **前端优化**: 提供流畅的用户交互体验
5. **时区感知**: 正确处理不同时区的时间

该系统的设计充分考虑了可扩展性和维护性，为未来的功能扩展奠定了良好的基础。通过模块化的架构设计和完善的测试覆盖，系统能够稳定地处理各种复杂的日程管理场景。