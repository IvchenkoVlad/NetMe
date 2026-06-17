# NetMe Foundation Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove dead code, fix critical bugs (CORS, JWT secret, API path, unprotected routes), add goose migrations, structured logging, repository interfaces, and auth test coverage.

**Architecture:** Single-pass backend cleanup. Handler→service→repository layers preserved. AuthRepository split into UserRepository + TokenRepository, each backed by an interface so handlers are mockable in tests. JWTService becomes injectable via constructor parameter. Goose replaces the custom migration loader.

**Tech Stack:** Go 1.22, Gin, PostgreSQL (lib/pq), goose v3, log/slog (stdlib), golang-jwt/v5, bcrypt

## Global Constraints

- Base API path: `/v1` (not `/api/v1`)
- All handlers use struct pattern with constructor injection
- No Redis dependency
- `JWT_SECRET_KEY` required at startup — fatal if missing
- `Access-Control-Allow-Origin` must never be `*` when credentials are used
- No sensitive data (tokens, passwords) in logs
- Working directory for all Go commands: `netme-backend/`

---

## File Map

**Create:**
- `internal/logger/logger.go` — `New() *slog.Logger`, JSON in prod, text otherwise
- `internal/repositories/interfaces.go` — `UserRepo` and `TokenRepo` interfaces
- `internal/repositories/user.go` — `UserRepository` struct implementing `UserRepo`
- `internal/repositories/token.go` — `TokenRepository` struct implementing `TokenRepo`
- `internal/handlers/users.go` — `UsersHandler` with `GetMe` and `DeleteMe`
- `internal/db/migrations/0001_create_users.sql`
- `internal/db/migrations/0002_create_refresh_tokens.sql`
- `internal/services/jwt_test.go`
- `internal/services/password_test.go`
- `internal/handlers/auth_test.go`

**Modify:**
- `internal/models/auth.go` — remove social fields, add `auth_provider`/`auth_provider_user_id`
- `internal/services/jwt.go` — accept secret as parameter, remove unused method
- `internal/middleware/middleware.go` — fix CORS, accept `*JWTService` in `AuthMiddleware`
- `internal/handlers/auth.go` — use interfaces, clean up `Logout`, remove `LogoutAllDevices`, remove `RegisterAuthRoutes`
- `internal/handlers/accounts.go` — struct pattern
- `internal/handlers/transactions.go` — struct pattern
- `internal/handlers/health.go` — remove hello endpoint
- `internal/app/app.go` — remove Redis, fix path, wire JWT secret, public/protected groups
- `cmd/migrate/main.go` — goose wrapper
- `go.mod` — add goose, remove redis
- `.env.example` — add `CORS_ALLOWED_ORIGINS`, remove `REDIS_URL`

**Delete:**
- `netme-mobile/app/` (entire folder)
- `docs/docs/` (entire folder)
- `internal/handlers/analytics.go`
- `internal/db/migrations.go`
- `internal/db/migrations/tables/` and `indices/` dirs
- `internal/repositories/auth.go`

---

### Task 1: Delete clutter

**Files:**
- Delete: `netme-mobile/app/`, `docs/docs/`, `internal/handlers/analytics.go`, `internal/db/migrations.go`, `internal/db/migrations/tables/`, `internal/db/migrations/indices/`
- Modify: `internal/handlers/health.go`, `internal/handlers/auth.go`, `internal/app/app.go`

**Interfaces:**
- Produces: clean file tree with no dead code, analytics routes unregistered

- [ ] **Step 1: Delete dead directories and files**

```bash
rm -rf netme-mobile/app
rm -rf docs/docs
rm netme-backend/internal/handlers/analytics.go
rm netme-backend/internal/db/migrations.go
rm -rf netme-backend/internal/db/migrations/tables
rm -rf netme-backend/internal/db/migrations/indices
```

- [ ] **Step 2: Rewrite health.go — remove hello endpoint**

Full new content of `netme-backend/internal/handlers/health.go`:

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
```

- [ ] **Step 3: Remove LogoutAllDevices from auth.go**

Delete these lines from `netme-backend/internal/handlers/auth.go`:

```go
func (h *AuthHandler) LogoutAllDevices(c *gin.Context, userID string) error {
	return h.authRepo.RevokeAllUserTokens(userID)
}
```

- [ ] **Step 4: Remove analytics route registration from app.go**

In `netme-backend/internal/app/app.go`, remove this line:

```go
handlers.RegisterAnalyticsRoutes(api, database)
```

Also remove the `"github.com/redis/go-redis/v9"` import from app.go and the redis client construction:

```go
// Remove these lines:
redisClient := redis.NewClient(&redis.Options{
    Addr: os.Getenv("REDIS_URL"),
})
```

And remove `redis *redis.Client` from the `App` struct and `a.redis.Close()` from `Close()`.

- [ ] **Step 5: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors. (Redis import will cause an error if not fully removed — fix any remaining redis references.)

- [ ] **Step 6: Run go mod tidy to drop redis**

```bash
cd netme-backend && go mod tidy
```

Expected: `github.com/redis/go-redis/v9` removed from `go.mod` and `go.sum`.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: remove dead code, analytics stub, legacy mobile folder, and Redis"
```

