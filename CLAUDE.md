# CLAUDE.md

> **助手指南**: 本文件为 Claude Code (claude.ai/code) 提供项目上下文与操作指南。

## 🌟 项目概述

**Memos** 是一个开源、自托管的笔记服务，支持 AI 增强功能（向量嵌入、语义搜索、LLM 聊天）。
*   **核心架构**: Go 后端 (Echo/Connect RPC) + React 前端 (Vite/Tailwind)。
*   **数据存储**: 支持 PostgreSQL (推荐, 支持 AI)、SQLite (支持 AI) 和 MySQL (仅基础功能)。

---

## ⚡ 快速开始

最常用的开发命令：

```bash
# 🚀 启动完整开发环境 (PostgreSQL -> 后端 -> 前端)
make start

# 🛑 停止所有服务
make stop

# 📜 查看日志 (api/web/db)
make logs
```

---

## 🛠 技术栈概览

| 领域       | 核心技术                                                            |
| :--------- | :------------------------------------------------------------------ |
| **后端**   | Go 1.25, Echo, Connect RPC, LangchainGo, pgvector                   |
| **前端**   | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI, React Query |
| **数据库** | PostgreSQL (生产推荐), SQLite (轻量级), MySQL (遗留)                |

---

<details>
<summary><strong>📦 常用命令详解 (Testing, Building, Docker)</strong></summary>

### 服务控制
*   `make start` / `make stop`: 启动/停止所有服务
*   `make status`: 查看服务状态
*   `make logs [backend|postgres]`: 查看日志
*   `make run` / `make dev`: 单独启动后端 (需先启动 DB)
*   `make web`: 单独启动前端

### Docker 管理 (PostgreSQL)
*   `make docker-up`: 启动 DB 容器
*   `make docker-down`: 停止 DB 容器
*   `make db-connect`: 连接 PG Shell
*   `make db-reset`: 重置 Schema
*   `make db-vector`: 验证 pgvector 扩展

### 测试 (Testing)
*   `make test`: 运行所有测试
*   `make test-ai`: 运行 AI 相关测试
*   `make test-embedding`: 运行 Embedding 测试
*   `make test-runner`: 运行 Runner 测试
*   `go test ./path/to/package -v`: 运行指定包测试

### 构建 (Building)
*   `make build`: 构建后端二进制
*   `make build-web`: 构建前端静态资源
*   `make build-all`: 构建全部

### 依赖管理
*   `make deps-all`: 安装前后端及 AI 依赖
</details>

<details>
<summary><strong>🏗 项目架构与目录结构</strong></summary>

### 目录结构
```text
memos/
├── cmd/memos/           # 🚀 主程序入口
├── server/              # 🌐 HTTP/gRPC 服务器 & 路由
├── plugin/              # 🔌 插件 (AI, 存储, Webhook)
├── store/               # 💾 数据存储层 (PG, SQLite, MySQL)
├── proto/               # 📜 Protobuf 定义 (API 契约)
├── web/                 # 🎨 React 前端应用
└── scripts/             # 🛠 开发脚本
```

### 核心组件
1.  **Server**: 基于 Echo 与 Connect RPC，启动时初始化 Profile -> DB -> Store -> Server。
2.  **Plugin System**: `plugin/ai/` 封装了 LLM、Embedding 和 Reranker 能力。
3.  **Runner**: `server/runner/` 处理后台异步任务（如生成向量嵌入）。
4.  **Database**:
    *   **Store Interface**: 定义在 `store/`。
    *   **Implementation**: `store/db/postgres` 等具体实现。
    *   **Migration**: `store/migration/` 管理版本迁移。

详细信息参考: `docs/PROJECT_STRUCTURE.md`
</details>

<details>
<summary><strong>⚙️ 环境变量与配置 (.env)</strong></summary>

请在根目录 `.env` 文件中配置。

### 基础配置
```bash
MEMOS_DRIVER=postgres
MEMOS_DSN=postgres://memos:memos@localhost:25432/memos?sslmode=disable
```

### AI 功能配置 (推荐 SiliconFlow / DeepSeek)
```bash
# 开关
MEMOS_AI_ENABLED=true

# Embedding (向量化)
MEMOS_AI_EMBEDDING_PROVIDER=siliconflow
MEMOS_AI_EMBEDDING_MODEL=BAAI/bge-m3

# Reranker (重排序)
MEMOS_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3

# LLM (大模型)
MEMOS_AI_LLM_PROVIDER=deepseek
MEMOS_AI_LLM_MODEL=deepseek-chat
MEMOS_AI_DEEPSEEK_API_KEY=your_key
```
</details>

<details>
<summary><strong>📝 开发规范与流程</strong></summary>

### Go 后端
*   **风格**: 遵循 Standard Go Project Layout。
*   **命名**: 文件名 `snake_case.go`，测试文件 `_test.go`。
*   **日志**: 使用 `log/slog`。
*   **配置**: 使用 Viper 读取环境变量。

### React 前端
*   **组件**: PascalCase (如 `MemoEditor.tsx`)。
*   **Hooks**: `use` 前缀 (如 `useMemoList.ts`)。
*   **样式**: Tailwind CSS 4 为主。
*   **国际化**: `web/src/locales/`。

### 新功能工作流
1.  **API**: 修改 `proto/api/` -> `make generate` -> 实现 `server/router/`。
2.  **DB**: 修改 `proto/store/` -> 添加 `store/` 接口 -> 实现 `store/db/` -> 添加迁移。
3.  **Plugin**: 在 `plugin/` 下实现新接口 -> `server/` 注册。
</details>

<details>
<summary><strong>❓ 常见问题排查 (Troubleshooting)</strong></summary>

*   **后端启动失败**:
    *   检查 Docker 容器: `make docker-up`
    *   检查 DB 连接: `make db-connect`
*   **AI 功能不可用**:
    *   **MySQL 不支持 AI**，请切换 PG 或 SQLite。
    *   确认 `MEMOS_AI_ENABLED=true`。
    *   验证 `pgvector`: 运行 `make db-vector`。
*   **前端端口**: 开发服务器默认在 `25173`，后端在 `28081`。
</details>
