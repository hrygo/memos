# P2-A003: 会话持久化服务

> **状态**: 🔲 待开发  
> **优先级**: P1 (重要)  
> **投入**: 3 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 4

---

## 1. 目标与背景

### 1.1 核心目标

实现会话上下文的持久化存储，支持服务重启后会话恢复、跨设备会话同步（本地部署场景）。

### 1.2 用户价值

- 服务重启后会话不丢失
- 跨设备无缝继续对话
- 减少重复说明背景 50%+

### 1.3 技术价值

- 会话状态与业务逻辑解耦
- 为多实例部署铺路
- 统一会话管理入口

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A001: 轻量记忆系统（会话数据结构）
- [x] S0-interface-contract: SessionService 接口

### 2.2 并行依赖

- P2-A001/A002: 可并行开发

### 2.3 后续依赖

- P3-B001: 预测性交互（依赖会话历史）
- P3-C002: 智能回顾（依赖会话数据）

---

## 3. 功能设计

### 3.1 架构图

```
                    会话持久化架构
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   ┌────────────────────────────────────────────────────┐  │
│   │                  SessionService                     │  │
│   │                                                     │  │
│   │  • SaveContext()    保存会话上下文                   │  │
│   │  • LoadContext()    加载会话上下文                   │  │
│   │  • ListSessions()   列出会话历史                    │  │
│   │  • DeleteSession()  删除会话                        │  │
│   └────────────────────────────────────────────────────┘  │
│                            │                               │
│          ┌─────────────────┼─────────────────┐            │
│          │                 │                 │            │
│          ▼                 ▼                 ▼            │
│   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐      │
│   │  内存缓存层   │ │  PostgreSQL  │ │  清理任务    │      │
│   │              │ │              │ │              │      │
│   │ • 热会话缓存 │ │ • 持久化存储 │ │ • 30天清理  │      │
│   │ • LRU淘汰   │ │ • JSONB序列化│ │ • 每日运行  │      │
│   └──────────────┘ └──────────────┘ └──────────────┘      │
│                                                            │
│   存储开销: ~1MB (10万条记录)                              │
│   查询性能: 索引优化，<10ms                                │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心接口定义

```go
// plugin/ai/session/service.go

type SessionService interface {
    // 保存会话上下文
    SaveContext(ctx context.Context, sessionID string, context *ConversationContext) error
    
    // 加载会话上下文
    LoadContext(ctx context.Context, sessionID string) (*ConversationContext, error)
    
    // 列出用户的会话历史
    ListSessions(ctx context.Context, userID int32, limit int) ([]*SessionSummary, error)
    
    // 删除会话（用户隐私控制）
    DeleteSession(ctx context.Context, sessionID string) error
    
    // 批量清理过期会话
    CleanupExpired(ctx context.Context, retentionDays int) (int64, error)
}
```

### 3.3 数据模型

```go
// plugin/ai/session/model.go

type ConversationContext struct {
    SessionID       string            `json:"session_id"`
    UserID          int32             `json:"user_id"`
    RecentMessages  []*Message        `json:"recent_messages"`  // 最近 10 轮
    LastAgentType   string            `json:"last_agent_type"`  // memo/schedule/amazing
    CurrentTopic    string            `json:"current_topic"`    // 当前话题
    CreatedAt       time.Time         `json:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at"`
}

type Message struct {
    Role      string    `json:"role"`       // user/assistant
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
    AgentType string    `json:"agent_type,omitempty"`
}

