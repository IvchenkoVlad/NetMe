.PHONY: help setup start dev db-up db-down db-reset backend mobile lint test backend-test mobile-test mobile-build-ios-testflight mobile-build-ios-release

help:
	@echo "NetMe Workspace — Development Commands"
	@echo ""
	@echo "Quick Start:"
	@echo "  make start                 ⭐ Start EVERYTHING (db, backend, mobile)"
	@echo ""
	@echo "Setup:"
	@echo "  make setup                 Install dependencies for backend + mobile"
	@echo "  make setup-backend         Install backend dependencies"
	@echo "  make setup-mobile          Install mobile dependencies"
	@echo ""
	@echo "Development:"
	@echo "  make dev                   Start all services (db, backend, mobile)"
	@echo "  make db-up                 Start PostgreSQL + Redis"
	@echo "  make db-down               Stop database services"
	@echo "  make db-reset              Wipe and reinitialize database"
	@echo "  make backend               Run Go backend server (port 8080)"
	@echo "  make mobile                Run Expo dev server (port 8081)"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test                  Run all tests (backend + mobile)"
	@echo "  make backend-test          Run backend tests"
	@echo "  make mobile-test           Run mobile tests"
	@echo "  make lint                  Lint backend + mobile"
	@echo ""
	@echo "Mobile Builds (iOS):"
	@echo "  make mobile-build-ios-testflight   Build for TestFlight (requires Apple account)"
	@echo "  make mobile-build-ios-release      Build for App Store (requires Apple account)"
	@echo ""

# Setup
setup: setup-backend setup-mobile
	@echo "✓ All dependencies installed"

setup-backend:
	@echo "Installing backend dependencies..."
	cd netme-backend && go mod download
	@echo "✓ Backend ready"

setup-mobile:
	@echo "Installing mobile dependencies..."
	cd netme-mobile && npm install
	@echo "✓ Mobile ready"

# Start everything at once
start: db-up
	@echo ""
	@echo "🚀 Starting all services..."
	@echo ""
	@echo "Backend will start on: http://localhost:8080"
	@echo "Mobile will start on: http://localhost:8081"
	@echo ""
	@echo "When Expo starts, press 'i' to open iOS Simulator"
	@echo ""
	@echo "Press Ctrl+C to stop everything"
	@echo ""
	@(cd netme-backend && go run cmd/server/main.go &) && (cd netme-mobile && npm start)

# Development
dev: db-up backend mobile
	@echo "All services running. Ctrl+C to stop."

db-up:
	@echo "Starting PostgreSQL + Redis..."
	docker-compose up -d postgres redis
	@echo "Waiting for database..."
	@sleep 3
	@cd netme-backend && go run cmd/migrate/main.go || true
	@echo "✓ Database ready"

db-down:
	docker-compose down

db-reset:
	@echo "Resetting database..."
	docker-compose down -v
	docker-compose up -d postgres redis
	@sleep 3
	@cd netme-backend && go run cmd/migrate/main.go
	@echo "✓ Database reset"

backend:
	@cd netme-backend && go run cmd/server/main.go

mobile:
	@cd netme-mobile && npm start

# Testing
test: backend-test mobile-test

backend-test:
	@cd netme-backend && go test ./...

mobile-test:
	@cd netme-mobile && npm test

# Quality
lint:
	@echo "Linting backend..."
	@cd netme-backend && golangci-lint run ./... || true
	@echo "Linting mobile..."
	@cd netme-mobile && npm run lint || true

# Mobile builds (iOS)
mobile-build-ios-testflight:
	@echo "Building for TestFlight (iOS)..."
	@cd netme-mobile && eas build --platform ios --profile testflight

mobile-build-ios-release:
	@echo "Building for App Store (iOS)..."
	@cd netme-mobile && eas build --platform ios --profile release

.DEFAULT_GOAL := help
