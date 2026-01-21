# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Memos is a privacy-first, lightweight note-taking service with advanced AI capabilities.
- **Core Architecture**: Go backend (Echo/Connect RPC) + React frontend (Vite/Tailwind)
- **Data Storage**: PostgreSQL (recommended, full AI support), SQLite (AI supported), MySQL (basic features only)
- **Key Features**: Semantic search, AI chat integration, schedule assistant, self-hosted with no telemetry

## Quick Start

```bash
# Start complete development environment (PostgreSQL -> Backend -> Frontend)
make start

# Stop all services
make stop

# View logs
make logs
```

Services:
- Frontend: http://localhost:25173
- Backend: http://localhost:28081
- PostgreSQL: localhost:25432

## Tech Stack

| Area     | Technologies                                                       |
|----------|--------------------------------------------------------------------|
| Backend  | Go 1.25, Echo, Connect RPC, LangchainGo, pgvector                 |
| Frontend | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI, React Query |
| Database | PostgreSQL (production), SQLite (lightweight), MySQL (legacy)      |

## Common Development Commands

### Service Control
- `make start` / `make stop`: Start/stop all services
- `make status`: Check service status
- `make logs [backend|postgres]`: View logs
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

## Project Architecture

### Directory Structure
```
memos/
├── cmd/memos/           # Main application entry point
├── server/              # HTTP/gRPC server & routing
│   ├── router/          # API handlers (v1 implementation)
│   ├── runner/          # Background task runners (e.g., embedding generation)
│   └── auth/            # Authentication & authorization
├── plugin/              # Plugin system
│   ├── ai/              # AI capabilities (Embedding, LLM, Reranker)
│   ├── scheduler/       # Task scheduling
│   ├── storage/         # Storage adapters (S3, local)
│   └── idp/             # Identity providers
├── store/               # Data storage layer
│   ├── db/              # Database implementations (PostgreSQL, SQLite, MySQL)
│   └── [interfaces]     # Storage abstractions
├── proto/               # Protobuf definitions (API contracts)
│   ├── api/             # API definitions
│   └── store/           # Storage definitions
├── web/                 # React frontend application
└── scripts/             # Development and build scripts
```

### Core Components

1. **Server Initialization**: Profile → DB → Store → Server
   - Uses Echo framework with Connect RPC for gRPC/HTTP
   - Auto-migration on startup

2. **Plugin System** (`plugin/ai/`):
   - LLM providers: DeepSeek, OpenAI, Ollama
   - Embedding: SiliconFlow (BAAI/bge-m3), OpenAI
   - Reranker: BAAI/bge-reranker-v2-m3
   - All AI features are optional (controlled by `MEMOS_AI_ENABLED`)

3. **Background Runners** (`server/runner/`):
   - Async embedding generation for memos
   - Task queue system for AI operations
   - Runs automatically when AI is enabled

4. **Storage Layer**:
   - Interface definitions in `store/`
   - Driver-specific implementations in `store/db/{postgres,sqlite,mysql}/`
   - Migration system in `store/migration/`

5. **Intelligent Query Engine**:
   - Adaptive retrieval (BM25 + Vector search with selective reranking)
   - Smart query routing (detects schedule vs. search queries)
   - Natural language date parsing
   - Schedule assistant with conflict detection

### API Design Pattern

1. **Protocol-first**: Modify `.proto` files in `proto/api/` or `proto/store/`
2. **Generate code**: Run `make generate` (if needed for proto changes)
3. **Implement handler**: Add implementation in `server/router/api/v1/`
4. **Storage layer**: Add interface in `store/` → implement in `store/db/{driver}/` → add migration

### Naming Conventions

- **Go files**: `snake_case.go` (e.g., `memo_embedding.go`)
- **Test files**: `*_test.go`
- **Go packages**: Simple lowercase, no underscores (e.g., `plugin/ai`, not `plugin/ai_service`)
- **React components**: PascalCase (e.g., `MemoEditor.tsx`)
- **React hooks**: `use` prefix (e.g., `useMemoList.ts`)
- **Scripts**: `kebab-case.sh` (e.g., `dev.sh`)

## Configuration

### Environment Variables (.env)

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

## Development Workflow

### Backend Development
- **Style**: Standard Go Project Layout
- **Logging**: Use `log/slog`
- **Configuration**: Viper for environment variables

### Frontend Development
- **Commands** (run in `web/` directory):
  - `pnpm dev`: Start dev server
  - `pnpm build`: Build for production
  - `pnpm lint`: Run TypeScript and Biome checks
  - `pnpm lint:fix`: Auto-fix linting issues
- **Styling**: Tailwind CSS 4 (primary), Radix UI components
- **State**: TanStack Query (React Query)
- **Internationalization**: `web/src/locales/`
- **Markdown**: React Markdown with KaTeX, Mermaid, GFM support

## Important Constraints

### AI Feature Support
- **PostgreSQL**: Full AI support (pgvector, hybrid search, reranking)
- **SQLite**: Basic AI support (vectors via sqlite-vec)
- **MySQL**: No AI support (legacy only)

### Common Issues

1. **Backend startup fails:**
   - Check Docker container: `make docker-up`
   - Check DB connection: `make db-connect`

2. **AI features unavailable:**
   - MySQL doesn't support AI, switch to PG or SQLite
   - Confirm `MEMOS_AI_ENABLED=true`
   - Verify pgvector: `make db-vector`

3. **Port conflicts:**
   - Frontend defaults to `25173`
   - Backend defaults to `28081`
   - PostgreSQL defaults to `25432`

## Testing with AI Features

When running tests with AI functionality:
1. Ensure PostgreSQL is running: `make docker-up`
2. Set environment variables for AI providers
3. Use specific test targets: `make test-ai`, `make test-embedding`

## Documentation

- `docs/PROJECT_STRUCTURE.md`: Detailed architecture
- `docs/ai-implementation-plan.md`: AI feature specifications
- `docs/schedule-assistant-implementation-plan.md`: Schedule feature specs
