# AI 能力建设实施方案 (Memos 私有化改造)

**版本**: 1.0.0
**定位**: 个人知识库与私人助理 (Personal Private Fork)
**主要目标**: 构建基于 RAG (语义检索) 的智能知识库，提供“与笔记对话”、“自动整理”等能力。

## 1. 核心架构决策

### 1.1 基础设施与环境
*   **服务器约束**: 阿里云 2核 2G 内存 (2C2G)。
*   **数据库**: **PostgreSQL + `pgvector`** (强制)。
    *   *理由*: 在单机资源有限的情况下，作为单一数据源（关系数据 + 向量数据）效率最高。放弃对 SQLite/MySQL 的兼容支持。
    *   *优化*: 针对 2G 内存进行参数调优 (见 4.1)。
*   **模型策略 (API First)**:
    *   **原则**: 严禁在本地运行模型推理，所有推理必须通过 API 完成。
    *   **LLM Provider**: 兼容 OpenAI 接口的供应商。
    *   **Embedding**: 通过 API 调用 `BAAI/bge-m3` (或同等能力模型)。
    *   **Reranker**: 必须引入 Rerank 步骤以提升检索精准度。

### 1.2 技术栈选择
*   **后端语言**: Go (复用现有 Memos 后端)。
*   **AI 框架**: **`tmc/langchaingo`**。
    *   *选型理由 (Vs go-openai)*: 虽然 `sashabaranov/go-openai` 更成熟，但它仅封装了 OpenAI API。为了避免重复造轮子（手动实现 RAG 流程、文档切片、甚至 Rerank 逻辑），`langchaingo` 提供的 orchestrator 能力是必须的。
    *   *风险控制*: 在 `go.mod` 中锁定版本，避免 Breaking Changes。
*   **配置管理**: **环境变量优先**。
    *   `MEMOS_AI_BASE_URL`: 模型 API 地址。
    *   `MEMOS_AI_API_KEY`: 鉴权密钥。
    *   `MEMOS_AI_EMBEDDING_MODEL`: 指定 Embedding 模型 (默认自动探测或指定 `bge-m3`)。

## 2. 详细改造计划

### 2.1 后端改造 (`server/`)

#### 2.1.1 新增 `server/ai` 模块
核心 AI 逻辑层，直接依赖 `pgx` 和 `langchaingo`，不设计复杂的 Interface 抽象。

*   **`provider.go` (模型管理)**:
    *   初始化 `langchaingo` 客户端。
    *   增加 **Exponential Backoff (指数退避)** 重试机制，应对网络波动。
    *   启动时调用 `ListModels` 接口，自动校验/探测 Embedding 模型可用性。
*   **`vector_store.go` (向量存储)**:
    *   实现基于 `pgvector` 的 CRUD 操作。
    *   核心 SQL: `SELECT id, content, 1 - (embedding <=> $1) as similarity FROM memos ORDER BY similarity DESC LIMIT $2`。
*   **`rag.go` (RAG 核心流)**:
    *   Pipeline: `Query -> Embedding API -> Vector Search (PG) -> Rerank API -> Compile Prompt -> LLM API -> Stream Response`。

#### 2.1.2 API 接口升级 (`server/router/api/v1`)
新增 AI 相关服务接口：

*   **`Chat` (Streaming)**:
    *   输入: 用户问题。
    *   输出: SSE (Server-Sent Events) 流式回答。
    *   逻辑: 执行上述 RAG 流程，并包含引用来源 (Citations)。
*   **`Summarize`**:
    *   输入: Memo ID。
    *   输出: 摘要文本 + 建议标签。

#### 2.1.3 数据库迁移 (`server/store/migration`)
*   新增 Migration 文件，启用扩展: `CREATE EXTENSION IF NOT EXISTS vector;`
*   修改 `memos` 表 (或新建 `memo_vectors` 表) 增加 `embedding` 字段 (vector 类型)。

### 2.2 前端改造 (`web/`)

#### 2.2.1 新增组件 (`web/src/components/AI`)
*   **`AIChatDrawer.tsx`**:
    *   右侧滑出式聊天窗口。
    *   支持“针对全库提问”和“针对当前视图提问”。
*   **`ThinkingBubble.tsx`**:
    *   展示 AI 思考过程 (如 "正在检索...", "正在重排结果...")。

#### 2.2.2 编辑器增强 (`MemoEditor`)
*   增加 "AI Actions" 工具栏 (✨ 图标)。
    *   功能: 续写、润色、提取 Tag。

### 2.3 运维配置 (DevOps)

#### `docker-compose.yml` 调优
针对 2G 内存服务器的 PostgreSQL 关键配置：

```yaml
services:
  db:
    image: pgvector/pgvector:pg16  # 修正: 使用带 pgvector 扩展的镜像
    command: 
      - "postgres"
      - "-c"
      - "shared_buffers=128MB"       # 限制共享内存
      - "-c"
      - "work_mem=4MB"               # 限制每连接内存
      - "-c"
      - "maintenance_work_mem=64MB"  # 限制维护任务内存
      - "-c"
      - "max_connections=50"         # 降低最大连接数
    environment:
      - POSTGRES_DB=memos
      - POSTGRES_USER=memos
      - POSTGRES_PASSWORD=memos
    volumes:
      - ~/.memos-db:/var/lib/postgresql/data
```

## 3. 实施步骤 (Roadmap)

### Phase 1: 基础设施与后端基础 (预计 2 天)
1.  [ ] 修改 `docker-compose.yml`，引入 PG+pgvector。
2.  [ ] 编写 Go Migration 脚本，启用 vector 扩展。
3.  [ ] 实现 `server/ai/provider.go`，跑通 BAAI Embedding API 和 LLM API。

### Phase 2: 核心 RAG 链路 (预计 2 天)
1.  [ ] 实现文档切片与 Embedding 入库逻辑 (Create/Update Memo 时触发)。
2.  [ ] 实现 `server/ai/vector_store.go` 检索逻辑。
3.  [ ] 接入 Reranker API 优化检索结果。
4.  [ ] 开发 `Chat` 接口，输出流式响应。

### Phase 3: 前端交互与体验 (预计 3 天)
1.  [ ] 开发 `AIChatDrawer` 组件。
2.  [ ] 集成编辑器 AI 辅助功能。
3.  [ ] 整体联调与 Prompt 优化。

## 4. 风险与对策
*   **API 延迟**: Rerank 步骤会增加延迟。
    *   *对策*: 前端 UI 展示详细的 Progress Step (如 "正在重排...")，缓解用户等待焦虑。
*   **脏数据**: 旧数据没有向量化。
    *   *对策*: 提供一个 admin API 或 cli 命令 `memos bin reindex`，用于后台批量重建索引。
*   **内存溢出**: 2G 内存较紧张。
    *   *对策*: 严格限制 PG 内存；Go 程序中控制并发协程数量。
