# Memos AI 能力实现方案

> 为 Memos 知识管理平台增加 AI 智能能力

> **⚠️ 数据库支持说明**
>
> AI 功能和所有后续新功能仅支持 **PostgreSQL** 和 **SQLite** 数据库。
>
> **原因**：
> - 向量搜索（pgvector）仅在 PostgreSQL 上可用
> - MySQL 缺乏 JSON 字段约束和高级触发器支持
> - 维护三数据库兼容性的成本过高
>
> **建议**：现有 MySQL 用户请迁移到 PostgreSQL 以获得完整功能。

## 项目现状
> 
> 截至当前，后端核心功能已全部完成，正处于前端集成阶段。
> 
> - **已完成 Backend**: 
>   - 基础设施 (Proto, Config, DB Migration)
>   - 模型层 (Embedding, Reranker, LLM)
>   - 存储层 (PostgreSQL Vector Search)
>   - 服务层 (Background Runner, gRPC APIs)
> - **待开发 Frontend**:
>   - React Hooks封装
>   - UI 组件集成 (Search, Chat, Tags)

## 技术架构

### 技术选型

| 组件           | 技术方案                              | 说明                      |
| -------------- | ------------------------------------- | ------------------------- |
| **向量数据库** | PostgreSQL + pgvector                 | 复用现有 PG，1:1 向量存储 |
| **向量模型**   | SiliconFlow `BAAI/bge-m3`             | 1024 维，中英双语         |
| **重排序**     | SiliconFlow `BAAI/bge-reranker-v2-m3` | 提升检索精度              |
| **大语言模型** | DeepSeek `deepseek-chat`              | 低成本，高质量            |
| **Go SDK**     | `tmc/langchaingo`                     | 统一多供应商调用          |

### 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ SemanticSearch│  │ AI Chat Box │  │ Tag Suggestions        │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │ gRPC / Connect
┌─────────────────────────────────────────────────────────────────┐
│                     AIService (Go gRPC)                         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                  Provider Abstraction                     │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐                │   │
│  │  │Embedding │  │ Reranker │  │   LLM    │                │   │
│  │  │ Provider │  │ Provider │  │ Provider │                │   │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘                │   │
│  │       │             │             │                       │   │
│  │  SiliconFlow   SiliconFlow    DeepSeek                   │   │
│  │  OpenAI        Cohere         OpenAI                     │   │
│  │  Ollama        -              Ollama                     │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                        Store Layer                              │
│  ┌──────────────────┐  ┌──────────────────────────────────────┐ │
│  │   memo_embedding │  │  PostgreSQL + pgvector               │ │
│  │   (1:1 外键)     │  │  - HNSW 索引                         │ │
│  └──────────────────┘  │  - Cosine 相似度                     │ │
│                        └──────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## 功能规格

### API 定义

```protobuf
service AIService {
  // 语义搜索
  rpc SemanticSearch(SemanticSearchRequest) returns (SemanticSearchResponse);
  // 标签推荐
  rpc SuggestTags(SuggestTagsRequest) returns (SuggestTagsResponse);
  // AI 对话 (流式)
  rpc ChatWithMemos(ChatWithMemosRequest) returns (stream ChatWithMemosResponse);
  // 相关笔记
  rpc GetRelatedMemos(GetRelatedMemosRequest) returns (GetRelatedMemosResponse);
}
```

### 语义搜索流程

```
Query → Embedding → pgvector(Top 10) → Rerank(Top 10) → Response
```

> 注: 2C2G环境下减少初始召回量以降低内存和CPU压力

### RAG 对话流程

1. 语义检索相关笔记 (Top 5)
2. 构建 Prompt (系统提示 + 笔记上下文 + 用户问题)
3. 流式调用 LLM 生成回答
4. 返回回答 + 引用来源

---

## 数据模型

### memo_embedding 表

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE memo_embedding (
    id SERIAL PRIMARY KEY,
    memo_id INTEGER NOT NULL REFERENCES memo(id) ON DELETE CASCADE,
    embedding vector(1024) NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT 'BAAI/bge-m3',
    created_ts BIGINT NOT NULL,
    updated_ts BIGINT NOT NULL,
    UNIQUE(memo_id, model)
);

