package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/models"
)

func newUsersRouter(userRepo *mockUserRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := handlers.NewUsersHandler(userRepo, nil)
	r := gin.New()
	// Simulate AuthMiddleware setting user_id
	r.GET("/v1/me", func(c *gin.Context) {
		c.Set("user_id", "user-delete-123")
		h.GetMe(c)
	})
	r.DELETE("/v1/me", func(c *gin.Context) {
		c.Set("user_id", "user-delete-123")
		h.DeleteMe(c)
	})
	return r
}

func TestDeleteMeSuccess(t *testing.T) {
	userRepo := newMockUserRepo()
	userRepo.users["delete@example.com"] = &models.User{
		ID:           "user-delete-123",
		Email:        "delete@example.com",
		AuthProvider: "local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	r := newUsersRouter(userRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/v1/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify user is removed from the store
	if _, exists := userRepo.users["delete@example.com"]; exists {
		t.Error("expected user to be deleted from store")
	}
}

func TestDeleteMeUserNotFound(t *testing.T) {
	userRepo := newMockUserRepo() // empty — no user with that ID

	r := newUsersRouter(userRepo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/v1/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != "db_error" {
		t.Errorf("expected error 'db_error', got %q", resp.Error)
	}
}
