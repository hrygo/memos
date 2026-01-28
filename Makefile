# DivineSense Makefile
# SPDX-License-Identifier: MIT

# Load .env file if exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# ===========================================================================
# Configuration
# ===========================================================================

.DEFAULT_GOAL := help

# Database Configuration (PostgreSQL)
DIVINESENSE_DRIVER ?= postgres
DIVINESENSE_DSN ?= postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable
POSTGRES_CONTAINER ?= divinesense-postgres-dev
POSTGRES_PORT ?= 25432
POSTGRES_USER ?= divinesense
POSTGRES_DB ?= divinesense

# AI Configuration
AI_EMBEDDING_PROVIDER ?= siliconflow
AI_LLM_PROVIDER ?= deepseek
AI_EMBEDDING_MODEL ?= BAAI/bge-m3
AI_RERANK_MODEL ?= BAAI/bge-reranker-v2-m3
AI_LLM_MODEL ?= deepseek-chat
AI_OPENAI_BASE_URL ?= https://api.siliconflow.cn/v1

# Paths
DOCKER_COMPOSE_DEV ?= docker/compose/dev.yml
DOCKER_COMPOSE_PROD ?= docker/compose/prod.yml
DEPLOY_DIR ?= deploy/aliyun
DEPLOY_SCRIPT ?= $(DEPLOY_DIR)/deploy.sh
SCRIPT_DIR ?= scripts

# Backend
BACKEND_BIN ?= bin/divinesense
BACKEND_CMD ?= cmd/divinesense
BACKEND_PORT ?= 28081

# Frontend
WEB_DIR ?= web

# ===========================================================================
# Phony Targets
# ===========================================================================

.PHONY: help run dev web test deps clean
.PHONY: docker-up docker-down docker-logs docker-reset
.PHONY: docker-prod-up docker-prod-down docker-prod-logs
.PHONY: db-connect db-reset db-vector
.PHONY: start stop restart status logs
.PHONY: logs-backend logs-frontend logs-postgres
.PHONY: logs-follow-backend logs-follow-frontend logs-follow-postgres
.PHONY: git-status git-diff git-log git-push
.PHONY: check-branch check-build check-test check-i18n check-i18n-hardcode check-all
.PHONY: prod-build prod-deploy prod-logs prod-status prod-backup prod-stop prod-restart
.PHONY: deps deps-web deps-ai deps-all
.PHONY: build build-web build-all
.PHONY: clean clean-all
.PHONY: test test-ai test-embedding test-runner

# ===========================================================================
# Development Commands
# ===========================================================================

##@ Development

run: ## 启动后端 (PostgreSQL + AI)
	@echo "Starting DivineSense with AI support..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) \
		DIVINESENSE_DSN=$(DIVINESENSE_DSN) \
		DIVINESENSE_AI_ENABLED=true \
		DIVINESENSE_AI_EMBEDDING_PROVIDER=$(AI_EMBEDDING_PROVIDER) \
		DIVINESENSE_AI_LLM_PROVIDER=$(AI_LLM_PROVIDER) \
		DIVINESENSE_AI_SILICONFLOW_API_KEY=$(SILICONFLOW_API_KEY) \
		DIVINESENSE_AI_DEEPSEEK_API_KEY=$(DEEPSEEK_API_KEY) \
		DIVINESENSE_AI_OPENAI_API_KEY=$(OPENAI_API_KEY) \
		DIVINESENSE_AI_OPENAI_BASE_URL=$(AI_OPENAI_BASE_URL) \
		DIVINESENSE_AI_EMBEDDING_MODEL=$(AI_EMBEDDING_MODEL) \
		DIVINESENSE_AI_RERANK_MODEL=$(AI_RERANK_MODEL) \
		DIVINESENSE_AI_LLM_MODEL=$(AI_LLM_MODEL) \
		go run ./$(BACKEND_CMD) --mode dev --port $(BACKEND_PORT)

dev: run ## Alias for run

