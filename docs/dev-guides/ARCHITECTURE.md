# Architecture & Context

## Project Overview

DivineSense (神识) is a privacy-first, lightweight note-taking service enhanced with AI-powered parrot agents.
- **Core Architecture**: Go backend (Echo/Connect RPC) + React frontend (Vite/Tailwind)
- **Data Storage**: PostgreSQL (production, full AI support), SQLite (development only, **no AI features**)
- **Key Features**: Multi-agent AI system, semantic search, schedule assistant, self-hosted with no telemetry
- **Ports**: Backend 28081, Frontend 25173, PostgreSQL 25432 (development)

## Tech Stack

| Area     | Technologies                                                        |
| -------- | ------------------------------------------------------------------- |
| Backend  | Go 1.25, Echo, Connect RPC, pgvector                                |
| Frontend | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI, TanStack Query |
| Database | PostgreSQL 16+ (production with AI), SQLite (dev only, **no AI**)  |
| AI       | DeepSeek V3 (LLM), SiliconFlow (Embedding, Reranker)                 |

---

## Project Architecture

### Directory Structure
```
divinesense/
├── cmd/divinesense/     # Main application entry point
├── server/              # HTTP/gRPC server & routing
│   ├── router/          # API handlers (v1 implementation)
│   ├── queryengine/     # Query routing & intent detection
│   ├── retrieval/       # Adaptive retrieval (BM25 + Vector)
│   ├── runner/          # Background task runners
│   ├── scheduler/       # Schedule management
│   └── service/         # Business logic layer
├── plugin/              # Plugin system
│   ├── ai/              # AI capabilities
│   │   ├── agent/       # Parrot agents (MemoParrot, ScheduleParrot, AmazingParrot)
│   │   ├── router/      # Three-layer intent routing
│   │   ├── vector/      # Embedding service
│   │   ├── memory/      # Episodic memory
│   │   ├── session/     # Conversation persistence
│   │   ├── cache/       # LRU cache layer
│   │   └── metrics/     # Agent performance tracking
│   ├── scheduler/       # Task scheduling
│   ├── storage/         # Storage adapters (S3, local)
│   └── idp/             # Identity providers
├── store/               # Data storage layer
│   ├── db/              # Database implementations
│   │   ├── postgres/    # PostgreSQL with pgvector
│   │   └── sqlite/      # SQLite (dev only, no AI)
│   └── [interfaces]     # Storage abstractions
├── proto/               # Protobuf definitions (API contracts)
│   ├── api/v1/          # API service definitions
│   └── store/           # Store service definitions
├── web/                 # React frontend application
│   ├── src/
│   │   ├── pages/       # Page components
│   │   ├── layouts/     # Layout components
│   │   ├── components/  # UI components
│   │   ├── locales/     # i18n translations (en, zh-Hans, zh-Hant)
│   │   └── hooks/       # React hooks
│   └── package.json
├── docs/                # Documentation
├── scripts/             # Development and build scripts
└── docker/              # Docker configurations
```

### Core Components

1. **Server Initialization**: Profile → DB → Store → Server
   - Uses Echo framework with Connect RPC for gRPC/HTTP transcoding
   - Auto-migration on startup

2. **Plugin System** (`plugin/ai/`):
   - LLM providers: DeepSeek, OpenAI, Ollama
   - Embedding: SiliconFlow (BAAI/bge-m3), OpenAI
   - Reranker: BAAI/bge-reranker-v2-m3
   - All AI features are optional (controlled by `DIVINESENSE_AI_ENABLED`)

3. **Background Runners** (`server/runner/`):
   - Async embedding generation for memos
   - Task queue system for AI operations
   - Runs automatically when AI is enabled

4. **Storage Layer**:
   - Interface definitions in `store/`
   - Driver-specific implementations in `store/db/{postgres,sqlite}/`
   - Migration system in `store/migration/`

5. **Intelligent Query Engine** (`server/queryengine/`):
   - Adaptive retrieval (BM25 + Vector search with selective reranking)
   - Smart query routing (detects schedule vs. search queries)
   - Natural language date parsing
   - Schedule assistant with conflict detection

---

## Parrot Agent Architecture

