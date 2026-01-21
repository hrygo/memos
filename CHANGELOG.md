# Changelog

All notable changes to this project will be documented in this file.

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
