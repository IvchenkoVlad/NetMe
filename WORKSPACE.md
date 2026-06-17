# NetMe Workspace Guide

This is a **separate-services workspace** — backend and mobile are independent but co-located for local development.

## Why Separate Services?

| Benefit | Why It Matters |
|---------|----------------|
| **Independent deployment** | Mobile can ship on TestFlight while backend is still in staging |
| **Separate release cycles** | Backend updates don't force mobile rebuilds |
| **Clear separation of concerns** | Frontend/backend teams don't interfere |
| **Easy to extract later** | Can split into separate GitHub repos anytime |
| **Better CI/CD** | Each service has its own test/build pipeline |

## Directory Organization

```
/Desktop/netme/                  ← Workspace root
├── netme-backend/               ← Go backend (can be separate repo)
│   ├── cmd/
│   │   ├── server/              ← HTTP API
│   │   ├── migrate/             ← DB migrations
│   │   └── worker/              ← Background jobs (TODO)
│   ├── internal/
│   │   ├── users/               ← User domain
│   │   ├── auth/                ← Auth logic
│   │   ├── accounts/            ← Bank accounts
│   │   ├── transactions/        ← Transaction handling
│   │   ├── categories/          ← Categories
│   │   ├── rules/               ← Merchant rules
│   │   ├── budgets/             ← Budget logic
│   │   ├── plaid/               ← Plaid integration
│   │   ├── sync/                ← Transaction sync pipeline
│   │   ├── jobs/                ← Job queue
│   │   ├── db/                  ← Database & migrations
│   │   ├── config/              ← Configuration
│   │   ├── logger/              ← Logging
│   │   ├── crypto/              ← Token encryption
│   │   └── server/              ← HTTP setup
│   ├── go.mod
│   ├── README.md
│   └── .gitignore
│
├── netme-mobile/                ← React Native app (can be separate repo)
│   ├── app/
│   │   ├── screens/             ← Screens (Home, Transactions, etc)
│   │   ├── components/          ← Reusable components
│   │   ├── hooks/               ← Custom hooks
│   │   ├── utils/               ← API client, helpers
│   │   ├── types/               ← TypeScript types
│   │   ├── stores/              ← Zustand state
│   │   ├── navigation/          ← Navigation setup
│   │   └── App.tsx              ← Root component
│   ├── app.json
│   ├── package.json
│   ├── eas.json                 ← Expo build config
│   ├── README.md
│   └── .gitignore
│
├── docs/                        ← Shared documentation
│   ├── ARCHITECTURE.md
│   ├── API.md
│   ├── DATABASE.md
│   └── MVP_PLAN.md
│
├── docker-compose.yml           ← PostgreSQL + Redis for local dev
├── Makefile                     ← Workspace-level commands
├── .env.example                 ← Example environment variables
├── README.md                    ← Workspace overview
└── WORKSPACE.md                 ← This file
```

## Local Development Workflow

### 1. Initial Setup

```bash
# From /Desktop/netme
make setup          # Install dependencies for both services
make db-up          # Start PostgreSQL + Redis
```

### 2. Development

**Option A: All services in background**
```bash
make dev            # Starts db, backend, mobile all together
# Then open simulator/device and navigate to app
```

**Option B: Separate terminals (recommended)**

Terminal 1: Database
```bash
cd /Desktop/netme
make db-up
```

Terminal 2: Backend
```bash
cd /Desktop/netme/netme-backend
go run cmd/server/main.go
# Runs on http://localhost:8080/api/v1
```

Terminal 3: Mobile
```bash
cd /Desktop/netme/netme-mobile
npm start
# Starts Expo dev server on port 8081
# Scan QR code with device or press `i` for iOS simulator
```

### 3. Testing

```bash
# From workspace root
make test           # Run all tests (backend + mobile)

# Or individually
cd netme-backend && go test ./...
cd netme-mobile && npm test
```

### 4. Code Quality

```bash
make lint           # Lint both services
```

## Git & Repository Structure

### Current: Single Git Repository

Everything is in one git repo (`/Desktop/netme/.git`).

**Advantages:**
- Easy initial setup
- Shared history
- Simple local development

**Disadvantages:**
- Can't deploy backend without mobile code
- Release notes are mixed

### Later: Separate Git Repositories

When ready to push to production, split into:

```
github.com/vladyslavivchenko/netme-backend
github.com/vladyslavivchenko/netme-mobile
github.com/vladyslavivchenko/netme-docs        (optional)
```

**How to split later:**
1. Create empty repos on GitHub
2. Use `git subtree split` or create new repos with history
3. Update CI/CD to deploy from separate repos
4. Both can have independent version numbers

For now, develop locally and commit everything to one repo.

## Database Persistence

PostgreSQL + Redis data persists in Docker volumes:

```bash
make db-reset       # Wipe everything and start fresh
make db-up          # Resume with existing data
make db-down        # Stop but keep data
```

### Database Setup

Migrations run automatically when you `make db-up`:

