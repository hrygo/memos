# Changelog

All notable changes to this project will be documented in this file.

## [v0.53.0] - 2026-01-26

### 📝 Documentation & Code Quality

- **Tailwind Grid Guidelines**: Added critical CSS pitfalls to CLAUDE.md - avoid `max-w-*` on Grid containers
- **Code Formatting**: Standardized AIChat component code style for consistency

---

## [v0.52.0] - 2026-01-25

### 💬 AI Chat Session Persistence

- **Conversation Memory**: AI conversations now persist across sessions with automatic context management
- **Context Separators**: Clear conversation context with visual separators (✂️) - prevents duplicate creation
- **Fixed Conversations**: 5 pinned conversations always visible in history (MEMO, SCHEDULE, AMAZING, CREATIVE, DEFAULT)
- **Real-time Message Count**: Message count updates immediately in conversation list (no page refresh needed)

### 📅 Schedule Optimization

- **Intelligent Conflict Resolution**: Auto-rescheduling with smart time slot suggestions
- **Enhanced Conflict Detection**: Improved detection of overlapping schedules
- **Recurrence Support**: Better handling of recurring events

### 🛡️ Security & Stability

- **Shell Hardening**: Deploy script now uses `tr` and `xargs` to sanitize environment variables
- **Goroutine Safety**: Added 5-second timeout protection for channel draining
- **Cross-platform**: Consistent file size checking using `wc -c` instead of `stat`

### 🔧 Refactoring

- **Parrot Framework**: Migrated DEFAULT parrot to standard parrot framework
- **Migration Consolidation**: PostgreSQL migrations consolidated to 0.51.0 baseline
- **Error Handling**: Improved error logging and DRY compliance

### 🚀 Deployment

- **Aliyun Production Scripts**: Complete deployment automation for Aliyun
- **China-Friendly Mirrors**: Docker registry and npm mirror configurations

---

## [v0.51.0] - 2026-01-23

### 📱 Mobile UI & UX Overhaul

- **Dynamic Navigation**: Fixed mobile header to display current Parrot Agent name and icon.
- **Streamlined Headers**: Simplified mobile sub-header to a single "Back to Nest" button for better chat immersion.
- **Interactive Feedback**: Added micro-scale touch feedback (`active:scale`) to all core buttons and agent cards.
- **Navigation Fix**: Resolved issue where clicking the Logo would cause the sidebar to flash and disappear.

### 🎨 Visual & i18n Polish

- **Unified Avatars**: All AI agents (including default assistant) now use high-quality image avatars instead of emojis.
- **Bilingual Identity**: Updated "Back" text to "返回鹦巢" / "Back to Nest" across en/zh-Hans/zh-Hant.
- **i18n Cleanup**: Optimized locale files by removing 50+ duplicate keys and fixing structure in all supported languages.

## [v0.50.0] - 2026-01-23

### 🦜 Parrot Multi-Agent System - First Release

- **Four Specialized Agents**: Complete implementation of Memo (灰灰), Schedule (金刚), Amazing (惊奇), and Creative (灵灵) Parrots
- **Agent Selection UI**: ParrotHub component with @-mention popover for quick agent switching
- **Metacognition API**: Agents now have self-awareness of capabilities, personality, and limitations
- **Bilingual Support**: Full i18n translations (en/zh-Hans) for all AI chat features
- **Static Assets**: Background images and icons for each parrot agent type
- **UI Polish**: Enhanced chat components with conflict detection and AI suggestions

### 🔧 Improvements

- **Performance**: Code cleanup and optimizations across web components
- **Refactoring**: Extracted common utilities to eliminate duplication
- **Schedule**: Week start day now defaults to Monday

## [v0.31.0] - 2026-01-21

### 🤖 Schedule Agent V2

- **Full Connect RPC Integration**: Migrated Schedule Agent to gRPC Connect protocols for robust streaming support.
- **Streaming Response**: Enabled real-time character streaming for smoother AI interactions, resolving previous gRPC-Gateway buffering issues.
- **Automated Testing Suite**: Added `scripts/test_schedule_agent.sh` and `QUICKSTART_AGENT.md` for comprehensive capabilities verification.
- **Agent Architecture**: Consolidated agent logic into `plugin/ai/agent/`, separating concerns between tools, core logic, and service layers.
- **Environment Management**: Improved dev scripts to handle `.env` loading and project root detection more intelligently.

## [v0.30.0] - 2026-01-21

### 📅 Intelligent Schedule Assistant

- **Smart Query Mode**: Introduced `AUTO`, `STANDARD`, and `STRICT` modes for precise schedule query control.
- **Explicit Year Support**: Parsing for full date formats (e.g., '2025年1月21日', '2025-01-21').
- **Relative Year Keywords**: Added support forms like "后年" (Year after next), "前年" (Year before last).

### 🧠 AI Architecture

- **Adaptive Retrieval**: Context-aware routing for Schedule vs Memo vs QA queries.
- **Query Optimization**: Enhanced filtering logic and schedule integration in search pipeline.

## [v0.26.1-ai.3] - 2026-01-21

### 📅 Schedule UI/UX Polish

- **Compact View**: Redesigned Schedule Calendar and Timeline for better information density and visual appeal.
- **Interaction Enhancements**: Unified "finger" cursors for all interactive elements, optimized "Today" button style.
- **Strict Conflict Policy**: Enforced backend conflict rules by removing "Create Anyway" and guiding users to "Modify/Adjust".
- **Date Formatting**: Standardized on "YYYY MMMM" format and Monday-start weeks.
- **Bug Fixes**: Resolved unused variables and React key warnings in Schedule components.

## [v0.26.1-ai.2] - 2026-01-21

### 🚀 Phase 1 Completion: Advanced AI Architecture

- **Adaptive Retrieval Engine**: Implemented a smart hybrid search system that dynamically switches between BM25 (keyword), Semantic (vector), and Hybrid strategies based on query intent.
- **Intelligent Query Routing**: Added `QueryRouter` to automatically classify user queries (Schedule vs. Memo vs. General QA) and route them to the most effective retrieval pipeline.
- **FinOps Cost Monitoring**: Integrated `CostMonitor` to track token usage and estimate costs for Embedding and LLM calls.
- **Service Modularization**: Refactored `AIService` into focused components (`ai_service_chat.go`, `ai_service_semantic.go`, `ai_service_intent.go`) for better maintainability.
- **Performance Optimization**: optimized Vector Search with parallelism and memory-efficient data structures.

## [v0.26.1-ai.1] - 2026-01-20

### ✨ New Features

- **AI Copilot Chat** - Interactive AI chat page with semantic search capabilities
- **Schedule Assistant** - New scheduling service with AI-powered time extraction
  - Proto definitions and gRPC/REST endpoints
  - Database migrations for MySQL, PostgreSQL, SQLite
  - Full CRUD operations for schedules

### 🔧 Improvements

- **Dev Scripts** - Improved `restart` command (app only, keeps PostgreSQL running)
- **Dev Scripts** - Fixed `stop` command to properly clean up orphan processes
- **i18n** - Simplified internationalization and improved language transition UX
- **Ports** - Updated development ports configuration

### 🐛 Bug Fixes

- Fixed "address already in use" errors after stop/restart
- Fixed `go run` orphan process cleanup on port binding
- Silenced secret context warnings in CI

### 📦 Infrastructure

- Refactored Docker setup for embedding store
- Removed deprecated dev container configs
- Cleaned up memos container service from `prod.yml`
