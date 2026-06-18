.PHONY: help install migrate migrate-down migrate-reset server run build build-linux clean test coverage lint fmt vet dev setup db-shell db-reset check-health logs docker-build docker-run docker-stop all

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-20s %s\n", $$1, $$2}'

install: ## Install Go dependencies
	go mod download
	go mod tidy

setup: install migrate ## Install + migrate

migrate: ## Run database migrations
	go run cmd/migrate/main.go up

migrate-down: ## Rollback last migration
	go run cmd/migrate/main.go down

migrate-reset: ## Rollback ALL migrations
	@read -p "Drop all tables? (y/N) " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		go run cmd/migrate/main.go down-to 0; \
	fi

server: ## Start server
	go run cmd/server/main.go

run: server ## Alias for server

dev: ## Start with hot reload (requires air)
	@command -v air >/dev/null 2>&1 || go install github.com/cosmtrek/air@latest
	air

build: ## Build binary
	go build -o bin/netme-server cmd/server/main.go

build-linux: ## Build Linux binary
	GOOS=linux GOARCH=amd64 go build -o bin/netme-server-linux cmd/server/main.go

clean: ## Clean build artifacts
	rm -rf bin/
	go clean -cache -testcache

test: ## Run tests
	go test -v ./...

coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint: ## Run linter
	@command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

db-shell: ## Open PostgreSQL shell
	psql -U netme -h localhost -d netme_dev

db-reset: ## Reset database
	@read -p "Delete all data? (y/N) " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		psql -U postgres -c "DROP DATABASE IF EXISTS netme_dev;"; \
		psql -U postgres -c "CREATE DATABASE netme_dev OWNER netme;"; \
		$(MAKE) migrate; \
	fi

check-health: ## Check server health
	@curl -s http://localhost:8080/healthz | jq .

logs: ## Show container logs
	@docker logs -f netme-backend 2>/dev/null

docker-build: ## Build Docker image
	docker build -t netme-backend:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env netme-backend:latest

docker-stop: ## Stop Docker container
	docker stop $$(docker ps -q --filter "ancestor=netme-backend:latest")

all: clean install migrate build ## Full rebuild

.DEFAULT_GOAL := help
