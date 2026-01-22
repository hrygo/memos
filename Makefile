# Memos Makefile

# 加载 .env 文件 (如果存在)
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help run dev web test deps clean
.PHONY: docker-up docker-down docker-logs docker-reset
.PHONY: db-connect db-reset db-vector
.PHONY: start stop restart status logs
.PHONY: logs-backend logs-frontend logs-postgres logs-follow-backend logs-follow-frontend logs-follow-postgres
.PHONY: git-status git-diff git-log git-push
.PHONY: check-branch check-build check-test

.DEFAULT_GOAL := help

# 数据库配置 (PostgreSQL)
MEMOS_DRIVER ?= postgres
MEMOS_DSN ?= postgres://memos:memos@localhost:25432/memos?sslmode=disable

# AI 配置
AI_EMBEDDING_PROVIDER ?= siliconflow
AI_LLM_PROVIDER ?= deepseek
AI_EMBEDDING_MODEL ?= BAAI/bge-m3
AI_RERANK_MODEL ?= BAAI/bge-reranker-v2-m3
AI_LLM_MODEL ?= deepseek-chat
AI_OPENAI_BASE_URL ?= https://api.siliconflow.cn/v1

# ===================================================================
# 开发
# ===================================================================

##@ 开发

run: ## 启动后端 (PostgreSQL + AI)
	@echo "Starting Memos with AI support..."
	@MEMOS_DRIVER=$(MEMOS_DRIVER) \
		MEMOS_DSN=$(MEMOS_DSN) \
		MEMOS_AI_ENABLED=true \
		MEMOS_AI_EMBEDDING_PROVIDER=$(AI_EMBEDDING_PROVIDER) \
		MEMOS_AI_LLM_PROVIDER=$(AI_LLM_PROVIDER) \
		MEMOS_AI_SILICONFLOW_API_KEY=$(SILICONFLOW_API_KEY) \
		MEMOS_AI_DEEPSEEK_API_KEY=$(DEEPSEEK_API_KEY) \
		MEMOS_AI_OPENAI_API_KEY=$(OPENAI_API_KEY) \
		MEMOS_AI_OPENAI_BASE_URL=$(AI_OPENAI_BASE_URL) \
		MEMOS_AI_EMBEDDING_MODEL=$(AI_EMBEDDING_MODEL) \
		MEMOS_AI_RERANK_MODEL=$(AI_RERANK_MODEL) \
		MEMOS_AI_LLM_MODEL=$(AI_LLM_MODEL) \
		go run ./cmd/memos --mode dev --port 28081

dev: run ## Alias for run

web: ## 启动前端开发服务器
	@cd web && pnpm dev

start: build ## 一键启动所有服务 (PostgreSQL -> 后端 -> 前端) - 自动构建最新版本
	@./scripts/dev.sh start

stop: ## 一键停止所有服务
	@./scripts/dev.sh stop

restart: build ## 重启所有服务 - 自动构建最新版本
	@./scripts/dev.sh restart

status: ## 查看所有服务状态
	@./scripts/dev.sh status

logs: ## 查看所有服务日志
	@./scripts/dev.sh logs

logs-backend: ## 查看后端日志
	@./scripts/dev.sh logs backend

logs-frontend: ## 查看前端日志
	@./scripts/dev.sh logs frontend

logs-postgres: ## 查看 PostgreSQL 日志
	@./scripts/dev.sh logs postgres

logs-follow-backend: ## 实时跟踪后端日志
	@./scripts/dev.sh logs backend -f

logs-follow-frontend: ## 实时跟踪前端日志
	@./scripts/dev.sh logs frontend -f

logs-follow-postgres: ## 实时跟踪 PostgreSQL 日志
	@./scripts/dev.sh logs postgres -f

##@ 依赖

deps: ## 安装后端依赖
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

deps-web: ## 安装前端依赖
	@cd web && pnpm install

deps-ai: ## 安装 AI 依赖
	@echo "Installing AI dependencies..."
	@go get github.com/tmc/langchaingo
	@go mod tidy

deps-all: deps deps-web ## 安装所有依赖

# ===================================================================
# Docker (PostgreSQL)
# ===================================================================

##@ Docker

