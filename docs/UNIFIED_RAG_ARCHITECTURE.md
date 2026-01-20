# 统一 RAG 架构设计方案

## 📋 目录

1. [架构目标](#架构目标)
2. [当前问题分析](#当前问题分析)
3. [新架构设计](#新架构设计)
4. [实现步骤](#实现步骤)
5. [数据流图](#数据流图)
6. [API 设计](#api-设计)
7. [前端适配](#前端适配)

---

## 🎯 架构目标

### 核心理念
**将笔记和日程作为统一的 RAG 数据源，通过向量检索 + 重排序 + LLM 意图识别，实现智能问答。**

### 关键特性

1. **统一向量检索**
   - 笔记：内容向量化
   - 日程：标题+描述+时间+地点组合向量化

2. **智能意图分类**
   - 纯笔记问答（如"搜索关于X的笔记"）
   - 纯日程查询（如"今天有什么安排"）
   - 混合场景（如"我最近关于项目X的工作安排和相关记录"）

3. **结构化响应**
   - AI 回复文本
   - 元数据（问题类型、置信度）
   - 结构化数据（日程列表、笔记列表）

4. **前端智能渲染**
   - 根据问题类型选择渲染方式
   - 纯日程查询：显示日程卡片
   - 混合场景：显示 AI 总结 + 卡片

---

## 🔍 当前问题分析

### 问题 1：双重查询导致不一致

```
用户："查看日程"
  ↓
后端：
  ├─ AI 分析笔记 → 返回 "没有日程信息"
  └─ SQL 查询日程 → 返回 4 个日程
  ↓
前端显示：
  ├─ AI 消息："没有关于'日程'的信息"
  └─ 日程卡片："找到 4 个日程"  ❌ 矛盾！
```

**根本原因**：
- AI 只分析笔记数据，不知道日程数据
- 日程通过独立 SQL 查询，与 AI 上下文分离

### 问题 2：日程未参与语义检索

当前架构：
```
用户查询 → 向量检索笔记 → AI 回复
         → SQL 查询日程 → 独立返回
```

问题：
- 无法处理语义模糊的日程查询（如"关于项目的会议"）
- 时间范围查询太死板
- 混合场景处理不佳

### 问题 3：前端显示逻辑复杂

前端需要：
- 解析 AI 回复判断类型
- 管理两套数据源（AI 文本 + 后端结构化）
- 处理数据一致性

---

## 🏗️ 新架构设计

### 核心流程

```
┌─────────────────────────────────────────────────────────────┐
│                      用户查询                                │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Phase 1: 统一向量检索                           │
├─────────────────────────────────────────────────────────────┤
│  1. 查询向量化 (Embedding Service)                           │
│  2. 向量搜索 (pgvector)                                      │
│     - Top 20 笔记 (threshold ≥ 0.6)                         │
│     - Top 20 日程 (threshold ≥ 0.6)                         │
│  3. 合并结果                                                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│           Phase 2: Reranker 重排序                           │
├─────────────────────────────────────────────────────────────┤
│  输入：Top 20 笔记 + Top 20 日程                              │
│  操作：                                                    │
│    - Reranker 重排序（笔记和日程一起）                        │
│    - 返回 Top 10 混合结果                                    │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│           Phase 3: LLM 意图识别与分类                         │
├─────────────────────────────────────────────────────────────┤
│  System Prompt：                                            │
│    "你将接收检索到的笔记和日程数据，请判断用户问题类型：      │
│     1. 纯笔记问答                                            │
│     2. 纯日程查询                                            │
│     3. 混合场景                                              │
│                                                             │
│     返回 JSON：                                             │
│     {                                                       │
│       'query_type': 'schedule_only',                       │
│       'confidence': 0.95,                                   │
│       'reasoning': '用户明确询问今天安排'                    │
│     }"                                                      │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Phase 4: 智能回复生成                           │
├─────────────────────────────────────────────────────────────┤
│  根据 query_type 选择回复策略：                              │
│                                                             │
│  【纯日程查询】                                              │
│    - 返回结构化日程列表                                      │
│    - AI 生成简短总结（可选）                                 │
│    - 标记 response_type: "schedule_data"                   │
│                                                             │
│  【纯笔记问答】                                              │
│    - 基于笔记内容生成回答                                    │
│    - 引用相关笔记                                            │
│    - 标记 response_type: "text_response"                   │
│                                                             │
│  【混合场景】                                                │
│    - 分别组织日程和笔记信息                                  │
│    - AI 生成综合回复                                         │
│    - 标记 response_type: "mixed"                           │
│    - 返回结构化日程数据（供前端渲染）                        │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                 Phase 5: 结构化响应                          │
├─────────────────────────────────────────────────────────────┤
│  ChatWithMemosResponse {                                    │
│    content: "AI 生成的回复文本",                             │
│    query_metadata: {                                        │
│      query_type: "schedule_only | note_only | mixed",      │
│      confidence: 0.95,                                      │
│      sources: ["memo/123", "schedule/456"]                 │
│    },                                                       │
│    schedule_data: [  // 仅当包含日程时返回                  │
│      { uid, title, startTs, endTs, ... }                   │
│    ],                                                       │
│    note_data: [  // 仅当包含笔记时返回                      │
│      { uid, content, snippet, score }                      │
│    ]                                                        │
│  }                                                          │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              前端智能渲染                                    │
├─────────────────────────────────────────────────────────────┤
│  if (query_metadata.query_type === 'schedule_only') {      │
│    // 只显示日程卡片，不显示 AI 回复                         │
│    renderScheduleCards(schedule_data);                     │
│  } else if (query_type === 'note_only') {                  │
│    // 只显示 AI 回复                                         │
│    renderAIMessage(content);                               │
│  } else {  // mixed                                         │
│    // 显示 AI 总结 + 日程卡片                                │
│    renderAIMessage(content);                               │
│    renderScheduleCards(schedule_data);                     │
│  }                                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## 🛠️ 实现步骤

### Step 1: 数据库 Schema 扩展

#### 1.1 创建日程向量表

```sql
-- 创建日程嵌入向量表
CREATE TABLE schedule_embedding (
    id SERIAL PRIMARY KEY,
    schedule_id INTEGER NOT NULL REFERENCES schedule(id) ON DELETE CASCADE,
    content TEXT NOT NULL,  -- 用于向量化的文本内容
    embedding vector(1024),  -- 假设使用 1024 维向量
    model VARCHAR(100) NOT NULL DEFAULT 'BAAI/bge-m3',
    created_ts BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT,
    updated_ts BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT,

    -- 索引优化
    UNIQUE(schedule_id, model)
);

-- 向量相似度索引（IVFFlat）
CREATE INDEX idx_schedule_embedding_vector
    ON schedule_embedding
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- 复合索引（快速查询）
CREATE INDEX idx_schedule_embedding_schedule_model
    ON schedule_embedding(schedule_id, model);
```

#### 1.2 日程文本内容生成

为了向量化日程，需要生成可检索的文本表示：

```go
// 构建日程的文本表示，用于向量化
func buildScheduleTextForEmbedding(schedule *store.Schedule) string {
    var builder strings.Builder

    // 格式：标题 + 描述 + 时间 + 地点
    builder.WriteString(fmt.Sprintf("日程标题：%s\n", schedule.Title))

    if schedule.Description != "" {
        builder.WriteString(fmt.Sprintf("描述：%s\n", schedule.Description))
    }

    // 时间信息（中文格式）
    startTime := time.Unix(schedule.StartTs, 0)
    builder.WriteString(fmt.Sprintf("时间：%s", startTime.Format("2006-01-02 15:04")))

    if schedule.EndTs != nil {
        endTime := time.Unix(*schedule.EndTs, 0)
        builder.WriteString(fmt.Sprintf(" 至 %s", endTime.Format("15:04")))
    }

    if schedule.AllDay {
        builder.WriteString("（全天）")
    }

    builder.WriteString("\n")

    if schedule.Location != "" {
        builder.WriteString(fmt.Sprintf("地点：%s\n", schedule.Location))
    }

    return builder.String()
}
```

### Step 2: 向量化服务

#### 2.1 创建/更新日程时自动向量化

```go
// 在 schedule_service.go 中添加
func (s *ScheduleService) CreateSchedule(ctx context.Context, req *pb.CreateScheduleRequest) (*pb.CreateScheduleResponse, error) {
    // 1. 创建日程
    schedule, err := s.Store.CreateSchedule(ctx, req.Schedule)
    if err != nil {
        return nil, err
    }

    // 2. 异步生成向量
    go func() {
        if s.EmbeddingService != nil {
            s.embedSchedule(context.Background(), schedule)
        }
    }()

    return &pb.CreateScheduleResponse{Schedule: schedule}, nil
}

// 向量化日程
func (s *ScheduleService) embedSchedule(ctx context.Context, schedule *store.Schedule) error {
    // 1. 构建文本
    content := buildScheduleTextForEmbedding(schedule)

    // 2. 生成向量
    vector, err := s.EmbeddingService.Embed(ctx, content)
    if err != nil {
        return fmt.Errorf("failed to embed schedule: %w", err)
    }

    // 3. 存储向量
    embedding := &store.ScheduleEmbedding{
        ScheduleID: schedule.ID,
        Content:    content,
        Embedding:  vector,
        Model:      "BAAI/bge-m3",
    }

    _, err = s.Store.UpsertScheduleEmbedding(ctx, embedding)
    return err
}
```

#### 2.2 Store 层实现

```go
// store/schedule_embedding.go
type ScheduleEmbedding struct {
    ID        int64
    ScheduleID int64
    Content   string
    Embedding []float32
    Model     string
    CreatedTs int64
    UpdatedTs int64
}

// UpsertScheduleEmbedding 创建或更新日程嵌入
func (s *Store) UpsertScheduleEmbedding(ctx context.Context, embedding *ScheduleEmbedding) (*ScheduleEmbedding, error) {
    // 实现 upsert 逻辑
    // ...
}

// GetScheduleEmbedding 获取日程嵌入
func (s *Store) GetScheduleEmbedding(ctx context.Context, scheduleID int64, model string) (*ScheduleEmbedding, error) {
    // ...
}
```

### Step 3: 向量检索统一接口

#### 3.1 扩展 VectorSearch 支持日程

```go
// store/store.go

// VectorSearchOptions 向量搜索选项
type VectorSearchOptions struct {
    UserID       int32
    Vector       []float32
    Limit        int
    MinScore     float32  // 最小相似度阈值
    SearchTypes  []string // ["memo", "schedule"] 搜索类型
}

// VectorSearchResult 统一的搜索结果
type VectorSearchResult struct {
    Type      string // "memo" or "schedule"
    Score     float32
    Memo      *Memo
    Schedule  *Schedule
}

// VectorSearch 统一的向量搜索
func (s *Store) VectorSearch(ctx context.Context, opts *VectorSearchOptions) ([]*VectorSearchResult, error) {
    // 同时搜索 memo_embedding 和 schedule_embedding
    // 返回混合结果
}
```

### Step 4: AI 服务改进

#### 4.1 统一 RAG 检索流程

```go
// ai_service.go

func (s *AIService) ChatWithMemos(req *pb.ChatWithMemosRequest, stream pb.AIService_ChatWithMemosServer) error {
    // ... 前置检查 ...

    // Phase 1: 统一向量检索
    results, err := s.unifiedVectorSearch(ctx, user.ID, req.Message)
    if err != nil {
        return err
    }

    // Phase 2: Reranker 重排序
    rerankedResults, err := s.rerankResults(ctx, req.Message, results)
    if err != nil {
        return err
    }

    // Phase 3: LLM 意图识别
    queryMetadata := s.detectQueryIntent(ctx, req.Message, rerankedResults)

    // Phase 4: 智能回复生成
    content, structuredData := s.generateResponse(ctx, req, queryMetadata, rerankedResults)

    // Phase 5: 流式发送
    // ...
}

// 统一向量搜索
func (s *AIService) unifiedVectorSearch(ctx context.Context, userID int32, query string) ([]*store.VectorSearchResult, error) {
    // 1. 查询向量化
    queryVector, err := s.EmbeddingService.Embed(ctx, query)
    if err != nil {
        return nil, err
    }

    // 2. 同时搜索笔记和日程
    results, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
        UserID:      userID,
        Vector:      queryVector,
        Limit:       20,
        MinScore:    0.6,
        SearchTypes: []string{"memo", "schedule"},
    })

    return results, err
}

// 意图识别
func (s *AIService) detectQueryIntent(ctx context.Context, query string, results []*store.VectorSearchResult) *QueryMetadata {
    // 构建 LLM prompt
    prompt := s.buildIntentDetectionPrompt(query, results)

    // 调用 LLM
    response, err := s.LLMService.Chat(ctx, []ai.Message{
        {Role: "system", Content: intentDetectionSystemPrompt},
        {Role: "user", Content: prompt},
    })

    // 解析 JSON 返回
    metadata := parseQueryMetadata(response)
    return metadata
}
```

#### 4.2 Prompt 工程

```go
const intentDetectionSystemPrompt = `
你是一个智能查询意图分类器。请分析用户问题和检索到的数据，判断查询类型。

## 查询类型

1. **schedule_only** (纯日程查询)
   特征：
   - 用户明确询问时间安排
   - 检索结果中日程相关度更高
   - 关键词："今天"、"明天"、"日程"、"安排"

2. **note_only** (纯笔记问答)
   特征：
   - 用户询问内容、信息搜索
   - 检索结果中笔记相关度更高
   - 关键词："搜索"、"查找"、"笔记"、"记录"

3. **mixed** (混合场景)
   特征：
   - 同时涉及笔记和日程
   - 需要综合信息
   - 例如："关于项目X的工作安排和相关记录"

## 返回格式

请以 JSON 格式返回（不要有其他内容）：
{
  "query_type": "schedule_only | note_only | mixed",
  "confidence": 0.0-1.0,
  "reasoning": "判断理由",
  "schedule_count": 0,
  "note_count": 0
}
`

const responseGenerationSystemPrompt = `
你是一个基于用户个人数据的 AI 助手。

## 任务

根据查询类型和检索到的数据，生成合适的回复。

### 纯日程查询 (schedule_only)

返回结构：
<<<QUERY_TYPE:schedule_only>>>
<<<SCHEDULE_COUNT:N>>>

然后简短总结（可选）：
"为您找到 N 个日程安排..."

### 纯笔记问答 (note_only)

基于笔记内容回答问题，引用相关笔记。

### 混合场景 (mixed)

1. 先总结日程
2. 再总结笔记
3. 使用清晰的结构分隔

返回结构：
<<<QUERY_TYPE:mixed>>>
<<<SCHEDULE_COUNT:N>>>
<<<NOTE_COUNT:M>>>

然后生成综合回复。
`
```

### Step 5: Protocol Buffers 定义

#### 5.1 更新 ai_service.proto

```protobuf
syntax = "proto3";

package api.v1;

service AIService {
  rpc ChatWithMemos (ChatWithMemosRequest) returns (stream ChatWithMemosResponse);
}

message ChatWithMemosRequest {
  string message = 1;
  repeated string history = 2;
}

message ChatWithMemosResponse {
  // 流式内容
  string content = 1;

  // 来源信息
  repeated string sources = 2;

  // 元数据（最后一条消息）
  QueryMetadata query_metadata = 3;

  // 结构化数据（仅当包含日程或笔记时）
  repeated ScheduleSummary schedules = 4;
  repeated NoteSummary notes = 5;

  // 完成标记
  bool done = 6;
}

message QueryMetadata {
  string query_type = 1;  // "schedule_only", "note_only", "mixed"
  float confidence = 2;
  string reasoning = 3;
  int32 schedule_count = 4;
  int32 note_count = 5;
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

message NoteSummary {
  string uid = 1;
  string content = 2;
  string snippet = 3;
  float score = 4;
}
```

### Step 6: 前端适配

#### 6.1 更新 AIChat 组件

```typescript
// web/src/pages/AIChat.tsx

interface ChatResponse {
  content: string;
  sources: string[];
  queryMetadata?: {
    queryType: 'schedule_only' | 'note_only' | 'mixed';
    confidence: number;
    reasoning: string;
    scheduleCount: number;
    noteCount: number;
  };
  schedules?: ScheduleSummary[];
  notes?: NoteSummary[];
  done: boolean;
}

// 处理流式响应
const handleStreamResponse = async () => {
  let fullResponse: ChatResponse = {
    content: '',
    sources: [],
    done: false
  };

  for await (const chunk of stream) {
    fullResponse = {
      ...fullResponse,
      ...chunk
    };

    // 更新内容
    if (chunk.content) {
      setItems(prev => [...prev, { content: chunk.content }]);
    }
  }

  // 流结束后，根据 query_type 决定显示方式
  if (fullResponse.queryMetadata) {
    const { queryType } = fullResponse.queryMetadata;

    if (queryType === 'schedule_only' && fullResponse.schedules?.length > 0) {
      // 只显示日程卡片，隐藏 AI 回复
      setScheduleQueryResult({
        schedules: fullResponse.schedules,
        title: '日程查询结果'
      });
      setShowAIMessage(false); // 隐藏 AI 文本
    } else if (queryType === 'mixed') {
      // 显示 AI 回复 + 日程卡片
      setShowAIMessage(true);
      setScheduleQueryResult({
        schedules: fullResponse.schedules || [],
        title: '相关日程'
      });
    } else {
      // 只显示 AI 回复
      setShowAIMessage(true);
    }
  }
};
```

#### 6.2 UI 渲染逻辑

```tsx
// 消息渲染
{items.map((item, index) => {
  if (!showAIMessage && item.role === 'assistant') {
    return null; // 纯日程查询时隐藏 AI 回复
  }

  return <MessageBubble key={index} message={item} />;
})}

// 日程卡片
{showScheduleQueryResult && scheduleQueryResult && (
  <ScheduleQueryResult
    title={scheduleQueryResult.title}
    schedules={scheduleQueryResult.schedules}
    onClose={() => setShowScheduleQueryResult(false)}
  />
)}
```

---

## 📊 数据流图

### 完整数据流

```
用户输入："今天有什么安排"
        ↓
┌──────────────────────────────────┐
│  Frontend: AIChat.tsx            │
└────────────┬─────────────────────┘
             │ gRPC Stream
             ▼
┌──────────────────────────────────┐
│  Backend: AIService.ChatWithMemos│
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│  Phase 1: Embedding              │
│  - 查询向量化                     │
│  - query → vector(1024)          │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│  Phase 2: Vector Search          │
│  - memo_embedding: Top 20        │
│  - schedule_embedding: Top 20    │
│  - threshold ≥ 0.6               │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│  Phase 3: Reranker               │
│  - 混合 Top 20 笔记 + Top 20 日程 │
│  - 重排序返回 Top 10             │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│  Phase 4: LLM Intent Detection   │
│  - 分析: 用户问"今天" → 日程查询  │
│  - 返回: query_type=schedule_only│
│  - confidence: 0.95              │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│  Phase 5: Response Generation    │
│  - 生成简短总结                   │
│  - 标记: <<<QUERY_TYPE:schedule>>>│
│  - 准备结构化日程数据             │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│  Stream Response                 │
│  - content: "为您找到 3 个日程..." │
│  - query_metadata: {...}         │
│  - schedules: [...]              │
│  - done: true                    │
└────────────┬─────────────────────┘
             │
             ▼
┌──────────────────────────────────┐
│  Frontend Rendering              │
│  - 检测 query_type=schedule_only │
│  - 只渲染 ScheduleQueryResult     │
│  - 不渲染 AI 消息                │
└──────────────────────────────────┘
```

---

## 🎨 UI/UX 改进

### 场景 1：纯日程查询

**输入**："今天有什么安排"

**后端响应**：
```json
{
  "query_metadata": {
    "query_type": "schedule_only",
    "confidence": 0.98,
    "schedule_count": 3
  },
  "schedules": [
    { "title": "团队周会", "start_ts": ..., "location": "会议室A" },
    { "title": "产品评审", "start_ts": ..., "location": "会议室B" },
    { "title": "客户会议", "start_ts": ..., "location": "线上" }
  ]
}
```

**前端渲染**：
```
┌─────────────────────────────────────┐
│ 📅 日程查询结果                     │
│ 找到 3 个日程                       │
├─────────────────────────────────────┤
│ 今天  14:00-15:00  团队周会 @会议室A│
│ 今天  16:00-17:00  产品评审 @会议室B│
│ 今天  19:00-20:00  客户会议 @线上   │
└─────────────────────────────────────┘
```

### 场景 2：混合场景

**输入**："我最近关于AI项目的工作安排和相关记录"

**后端响应**：
```json
{
  "query_metadata": {
    "query_type": "mixed",
    "confidence": 0.85,
    "schedule_count": 2,
    "note_count": 5
  },
  "content": "关于您最近关于AI项目的工作安排...",
  "schedules": [...],
  "notes": [...]
}
```

**前端渲染**：
```
┌─────────────────────────────────────┐
│ 🤖 关于您最近关于AI项目的工作安排： │
│                                     │
│ **日程安排**（2个）：               │
│ - 明天 10:00: AI技术评审            │
│ - 后天 14:00: 架构设计讨论          │
│                                     │
│ **相关笔记**：                      │
│ - AI项目架构设计 (相关度 95%)      │
│ - 技术选型分析 (相关度 88%)        │
└─────────────────────────────────────┘
└─────────────────────────────────────┘
│ 📅 相关日程                         │
│ [日程卡片...]                       │
└─────────────────────────────────────┘
```

---

## 📈 性能优化

### 1. 向量检索优化

- **IVFFlat 索引**：加速向量相似度搜索
- **批量查询**：笔记和日程并行检索
- **结果缓存**：相同查询 30 秒内复用

### 2. LLM 调用优化

- **意图识别**：使用小模型（如 GPT-3.5）
- **流式响应**：快速反馈用户体验
- **异步向量化**：日程创建时异步生成向量

### 3. 数据库优化

```sql
-- 复合索引
CREATE INDEX idx_schedule_schedule_user_time
  ON schedule(creator_id, start_ts, end_ts)
  WHERE row_status = 'NORMAL';

-- 向量索引（IVFFlat）
CREATE INDEX idx_schedule_embedding_vector_ivfflat
  ON schedule_embedding
  USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);
```

---

## ✅ 验收标准

### 功能验收

1. ✅ 日程能够向量化存储
2. ✅ 统一的向量检索支持笔记和日程
3. ✅ LLM 能够正确识别 3 种查询类型
4. ✅ 纯日程查询时只显示日程卡片
5. ✅ 混合场景时显示 AI 回复 + 日程卡片
6. ✅ 无矛盾信息显示

### 性能验收

1. ✅ 向量检索延迟 < 200ms
2. ✅ 端到端响应延迟 < 2s
3. ✅ 支持并发 100+ 用户

### 质量验收

1. ✅ 意图识别准确率 > 90%
2. ✅ 日程检索准确率 > 85%
3. ✅ 混合场景处理准确率 > 80%

---

## 🚀 实施计划

### Phase 1: 数据库改造（1-2天）
- [ ] 创建 schedule_embedding 表
- [ ] 实现向量化逻辑
- [ ] 数据迁移（历史日程向量化）

### Phase 2: 后端实现（2-3天）
- [ ] 扩展 Store 层向量搜索
- [ ] 改进 AI 服务 RAG 流程
- [ ] 实现意图识别
- [ ] 更新 Protocol Buffers

### Phase 3: 前端适配（1-2天）
- [ ] 更新 AIChat 组件
- [ ] 实现智能渲染逻辑
- [ ] UI/UX 优化

### Phase 4: 测试与优化（1-2天）
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] 用户测试

**总计**：5-9 天

---

## 📚 参考资料

- [pgvector 文档](https://github.com/pgvector/pgvector)
- [RAG 架构最佳实践](https://www.anthropic.com/index/retrieval-augmented-generation)
- [LangChain 多向量检索](https://python.langchain.com/docs/use_cases/multi_vector_retriever/)
