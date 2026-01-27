# Amazing Parrot 综合助手代理

<cite>
**本文档引用的文件**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go)
- [scheduler_v2.go](file://plugin/ai/agent/scheduler_v2.go)
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go)
- [cache.go](file://plugin/ai/agent/cache.go)
- [types.go](file://plugin/ai/agent/types.go)
- [base_tool.go](file://plugin/ai/agent/base_tool.go)
- [tool_adapter.go](file://plugin/ai/agent/tool_adapter.go)
- [util.go](file://plugin/ai/agent/util.go)
- [error_class.go](file://plugin/ai/agent/error_class.go)
- [context.go](file://plugin/ai/agent/context.go)
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go)
- [llm_intent_classifier.go](file://plugin/ai/agent/llm_intent_classifier.go)
- [memo_search.go](file://plugin/ai/agent/tools/memo_search.go)
- [scheduler.go](file://plugin/ai/agent/tools/scheduler.go)
- [README.md](file://README.md)
- [SPEC-003-AGENT-AMAZING.md](file://docs/specs/SPEC-003-AGENT-AMAZING.md)
- [parrot.ts](file://web/src/types/parrot.ts)
- [useCapabilityRouter.ts](file://web/src/hooks/useCapabilityRouter.ts)
</cite>

## 更新摘要
**所做更改**
- 更新了意图分类器优化部分，反映 isUpdateIntent 函数移除多余参数的变更
- 新增了主题统一化变更说明，反映前端鹦鹉主题配置的统一设计语言
- 更新了前端路由逻辑变更，反映意图识别已迁移至后端的架构调整
- 完善了综合助手代理的主题配置和视觉设计说明

## 目录
1. [简介](#简介)
2. [项目结构](#项目结构)
3. [核心组件](#核心组件)
4. [架构总览](#架构总览)
5. [详细组件分析](#详细组件分析)
6. [依赖关系分析](#依赖关系分析)
7. [性能考虑](#性能考虑)
8. [故障排查指南](#故障排查指南)
9. [结论](#结论)
10. [附录](#附录)

## 简介
Amazing Parrot（惊奇）是 Memos 多代理系统中的中枢协调者，负责两阶段并发检索：意图分析阶段与并发执行阶段。它结合笔记检索与日程查询能力，通过 LLM 规划检索计划，使用并发工具调用获取结果，并最终进行答案合成，为用户提供一站式信息助手体验。

**更新** 本版本反映了 Parrot 代理系统主题统一化变更，采用统一设计语言替代原有的特定代理主题配置，同时体现了意图分类器优化，移除了 isUpdateIntent 函数中的多余参数。

## 项目结构
- 后端采用 Go + Echo + Connect RPC 架构，前端使用 React + Vite。
- AI 代理层位于 plugin/ai/agent，包含多代理实现与工具集。
- 查询路由与自适应检索位于 server/queryengine 与 server/retrieval，支撑 RAG 管线。
- 存储层使用 PostgreSQL（生产），Redis 可选作为二级缓存。

```mermaid
graph TB
subgraph "前端"
UI["React 前端<br/>AI 聊天 + 鹦鹉枢纽"]
end
subgraph "后端"
API["API 层<br/>Connect RPC"]
AGENT["代理层<br/>AmazingParrot / MemoParrot / ScheduleParrot"]
RETRIEVAL["检索层<br/>QueryRouter + AdaptiveRetriever"]
STORE["存储层<br/>PostgreSQL + Redis"]
end
UI --> API
API --> AGENT
AGENT --> RETRIEVAL
RETRIEVAL --> STORE
```

**图表来源**
- [README.md](file://README.md#L157-L198)

**章节来源**
- [README.md](file://README.md#L1-L365)

## 核心组件
- AmazingParrot：中枢协调者，实现两阶段并发检索与答案合成。
- MemoParrot：笔记检索代理，采用 ReAct 循环与工具调用。
- ScheduleParrotV2：日程管理代理，基于原生 LLM 工具调用框架。
- 工具集：memo_search、schedule_query、schedule_add、find_free_time 等。
- 缓存系统：LRU 缓存，支持 TTL 与命中率统计。
- 错误分类：将错误分为瞬时、永久与冲突三类，指导重试与处理策略。
- 事件与 UI：统一事件类型，支持前端生成式 UI（UI 事件）。
- **新增** 意图分类器：优化后的分类器，移除了 isUpdateIntent 函数的多余参数，提升了代码简洁性和维护性。

**章节来源**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L19-L92)
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go#L26-L66)
- [scheduler_v2.go](file://plugin/ai/agent/scheduler_v2.go#L16-L91)
- [cache.go](file://plugin/ai/agent/cache.go#L10-L74)
- [error_class.go](file://plugin/ai/agent/error_class.go#L17-L82)
- [types.go](file://plugin/ai/agent/types.go#L10-L139)
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go#L166-L174)

## 架构总览
AmazingParrot 的工作流分为四个阶段：
1. 缓存检查：命中则直接返回，未命中进入规划阶段。
2. 意图分析与检索计划：使用 LLM 生成检索计划（memo_search、schedule_query、find_free_time、direct_answer）。
3. 并发检索执行：按计划并发调用工具，收集结构化结果。
4. 答案合成：将检索结果注入提示词，流式输出最终答案。

```mermaid
sequenceDiagram
participant U as "用户"
participant AP as "AmazingParrot"
participant L as "LLM"
participant MT as "MemoSearchTool"
participant ST as "ScheduleQueryTool/FindFreeTimeTool"
U->>AP : "输入 + 历史"
AP->>AP : "缓存检查"
AP->>L : "意图分析 + 规划检索"
L-->>AP : "检索计划"
AP->>MT : "并发执行memo_search"
AP->>ST : "并发执行schedule_query/find_free_time"
MT-->>AP : "结构化结果"
ST-->>AP : "结构化结果"
AP->>L : "答案合成流式"
L-->>AP : "答案片段"
AP-->>U : "流式答案"
```

**图表来源**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L106-L184)
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L186-L225)
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L227-L387)
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L389-L451)

**章节来源**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L100-L184)

## 详细组件分析

### AmazingParrot 组件分析
- 结构与职责
  - 持有 LLM、缓存、MemoSearchTool、ScheduleQueryTool、ScheduleAddTool、FindFreeTimeTool、ScheduleUpdateTool。
  - 提供 ExecuteWithCallback、planRetrieval、executeConcurrentRetrieval、synthesizeAnswer、SelfDescribe 等方法。
- 两阶段并发检索
  - 规划阶段：构建系统提示词，附加历史消息，调用 LLM 生成检索计划。
  - 执行阶段：并发调用工具，使用互斥锁保护回调与结果写入，支持 UI 事件（memo_query_result、schedule_query_result）。
  - 合成阶段：将检索结果注入提示词，流式输出最终答案。
- 缓存策略
  - 使用 GenerateCacheKey（SHA256 哈希）避免长输入导致内存问题。
  - 支持 TTL 与命中率统计，便于监控与优化。
- 错误处理
  - 包装 ParrotError，记录阶段与操作。
  - 错误分类（瞬时/永久/冲突），指导重试与处理。

```mermaid
classDiagram
class AmazingParrot {
-llm : LLMService
-cache : LRUCache
-userID : int32
-memoSearchTool : MemoSearchTool
-scheduleQueryTool : ScheduleQueryTool
-scheduleAddTool : ScheduleAddTool
-findFreeTimeTool : FindFreeTimeTool
-scheduleUpdateTool : ScheduleUpdateTool
+ExecuteWithCallback(ctx, input, history, callback) error
+planRetrieval(ctx, input, history, callback) *retrievalPlan
+executeConcurrentRetrieval(ctx, plan, callback) map[string]string
+synthesizeAnswer(ctx, input, history, results, callback) string
+SelfDescribe() ParrotSelfCognition
+GetStats() CacheStats
}
```

**图表来源**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L22-L92)

**章节来源**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L19-L92)
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L186-L225)
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L227-L387)
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L389-L451)
- [cache.go](file://plugin/ai/agent/cache.go#L10-L74)
- [types.go](file://plugin/ai/agent/types.go#L334-L342)
- [error_class.go](file://plugin/ai/agent/error_class.go#L214-L231)

### MemoParrot 组件分析
- 工作模式：ReAct 循环（先搜索，后回答），最终答案采用流式输出提升用户体验。
- 工具调用：识别 LLM 输出中的 TOOL/INPUT 标记，调用 memo_search 工具。
- 缓存：独立缓存，支持命中率统计。
- 自我描述：提供元认知自我理解，包含鸟类身份、情感表达、个性与能力等。

```mermaid
flowchart TD
Start(["开始"]) --> BuildPrompt["构建系统提示词"]
BuildPrompt --> Loop{"迭代次数 < 最大限制?"}
Loop --> |否| FinalAnswer["流式输出最终答案"]
Loop --> |是| LLM["LLM 推理"]
LLM --> Parse{"解析到工具调用?"}
Parse --> |是| ToolCall["执行工具调用"]
ToolCall --> Append["追加工具结果到历史"]
Append --> Loop
Parse --> |否| FinalAnswer
FinalAnswer --> End(["结束"])
```

**图表来源**
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go#L139-L289)

**章节来源**
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go#L26-L66)
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go#L139-L289)
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go#L291-L332)

### ScheduleParrotV2 组件分析
- 原生 LLM 工具调用：基于 Agent/ToolWithSchema 框架，无需 LangChainGo 依赖。
- UI 事件：在工具调用过程中注入 UI 事件（如 ui_schedule_suggestion、ui_conflict_resolution），支持生成式 UI。
- 冲突处理：解析工具结果，检测冲突并提供替代时间槽或处理建议。

```mermaid
sequenceDiagram
participant U as "用户"
participant SP as "ScheduleParrotV2"
participant A as "Agent"
participant L as "LLM"
participant T as "工具集"
U->>SP : "输入 + 会话上下文"
SP->>A : "RunWithCallback"
A->>L : "ChatWithTools"
L-->>A : "内容 + 工具调用"
A->>T : "执行工具"
T-->>A : "工具结果"
A-->>SP : "事件回调tool_use/tool_result/answer"
SP-->>U : "UI 事件ui_schedule_suggestion/..."
```

**图表来源**
- [scheduler_v2.go](file://plugin/ai/agent/scheduler_v2.go#L175-L196)
- [tool_adapter.go](file://plugin/ai/agent/tool_adapter.go#L129-L207)

**章节来源**
- [scheduler_v2.go](file://plugin/ai/agent/scheduler_v2.go#L16-L91)
- [scheduler_v2.go](file://plugin/ai/agent/scheduler_v2.go#L175-L196)
- [tool_adapter.go](file://plugin/ai/agent/tool_adapter.go#L79-L117)
- [tool_adapter.go](file://plugin/ai/agent/tool_adapter.go#L129-L207)

### 意图分类器优化
**更新** 意图分类器经过优化，移除了 isUpdateIntent 函数中的多余参数，提升了代码简洁性和可维护性。

- 分类器类型：TaskIntent，包含 simple_create、simple_query、simple_update、batch_create、conflict_resolve、multi_query 六种意图类型。
- 分类逻辑：优先检查批量操作，然后检查更新意图，再检查查询意图，最后默认为创建意图。
- **优化后** isUpdateIntent 函数签名简化为 `(ic *IntentClassifier) isUpdateIntent(_, lowerInput string) bool`，移除了第一个 input 参数。

```mermaid
flowchart TD
Input["用户输入"] --> Classify["IntentClassifier.Classify"]
Classify --> Batch{"批量意图?"}
Batch --> |是| BatchCreate["IntentBatchCreate"]
Batch --> |否| Update{"更新意图?"}
Update --> |是| SimpleUpdate["IntentSimpleUpdate"]
Update --> |否| Query{"查询意图?"}
Query --> |是| SimpleQuery["IntentSimpleQuery"]
Query --> |否| SimpleCreate["IntentSimpleCreate"]
```

**图表来源**
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go#L104-L125)

**章节来源**
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go#L31-L47)
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go#L104-L125)
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go#L166-L174)

### 工具与适配器
- 工具接口：Tool/ToolWithSchema，支持参数 Schema（JSON）。
- 基础工具：BaseTool 提供超时、校验、执行封装。
- 工具适配：NativeTool/ToolFromLegacy 将现有工具适配到新框架。
- 工具集：
  - MemoSearchTool：语义/关键词检索，支持结构化结果。
  - ScheduleQueryTool：按时间范围查询日程，支持结构化结果。
  - ScheduleAddTool：创建日程，内置冲突检测与自动调整。
  - FindFreeTimeTool：查找可用空闲时间（8:00-22:00）。

```mermaid
classDiagram
class Tool {
<<interface>>
+Name() string
+Description() string
+Run(ctx, input) string
}
class ToolWithSchema {
<<interface>>
+Parameters() map[string]interface{}
}
class BaseTool {
-name : string
-description : string
-execute : func
-validate : func
-timeout : time.Duration
+Run(ctx, input) string
}
class NativeTool {
-name : string
-description : string
-execute : func
-params : map[string]interface{}
+Parameters() map[string]interface{}
+Run(ctx, input) string
}
Tool <|.. BaseTool
ToolWithSchema <|.. NativeTool
```

**图表来源**
- [base_tool.go](file://plugin/ai/agent/base_tool.go#L10-L135)
- [tool_adapter.go](file://plugin/ai/agent/tool_adapter.go#L12-L77)

**章节来源**
- [base_tool.go](file://plugin/ai/agent/base_tool.go#L10-L135)
- [tool_adapter.go](file://plugin/ai/agent/tool_adapter.go#L12-L77)
- [memo_search.go](file://plugin/ai/agent/tools/memo_search.go#L53-L77)
- [scheduler.go](file://plugin/ai/agent/tools/scheduler.go#L132-L144)

### 缓存与错误分类
- 缓存：LRU + TTL，支持命中/未命中统计，提供 Clear/Size/Stats 等操作。
- 错误分类：瞬时（网络/超时）、永久（校验/权限）、冲突（日程冲突）三类，支持重试延时与动作提示。

```mermaid
flowchart TD
E["错误发生"] --> Classify["分类错误"]
Classify --> Transient{"瞬时错误?"}
Transient --> |是| Retry["建议重试含延时"]
Transient --> |否| Permanent{"永久错误?"}
Permanent --> |是| Fail["终止并返回"]
Permanent --> |否| Conflict{"冲突错误?"}
Conflict --> |是| Action["提供动作提示如 find_free_time"]
Conflict --> |否| Default["默认永久错误"]
```

**图表来源**
- [error_class.go](file://plugin/ai/agent/error_class.go#L84-L149)

**章节来源**
- [cache.go](file://plugin/ai/agent/cache.go#L10-L74)
- [cache.go](file://plugin/ai/agent/cache.go#L178-L197)
- [error_class.go](file://plugin/ai/agent/error_class.go#L17-L82)
- [error_class.go](file://plugin/ai/agent/error_class.go#L214-L231)

### 主题统一化变更
**新增** Parrot 代理系统进行了主题统一化变更，移除了特定代理主题配置，采用统一设计语言。

- 统一主题配置：所有代理共享相同的主题配置结构，包括气泡背景、文本颜色、图标样式等。
- 设计语言统一：采用一致的颜色系统和视觉元素，确保跨代理的一致用户体验。
- 主题映射：MEMO、SCHEDULE、AMAZING 三种代理现在使用统一的主题配置，避免了重复定义。

**章节来源**
- [parrot.ts](file://web/src/types/parrot.ts#L294-L360)

### 前端路由逻辑变更
**更新** 前端路由逻辑已迁移至后端，前端不再进行意图识别，始终返回 AUTO。

- 后端路由：意图识别逻辑已移至后端 ChatRouter，使用规则+LLM 混合方式进行更准确的意图识别。
- 前端辅助：前端仅提供 UI 辅助函数，包括能力信息获取、类型转换等功能。
- 兼容性：前端仍保留路由函数，但实际路由逻辑由后端处理，确保向后兼容。

**章节来源**
- [useCapabilityRouter.ts](file://web/src/hooks/useCapabilityRouter.ts#L6-L20)
- [useCapabilityRouter.ts](file://web/src/hooks/useCapabilityRouter.ts#L29-L100)

## 依赖关系分析
- 组件耦合
  - AmazingParrot 依赖 LLM、缓存、MemoSearchTool、ScheduleQueryTool、ScheduleAddTool、FindFreeTimeTool、ScheduleUpdateTool。
  - MemoParrot 依赖 LLM、AdaptiveRetriever、MemoSearchTool。
  - ScheduleParrotV2 依赖 LLM、ScheduleQueryTool、ScheduleAddTool、FindFreeTimeTool。
- 外部依赖
  - LLMService：提供 Chat/ChatStream/ChatWithTools 等能力。
  - AdaptiveRetriever：提供检索能力（BM25 + 向量 + 重排序）。
  - schedule.Service：提供日程查询/创建/更新/冲突检测。
- 事件与 UI
  - 统一事件类型（thinking/tool_use/tool_result/answer/...），支持 UI 事件（ui_schedule_suggestion/ui_conflict_resolution 等）。

```mermaid
graph TB
AP["AmazingParrot"] --> LLM["LLMService"]
AP --> Cache["LRUCache"]
AP --> MTool["MemoSearchTool"]
AP --> SQuery["ScheduleQueryTool"]
AP --> SAdd["ScheduleAddTool"]
AP --> SFree["FindFreeTimeTool"]
AP --> SUpd["ScheduleUpdateTool"]
MP["MemoParrot"] --> LLM
MP --> Retriever["AdaptiveRetriever"]
MP --> MTool
SP["ScheduleParrotV2"] --> LLM
SP --> Svc["schedule.Service"]
SP --> SQuery
SP --> SAdd
SP --> SFree
```

**图表来源**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L22-L92)
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go#L28-L66)
- [scheduler_v2.go](file://plugin/ai/agent/scheduler_v2.go#L18-L91)

**章节来源**
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L22-L92)
- [memo_parrot.go](file://plugin/ai/agent/memo_parrot.go#L28-L66)
- [scheduler_v2.go](file://plugin/ai/agent/scheduler_v2.go#L18-L91)

## 性能考虑
- 并发检索：使用 goroutine 并发执行多个工具，显著降低端到端延迟。
- 缓存优化：两层缓存（应用层 LRU + 可选 Redis），命中率高时可减少 LLM 与数据库压力。
- 流式输出：最终答案采用流式输出，提升用户体验。
- 超时与重试：工具层与代理层均设置超时，错误分类指导重试策略。
- 查询路由：QueryRouter 基于意图选择最优检索策略，减少无关计算。
- **优化** 意图分类器优化：移除 isUpdateIntent 函数多余参数，减少了函数调用开销，提升了分类效率。

## 故障排查指南
- 常见错误类型
  - 瞬时错误：网络波动、服务不可达、超时。建议短暂延时后重试。
  - 永久错误：输入校验失败、权限不足、资源不存在。需修正输入或权限。
  - 冲突错误：日程冲突。建议使用 find_free_time 或调整时间。
- 排查步骤
  - 检查缓存命中情况（GetStats）与缓存键（GenerateCacheKey）。
  - 查看事件回调（thinking/tool_use/tool_result/answer）定位问题阶段。
  - 分析 LLM 响应与工具调用是否符合预期。
  - 检查数据库连接、索引与检索策略配置。
  - **新增** 检查意图分类器：确认 isUpdateIntent 函数正确识别更新意图。
- 相关实现参考
  - 错误分类与重试：ClassifyError/ShouldRetry/GetRetryDelay/GetActionHint。
  - 缓存统计：LRUCache.Stats。
  - 事件常量：EventType*。
  - **新增** 主题配置：检查统一主题配置是否正确应用到所有代理。

**章节来源**
- [error_class.go](file://plugin/ai/agent/error_class.go#L84-L149)
- [error_class.go](file://plugin/ai/agent/error_class.go#L214-L231)
- [cache.go](file://plugin/ai/agent/cache.go#L178-L197)
- [types.go](file://plugin/ai/agent/types.go#L117-L139)
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go#L166-L174)
- [parrot.ts](file://web/src/types/parrot.ts#L294-L360)

## 结论
Amazing Parrot 通过"意图分析 + 并发执行 + 答案合成"的两阶段架构，实现了笔记与日程的跨域协同检索。配合完善的缓存、错误分类与 UI 事件体系，为用户提供了高效、稳定且体验友好的综合信息助手。**更新** 最新的主题统一化变更和意图分类器优化进一步提升了系统的整体质量和用户体验。后续可进一步引入 Reranker 与更精细的路由策略，持续优化检索精度与响应速度。

## 附录

### 使用示例与配置选项
- 使用示例
  - 直接问答：输入"今天有空吗"，代理将自动规划并查询日程。
  - 笔记检索：输入"Python 笔记"，代理将并发检索并合成答案。
  - 混合查询：输入"查找项目相关笔记并查看下周会议"，代理将并行执行两类检索。
- 配置选项
  - 缓存大小与 TTL：可通过构造函数参数调整（默认 100 条，5 分钟）。
  - 工具超时：工具层默认 30 秒，可根据场景调整。
  - 事件回调：支持实时进度反馈与 UI 事件推送。
  - 自我描述：SelfDescribe 提供元认知信息，便于前端展示与交互。
  - **新增** 主题配置：统一的视觉设计，支持三种代理的统一主题风格。
  - **新增** 意图识别：优化后的分类器，提供更准确的意图判断。

**章节来源**
- [SPEC-003-AGENT-AMAZING.md](file://docs/specs/SPEC-003-AGENT-AMAZING.md#L1-L100)
- [README.md](file://README.md#L47-L106)
- [amazing_parrot.go](file://plugin/ai/agent/amazing_parrot.go#L49-L92)
- [cache.go](file://plugin/ai/agent/cache.go#L52-L74)
- [base_tool.go](file://plugin/ai/agent/base_tool.go#L38-L53)
- [types.go](file://plugin/ai/agent/types.go#L117-L139)
- [parrot.ts](file://web/src/types/parrot.ts#L294-L360)
- [intent_classifier.go](file://plugin/ai/agent/intent_classifier.go#L166-L174)