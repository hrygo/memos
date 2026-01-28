# CLAUDE.md

> Primary context for AI assistance. See Documentation Index below for detailed guides.

## Product Vision

**DivineSense (神识)**: AI agent-powered personal "second brain" — automates tasks, filters high-value information, amplifies productivity through technical leverage.

---

## Essentials

**Quick Start**: `make start` → localhost:25173 (Frontend) / 28081 (Backend)

| Command | Action |
|:--------|:-------|
| `make start` | Full stack up (PostgreSQL + Backend + Frontend) |
| `make stop` | Stop all services |
| `make test` | Run backend tests |
| `make build-all` | Build binary + static assets |
| `make check-all` | Run all pre-commit checks (build, test, i18n) |

**Tech Stack**: Go 1.25 + React 18 (Vite/Tailwind 4) + PostgreSQL (prod) / SQLite (dev)

---

## Critical Rules

### 1. Internationalization (i18n)
- **ALL UI text must use `t("key")`** — no hardcoded strings
- Translation keys must exist in **both** `en.json` and `zh-Hans.json`
- Verify: `make check-i18n`

### 2. Database Strategy
- **PostgreSQL**: Production environment. Full AI support (pgvector).
- **SQLite**: Development only. **No AI features available**.
- Always use PostgreSQL when testing AI-related features.

### 3. Code Style

**Go:**
- `snake_case.go` file naming
- Use `log/slog` for structured logging
- Follow standard Go project layout

**React/TypeScript:**
- PascalCase for components: `UserProfile.tsx`
- `use` prefix for hooks: `useUserData()`
- Use Tailwind CSS classes for styling (see below)

**AI Routing:**
- Backend `ChatRouter` handles intent classification
- Location: `plugin/ai/agent/chat_router.go`
- Rule-based matching (0ms) → LLM fallback (~400ms)
- Routes to: MEMO / SCHEDULE / AMAZING agents

### 4. Tailwind CSS 4 — CRITICAL

> **NEVER use semantic `max-w-sm/md/lg/xl`** — they resolve to ~16px in Tailwind v4.
>
> **Use explicit values**: `max-w-[24rem]`, `max-w-[28rem]`, etc.

See `docs/dev-guides/FRONTEND.md` for detailed Tailwind v4 gotchas.

### 5. Git Conventions

Follow conventional commits:

| Type | Scope | Example |
|:-----|:------|:--------|
| `feat` | Feature area | `feat(ai): add intent router` |
| `fix` | Bug area | `fix(db): resolve race condition` |
| `refactor` | Code area | `refactor(frontend): extract hooks` |
| `perf` | N/A | `perf(query): optimize vector search` |
| `docs` | N/A | `docs(readme): update quick start` |

**Format**: `<type>(<scope>): <description>`

**Always include**:
```
Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

### 6. Testing

- Run `make test` before committing
- AI tests: `make test-ai`
- For database changes, verify migrations work

---

## Documentation Index

| Domain | File | When to Reference |
|:--------|:-----|:------------------|
| **Backend** | `docs/dev-guides/BACKEND_DB.md` | API design, DB schema, Docker setup, .env config |
| **Frontend** | `docs/dev-guides/FRONTEND.md` | Layout structure, Tailwind pitfalls, component patterns |
| **Architecture** | `docs/dev-guides/ARCHITECTURE.md` | Project structure, AI agent system, data flow |

---

## Key Project Paths

| Area | Path | Purpose |
|:-----|:-----|:--------|
| API Handlers | `server/router/api/v1/` | REST/Connect RPC endpoints |
| AI Agents | `plugin/ai/agent/` | Parrot agents (MEMO, SCHEDULE, AMAZING) |
| AI Services | `plugin/ai/{memory,router,vector,aitime,cache,metrics,session}/` | AI infrastructure |
| Query Engine | `server/queryengine/` | Hybrid RAG retrieval (BM25 + vector) |
| Frontend Pages | `web/src/pages/` | Page components |
| Layouts | `web/src/layouts/` | Shared layout components |
| DB Models | `store/db/postgres/` | PostgreSQL models |
| DB Migrations | `store/migration/postgres/` | Schema migrations |

---

## Common Tasks

### Add a New API Endpoint
1. Create handler in `server/router/api/v1/`
2. Add route in `server/router/api/v1/routes.go`
3. Update proto files if using Connect RPC
4. Run `make check-build` to verify

### Add a New Frontend Page
1. Create component in `web/src/pages/`
2. Add route in `web/src/router/`
3. Add i18n keys to both `en.json` and `zh-Hans.json`
4. Run `make check-i18n` to verify

### Modify Database Schema
1. Create migration in `store/migration/postgres/`
2. Update models in `store/db/postgres/`
3. Test with `make db-reset` (dev environment only!)
4. Run `make test` to verify

### Add AI Feature
1. Determine agent type (MEMO/SCHEDULE/AMAZING)
2. Update agent in `plugin/ai/agent/`
3. Add routing rules in `chat_router.go`
4. Test with PostgreSQL (required for AI features)

---

## Pre-Commit Checklist

Before committing, run:

```bash
make check-all
```

This verifies:
- Build passes (`go build ./...`)
- Tests pass (`go test ./...`)
- i18n keys are complete

---

## Environment Variables

Key `.env` variables (see `.env.example`):

| Variable | Purpose | Default |
|:---------|:--------|:--------|
| `DIVINESENSE_DRIVER` | Database driver | `postgres` |
| `DIVINESENSE_DSN` | Database connection string | — |
| `DIVINESENSE_AI_ENABLED` | Enable AI features | `false` |
| `DIVINESENSE_AI_EMBEDDING_PROVIDER` | Embedding API provider | `siliconflow` |
| `DIVINESENSE_AI_LLM_PROVIDER` | LLM provider | `deepseek` |
| `SILICONFLOW_API_KEY` | SiliconFlow API key | — |
| `DEEPSEEK_API_KEY` | DeepSeek API key | — |
| `OPENAI_API_KEY` | OpenAI API key | — |

---

## Troubleshooting

| Issue | Solution |
|:------|:---------|
| AI not working | Ensure PostgreSQL is running and `DIVINESENSE_AI_ENABLED=true` |
| Tailwind styles broken | Use explicit values (`max-w-[24rem]`) not semantic (`max-w-md`) |
| i18n check fails | Add missing keys to both `web/src/locales/en.json` and `zh-Hans.json` |
| Build fails | Run `make deps` to update Go modules |
| Tests fail | Ensure PostgreSQL is running on port 25432 |
