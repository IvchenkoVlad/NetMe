# Auth Security Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix four critical auth security issues: account takeover via Google OAuth, wrong Google token type, no rate limiting, and non-rotating refresh tokens.

**Architecture:** Backend fixes in `netme-backend/` touch repositories, a new services layer for Google token verification, middleware, and handlers. Mobile fix in `netme-mobile/` updates the token field name and persists rotated tokens. Tasks 1–2 are sequential (repo fix before handler fix). Tasks 3–4 are independent. Task 5 (mobile) depends on Tasks 2 and 4.

**Tech Stack:** Go 1.25+, Gin, `google.golang.org/api/idtoken`, `golang.org/x/time/rate`, React Native / Expo, expo-secure-store

## Global Constraints

- Working directory for all Go commands: `netme-backend/`
- Base API path: `/v1`
- No sensitive data (tokens, passwords, DB errors) in HTTP responses
- All handlers use struct pattern with constructor injection
- `GoogleAuth` request field is `id_token` (not `access_token`)
- Rate limits: login/refresh/google = 10 req/min burst 10; register = 5 req/min burst 5
- Refresh rotation: new token stored before old token revoked; revoke failure is logged, not fatal

---

## File Map

**Create:**
- `netme-backend/internal/repositories/errors.go` — sentinel `ErrEmailTakenByOtherProvider`
- `netme-backend/internal/services/google.go` — `GoogleVerifier` interface + `GoogleIDTokenVerifier` prod impl
- `netme-backend/internal/middleware/ratelimit.go` — `RateLimiter(r, b)` per-IP token bucket

**Modify:**
- `netme-backend/internal/repositories/user.go` — rewrite `FindOrCreateGoogleUser` (3-step safe lookup)
- `netme-backend/internal/handlers/auth.go` — inject `GoogleVerifier`, fix `GoogleAuth`, update `Refresh` for rotation; add 409 mapping
- `netme-backend/internal/handlers/auth_test.go` — add mock verifier, add Google auth tests, add rotation test
- `netme-backend/internal/app/app.go` — wire `GoogleIDTokenVerifier`, apply per-route rate limiting
- `netme-backend/.env.example` — add `GOOGLE_CLIENT_ID`
- `netme-backend/go.mod` — add `google.golang.org/api`, `golang.org/x/time`
- `netme-mobile/src/services/authService.ts` — rename field to `id_token` in `loginWithGoogle`
- `netme-mobile/src/context/AuthContext.tsx` — persist rotated refresh token in `refreshAccessToken`

---

### Task 1: Sentinel error + fix `FindOrCreateGoogleUser`

**Files:**
- Create: `netme-backend/internal/repositories/errors.go`
- Modify: `netme-backend/internal/repositories/user.go`

**Interfaces:**
- Produces: `repositories.ErrEmailTakenByOtherProvider` — sentinel error used by Task 2's handler
- Produces: `UserRepository.FindOrCreateGoogleUser(googleID, email string) (*models.User, error)` — safe 3-step lookup

- [ ] **Step 1: Create sentinel error file**

Create `netme-backend/internal/repositories/errors.go`:

```go
package repositories

import "errors"

// ErrEmailTakenByOtherProvider is returned when a Google sign-in email matches
// an existing account that uses a different auth provider.
var ErrEmailTakenByOtherProvider = errors.New("email is already registered with a different login method")
```

- [ ] **Step 2: Rewrite `FindOrCreateGoogleUser` in user.go**

Replace the existing `FindOrCreateGoogleUser` method (currently lines ~76–95) in `netme-backend/internal/repositories/user.go` with:

```go
func (r *UserRepository) FindOrCreateGoogleUser(googleID, email string) (*models.User, error) {
	user := &models.User{}

	// Step 1: existing Google user — normal login path
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE auth_provider = 'google' AND auth_provider_user_id = $1`,
		googleID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.AuthProvider,
		&user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Step 2: email exists with a different provider — refuse, prevent account takeover
	var existingProvider string
	err = r.db.QueryRow(`SELECT auth_provider FROM users WHERE email = $1`, email).Scan(&existingProvider)
	if err == nil {
		return nil, ErrEmailTakenByOtherProvider
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Step 3: new user — create Google account
	user = &models.User{}
	err = r.db.QueryRow(
		`INSERT INTO users (email, auth_provider, auth_provider_user_id, created_at, updated_at)
		 VALUES ($1, 'google', $2, now(), now())
		 RETURNING id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at`,
		email, googleID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.AuthProvider,
		&user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}
```

- [ ] **Step 3: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/repositories/errors.go internal/repositories/user.go
git commit -m "fix: prevent account takeover in FindOrCreateGoogleUser — check provider before upsert"
```

---

### Task 2: Fix `GoogleAuth` handler — use ID token, inject verifier, add tests

**Files:**
- Create: `netme-backend/internal/services/google.go`
- Modify: `netme-backend/internal/handlers/auth.go`
- Modify: `netme-backend/internal/handlers/auth_test.go`
- Modify: `netme-backend/internal/app/app.go`
- Modify: `netme-backend/.env.example`
- Modify: `netme-backend/go.mod`

**Interfaces:**
- Consumes: `repositories.ErrEmailTakenByOtherProvider` from Task 1
- Produces: `services.GoogleVerifier` interface — `Validate(ctx, idToken, audience string) (googleID, email string, err error)`
- Produces: `services.NewGoogleIDTokenVerifier(clientID string) *GoogleIDTokenVerifier`
- Produces: `handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc, googleVerifier)` — updated 4-arg constructor

- [ ] **Step 1: Add `google.golang.org/api` dependency**

```bash
cd netme-backend && go get google.golang.org/api/idtoken
```

Expected: `go.mod` updated with `google.golang.org/api`.

- [ ] **Step 2: Create `services/google.go`**

Create `netme-backend/internal/services/google.go`:

```go
package services

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
)

// GoogleVerifier abstracts Google ID token verification for testability.
type GoogleVerifier interface {
	Validate(ctx context.Context, idToken, audience string) (googleID string, email string, err error)
}

// GoogleIDTokenVerifier verifies Google ID tokens using Google's public keys.
type GoogleIDTokenVerifier struct {
	clientID string
}

func NewGoogleIDTokenVerifier(clientID string) *GoogleIDTokenVerifier {
	return &GoogleIDTokenVerifier{clientID: clientID}
}

func (v *GoogleIDTokenVerifier) Validate(ctx context.Context, idToken, audience string) (string, string, error) {
	payload, err := idtoken.Validate(ctx, idToken, audience)
	if err != nil {
		return "", "", err
	}
	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return "", "", fmt.Errorf("email not present in Google ID token")
	}
	return payload.Subject, email, nil
}
```

- [ ] **Step 3: Write failing tests for Google auth paths**

Add to `netme-backend/internal/handlers/auth_test.go`:

First, add a mock Google verifier near the top of the file (after the mock repos):

```go
// --- Mock GoogleVerifier ---

type mockGoogleVerifier struct {
	googleID string
	email    string
	err      error
}

func (m *mockGoogleVerifier) Validate(_ context.Context, _, _ string) (string, string, error) {
	return m.googleID, m.email, m.err
}
```

Update `newTestAuthRouter` to accept a verifier parameter and add the Google route:

```go
func newTestAuthRouter() (*gin.Engine, *mockUserRepo, *mockTokenRepo) {
	return newTestAuthRouterWithVerifier(&mockGoogleVerifier{
		googleID: "google-123",
		email:    "google@example.com",
	})
}

func newTestAuthRouterWithVerifier(verifier services.GoogleVerifier) (*gin.Engine, *mockUserRepo, *mockTokenRepo) {
	gin.SetMode(gin.TestMode)
	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	jwtSvc := services.NewJWTService(testAuthSecret)
	h := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc, verifier)

	r := gin.New()
	r.POST("/v1/auth/register", h.Register)
	r.POST("/v1/auth/login", h.Login)
	r.POST("/v1/auth/refresh", h.Refresh)
	r.POST("/v1/auth/logout", h.Logout)
	r.POST("/v1/auth/google", h.GoogleAuth)
	return r, userRepo, tokenRepo
}
```

Add three test functions:

```go
func TestGoogleAuthSuccess(t *testing.T) {
	verifier := &mockGoogleVerifier{googleID: "g-uid-1", email: "guser@example.com"}
	r, _, _ := newTestAuthRouterWithVerifier(verifier)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/google",
		jsonBody(t, map[string]string{"id_token": "valid-google-id-token"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp models.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
}

func TestGoogleAuthEmailConflict(t *testing.T) {
	// Verifier returns valid Google token, but email already exists with local provider
	verifier := &mockGoogleVerifier{googleID: "g-uid-2", email: "existing@example.com"}
	r, userRepo, _ := newTestAuthRouterWithVerifier(verifier)

	// Pre-populate mock with a local account using the same email
	userRepo.users["existing@example.com"] = &models.User{
		ID:           "local-user-1",
		Email:        "existing@example.com",
		PasswordHash: "somehash",
		AuthProvider: "local",
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/google",
		jsonBody(t, map[string]string{"id_token": "valid-token"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGoogleAuthInvalidToken(t *testing.T) {
	verifier := &mockGoogleVerifier{err: errors.New("invalid ID token")}
	r, _, _ := newTestAuthRouterWithVerifier(verifier)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/google",
		jsonBody(t, map[string]string{"id_token": "bad-token"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 4: Run tests to confirm they fail**

```bash
cd netme-backend && go test ./internal/handlers/... -v -run TestGoogle
```

Expected: compilation error (handlers.NewAuthHandler wrong arg count, GoogleVerifier not imported).

- [ ] **Step 5: Update `AuthHandler` and `GoogleAuth` in `auth.go`**

Add `googleVerifier services.GoogleVerifier` field to `AuthHandler`:

```go
type AuthHandler struct {
	userRepo        repositories.UserRepo
	tokenRepo       repositories.TokenRepo
	jwtService      *services.JWTService
	passwordService *services.PasswordService
	googleVerifier  services.GoogleVerifier
}

func NewAuthHandler(userRepo repositories.UserRepo, tokenRepo repositories.TokenRepo, jwtSvc *services.JWTService, googleVerifier services.GoogleVerifier) *AuthHandler {
	return &AuthHandler{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		jwtService:      jwtSvc,
		passwordService: services.NewPasswordService(),
		googleVerifier:  googleVerifier,
	}
}
```

Replace the `GoogleAuth` method entirely:

```go
func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req struct {
		IDToken string `json:"id_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	googleID, email, err := h.googleVerifier.Validate(c.Request.Context(), req.IDToken, "")
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid_token", Message: "Failed to verify Google token"})
		return
	}

	user, err := h.userRepo.FindOrCreateGoogleUser(googleID, email)
	if err != nil {
		if errors.Is(err, repositories.ErrEmailTakenByOtherProvider) {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "email_conflict",
				Message: "An account with this email already exists. Please log in with your password.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "db_error", Message: "Failed to sign in with Google"})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate token"})
		return
	}

	refreshTokenString, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate refresh token"})
		return
	}

	refreshToken, err := h.tokenRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to store refresh token"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900,
		User:         user,
	})
}
```

Add `"errors"` to imports in `auth.go` (needed for `errors.Is`). Also add `"github.com/vladyslavivchenko/netme/internal/repositories"` if not already present. Remove `"encoding/json"`, `"fmt"`, `"io"`, `"net/http"` package-level imports that were only used by the old `http.Get` approach (keep `"net/http"` for status codes).

- [ ] **Step 6: Update mock `FindOrCreateGoogleUser` in `auth_test.go`**

Update `mockUserRepo` to handle the Google user lookup — the mock's `FindOrCreateGoogleUser` should check if the email exists with a different provider:

```go
func (m *mockUserRepo) FindOrCreateGoogleUser(googleID, email string) (*models.User, error) {
	// Check for existing Google user
	for _, u := range m.users {
		if u.AuthProvider == "google" && u.AuthProviderUserID != nil && *u.AuthProviderUserID == googleID {
			return u, nil
		}
	}
	// Check for email conflict with different provider
	if existing, ok := m.users[email]; ok && existing.AuthProvider != "google" {
		return nil, repositories.ErrEmailTakenByOtherProvider
	}
	// Create new Google user
	googleIDCopy := googleID
	user := &models.User{
		ID:                 "google-user-" + googleID,
		Email:              email,
		AuthProvider:       "google",
		AuthProviderUserID: &googleIDCopy,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	m.users[email] = user
	return user, nil
}
```

Also add `"github.com/vladyslavivchenko/netme/internal/repositories"` and `"context"` imports to `auth_test.go`.

- [ ] **Step 7: Update `app.go` to wire `GoogleIDTokenVerifier` and update `NewAuthHandler` call**

In `netme-backend/internal/app/app.go`:

```go
googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
googleVerifier := services.NewGoogleIDTokenVerifier(googleClientID)

authHandler := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc, googleVerifier)
```

- [ ] **Step 8: Update `.env.example`**

Add to `netme-backend/.env.example`:

```
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
```

- [ ] **Step 9: Run tests**

```bash
cd netme-backend && go test ./... -v -run TestGoogle
```

Expected: `TestGoogleAuthSuccess`, `TestGoogleAuthEmailConflict`, `TestGoogleAuthInvalidToken` all PASS.

- [ ] **Step 10: Run full test suite**

```bash
cd netme-backend && go test ./... -v
```

Expected: all tests PASS (13 existing + 3 new = 16 total).

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "fix: replace Google access token with ID token verification; inject GoogleVerifier; map email conflict to 409"
```

---

### Task 3: Rate limiting middleware

**Files:**
- Create: `netme-backend/internal/middleware/ratelimit.go`
- Modify: `netme-backend/internal/app/app.go`
- Modify: `netme-backend/go.mod`

**Interfaces:**
- Produces: `middleware.RateLimiter(r rate.Limit, b int) gin.HandlerFunc`

- [ ] **Step 1: Add `golang.org/x/time` dependency**

```bash
cd netme-backend && go get golang.org/x/time/rate
```

- [ ] **Step 2: Write failing rate limiter test**

Create `netme-backend/internal/middleware/ratelimit_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/middleware"
	"golang.org/x/time/rate"
)

func TestRateLimiterBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Burst of 3 — 4th request from same IP must get 429
	r.POST("/test", middleware.RateLimiter(rate.Limit(100), 3), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 1; i <= 4; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "1.2.3.4:9999"
		r.ServeHTTP(w, req)

		if i <= 3 && w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
		if i == 4 && w.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: expected 429, got %d", i, w.Code)
		}
	}
}

func TestRateLimiterDifferentIPsNotBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/test", middleware.RateLimiter(rate.Limit(100), 1), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i, ip := range []string{"1.2.3.4:0", "5.6.7.8:0", "9.10.11.12:0"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = ip
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d from %s: expected 200, got %d", i+1, ip, w.Code)
		}
	}
}
```

- [ ] **Step 3: Run tests to confirm they fail**

```bash
cd netme-backend && go test ./internal/middleware/... -v
```

Expected: compilation error (`middleware.RateLimiter` not defined).

- [ ] **Step 4: Implement `ratelimit.go`**

Create `netme-backend/internal/middleware/ratelimit.go`:

```go
package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"golang.org/x/time/rate"
)

type ipLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func newIPLimiterStore(r rate.Limit, b int) *ipLimiterStore {
	return &ipLimiterStore{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (s *ipLimiterStore) allow(ip string) bool {
	s.mu.Lock()
	l, ok := s.limiters[ip]
	if !ok {
		l = rate.NewLimiter(s.r, s.b)
		s.limiters[ip] = l
	}
	s.mu.Unlock()
	return l.Allow()
}

// RateLimiter returns a per-IP token-bucket rate limiting middleware.
// r is the sustained rate (events per second); b is the burst size.
func RateLimiter(r rate.Limit, b int) gin.HandlerFunc {
	store := newIPLimiterStore(r, b)
	return func(c *gin.Context) {
		if !store.allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many attempts. Please try again later.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 5: Run tests**

```bash
cd netme-backend && go test ./internal/middleware/... -v
```

Expected: `TestRateLimiterBlocks` and `TestRateLimiterDifferentIPsNotBlocked` both PASS.

- [ ] **Step 6: Wire rate limiting in `app.go`**

Add `"time"` and `"golang.org/x/time/rate"` to imports in `netme-backend/internal/app/app.go`.

Replace the auth route registrations with per-route rate limiters:

```go
auth := v1.Group("/auth")
{
	auth.POST("/register", middleware.RateLimiter(rate.Every(12*time.Second), 5), authHandler.Register)
	auth.POST("/login",    middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.Login)
	auth.POST("/refresh",  middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.Refresh)
	auth.POST("/google",   middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.GoogleAuth)
}
```

`rate.Every(12*time.Second)` = 5 req/min. `rate.Every(6*time.Second)` = 10 req/min.

- [ ] **Step 7: Verify full build**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 8: Commit**

```bash
git add internal/middleware/ratelimit.go internal/middleware/ratelimit_test.go internal/app/app.go go.mod go.sum
git commit -m "feat: add per-IP rate limiting on auth endpoints (login/refresh 10/min, register 5/min)"
```

---

### Task 4: Refresh token rotation

**Files:**
- Modify: `netme-backend/internal/handlers/auth.go` — update `Refresh` method
- Modify: `netme-backend/internal/handlers/auth_test.go` — add rotation test

**Interfaces:**
- Consumes: `TokenRepo.CreateRefreshToken`, `TokenRepo.RevokeRefreshToken(token, userID string)` — already in interface

- [ ] **Step 1: Write failing rotation test**

Add to `netme-backend/internal/handlers/auth_test.go`:

```go
func TestRefreshRotatesToken(t *testing.T) {
	r, userRepo, tokenRepo := newTestAuthRouter()

	// Register to get initial tokens
	regW := httptest.NewRecorder()
	regReq, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "rotate@example.com", "password": "password123"}))
	regReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(regW, regReq)
	if regW.Code != http.StatusCreated {
		t.Fatalf("register failed: %d %s", regW.Code, regW.Body.String())
	}

	var regResp models.AuthResponse
	json.Unmarshal(regW.Body.Bytes(), &regResp)
	originalRefreshToken := regResp.RefreshToken

	// Pre-populate userRepo so Refresh handler can find the user
	_ = userRepo
	_ = tokenRepo

	// Call refresh
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/refresh",
		jsonBody(t, map[string]string{"refresh_token": originalRefreshToken}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var refreshResp models.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &refreshResp)

	if refreshResp.RefreshToken == originalRefreshToken {
		t.Error("expected rotated refresh token, got same token back")
	}
	if refreshResp.RefreshToken == "" {
		t.Error("expected non-empty rotated refresh token")
	}

	// Original token must be revoked
	rt, _ := tokenRepo.GetRefreshToken(originalRefreshToken)
	if rt.RevokedAt == nil {
		t.Error("expected original refresh token to be revoked after rotation")
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd netme-backend && go test ./internal/handlers/... -v -run TestRefreshRotates
```

Expected: FAIL — `refreshResp.RefreshToken == originalRefreshToken` (not rotated yet).

- [ ] **Step 3: Update `Refresh` handler in `auth.go`**

Replace the existing `Refresh` method with:

```go
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	valid, err := h.tokenRepo.IsRefreshTokenValid(req.RefreshToken)
	if !valid || err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token is invalid or expired",
		})
		return
	}

	refreshTokenRecord, err := h.tokenRepo.GetRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token not found",
		})
		return
	}

	user, err := h.userRepo.GetUserByID(refreshTokenRecord.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "user_not_found",
			Message: "User not found",
		})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate token"})
		return
	}

	// Rotate: create new refresh token before revoking old one
	newRefreshTokenString, err := h.jwtService.GenerateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to generate refresh token"})
		return
	}

	newRefreshToken, err := h.tokenRepo.CreateRefreshToken(user.ID, newRefreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to store refresh token"})
		return
	}

	// Revoke old token — log failure but don't block (old token expires naturally)
	if err := h.tokenRepo.RevokeRefreshToken(req.RefreshToken, user.ID); err != nil {
		slog.Warn("failed to revoke old refresh token during rotation", "user_id", user.ID, "error", err)
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken.Token,
		ExpiresIn:    900,
		User:         user,
	})
}
```

- [ ] **Step 4: Run all tests**

```bash
cd netme-backend && go test ./... -v
```

Expected: all tests PASS including `TestRefreshRotatesToken`.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/auth.go internal/handlers/auth_test.go
git commit -m "fix: rotate refresh tokens on each use — issue new token, revoke old"
```

---

### Task 5: Mobile — rename `id_token` field + persist rotated refresh token

**Files:**
- Modify: `netme-mobile/src/services/authService.ts`
- Modify: `netme-mobile/src/context/AuthContext.tsx`

**Interfaces:**
- Consumes: backend now expects `id_token` field (not `access_token`) in `POST /v1/auth/google`
- Consumes: `POST /v1/auth/refresh` now returns a new `refresh_token` on every call

- [ ] **Step 1: Rename field in `authService.ts`**

In `netme-mobile/src/services/authService.ts`, update `loginWithGoogle`:

```typescript
async loginWithGoogle(googleIDToken: string): Promise<AuthResponse> {
  const response = await this.api.post<AuthResponse>('/auth/google', {
    id_token: googleIDToken,
  });
  return response.data;
}
```

The parameter is renamed from `googleAccessToken` to `googleIDToken` to make clear it must be an ID token, not an access token.

- [ ] **Step 2: Persist rotated refresh token in `AuthContext.tsx`**

In `netme-mobile/src/context/AuthContext.tsx`, update `refreshAccessToken`:

```typescript
const refreshAccessToken = async (): Promise<boolean> => {
  try {
    if (!refreshToken) {
      return false;
    }

    const response = await authService.refresh(refreshToken);

    setAccessToken(response.access_token);
    setRefreshToken(response.refresh_token);  // persist rotated token
    setUser(response.user);

    await secureStorage.saveAccessToken(response.access_token);
    await secureStorage.saveRefreshToken(response.refresh_token);  // persist rotated token
    await secureStorage.saveUser(JSON.stringify(response.user));

    return true;
  } catch (error) {
    console.error('Token refresh failed:', error);
    await clearAuth();
    return false;
  }
};
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd netme-mobile && npx tsc --noEmit 2>&1
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/services/authService.ts src/context/AuthContext.tsx
git commit -m "fix: rename Google token field to id_token; persist rotated refresh token"
```
