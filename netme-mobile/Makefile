.PHONY: help install start ios android web build clean setup lint format test

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
NC := \033[0m # No Color

help: ## Display this help message
	@echo "$(BLUE)NetMe Mobile - Available Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Quick Start:$(NC)"
	@echo "  make setup        - Install everything"
	@echo "  make start        - Start dev server"
	@echo "  make ios          - Run on iOS simulator"
	@echo "  make android      - Run on Android emulator"
	@echo "  make web          - Run on web browser"
	@echo ""

install: ## Install npm dependencies
	@echo "$(BLUE)Installing dependencies...$(NC)"
	npm install
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

install-secure-store: ## Install secure storage (Keychain/Secure Storage)
	@echo "$(BLUE)Installing expo-secure-store...$(NC)"
	npx expo install expo-secure-store
	@echo "$(GREEN)✓ Secure store installed$(NC)"

setup: install install-secure-store ## Complete setup (install all dependencies)
	@echo "$(BLUE)Setting up environment...$(NC)"
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(GREEN)✓ .env file created (update with your backend URL)$(NC)"; \
	else \
		echo "$(GREEN)✓ .env file already exists$(NC)"; \
	fi
	@echo "$(GREEN)✓ Setup complete$(NC)"
	@echo ""
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "  1. Update .env with your backend URL"
	@echo "  2. Run: make start"
	@echo "  3. Choose: i (iOS), a (Android), or w (Web)"

start: ## Start Expo dev server
	@echo "$(BLUE)Starting Expo development server...$(NC)"
	npm start

dev: start ## Alias for start

ios: ## Run on iOS simulator
	@echo "$(BLUE)Running on iOS simulator...$(NC)"
	npm run ios

android: ## Run on Android emulator
	@echo "$(BLUE)Running on Android emulator...$(NC)"
	npm run android

web: ## Run on web browser
	@echo "$(BLUE)Running on web browser...$(NC)"
	npm run web

build-ios: ## Build for iOS (requires Xcode)
	@echo "$(BLUE)Building for iOS...$(NC)"
	npx eas build --platform ios
	@echo "$(GREEN)✓ iOS build complete$(NC)"

build-android: ## Build for Android (requires Android SDK)
	@echo "$(BLUE)Building for Android...$(NC)"
	npx eas build --platform android
	@echo "$(GREEN)✓ Android build complete$(NC)"

build-web: ## Build for web
	@echo "$(BLUE)Building for web...$(NC)"
	npm run build:web
	@echo "$(GREEN)✓ Web build complete$(NC)"

clean: ## Clean cache and build artifacts
	@echo "$(BLUE)Cleaning cache and artifacts...$(NC)"
	rm -rf node_modules/.cache
	rm -rf .expo
	@echo "$(GREEN)✓ Clean complete$(NC)"

clean-full: ## Full clean (removes node_modules)
	@echo "$(RED)Removing node_modules...$(NC)"
	rm -rf node_modules package-lock.json
	@echo "$(GREEN)✓ Full clean complete$(NC)"
	@echo "$(YELLOW)Run 'make setup' to reinstall$(NC)"

reinstall: clean-full setup ## Full reinstall
	@echo "$(GREEN)✓ Reinstall complete$(NC)"

lint: ## Run ESLint
	@echo "$(BLUE)Running linter...$(NC)"
	npm run lint

lint-fix: ## Fix linting issues
	@echo "$(BLUE)Fixing linting issues...$(NC)"
	npm run lint -- --fix
	@echo "$(GREEN)✓ Linting fixed$(NC)"

format: ## Format code with Prettier
	@echo "$(BLUE)Formatting code...$(NC)"
	npm run format
	@echo "$(GREEN)✓ Code formatted$(NC)"

format-check: ## Check code formatting
	@echo "$(BLUE)Checking code format...$(NC)"
	npm run format:check

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	npm test

test-watch: ## Run tests in watch mode
	@echo "$(BLUE)Running tests (watch mode)...$(NC)"
	npm test -- --watch

type-check: ## Run TypeScript type checking
	@echo "$(BLUE)Type checking...$(NC)"
	npx tsc --noEmit

env-setup: ## Setup environment file from example
	@echo "$(BLUE)Setting up .env file...$(NC)"
	@if [ -f .env ]; then \
		echo "$(YELLOW).env already exists, skipping$(NC)"; \
	else \
		cp .env.example .env; \
		echo "$(GREEN)✓ .env file created$(NC)"; \
		echo "$(YELLOW)Update .env with your backend URL:$(NC)"; \
		echo "  REACT_APP_API_URL=http://localhost:8080/api/v1"; \
	fi

cache-clear: ## Clear Metro bundler cache
	@echo "$(BLUE)Clearing Metro cache...$(NC)"
	npm start -- --clear
	@echo "$(GREEN)✓ Cache cleared$(NC)"

tunnel: ## Use tunnel for external device testing
	@echo "$(BLUE)Starting with tunnel...$(NC)"
	npx expo start --tunnel

lan: ## Use LAN for local network testing
	@echo "$(BLUE)Starting with LAN...$(NC)"
	npx expo start --lan

check-env: ## Check if .env file exists and is valid
	@echo "$(BLUE)Checking environment...$(NC)"
	@if [ -f .env ]; then \
		echo "$(GREEN)✓ .env file exists$(NC)"; \
		echo ""; \
		echo "$(BLUE)Current settings:$(NC)"; \
		cat .env; \
	else \
		echo "$(RED)✗ .env file not found$(NC)"; \
		echo "Run: make env-setup"; \
	fi

test-backend-connection: ## Test connection to backend
	@echo "$(BLUE)Testing backend connection...$(NC)"
	@curl -s http://localhost:8080/healthz | jq . && echo "$(GREEN)✓ Backend is running$(NC)" || echo "$(RED)✗ Backend not responding$(NC)"

info: ## Show project info
	@echo "$(BLUE)NetMe Mobile Project Info$(NC)"
	@echo ""
	@echo "Framework: React Native with Expo"
	@echo "Language: TypeScript"
	@echo "Package Manager: npm"
	@echo ""
	@echo "Key Directories:"
	@echo "  src/                - Source code"
	@echo "  src/screens/        - Screen components"
	@echo "  src/context/        - Context providers"
	@echo "  src/services/       - API services"
	@echo "  src/navigation/     - Navigation config"
	@echo ""
	@echo "Configuration:"
	@echo "  .env.example        - Environment template"
	@echo "  app.json            - Expo configuration"
	@echo "  package.json        - Dependencies"
	@echo ""

reset-cache: clean cache-clear ## Reset all caches
	@echo "$(GREEN)✓ All caches reset$(NC)"

.DEFAULT_GOAL := help
