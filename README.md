# DivineSense (神识)

**AI-Powered Personal Second Brain** — Automate tasks, filter information, amplify productivity.

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB.svg)](https://react.dev/)

> Forked from [usememos/memos](https://github.com/usememos/memos), enhanced with AI agents.

---

## Why DivineSense?

| **Efficiency** | **Knowledge** | **AI Agents** | **Privacy** |
|:--------------:|:-------------:|:-------------:|:-----------:|
| Automate tasks | Smart storage | Intent routing | Self-hosted |
| Save time | Semantic search | Multi-agent | Data privacy |

---

## Quick Start

### Docker (Basic Notes)

```bash
docker run -d --name divinesense -p 5230:5230 -v ~/.divinesense:/var/opt/divinesense hrygo/divinesense:stable
```

Access at http://localhost:5230

### Full AI Features (PostgreSQL Required)

```bash
# 1. Clone repository
git clone https://github.com/hrygo/divinesense.git && cd divinesense

# 2. Configure environment
cp .env.example .env
# Edit .env and add your API keys

# 3. Install dependencies
make deps-all

# 4. Start all services (PostgreSQL + Backend + Frontend)
make start
```

Access at http://localhost:25173

<details>
<summary><b>Service Management</b></summary>

```bash
make status   # Check service status
make logs     # View logs
make stop     # Stop services
make restart  # Restart services
```

</details>

---

## Features

### Note Taking
- Quick capture with Markdown support
- Tag-based organization (`#tag`)
- Timeline view
- File attachments
- Semantic search

### Schedule Management
- Calendar views (month/week/day)
- Natural language input
- Conflict detection
- Drag-and-drop rescheduling
- Recurring events

### AI Agents

Three specialized agents working together:

| Agent | Purpose | Example |
|:-----:|:--------|:--------|
| **HuiHui** | Knowledge | "What did I write about React?" |
| **JinGang** | Schedule | "Schedule tomorrow's meeting" |
| **Amazing** | Assistant | "Summarize my week" |

**Smart Routing**: Automatically detects intent — no manual agent selection needed.

**Session Memory**: Conversation context persists across sessions.

---

## Tech Stack

| Layer | Technology |
|:-----||:----------|
| Backend | Go 1.25+, Echo, Connect RPC |
| Frontend | React 18, Vite, Tailwind CSS, Radix UI |
| Database | PostgreSQL 16+ (pgvector) |
| AI | DeepSeek V3, bge-m3, bge-reranker-v2-m3 |

### Hybrid RAG Retrieval

```
Query → QueryRouter → BM25 + pgvector → Reranker → RRF Fusion
```

- **Vector Search**: pgvector + HNSW index
- **Full-Text**: PostgreSQL FTS + BM25
- **Reranker**: BAAI/bge-reranker-v2-m3
- **Embedding**: BAAI/bge-m3 (1024d)
- **LLM**: DeepSeek V3

---

## Development

```bash
make start     # Start all services
make stop      # Stop all services
make status    # Check service status
make logs      # View logs
make test      # Run tests
```

**Documentation**:
- [Backend & Database](docs/dev-guides/BACKEND_DB.md)
- [Frontend Architecture](docs/dev-guides/FRONTEND.md)
- [System Architecture](docs/dev-guides/ARCHITECTURE.md)

---

## License

[MIT](LICENSE) — Free to use, modify, and distribute.