---

### Task 2: Fix User model

**Files:**
- Modify: `internal/models/auth.go`

**Interfaces:**
- Produces: `models.User` with fields `ID, Email, PasswordHash, AuthProvider, AuthProviderUserID, CreatedAt, UpdatedAt`

- [ ] **Step 1: Rewrite models/auth.go**

Full new content of `netme-backend/internal/models/auth.go`:

```go
package models

import "time"

type User struct {
	ID                 string    `json:"id"`
	Email              string    `json:"email"`
	PasswordHash       string    `json:"-"`
	AuthProvider       string    `json:"auth_provider"`
	AuthProviderUserID *string   `json:"auth_provider_user_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         *User  `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
```

- [ ] **Step 2: Update SQL queries in repositories/auth.go**

The existing `auth.go` in repositories references old fields. Update all three SELECT statements to use the new columns. Replace the full file content:

```go
package repositories

import (
	"database/sql"
	"errors"
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUser(email, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`INSERT INTO users (email, password_hash, auth_provider, created_at, updated_at)
		 VALUES ($1, $2, 'local', now(), now())
		 RETURNING id, email, auth_provider, auth_provider_user_id, created_at, updated_at`,
		email, passwordHash,
	).Scan(
		&user.ID, &user.Email, &user.AuthProvider,
		&user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = passwordHash
	return user, nil
}

func (r *AuthRepository) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.AuthProvider, &user.AuthProviderUserID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) GetUserByID(userID string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE id = $1`,
		userID,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.AuthProvider, &user.AuthProviderUserID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) UpdateLastLogin(userID string) error {
	_, err := r.db.Exec(
		`UPDATE users SET updated_at = now() WHERE id = $1`,
		userID,
	)
	return err
}

func (r *AuthRepository) CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{}
	err := r.db.QueryRow(
		`INSERT INTO refresh_tokens (user_id, token, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, now(), now())
		 RETURNING id, user_id, token, expires_at, revoked_at, created_at, updated_at`,
		userID, token, expiresAt,
	).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
		&rt.RevokedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (r *AuthRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{}
	err := r.db.QueryRow(
		`SELECT id, user_id, token, expires_at, revoked_at, created_at, updated_at
		 FROM refresh_tokens WHERE token = $1`,
		token,
	).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
		&rt.RevokedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("refresh token not found")
		}
		return nil, err
	}
	return rt, nil
}

func (r *AuthRepository) RevokeRefreshToken(token string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE token = $1`,
		token,
	)
	return err
}

func (r *AuthRepository) RevokeAllUserTokens(userID string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE user_id = $1`,
		userID,
	)
	return err
}

func (r *AuthRepository) IsRefreshTokenValid(token string) (bool, error) {
	rt, err := r.GetRefreshToken(token)
	if err != nil {
		return false, err
	}
	if rt.RevokedAt != nil {
		return false, errors.New("token is revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return false, errors.New("token is expired")
	}
	return true, nil
}
```

- [ ] **Step 3: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/models/auth.go internal/repositories/auth.go
git commit -m "refactor: align User model to spec — remove social fields, add auth_provider"
```

---

### Task 3: Fix infrastructure bugs + injectable JWTService

**Files:**
- Modify: `internal/services/jwt.go`
- Modify: `internal/middleware/middleware.go`
- Modify: `internal/app/app.go`
- Modify: `internal/handlers/auth.go`
- Modify: `.env.example`

**Interfaces:**
- Produces: `services.NewJWTService(secretKey string) *JWTService`
- Produces: `middleware.AuthMiddleware(jwtSvc *services.JWTService) gin.HandlerFunc`
- Produces: `middleware.CORSMiddleware() gin.HandlerFunc` (reads `CORS_ALLOWED_ORIGINS`)
- Produces: `handlers.NewAuthHandler(db *sql.DB, jwtSvc *services.JWTService) *AuthHandler`

- [ ] **Step 1: Rewrite services/jwt.go — injectable secret, remove unused method**

Full new content of `netme-backend/internal/services/jwt.go`:

```go
package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey string
}

func NewJWTService(secretKey string) *JWTService {
	return &JWTService{secretKey: secretKey}
}

