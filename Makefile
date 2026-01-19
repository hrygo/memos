# Memos Makefile

# 加载 .env 文件 (如果存在)
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: help run dev web test deps clean
.PHONY: docker-up docker-down docker-logs docker-reset
.PHONY: docker-prod-up docker-prod-down docker-prod-logs
.PHONY: docker-full-up docker-full-down docker-full-logs docker-full-rebuild
.PHONY: docker-full-ps docker-full-exec-backend docker-full-exec-frontend
.PHONY: db-connect db-reset db-vector
.PHONY: start stop restart status logs

.DEFAULT_GOAL := help

# 数据库配置 (PostgreSQL)
MEMOS_DRIVER ?= postgres
MEMOS_DSN ?= postgres://memos:memos@localhost:5432/memos?sslmode=disable

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

run: ## 启动后端 (PostgreSQL)
	@echo "Starting Memos backend..."
	@MEMOS_DRIVER=$(MEMOS_DRIVER) MEMOS_DSN=$(MEMOS_DSN) go run ./cmd/memos --mode dev --port 8081

dev: run ## Alias for run

run-ai: ## 启动后端 + AI 支持
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
		go run ./cmd/memos --mode dev --port 8081

web: ## 启动前端开发服务器
	@cd web && pnpm dev

start: ## 一键启动所有服务 (PostgreSQL -> 后端 -> 前端)
	@./scripts/dev.sh start

stop: ## 一键停止所有服务
	@./scripts/dev.sh stop

restart: ## 重启所有服务
	@./scripts/dev.sh restart

status: ## 查看所有服务状态
	@./scripts/dev.sh status

logs: ## 查看所有服务日志 (使用 make logs backend 查看特定服务)
	@./scripts/dev.sh logs $(filter-out $@,$(MAKECMDGOALS))

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
# 全 Docker 开发环境 (PostgreSQL + 后端 + 前端)
# ===================================================================

##@ 全 Docker 开发

docker-full-up: ## 启动全 Docker 开发环境 (所有服务)
	@echo "Starting full Docker development environment..."
	@docker compose -f docker/compose/full-dev.yml up -d --build
	@echo ""
	@echo "等待服务启动..."
	@sleep 5
	@echo ""
	@echo "服务已启动!"
	@echo "  - 前端: http://localhost:5173"
	@echo "  - 后端: http://localhost:8081"
	@echo "  - 数据库: localhost:5432"
	@echo ""
	@echo "查看日志: make docker-full-logs"
	@echo "查看状态: make docker-full-ps"

docker-full-down: ## 停止全 Docker 开发环境
	@echo "Stopping full Docker development environment..."
	@docker compose -f docker/compose/full-dev.yml down
	@echo "服务已停止"

docker-full-logs: ## 查看全 Docker 日志 (指定服务: make docker-full-logs backend)
	@docker compose -f docker/compose/full-dev.yml logs -f $(filter-out $@,$(MAKECMDGOALS))

docker-full-rebuild: ## 重新构建并启动全 Docker 环境
	@echo "Rebuilding full Docker development environment..."
	@docker compose -f docker/compose/full-dev.yml up -d --build --force-recreate

docker-full-ps: ## 查看全 Docker 容器状态
	@docker compose -f docker/compose/full-dev.yml ps

docker-full-exec-backend: ## 进入后端容器 shell
	@docker exec -it memos-backend-dev sh

docker-full-exec-frontend: ## 进入前端容器 shell
	@docker exec -it memos-frontend-dev sh

docker-full-exec-postgres: ## 连接 PostgreSQL shell
	@docker exec -it memos-postgres-dev psql -U memos -d memos

# ===================================================================
# 数据库
# ===================================================================

##@ 数据库

db-connect: ## 连接 PostgreSQL shell
	@docker exec -it memos-postgres-dev psql -U memos -d memos

db-reset: ## 重置数据库 schema
	@echo "Resetting database schema..."
	@docker exec memos-postgres-dev psql -U memos -d memos -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@go run ./cmd/memos --mode dev --driver postgres --dsn "postgres://memos:memos@localhost:5432/memos?sslmode=disable" --migrate

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
# 帮助
# ===================================================================

help: ## 显示此帮助信息
	@printf "\033[1m\033[36m\nMemos Development Commands\033[0m\n\n"
	@printf "\033[1m一键操作:\033[0m\n"
	@printf "  start                一键启动所有服务 (PostgreSQL -> 后端 -> 前端)\n"
	@printf "  stop                 一键停止所有服务\n"
	@printf "  restart              重启所有服务\n"
	@printf "  status               查看所有服务状态\n"
	@printf "  logs                 查看所有服务日志\n"
	@printf "  logs backend         查看后端日志 (支持: postgres/backend/frontend)\n"
	@printf "\n\033[1m开发:\033[0m\n"
	@printf "  run                  启动后端 (PostgreSQL)\n"
	@printf "  dev                  Alias for run\n"
	@printf "  run-ai               启动后端 + AI 支持\n"
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
	@printf "  docker-prod-up       启动生产环境\n"
	@printf "  docker-prod-down     停止生产环境\n"
	@printf "\n\033[1m全 Docker 开发 (推荐):\033[0m\n"
	@printf "  docker-full-up       启动全 Docker 开发环境 (PG + 后端 + 前端)\n"
	@printf "  docker-full-down     停止全 Docker 开发环境\n"
	@printf "  docker-full-logs     查看所有日志 (可指定服务名)\n"
	@printf "  docker-full-rebuild  重新构建并启动\n"
	@printf "  docker-full-ps       查看容器状态\n"
	@printf "  docker-full-exec-backend   进入后端容器\n"
	@printf "  docker-full-exec-frontend  进入前端容器\n"
	@printf "  docker-full-exec-postgres  连接 PostgreSQL\n"
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
	@printf "\n\033[1mQuick Start:\033[0m\n"
	@printf "\n\033[1m方式一: 全 Docker 方案 (推荐，无需本地安装 Go/Node)\033[0m\n"
	@printf "  1. make docker-full-up         # 一键启动所有服务\n"
	@printf "  2. make docker-full-logs       # 查看日志\n"
	@printf "  3. make docker-full-down       # 停止服务\n"
	@printf "\n\033[1m方式二: 本地开发 (需要安装 Go 1.25+ 和 pnpm)\033[0m\n"
	@printf "  1. cp .env.example .env        # 复制环境变量模板\n"
	@printf "  2. 编辑 .env 填入 API Keys\n"
	@printf "  3. make start                   # 一键启动所有服务\n"
	@printf "  4. make status                  # 查看服务状态\n"
	@printf ""