### Agent Types (`plugin/ai/agent/`)

| AgentType  | Parrot Name | File                 | Chinese Name | Description                               |
|:----------:|:-----------|:---------------------|:------------|:-----------------------------------------|
| `MEMO`     | HuiHui     | `memo_parrot.go`     | 灰灰         | Memo search and retrieval specialist      |
| `SCHEDULE` | JinGang    | `schedule_parrot_v2.go` | 金刚       | Schedule creation and management          |
| `AMAZING`  | Amazing    | `amazing_parrot.go`  | 惊奇         | Comprehensive assistant (memo + schedule) |

### Agent Router

**Location**: `plugin/ai/agent/chat_router.go`

The ChatRouter implements a **three-layer** intent classification system:

```
User Input → ChatRouter.Route()
                  ↓
           routerService? ─Yes→ Three-layer routing
                  │          (Rule + History + LLM)
                  │
                  No (backward compat)
                  ↓
           routeByRules()     ← Fast path (0ms)
                  ↓
         Match Found? ─Yes→ Return (confidence ≥0.80)
                  │
                  No
                  ↓
           routeByLLM()       ← Slow path (~400ms)
                  ↓
         Qwen2.5-7B-Instruct
         (Strict JSON Schema)
                  ↓
           Route Result
```

**Three-Layer Routing** (when `router.Service` is configured):
1. **Rule-based** (0ms): Keyword matching for common patterns
2. **History-aware** (~10ms): Conversation context matching
3. **LLM fallback** (~400ms): Semantic understanding for ambiguous inputs

**Rule-based Matching**:
- Schedule keywords: `日程`, `schedule`, `会议`, `meeting`, `提醒`, `remind`, time words (`今天`, `明天`, `周X`, `点`, `分`)
- Memo keywords: `笔记`, `memo`, `note`, `搜索`, `search`, `查找`, `find`, `写过`, `关于`
- Amazing keywords: `综合`, `总结`, `summary`, `本周工作`, `周报`

**LLM Fallback**:
- Model: `Qwen/Qwen2.5-7B-Instruct` via SiliconFlow
- Max tokens: 30 (minimal response)
- Strict JSON schema: `{"route": "memo|schedule|amazing", "confidence": 0.0-1.0}`

### Agent Tools

**Location**: `plugin/ai/agent/tools/`

| Tool            | File            | Description                          |
|:---------------|:----------------|:-------------------------------------|
| `memo_search`   | `memo_search.go` | Semantic memo search with RRF fusion |
| `schedule_add`  | `scheduler.go`   | Create new schedule                   |
| `schedule_query`| `scheduler.go`   | Query existing schedules              |
| `schedule_update`| `scheduler.go`  | Update existing schedule              |
| `find_free_time`| `scheduler.go`   | Find available time slots            |

### Schedule Agent V2

**Location**: `plugin/ai/agent/scheduler_v2.go`

Implements a native tool-calling loop (no ReAct needed for modern LLMs):

**Features**:
- Direct function calling with structured parameters
- Default 1-hour duration
- Automatic conflict detection
- Timezone-aware scheduling

---

## AI Services (`plugin/ai/`)

### Service Overview

| Service | Package | Description |
| ------- | ------- | ----------- |
| Memory | `memory/` | Episodic memory & user preferences |
| Session | `session/` | Conversation persistence (30-day retention) |
| Router | `router/` | Three-layer intent classification & routing |
| Cache | `cache/` | LRU cache with TTL for query results |
| Metrics | `metrics/` | Agent & tool performance tracking (A/B testing) |
| Vector | `vector/` | Embedding service with multiple providers |

### Session Service (`plugin/ai/session/`)

Provides conversation persistence for AI agents:

**Components**:
- `store.go`: PostgreSQL persistence + write-through cache (30min TTL)
- `recovery.go`: Session recovery workflow + sliding window (max 20 messages)
- `cleanup.go`: Background job for expired session cleanup (default: 30 days)

**Database**: `conversation_context` table (JSONB storage)

### Context Builder (`plugin/ai/context/`)

Assembles LLM context with intelligent token budget allocation:

