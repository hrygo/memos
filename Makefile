.PHONY: help build run test clean docker-up docker-down docker-logs db-migrate frontend

# ==============================================
#  COLORS & STYLES
# ==============================================
RESET  := \033[0m
BOLD   := \033[1m
DIM    := \033[2m

# Colors
BLACK   := \033[30m
RED     := \033[31m
GREEN   := \033[32m
YELLOW  := \033[33m
BLUE    := \033[34m
MAGENTA := \033[35m
CYAN    := \033[36m
WHITE   := \033[37m

# Bright colors
BRIGHT_RED     := \033[91m
BRIGHT_GREEN   := \033[92m
BRIGHT_YELLOW  := \033[93m
BRIGHT_BLUE    := \033[94m
BRIGHT_MAGENTA := \033[95m
BRIGHT_CYAN    := \033[96m

# Background colors
BG_BLACK  := \033[40m
BG_RED    := \033[41m
BG_GREEN  := \033[42m
BG_YELLOW := \033[43m
BG_BLUE   := \033[44m

# ==============================================
#  CONFIGURATION
# ==============================================
GO_FILES := $(shell find . -name '*.go' -type f -not -path "./build/*" -not -path "./vendor/*")
PROTO_FILES := $(shell find proto -name '*.proto' -type f)
DOCKER_COMPOSE := docker compose

# ==============================================
#  HELP
# ==============================================
help: ## Show this help message
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)╔═══════════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(BOLD)$(BRIGHT_CYAN)║$(RESET)$(BOLD) $(BRIGHT_YELLOW)  Memos Development Environment$(RESET)                    $(BOLD)$(BRIGHT_CYAN)║$(RESET)"
	@echo "$(BOLD)$(BRIGHT_CYAN)╚═══════════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "$(BOLD)$(BLUE)Usage:$(RESET)"
	@echo "  $(CYAN)make$(RESET) $(GREEN)<target>$(RESET)"
	@echo ""
	@echo "$(BOLD)$(BLUE)Available Commands:$(RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { \
		if (length($$1) > 0) { \
			cmd = $$1; \
			desc = $$2; \
			printf "  $(BRIGHT_GREEN)%-20s$(RESET) %s\n", cmd, desc; \
		} \
	}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BOLD)$(BLUE)Examples:$(RESET)"
	@echo "  $(CYAN)make$(RESET) $(GREEN)up$(RESET)       # Start all services"
	@echo "  $(CYAN)make$(RESET) $(GREEN)dev$(RESET)       # Start development server"
	@echo "  $(CYAN)make$(RESET) $(GREEN)test$(RESET)      # Run tests"
	@echo "  $(CYAN)make$(RESET) $(GREEN)build$(RESET)     # Build the project"
	@echo ""

# ==============================================
#  DEVELOPMENT
# ==============================================
run: ## Run the server (requires PostgreSQL)
	@echo "$(BOLD)$(CYAN)▶ Starting Memos server...$(RESET)"
	@echo "$(DIM)Using local PostgreSQL connection$(RESET)"
	@go run ./cmd/memos --mode dev --port 8081

dev: ## Start development server with hot reload
	@echo "$(BOLD)$(CYAN)▶ Starting development server...$(RESET)"
	@echo "$(DIM)Press Ctrl+C to stop$(RESET)"
	@air || echo "$(YELLOW)air not installed. Run: go install github.com/cosmtrek/air@latest$(RESET)"

dev-docker: ## Start development environment with Docker
	@echo "$(BOLD)$(CYAN)▶ Starting Docker development environment...$(RESET)"
	@$(DOCKER_COMPOSE) up --build
	@$(DOCKER_COMPOSE) down

# ==============================================
#  BUILD
# ==============================================
build: ## Build the backend binary
	@echo "$(BOLD)$(CYAN)🔨 Building backend...$(RESET)"
	@go build -o bin/memos ./cmd/memos
	@echo "$(BOLD)$(GREEN)✓ Build complete: bin/memos$(RESET)"

build-frontend: ## Build the frontend
	@echo "$(BOLD)$(CYAN)🔨 Building frontend...$(RESET)"
	@cd web && pnpm build
	@echo "$(BOLD)$(GREEN)✓ Frontend build complete$(RESET)"

build-all: build build-frontend ## Build both backend and frontend
	@echo "$(BOLD)$(GREEN)✓ All builds complete$(RESET)"

release: ## Build release version with embedded frontend
	@echo "$(BOLD)$(CYAN)🔨 Building release...$(RESET)"
	@cd web && pnpm release
	@echo "$(BOLD)$(GREEN)✓ Release build complete$(RESET)"

# ==============================================
#  DOCKER
# ==============================================
up: ## Start all services (PostgreSQL + Memos)
	@echo "$(BOLD)$(CYAN)🐳 Starting services...$(RESET)"
	@$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)✓ Services started$(RESET)"
	@echo ""
	@echo "$(BOLD)$(BLUE)Services:$(RESET)"
	@echo "  $(CYAN)Memos:$(RESET)  http://localhost:8081"
	@echo "  $(CYAN)PostgreSQL:$(RESET) localhost:5433"
	@echo ""
	@echo "Run '$(CYAN)make logs$(RESET)' to view logs"

down: ## Stop all services
	@echo "$(BOLD)$(CYAN)🐳 Stopping services...$(RESET)"
	@$(DOCKER_COMPOSE) down
	@echo "$(GREEN)✓ Services stopped$(RESET)"

restart: down up ## Restart all services

logs: ## View service logs
	@$(DOCKER_COMPOSE) logs -f

logs-backend: ## View backend logs only
	@$(DOCKER_COMPOSE) logs -f memos

logs-db: ## View database logs only
	@$(DOCKER_COMPOSE) logs -f db

ps: ## Show running containers
	@$(DOCKER_COMPOSE) ps

shell: ## Open shell in the backend container
	@$(DOCKER_COMPOSE) exec memos sh

db-shell: ## Open PostgreSQL shell
	@$(DOCKER_COMPOSE) exec db psql -U memos -d memos

# ==============================================
#  DATABASE
# ==============================================
db-migrate: ## Run database migrations
	@echo "$(BOLD)$(CYAN)▶ Running migrations...$(RESET)"
	@go run ./cmd/memos migrate

db-reset: ## Reset database (WARNING: deletes all data)
	@echo "$(BOLD)$(RED)⚠ This will delete all data! Type 'yes' to confirm: $(RESET)"
	@read -r confirmation; \
	if [ "$$confirmation" = "yes" ]; then \
		$(DOCKER_COMPOSE) down -v; \
		$(DOCKER_COMPOSE) up -d; \
		echo "$(GREEN)✓ Database reset$(RESET)"; \
	else \
		echo "$(YELLOW)Cancelled$(RESET)"; \
	fi

db-status: ## Show database status
	@echo "$(BOLD)$(CYAN)📊 Database Status:$(RESET)"
	@echo ""
	@$(DOCKER_COMPOSE) exec db psql -U memos -d memos -c "\dt" 2>/dev/null || echo "$(YELLOW)Database not running$(RESET)"

# ==============================================
#  FRONTEND
# ==============================================
frontend-dev: ## Start frontend development server
	@echo "$(BOLD)$(CYAN)▶ Starting frontend dev server...$(RESET)"
	@cd web && pnpm dev

frontend-lint: ## Lint frontend code
	@echo "$(BOLD)$(CYAN)▶ Linting frontend...$(RESET)"
	@cd web && pnpm lint

frontend-lint-fix: ## Fix frontend linting issues
	@cd web && pnpm lint:fix

frontend-format: ## Format frontend code
	@cd web && pnpm format

# ==============================================
#  TESTING
# ==============================================
test: ## Run all tests
	@echo "$(BOLD)$(CYAN)▶ Running tests...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo ""
	@echo "$(BOLD)$(CYAN)Coverage Report:$(RESET)"
	@go tool cover -func=coverage.out | tail -1

test-short: ## Run quick tests (skip integration)
	@echo "$(BOLD)$(CYAN)▶ Running quick tests...$(RESET)"
	@go test -short -v ./...

test-backend: ## Run backend tests only
	@echo "$(BOLD)$(CYAN)▶ Running backend tests...$(RESET)"
	@go test -v ./store/... ./server/... ./cmd/...

bench: ## Run benchmarks
	@echo "$(BOLD)$(CYAN)▶ Running benchmarks...$(RESET)"
	@go test -bench=. -benchmem ./...

# ==============================================
#  CODE QUALITY
# ==============================================
lint: ## Run Go linter
	@echo "$(BOLD)$(CYAN)▶ Running linters...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Installing...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

fmt: ## Format Go code
	@echo "$(BOLD)$(CYAN)▶ Formatting code...$(RESET)"
	@goimports -w .
	@echo "$(GREEN)✓ Code formatted$(RESET)

vet: ## Run go vet
	@echo "$(BOLD)$(CYAN)▶ Running go vet...$(RESET)"
	@go vet ./...

tidy: ## Tidy go modules
	@echo "$(BOLD)$(CYAN)▶ Tidying modules...$(RESET)"
	@go mod tidy
	@echo "$(GREEN)✓ Modules tidied$(RESET)"

proto: ## Regenerate protocol buffers
	@echo "$(BOLD)$(CYAN)▶ Generating proto files...$(RESET)"
	@cd proto && buf generate
	@echo "$(GREEN)✓ Proto files generated$(RESET)"

# ==============================================
#  CLEAN
# ==============================================
clean: ## Clean build artifacts
	@echo "$(BOLD)$(CYAN)▶ Cleaning...$(RESET)"
	@rm -rf bin/
	@rm -f coverage.out
	@cd web && rm -rf dist/ node_modules/.vite
	@echo "$(GREEN)✓ Cleaned$(RESET)"

clean-all: clean ## Clean everything including Docker volumes
	@echo "$(BOLD)$(CYAN)▶ Deep cleaning...$(RESET)"
	@$(DOCKER_COMPOSE) down -v
	@rm -rf bin/
	@rm -f coverage.out
	@cd web && rm -rf dist/ node_modules
	@echo "$(GREEN)✓ Deep cleaned$(RESET)"

# ==============================================
#  INSTALLATION
# ==============================================
install-tools: ## Install development tools
	@echo "$(BOLD)$(CYAN)▶ Installing development tools...$(RESET)"
	@echo "Installing air..."
	@go install github.com/cosmtrek/air@latest
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing buf..."
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "$(GREEN)✓ Tools installed$(RESET)"

install-frontend: ## Install frontend dependencies
	@echo "$(BOLD)$(CYAN)▶ Installing frontend dependencies...$(RESET)"
	@cd web && pnpm install
	@echo "$(GREEN)✓ Frontend dependencies installed$(RESET)"

# ==============================================
#  AI FEATURES
# ==============================================
ai-embed: ## Generate embeddings for existing memos
	@echo "$(BOLD)$(CYAN)▶ Generating embeddings...$(RESET)"
	@echo "$(YELLOW)Set MEMOS_AI_API_KEY environment variable first$(RESET)"
	@if [ -z "$$MEMOS_AI_API_KEY" ]; then \
		echo "$(RED)Error: MEMOS_AI_API_KEY not set$(RESET)"; \
		exit 1; \
	fi
	@echo "$(DIM)This may take a while...$(RESET)"

# ==============================================
#  INFO
# ==============================================
info: ## Show project information
	@echo ""
	@echo "$(BOLD)$(BRIGHT_CYAN)╔═══════════════════════════════════════════════════════════════╗$(RESET)"
	@echo "$(BOLD)$(BRIGHT_CYAN)║$(RESET)$(BOLD) $(BRIGHT_YELLOW)  Memos Project Information$(RESET)                       $(BOLD)$(BRIGHT_CYAN)║$(RESET)"
	@echo "$(BOLD)$(BRIGHT_CYAN)╚═══════════════════════════════════════════════════════════════╝$(RESET)"
	@echo ""
	@echo "$(BOLD)$(BLUE)Project:$(RESET) Memos - An open source, lightweight note-taking service"
	@echo "$(BOLD)$(BLUE)Version:$(RESET) $(shell git describe --tags --always 2>/dev/null || echo 'unknown')"
	@echo "$(BOLD)$(BLUE)Go:$(RESET) $(shell go version | awk '{print $$3}')"
	@echo ""
	@echo "$(BOLD)$(BLUE)Directories:$(RESET)"
	@echo "  $(CYAN)cmd/$(RESET)      - Application entry point"
	@echo "  $(CYAN)server/$(RESET)   - Server implementation"
	@echo "  $(CYAN)store/$(RESET)    - Data layer"
	@echo "  $(CYAN)web/$(RESET)      - Frontend (React + TypeScript)"
	@echo "  $(CYAN)proto/$(RESET)    - Protocol buffers"
	@echo ""
	@echo "$(BOLD)$(BLUE)Quick Start:$(RESET)"
	@echo "  1. $(CYAN)make up$(RESET)           - Start services"
	@echo "  2. $(CYAN)make frontend-dev$(RESET)  - Start frontend"
	@echo "  3. Open http://localhost:8081"
	@echo ""

# Default target
.DEFAULT_GOAL := help
