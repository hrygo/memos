# Memos

> This project is a fork of [usememos/memos](https://github.com/usememos/memos).

Memos is a **privacy-first, AI-powered personal intelligence assistant** that combines lightweight note-taking with smart schedule management and multi-agent AI capabilities.

## ✨ Highlights

- 🦜 **Multi-Agent AI System** – Four specialized "Parrot Agents" handle different tasks with unique personalities
- 🧠 **Intelligent RAG Pipeline** – Hybrid retrieval with BM25 + Vector Search + Reranking for accurate results
- 📅 **Smart Schedule Management** – Natural language schedule creation with conflict detection
- 🔒 **Privacy First** – Self-hosted, no telemetry, your data stays yours

---

## 🦜 Parrot AI Agents

Memos uses a **multi-agent architecture** where specialized AI assistants (modeled after parrot species) handle different tasks:

| Agent        | Name     | Bird Species              | Specialization          | Key Capabilities                                                  |
| ------------ | -------- | ------------------------- | ----------------------- | ----------------------------------------------------------------- |
| 🦜 `MEMO`     | **灰灰** | 非洲灰鹦鹉 (African Grey) | Note Search & Retrieval | Semantic search, memo summary, RAG Q&A                            |
| 📅 `SCHEDULE` | **金刚** | 金刚鹦鹉 (Macaw)          | Schedule Management     | Create/query/update schedules, conflict detection, find free time |
| ⭐ `AMAZING`  | **惊奇** | 亚马逊鹦鹉 (Amazon)       | Comprehensive Assistant | Parallel memo + schedule retrieval, integrated analysis           |
| 💡 `CREATIVE` | **灵灵** | 虎皮鹦鹉 (Budgerigar)     | Creative Writing        | Brainstorming, content generation, text improvement               |

### Agent Selection

Use the `@` symbol in the AI chat to switch between agents, or click agent cards in the Parrot Hub.

### Agent Technical Details

<details>
<summary><b>🦜 灰灰 (MEMO) – Memory & Retrieval Specialist</b></summary>

**Working Style**: ReAct loop – search first, then answer based on retrieved evidence

**Tools**:
- `memo_search` – Semantic search across all memos with embedding similarity

**Fun Fact**: Named after the famous African Grey parrot Alex, who could understand 100+ vocabulary concepts!
</details>

<details>
<summary><b>📅 金刚 (SCHEDULE) – Time Management Expert</b></summary>

**Working Style**: ReAct loop with direct efficient approach – defaults to 1 hour duration, auto conflict detection

**Tools**:
- `schedule_add` – Create new schedules with automatic conflict check
- `schedule_query` – Query schedules by time range
- `schedule_update` – Modify existing schedules
- `find_free_time` – Find available time slots (8:00-22:00)

**Fun Fact**: Macaws are known for their punctuality in nature, always following consistent daily routines!
</details>

<details>
<summary><b>⭐ 惊奇 (AMAZING) – Comprehensive Multi-Task Assistant</b></summary>

**Working Style**: Two-phase concurrent retrieval – Intent Analysis → Parallel Tool Execution → Answer Synthesis

**Tools**: Combines capabilities of MEMO and SCHEDULE agents

**Fun Fact**: Amazon parrots are among the most talkative parrots – just like Amazing demonstrates multiple superpowers in one conversation!
</details>

<details>
<summary><b>💡 灵灵 (CREATIVE) – Creative Inspiration Muse</b></summary>

**Working Style**: Pure LLM creative mode – no tools, free imagination

**Fun Fact**: Budgerigars are the smallest parrots but have infinite creativity and vitality!
</details>

---

## 🧠 Core Technology Stack

### Intelligent RAG Pipeline

```
Query → QueryRouter → Cache Check
                         ├─ Cache Hit → Return (60% hit rate)
                         └─ Cache Miss → AdaptiveRetriever → Update Cache
                                              │
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                           BM25           Vector          Reranker
                       (PostgreSQL)    (pgvector)    (bge-reranker-v2-m3)
                              │               │               │
                              └───────────────┴───────────────┘
                                              │
                                          RRF Fusion → Results
```

| Component            | Technology              | Purpose                                               |
| -------------------- | ----------------------- | ----------------------------------------------------- |
| **Vector Search**    | pgvector + HNSW         | Similarity-based retrieval (m=16, ef_construction=64) |
| **Full-text Search** | PostgreSQL FTS + BM25   | Keyword-based retrieval with tsvector/GIN indexes     |
| **Reranker**         | BAAI/bge-reranker-v2-m3 | Cross-encoder reranking for precision                 |
| **Embedding**        | BAAI/bge-m3 (1024d)     | Dense vector embeddings via SiliconFlow               |
| **LLM**              | DeepSeek V3             | Reasoning, summarization, agent execution             |

### Smart Query Routing

The `QueryRouter` automatically detects query intent and routes to the optimal retrieval strategy:

| Strategy                      | Trigger                        | Use Case                |
| ----------------------------- | ------------------------------ | ----------------------- |
| `schedule_bm25_only`          | Time keywords ("今天", "本周") | Schedule queries        |
| `memo_semantic_only`          | Conceptual queries             | Pure vector search      |
| `hybrid_bm25_weighted`        | Mixed keywords                 | BM25 + Vector fusion    |
| `hybrid_with_time_filter`     | Time + keywords                | Filtered hybrid search  |
| `full_pipeline_with_reranker` | Complex queries                | Full RAG with reranking |

### Schedule Intelligence

- **Natural Language Parsing** – "明天下午3点开会" → creates schedule at tomorrow 15:00
- **Conflict Detection** – Automatic check for overlapping schedules
- **Free Time Finder** – Suggests available slots within 8:00-22:00 window
- **Recurrence Support** – RRULE-based repeating schedules (daily/weekly/monthly)

---

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                       Frontend (React + Vite)                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ Memo Editor │  │  Calendar   │  │    AI Chat + Parrot Hub │  │
│  │ + Attachment │  │   View     │  │    + Agent Selection    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │ Connect RPC (HTTP/2)
┌─────────────────────────────────────────────────────────────────┐
│                     Backend (Go + Echo)                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   MemoService   │  │ ScheduleService │  │   AIService     │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                  Parrot Agent Layer                          ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        ││
│  │  │MemoParrot│ │Schedule  │ │ Amazing  │ │Creative  │        ││
│  │  │  (灰灰)   │ │ Parrot   │ │ Parrot   │ │ Parrot   │        ││
│  │  │          │ │  (金刚)   │ │  (惊奇)   │ │  (灵灵)   │        ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘        ││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │  QueryRouter    │  │AdaptiveRetriever│  │  CostMonitor    │  │
│  │ (Intent Route)  │  │ (Hybrid Search) │  │   (FinOps)      │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                     Storage & AI Layer                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │   PostgreSQL    │  │  Redis (Opt)    │  │  AI Providers   │  │
│  │  ├─ memo        │  │  ├─ L1 Cache    │  │  ├─ Embedding   │  │
│  │  ├─ schedule    │  │  └─ Session     │  │  ├─ Reranker    │  │
│  │  ├─ pgvector    │  │                 │  │  └─ LLM         │  │
│  │  └─ memo_embed  │  │                 │  │                 │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Getting Started

### Prerequisites

- Go 1.25+
- Node.js 22+ & pnpm
- Docker (for PostgreSQL)

### Local Development

```bash
# 1. Clone the repository
git clone https://github.com/hrygo/memos.git
cd memos

# 2. Install dependencies
make deps-all

# 3. Start development environment
make start
```

This automatically starts:
- **PostgreSQL** (Docker container with pgvector)
- **Backend** at http://localhost:28081
- **Frontend** at http://localhost:25173

### Build

```bash
# Backend
make build

# Frontend
make build-web
```

### Docker Deployment

```bash
docker run -d \
  --name memos \
  -p 5230:5230 \
  -v ~/.memos:/var/opt/memos \
  hrygo/memos:stable
```

---

## 🛠️ Tech Stack

### Backend

| Component     | Technology          | Purpose                       |
| ------------- | ------------------- | ----------------------------- |
| Language      | Go 1.25+            | High-performance, concurrent  |
| Framework     | Echo + Connect RPC  | gRPC-HTTP transcoding         |
| Database      | PostgreSQL 16+      | Primary storage with pgvector |
| Vector Engine | pgvector (HNSW)     | Similarity search             |
| Caching       | Redis 7+ (optional) | L2 cache, session             |

### Frontend

| Component | Technology              | Purpose                       |
| --------- | ----------------------- | ----------------------------- |
| Framework | React 18                | Concurrent features, Suspense |
| Build     | Vite 7                  | Fast HMR, optimized builds    |
| State     | TanStack Query          | Server state, caching         |
| UI        | Radix UI + Tailwind CSS | Accessible, themeable         |
| Calendar  | FullCalendar            | Schedule visualization        |

### AI Services

| Service   | Provider    | Model                   |
| --------- | ----------- | ----------------------- |
| Embedding | SiliconFlow | BAAI/bge-m3 (1024d)     |
| Reranking | SiliconFlow | BAAI/bge-reranker-v2-m3 |
| LLM       | DeepSeek    | DeepSeek V3             |

---

## 📊 Database Support

| Database   | Status         | AI Features                       | Recommended Use  |
| ---------- | -------------- | --------------------------------- | ---------------- |
| PostgreSQL | ✅ Full Support | ✅ Vector, BM25, Hybrid, Reranking | Production       |
| SQLite     | ⚠️ Limited      | ❌ No vector search                | Development only |
| MySQL      | ❌ Removed      | ❌                                 | N/A              |

> **Note**: MySQL support has been removed due to lack of AI features.

---

## 📁 Project Structure

```
memos/
├── cmd/memos/                # Main application entry point
├── server/                   # Go backend
│   ├── router/api/v1/       # API handlers (Connect RPC)
│   ├── queryengine/         # Query routing & intent detection
│   ├── retrieval/           # Adaptive retrieval (BM25 + Vector)
│   ├── runner/              # Background task runners
│   ├── scheduler/           # Schedule management
│   └── service/             # Business logic layer
├── plugin/ai/               # AI components
│   ├── agent/               # Parrot agents
│   │   ├── memo_parrot.go
│   │   ├── schedule_parrot.go
│   │   ├── amazing_parrot.go
│   │   ├── creative_parrot.go
│   │   ├── scheduler.go     # Schedule agent orchestrator
│   │   └── tools/           # Agent tools (scheduler, memo_search)
│   ├── embedding.go         # Embedding service
│   ├── reranker.go          # Reranking service
│   └── llm.go               # LLM service
├── store/                   # Data storage layer
│   └── db/                  # Database implementations
│       ├── postgres/        # PostgreSQL (production)
│       └── sqlite/          # SQLite (development)
├── proto/                   # Protocol buffers
├── web/                     # React frontend
│   └── src/
│       ├── components/      # UI components
│       ├── layouts/         # Page layouts
│       ├── pages/           # Route pages
│       └── hooks/           # React hooks
├── docs/                    # Documentation
│   └── dev-guides/          # Developer guides
└── scripts/                 # Development & deployment scripts
```

---

## 📖 Documentation

### Developer Guides
| Document                                  | Description                                     |
| ----------------------------------------- | ----------------------------------------------- |
| [BACKEND_DB.md](docs/dev-guides/BACKEND_DB.md)   | Backend development & database policy           |
| [FRONTEND.md](docs/dev-guides/FRONTEND.md)       | Frontend architecture & layout patterns         |
| [ARCHITECTURE.md](docs/dev-guides/ARCHITECTURE.md) | Project architecture & Parrot Agent details     |
| [QUICKSTART_AGENT.md](docs/dev-guides/QUICKSTART_AGENT.md) | Agent quick start guide                         |

### Design Documents
| Document                                                                            | Description                                     |
| ----------------------------------------------------------------------------------- | ----------------------------------------------- |
| [MEMOS_REFACTOR_PLAN.md](docs/MEMOS_REFACTOR_PLAN.md)                               | Full refactoring roadmap (6-8 months, 5 phases) |
| [parrot-agents-final-technical-spec.md](docs/parrot-agents-final-technical-spec.md) | Technical specification v2.0                    |
| [parrot-agents-executive-summary-v2.md](docs/parrot-agents-executive-summary-v2.md) | Executive summary                               |

---

## 📄 License

[MIT](LICENSE)