CREATE INDEX idx_memo_embedding_hnsw
ON memo_embedding USING hnsw (embedding vector_cosine_ops)
WITH (m = 8, ef_construction = 32);  -- 2C2G优化
```

---

## 配置

### 环境变量

```bash
# 启用 AI
MEMOS_AI_ENABLED=true

# SiliconFlow (向量 + 重排序)
MEMOS_AI_SILICONFLOW_API_KEY=sk-xxx

# DeepSeek (LLM)
MEMOS_AI_DEEPSEEK_API_KEY=sk-xxx

# 可选：切换供应商
MEMOS_AI_EMBEDDING_PROVIDER=siliconflow  # siliconflow | openai | ollama
MEMOS_AI_LLM_PROVIDER=deepseek           # deepseek | openai | ollama
```

### 参数配置

| 参数             | 值     | 说明                               |
| ---------------- | ------ | ---------------------------------- |
| 向量维度         | 1024   | bge-m3 原生维度                    |
| 索引类型         | HNSW   | m=8, ef_construction=32 (2C2G优化) |
| 向量召回         | Top 10 | 初始召回量 (2C2G优化)              |
| Rerank 返回      | Top 10 | 最终返回量                         |
| RAG 上下文       | 5 条   | 送入 LLM 的笔记数                  |
| LLM temperature  | 0.7    | 平衡创造性和准确性                 |
| Embedding 批处理 | 8      | 后台任务批次大小 (2C2G)            |
| 后台任务间隔     | 2 分钟 | 向量生成轮询间隔 (2C2G)            |

---

## 文件清单

### 新增文件

| 文件                                  | 说明                         |
| ------------------------------------- | ---------------------------- |
| `proto/api/v1/ai_service.proto`       | gRPC 服务定义                |
| `plugin/ai/config.go`                 | AI 配置解析                  |
| `plugin/ai/embedding.go`              | Embedding 服务 (langchaingo) |
| `plugin/ai/reranker.go`               | Reranker 服务                |
| `plugin/ai/llm.go`                    | LLM 服务 (langchaingo)       |
| `store/memo_embedding.go`             | MemoEmbedding 模型           |
| `store/db/postgres/memo_embedding.go` | 向量搜索实现                 |
| `server/router/api/v1/ai_service.go`  | AI gRPC 服务                 |
| `server/runner/embedding/runner.go`   | 后台向量生成                 |
| `web/src/hooks/useAIQueries.ts`       | AI React Query Hooks         |

### 修改文件

| 文件                          | 说明             |
| ----------------------------- | ---------------- |
| `internal/profile/profile.go` | 添加 AI 配置字段 |
| `store/driver.go`             | 扩展 Driver 接口 |
| `server/server.go`            | 注册 AI 服务     |

### 数据库迁移

| 文件                                                | 说明         |
| --------------------------------------------------- | ------------ |
| `store/migration/postgres/0.30/1__add_pgvector.sql` | 向量表和索引 |

---

## 成本估算

### 月度成本 (个人使用)

假设：1,000 条笔记，每日 20 次搜索，10 次对话

| 项目                    | 月成本      |
| ----------------------- | ----------- |
| Embedding (SiliconFlow) | ~$0.01      |
| Rerank (SiliconFlow)    | ~$0.01      |
| LLM (DeepSeek)          | ~$0.50      |
| **总计**                | **< $1/月** |

---

## 里程碑

| 阶段 | 工作内容                   | 时间 | 状态     |
| ---- | -------------------------- | ---- | -------- |
| M1   | Proto + 配置 + Migration   | 2 天 | ✅ 已完成 |
| M2   | AI 插件 (langchaingo 集成) | 3 天 | ✅ 已完成 |
| M3   | Store 层向量搜索           | 2 天 | ✅ 已完成 |
| M4   | gRPC 服务实现              | 2 天 | ✅ 已完成 |
| M5   | 前端集成                   | 2 天 | ✅ 已完成 |
| M6   | 测试 + 文档                | 1 天 | ✅ 已完成 |

**总计：12 工作日**