web: ## 启动前端开发服务器
	@cd $(WEB_DIR) && pnpm dev

start: build ## 一键启动所有服务 (自动构建最新版本)
	@$(SCRIPT_DIR)/dev.sh start

stop: ## 一键停止所有服务
	@$(SCRIPT_DIR)/dev.sh stop

restart: build ## 重启所有服务 (自动构建最新版本)
	@$(SCRIPT_DIR)/dev.sh restart

status: ## 查看所有服务状态
	@$(SCRIPT_DIR)/dev.sh status

logs: ## 查看所有服务日志
	@$(SCRIPT_DIR)/dev.sh logs

logs-backend: ## 查看后端日志
	@$(SCRIPT_DIR)/dev.sh logs backend

logs-frontend: ## 查看前端日志
	@$(SCRIPT_DIR)/dev.sh logs frontend

logs-postgres: ## 查看 PostgreSQL 日志
	@$(SCRIPT_DIR)/dev.sh logs postgres

logs-follow-backend: ## 实时跟踪后端日志
	@$(SCRIPT_DIR)/dev.sh logs backend -f

logs-follow-frontend: ## 实时跟踪前端日志
	@$(SCRIPT_DIR)/dev.sh logs frontend -f

logs-follow-postgres: ## 实时跟踪 PostgreSQL 日志
	@$(SCRIPT_DIR)/dev.sh logs postgres -f

# ===========================================================================
# Dependencies
# ===========================================================================

##@ Dependencies

deps: ## 安装后端依赖
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

deps-web: ## 安装前端依赖
	@cd $(WEB_DIR) && pnpm install

deps-ai: ## 安装 AI 依赖
	@echo "Installing AI dependencies..."
	@go get github.com/tmc/langchaingo
	@go mod tidy

deps-all: deps deps-web ## 安装所有依赖

# ===========================================================================
# Docker (PostgreSQL)
# ===========================================================================

##@ Docker

docker-up: ## 启动 PostgreSQL
	@echo "Starting PostgreSQL..."
	@docker compose -f $(DOCKER_COMPOSE_DEV) up -d

docker-down: ## 停止 PostgreSQL
	@echo "Stopping PostgreSQL..."
	@docker compose -f $(DOCKER_COMPOSE_DEV) down --remove-orphans

docker-logs: ## 查看 PostgreSQL 日志
	@docker compose -f $(DOCKER_COMPOSE_DEV) logs -f postgres

docker-reset: ## 重置 PostgreSQL 数据 (危险!)
	@echo "Resetting PostgreSQL data..."
	@docker compose -f $(DOCKER_COMPOSE_DEV) down -v
	@docker volume rm divinesense_postgres_data 2>/dev/null || true
	@$(MAKE) docker-up

docker-prod-up: ## 启动生产环境
	@echo "Starting production environment..."
	@docker compose -f $(DOCKER_COMPOSE_PROD) up -d

docker-prod-down: ## 停止生产环境
	@echo "Stopping production environment..."
	@docker compose -f $(DOCKER_COMPOSE_PROD) down

docker-prod-logs: ## 查看生产环境日志
	@docker compose -f $(DOCKER_COMPOSE_PROD) logs -f

# ===========================================================================
# Database Commands
# ===========================================================================

##@ Database

db-connect: ## 连接 PostgreSQL shell
	@docker exec -it $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

db-reset: ## 重置数据库 schema
	@echo "Resetting database schema..."
	@docker exec $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@go run ./$(BACKEND_CMD) --mode dev --driver postgres --dsn "$(DIVINESENSE_DSN)" --migrate

db-vector: ## 验证 pgvector 扩展
	@docker exec $(POSTGRES_CONTAINER) psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"

# ===========================================================================
# Test Commands
# ===========================================================================

##@ Testing

test: ## 运行所有测试
	@echo "Running tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test ./... -v -timeout 30s

test-ai: ## 运行 AI 测试
	@echo "Running AI tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test ./plugin/ai/... -v