func (j *JWTService) GenerateAccessToken(userID, email string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

func (j *JWTService) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (j *JWTService) VerifyAccessToken(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 2: Rewrite middleware/middleware.go — fix CORS, accept JWTService**

Full new content of `netme-backend/internal/middleware/middleware.go`:

```go
package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/services"
)

func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:8081"
	}
	originSet := make(map[string]bool)
	for _, o := range strings.Split(allowedOrigins, ",") {
		originSet[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if originSet[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func AuthMiddleware(jwtService *services.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "missing_token",
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_format",
				Message: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		claims, err := jwtService.VerifyAccessToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}
```

- [ ] **Step 3: Rewrite app.go — JWT secret fatal, /v1 path, public/protected groups**

Full new content of `netme-backend/internal/app/app.go`:

```go
package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/db"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/middleware"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type App struct {
	db     *sql.DB
	router *gin.Engine
	log    *slog.Logger
}

func New() (*App, error) {
	log := newLogger()

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Error("JWT_SECRET_KEY environment variable is required")
		os.Exit(1)
	}

	database, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	log.Info("database connected")

	jwtSvc := services.NewJWTService(jwtSecret)
	authHandler := handlers.NewAuthHandler(database, jwtSvc)

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())
	router.GET("/healthz", handlers.HealthHandler())

	v1 := router.Group("/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
	}

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(jwtSvc))
	{
		protected.POST("/auth/logout", authHandler.Logout)
		handlers.RegisterAccountRoutes(protected, database)
		handlers.RegisterTransactionRoutes(protected, database)
	}

	return &App{
		db:     database,
		router: router,
		log:    log,
	}, nil
}

func (a *App) Start() error {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	a.log.Info("starting server", "port", port)
	return a.router.Run(":" + port)
}

func (a *App) Close() error {
	return a.db.Close()
}

func newLogger() *slog.Logger {
	if os.Getenv("API_ENV") == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
```

Note: `GET /v1/me` and `DELETE /v1/me` are added in Task 5 after UsersHandler is created.

- [ ] **Step 4: Update auth handler — use JWTService parameter, clean up Logout**

Replace `AuthHandler` struct and constructor in `netme-backend/internal/handlers/auth.go`. Also remove `RegisterAuthRoutes` function (routes are now in app.go), and clean up `Logout` to remove the manual header parsing (auth is now handled by middleware):

```go
package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type AuthHandler struct {
	authRepo        *repositories.AuthRepository
	jwtService      *services.JWTService
	passwordService *services.PasswordService
}

func NewAuthHandler(db *sql.DB, jwtSvc *services.JWTService) *AuthHandler {
	return &AuthHandler{
		authRepo:        repositories.NewAuthRepository(db),
		jwtService:      jwtSvc,
		passwordService: services.NewPasswordService(),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	existingUser, _ := h.authRepo.GetUserByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "user_exists",
			Message: "User with this email already exists",
		})
		return
	}

	passwordHash, err := h.passwordService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "hash_error",
			Message: "Failed to hash password",
		})
		return
	}

	user, err := h.authRepo.CreateUser(req.Email, passwordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "creation_error",
			Message: "Failed to create user",
		})
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

	refreshToken, err := h.authRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to store refresh token"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusCreated, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900,
		User:         user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, err := h.authRepo.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or password",
		})
		return
	}

	if user.PasswordHash == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_credentials",
			Message: "No password set for this account",
		})
		return
	}

	if err := h.passwordService.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or password",
		})
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

	refreshToken, err := h.authRepo.CreateRefreshToken(user.ID, refreshTokenString, time.Now().Add(7*24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_error", Message: "Failed to store refresh token"})
		return
	}

	h.authRepo.UpdateLastLogin(user.ID)

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresIn:    900,
		User:         user,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	valid, err := h.authRepo.IsRefreshTokenValid(req.RefreshToken)
	if !valid || err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token is invalid or expired",
		})
		return
	}

	refreshTokenRecord, err := h.authRepo.GetRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_token",
			Message: "Refresh token not found",
		})
		return
	}

	user, err := h.authRepo.GetUserByID(refreshTokenRecord.UserID)
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

	user.PasswordHash = ""
	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
		ExpiresIn:    900,
		User:         user,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.authRepo.RevokeRefreshToken(req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "revoke_error",
			Message: "Failed to revoke token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
```

- [ ] **Step 5: Update .env.example**

Full new content of `netme-backend/.env.example`:

```
DATABASE_URL=postgres://netme:devpassword@localhost:5432/netme_dev?sslmode=disable
API_PORT=8080
API_ENV=development
JWT_SECRET_KEY=change-this-to-a-long-random-secret-in-production
CORS_ALLOWED_ORIGINS=http://localhost:8081
MOBILE_API_URL=http://localhost:8080
```

- [ ] **Step 6: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "fix: injectable JWTService, CORS origin allowlist, /v1 path, JWT secret fatal"
```

---

### Task 4: Add goose migrations

**Files:**
- Create: `internal/db/migrations/0001_create_users.sql`
- Create: `internal/db/migrations/0002_create_refresh_tokens.sql`
- Modify: `cmd/migrate/main.go`
- Modify: `go.mod` (add goose)

**Interfaces:**
- Produces: `go run ./cmd/migrate up` creates `users` and `refresh_tokens` tables

- [ ] **Step 1: Add goose dependency**

```bash
cd netme-backend && go get github.com/pressly/goose/v3
```

- [ ] **Step 2: Create users migration**

Create `netme-backend/internal/db/migrations/0001_create_users.sql`:

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

- [ ] **Step 3: Create refresh_tokens migration**

Create `netme-backend/internal/db/migrations/0002_create_refresh_tokens.sql`:

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
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);

-- +goose Down
DROP TABLE refresh_tokens;
```

- [ ] **Step 4: Rewrite cmd/migrate/main.go**

Full new content of `netme-backend/cmd/migrate/main.go`:

```go
package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func main() {
	godotenv.Load()

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"up"}
	}

	if err := goose.Run(args[0], db, "internal/db/migrations", args[1:]...); err != nil {
		log.Fatalf("goose %s: %v", args[0], err)
	}
}
```

- [ ] **Step 5: Start Docker DB**

```bash
cd /Users/vivchenko/vs_code/NetMe && docker-compose up -d
```

Wait for healthy status:

```bash
docker-compose ps
```

Expected: `db` service shows `healthy`.

- [ ] **Step 6: Run migrations**

```bash
cd netme-backend && go run ./cmd/migrate up
```

Expected output:
```
goose: successfully migrated database to version: 2
```

- [ ] **Step 7: Verify tables exist**

```bash
docker exec netme-db psql -U netme -d netme_dev -c "\dt"
```

Expected: `goose_db_version`, `refresh_tokens`, `users` listed.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat: add goose migrations for users and refresh_tokens tables"
```

---

### Task 5: Add users handler and wire /v1/me

**Files:**
- Create: `internal/handlers/users.go`
- Modify: `internal/app/app.go` (register users routes in protected group)

**Interfaces:**
- Consumes: `repositories.AuthRepository.GetUserByID(id string) (*models.User, error)` from Task 2
- Produces: `GET /v1/me` → `200 models.User` (PasswordHash cleared)
- Produces: `DELETE /v1/me` → `501 ErrorResponse`

- [ ] **Step 1: Create users.go handler**

Create `netme-backend/internal/handlers/users.go`:

```go
package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type UsersHandler struct {
	authRepo *repositories.AuthRepository
}

func NewUsersHandler(db *sql.DB) *UsersHandler {
	return &UsersHandler{authRepo: repositories.NewAuthRepository(db)}
}

func (h *UsersHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.authRepo.GetUserByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "user_not_found",
			Message: "User not found",
		})
		return
	}
	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) DeleteMe(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Account deletion is not yet available",
	})
}
```

- [ ] **Step 2: Add users routes to protected group in app.go**

In `netme-backend/internal/app/app.go`, add after `authHandler` construction:

```go
usersHandler := handlers.NewUsersHandler(database)
```

And add to the protected group block:

```go
protected.GET("/me", usersHandler.GetMe)
protected.DELETE("/me", usersHandler.DeleteMe)
```

The full protected block becomes:

```go
protected := v1.Group("")
protected.Use(middleware.AuthMiddleware(jwtSvc))
{
    protected.POST("/auth/logout", authHandler.Logout)
    protected.GET("/me", usersHandler.GetMe)
    protected.DELETE("/me", usersHandler.DeleteMe)
    handlers.RegisterAccountRoutes(protected, database)
    handlers.RegisterTransactionRoutes(protected, database)
}
```

- [ ] **Step 3: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Smoke test /v1/me**

Start the server (ensure DB is running):

```bash
cd netme-backend && JWT_SECRET_KEY=testsecret DATABASE_URL=postgres://netme:devpassword@localhost:5432/netme_dev?sslmode=disable go run ./cmd/server
```

Register a user:

```bash
curl -s -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' | jq .
```

Expected: `201` with `access_token`, `refresh_token`, `user`.

Call GET /v1/me with the token:

```bash
curl -s http://localhost:8080/v1/me \
  -H "Authorization: Bearer <access_token>" | jq .
