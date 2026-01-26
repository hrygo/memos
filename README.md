# Memos

**隐私优先的 AI 笔记助手** — 轻量级笔记 + 智能日程 + 多智能体 AI，数据完全由你掌控。

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB.svg)](https://react.dev/)

> 基于 [usememos/memos](https://github.com/usememos/memos) 二次开发，增强 AI 能力。

---

## 为什么选择 Memos？

| 🔒 **隐私优先** | 📝 **轻量笔记** | 🤖 **AI 增强** | 📅 **智能日程** |
|:---:|:---:|:---:|:---:|
| 自托管，无遥测<br/>数据完全私有 | 打开即写<br/>Markdown 支持 | 语义搜索<br/>智能问答 | 自然语言输入<br/>冲突检测 |

---

## 快速体验

**Docker 一键启动**（基础笔记功能，内置 SQLite）：

```bash
docker run -d --name memos -p 5230:5230 -v ~/.memos:/var/opt/memos hrygo/memos:stable
```

**启用 AI 功能**（需要 PostgreSQL + API Key）：

```bash
# 1. 克隆仓库
git clone https://github.com/hrygo/memos.git && cd memos

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 填入 API Key（见文件内详细说明）

# 3. 安装依赖
make deps-all

# 4. 启动 (PostgreSQL + 后端 + 前端)
make start
```

打开 http://localhost:25173 开始使用！

<details>
<summary><b>服务管理命令</b></summary>

```bash
make status   # 查看服务状态
make logs     # 查看日志
make stop     # 停止服务
make restart  # 重启服务
```

</details>

---

## 核心功能

### 📝 笔记管理

- **快速记录** — 打开即写，支持 Markdown
- **标签分类** — `#标签` 自动归类
- **时间线** — 按时间流浏览笔记
- **附件上传** — 图片、文件嵌入
- **搜索过滤** — 关键词、标签、时间范围

### 📅 日程管理

- **日历视图** — 月/周/日多视图切换
- **自然语言** — "明天下午3点开会" 直接创建
- **冲突检测** — 自动提醒时间冲突
- **拖拽调整** — 日历上直接拖动
- **重复规则** — 每天/周/月自动重复

### 🦜 AI 智能体

三个专业化的"鹦鹉智能体"协作处理不同任务：

| 智能体 | 专长 | 示例 |
|:---:|:---|:---|
| 🦜 **灰灰** | 笔记搜索 | "我之前写过关于 React 的笔记吗？" |
| 📅 **金刚** | 日程管理 | "帮我安排明天下午的会议" |
| ⭐ **惊奇** | 综合助手 | "总结一下本周的工作和日程" |

**智能路由**：输入后自动识别意图，无需手动选择。

---

## 技术亮点

<details>
<summary><b>混合 RAG 检索</b> — BM25 + 向量搜索 + 重排序</summary>

```
查询 → QueryRouter → BM25 + pgvector → Reranker → RRF 融合
```

- **向量搜索**: pgvector + HNSW 索引
- **全文搜索**: PostgreSQL FTS + BM25
- **重排序**: BAAI/bge-reranker-v2-m3
- **嵌入模型**: BAAI/bge-m3 (1024d)
- **LLM**: DeepSeek V3

</details>

<details>
<summary><b>系统架构</b></summary>

```
前端 (React + Vite)
    │ Connect RPC
后端 (Go + Echo)
    ├── API 服务层
    ├── 智能体层 (ChatRouter → Parrot Agents)
    └── 检索层 (QueryRouter + AdaptiveRetriever)
    │
存储 (PostgreSQL + pgvector) + AI 服务 (SiliconFlow/DeepSeek)
```

</details>

<details>
<summary><b>技术栈明细</b></summary>

| 层 | 技术 |
|:---|:---|
| 后端 | Go 1.25+, Echo, Connect RPC |
| 前端 | React 18, Vite 7, Tailwind CSS, Radix UI |
| 数据库 | PostgreSQL 16+ (pgvector) |
| AI | DeepSeek V3, bge-m3, bge-reranker-v2-m3 |

</details>

---

## 开发者

```bash
make start     # 启动所有服务
make stop      # 停止所有服务
make status    # 查看服务状态
make logs      # 查看日志
make test      # 运行测试
```

**开发文档**：
- [后端 & 数据库](docs/dev-guides/BACKEND_DB.md)
- [前端架构](docs/dev-guides/FRONTEND.md)
- [系统架构](docs/dev-guides/ARCHITECTURE.md)

---

## 许可证

[MIT](LICENSE) — 自由使用、修改、分发。