test-embedding: ## 运行 Embedding 测试
	@echo "Running Embedding tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test ./plugin/ai/... -run Embedding -v

test-runner: ## 运行 Runner 测试
	@echo "Running Runner tests..."
	@DIVINESENSE_DRIVER=$(DIVINESENSE_DRIVER) DIVINESENSE_DSN=$(DIVINESENSE_DSN) go test ./server/runner/embedding/... -v

# ===========================================================================
# Build Commands
# ===========================================================================

##@ Build

build: ## 构建后端
	@echo "Building backend..."
	@go build -o $(BACKEND_BIN) ./$(BACKEND_CMD)

build-web: ## 构建前端
	@echo "Building frontend..."
	@cd $(WEB_DIR) && pnpm build

build-all: build build-web ## 构建前后端

# ===========================================================================
# Clean Commands
# ===========================================================================

##@ Clean

clean: ## 清理构建文件
	@rm -rf bin/
	@cd $(WEB_DIR) && rm -rf dist/ node_modules/.vite

clean-all: clean ## 清理所有
	@cd $(WEB_DIR) && rm -rf node_modules/
	@go clean -modcache

# ===========================================================================
# Git Workflow Commands
# ===========================================================================

##@ Git Workflow

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
		echo "WARNING: You are on main branch. Consider creating a feature branch."; \
	fi

check-build: ## 检查编译
	@echo "Checking build..."
	@go build ./... || { echo "Build failed"; exit 1; }
	@echo "Build OK"

check-test: ## 检查测试
	@echo "Running tests..."
	@go test ./... -short -timeout 30s || { echo "Tests failed"; exit 1; }
	@echo "Tests OK"

check-i18n: ## 检查 i18n 翻译完整性 (强制)
	@echo "Checking i18n translations..."
	@chmod +x $(SCRIPT_DIR)/check-i18n.sh
	@$(SCRIPT_DIR)/check-i18n.sh

check-i18n-hardcode: ## 检查前端硬编码文本
	@echo "Checking hardcoded text..."
	@chmod +x $(SCRIPT_DIR)/check-i18n-hardcode.sh
	@$(SCRIPT_DIR)/check-i18n-hardcode.sh

check-all: check-build check-test check-i18n ## 运行所有检查

# ===========================================================================
# Production Deployment Commands (2C2G)
# ===========================================================================

##@ Production Deployment

prod-build: ## 构建生产镜像
	@echo "Building production image..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) build

prod-deploy: ## 部署到生产环境
	@echo "Deploying to production..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) deploy

prod-restart: ## 重启生产服务
	@echo "Restarting production services..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) restart

prod-stop: ## 停止生产服务
	@echo "Stopping production services..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) stop

prod-logs: ## 查看生产服务日志
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) logs

prod-status: ## 查看生产服务状态
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) status

prod-backup: ## 备份生产数据库
	@echo "Backing up production database..."
	@chmod +x $(DEPLOY_SCRIPT)
	@$(DEPLOY_SCRIPT) backup

# ===========================================================================
# Help
# ===========================================================================

##@ Help

help: ## 显示此帮助信息
	@printf "\033[1m\033[36m\nDivineSense Development Commands\033[0m\n\n"
	@awk 'BEGIN {FS = ":.*##"; section = ""; \
		printf "\033[1mQuick Start:\033[0m\n"; \
		printf "  1. make docker-up               # 启动 PostgreSQL\n"; \
		printf "  2. make start                   # 启动后端 + 前端\n"; \
		printf "  3. 访问 http://localhost:25173   # 打开前端\n\n";} \
		/^##@/ { section = $$0;gsub(/^##@ /, "", section); \
			if (section != "Help") printf "\n\033[1m%s:\033[0m\n", section; next } \
		/^[a-zA-Z0-9_%-]+:.*?##/ { \
			cmd = $$1; desc = $$2; \
			gsub(/^## /, "", desc); \
			printf "  \033[36m%-20s\033[0m %s\n", cmd, desc }' $(MAKEFILE_LIST)
