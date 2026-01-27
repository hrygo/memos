# S0-interface-contract: 公共服务接口契约

> **状态**: 🔲 待开发  
> **优先级**: P0 (核心)  
> **投入**: 5 人天  
> **负责团队**: 团队 A（定义）+ 团队 B/C（验证）  
> **Sprint**: Sprint 0

---

## 1. 目标与背景

### 1.1 核心目标

定义团队 A 提供的 7 个公共服务接口和公共数据模型，消除团队 B/C 并行开发的阻塞。

### 1.2 用户价值

- 无直接用户价值，为内部开发准备

### 1.3 技术价值

- 明确团队间边界和职责
- 支持并行开发，缩短整体交付周期
- 建立契约测试机制，保证集成质量

---

## 2. 依赖关系

### 2.1 前置依赖

- 无

### 2.2 并行依赖

- 无

### 2.3 后续依赖

- Phase 1/2/3 所有 Spec 均依赖此接口定义

---

## 3. 接口定义

### 3.1 记忆服务接口 (MemoryService)

```go
// plugin/ai/memory/interface.go

package memory

import (
    "context"
    "time"
)

// MemoryService 统一记忆服务接口
// 消费团队: B (助理+日程), C (笔记增强)
type MemoryService interface {
    // ========== 短期记忆 (会话内) ==========
    
    // GetRecentMessages 获取会话内最近消息
    // limit: 返回消息数量上限，建议 10
    GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)
    
    // AddMessage 添加消息到会话
    AddMessage(ctx context.Context, sessionID string, msg Message) error
    
    // ========== 长期记忆 (跨会话) ==========
    
    // SaveEpisode 保存情景记忆
    SaveEpisode(ctx context.Context, episode EpisodicMemory) error
    
    // SearchEpisodes 搜索情景记忆
    // query: 搜索关键词，空字符串返回最近记录
    // limit: 返回数量上限
    SearchEpisodes(ctx context.Context, query string, limit int) ([]EpisodicMemory, error)
    
    // ========== 用户偏好 ==========
    
    // GetPreferences 获取用户偏好
    GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error)
    
    // UpdatePreferences 更新用户偏好
    UpdatePreferences(ctx context.Context, userID int32, prefs *UserPreferences) error
}

// Message 会话消息
type Message struct {
    Role      string    `json:"role"`       // "user" | "assistant" | "system"
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}

// EpisodicMemory 情景记忆
type EpisodicMemory struct {
    ID         int64     `json:"id"`
    UserID     int32     `json:"user_id"`
    Timestamp  time.Time `json:"timestamp"`
    AgentType  string    `json:"agent_type"`   // memo/schedule/amazing/assistant
    UserInput  string    `json:"user_input"`
    Outcome    string    `json:"outcome"`      // success/failure
    Summary    string    `json:"summary"`
    Importance float32   `json:"importance"`   // 0-1
}

// UserPreferences 用户偏好
type UserPreferences struct {
    Timezone           string            `json:"timezone"`
    DefaultDuration    int               `json:"default_duration"`     // 分钟
    PreferredTimes     []string          `json:"preferred_times"`      // ["09:00", "14:00"]
    FrequentLocations  []string          `json:"frequent_locations"`
    CommunicationStyle string            `json:"communication_style"`  // concise/detailed
    TagPreferences     []string          `json:"tag_preferences"`
    CustomSettings     map[string]any    `json:"custom_settings"`
}
```

---

### 3.2 LLM 路由服务接口 (RouterService)