```bash
# Manual migrations
cd netme-backend
go run cmd/migrate/main.go up      # Apply pending
go run cmd/migrate/main.go down    # Rollback one
```

## Environment Variables

### Workspace Level (`.env.local`)

```bash
cp .env.example .env.local
# Edit .env.local with your values
```

### Service Level

Each service reads from workspace `.env.local`:
- Backend reads `DATABASE_URL`, `REDIS_URL`, `API_PORT`, etc.
- Mobile reads `MOBILE_API_URL`, `PLAID_CLIENT_ID`, etc.

## Building & Deployment

### Before MVP Submission

Everything is local. No deployment yet.

```bash
make backend        # Run locally
make mobile         # Run in Expo dev server
```

### When Ready for Beta (TestFlight)

```bash
cd netme-mobile
npm install -g eas-cli
eas login           # Create/login to Expo account
eas build --platform ios --profile testflight
# Prompts for Apple Developer account setup
# Builds in cloud, uploads to TestFlight automatically
```

### When Ready for App Store

```bash
eas build --platform ios --profile release
eas submit --platform ios
```

See `netme-mobile/README.md` for detailed instructions.

## Making Code Changes

### Backend

```bash
# Edit code in netme-backend/
# Changes hot-reload (Gin watches for changes)
# OR restart: Ctrl+C, then `go run cmd/server/main.go`

# Add new endpoint
# 1. Create handler in internal/{domain}/handler.go
# 2. Create service logic in internal/{domain}/service.go
# 3. Create repository in internal/{domain}/repository.go
# 4. Wire it up in internal/server/routes.go
# 5. Write tests
```

### Mobile

```bash
# Edit code in netme-mobile/
# Changes hot-reload automatically in Expo
# Just save the file, app updates on device/simulator

# Add new screen
# 1. Create screen in app/screens/SomethingScreen.tsx
# 2. Add to navigation in app/navigation/
# 3. Add route in app/App.tsx
```

## Debugging

### Backend

```bash
# View logs
tail -f /var/log/netme-backend.log

# Enable debug logging
export LOG_LEVEL=debug

# Use Postman/curl to test endpoints
curl http://localhost:8080/api/v1/healthz
```

### Mobile

```bash
# Expo DevTools
# Press 'i' in terminal running npm start
# Opens in Expo Go app on device

# React DevTools
npm install --save-dev @react-devtools/core

# Network debugging
# Check API calls in Reactotron or browser dev tools
```

## Common Tasks

### Add a new feature

1. **Backend:**
   - Create domain module in `internal/{feature}/`
   - Implement handler, service, repository
   - Add database migration if needed
   - Add endpoint in routes
   - Write tests

2. **Mobile:**
   - Create screen in `app/screens/`
   - Create components as needed
   - Create types in `app/types/`
   - Create store if state needed
   - Add navigation
   - Add API call hook in `app/hooks/`

3. **Integration:**
   - Test backend endpoint with curl
   - Test mobile app calls endpoint
   - Check data flows both directions

### Fix a bug

1. Identify which service (backend or mobile)
2. Write a failing test (if possible)
3. Fix the code
4. Test locally
5. Commit with explanation

### Reset everything

```bash
make db-reset       # Wipe database
# Backend restarts automatically
# Mobile hot-reloads
```

## Troubleshooting

### Backend won't start

```bash
# Check database is running
docker-compose ps

# Check port 8080 isn't in use
lsof -i :8080

# Check logs
cd netme-backend && go run cmd/server/main.go 2>&1 | head -50
```

### Mobile won't connect to backend

```bash
# Check MOBILE_API_URL is correct
grep MOBILE_API_URL .env.local

# Check backend is running
curl http://localhost:8080/api/v1/healthz

# Check firewall allows localhost:8080
# On Mac, usually allowed automatically
```

### Database migration fails

```bash
# View current schema
cd netme-backend
go run cmd/migrate/main.go status

# Rollback
go run cmd/migrate/main.go down

# Check migration file for syntax errors
ls internal/db/migrations/tables/
```

## Next Steps

1. ✅ **Workspace created** — You're here
2. **Scaffold backend domains** — Create service/repo layers
3. **Scaffold mobile screens** — Create screen components
4. **Implement authentication** — Login/signup flow
5. **Connect backend ↔ mobile** — First API call
6. **Implement bank linking** — Plaid integration
7. **Implement transactions** — Full flow
8. **Build dashboard** — Summary view
9. **Local testing** — Make sure everything works
10. **TestFlight beta** — Submit to Apple
11. **App Store** — Public release

## Resources

- **Backend:** `netme-backend/README.md`
- **Mobile:** `netme-mobile/README.md`
- **Architecture:** `docs/ARCHITECTURE.md`
- **API:** `docs/API.md`
- **Database:** `docs/DATABASE.md`
- **MVP Plan:** `docs/MVP_PLAN.md` (the full spec)

## Questions?

Refer to the README files in each service, or the docs folder for architecture and API details.
