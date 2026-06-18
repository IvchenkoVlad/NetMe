package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
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
	stored := &models.User{
		ID:           "user-" + email,
		Email:        email,
		PasswordHash: passwordHash,
		AuthProvider: "local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[email] = stored
	// Return a copy so callers zeroing PasswordHash don't corrupt the stored record.
	returned := *stored
	return &returned, nil
}

func (m *mockUserRepo) GetUserByEmail(email string) (*models.User, error) {
	user, ok := m.users[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	copy := *user
	return &copy, nil
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

func (m *mockUserRepo) DeleteUser(userID string) error {
	for email, u := range m.users {
		if u.ID == userID {
			delete(m.users, email)
			return nil
		}
	}
	return errors.New("user not found")
}

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

func (m *mockTokenRepo) RevokeRefreshToken(token, userID string) error {
	rt, ok := m.tokens[token]
	if !ok {
		return errors.New("refresh token not found")
	}
	if rt.UserID != userID {
		return errors.New("token does not belong to user")
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

// --- Mock GoogleVerifier ---

type mockGoogleVerifier struct {
	googleID string
	email    string
	err      error
}

func (m *mockGoogleVerifier) Validate(_ context.Context, _, _ string) (string, string, error) {
	return m.googleID, m.email, m.err
}

// --- Helpers ---

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

func TestRegisterNormalizesEmail(t *testing.T) {
	r, _, _ := newTestAuthRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "  USER@EXAMPLE.COM  ", "password": "password123"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.AuthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.User.Email != "user@example.com" {
		t.Errorf("expected normalized email 'user@example.com', got %q", resp.User.Email)
	}
}

func TestLoginNormalizesEmail(t *testing.T) {
	r, userRepo, _ := newTestAuthRouter()

	// Register with lowercase first
	regReq, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "mixed@example.com", "password": "password123"}))
	regReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), regReq)
	_ = userRepo

	// Login with mixed case
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login",
		jsonBody(t, map[string]string{"email": "MIXED@EXAMPLE.COM", "password": "password123"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for normalized login, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterConcurrentDuplicate(t *testing.T) {
	// Simulate TOCTOU: GetUserByEmail passes but CreateUser returns pq unique violation
	userRepo := &concurrentMockUserRepo{}
	tokenRepo := newMockTokenRepo()
	jwtSvc := services.NewJWTService(testAuthSecret)
	h := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc, &mockGoogleVerifier{})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v1/auth/register", h.Register)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/register",
		jsonBody(t, map[string]string{"email": "race@example.com", "password": "password123"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for concurrent duplicate, got %d: %s", w.Code, w.Body.String())
	}
}

// concurrentMockUserRepo simulates a TOCTOU race: pre-check passes but INSERT hits unique constraint.
type concurrentMockUserRepo struct{}

func (m *concurrentMockUserRepo) CreateUser(email, passwordHash string) (*models.User, error) {
	return nil, &pq.Error{Code: "23505"}
}
func (m *concurrentMockUserRepo) GetUserByEmail(email string) (*models.User, error) {
	return nil, errors.New("user not found")
}
func (m *concurrentMockUserRepo) GetUserByID(id string) (*models.User, error) {
	return nil, errors.New("user not found")
}
func (m *concurrentMockUserRepo) UpdateLastLogin(userID string) error { return nil }
func (m *concurrentMockUserRepo) FindOrCreateGoogleUser(googleID, email string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *concurrentMockUserRepo) DeleteUser(userID string) error { return nil }
