# Architecture & Context

## Project Overview

Memos is a privacy-first, lightweight note-taking service with AI-powered parrot agents.
- **Core Architecture**: Go backend (Echo/Connect RPC) + React frontend (Vite/Tailwind)
- **Data Storage**: PostgreSQL (production, full AI support), SQLite (development only, **no AI features**)
- **Key Features**: Multi-agent AI system, semantic search, schedule assistant, self-hosted with no telemetry
- **Ports**: Backend 28081, Frontend 25173, PostgreSQL 25432 (development)

## Tech Stack

| Area     | Technologies                                                        |
| -------- | ------------------------------------------------------------------- |
| Backend  | Go 1.25, Echo, Connect RPC, pgvector                                |
| Frontend | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI, React Query |
| Database | PostgreSQL (production with AI), SQLite (dev only, **no AI**)       |
| AI       | DeepSeek (LLM), SiliconFlow (Embedding, Reranker)                   |

## Project Architecture

### Directory Structure
```
memos/
├── cmd/memos/           # Main application entry point
├── server/              # HTTP/gRPC server & routing
│   ├── router/          # API handlers (v1 implementation)
│   ├── queryengine/     # Query routing & intent detection
│   ├── retrieval/       # Adaptive retrieval (BM25 + Vector)
│   ├── runner/          # Background task runners
│   ├── scheduler/       # Schedule management
│   └── service/         # Business logic layer
├── plugin/              # Plugin system
│   ├── ai/              # AI capabilities
│   │   ├── agent/       # Parrot agents
│   │   ├── schedule/    # Schedule AI components
│   │   └── config.go    # AI configuration
│   ├── scheduler/       # Task scheduling
│   ├── storage/         # Storage adapters (S3, local)
│   └── idp/             # Identity providers
├── store/               # Data storage layer
│   ├── db/              # Database implementations
│   └── [interfaces]     # Storage abstractions
├── proto/               # Protobuf definitions (API contracts)
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
   - Driver-specific implementations in `store/db/{postgres,sqlite}/`
   - Migration system in `store/migration/`

5. **Intelligent Query Engine** (`server/queryengine/`):
   - Adaptive retrieval (BM25 + Vector search with selective reranking)
   - Smart query routing (detects schedule vs. search queries)
   - Natural language date parsing
   - Schedule assistant with conflict detection

## Parrot Agent Architecture

### Agent Types (`plugin/ai/agent/`)

| AgentType  | Parrot Name | File                 | Description                               |
| ---------- | ----------- | -------------------- | ----------------------------------------- |
| `MEMO`     | 灰灰        | `memo_parrot.go`     | Memo search and retrieval specialist      |
| `SCHEDULE` | 金刚        | `schedule_parrot.go` | Schedule creation and management          |
| `AMAZING`  | 惊奇        | `amazing_parrot.go`  | Comprehensive assistant (memo + schedule) |

### Agent Router

**Location**: `plugin/ai/agent/chat_router.go`

The ChatRouter implements a **hybrid Rule + LLM** intent classification system for intelligent agent routing:

```
User Input → ChatRouter.Route()
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

**Rule-based Matching**:
- Schedule keywords: 日程, schedule, 会议, meeting, 提醒, 时间词 (今天/明天/周X)
- Memo keywords: 笔记, memo, 搜索, 查找, 写过, 关于
- Amazing keywords: 综合, 总结一下, 本周工作, 周报

**LLM Fallback** (for ambiguous inputs):
- Model: `Qwen/Qwen2.5-7B-Instruct` via SiliconFlow
- Max tokens: 30 (minimal response)
- Strict JSON schema enforces valid output: `{"route": "memo|schedule|amazing", "confidence": 0.0-1.0}`

**Integration** (`server/router/api/v1/ai_service_chat.go`):
```go
func (s *AIService) createChatHandler() aichat.Handler {
    factory := aichat.NewAgentFactory(...)
    parrotHandler := aichat.NewParrotHandler(factory, s.LLMService)
    
    // Auto-routing enabled when IntentClassifier is configured
    if s.IntentClassifierConfig != nil && s.IntentClassifierConfig.Enabled {
        chatRouter := aichat.NewChatRouter(s.IntentClassifierConfig)
        parrotHandler.SetChatRouter(chatRouter)
    }
    return aichat.NewRoutingHandler(parrotHandler)
}
```

**Frontend**: Routing logic removed from `useCapabilityRouter.ts` - always sends `AUTO` type, backend decides.

### Schedule Agent

**Location**: `plugin/ai/agent/scheduler.go`

Implements a ReAct-style loop with tool execution:

**Tools** (`plugin/ai/agent/tools/scheduler.go`):
- `schedule_add`: Create new schedule
- `schedule_query`: Query existing schedules
- `schedule_update`: Update existing schedule
- `find_free_time`: Find available time slots

**System Prompt**: Directs LLM to extract date/time from input, default to 1-hour duration, and use selected date when unspecified.

**Frontend Integration** (`web/src/components/AIChat/ScheduleChatInput.tsx`):
```typescript
const message = buildScheduleMessage(userInput, selectedDate);
// Result: "当前选中日期: 2026-01-23\n吃午饭"

await chatHook.stream({
  message,
  agentType: ParrotAgentType.SCHEDULE,
  userTimezone: ...
});
```

### Agent Tools

**Location**: `plugin/ai/agent/tools/`

| Tool         | File           | Description                  |
| ------------ | -------------- | ---------------------------- |
| memo_search  | `memo_search.go` | Semantic memo search       |
| scheduler    | `scheduler.go`   | Schedule CRUD operations   |
