# CLAUDE.md

> **Primary source of truth**. For detailed guides, see **Documentation Index** below.

## ⚡ Essentials

**Start**: `make start` → localhost:25173 (Frontend) / 28081 (Backend)

| Command          | Action                      |
| :--------------- | :-------------------------- |
| `make start`     | Full stack up               |
| `make stop`      | Stop all                    |
| `make test`      | Backend tests               |
| `make build-all` | Build binary + static assets|

**Stack**: Go 1.25 + React 18 (Vite/Tailwind 4) + PostgreSQL (Prod) / SQLite (Dev)

---

## ⚠️ Critical Rules

### 1. i18n - No hardcoded text
- Use `t("key")` for all UI text
- Keys must exist in **both** `en.json` and `zh-Hans.json`
- Verify: `make check-i18n`

### 2. Database
- **PostgreSQL**: Production. Full AI support.
- **SQLite**: Dev only. **No AI features**.
- **MySQL**: Not supported.

### 3. Code Style
- **Go**: `snake_case.go`, `log/slog`
- **React**: PascalCase components, `use` prefix hooks
- **AI Routing**: Backend `ChatRouter` handles intent classification (`plugin/ai/agent/chat_router.go`)
  - Rule-based matching (0ms) → LLM fallback (~400ms) for uncertain inputs
  - Routes to: MEMO / SCHEDULE / AMAZING agents

### 4. Tailwind CSS 4 - **CRITICAL**
> **Never use semantic `max-w-sm/md/lg/xl`** - they resolve to ~16px in Tailwind 4.
> Use explicit values: `max-w-[24rem]`, `max-w-[28rem]`, etc.
> See `docs/dev-guides/FRONTEND.md` for details.

---

## 📚 Documentation Index

| Domain       | File                              | When to Load                     |
| :----------- | :-------------------------------- | :------------------------------- |
| **Backend**  | `docs/dev-guides/BACKEND_DB.md`   | API, DB, Docker, .env            |
| **Frontend** | `docs/dev-guides/FRONTEND.md`     | Layout, Tailwind pitfalls        |
| **Architecture** | `docs/dev-guides/ARCHITECTURE.md` | Project structure, AI agents |

### Key Paths

| Area           | Path                          |
| :------------- | :---------------------------- |
| API Handlers   | `server/router/api/v1/`       |
| AI Agents      | `plugin/ai/agent/`            |
| Query Engine   | `server/queryengine/`         |
| Frontend Pages | `web/src/pages/`              |
| Layouts        | `web/src/layouts/`            |
| DB Models      | `store/db/postgres/`          |