docker-up: ## 启动 PostgreSQL
	@echo "Starting PostgreSQL..."
	@docker compose -f docker/compose/dev.yml up -d

docker-down: ## 停止 PostgreSQL
	@echo "Stopping PostgreSQL..."
	@docker compose -f docker/compose/dev.yml down --remove-orphans

docker-logs: ## 查看 PostgreSQL 日志
	@docker compose -f docker/compose/dev.yml logs -f postgres

docker-reset: ## 重置 PostgreSQL 数据 (危险!)
	@echo "Resetting PostgreSQL data..."
	@docker compose -f docker/compose/dev.yml down -v
	@docker volume rm memos_postgres_data 2>/dev/null || true
	@make docker-up

# 生产环境部署
docker-prod-up: ## 启动生产环境
	@echo "Starting production environment..."
	@docker compose -f docker/compose/prod.yml up -d

docker-prod-down: ## 停止生产环境
	@echo "Stopping production environment..."
	@docker compose -f docker/compose/prod.yml down

docker-prod-logs: ## 查看生产环境日志
	@docker compose -f docker/compose/prod.yml logs -f

# ===================================================================
# 数据库
# ===================================================================

##@ 数据库

db-connect: ## 连接 PostgreSQL shell
	@docker exec -it memos-postgres-dev psql -U memos -d memos

db-reset: ## 重置数据库 schema
	@echo "Resetting database schema..."
	@docker exec memos-postgres-dev psql -U memos -d memos -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@go run ./cmd/memos --mode dev --driver postgres --dsn "postgres://memos:memos@localhost:25432/memos?sslmode=disable" --migrate

db-vector: ## 验证 pgvector 扩展
	@docker exec memos-postgres-dev psql -U memos -d memos -c "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"

# ===================================================================
# 测试
# ===================================================================

##@ 测试

test: ## 运行所有测试
	@echo "Running tests..."
	@MEMOS_DRIVER=$(MEMOS_DRIVER) MEMOS_DSN=$(MEMOS_DSN) go test ./... -v -timeout 30s

test-ai: ## 运行 AI 测试
	@echo "Running AI tests..."
	@MEMOS_DRIVER=$(MEMOS_DRIVER) MEMOS_DSN=$(MEMOS_DSN) go test ./plugin/ai/... -v

test-embedding: ## 运行 Embedding 测试
	@echo "Running Embedding tests..."
	@MEMOS_DRIVER=$(MEMOS_DRIVER) MEMOS_DSN=$(MEMOS_DSN) go test ./plugin/ai/... -run Embedding -v

test-runner: ## 运行 Runner 测试
	@echo "Running Runner tests..."
	@MEMOS_DRIVER=$(MEMOS_DRIVER) MEMOS_DSN=$(MEMOS_DSN) go test ./server/runner/embedding/... -v

# ===================================================================
# 构建
# ===================================================================

##@ 构建

build: ## 构建后端
	@echo "Building backend..."
	@go build -o bin/memos ./cmd/memos

build-web: ## 构建前端
	@echo "Building frontend..."
	@cd web && pnpm build

build-all: build build-web ## 构建前后端

# ===================================================================
# 清理
# ===================================================================

##@ 清理

clean: ## 清理构建文件
	@rm -rf bin/
	@cd web && rm -rf dist/ node_modules/.vite

clean-all: clean ## 清理所有
	@cd web && rm -rf node_modules/
	@go clean -modcache

# ===================================================================
# Git 工作流
# ===================================================================

##@ Git 工作流

git-status: ## 查看 Git 状态
	@echo "Current Git status:"
	@git status --short

git-diff: ## 查看变更详情
	@echo "Showing changes..."
	@git diff --stat

git-log: ## 查看最近提交
	@echo "Recent commits:"
	@git log --oneline -10

git-push: ## 推送到远程 (需先检查)
	@echo "Checking branch and pushing..."
	@if [ "$$(git branch --show-current)" = "main" ]; then \
		echo "ERROR: Cannot push to main directly. Create a feature branch first."; \
		exit 1; \
	fi
	@git push origin "$$(git branch --show-current)"

check-branch: ## 检查当前分支
	@echo "Current branch: $$(git branch --show-current)"
	@if [ "$$(git branch --show-current)" = "main" ]; then \
		echo "⚠️  You are on main branch. Consider creating a feature branch."; \
	fi