type SessionSummary struct {
    SessionID     string    `json:"session_id"`
    Title         string    `json:"title"`         // 首条消息摘要
    MessageCount  int       `json:"message_count"`
    LastAgentType string    `json:"last_agent_type"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

### 3.4 数据库 Schema

```sql
-- store/db/postgres/migration/conversation_context.sql

CREATE TABLE IF NOT EXISTS conversation_context (
    id            SERIAL PRIMARY KEY,
    session_id    VARCHAR(64) NOT NULL UNIQUE,
    user_id       INTEGER NOT NULL,
    context_data  JSONB NOT NULL,
    created_ts    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_ts    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引优化
CREATE INDEX idx_conversation_user ON conversation_context(user_id);
CREATE INDEX idx_conversation_updated ON conversation_context(updated_ts);
CREATE INDEX idx_conversation_session ON conversation_context(session_id);

-- 自动更新 updated_ts
CREATE OR REPLACE FUNCTION update_conversation_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_ts = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER conversation_timestamp_trigger
BEFORE UPDATE ON conversation_context
FOR EACH ROW
EXECUTE FUNCTION update_conversation_timestamp();
```

### 3.5 存储实现

```go
// plugin/ai/session/store.go

type sessionStore struct {
    db    *sql.DB
    cache CacheService
}

func NewSessionStore(db *sql.DB, cache CacheService) SessionService {
    return &sessionStore{
        db:    db,
        cache: cache,
    }
}

func (s *sessionStore) SaveContext(ctx context.Context, sessionID string, context *ConversationContext) error {
    // 序列化上下文
    data, err := json.Marshal(context)
    if err != nil {
        return fmt.Errorf("failed to marshal context: %w", err)
    }
    
    // Upsert 到数据库
    query := `
        INSERT INTO conversation_context (session_id, user_id, context_data)
        VALUES ($1, $2, $3)
        ON CONFLICT (session_id) 
        DO UPDATE SET context_data = $3, updated_ts = CURRENT_TIMESTAMP
    `
    
    _, err = s.db.ExecContext(ctx, query, sessionID, context.UserID, data)
    if err != nil {
        return fmt.Errorf("failed to save context: %w", err)
    }
    
    // 更新缓存
    cacheKey := fmt.Sprintf("session:%s", sessionID)
    s.cache.Set(cacheKey, context, 30*time.Minute)
    
    return nil
}

func (s *sessionStore) LoadContext(ctx context.Context, sessionID string) (*ConversationContext, error) {
    // 先查缓存
    cacheKey := fmt.Sprintf("session:%s", sessionID)
    if cached, ok := s.cache.Get(cacheKey); ok {
        return cached.(*ConversationContext), nil
    }
    
    // 查数据库
    query := `SELECT context_data FROM conversation_context WHERE session_id = $1`
    
    var data []byte
    err := s.db.QueryRowContext(ctx, query, sessionID).Scan(&data)
    if err == sql.ErrNoRows {
        return nil, nil  // 新会话
    }
    if err != nil {
        return nil, fmt.Errorf("failed to load context: %w", err)
    }
    
    // 反序列化
    var context ConversationContext
    if err := json.Unmarshal(data, &context); err != nil {
        return nil, fmt.Errorf("failed to unmarshal context: %w", err)
    }
    
    // 写入缓存
    s.cache.Set(cacheKey, &context, 30*time.Minute)
    
    return &context, nil
}

func (s *sessionStore) ListSessions(ctx context.Context, userID int32, limit int) ([]*SessionSummary, error) {
    query := `
        SELECT session_id, context_data, updated_ts
        FROM conversation_context 
        WHERE user_id = $1
        ORDER BY updated_ts DESC
        LIMIT $2
    `
    
    rows, err := s.db.QueryContext(ctx, query, userID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var summaries []*SessionSummary
    for rows.Next() {
        var sessionID string
        var data []byte
        var updatedAt time.Time
        
        if err := rows.Scan(&sessionID, &data, &updatedAt); err != nil {
            continue
        }
        
        var context ConversationContext
        if err := json.Unmarshal(data, &context); err != nil {
            continue
        }
        
        summary := &SessionSummary{
            SessionID:     sessionID,
            Title:         extractTitle(&context),
            MessageCount:  len(context.RecentMessages),
            LastAgentType: context.LastAgentType,
            UpdatedAt:     updatedAt,
        }
        summaries = append(summaries, summary)
    }
    
    return summaries, nil
}

func (s *sessionStore) DeleteSession(ctx context.Context, sessionID string) error {
    query := `DELETE FROM conversation_context WHERE session_id = $1`
    
    _, err := s.db.ExecContext(ctx, query, sessionID)
    if err != nil {
        return err
    }
    
    // 清除缓存
    cacheKey := fmt.Sprintf("session:%s", sessionID)
    s.cache.Delete(cacheKey)
    
    return nil
}

func (s *sessionStore) CleanupExpired(ctx context.Context, retentionDays int) (int64, error) {
    query := `
        DELETE FROM conversation_context 
        WHERE updated_ts < NOW() - INTERVAL '%d days'
    `
    
    result, err := s.db.ExecContext(ctx, fmt.Sprintf(query, retentionDays))
    if err != nil {
        return 0, err
    }
    
    return result.RowsAffected()
}

// 提取会话标题（首条用户消息的前 50 字符）
func extractTitle(context *ConversationContext) string {
    for _, msg := range context.RecentMessages {
        if msg.Role == "user" {
            title := msg.Content
            if len(title) > 50 {
                title = title[:50] + "..."
            }
            return title
        }
    }
    return "新对话"
}
```

### 3.6 会话恢复工作流

```go
// plugin/ai/session/recovery.go

type SessionRecovery struct {
    sessionSvc SessionService
    memorySvc  MemoryService
}

func (r *SessionRecovery) RecoverSession(ctx context.Context, sessionID string, userID int32) (*ConversationContext, error) {
    // 1. 尝试加载已有会话
    existing, err := r.sessionSvc.LoadContext(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    if existing != nil {
        // 会话存在，直接返回
        return existing, nil
    }
    
    // 2. 新会话：初始化上下文
    newContext := &ConversationContext{
        SessionID:      sessionID,
        UserID:         userID,
        RecentMessages: make([]*Message, 0, 10),
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }
    
    // 3. 加载用户偏好（如果有）
    prefs, _ := r.memorySvc.GetUserPreferences(ctx, userID)
    if prefs != nil {
        // 可选：根据偏好设置初始状态
    }
    
    return newContext, nil
}

// 添加消息并自动保存
func (r *SessionRecovery) AppendMessage(ctx context.Context, sessionID string, msg *Message) error {
    session, err := r.sessionSvc.LoadContext(ctx, sessionID)
    if err != nil || session == nil {
        return fmt.Errorf("session not found: %s", sessionID)
    }
    
    // 添加消息
    session.RecentMessages = append(session.RecentMessages, msg)
    
    // 保持滑动窗口（最多 10 轮 = 20 条消息）
    if len(session.RecentMessages) > 20 {
        session.RecentMessages = session.RecentMessages[len(session.RecentMessages)-20:]
    }
    
    session.UpdatedAt = time.Now()
    if msg.AgentType != "" {
        session.LastAgentType = msg.AgentType
    }
    
    // 保存
    return r.sessionSvc.SaveContext(ctx, sessionID, session)
}
```

### 3.7 清理任务

```go
// plugin/ai/session/cleanup.go

type SessionCleanupJob struct {
    sessionSvc     SessionService
    retentionDays  int
    ticker         *time.Ticker
}

func NewSessionCleanupJob(svc SessionService, retentionDays int) *SessionCleanupJob {
    return &SessionCleanupJob{
        sessionSvc:    svc,
        retentionDays: retentionDays,
    }
}

func (j *SessionCleanupJob) Start(ctx context.Context) {
    // 每天凌晨 3 点运行
    j.ticker = time.NewTicker(24 * time.Hour)
    
    go func() {
        for {
            select {
            case <-ctx.Done():
                j.ticker.Stop()
                return
            case <-j.ticker.C:
                deleted, err := j.sessionSvc.CleanupExpired(ctx, j.retentionDays)
                if err != nil {
                    slog.Error("session cleanup failed", "error", err)
                } else {
                    slog.Info("session cleanup completed", "deleted", deleted)
                }
            }
        }
    }()
}
```

---

## 4. 实现路径

### Day 1: 数据模型与存储层

- [ ] 创建数据库迁移脚本
- [ ] 实现 `SessionService` 接口
- [ ] 实现基本 CRUD 操作

### Day 2: 缓存与会话恢复

- [ ] 集成缓存层
- [ ] 实现 `SessionRecovery` 工作流
- [ ] 消息追加与滑动窗口

### Day 3: 清理任务与集成

- [ ] 实现清理任务
- [ ] 与 Agent 集成
- [ ] 单元测试 + 集成测试

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/session/service.go` | 接口定义 |
| `plugin/ai/session/model.go` | 数据模型 |
| `plugin/ai/session/store.go` | 存储实现 |
| `plugin/ai/session/recovery.go` | 会话恢复 |
| `plugin/ai/session/cleanup.go` | 清理任务 |
| `store/db/postgres/migration/xxx_conversation_context.sql` | 数据库迁移 |
| `plugin/ai/session/*_test.go` | 单元测试 |

### 5.2 配置项

```yaml
# configs/ai.yaml
session:
  retention_days: 30
  max_messages: 20
  cache_ttl: 30m
  cleanup_hour: 3  # 凌晨 3 点
```

---

## 6. 验收标准

### 6.1 功能验收

- [ ] 会话正确保存和加载
- [ ] 服务重启后会话恢复
- [ ] 滑动窗口正确截断
- [ ] 30 天过期自动清理

### 6.2 性能验收

- [ ] 保存延迟 < 50ms
- [ ] 加载延迟 < 10ms（缓存命中）
- [ ] 加载延迟 < 50ms（数据库）

### 6.3 隐私验收

- [ ] 用户可手动删除会话
- [ ] 数据仅本地存储
- [ ] 自动过期清理生效

### 6.4 测试用例

```go
func TestSessionPersistence(t *testing.T) {
    // 创建会话
    ctx := &ConversationContext{
        SessionID: "test-session-1",
        UserID:    1,
        RecentMessages: []*Message{
            {Role: "user", Content: "你好"},
            {Role: "assistant", Content: "你好！"},
        },
    }
    
    // 保存
    err := svc.SaveContext(context.Background(), ctx.SessionID, ctx)
    assert.NoError(t, err)
    
    // 加载
    loaded, err := svc.LoadContext(context.Background(), ctx.SessionID)
    assert.NoError(t, err)
    assert.Equal(t, ctx.SessionID, loaded.SessionID)
    assert.Equal(t, 2, len(loaded.RecentMessages))
}

func TestSlidingWindow(t *testing.T) {
    ctx := &ConversationContext{
        SessionID:      "test-session-2",
        UserID:         1,
        RecentMessages: make([]*Message, 25), // 超过 20
    }
    
    recovery := &SessionRecovery{sessionSvc: svc}
    
    // 追加消息后应保持 20 条
    recovery.AppendMessage(...)
    
    loaded, _ := svc.LoadContext(context.Background(), ctx.SessionID)
    assert.LessOrEqual(t, len(loaded.RecentMessages), 20)
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 3 人天 | 服务重启会话恢复 |
| 存储: ~1MB (10万记录) | 跨会话记忆能力 |
| 维护: 自动清理 | 用户体验显著提升 |

### 收益计算

- 减少用户重复说明背景 50%+
- 服务升级无感知（会话不丢失）
- 支撑后续预测性交互功能

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 数据库膨胀 | 中 | 中 | 30天自动清理 + 索引优化 |
| JSON序列化兼容 | 低 | 中 | 版本化字段，向后兼容 |
| 缓存不一致 | 低 | 低 | Write-through 策略 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 4 Day 1 | 数据模型与存储层 | TBD |
| Sprint 4 Day 2 | 缓存与会话恢复 | TBD |
| Sprint 4 Day 3 | 清理任务与集成 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../research/00-master-roadmap.md)  
> **研究文档**: [assistant-roadmap.md](../../../research/assistant-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