```

Expected: `200` with user object, no `password_hash` field.

Call without token:

```bash
curl -s http://localhost:8080/v1/me | jq .
```

Expected: `401 missing_token`.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/users.go internal/app/app.go
git commit -m "feat: add GET /v1/me and DELETE /v1/me (501) endpoints"
```

---

### Task 6: Split repositories and add interfaces

**Files:**
- Create: `internal/repositories/interfaces.go`
- Create: `internal/repositories/user.go`
- Create: `internal/repositories/token.go`
- Delete: `internal/repositories/auth.go`
- Modify: `internal/handlers/auth.go` (use interfaces)
- Modify: `internal/handlers/users.go` (use UserRepo interface)
- Modify: `internal/app/app.go` (pass repos to handlers)

**Interfaces:**
- Produces: `repositories.UserRepo` interface
- Produces: `repositories.TokenRepo` interface
- Produces: `repositories.NewUserRepository(db *sql.DB) *UserRepository`
- Produces: `repositories.NewTokenRepository(db *sql.DB) *TokenRepository`

- [ ] **Step 1: Create repositories/interfaces.go**

Create `netme-backend/internal/repositories/interfaces.go`:

```go
package repositories

import (
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type UserRepo interface {
	CreateUser(email, passwordHash string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateLastLogin(userID string) error
}

type TokenRepo interface {
	CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error)
	GetRefreshToken(token string) (*models.RefreshToken, error)
	RevokeRefreshToken(token string) error
	RevokeAllUserTokens(userID string) error
	IsRefreshTokenValid(token string) (bool, error)
}
```

