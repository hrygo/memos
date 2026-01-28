# Backend & Database Guide

## Database Support Policy

### PostgreSQL (Production - Full Support)
- **Status**: Primary database for production use
- **AI Features**: Full support (pgvector, hybrid search, reranking, session memory)
- **Recommended for**: All production deployments
- **Maintained**: Actively maintained and tested
- **Port**: 25432 (development)
- **Version**: PostgreSQL 16+

### SQLite (Development Only - No AI Features)
- **Status**: Development and testing only
- **AI Features**: **NOT SUPPORTED** - Vector search, conversation persistence, reranking disabled
- **Recommended for**: Local development of non-AI features only
- **Limitations**:
  - No AI conversation persistence (use PostgreSQL for AI features)
  - No vector search, BM25, hybrid search, or reranking
  - No concurrent write support
  - No full-text search (FTS5 not guaranteed)
- **Maintained**: Best-effort basis for non-AI features only
- **Migration**: Use PostgreSQL for production AI features

### MySQL (Removed)
- **Status**: **NOT SUPPORTED** - All MySQL support has been removed
- **Migration**: Use PostgreSQL for production or SQLite for development
- **Reason**: MySQL support was removed due to lack of AI features and maintenance burden

---

## Backend Development

### Tech Stack
- **Language**: Go 1.25+
- **Framework**: Echo (HTTP) + Connect RPC (gRPC-HTTP transcoding)
- **Logging**: `log/slog`
- **Configuration**: Environment variables via `.env` file

### API Design Pattern

1. **Protocol-first**: Modify `.proto` files in `proto/api/` or `proto/store/`
2. **Generate code**: Run `make generate` (if needed for proto changes)
3. **Implement handler**: Add implementation in `server/router/api/v1/`
4. **Storage layer**: Add interface in `store/` → implement in `store/db/{driver}/` → add migration

### Naming Conventions

| Type | Convention | Example |
|:-----|:-----------|:--------|
| Go files | `snake_case.go` | `memo_embedding.go` |
| Test files | `*_test.go` | `memo_parrot_test.go` |
| Go packages | Simple lowercase | `plugin/ai` (not `plugin/ai_service`) |
| Scripts | `kebab-case.sh` | `dev.sh` |
| Constants | `PascalCase` | `DefaultCacheTTL` |

---

## Common Development Commands

### Service Control
```bash
make start       # Start all services (PostgreSQL + Backend + Frontend)
make stop        # Stop all services
make status      # Check service status
make logs        # View all logs
make logs-backend # View backend logs
make logs-follow-backend # Real-time backend logs
make run         # Start backend only (requires DB running first)
make web         # Start frontend only
```

### Docker (PostgreSQL)
```bash
make docker-up      # Start DB container
make docker-down    # Stop DB container
make db-connect     # Connect to PG shell
make db-reset       # Reset database schema (destructive!)
make db-vector      # Verify pgvector extension
```

### Testing
```bash
make test           # Run all tests
make test-ai        # Run AI-related tests
make test-embedding  # Run embedding tests
make test-runner    # Run background runner tests
go test ./path/to/package -v  # Run specific package tests
```

### Building
```bash
make build       # Build backend binary
make build-web   # Build frontend static assets
make build-all   # Build both frontend and backend
```

### Dependencies
```bash
make deps-all    # Install all dependencies (backend, frontend, AI)
make deps        # Install backend dependencies only
make deps-web    # Install frontend dependencies only
make deps-ai     # Install AI dependencies only
```

---

## Configuration (.env)

### Environment Variables

**Database:**
```bash
DIVINESENSE_DRIVER=postgres
DIVINESENSE_DSN=postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable
```

