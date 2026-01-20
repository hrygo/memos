# Changelog

All notable changes to this project will be documented in this file.

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