- [ ] **Step 2: Create repositories/user.go**

Create `netme-backend/internal/repositories/user.go`:

```go
package repositories

import (
	"database/sql"
	"errors"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(email, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`INSERT INTO users (email, password_hash, auth_provider, created_at, updated_at)
		 VALUES ($1, $2, 'local', now(), now())
		 RETURNING id, email, auth_provider, auth_provider_user_id, created_at, updated_at`,
		email, passwordHash,
	).Scan(
		&user.ID, &user.Email, &user.AuthProvider,
		&user.AuthProviderUserID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = passwordHash
	return user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.AuthProvider, &user.AuthProviderUserID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetUserByID(userID string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		`SELECT id, email, password_hash, auth_provider, auth_provider_user_id, created_at, updated_at
		 FROM users WHERE id = $1`,
		userID,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.AuthProvider, &user.AuthProviderUserID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateLastLogin(userID string) error {
	_, err := r.db.Exec(
		`UPDATE users SET updated_at = now() WHERE id = $1`,
		userID,
	)
	return err
}
```

- [ ] **Step 3: Create repositories/token.go**

Create `netme-backend/internal/repositories/token.go`:

```go
package repositories

import (
	"database/sql"
	"errors"
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type TokenRepository struct {
	db *sql.DB
}

func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{}
	err := r.db.QueryRow(
		`INSERT INTO refresh_tokens (user_id, token, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, now(), now())
		 RETURNING id, user_id, token, expires_at, revoked_at, created_at, updated_at`,
		userID, token, expiresAt,
	).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
		&rt.RevokedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (r *TokenRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{}
	err := r.db.QueryRow(
		`SELECT id, user_id, token, expires_at, revoked_at, created_at, updated_at
		 FROM refresh_tokens WHERE token = $1`,
		token,
	).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
		&rt.RevokedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("refresh token not found")
		}
		return nil, err
	}
	return rt, nil
}

func (r *TokenRepository) RevokeRefreshToken(token string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE token = $1`,
		token,
	)
	return err
}

func (r *TokenRepository) RevokeAllUserTokens(userID string) error {
	_, err := r.db.Exec(
		`UPDATE refresh_tokens SET revoked_at = now(), updated_at = now() WHERE user_id = $1`,
		userID,
	)
	return err
}

