# Memos

> This project is a fork of [usememos/memos](https://github.com/usememos/memos).

Memos is a privacy-first, lightweight note-taking service.

## Key Features

### 🧠 Advanced AI Copilot
> [Implementation Plan](docs/ai-implementation-plan.md) | [RAG Architecture](docs/MEMOS_OPTIMAL_RAG_SOLUTION.md)

- **Optimal RAG Pipeline**: Implements **Adaptive Retrieval** and **Smart Query Routing** to balance performance and cost.
- **Hybrid Search**: Combines keyword (BM25) and semantic (Vector) search with **Selective Reranking** for high accuracy.
- **Tech Stack**:
    - **Vector DB**: PostgreSQL + `pgvector`
    - **Models**: SiliconFlow (`bge-m3` embedding, `bge-reranker-v2-m3`) + DeepSeek V3.

### 📅 Schedule Assistant
> [Implementation Plan](docs/schedule-assistant-implementation-plan.md)

- **Natural Language Input**: Create schedules conversationally (e.g., "Meeting tomorrow at 3 PM").
- **Smart Integration**: Built directly into the AI Chat interface with proactive suggestions and conflict detection.
- **Database**: Integrated `schedule` system supporting PostgreSQL and SQLite.

### 🛡️ Core Reliability
- **Privacy First**: Fully self-hosted with no telemetry.
- **Markdown Native**: Pure text experience.
- **Performance**: High-concurrency Go backend + React frontend.

## Getting Started

### Local Development

This project uses a `Makefile` to simplify development tasks.

**Prerequisites**:
- Go 1.25+
- Node.js & pnpm
- Docker (for database dependencies)

**Commands**:

1.  **Install Dependencies**:
    ```bash
    make deps-all
    ```

2.  **Start Development Environment**:
    ```bash
    make start
    ```
    This automatically starts the PostgreSQL container, backend server, and frontend dev server.
    - Frontend: http://localhost:25173
    - Backend: http://localhost:28081

3.  **Build**:
    - Backend: `make build`
    - Frontend: `make build-web`

### Docker

```bash
docker run -d \
  --name memos \
  -p 5230:5230 \
  -v ~/.memos:/var/opt/memos \
  neosmemo/memos:stable
```

## Tech Stack

- **Backend**: Go, Echo, gRPC-Gateway
- **Frontend**: React, Vite, TailwindCSS
- **Database**: SQLite (Default), PostgreSQL, MySQL

## License

[MIT](LICENSE)