```go
// plugin/ai/router/interface.go

package router

import "context"

// RouterService LLM 路由服务接口
// 消费团队: B (助理+日程), C (笔记增强)
type RouterService interface {
    // ClassifyIntent 意图分类
    // 返回: 意图类型, 置信度 (0-1), 错误
    // 实现: 规则优先 (0ms) → LLM 兜底 (~400ms)
    ClassifyIntent(ctx context.Context, input string) (Intent, float32, error)
    
    // SelectModel 根据任务类型选择模型
    // 返回: 模型配置 (本地/云端)
    SelectModel(ctx context.Context, task TaskType) (ModelConfig, error)
}

// Intent 意图类型
type Intent string

const (
    IntentMemoSearch     Intent = "memo_search"
    IntentMemoCreate     Intent = "memo_create"
    IntentScheduleQuery  Intent = "schedule_query"
    IntentScheduleCreate Intent = "schedule_create"
    IntentScheduleUpdate Intent = "schedule_update"
    IntentBatchSchedule  Intent = "batch_schedule"
    IntentAmazing        Intent = "amazing"
    IntentUnknown        Intent = "unknown"
)

// TaskType 任务类型 (用于模型选择)
type TaskType string

const (
    TaskIntentClassification TaskType = "intent_classification"
    TaskEntityExtraction     TaskType = "entity_extraction"
    TaskSimpleQA             TaskType = "simple_qa"
    TaskComplexReasoning     TaskType = "complex_reasoning"
    TaskSummarization        TaskType = "summarization"
    TaskTagSuggestion        TaskType = "tag_suggestion"
)

// ModelConfig 模型配置
type ModelConfig struct {
    Provider    string `json:"provider"`     // local/cloud
    Model       string `json:"model"`        // 模型名称
    MaxTokens   int    `json:"max_tokens"`
    Temperature float32 `json:"temperature"`
}
```

---

### 3.3 向量检索服务接口 (VectorService)

```go
// plugin/ai/vector/interface.go

package vector

import "context"

// VectorService 向量检索服务接口
// 消费团队: C (笔记增强)
type VectorService interface {
    // StoreEmbedding 存储向量
    StoreEmbedding(ctx context.Context, docID string, vector []float32, metadata map[string]any) error
    
    // SearchSimilar 相似度检索
    // filter: 过滤条件 (user_id, created_after 等)
    SearchSimilar(ctx context.Context, vector []float32, limit int, filter map[string]any) ([]VectorResult, error)
    
    // HybridSearch 混合检索 (向量 + 关键词)
    HybridSearch(ctx context.Context, query string, limit int) ([]SearchResult, error)
}

// VectorResult 向量检索结果
type VectorResult struct {
    DocID      string         `json:"doc_id"`
    Score      float32        `json:"score"`      // 相似度分数 0-1
    Metadata   map[string]any `json:"metadata"`
}

// SearchResult 混合检索结果
type SearchResult struct {
    Name       string  `json:"name"`       // memo UID
    Content    string  `json:"content"`
    Score      float32 `json:"score"`
    MatchType  string  `json:"match_type"` // vector/keyword/hybrid
}
```

---

### 3.4 时间解析服务接口 (TimeService)

```go
// plugin/ai/time/interface.go

package aitime

import (
    "context"
    "time"
)

// TimeService 时间解析服务接口
// 消费团队: B (助理+日程)
type TimeService interface {
    // Normalize 标准化时间表达
    // 支持: "明天3点", "下午三点", "2026-1-28", "15:00"
    // 返回: 标准化的 time.Time
    Normalize(ctx context.Context, input string, timezone string) (time.Time, error)
    
    // ParseNaturalTime 解析自然语言时间
    // reference: 参考时间点 (通常为当前时间)
    // 返回: 时间范围
    ParseNaturalTime(ctx context.Context, input string, reference time.Time) (TimeRange, error)
}

// TimeRange 时间范围
type TimeRange struct {
    Start time.Time `json:"start"`
    End   time.Time `json:"end"`
}
```

---

### 3.5 缓存服务接口 (CacheService)

```go
// plugin/ai/cache/interface.go

package cache

import (
    "context"
    "time"
)

// CacheService 缓存服务接口
// 消费团队: B (助理+日程), C (笔记增强)
type CacheService interface {
    // Get 获取缓存
    // 返回: 值, 是否存在
    Get(ctx context.Context, key string) ([]byte, bool)
    
    // Set 设置缓存
    // ttl: 过期时间
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    
    // Invalidate 使缓存失效
    // pattern: 支持通配符 (user:123:*)
    Invalidate(ctx context.Context, pattern string) error
}
```

---

### 3.6 评估指标服务接口 (MetricsService)

