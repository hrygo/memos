# Backend & Database Guide

## Database Support Policy

### PostgreSQL (Production - Full Support)
- **Status**: Primary database for production use
- **AI Features**: Full support (pgvector, hybrid search, reranking)
- **Recommended for**: All production deployments
- **Maintained**: Actively maintained and tested
- **Port**: 25432 (development)

### SQLite (Development Only - No AI Features)
- **Status**: Development and testing only
- **AI Features**: **NOT SUPPORTED** - Conversation persistence, vector search, reranking disabled
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

## Backend Development

### Tech Stack
- **Language**: Go 1.25+
- **Framework**: Echo (HTTP) + Connect RPC (gRPC-HTTP transcoding)
- **Logging**: `log/slog`
- **Configuration**: Viper for environment variables

### API Design Pattern

1. **Protocol-first**: Modify `.proto` files in `proto/api/` or `proto/store/`
2. **Generate code**: Run `make generate` (if needed for proto changes)
3. **Implement handler**: Add implementation in `server/router/api/v1/`
4. **Storage layer**: Add interface in `store/` → implement in `store/db/{driver}/` → add migration

### Naming Conventions

- **Go files**: `snake_case.go` (e.g., `memo_embedding.go`)
- **Test files**: `*_test.go`
- **Go packages**: Simple lowercase, no underscores (e.g., `plugin/ai`, not `plugin/ai_service`)
- **Scripts**: `kebab-case.sh` (e.g., `dev.sh`)

## Common Development Commands

### Service Control
- `make start` / `make stop`: Start/stop all services
- `make status`: Check service status
- `make logs [backend|postgres]`: View logs
- `make logs-follow-backend`: Real-time backend logs
- `make run` / `make dev`: Start backend only (requires DB running first)
- `make web`: Start frontend only

### Docker (PostgreSQL)
- `make docker-up`: Start DB container
- `make docker-down`: Stop DB container
- `make db-connect`: Connect to PG shell
- `make db-reset`: Reset database schema (destructive)
- `make db-vector`: Verify pgvector extension

### Testing
- `make test`: Run all tests
- `make test-ai`: Run AI-related tests
- `make test-embedding`: Run embedding tests
- `make test-runner`: Run background runner tests
- `go test ./path/to/package -v`: Run specific package tests

### Building
- `make build`: Build backend binary
- `make build-web`: Build frontend static assets
- `make build-all`: Build both frontend and backend

### Dependencies
- `make deps-all`: Install all dependencies (backend, frontend, AI)

## Configuration (.env)

### Environment Variables

**Database:**
```bash
MEMOS_DRIVER=postgres
MEMOS_DSN=postgres://memos:memos@localhost:25432/memos?sslmode=disable
```

**AI (SiliconFlow/DeepSeek recommended):**
```bash
MEMOS_AI_ENABLED=true
MEMOS_AI_EMBEDDING_PROVIDER=siliconflow
MEMOS_AI_EMBEDDING_MODEL=BAAI/bge-m3
MEMOS_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3
MEMOS_AI_LLM_PROVIDER=deepseek
MEMOS_AI_LLM_MODEL=deepseek-chat
MEMOS_AI_DEEPSEEK_API_KEY=your_key
```

**Configuration Priority:**
1. System environment variables (direnv supported)
2. `.env` file
3. Code defaults

## Key Components

### AI Agent Routing
All AI chat logic routes through `ChatRouter` in `plugin/ai/agent/`:
- **MemoParrot** (灰灰): Memo search and retrieval
- **ScheduleParrot** (金刚): Schedule management
- **AmazingParrot** (惊奇): Combined memo + schedule

### Query Engine
Located in `server/queryengine/`:
- Intent detection and routing
- Smart query strategies based on time keywords
- Adaptive retrieval selection

### Retrieval System
Located in `server/retrieval/`:
- Hybrid BM25 + Vector search
- Reranking pipeline
- Caching layer

## AI Database Schema

### Core AI Tables (PostgreSQL only)

| Table | Purpose | Key Columns |
| ----- | ------- | ----------- |
| `memo_embedding` | Vector embeddings for semantic search | `memo_id`, `embedding` (vector) |
| `conversation_context` | Session persistence for AI agents | `session_id`, `user_id`, `context_data` (JSONB) |
| `episodic_memory` | Long-term user memory | `user_id`, `summary`, `embedding` |
| `user_preferences` | User communication preferences | `user_id`, `preferences` (JSONB) |
| `agent_metrics` | Agent performance tracking | `agent_type`, `success_rate`, `avg_latency` |

### conversation_context Schema

```sql
CREATE TABLE conversation_context (
  id            SERIAL PRIMARY KEY,
  session_id    VARCHAR(64) NOT NULL UNIQUE,
  user_id       INTEGER NOT NULL REFERENCES "user"(id),
  agent_type    VARCHAR(20) NOT NULL,  -- 'memo', 'schedule', 'amazing', 'assistant'
  context_data  JSONB NOT NULL,        -- messages + metadata
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
