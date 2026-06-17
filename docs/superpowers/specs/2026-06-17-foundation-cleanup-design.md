# NetMe Foundation Cleanup — Design Spec

**Date:** 2026-06-17  
**Scope:** Cleanup, bug fixes, migrations, handler alignment, logging, and auth tests.  
**Approach:** Single coordinated pass (Option A).

---

## 1. Deletions / Clutter Removal

| Target | Action |
|---|---|
| `netme-mobile/app/` | Delete entire folder (dead code, conflicts with `src/`) |
| `docs/docs/` | Delete duplicate nested folder |
| `internal/handlers/analytics.go` | Delete (net-worth is out of MVP scope; all stubs) |
| Redis in `app.go` + `go.mod` | Remove — plan says Postgres jobs first |
| `GET /api/v1/hello` | Remove from health handler |
| `AuthHandler.LogoutAllDevices` | Remove — dead code, wrong signature |
| `internal/db/migrations.go` | Remove — replaced by goose |
| `internal/db/migrations/tables/`, `indices/` | Remove — replaced by flat goose migrations dir |

---

## 2. Bug Fixes

### CORS
Replace `Access-Control-Allow-Origin: *` with an origin allowlist read from `CORS_ALLOWED_ORIGINS` env var (comma-separated). Default: `http://localhost:8081`. Fixes browser rejection of credentialed requests with wildcard origin.

### Auth middleware not applied to protected routes
Create a protected route group in `app.go`. All non-auth endpoints are registered under it with `AuthMiddleware()` applied.

### Hardcoded JWT secret fallback
Remove `"your-secret-key-change-in-production"` default. If `JWT_SECRET_KEY` is empty at startup, `log.Fatal` and exit.

### Logout not protected
Move `POST /v1/auth/logout` into the protected group so `AuthMiddleware` validates the access token before the handler runs. Remove the manual header-parsing logic from the `Logout` handler.

### API base path
Change `/api/v1` → `/v1` to match the MVP spec.

### User model alignment
Remove: `display_name`, `picture_url`, `last_login_at`.  
Add: `auth_provider TEXT NOT NULL DEFAULT 'local'`, `auth_provider_user_id TEXT UNIQUE`.

---

## 3. Database Migrations (goose)

**Migration tooling:** Replace custom loader with `goose` (`github.com/pressly/goose/v3`).

**Migration files** (flat `internal/db/migrations/` directory):

```
0001_create_users.sql
0002_create_refresh_tokens.sql
```

**`0001_create_users.sql`:**
```sql
-- +goose Up
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT,
  auth_provider TEXT NOT NULL DEFAULT 'local',
  auth_provider_user_id TEXT UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
```

**`0002_create_refresh_tokens.sql`:**
```sql
-- +goose Up
CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token TEXT UNIQUE NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- +goose Down
DROP TABLE refresh_tokens;
```

**`cmd/migrate/main.go`:** Thin goose wrapper supporting `up`, `down`, `status` commands.

---

## 4. Handler Pattern + New Endpoints

### Pattern
All handlers use struct-based pattern (matching current `auth.go`). Each package has:
- `Handler` struct with injected dependencies
- `NewHandler(db *sql.DB) *Handler` constructor
- `RegisterRoutes(router *gin.RouterGroup)` method

### Repository split
`AuthRepository` is split into:
- `UserRepository` — `CreateUser`, `GetUserByID`, `GetUserByEmail`, `UpdateLastLogin`
- `AuthRepository` (renamed `TokenRepository`) — `CreateRefreshToken`, `GetRefreshToken`, `RevokeRefreshToken`, `RevokeAllUserTokens`, `IsRefreshTokenValid`

Both receive `*sql.DB` in their constructors. `AuthHandler` is updated to use both.

### New endpoints
- `GET /v1/me` — reads `user_id` from context (set by `AuthMiddleware`), queries `UserRepository.GetUserByID`, returns user without `password_hash`
- `DELETE /v1/me` — returns `501 Not Implemented` (Milestone 8)

### Route grouping in `app.go`
```
Public:
  GET  /healthz
  POST /v1/auth/register
  POST /v1/auth/login
  POST /v1/auth/refresh

Protected (AuthMiddleware):
  POST   /v1/auth/logout
  GET    /v1/me
  DELETE /v1/me
  GET    /v1/accounts
  GET    /v1/accounts/:id
  GET    /v1/transactions
  GET    /v1/transactions/:id
```

---

## 5. Structured Logging

Use `log/slog` (stdlib). No new dependency.

- `internal/logger/logger.go` — initializes a global `*slog.Logger`; JSON handler when `API_ENV=production`, text handler otherwise
- `App` struct holds a `*slog.Logger`; passed to handlers that need it
- Log: server start/stop, DB connection success/failure, JWT secret missing
- Do not log: tokens, passwords, full request bodies, financial data

---

## 6. Tests

### `internal/services/jwt_test.go`
- Generate access token → verify returns correct claims
- Verify expired token → returns error
- Verify tampered token → returns error
- Verify token with wrong signing method → returns error

### `internal/services/password_test.go`
- Hash password → verify correct password succeeds
- Hash password → verify wrong password fails
- Hash empty string → error or expected behavior

### `internal/handlers/auth_test.go`
Handler tests use `httptest.NewRecorder` and a mock repository interface.

Define interfaces:
```go
type UserRepo interface {
    CreateUser(email, passwordHash string) (*models.User, error)
    GetUserByEmail(email string) (*models.User, error)
    GetUserByID(id string) (*models.User, error)
}

type TokenRepo interface {
    CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error)
    IsRefreshTokenValid(token string) (bool, error)
    GetRefreshToken(token string) (*models.RefreshToken, error)
    RevokeRefreshToken(token string) error
}
```

Test cases:
- `POST /v1/auth/register` happy path → 201 with tokens
- `POST /v1/auth/register` duplicate email → 409
- `POST /v1/auth/login` wrong password → 401
- `POST /v1/auth/login` unknown email → 401
- `POST /v1/auth/refresh` invalid token → 401

---

## Out of Scope for This Pass

- Plaid integration
- Accounts/transactions/budgets/categories implementation
- Worker binary
- Push notifications
- Account deletion logic (DELETE /v1/me returns 501)
