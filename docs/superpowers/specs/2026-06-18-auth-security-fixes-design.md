# Auth Security Fixes — Design Spec

**Date:** 2026-06-18
**Scope:** Four critical auth fixes required before any real users or TestFlight distribution.

---

## 1. Fix 1 — Account Takeover in `FindOrCreateGoogleUser`

### Problem
`user.go:88–93` uses `ON CONFLICT (email) DO UPDATE SET auth_provider_user_id = $2`. If a user registered with email/password, an attacker controlling a Google account with the same email can sign in and take over the local account.

### Fix
Replace the upsert with a three-step safe lookup in `FindOrCreateGoogleUser`:

1. Query `WHERE auth_provider = 'google' AND auth_provider_user_id = $1` — if found, return user (existing Google login, normal path).
2. Query `WHERE email = $1` — if found with a **different** `auth_provider`, return sentinel error `ErrEmailTakenByOtherProvider`.
3. If neither found — `INSERT` new user with `auth_provider = 'google'`, `auth_provider_user_id = googleID`.

`GoogleAuth` handler maps `ErrEmailTakenByOtherProvider` → `409 Conflict`:
```json
{"error": "email_conflict", "message": "An account with this email already exists. Please log in with your password."}
```

No implicit account merging. Explicit account linking is a post-MVP feature.

**Files:**
- `internal/repositories/user.go` — rewrite `FindOrCreateGoogleUser`
- `internal/repositories/errors.go` (new) — define `ErrEmailTakenByOtherProvider`
- `internal/handlers/auth.go` — map error to 409 in `GoogleAuth`

---

## 2. Fix 2 — Google OAuth Uses Wrong Token Type

### Problem
`auth.go:248` sends a Google **access token** to `/oauth2/v1/userinfo` via `http.Get`. This is the wrong token type (access tokens are short-lived and client-scoped), uses the deprecated v1 endpoint, has no HTTP timeout, and leaks DB errors in the response.

### Fix
Mobile sends a Google **ID token** (JWT). Backend verifies it with `google.golang.org/api/idtoken`:

```go
payload, err := idtoken.Validate(c.Request.Context(), req.IDToken, os.Getenv("GOOGLE_CLIENT_ID"))
```

- `payload.Subject` → Google user ID
- `payload.Claims["email"]` → email
- No outbound HTTP call on the verification path (uses cached Google public keys)
- Request body field: `access_token` → `id_token`
- `GOOGLE_CLIENT_ID` added to `.env.example`
- Error responses never expose internal error details

**Files:**
- `internal/handlers/auth.go` — replace `http.Get` block with `idtoken.Validate`; rename request field
- `netme-backend/.env.example` — add `GOOGLE_CLIENT_ID`
- `netme-mobile/src/services/authService.ts` — rename field sent to `id_token`
- `go.mod` — add `google.golang.org/api`

---

## 3. Fix 3 — No Rate Limiting on Auth Endpoints

### Problem
`login`, `register`, and `refresh` have no rate limiting. Unlimited brute-force attempts possible.

### Fix
New `internal/middleware/ratelimit.go` using `golang.org/x/time/rate`:

- Per-IP limiter store: `map[string]*rate.Limiter` guarded by `sync.Mutex`
- `RateLimiter(r rate.Limit, b int) gin.HandlerFunc` — parameterised constructor
- IP extracted from `c.ClientIP()`
- Limits applied per-route in `app.go`:
  - `POST /v1/auth/login` — 10 req/min (burst 10)
  - `POST /v1/auth/refresh` — 10 req/min (burst 10)
  - `POST /v1/auth/register` — 5 req/min (burst 5)
  - `POST /v1/auth/google` — 10 req/min (burst 10)
- 429 response:
  ```json
  {"error": "rate_limit_exceeded", "message": "Too many attempts. Please try again later."}
  ```
- No background cleanup for MVP (in-memory accumulation acceptable at MVP scale)

**Files:**
- `internal/middleware/ratelimit.go` (new)
- `internal/app/app.go` — apply per-route middleware

---

## 4. Fix 4 — Refresh Tokens Not Rotated

### Problem
`Refresh` handler returns `RefreshToken: req.RefreshToken` — the same token forever. A stolen refresh token gives indefinite access until 7-day expiry.

### Fix
Rotate on every successful refresh:

1. Validate old token (existing check)
2. Fetch user (existing)
3. Generate new refresh token string
4. `CreateRefreshToken` (store new) — if this fails, return 500, old token untouched
5. `RevokeRefreshToken(oldToken, userID)` — if this fails, log warning but still return success with new token (old token expires naturally; avoids locking user out on transient DB error)
6. Return new refresh token in response

**Mobile:** `AuthContext.tsx` `refreshAccessToken` must also:
- `setRefreshToken(response.refresh_token)`
- `secureStorage.saveRefreshToken(response.refresh_token)`

**Files:**
- `internal/handlers/auth.go` — update `Refresh` method
- `netme-mobile/src/context/AuthContext.tsx` — persist rotated token

---

## Out of Scope for This Pass

- Sign in with Apple (separate milestone)
- Account deletion implementation
- Email normalization
- TOCTOU race fix on register
- Mobile type definition cleanup
- Expired token check during bootstrap
