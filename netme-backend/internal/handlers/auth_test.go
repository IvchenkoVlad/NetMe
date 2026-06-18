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

func (m *mockUserRepo) FindOrCreateGoogleUser(googleID, email string) (*models.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	u := &models.User{ID: "google-" + googleID, Email: email, AuthProvider: "google"}
	m.users[email] = u
	return u, nil
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