check-build: ## 检查编译
	@echo "Checking build..."
	@go build ./... || { echo "❌ Build failed"; exit 1; }
	@echo "✅ Build OK"

check-test: ## 检查测试
	@echo "Running tests..."
	@go test ./... -short -timeout 30s || { echo "❌ Tests failed"; exit 1; }
	@echo "✅ Tests OK"

# ===================================================================
# 帮助
# ===================================================================

help: ## 显示此帮助信息
	@printf "\033[1m\033[36m\nMemos Development Commands\033[0m\n\n"
	@printf "\033[1m一键操作:\033[0m\n"
	@printf "  start                一键启动所有服务 (自动编译最新版本)\n"
	@printf "  stop                 一键停止所有服务\n"
	@printf "  restart              重启所有服务 (自动编译最新版本)\n"
	@printf "  status               查看所有服务状态\n"
	@printf "\n\033[1m日志查看:\033[0m\n"
	@printf "  logs                 查看所有服务日志\n"
	@printf "  logs-backend         查看后端日志\n"
	@printf "  logs-frontend        查看前端日志\n"
	@printf "  logs-postgres        查看 PostgreSQL 日志\n"
	@printf "  logs-follow-backend  实时跟踪后端日志\n"
	@printf "  logs-follow-frontend 实时跟踪前端日志\n"
	@printf "  logs-follow-postgres 实时跟踪 PostgreSQL 日志\n"
	@printf "\n\033[1m开发:\033[0m\n"
	@printf "  run                  启动后端 (PostgreSQL + AI)\n"
	@printf "  dev                  Alias for run\n"
	@printf "  web                  启动前端开发服务器\n"
	@printf "\n\033[1m依赖:\033[0m\n"
	@printf "  deps                 安装后端依赖\n"
	@printf "  deps-web             安装前端依赖\n"
	@printf "  deps-ai              安装 AI 依赖\n"
	@printf "  deps-all             安装所有依赖\n"
	@printf "\n\033[1mDocker:\033[0m\n"
	@printf "  docker-up            启动开发环境 PostgreSQL\n"
	@printf "  docker-down          停止开发环境 PostgreSQL\n"
	@printf "  docker-logs          查看 PostgreSQL 日志\n"
	@printf "  docker-reset         重置 PostgreSQL 数据 (危险!)\n"
	@printf "  docker-prod-up       启动生产环境 (PostgreSQL)\n"
	@printf "  docker-prod-down     停止生产环境\n"
	@printf "\n\033[1m数据库:\033[0m\n"
	@printf "  db-connect           连接 PostgreSQL shell\n"
	@printf "  db-reset             重置数据库 schema\n"
	@printf "  db-vector            验证 pgvector 扩展\n"
	@printf "\n\033[1m测试:\033[0m\n"
	@printf "  test                 运行所有测试\n"
	@printf "  test-ai              运行 AI 测试\n"
	@printf "  test-embedding       运行 Embedding 测试\n"
	@printf "  test-runner          运行 Runner 测试\n"
	@printf "\n\033[1m构建:\033[0m\n"
	@printf "  build                构建后端\n"
	@printf "  build-web            构建前端\n"
	@printf "  build-all            构建前后端\n"
	@printf "\n\033[1m清理:\033[0m\n"
	@printf "  clean                清理构建文件\n"
	@printf "  clean-all            清理所有\n"
	@printf "\n\033[1mGit 工作流:\033[0m\n"
	@printf "  git-status           查看当前状态\n"
	@printf "  git-diff             查看变更统计\n"
	@printf "  git-log              查看最近提交\n"
	@printf "  git-push             推送到远程 (禁止直接推 main)\n"
	@printf "  check-branch         检查当前分支\n"
	@printf "  check-build          检查编译\n"
	@printf "  check-test           检查测试\n"
	@printf "\n\033[1mQuick Start:\033[0m\n"
	@printf "  1. make docker-up               # 启动 PostgreSQL\n"
	@printf "  2. make start                   # 启动后端 + 前端\n"
	@printf "  3. 访问 http://localhost:25173   # 打开前端\n"
	@printf ""
