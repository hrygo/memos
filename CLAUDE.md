# CLAUDE.md

> **Guidance for Claude Code**: This file is your primary source of truth. Read this first.
> For detailed implementation guides, refer to the **Documentation Index** below.

## ⚡ Essentials (Tl;Dr)

**One-line Start**: `make start` (Up database -> Backend -> Frontend)

| Action    | Command          | Context                                   |
| :-------- | :--------------- | :---------------------------------------- |
| **Start** | `make start`     | Runs full stack (localhost:25173 / 28081) |
| **Stop**  | `make stop`      | Stops all services                        |
| **Test**  | `make test`      | Runs backend tests                        |
| **Build** | `make build-all` | Builds binary & static assets             |
| **Logs**  | `make logs`      | Views combined logs                       |

**Tech Stack summary**: Go 1.25, React 18 (Vite), PostgreSQL (Prod)/SQLite (Dev).

---

## ⚠️ Critical Rules

### 1. Internationalization (i18n) - **STRICT**
*   **No Hardcoding**: Never hardcode UI text. Use `t("key")`.
*   **Dual Entry**: Every key must exist in **BOTH** `en.json` and `zh-Hans.json`.
*   **Verify**: Run `make check-i18n` before committing.

### 2. Database Policy
*   **PostgreSQL**: **Primary**. Supports ALL AI features (vector search, reranking).
*   **SQLite**: **Limited**. Dev/Testing only. No complex AI/Hybrid search.
*   **MySQL**: **UNSUPPORTED**. Do not implement or suggest.

### 3. Code Style
*   **Go**: Standard layout (`cmd/`, `server/`, `store/`). Use `log/slog`.
*   **React**: PascalCase components. `use` prefix for hooks. `feature-based` naming.
*   **Agent Routing**: All AI chat logic routes via `ParrotRouter` (`server/router/api/v1/ai_service_chat.go`).

---

## 📚 Documentation Index

**Load these files ONLY when working on the specific domain:**

| Domain           | File Path                         | Content                                                           |
| :--------------- | :-------------------------------- | :---------------------------------------------------------------- |
| **Backend & DB** | `docs/dev-guides/BACKEND_DB.md`   | API Design, DB Policy, Docker workflows, Config (.env), Commands. |
| **Frontend**     | `docs/dev-guides/FRONTEND.md`     | Layout Architecture (Feature Layouts), Styling, Commands.         |
| **Architecture** | `docs/dev-guides/ARCHITECTURE.md` | Project Structure, Core Components, **Parrot Agent** details.     |
| **Agent Dev**    | `docs/dev-guides/QUICKSTART_AGENT.md` | Agent quick start guide and testing.                            |

### Quick Path Reference

*   **API Handlers**: `server/router/api/v1/`
*   **AI Agents**: `plugin/ai/agent/`
*   **Query Engine**: `server/queryengine/`
*   **Retrieval**: `server/retrieval/`
*   **Frontend Pages**: `web/src/pages/`
*   **Layouts**: `web/src/layouts/`
*   **DB Models**: `store/db/postgres/`

### Key Files

*   **Agent Routing**: `plugin/ai/agent/parrot_router.go`
*   **Scheduler Agent**: `plugin/ai/agent/scheduler.go`
*   **AI Chat Handler**: `server/router/api/v1/ai_service_chat.go`