```go
// plugin/ai/metrics/interface.go

package metrics

import (
    "context"
    "time"
)

// MetricsService 评估指标服务接口
// 消费团队: B (助理+日程), C (笔记增强)
type MetricsService interface {
    // RecordRequest 记录请求指标
    RecordRequest(ctx context.Context, agentType string, latency time.Duration, success bool)
    
    // RecordToolCall 记录工具调用指标
    RecordToolCall(ctx context.Context, toolName string, latency time.Duration, success bool)
    
    // GetStats 获取统计数据
    GetStats(ctx context.Context, timeRange TimeRange) (*AgentMetrics, error)
}

// AgentMetrics Agent 指标
type AgentMetrics struct {
    RequestCount  int64                    `json:"request_count"`
    SuccessCount  int64                    `json:"success_count"`
    LatencyP50    time.Duration            `json:"latency_p50"`
    LatencyP95    time.Duration            `json:"latency_p95"`
    AgentStats    map[string]*AgentStat    `json:"agent_stats"`
    ErrorsByType  map[string]int64         `json:"errors_by_type"`
}

// AgentStat 单个 Agent 统计
type AgentStat struct {
    Count      int64         `json:"count"`
    SuccessRate float32      `json:"success_rate"`
    AvgLatency time.Duration `json:"avg_latency"`
}
```

---

### 3.7 会话持久化服务接口 (SessionService)

```go
// plugin/ai/session/interface.go

package session

import "context"

// SessionService 会话持久化服务接口
// 消费团队: B (助理+日程)
type SessionService interface {
    // SaveContext 保存会话上下文
    SaveContext(ctx context.Context, sessionID string, context *ConversationContext) error
    
    // LoadContext 加载会话上下文
    LoadContext(ctx context.Context, sessionID string) (*ConversationContext, error)
    
    // ListSessions 列出用户会话
    ListSessions(ctx context.Context, userID int32, limit int) ([]SessionSummary, error)
}

// ConversationContext 会话上下文
type ConversationContext struct {
    SessionID    string                 `json:"session_id"`
    UserID       int32                  `json:"user_id"`
    AgentType    string                 `json:"agent_type"`
    Messages     []Message              `json:"messages"`
    Metadata     map[string]any         `json:"metadata"`
    CreatedAt    int64                  `json:"created_at"`
    UpdatedAt    int64                  `json:"updated_at"`
}

// SessionSummary 会话摘要
type SessionSummary struct {
    SessionID   string `json:"session_id"`
    AgentType   string `json:"agent_type"`
    LastMessage string `json:"last_message"`
    UpdatedAt   int64  `json:"updated_at"`
}
```

---

## 4. 数据库 Schema

### 4.1 情景记忆表

```sql
-- store/db/postgres/migration/xxx_add_episodic_memory.sql

CREATE TABLE episodic_memory (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES "user"(id),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    agent_type VARCHAR(20) NOT NULL,
    user_input TEXT NOT NULL,
    outcome VARCHAR(20) NOT NULL DEFAULT 'success',
    summary TEXT,
    importance REAL DEFAULT 0.5,
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_episodic_user_time (user_id, timestamp DESC),
    INDEX idx_episodic_agent (agent_type)
);
```

### 4.2 用户偏好表

```sql
-- store/db/postgres/migration/xxx_add_user_preferences.sql

CREATE TABLE user_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES "user"(id),
    preferences JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 4.3 会话上下文表

```sql
-- store/db/postgres/migration/xxx_add_conversation_context.sql

CREATE TABLE conversation_context (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL REFERENCES "user"(id),
    agent_type VARCHAR(20) NOT NULL,
    context_data JSONB NOT NULL DEFAULT '{}',
    created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
    updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
    
    INDEX idx_conversation_user (user_id),
    INDEX idx_conversation_updated (updated_ts)
);
```

---

## 5. Mock 实现规范

### 5.1 Mock 实现要求

每个接口必须提供 Mock 实现，用于团队 B/C 并行开发：

```go
// plugin/ai/memory/mock.go

package memory

type MockMemoryService struct {
    messages    map[string][]Message
    episodes    []EpisodicMemory
    preferences map[int32]*UserPreferences
}

func NewMockMemoryService() *MockMemoryService {
    return &MockMemoryService{
        messages:    make(map[string][]Message),
        episodes:    make([]EpisodicMemory, 0),
        preferences: make(map[int32]*UserPreferences),
    }
}