func (r *TokenRepository) IsRefreshTokenValid(token string) (bool, error) {
	rt, err := r.GetRefreshToken(token)
	if err != nil {
		return false, err
	}
	if rt.RevokedAt != nil {
		return false, errors.New("token is revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return false, errors.New("token is expired")
	}
	return true, nil
}
```

- [ ] **Step 4: Update AuthHandler to use interfaces**

Replace the `AuthHandler` struct, constructor, and update all method calls in `netme-backend/internal/handlers/auth.go`:

```go
type AuthHandler struct {
	userRepo        repositories.UserRepo
	tokenRepo       repositories.TokenRepo
	jwtService      *services.JWTService
	passwordService *services.PasswordService
}

func NewAuthHandler(userRepo repositories.UserRepo, tokenRepo repositories.TokenRepo, jwtSvc *services.JWTService) *AuthHandler {
	return &AuthHandler{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		jwtService:      jwtSvc,
		passwordService: services.NewPasswordService(),
	}
}
```

Update all method bodies replacing `h.authRepo` calls:
- User operations (`GetUserByEmail`, `CreateUser`, `GetUserByID`, `UpdateLastLogin`) → `h.userRepo`
- Token operations (`CreateRefreshToken`, `IsRefreshTokenValid`, `GetRefreshToken`, `RevokeRefreshToken`) → `h.tokenRepo`

Remove `"database/sql"` import — no longer needed in auth.go.

- [ ] **Step 5: Update UsersHandler to use UserRepo interface**

Replace `UsersHandler` in `netme-backend/internal/handlers/users.go`:

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type UsersHandler struct {
	userRepo repositories.UserRepo
}

func NewUsersHandler(userRepo repositories.UserRepo) *UsersHandler {
	return &UsersHandler{userRepo: userRepo}
}

func (h *UsersHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.userRepo.GetUserByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "user_not_found",
			Message: "User not found",
		})
		return
	}
	user.PasswordHash = ""
	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) DeleteMe(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Account deletion is not yet available",
	})
}
```

- [ ] **Step 6: Update app.go to construct repos and pass to handlers**

In `netme-backend/internal/app/app.go`, replace the handler construction block:

```go
userRepo := repositories.NewUserRepository(database)
tokenRepo := repositories.NewTokenRepository(database)
jwtSvc := services.NewJWTService(jwtSecret)

authHandler := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc)
usersHandler := handlers.NewUsersHandler(userRepo)
```

Add import: `"github.com/vladyslavivchenko/netme/internal/repositories"`

- [ ] **Step 7: Delete old auth repository**

```bash
rm netme-backend/internal/repositories/auth.go
```

- [ ] **Step 8: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "refactor: split AuthRepository into UserRepository + TokenRepository with interfaces"
```

---

### Task 7: Convert stub handlers to struct pattern

**Files:**
- Modify: `internal/handlers/accounts.go`
- Modify: `internal/handlers/transactions.go`

**Interfaces:**
- Produces: `handlers.NewAccountsHandler(db *sql.DB) *AccountsHandler` with `RegisterRoutes(*gin.RouterGroup)`
- Produces: `handlers.NewTransactionsHandler(db *sql.DB) *TransactionsHandler` with `RegisterRoutes(*gin.RouterGroup)`

- [ ] **Step 1: Rewrite accounts.go**

Full new content of `netme-backend/internal/handlers/accounts.go`:

```go
package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type AccountsHandler struct {
	db *sql.DB
}

func NewAccountsHandler(db *sql.DB) *AccountsHandler {
	return &AccountsHandler{db: db}
}

func RegisterAccountRoutes(r *gin.RouterGroup, db *sql.DB) {
	NewAccountsHandler(db).RegisterRoutes(r)
}

func (h *AccountsHandler) RegisterRoutes(r *gin.RouterGroup) {
	accounts := r.Group("/accounts")
	{
		accounts.GET("", h.ListAccounts)
		accounts.GET("/:id", h.GetAccount)
	}
}

func (h *AccountsHandler) ListAccounts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Accounts endpoint not yet implemented",
	})
}

func (h *AccountsHandler) GetAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Accounts endpoint not yet implemented",
	})
}
```

- [ ] **Step 2: Rewrite transactions.go**

Full new content of `netme-backend/internal/handlers/transactions.go`:

```go
package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type TransactionsHandler struct {
	db *sql.DB
}

func NewTransactionsHandler(db *sql.DB) *TransactionsHandler {
	return &TransactionsHandler{db: db}
}

func RegisterTransactionRoutes(r *gin.RouterGroup, db *sql.DB) {
	NewTransactionsHandler(db).RegisterRoutes(r)
}

func (h *TransactionsHandler) RegisterRoutes(r *gin.RouterGroup) {
	txns := r.Group("/transactions")
	{
		txns.GET("", h.ListTransactions)
		txns.GET("/:id", h.GetTransaction)
	}
}

func (h *TransactionsHandler) ListTransactions(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Transactions endpoint not yet implemented",
	})
}

func (h *TransactionsHandler) GetTransaction(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error:   "not_implemented",
		Message: "Transactions endpoint not yet implemented",
	})
}
```

- [ ] **Step 3: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/accounts.go internal/handlers/transactions.go
git commit -m "refactor: convert accounts and transactions handlers to struct pattern"
```

---

### Task 8: JWT service unit tests

**Files:**
- Create: `internal/services/jwt_test.go`

**Interfaces:**
- Consumes: `services.NewJWTService(secretKey string) *JWTService` from Task 3
- Consumes: `services.JWTClaims` (exported struct from jwt.go)

- [ ] **Step 1: Create jwt_test.go**

Create `netme-backend/internal/services/jwt_test.go`:

