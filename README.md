# Memos

> This project is a fork of [usememos/memos](https://github.com/usememos/memos).

Memos is a privacy-first, lightweight note-taking service with AI-powered parrot agents.

## Key Features

### 🦜 Parrot AI Agents

Memos uses a multi-agent architecture where specialized AI agents handle different tasks:

| Agent | Name | Role |
|-------|------|------|
| `DEFAULT` | 默认助手 | RAG-based chat with note context |
| `MEMO` | 灰灰 | Memo search and retrieval specialist |
| `SCHEDULE` | 金刚 | Schedule creation and management |
| `AMAZING` | 惊奇 | Comprehensive assistant (memo + schedule) |
| `CREATIVE` | 灵灵 | Creative writing and brainstorming |

**How it works**: Frontend specifies `agentType` in requests → Backend routes to corresponding Parrot Agent → Agent uses tools to complete tasks.

### 🧠 Intelligent RAG Pipeline
- **Adaptive Retrieval**: Combines BM25 and vector search with selective reranking
- **Smart Query Routing**: Detects intent and routes to appropriate agent
- **Tech Stack**: PostgreSQL + pgvector, SiliconFlow (bge-m3, bge-reranker-v2-m3), DeepSeek V3

### 📅 Smart Schedule Management
- **Natural Language**: "吃午饭" → creates schedule at selected date
- **Context-Aware**: Automatically uses calendar-selected date
- **Conflict Detection**: Backend validates and suggests alternatives
- **Tool-Based**: Agents use `schedule_add`, `schedule_query`, `find_free_time` tools

### 🛡️ Privacy First
- Fully self-hosted with no telemetry
- Markdown-native plain text
- High-performance Go backend + React frontend

## Getting Started

### Local Development

This project uses a `Makefile` to simplify development tasks.

**Prerequisites**:
- Go 1.25+
- Node.js & pnpm
- Docker (for database dependencies)

**Commands**:

1.  **Install Dependencies**:
    ```bash
    make deps-all
    ```

2.  **Start Development Environment**:
    ```bash
    make start
    ```
    This automatically starts the PostgreSQL container, backend server, and frontend dev server.
    - Frontend: http://localhost:25173
    - Backend: http://localhost:28081

3.  **Build**:
    - Backend: `make build`
    - Frontend: `make build-web`

### Docker

```bash
docker run -d \
  --name memos \
  -p 5230:5230 \
  -v ~/.memos:/var/opt/memos \
  neosmemo/memos:stable
```

## Tech Stack

- **Backend**: Go, Echo, gRPC-Gateway
- **Frontend**: React, Vite, TailwindCSS
- **Database**: PostgreSQL (Production), SQLite (Dev/Testing only)

## Database Support

| Database | Status | AI Features | Recommended Use |
|----------|--------|-------------|-----------------|
| PostgreSQL | ✅ Full Support | ✅ Vector, BM25, Hybrid, Reranking | Production |
| SQLite | ⚠️ Limited | ❌ No vector search | Development, Single-user |
| MySQL | ❌ Not Supported | ❌ | N/A (Removed) |

**Note**: MySQL support has been removed due to lack of AI features and high maintenance cost.

## License

[MIT](LICENSE)
