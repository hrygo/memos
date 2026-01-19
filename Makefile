# Memos Makefile
# For 2C2G production deployment optimization

.PHONY: help build run test lint clean
.PHONY: web-build web-dev web-lint web-install
.PHONY: proto-generate proto-lint proto-breaking
.PHONY: docker-up docker-down docker-logs docker-ps
.PHONY: docker-prod-up docker-prod-down
.PHONY: db-migrate db-verify
.PHONY: ai-test ai-deps

# 默认目标
.DEFAULT_GOAL := help

# ===================================================================
# 开发环境
# ===================================================================

##@ 开发

run: ## 启动后端开发服务器 (Go)
	@echo "Starting Memos backend server..."
	@go run ./cmd/memos --mode dev --port 8081

dev: run ## Alias for run

web-dev: ## 启动前端开发服务器
	@echo "Starting frontend dev server..."
	@cd web && pnpm dev

dev-all: ## 同时启动前后端 (需要 tmux 或类似工具)
	@echo "Use 'tmux' or separate terminals for backend and frontend"

##@ 构建

build: ## 构建后端
	@echo "Building backend..."
	@go build -o bin/memos ./cmd/memos

web-build: ## 构建前端
	@echo "Building frontend..."
	@cd web && pnpm build

build-all: build web-build ## 构建前后端

release: web-build ## 前端 release (构建并复制)
	@echo "Building frontend release..."
	@cd web && pnpm release

##@ 测试

test: ## 运行所有后端测试
	@echo "Running backend tests..."
	@go test ./... -v -timeout 30s

test-short: ## 快速测试 (跳过集成测试)
	@echo "Running quick tests..."
	@go test ./... -short -v

test-coverage: ## 测试覆盖率
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-postgres: ## PostgreSQL 测试 (需要 Docker)
	@echo "Running PostgreSQL tests..."
	@DRIVER=postgres \
		DSN="postgres://memos:memos@localhost:5432/memos?sslmode=disable" \
		go test ./... -run TestMemo -v

test-ai: ## AI 功能测试
	@echo "Running AI feature tests..."
	@DRIVER=postgres \
		DSN="postgres://memos:memos@localhost:5432/memos?sslmode=disable" \
		go test ./plugin/ai/... -v

##@ Lint

lint: ## 运行 Go linter
	@echo "Running golangci-lint..."
	@golangci-lint run

web-lint: ## 前端 lint 检查
	@echo "Running frontend lint..."
	@cd web && pnpm lint

web-lint-fix: ## 前端 lint 自动修复
	@echo "Fixing frontend lint issues..."
	@cd web && pnpm lint:fix

format: ## 格式化 Go 代码
	@echo "Formatting Go code..."
	@goimports -w .

web-format: ## 格式化前端代码
	@echo "Formatting frontend code..."
	@cd web && pnpm format

##@ 依赖

deps: ## 安装 Go 依赖
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

web-install: ## 安装前端依赖
	@echo "Installing frontend dependencies..."
	@cd web && pnpm install

ai-deps: ## 安装 AI 相关 Go 依赖
	@echo "Installing AI dependencies..."
	@go get github.com/tmc/langchaingo
	@go mod tidy

deps-all: deps web-install ## 安装所有依赖

# ===================================================================
# Proto
# ===================================================================

##@ Protocol Buffers

proto-generate: ## 生成 Proto 代码
	@echo "Generating Proto code..."
	@cd proto && buf generate

proto-lint: ## Lint Proto 文件
	@echo "Linting Proto files..."
	@cd proto && buf lint

proto-breaking: ## 检查 Proto 破坏性变更
	@echo "Checking breaking changes..."
	@cd proto && buf breaking --against .git#main

proto-update: proto-generate ## 更新 Proto (alias)

# ===================================================================
# Docker
# ===================================================================

##@ Docker (开发环境)

docker-up: ## 启动开发环境 Docker 服务
	@echo "Starting dev Docker services..."
	@docker compose -f docker-compose.dev.yml up -d

docker-down: ## 停止开发环境 Docker 服务
	@echo "Stopping dev Docker services..."
	@docker compose -f docker-compose.dev.yml down --remove-orphans