**AI (SiliconFlow/DeepSeek recommended):**
```bash
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

**Configuration Priority:**
1. System environment variables (direnv supported)
2. `.env` file
3. Code defaults

---

## Key Components

### AI Agent System

All AI chat logic routes through `ChatRouter` in `plugin/ai/agent/`:

| Agent | File | Purpose | Tools |
|:-----|:-----|:--------|:------|
| **MemoParrot** | `memo_parrot.go` | Memo search and retrieval | `memo_search` |
| **ScheduleParrotV2** | `schedule_parrot_v2.go` | Schedule management | `schedule_add`, `schedule_query`, `schedule_update`, `find_free_time` |
| **AmazingParrot** | `amazing_parrot.go` | Combined memo + schedule | All tools with concurrent execution |

**Chat Routing Flow** (`chat_router.go`):
```
Input → Rule-based (0ms) → History-aware (~10ms) → LLM fallback (~400ms)
         ↓                    ↓                      ↓
      Keywords          Conversation context    Semantic understanding
```

### Query Engine

Located in `server/queryengine/`:
- Intent detection and routing
- Smart query strategies based on time keywords
- Adaptive retrieval selection

### Retrieval System

Located in `server/retrieval/`:
- Hybrid BM25 + Vector search (`AdaptiveRetriever`)
- Reranking pipeline
- LRU cache layer for query results

---

## AI Database Schema (PostgreSQL)

### Core AI Tables

| Table | Purpose | Key Columns |
|:-----|:--------|:------------|
| `memo_embedding` | Vector embeddings for semantic search | `memo_id`, `embedding` (vector(1024)) |
| `conversation_context` | Session persistence for AI agents | `session_id`, `user_id`, `context_data` (JSONB) |
| `episodic_memory` | Long-term user memory | `user_id`, `summary`, `embedding` (vector) |
| `user_preferences` | User communication preferences | `user_id`, `preferences` (JSONB) |
| `agent_metrics` | Agent performance tracking | `agent_type`, `prompt_version`, `success_rate`, `avg_latency` |

### conversation_context Schema

```sql
CREATE TABLE conversation_context (
  id            SERIAL PRIMARY KEY,
  session_id    VARCHAR(64) NOT NULL UNIQUE,
  user_id       INTEGER NOT NULL REFERENCES "user"(id),
  agent_type    VARCHAR(20) NOT NULL,  -- 'memo', 'schedule', 'amazing'
  context_data  JSONB NOT NULL,         -- messages + metadata
  created_ts    BIGINT NOT NULL,
  updated_ts    BIGINT NOT NULL
);

-- Indexes
CREATE INDEX idx_conversation_context_user ON conversation_context(user_id);
CREATE INDEX idx_conversation_context_updated ON conversation_context(updated_ts DESC);
```

**context_data Structure**:
```json
{
  "messages": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ],
  "metadata": {"topic": "...", ...}
}
```

**Retention**: Sessions auto-expire after 30 days (configurable via cleanup job).

### agent_metrics Schema

```sql
CREATE TABLE agent_metrics (
  id             SERIAL PRIMARY KEY,
  agent_type     VARCHAR(20) NOT NULL,
  prompt_version VARCHAR(20) NOT NULL,  -- A/B testing
  success_count  INTEGER DEFAULT 0,
  failure_count  INTEGER DEFAULT 0,
  avg_latency_ms BIGINT DEFAULT 0,
  updated_ts     BIGINT NOT NULL
);
```

---

## Directory Structure

| Path | Purpose |
|:-----|:--------|
| `cmd/divinesense/` | Main application entry point |
| `server/router/api/v1/` | REST/Connect RPC API handlers |
| `server/service/` | Business logic layer |
| `server/retrieval/` | Hybrid search (BM25 + vector) |
| `server/queryengine/` | Query analysis and routing |
| `plugin/ai/agent/` | AI agents (MemoParrot, ScheduleParrot, AmazingParrot) |
| `plugin/ai/router/` | Three-layer intent routing |
| `plugin/ai/vector/` | Embedding service |
| `store/` | Data access layer interface |
| `store/db/postgres/` | PostgreSQL implementation |
| `store/migration/postgres/` | Database migrations |
| `proto/api/v1/` | Connect RPC protocol definitions |
| `proto/store/` | Store protocol definitions |
| `web/` | Frontend (React + Vite) |
