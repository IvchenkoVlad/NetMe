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
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

// ─── Mock ────────────────────────────────────────────────────────────────────

type mockTxnRepo struct {
	txns map[string]*models.Transaction
}

func newMockTxnRepo() *mockTxnRepo {
	catID := "cat-1"
	return &mockTxnRepo{txns: map[string]*models.Transaction{
		"txn-1": {
			ID:                 "txn-1",
			UserID:             "user-1",
			AccountID:          "acc-1",
			PlaidTransactionID: "plaid-1",
			Amount:             12.50,
			CurrencyCode:       "USD",
			Name:               "Starbucks",
			Date:               "2026-07-01",
			Pending:            false,
			CategoryID:         &catID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}}
}

func (m *mockTxnRepo) GetTransactionByID(userID, id string) (*models.Transaction, error) {
	t, ok := m.txns[id]
	if !ok || t.UserID != userID {
		return nil, sql.ErrNoRows
	}
	return t, nil
}

func (m *mockTxnRepo) PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error) {
	t, ok := m.txns[txnID]
	if !ok || t.UserID != userID {
		return nil, sql.ErrNoRows
	}
	t.CategoryID = &categoryID
	return t, nil
}

func (m *mockTxnRepo) GetTransactionsByUserID(userID, accountID, month string, limit, offset int) ([]*models.Transaction, error) {
	return nil, nil
}

// ─── Helper ──────────────────────────────────────────────────────────────────

func txnRouter(repo repositories.TxnRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	g := r.Group("/v1")
	handlers.RegisterTransactionRoutes(g, repo)
	return r
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestGetTransaction_found(t *testing.T) {
	r := txnRouter(newMockTxnRepo())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/v1/transactions/txn-1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	txn := resp["transaction"].(map[string]any)
	if txn["id"] != "txn-1" {
		t.Fatalf("unexpected id: %v", txn["id"])
	}
}

func TestGetTransaction_notFound(t *testing.T) {
	r := txnRouter(newMockTxnRepo())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/v1/transactions/no-such", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestPatchTransaction_ok(t *testing.T) {
	r := txnRouter(newMockTxnRepo())
	body, _ := json.Marshal(map[string]string{"category_id": "cat-new"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PATCH", "/v1/transactions/txn-1", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	txn := resp["transaction"].(map[string]any)
	if txn["category_id"] != "cat-new" {
		t.Fatalf("expected category_id cat-new got %v", txn["category_id"])
	}
}

func TestPatchTransaction_missingCategoryID(t *testing.T) {
	r := txnRouter(newMockTxnRepo())
	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PATCH", "/v1/transactions/txn-1", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}