```go
package services_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vladyslavivchenko/netme/internal/services"
)

const testSecret = "test-secret-key-32-chars-minimum!!"

func TestGenerateAndVerifyAccessToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	token, err := svc.GenerateAccessToken("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	claims, err := svc.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("expected no error verifying token, got %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got %q", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got %q", claims.Email)
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	claims := services.JWTClaims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	_, err := svc.VerifyAccessToken(tokenString)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestVerifyTamperedToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	token, _ := svc.GenerateAccessToken("user-123", "test@example.com")
	_, err := svc.VerifyAccessToken(token + "tampered")
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestVerifyTokenWrongSecret(t *testing.T) {
	svc1 := services.NewJWTService(testSecret)
	svc2 := services.NewJWTService("different-secret-key-32-chars!!!")

	token, _ := svc1.GenerateAccessToken("user-123", "test@example.com")
	_, err := svc2.VerifyAccessToken(token)
	if err == nil {
		t.Fatal("expected error verifying token with wrong secret, got nil")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	svc := services.NewJWTService(testSecret)

	token1, err := svc.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	token2, _ := svc.GenerateRefreshToken()

	if len(token1) != 64 {
		t.Errorf("expected 64-char hex token, got length %d", len(token1))
	}
	if token1 == token2 {
		t.Error("expected unique refresh tokens, got identical tokens")
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd netme-backend && go test ./internal/services/... -v
```

Expected: all 5 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/services/jwt_test.go
git commit -m "test: add JWT service unit tests"
```

---

### Task 9: Password service unit tests

**Files:**
- Create: `internal/services/password_test.go`

**Interfaces:**
- Consumes: `services.NewPasswordService() *PasswordService` with `HashPassword(string) (string, error)` and `VerifyPassword(hash, password string) error`

- [ ] **Step 1: Create password_test.go**

Create `netme-backend/internal/services/password_test.go`:

```go
package services_test

import (
	"testing"

	"github.com/vladyslavivchenko/netme/internal/services"
)