docker-restart: docker-down docker-up ## 重启 Docker 服务

docker-logs: ## 查看 Docker 日志
	@docker compose -f docker-compose.dev.yml logs -f postgres

docker-ps: ## 查看 Docker 容器状态
	@docker compose -f docker-compose.dev.yml ps

##@ Docker (生产环境)

docker-prod-up: ## 启动生产环境 Docker 服务
	@echo "Starting production Docker services..."
	@docker compose -f docker-compose.prod.yml up -d

docker-prod-down: ## 停止生产环境 Docker 服务
	@echo "Stopping production Docker services..."
	@docker compose -f docker-compose.prod.yml down

docker-prod-logs: ## 查看生产环境日志
	@docker compose -f docker-compose.prod.yml logs -f

# ===================================================================
# 数据库
# ===================================================================

##@ 数据库

db-connect: ## 连接 PostgreSQL (psql)
	@docker exec -it memos-postgres-dev psql -U memos -d memos

db-shell: ## 连接到 PostgreSQL shell (alias)
	@make db-connect

db-migrate: ## 运行数据库迁移
	@echo "Running database migrations..."
	@go run ./cmd/memos --migrate

db-verify: ## 验证数据库 schema
	@echo "Verifying database schema..."
	@docker exec memos-postgres-dev psql -U memos -d memos -c "\dt+"

db-reset: ## 重置数据库 (危险!)
	@echo "Resetting database..."
	@docker exec memos-postgres-dev psql -U memos -d memos -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@make db-migrate

db-test-vector: ## 测试 pgvector 功能
	@echo "Testing pgvector..."
	@docker exec memos-postgres-dev psql -U memos -d memos -c "SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"

# ===================================================================
# AI 开发
# ===================================================================

##@ AI 开发

ai-install: ai-deps ## 安装 AI 依赖 (alias)

ai-test: test-ai ## 测试 AI 功能 (alias)

ai-embed: ## 测试 Embedding 服务 (需要 API Key)
	@echo "Testing Embedding service..."
	@MEMOS_AI_ENABLED=true \
		MEMOS_AI_SILICONFLOW_API_KEY=${SILICONFLOW_API_KEY} \
		go run ./cmd/memos --embed-test

ai-chat: ## 测试 LLM 聊天 (需要 API Key)
	@echo "Testing LLM chat..."
	@MEMOS_AI_ENABLED=true \
		MEMOS_AI_DEEPSEEK_API_KEY=${DEEPSEEK_API_KEY} \
		go run ./cmd/memos --chat-test

# ===================================================================
# 清理
# ===================================================================

##@ 清理

clean: ## 清理构建文件
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@cd web && rm -rf dist/ node_modules/.vite

clean-all: clean ## 清理所有 (包括依赖)
	@echo "Cleaning all..."
	@cd web && rm -rf node_modules/
	@go clean -modcache

clean-docker: ## 清理 Docker 资源
	@echo "Cleaning Docker resources..."
	@docker compose -f docker-compose.dev.yml down -v
	@docker volume rm memos_postgres_data 2>/dev/null || true

# ===================================================================
# 帮助
# ===================================================================

##@ 帮助

help: ## 显示帮助信息
	@echo "Memos Development Commands"
	@echo ""
	@grep -E '^##@' $(MAKEFILE_LIST) | awk 'BEGIN {FS = "##@ "}; {printf "\033[36m%s\033[0m\n", $$2}' | sort
	@echo ""
	@echo "Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' | sort
	@echo ""
	@echo "Quick Start:"
	@echo "  make docker-up      # 启动 PostgreSQL (开发环境)"
	@echo "  make deps           # 安装依赖"
	@echo "  make run            # 启动后端"
	@echo "  make web-dev        # 启动前端 (另一个终端)"
	@echo ""
	@echo "Production (2C2G):"
	@echo "  make docker-prod-up # 启动生产环境"
	@echo ""
	@echo "AI Development:"
	@echo "  make ai-deps        # 安装 AI 依赖"
	@echo "  make proto-generate # 生成 Proto 代码"
	@echo "  make test-ai        # 测试 AI 功能"
