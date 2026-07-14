package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/models"
)

// ─── Mock ────────────────────────────────────────────────────────────────────

type mockRulesRepo struct {
	rules map[string]*models.CategoryRule
}

func newMockRulesRepo() *mockRulesRepo {
	cat := &models.Category{ID: "cat-1", Name: "Coffee", Icon: "☕", Color: "#92400e"}
	return &mockRulesRepo{rules: map[string]*models.CategoryRule{
		"rule-1": {
			ID: "rule-1", UserID: "user-1",
			NormalizedMerchant: "starbucks", CategoryID: "cat-1",
			Category: cat, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}}
}

func (m *mockRulesRepo) Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error) {
	r := &models.CategoryRule{
		ID: "rule-new", UserID: userID,
		NormalizedMerchant: normalizedMerchant, CategoryID: categoryID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.rules[r.ID] = r
	return r, nil
}

func (m *mockRulesRepo) ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error) {
	return 3, nil
}

func (m *mockRulesRepo) List(userID string) ([]*models.CategoryRule, error) {
	var out []*models.CategoryRule
	for _, r := range m.rules {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockRulesRepo) Delete(userID, id string) error {
	r, ok := m.rules[id]
	if !ok || r.UserID != userID {
		return sql.ErrNoRows
	}
	delete(m.rules, id)
	return nil
}

// ─── Helper ──────────────────────────────────────────────────────────────────

func rulesRouter(repo handlers.RulesRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	g := r.Group("/v1")
	handlers.RegisterRulesRoutes(g, repo)
	return r
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestCreateRule_ok(t *testing.T) {
	r := rulesRouter(newMockRulesRepo())
	body, _ := json.Marshal(map[string]any{
		"normalized_merchant": "starbucks",
		"category_id":         "cat-1",
		"apply_to_past":       true,
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/v1/rules", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["updated_count"].(float64) != 3 {
		t.Fatalf("expected updated_count 3 got %v", resp["updated_count"])
	}
}

func TestCreateRule_missingFields(t *testing.T) {
	r := rulesRouter(newMockRulesRepo())
	body, _ := json.Marshal(map[string]any{"normalized_merchant": "starbucks"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/v1/rules", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestListRules_ok(t *testing.T) {
	r := rulesRouter(newMockRulesRepo())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/v1/rules", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	rules := resp["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule got %d", len(rules))
	}
}

func TestDeleteRule_ok(t *testing.T) {
	r := rulesRouter(newMockRulesRepo())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/v1/rules/rule-1", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w.Code)
	}
}

func TestDeleteRule_notFound(t *testing.T) {
	r := rulesRouter(newMockRulesRepo())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("DELETE", "/v1/rules/no-such", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}
