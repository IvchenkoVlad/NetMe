.PHONY: help install migrate server run build clean test dev setup lint

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Display this help message
	@echo "$(BLUE)NetMe Backend - Available Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-15s$(NC) %s\n", $$1, $$2}'
	@echo ""

install: ## Install Go dependencies
	@echo "$(BLUE)Installing dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

setup: install ## Complete setup (install + migrate)
	@echo "$(BLUE)Running complete setup...$(NC)"
	@$(MAKE) migrate
	@echo "$(GREEN)✓ Setup complete$(NC)"

migrate: ## Run database migrations
	@echo "$(BLUE)Running migrations...$(NC)"
	go run cmd/migrate/main.go up
	@echo "$(GREEN)✓ Migrations complete$(NC)"

migrate-down: ## Rollback database migrations
	@echo "$(BLUE)Rolling back migrations...$(NC)"
	@echo "$(RED)Note: Implement rollback logic in cmd/migrate/main.go$(NC)"

server: migrate ## Run migrations then start server (main entry point)
	@echo "$(BLUE)Starting backend server...$(NC)"
	go run cmd/server/main.go

run: server ## Alias for server

dev: ## Start backend with hot reload (requires air)
	@echo "$(BLUE)Starting backend with hot reload...$(NC)"
	@command -v air >/dev/null 2>&1 || { echo "$(RED)Installing air...$(NC)"; go install github.com/cosmtrek/air@latest; }
	air

build: ## Build backend binary
	@echo "$(BLUE)Building backend...$(NC)"
	go build -o bin/netme-server cmd/server/main.go
	@echo "$(GREEN)✓ Binary created at bin/netme-server$(NC)"

build-linux: ## Build Linux binary (for Docker/deployment)
	@echo "$(BLUE)Building Linux binary...$(NC)"
	GOOS=linux GOARCH=amd64 go build -o bin/netme-server-linux cmd/server/main.go
	@echo "$(GREEN)✓ Linux binary created at bin/netme-server-linux$(NC)"

clean: ## Clean build artifacts and cache
	@echo "$(BLUE)Cleaning...$(NC)"
	rm -rf bin/
	go clean -cache -testcache
	@echo "$(GREEN)✓ Clean complete$(NC)"

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	go test -v ./...

coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	@echo "$(GREEN)✓ Coverage report generated$(NC)"

lint: ## Run linter (requires golangci-lint)
	@echo "$(BLUE)Running linter...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "$(RED)Installing golangci-lint...$(NC)"; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run ./...

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running vet...$(NC)"
	go vet ./...

db-shell: ## Open PostgreSQL shell
	@echo "$(BLUE)Opening PostgreSQL shell...$(NC)"
	psql -U netme -h localhost -d netme_dev

db-reset: ## Reset database (WARNING: Deletes all data)
	@echo "$(RED)WARNING: This will delete all data in netme_dev database$(NC)"
	@read -p "Are you sure? (y/N) " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		psql -U postgres -c "DROP DATABASE IF EXISTS netme_dev;"; \
		psql -U postgres -c "CREATE DATABASE netme_dev OWNER netme;"; \
		$(MAKE) migrate; \
		echo "$(GREEN)✓ Database reset and migrated$(NC)"; \
	else \
		echo "Cancelled"; \
	fi

check-health: ## Check if server is running
	@echo "$(BLUE)Checking server health...$(NC)"
	@curl -s http://localhost:8080/healthz | jq . || echo "$(RED)Server not responding$(NC)"

logs: ## Show recent logs (requires jq)
	@echo "$(BLUE)Recent logs...$(NC)"
	@docker logs -f netme-backend 2>/dev/null || echo "$(RED)Docker container not found$(NC)"

docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t netme-backend:latest .
	@echo "$(GREEN)✓ Docker image built$(NC)"

docker-run: ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -p 8080:8080 --env-file .env netme-backend:latest
	@echo "$(GREEN)✓ Container running$(NC)"

docker-stop: ## Stop Docker container
	@echo "$(BLUE)Stopping Docker container...$(NC)"
	docker stop $$(docker ps -q --filter "ancestor=netme-backend:latest")
	@echo "$(GREEN)✓ Container stopped$(NC)"

all: clean install migrate build ## Full rebuild

.DEFAULT_GOAL := help
