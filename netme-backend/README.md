# NetMe Backend

Go REST API for the NetMe personal finance app.

## Stack

- **Language:** Go 1.22
- **Framework:** Gin (HTTP router)
- **Database:** PostgreSQL
- **Cache/Jobs:** Redis
- **Architecture:** Modular monolith with handler-service-repository pattern

## Structure

```
cmd/
  server/          ← Main API server
  migrate/         ← Database migrations
  worker/          ← Background job worker (TODO)

internal/
  users/           ← User domain (handler, service, repository, model)
  auth/            ← Authentication logic
  accounts/        ← Bank accounts
  transactions/    ← Transactions
  categories/      ← Categories
  rules/           ← Category rules (merchant mapping)
  budgets/         ← Monthly budgets
  dashboard/       ← Dashboard aggregation
  institutions/    ← Bank institutions
  plaid/           ← Plaid API integration
  sync/            ← Transaction sync pipeline
  recurring/       ← Recurring transaction detection
  jobs/            ← Background job queue
  db/              ← Database setup, migrations
  config/          ← Configuration management
  logger/          ← Structured logging
  crypto/          ← Token encryption
  server/          ← HTTP server setup
  models/          ← Shared data models
```

## Quick Start

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- Redis 7+

### Local Development

```bash
# Install dependencies
go mod download

# Start database services (from workspace root)
make db-up

# Run migrations
go run cmd/migrate/main.go

# Start API server (port 8080)
go run cmd/server/main.go
```

### Development Commands

```bash
make help                # Show all workspace commands
make backend             # Run backend server
make backend-test        # Run tests
make db-reset            # Reset database
```

## API

- **Base URL:** `http://localhost:8080/api/v1`
- **Health:** `GET /healthz`
- **Docs:** See `docs/API.md` in workspace root

## Databases

### PostgreSQL

Users, accounts, transactions, categories, rules, budgets, etc.

Connection: `postgres://netme:devpassword@localhost:5432/netme_dev`

Migrations are in `internal/db/migrations/`

### Redis

Cache and job queue. Upgrade to Asynq later if needed.

Connection: `redis://localhost:6379`

## Architecture Pattern

Every feature follows this pattern:

```
HTTP Handler (parse, validate, call service)
    ↓
Service (business logic, call repository)
    ↓
Repository (database queries)
    ↓
Database
```

Example:
```
handlers/accounts.go
    ↓ calls
accounts/service.go
    ↓ calls
accounts/repository.go
    ↓ queries
PostgreSQL
```

## Key Endpoints (MVP)

**Auth:**
- POST `/auth/register` — Create account
- POST `/auth/login` — Get JWT
- DELETE `/me` — Delete account

**Accounts:**
- GET `/accounts` — List accounts
- GET `/accounts/:id` — Get account detail
- PATCH `/accounts/:id` — Hide/show account

**Transactions:**
- GET `/transactions?month=2026-06&account_id=&category_id=` — List
- GET `/transactions/:id` — Detail
- PATCH `/transactions/:id` — Update category/exclude

**Categories:**
- GET `/categories` — List
- POST `/categories` — Create
- PATCH `/categories/:id` — Update
- DELETE `/categories/:id` — Delete

**Budgets:**
- GET `/budgets?month=2026-06` — Get budgets
- PUT `/budgets?month=2026-06` — Set budgets
- GET `/budgets/summary?month=2026-06` — Summary

**Dashboard:**
- GET `/dashboard?month=2026-06` — Dashboard data

## Database Schema

See `docs/DATABASE.md` in workspace root.

Key tables:
- `users` — User accounts
- `accounts` — Bank accounts
- `transactions` — Transactions
- `categories` — Categories
- `category_rules` — Merchant → category mapping
- `budgets` — Monthly budgets
- `linked_items` — Plaid connections
- `institutions` — Banks

## Testing

```bash
go test ./...              # Run all tests
go test -v ./...           # Verbose
go test ./internal/users   # Test one package
```

## Environment Variables

See `.env.example` in workspace root. Typically:

```
DATABASE_URL=postgres://netme:devpassword@localhost:5432/netme_dev
REDIS_URL=redis://localhost:6379
API_PORT=8080
API_ENV=development
MOBILE_API_URL=http://localhost:8080/api/v1
```

## Next Steps

1. Complete domain modules (implement services and repositories)
2. Implement auth (JWT or managed provider)
3. Implement Plaid integration
4. Build transaction sync pipeline
5. Add tests throughout

## Resources

- **Go:** https://golang.org
- **Gin:** https://github.com/gin-gonic/gin
- **PostgreSQL:** https://www.postgresql.org
- **MVP Plan:** `docs/MVP_PLAN.md` (workspace root)