```
Token Budget Allocation (with retrieval):
┌─────────────────────────────────────────┐
│ System Prompt      │ 500 tokens (fixed) │
│ User Preferences   │ 10%                │
│ Short-term Memory  │ 40%                │
│ Long-term Memory   │ 15%                │
│ Retrieval Results  │ 45%                │
└─────────────────────────────────────────┘
```

---

## Retrieval System (`server/retrieval/`)

### AdaptiveRetriever

Hybrid BM25 + Vector search with intelligent fusion:

| Strategy | Description |
|:--------|:------------|
| `BM25Only` | Keyword-only search (fast, low quality) |
| `SemanticOnly` | Vector-only search (slower, semantic) |
| `HybridStandard` | BM25 + Vector with RRF fusion (balanced) |
| `FullPipeline` | Hybrid + Reranker (highest quality, slowest) |

### RRF Fusion

Reciprocal Rank Fusion for merging BM25 and vector results:
```
score = Σ weight_i / (60 + rank_i)
```

### Reranker

BAAI/bge-reranker-v2-m3 for result refinement (configurable via strategy).

---

## Frontend Architecture (`web/src/`)

### Page Components

| Path | Component | Layout | Purpose |
|:-----|:----------|:-------|:--------|
| `/` | `Home.tsx` | MainLayout | Main timeline with memo composer |
| `/explore` | `Explore.tsx` | MainLayout | Search and explore content |
| `/archived` | `Archived.tsx` | MainLayout | Archived memos |
| `/chat` | `AIChat.tsx` | AIChatLayout | AI chat interface with auto-routing |
| `/schedule` | `Schedule.tsx` | ScheduleLayout | Calendar view with FullCalendar |
| `/review` | `Review.tsx` | MainLayout | Daily review |
| `/setting` | `Setting.tsx` | MainLayout | User settings |
| `/u/:username` | `UserProfile.tsx` | MainLayout | Public user profile |

### Layout Hierarchy

```
RootLayout (global Nav + auth)
    │
    ├── MainLayout (collapsible sidebar: MemoExplorer)
    │   └── /, /explore, /archived, /u/:username
    │
    ├── AIChatLayout (fixed sidebar: AIChatSidebar)
    │   └── /chat
    │
    └── ScheduleLayout (fixed sidebar: ScheduleCalendar)
        └── /schedule
```

---

## Data Flow

### AI Chat Flow

```
Frontend (AIChat.tsx)
    │ (WebSocket / SSE)
    ↓
Backend (ai_service_chat.go)
    │
    ↓ ChatRouter.Route()
    │   → Rule-based (0ms)
    │   → History-aware (~10ms)
    │   → LLM fallback (~400ms)
    ↓
Agent Execution
    │   → MemoParrot (memo_search tool)
    │   → ScheduleParrotV2 (scheduler tools)
    │   → AmazingParrot (concurrent tools)
    ↓
Response Streaming
    │   → Event types: thinking, tool_use, tool_result, answer
    ↓
Frontend UI Update
```

---

## AI Database Schema (PostgreSQL)

### Core Tables

| Table | Purpose |
|:-----|:--------|
| `memo_embedding` | Vector embeddings (1024d) for semantic search |
| `conversation_context` | Session persistence with 30-day retention |
| `episodic_memory` | Long-term user memory and learnings |
| `user_preferences` | User communication preferences |
| `agent_metrics` | A/B testing metrics (prompt versions, latency, success rate) |

---

## Environment Configuration

### Key Variables

```bash
# Database
DIVINESENSE_DRIVER=postgres
DIVINESENSE_DSN=postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable

# AI
DIVINESENSE_AI_ENABLED=true
DIVINESENSE_AI_EMBEDDING_PROVIDER=siliconflow
DIVINESENSE_AI_EMBEDDING_MODEL=BAAI/bge-m3
DIVINESENSE_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3
DIVINESENSE_AI_LLM_PROVIDER=deepseek
DIVINESENSE_AI_LLM_MODEL=deepseek-chat
DIVINESENSE_AI_DEEPSEEK_API_KEY=your_key
DIVINESENSE_AI_SILICONFLOW_API_KEY=your_key
DIVINESENSE_AI_OPENAI_BASE_URL=https://api.siliconflow.cn/v1
```