func TestHashAndVerifyPassword(t *testing.T) {
	svc := services.NewPasswordService()

	hash, err := svc.HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "mypassword123" {
		t.Fatal("hash must not equal plaintext password")
	}

	if err := svc.VerifyPassword(hash, "mypassword123"); err != nil {
		t.Errorf("expected correct password to verify, got %v", err)
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	svc := services.NewPasswordService()

	hash, _ := svc.HashPassword("correctpassword")
	if err := svc.VerifyPassword(hash, "wrongpassword"); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestHashesAreUnique(t *testing.T) {
	svc := services.NewPasswordService()

	hash1, _ := svc.HashPassword("samepassword")
	hash2, _ := svc.HashPassword("samepassword")

	if hash1 == hash2 {
		t.Error("expected unique hashes for same password (bcrypt salt), got identical hashes")
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd netme-backend && go test ./internal/services/... -v
```

Expected: all 8 tests PASS (5 JWT + 3 password).

- [ ] **Step 3: Commit**

```bash
git add internal/services/password_test.go
git commit -m "test: add password service unit tests"
```

---

### Task 10: Auth handler tests

**Files:**
- Modify: `internal/handlers/auth.go` — update `NewAuthHandler` to accept interfaces (enables injection of mocks)
- Modify: `internal/app/app.go` — construct repos explicitly before passing to handler
- Create: `internal/handlers/auth_test.go`

**Interfaces:**
- Consumes: `repositories.UserRepo` and `repositories.TokenRepo` interfaces from Task 6
- Consumes: `services.NewJWTService(secretKey string)` from Task 3

- [ ] **Step 1: Update NewAuthHandler to accept interfaces**

In `netme-backend/internal/handlers/auth.go`, change the constructor signature. Remove `"database/sql"` import:

```go
func NewAuthHandler(userRepo repositories.UserRepo, tokenRepo repositories.TokenRepo, jwtSvc *services.JWTService) *AuthHandler {
	return &AuthHandler{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		jwtService:      jwtSvc,
		passwordService: services.NewPasswordService(),
	}
}
```

Update `AuthHandler` struct to use interfaces:

```go
type AuthHandler struct {
	userRepo        repositories.UserRepo
	tokenRepo       repositories.TokenRepo
	jwtService      *services.JWTService
	passwordService *services.PasswordService
}
```

Update all method bodies: replace all `h.authRepo.CreateUser`, `h.authRepo.GetUserByEmail`, `h.authRepo.GetUserByID`, `h.authRepo.UpdateLastLogin` with `h.userRepo.*`, and replace all `h.authRepo.CreateRefreshToken`, `h.authRepo.IsRefreshTokenValid`, `h.authRepo.GetRefreshToken`, `h.authRepo.RevokeRefreshToken` with `h.tokenRepo.*`.

- [ ] **Step 2: Update app.go to pass repos explicitly**

In `netme-backend/internal/app/app.go`, update handler construction:

```go
userRepo := repositories.NewUserRepository(database)
tokenRepo := repositories.NewTokenRepository(database)
jwtSvc := services.NewJWTService(jwtSecret)

authHandler := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc)
usersHandler := handlers.NewUsersHandler(userRepo)
```

- [ ] **Step 3: Verify backend compiles**

```bash
cd netme-backend && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Create auth_test.go with mocks**

Create `netme-backend/internal/handlers/auth_test.go`:

```go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/services"
)

const testAuthSecret = "test-secret-key-32-chars-minimum!!"

// --- Mock UserRepo ---

type mockUserRepo struct {
	users map[string]*models.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*models.User)}
}

func (m *mockUserRepo) CreateUser(email, passwordHash string) (*models.User, error) {
	if _, exists := m.users[email]; exists {
		return nil, errors.New("duplicate key")
	}
	user := &models.User{
		ID:           "user-" + email,
		Email:        email,
		PasswordHash: passwordHash,
		AuthProvider: "local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[email] = user
	return user, nil
}

func (m *mockUserRepo) GetUserByEmail(email string) (*models.User, error) {
	user, ok := m.users[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockUserRepo) GetUserByID(id string) (*models.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) UpdateLastLogin(userID string) error { return nil }

// --- Mock TokenRepo ---

type mockTokenRepo struct {
	tokens map[string]*models.RefreshToken
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{tokens: make(map[string]*models.RefreshToken)}
}

func (m *mockTokenRepo) CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{
		ID:        "rt-" + token,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.tokens[token] = rt
	return rt, nil
}

func (m *mockTokenRepo) GetRefreshToken(token string) (*models.RefreshToken, error) {
	rt, ok := m.tokens[token]
	if !ok {
		return nil, errors.New("refresh token not found")
	}
	return rt, nil
}

func (m *mockTokenRepo) RevokeRefreshToken(token string) error {
	rt, ok := m.tokens[token]
	if !ok {
		return errors.New("refresh token not found")
	}
	now := time.Now()
	rt.RevokedAt = &now
	return nil
}

func (m *mockTokenRepo) RevokeAllUserTokens(userID string) error {
	now := time.Now()
	for _, rt := range m.tokens {
		if rt.UserID == userID {
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockTokenRepo) IsRefreshTokenValid(token string) (bool, error) {
	rt, err := m.GetRefreshToken(token)
	if err != nil {
		return false, err
	}
	if rt.RevokedAt != nil {
		return false, errors.New("token is revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return false, errors.New("token is expired")
	}
	return true, nil
}

// --- Helpers ---

func newTestAuthRouter() (*gin.Engine, *mockUserRepo, *mockTokenRepo) {
	gin.SetMode(gin.TestMode)
	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	jwtSvc := services.NewJWTService(testAuthSecret)
	h := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc)

	r := gin.New()
	r.POST("/v1/auth/register", h.Register)
	r.POST("/v1/auth/login", h.Login)
	r.POST("/v1/auth/refresh", h.Refresh)
	r.POST("/v1/auth/logout", h.Logout)
	return r, userRepo, tokenRepo
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	return bytes.NewReader(b)
}

// --- Tests ---

func TestRegisterHappyPath(t *testing.T) {
	r, _, _ := newTestAuthRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "new@example.com", "password": "password123"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	if resp.User == nil || resp.User.Email != "new@example.com" {
		t.Errorf("expected user with email 'new@example.com', got %v", resp.User)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	r, _, _ := newTestAuthRouter()

	body := map[string]string{"email": "dup@example.com", "password": "password123"}

	// First registration
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/v1/auth/register", jsonBody(t, body))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first registration expected 201, got %d", w1.Code)
	}

	// Second registration — same email
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/v1/auth/register", jsonBody(t, body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate email, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestLoginWrongPassword(t *testing.T) {
	r, _, _ := newTestAuthRouter()

	// Register
	regReq, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "user@example.com", "password": "correctpass"}))
	regReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), regReq)

	// Login with wrong password
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login",
		jsonBody(t, map[string]string{"email": "user@example.com", "password": "wrongpass"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	r, _, _ := newTestAuthRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login",
		jsonBody(t, map[string]string{"email": "nobody@example.com", "password": "pass"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefreshWithInvalidToken(t *testing.T) {
	r, _, _ := newTestAuthRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/refresh",
		jsonBody(t, map[string]string{"refresh_token": "nonexistent-token"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 5: Run all tests**

```bash
cd netme-backend && go test ./... -v
```

Expected: all tests PASS (JWT tests, password tests, handler tests).

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/auth_test.go internal/handlers/auth.go internal/app/app.go
git commit -m "test: add auth handler tests with mock repositories; NewAuthHandler accepts interfaces"
```
