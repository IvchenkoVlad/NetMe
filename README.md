# NetMe — Workspace

A personal finance app with separate backend and mobile services.

## Workspace Structure

```
netme/
├── netme-backend/          ← Go REST API (can be separate repo)
├── netme-mobile/           ← React Native app (can be separate repo)
├── docs/                   ← Shared documentation
├── docker-compose.yml      ← Local development services
├── Makefile                ← Workspace commands
└── .env.example            ← Example environment variables
```

## Quick Start

```bash
# Install all dependencies
make setup

# Start all local services (postgres, redis, backend, mobile)
make dev

# Or run services separately
make db-up        # PostgreSQL + Redis
make backend      # Go API (port 8080)
make mobile       # Expo dev server (port 8081)
```

## Development

Each service is independent but can share local infrastructure:

### Backend (`netme-backend/`)
- **Tech:** Go 1.22, Gin, PostgreSQL, Redis
- **Start:** `make backend`
- **Tests:** `make backend-test`
- **Docs:** `netme-backend/README.md`

### Mobile (`netme-mobile/`)
- **Tech:** React Native, Expo, TypeScript
- **Start:** `make mobile`
- **Tests:** `make mobile-test`
- **Docs:** `netme-mobile/README.md`

## Deployment

### Backend
- Local: `make backend` → runs on port 8080
- Staging/Production: Docker container to cloud platform (Render, Fly.io, AWS)

### Mobile
- Local: Expo dev server
- TestFlight (iOS): `make mobile-build-ios-testflight`
- App Store (iOS): `make mobile-build-ios-release`

## Database

PostgreSQL + Redis are provided via `docker-compose.yml`.

```bash
# Start only database services
make db-up

# Reset database (wipe everything)
make db-reset
```

## Environment Variables

Copy `.env.example` to `.env.local`:

```bash
cp .env.example .env.local
```

Backend and mobile read from their respective `.env` files.

## Next Steps

1. **Backend setup:** `cd netme-backend && make setup`
2. **Mobile setup:** `cd netme-mobile && npm install`
3. **Start dev:** `make dev`
4. **Build features:** Follow the MVP plan in `docs/MVP_PLAN.md`

## MVP Roadmap

See `docs/MVP_PLAN.md` for the full implementation roadmap (8 milestones, from foundation to beta-ready).

## Useful Commands

```bash
# Workspace level
make setup          # Install all dependencies
make dev            # Start everything
make db-up          # Start postgres + redis
make db-reset       # Wipe and reinit database
make backend        # Run backend in foreground
make mobile         # Run mobile dev server
make lint           # Lint both services
make test           # Run all tests

# Service level (see each service's README and Makefile)
cd netme-backend && make help
cd netme-mobile && npm run
```

## Questions?

- Backend docs: `netme-backend/README.md`
- Mobile docs: `netme-mobile/README.md`
- Architecture: `docs/ARCHITECTURE.md`
- API spec: `docs/API.md`
- Database: `docs/DATABASE.md`
