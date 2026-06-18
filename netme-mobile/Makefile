.PHONY: help install setup start dev ios android web build-ios build-android clean clean-full reinstall lint lint-fix format type-check cache-clear tunnel lan

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-20s %s\n", $$1, $$2}'

install: ## Install dependencies
	npm install

setup: install ## Install dependencies

start: ## Start Expo dev server
	npm start

dev: start ## Alias for start

ios: ## Run on iOS simulator
	npm run ios

android: ## Run on Android emulator
	npm run android

web: ## Run on web browser
	npm run web

build-ios: ## EAS build for iOS
	npx eas build --platform ios

build-android: ## EAS build for Android
	npx eas build --platform android

clean: ## Clear cache
	rm -rf node_modules/.cache .expo

clean-full: ## Remove node_modules
	rm -rf node_modules package-lock.json

reinstall: clean-full install ## Full reinstall

cache-clear: ## Clear Metro cache
	npm start -- --clear

tunnel: ## Start with tunnel
	npx expo start --tunnel

lan: ## Start with LAN
	npx expo start --lan

lint: ## Run ESLint
	npm run lint

lint-fix: ## Fix lint issues
	npm run lint -- --fix

format: ## Format with Prettier
	npm run format

type-check: ## TypeScript check
	npx tsc --noEmit

.DEFAULT_GOAL := help