// 实现 MemoryService 接口的所有方法...
```

### 5.2 Mock 数据要求

Mock 实现应包含合理的测试数据：
- 至少 10 条示例消息
- 至少 5 条情景记忆
- 默认用户偏好配置

---

## 6. 契约测试

### 6.1 测试用例

```go
// plugin/ai/memory/interface_test.go

func TestMemoryServiceContract(t *testing.T) {
    // 测试 GetRecentMessages
    t.Run("GetRecentMessages_ReturnsMessages", func(t *testing.T) {
        svc := NewMockMemoryService()
        // 添加测试数据
        svc.AddMessage(ctx, "session1", Message{...})
        
        // 验证返回
        msgs, err := svc.GetRecentMessages(ctx, "session1", 10)
        assert.NoError(t, err)
        assert.Len(t, msgs, 1)
    })
    
    // 测试 SaveEpisode
    t.Run("SaveEpisode_StoresData", func(t *testing.T) {
        // ...
    })
}
```

### 6.2 验收标准

- [ ] 所有接口方法有对应测试用例
- [ ] Mock 实现通过所有测试
- [ ] 团队 B 调用 Mock 成功
- [ ] 团队 C 调用 Mock 成功

---

## 7. 交付物清单

### 7.1 代码文件

- [ ] `plugin/ai/memory/interface.go` - 记忆服务接口
- [ ] `plugin/ai/memory/mock.go` - 记忆服务 Mock
- [ ] `plugin/ai/router/interface.go` - 路由服务接口
- [ ] `plugin/ai/router/mock.go` - 路由服务 Mock
- [ ] `plugin/ai/vector/interface.go` - 向量服务接口
- [ ] `plugin/ai/vector/mock.go` - 向量服务 Mock
- [ ] `plugin/ai/time/interface.go` - 时间服务接口
- [ ] `plugin/ai/time/mock.go` - 时间服务 Mock
- [ ] `plugin/ai/cache/interface.go` - 缓存服务接口
- [ ] `plugin/ai/cache/mock.go` - 缓存服务 Mock
- [ ] `plugin/ai/metrics/interface.go` - 指标服务接口
- [ ] `plugin/ai/metrics/mock.go` - 指标服务 Mock
- [ ] `plugin/ai/session/interface.go` - 会话服务接口
- [ ] `plugin/ai/session/mock.go` - 会话服务 Mock

### 7.2 测试文件

- [ ] `plugin/ai/memory/interface_test.go`
- [ ] `plugin/ai/router/interface_test.go`
- [ ] `plugin/ai/vector/interface_test.go`
- [ ] `plugin/ai/time/interface_test.go`
- [ ] `plugin/ai/cache/interface_test.go`
- [ ] `plugin/ai/metrics/interface_test.go`
- [ ] `plugin/ai/session/interface_test.go`

### 7.3 数据库迁移

- [ ] `store/db/postgres/migration/xxx_add_episodic_memory.sql`
- [ ] `store/db/postgres/migration/xxx_add_user_preferences.sql`
- [ ] `store/db/postgres/migration/xxx_add_conversation_context.sql`

---

## 8. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 5 人天 |
| 预期收益 | 消除并行开发阻塞，节省 2-3 周等待时间 |
| 风险评估 | 低 |
| 回报周期 | 立即 |

---

## 9. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 接口设计不完整 | 中 | 高 | 评审会议 + 迭代修订 |
| Mock 实现与真实实现差异大 | 中 | 中 | 契约测试 + 集成测试 |

---

## 10. 实施计划

### 10.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 接口设计 + 评审 |
| Day 2-3 | 2人天 | Mock 实现 |
| Day 4 | 1人天 | 契约测试编写 |
| Day 5 | 1人天 | 团队 B/C 集成验证 |

### 10.2 检查点

- [ ] Day 1: 接口定义评审通过
- [ ] Day 3: Mock 实现完成
- [ ] Day 5: 三方联调通过

---

## 附录

### A. 参考资料

- [主路线图](../../research/00-master-roadmap.md)
- [智能助理调研](../../research/assistant-research.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
