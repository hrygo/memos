# Architecture & Context

## Project Overview

Memos is a privacy-first, lightweight note-taking service with AI-powered parrot agents.
- **Core Architecture**: Go backend (Echo/Connect RPC) + React frontend (Vite/Tailwind)
- **Data Storage**: PostgreSQL (production, full AI support), SQLite (limited, dev/testing only)
- **Key Features**: Multi-agent AI system, semantic search, schedule assistant, self-hosted with no telemetry

## Tech Stack

| Area     | Technologies                                                        |
| -------- | ------------------------------------------------------------------- |
| Backend  | Go 1.25, Echo, Connect RPC, LangchainGo, pgvector                   |
| Frontend | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI, React Query |
| Database | PostgreSQL (production), SQLite (dev/testing only)                  |

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
│   ├── db/              # Database implementations (PostgreSQL, SQLite)
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

## Parrot Agent Architecture

### Agent Types (`plugin/ai/agent/`)

| AgentType  | Parrot Name | Description                               | File                                 |
| ---------- | ----------- | ----------------------------------------- | ------------------------------------ |
| `DEFAULT`  | 默认助手    | RAG-based chat with memo context          | -                                    |
| `MEMO`     | 灰灰        | Memo search and retrieval specialist      | `memo_parrot.go`                     |
| `SCHEDULE` | 金刚        | Schedule creation and management          | `schedule_parrot.go`, `scheduler.go` |
| `AMAZING`  | 惊奇        | Comprehensive assistant (memo + schedule) | `amazing_parrot.go`                  |
| `CREATIVE` | 灵灵        | Creative writing and brainstorming        | `creative_parrot.go`                 |

### Agent Routing (`server/router/api/v1/ai_service_chat.go`)

**Routing Logic** (line 179-181):
```go
if req.AgentType != v1pb.AgentType_AGENT_TYPE_DEFAULT {
    return s.chatWithParrot(ctx, req, req, stream)  // Parrot Agent path
}
// Otherwise: DEFAULT agent path (legacy RAG)
```

**Frontend Usage**: Set `agentType` in `ChatWithMemosRequest` to route to specific agent.

### Schedule Agent (`scheduler.go`)

The Schedule Agent implements a ReAct-style loop with tool execution:

**Tools** (`plugin/ai/agent/tools/scheduler.go`):
- `schedule_add`: Create new schedule
- `schedule_query`: Query existing schedules
- `schedule_update`: Update existing schedule
- `find_free_time`: Find available time slots

**System Prompt**: Directs LLM to extract date/time from input, default to 1-hour duration, and use selected date when unspecified.

**Frontend Integration** (`ScheduleChatInput.tsx`):
```typescript
const message = buildScheduleMessage(userInput, selectedDate);
// Result: "当前选中日期: 2026-01-23\n吃午饭"

await chatHook.stream({
  message,
  agentType: ParrotAgentType.SCHEDULE,
  userTimezone: ...
});
```
