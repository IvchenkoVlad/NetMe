# Category Correction Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow users to correct a transaction's category and optionally create a merchant→category rule that applies to future (and past) transactions.

**Architecture:** Two separate API round trips from mobile — `PATCH /v1/transactions/:id` fixes the single transaction, `POST /v1/rules` creates/overwrites the merchant rule with optional inline backfill (≤500 rows). Rules are stored in a new `category_rules` table keyed on `(user_id, normalized_merchant)`. A new `category_id` UUID column on `transactions` stores user overrides and takes precedence over Plaid's text category during budget summaries.

**Tech Stack:** Go 1.22, gin, lib/pq, goose migrations, React Native with @react-navigation/native-stack, axios (via authService.api).

## Global Constraints

- Every DB query must filter by `user_id` — no cross-user data leakage.
- Follow the existing handler→repository pattern (no service layer for new code; budget and plaid repos call DB directly).
- Use `*sql.DB` and raw SQL — no ORM.
- Migrations use goose format with `-- +goose Up` / `-- +goose Down` markers.
- Migration files live in `netme-backend/internal/db/migrations/` and are numbered sequentially.
- Go module path is `github.com/vladyslavivchenko/netme`.
- React Native screens follow the existing `GLASS` style constant and dark theme (`#0f172a` background, `#2dd4a7` accent).
- No new npm packages — use only libraries already in the project.
- All new Go handler tests use mock structs (not the real DB), following the pattern in `internal/handlers/auth_test.go`.

---

## File Map

**Create:**
- `netme-backend/internal/db/migrations/0008_category_correction.sql` — adds `category_id` to transactions; creates `category_rules` table
- `netme-backend/internal/models/rules.go` — `CategoryRule` struct
- `netme-backend/internal/repositories/rules.go` — `RulesRepository` with `Upsert`, `List`, `Delete`, `ApplyToPast`
- `netme-backend/internal/handlers/rules.go` — `CreateRule`, `ListRules`, `DeleteRule` HTTP handlers + tests
- `netme-mobile/src/services/transactionService.ts` — `getTransaction`, `patchTransaction`, `createRule`, `listRules` API calls
- `netme-mobile/src/screens/TransactionDetailScreen.tsx` — detail view, category picker bottom sheet, rule prompt

**Modify:**
- `netme-backend/internal/models/plaid.go` — add `CategoryID *string json:"category_id,omitempty"` to `Transaction`
- `netme-backend/internal/repositories/plaid.go` — add `GetTransactionByID`, `PatchTransactionCategory`
- `netme-backend/internal/handlers/transactions.go` — add `GetTransaction`, `PatchTransaction` handlers
- `netme-backend/internal/app/app.go` — instantiate `RulesRepository`, register rules routes
- `netme-mobile/src/navigation/RootNavigator.tsx` — add `TransactionDetail` screen to `AppStack`
- `netme-mobile/src/screens/AccountsScreen.tsx` — make transaction rows tappable (`onPress` → navigate to `TransactionDetail`)

---

### Task 1: DB Migration

**Files:**
- Create: `netme-backend/internal/db/migrations/0008_category_correction.sql`

**Interfaces:**
- Produces: `transactions.category_id` UUID column; `category_rules` table with columns `id, user_id, normalized_merchant, category_id, created_at, updated_at`

- [ ] **Step 1: Write the migration file**

```sql
-- +goose Up

ALTER TABLE transactions
  ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE SET NULL;

CREATE INDEX idx_transactions_category_id ON transactions(category_id);

CREATE TABLE category_rules (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  normalized_merchant TEXT        NOT NULL,
  category_id         UUID        NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, normalized_merchant)
);

CREATE INDEX idx_category_rules_user_id ON category_rules(user_id);

-- +goose Down

DROP TABLE category_rules;
ALTER TABLE transactions DROP COLUMN category_id;
```

- [ ] **Step 2: Run the migration**

```bash
cd netme-backend
DATABASE_URL="postgres://postgres:postgres@localhost:5432/netme?sslmode=disable" go run ./cmd/migrate up
```

Expected output: `OK    0008_category_correction.sql`

- [ ] **Step 3: Verify schema**

```bash
DATABASE_URL="postgres://postgres:postgres@localhost:5432/netme?sslmode=disable" psql $DATABASE_URL -c "\d transactions" | grep category_id
DATABASE_URL="postgres://postgres:postgres@localhost:5432/netme?sslmode=disable" psql $DATABASE_URL -c "\d category_rules"
```

Expected: `category_id` column present on transactions; `category_rules` table exists with correct columns.

- [ ] **Step 4: Commit**

```bash
git add netme-backend/internal/db/migrations/0008_category_correction.sql
git commit -m "feat: add category_id to transactions and create category_rules table"
```

---

### Task 2: Transaction Model + Repository Methods

**Files:**
- Modify: `netme-backend/internal/models/plaid.go`
- Modify: `netme-backend/internal/repositories/plaid.go`

**Interfaces:**
- Produces:
  - `Transaction.CategoryID *string` — user-set category UUID
  - `PlaidRepository.GetTransactionByID(userID, id string) (*models.Transaction, error)` — returns nil, sql.ErrNoRows if not found or wrong user
  - `PlaidRepository.PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error)` — updates and returns full transaction

- [ ] **Step 1: Add CategoryID to Transaction model**

In `netme-backend/internal/models/plaid.go`, add `CategoryID` field to the `Transaction` struct after `Pending`:

```go
type Transaction struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	AccountID          string    `json:"account_id"`
	PlaidTransactionID string    `json:"plaid_transaction_id"`
	Amount             float64   `json:"amount"`
	CurrencyCode       string    `json:"currency_code"`
	Name               string    `json:"name"`
	MerchantName       *string   `json:"merchant_name,omitempty"`
	Date               string    `json:"date"`
	AuthorizedDate     *string   `json:"authorized_date,omitempty"`
	Category           *string   `json:"category,omitempty"`
	CategoryDetailed   *string   `json:"category_detailed,omitempty"`
	PaymentChannel     *string   `json:"payment_channel,omitempty"`
	Pending            bool      `json:"pending"`
	CategoryID         *string   `json:"category_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Update GetTransactionsByUserID to scan category_id**

In `netme-backend/internal/repositories/plaid.go`, update `GetTransactionsByUserID` — add `category_id` to the SELECT and add `&t.CategoryID` to the Scan call:

```go
func (r *PlaidRepository) GetTransactionsByUserID(userID, accountID string, limit, offset int) ([]*models.Transaction, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name,
		        to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		        category, category_detailed, payment_channel, pending, category_id, created_at, updated_at
		 FROM transactions
		 WHERE user_id = $1 AND ($2 = '' OR account_id::text = $2)
		 ORDER BY date DESC, created_at DESC
		 LIMIT $3 OFFSET $4`, userID, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []*models.Transaction
	for rows.Next() {
		t := &models.Transaction{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID, &t.Amount, &t.CurrencyCode,
			&t.Name, &t.MerchantName, &t.Date, &t.AuthorizedDate,
			&t.Category, &t.CategoryDetailed, &t.PaymentChannel, &t.Pending, &t.CategoryID,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}
```

- [ ] **Step 3: Add GetTransactionByID**

Append to `netme-backend/internal/repositories/plaid.go`:

```go
func (r *PlaidRepository) GetTransactionByID(userID, id string) (*models.Transaction, error) {
	t := &models.Transaction{}
	err := r.db.QueryRow(
		`SELECT id, user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name,
		        to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		        category, category_detailed, payment_channel, pending, category_id, created_at, updated_at
		 FROM transactions WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID, &t.Amount, &t.CurrencyCode,
		&t.Name, &t.MerchantName, &t.Date, &t.AuthorizedDate,
		&t.Category, &t.CategoryDetailed, &t.PaymentChannel, &t.Pending, &t.CategoryID,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}
```

- [ ] **Step 4: Add PatchTransactionCategory**

Append to `netme-backend/internal/repositories/plaid.go`:

```go
func (r *PlaidRepository) PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error) {
	t := &models.Transaction{}
	err := r.db.QueryRow(
		`UPDATE transactions SET category_id = $1, updated_at = now()
		 WHERE id = $2 AND user_id = $3
		 RETURNING id, user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name,
		           to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		           category, category_detailed, payment_channel, pending, category_id, created_at, updated_at`,
		categoryID, txnID, userID,
	).Scan(&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID, &t.Amount, &t.CurrencyCode,
		&t.Name, &t.MerchantName, &t.Date, &t.AuthorizedDate,
		&t.Category, &t.CategoryDetailed, &t.PaymentChannel, &t.Pending, &t.CategoryID,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}
```

- [ ] **Step 5: Build to verify no compile errors**

```bash
cd netme-backend && go build ./...
```

Expected: no output (success).

- [ ] **Step 6: Commit**

```bash
git add netme-backend/internal/models/plaid.go netme-backend/internal/repositories/plaid.go
git commit -m "feat: add category_id to Transaction model and repo methods GetTransactionByID, PatchTransactionCategory"
```

---

### Task 3: GET + PATCH /v1/transactions/:id Handlers

**Files:**
- Modify: `netme-backend/internal/handlers/transactions.go`

**Interfaces:**
- Consumes: `PlaidRepository.GetTransactionByID(userID, id string)`, `PlaidRepository.PatchTransactionCategory(userID, txnID, categoryID string)`
- Produces:
  - `GET /v1/transactions/:id` → 200 `{"transaction": {...}}` or 404
  - `PATCH /v1/transactions/:id` body `{"category_id":"uuid"}` → 200 `{"transaction": {...}}` or 400/404

- [ ] **Step 1: Write failing handler tests**

Create `netme-backend/internal/handlers/transactions_test.go`:

```go
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

func (m *mockTxnRepo) GetTransactionsByUserID(userID, accountID string, limit, offset int) ([]*models.Transaction, error) {
	return nil, nil
}

// ─── Helper ──────────────────────────────────────────────────────────────────

func txnRouter(repo handlers.TxnRepo) *gin.Engine {
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd netme-backend && go test ./internal/handlers/... -run TestGetTransaction -v
```

Expected: FAIL — `handlers.TxnRepo` undefined, `RegisterTransactionRoutes` signature mismatch.

- [ ] **Step 3: Define TxnRepo interface and update handler**

Replace the entire content of `netme-backend/internal/handlers/transactions.go`:

```go
package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

// TxnRepo is the subset of PlaidRepository used by transaction handlers.
type TxnRepo interface {
	GetTransactionsByUserID(userID, accountID string, limit, offset int) ([]*models.Transaction, error)
	GetTransactionByID(userID, id string) (*models.Transaction, error)
	PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error)
}

type TransactionsHandler struct {
	repo TxnRepo
}

func NewTransactionsHandler(repo TxnRepo) *TransactionsHandler {
	return &TransactionsHandler{repo: repo}
}

func RegisterTransactionRoutes(r *gin.RouterGroup, repo TxnRepo) {
	h := NewTransactionsHandler(repo)
	txns := r.Group("/transactions")
	{
		txns.GET("", h.ListTransactions)
		txns.GET("/:id", h.GetTransaction)
		txns.PATCH("/:id", h.PatchTransaction)
	}
}

func (h *TransactionsHandler) ListTransactions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	limit := 50
	offset := 0
	accountID := c.Query("account_id")
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	txns, err := h.repo.GetTransactionsByUserID(uid, accountID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "failed to load transactions",
		})
		return
	}
	if txns == nil {
		txns = []*models.Transaction{}
	}
	c.JSON(http.StatusOK, gin.H{"transactions": txns})
}

func (h *TransactionsHandler) GetTransaction(c *gin.Context) {
	userID, _ := c.Get("user_id")
	txn, err := h.repo.GetTransactionByID(userID.(string), c.Param("id"))
	if err == sql.ErrNoRows || txn == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to load transaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}

func (h *TransactionsHandler) PatchTransaction(c *gin.Context) {
	var req struct {
		CategoryID string `json:"category_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: "category_id is required"})
		return
	}
	userID, _ := c.Get("user_id")
	txn, err := h.repo.PatchTransactionCategory(userID.(string), c.Param("id"), req.CategoryID)
	if err == sql.ErrNoRows || txn == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to update transaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd netme-backend && go test ./internal/handlers/... -run "TestGetTransaction|TestPatchTransaction" -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add netme-backend/internal/handlers/transactions.go netme-backend/internal/handlers/transactions_test.go
git commit -m "feat: add GET /v1/transactions/:id and PATCH /v1/transactions/:id"
```

---

### Task 4: CategoryRule Model + RulesRepository

**Files:**
- Create: `netme-backend/internal/models/rules.go`
- Create: `netme-backend/internal/repositories/rules.go`

**Interfaces:**
- Produces:
  - `models.CategoryRule{ID, UserID, NormalizedMerchant, CategoryID, Category *models.Category, CreatedAt}`
  - `RulesRepository.Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error)`
  - `RulesRepository.ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error)` — updates ≤500 most recent non-pending transactions; returns rows updated
  - `RulesRepository.List(userID string) ([]*models.CategoryRule, error)`
  - `RulesRepository.Delete(userID, id string) error`

- [ ] **Step 1: Create the model**

Create `netme-backend/internal/models/rules.go`:

```go
package models

import "time"

type CategoryRule struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	NormalizedMerchant string    `json:"normalized_merchant"`
	CategoryID         string    `json:"category_id"`
	Category           *Category `json:"category,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Create the repository**

Create `netme-backend/internal/repositories/rules.go`:

```go
package repositories

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type RulesRepository struct {
	db *sql.DB
}

func NewRulesRepository(db *sql.DB) *RulesRepository {
	return &RulesRepository{db: db}
}

func (r *RulesRepository) Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error) {
	rule := &models.CategoryRule{}
	err := r.db.QueryRow(
		`INSERT INTO category_rules (user_id, normalized_merchant, category_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, normalized_merchant)
		 DO UPDATE SET category_id = EXCLUDED.category_id, updated_at = now()
		 RETURNING id, user_id, normalized_merchant, category_id, created_at, updated_at`,
		userID, normalizedMerchant, categoryID,
	).Scan(&rule.ID, &rule.UserID, &rule.NormalizedMerchant, &rule.CategoryID, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	cat, err := r.getCategoryByID(categoryID, userID)
	if err == nil {
		rule.Category = cat
	}
	return rule, nil
}

func (r *RulesRepository) ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error) {
	res, err := r.db.Exec(
		`UPDATE transactions SET category_id = $1, updated_at = now()
		 WHERE id IN (
		   SELECT id FROM transactions
		   WHERE user_id = $2
		     AND LOWER(TRIM(COALESCE(merchant_name, name))) = $3
		     AND pending = false
		   ORDER BY date DESC
		   LIMIT 500
		 )`,
		categoryID, userID, normalizedMerchant,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *RulesRepository) List(userID string) ([]*models.CategoryRule, error) {
	rows, err := r.db.Query(
		`SELECT cr.id, cr.user_id, cr.normalized_merchant, cr.category_id, cr.created_at, cr.updated_at,
		        c.id, c.user_id, c.name, c.icon, c.color, c.is_income, c.sort_order, c.plaid_primary_categories, c.created_at, c.updated_at
		 FROM category_rules cr
		 JOIN categories c ON c.id = cr.category_id
		 WHERE cr.user_id = $1
		 ORDER BY cr.normalized_merchant`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.CategoryRule
	for rows.Next() {
		rule := &models.CategoryRule{}
		cat := &models.Category{}
		if err := rows.Scan(
			&rule.ID, &rule.UserID, &rule.NormalizedMerchant, &rule.CategoryID, &rule.CreatedAt, &rule.UpdatedAt,
			&cat.ID, &cat.UserID, &cat.Name, &cat.Icon, &cat.Color, &cat.IsIncome, &cat.SortOrder,
			pq.Array(&cat.PlaidPrimaryCategories), &cat.CreatedAt, &cat.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rule.Category = cat
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *RulesRepository) Delete(userID, id string) error {
	res, err := r.db.Exec(`DELETE FROM category_rules WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *RulesRepository) getCategoryByID(categoryID, userID string) (*models.Category, error) {
	cat := &models.Category{}
	err := r.db.QueryRow(
		`SELECT id, user_id, name, icon, color, is_income, sort_order, plaid_primary_categories, created_at, updated_at
		 FROM categories WHERE id = $1 AND user_id = $2`, categoryID, userID,
	).Scan(&cat.ID, &cat.UserID, &cat.Name, &cat.Icon, &cat.Color, &cat.IsIncome, &cat.SortOrder,
		pq.Array(&cat.PlaidPrimaryCategories), &cat.CreatedAt, &cat.UpdatedAt)
	return cat, err
}
```

- [ ] **Step 3: Build to verify no compile errors**

```bash
cd netme-backend && go build ./...
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add netme-backend/internal/models/rules.go netme-backend/internal/repositories/rules.go
git commit -m "feat: add CategoryRule model and RulesRepository"
```

---

### Task 5: POST + GET + DELETE /v1/rules Handlers

**Files:**
- Create: `netme-backend/internal/handlers/rules.go`

**Interfaces:**
- Consumes: `RulesRepository.Upsert`, `RulesRepository.ApplyToPast`, `RulesRepository.List`, `RulesRepository.Delete`
- Produces:
  - `POST /v1/rules` body `{"normalized_merchant":"starbucks","category_id":"uuid","apply_to_past":false}` → 201 `{"rule":{...},"updated_count":0}`
  - `GET /v1/rules` → 200 `{"rules":[...]}`
  - `DELETE /v1/rules/:id` → 204 or 404

- [ ] **Step 1: Write failing handler tests**

Create `netme-backend/internal/handlers/rules_test.go`:

```go
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
```

- [ ] **Step 2: Run to confirm they fail**

```bash
cd netme-backend && go test ./internal/handlers/... -run "TestCreateRule|TestListRules|TestDeleteRule" -v
```

Expected: FAIL — `handlers.RulesRepo` undefined, `RegisterRulesRoutes` undefined.

- [ ] **Step 3: Implement the handler**

Create `netme-backend/internal/handlers/rules.go`:

```go
package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
)

// RulesRepo is the subset of RulesRepository used by rules handlers.
type RulesRepo interface {
	Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error)
	ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error)
	List(userID string) ([]*models.CategoryRule, error)
	Delete(userID, id string) error
}

type RulesHandler struct {
	repo RulesRepo
}

func NewRulesHandler(repo RulesRepo) *RulesHandler {
	return &RulesHandler{repo: repo}
}

func RegisterRulesRoutes(r *gin.RouterGroup, repo RulesRepo) {
	h := NewRulesHandler(repo)
	rules := r.Group("/rules")
	{
		rules.POST("", h.CreateRule)
		rules.GET("", h.ListRules)
		rules.DELETE("/:id", h.DeleteRule)
	}
}

func (h *RulesHandler) CreateRule(c *gin.Context) {
	var req struct {
		NormalizedMerchant string `json:"normalized_merchant" binding:"required"`
		CategoryID         string `json:"category_id" binding:"required"`
		ApplyToPast        bool   `json:"apply_to_past"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	uid := userID.(string)

	rule, err := h.repo.Upsert(uid, req.NormalizedMerchant, req.CategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to save rule"})
		return
	}

	var updatedCount int64
	if req.ApplyToPast {
		updatedCount, err = h.repo.ApplyToPast(uid, req.NormalizedMerchant, req.CategoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to apply rule to past"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"rule": rule, "updated_count": updatedCount})
}

func (h *RulesHandler) ListRules(c *gin.Context) {
	userID, _ := c.Get("user_id")
	rules, err := h.repo.List(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to load rules"})
		return
	}
	if rules == nil {
		rules = []*models.CategoryRule{}
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (h *RulesHandler) DeleteRule(c *gin.Context) {
	userID, _ := c.Get("user_id")
	err := h.repo.Delete(userID.(string), c.Param("id"))
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "not_found", Message: "rule not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database_error", Message: "failed to delete rule"})
		return
	}
	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd netme-backend && go test ./internal/handlers/... -run "TestCreateRule|TestListRules|TestDeleteRule" -v
```

Expected: all 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add netme-backend/internal/handlers/rules.go netme-backend/internal/handlers/rules_test.go
git commit -m "feat: add POST/GET/DELETE /v1/rules handlers"
```

---

### Task 6: Wire Rules into app.go

**Files:**
- Modify: `netme-backend/internal/app/app.go`

**Interfaces:**
- Consumes: `repositories.NewRulesRepository(database)`, `handlers.RegisterRulesRoutes(protected, rulesRepo)`

- [ ] **Step 1: Add RulesRepository instantiation and route registration**

In `netme-backend/internal/app/app.go`, after `budgetRepo := repositories.NewBudgetRepository(database)` add:

```go
rulesRepo := repositories.NewRulesRepository(database)
```

In the `protected` route group block, after `handlers.RegisterBudgetRoutes(protected, budgetRepo)` add:

```go
handlers.RegisterRulesRoutes(protected, rulesRepo)
```

- [ ] **Step 2: Build and run the full test suite**

```bash
cd netme-backend && go build ./... && go test ./...
```

Expected: build succeeds; all tests pass.

- [ ] **Step 3: Smoke-test the server locally**

```bash
cd netme-backend && go run ./cmd/server
```

In another terminal:
```bash
curl -s http://localhost:8080/healthz
```

Expected: `{"status":"ok"}` (or whatever the health handler returns).

- [ ] **Step 4: Commit**

```bash
git add netme-backend/internal/app/app.go
git commit -m "feat: register RulesRepository and rules routes in app"
```

---

### Task 7: Mobile transactionService.ts

**Files:**
- Create: `netme-mobile/src/services/transactionService.ts`

**Interfaces:**
- Produces:
  - `transactionService.getTransaction(id: string): Promise<Transaction>`
  - `transactionService.patchTransaction(id: string, categoryId: string): Promise<Transaction>`
  - `transactionService.createRule(normalizedMerchant: string, categoryId: string, applyToPast: boolean): Promise<{rule: CategoryRule, updated_count: number}>`
  - `transactionService.listRules(): Promise<CategoryRule[]>`

```ts
export interface Transaction {
  id: string;
  user_id: string;
  account_id: string;
  plaid_transaction_id: string;
  amount: number;
  currency_code: string;
  name: string;
  merchant_name?: string;
  date: string;
  authorized_date?: string;
  category?: string;
  category_detailed?: string;
  payment_channel?: string;
  pending: boolean;
  category_id?: string;
}

export interface CategoryRule {
  id: string;
  normalized_merchant: string;
  category_id: string;
  category?: {
    id: string;
    name: string;
    icon: string;
    color: string;
    is_income: boolean;
  };
  created_at: string;
}
```

- [ ] **Step 1: Create the service file**

Create `netme-mobile/src/services/transactionService.ts`:

```ts
import { authService } from './authService';

const api = authService.api;

export interface Transaction {
  id: string;
  user_id: string;
  account_id: string;
  plaid_transaction_id: string;
  amount: number;
  currency_code: string;
  name: string;
  merchant_name?: string;
  date: string;
  authorized_date?: string;
  category?: string;
  category_detailed?: string;
  payment_channel?: string;
  pending: boolean;
  category_id?: string;
}

export interface CategoryRule {
  id: string;
  normalized_merchant: string;
  category_id: string;
  category?: {
    id: string;
    name: string;
    icon: string;
    color: string;
    is_income: boolean;
  };
  created_at: string;
}

export const transactionService = {
  getTransaction: async (id: string): Promise<Transaction> => {
    const { data } = await api.get(`/transactions/${id}`);
    return data.transaction;
  },

  patchTransaction: async (id: string, categoryId: string): Promise<Transaction> => {
    const { data } = await api.patch(`/transactions/${id}`, { category_id: categoryId });
    return data.transaction;
  },

  createRule: async (
    normalizedMerchant: string,
    categoryId: string,
    applyToPast: boolean,
  ): Promise<{ rule: CategoryRule; updated_count: number }> => {
    const { data } = await api.post('/rules', {
      normalized_merchant: normalizedMerchant,
      category_id: categoryId,
      apply_to_past: applyToPast,
    });
    return data;
  },

  listRules: async (): Promise<CategoryRule[]> => {
    const { data } = await api.get('/rules');
    return data.rules ?? [];
  },
};
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd netme-mobile && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add netme-mobile/src/services/transactionService.ts
git commit -m "feat: add transactionService with getTransaction, patchTransaction, createRule, listRules"
```

---

### Task 8: TransactionDetailScreen

**Files:**
- Create: `netme-mobile/src/screens/TransactionDetailScreen.tsx`

**Interfaces:**
- Consumes: `transactionService.getTransaction`, `transactionService.patchTransaction`, `transactionService.createRule`; `budgetService` (for `getCategories`)
- Route params: `{ transactionId: string }`

- [ ] **Step 1: Create the screen**

Create `netme-mobile/src/screens/TransactionDetailScreen.tsx`:

```tsx
import React, { useCallback, useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Modal,
  StyleSheet,
  Switch,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { transactionService, Transaction } from '../services/transactionService';
import { budgetService } from '../services/budgetService';

// ─── Types ────────────────────────────────────────────────────────────────────

interface Category {
  id: string;
  name: string;
  icon: string;
  color: string;
  is_income: boolean;
}

// ─── Constants ────────────────────────────────────────────────────────────────

const GLASS = {
  backgroundColor: 'rgba(255,255,255,0.06)',
  borderRadius: 16,
  borderWidth: 1,
  borderColor: 'rgba(255,255,255,0.1)',
} as const;

const fmt = (amount: number, currency = 'USD') =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(Math.abs(amount));

const fmtDate = (dateStr: string) => {
  if (!dateStr) return '';
  const [y, m, d] = dateStr.split('T')[0].split('-').map(Number);
  if (!y || !m || !d) return dateStr;
  return new Date(y, m - 1, d).toLocaleDateString('en-US', {
    month: 'long', day: 'numeric', year: 'numeric',
  });
};

const normalize = (name: string) => name.toLowerCase().trim();

// ─── Component ────────────────────────────────────────────────────────────────

export const TransactionDetailScreen: React.FC<{ route: any; navigation: any }> = ({
  route,
  navigation,
}) => {
  const insets = useSafeAreaInsets();
  const { transactionId } = route.params as { transactionId: string };

  const [txn, setTxn] = useState<Transaction | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pickerVisible, setPickerVisible] = useState(false);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [t, cats] = await Promise.all([
        transactionService.getTransaction(transactionId),
        budgetService.getCategories(),
      ]);
      setTxn(t);
      setCategories(cats);
    } catch {
      setError('Failed to load transaction.');
    } finally {
      setLoading(false);
    }
  }, [transactionId]);

  useEffect(() => { load(); }, [load]);

  const currentCategory = categories.find(c => c.id === txn?.category_id);
  const merchantDisplay = txn?.merchant_name ?? txn?.name ?? '';

  const onCategorySelect = async (cat: Category) => {
    if (!txn) return;
    setPickerVisible(false);
    setSaving(true);
    try {
      const updated = await transactionService.patchTransaction(txn.id, cat.id);
      setTxn(updated);
      showRulePrompt(cat);
    } catch {
      Alert.alert('Error', 'Could not update category. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const showRulePrompt = (cat: Category) => {
    Alert.alert(
      'Create Rule',
      `Always categorize "${merchantDisplay}" as ${cat.icon} ${cat.name}?`,
      [
        { text: 'No', style: 'cancel' },
        {
          text: 'Yes',
          onPress: () => showApplyToPastPrompt(cat),
        },
      ],
    );
  };

  const showApplyToPastPrompt = (cat: Category) => {
    Alert.alert(
      'Fix Past Transactions',
      `Also update past transactions from "${merchantDisplay}"?`,
      [
        {
          text: 'No, future only',
          onPress: () => saveRule(cat, false),
        },
        {
          text: 'Yes, fix past too',
          onPress: () => saveRule(cat, true),
        },
      ],
    );
  };

  const saveRule = async (cat: Category, applyToPast: boolean) => {
    try {
      const result = await transactionService.createRule(
        normalize(merchantDisplay),
        cat.id,
        applyToPast,
      );
      const msg =
        applyToPast && result.updated_count > 0
          ? `Rule saved — ${result.updated_count} past transaction${result.updated_count === 1 ? '' : 's'} updated.`
          : 'Rule saved.';
      Alert.alert('Done', msg);
    } catch {
      Alert.alert('Error', 'Rule could not be saved. Please try again.');
    }
  };

  if (loading) {
    return (
      <View style={[styles.container, { paddingTop: insets.top }]}>
        <ActivityIndicator size="large" color="#2dd4a7" style={{ marginTop: 80 }} />
      </View>
    );
  }

  if (error || !txn) {
    return (
      <View style={[styles.container, { paddingTop: insets.top }]}>
        <TouchableOpacity onPress={() => navigation.goBack()} style={styles.backBtn}>
          <Text style={styles.backBtnText}>← Back</Text>
        </TouchableOpacity>
        <Text style={styles.errorText}>{error ?? 'Transaction not found.'}</Text>
      </View>
    );
  }

  const expenseCategories = categories.filter(c => !c.is_income);
  const incomeCategories = categories.filter(c => c.is_income);

  return (
    <View style={[styles.container, { paddingTop: insets.top }]}>
      {/* Header */}
      <TouchableOpacity onPress={() => navigation.goBack()} style={styles.backBtn}>
        <Text style={styles.backBtnText}>← Back</Text>
      </TouchableOpacity>

      {/* Amount + Merchant */}
      <View style={[GLASS, styles.card]}>
        <Text style={styles.amount}>
          {txn.amount < 0 ? '+' : '-'}{fmt(txn.amount, txn.currency_code)}
        </Text>
        <Text style={styles.merchant}>{merchantDisplay}</Text>
        <Text style={styles.meta}>{fmtDate(txn.date)}</Text>
        {txn.pending && <Text style={styles.badge}>PENDING</Text>}
      </View>

      {/* Category */}
      <TouchableOpacity
        style={[GLASS, styles.card, styles.row]}
        onPress={() => setPickerVisible(true)}
        disabled={saving}
      >
        <Text style={styles.label}>Category</Text>
        <View style={styles.row}>
          {currentCategory ? (
            <Text style={styles.categoryChip}>
              {currentCategory.icon} {currentCategory.name}
            </Text>
          ) : (
            <Text style={styles.categoryChipEmpty}>Tap to categorize</Text>
          )}
          <Text style={styles.chevron}> ›</Text>
        </View>
      </TouchableOpacity>

      {saving && <ActivityIndicator color="#2dd4a7" style={{ marginTop: 12 }} />}

      {/* Category Picker Modal */}
      <Modal
        visible={pickerVisible}
        transparent
        animationType="slide"
        onRequestClose={() => setPickerVisible(false)}
      >
        <TouchableOpacity
          style={styles.backdrop}
          activeOpacity={1}
          onPress={() => setPickerVisible(false)}
        />
        <View style={styles.sheet}>
          <Text style={styles.sheetTitle}>Select Category</Text>
          <FlatList
            data={[
              { title: 'Expenses', data: expenseCategories },
              { title: 'Income', data: incomeCategories },
            ]}
            keyExtractor={item => item.title}
            renderItem={({ item: section }) => (
              <View>
                <Text style={styles.sectionHeader}>{section.title}</Text>
                {section.data.map(cat => (
                  <TouchableOpacity
                    key={cat.id}
                    style={styles.catRow}
                    onPress={() => onCategorySelect(cat)}
                  >
                    <Text style={styles.catIcon}>{cat.icon}</Text>
                    <Text style={styles.catName}>{cat.name}</Text>
                    {txn.category_id === cat.id && (
                      <Text style={styles.checkmark}>✓</Text>
                    )}
                  </TouchableOpacity>
                ))}
              </View>
            )}
          />
        </View>
      </Modal>
    </View>
  );
};

// ─── Styles ──────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0f172a', padding: 16 },
  backBtn: { marginBottom: 16 },
  backBtnText: { color: '#2dd4a7', fontSize: 16 },
  card: { padding: 20, marginBottom: 12 },
  row: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  amount: { fontSize: 36, fontWeight: '700', color: '#f1f5f9', marginBottom: 4 },
  merchant: { fontSize: 18, color: '#cbd5e1', marginBottom: 4 },
  meta: { fontSize: 14, color: '#64748b' },
  badge: {
    marginTop: 8, alignSelf: 'flex-start',
    backgroundColor: 'rgba(251,191,36,0.2)', color: '#fbbf24',
    paddingHorizontal: 8, paddingVertical: 2, borderRadius: 6, fontSize: 11, fontWeight: '600',
  },
  label: { fontSize: 14, color: '#94a3b8' },
  categoryChip: { fontSize: 15, color: '#f1f5f9' },
  categoryChipEmpty: { fontSize: 15, color: '#64748b' },
  chevron: { fontSize: 20, color: '#64748b' },
  errorText: { color: '#f87171', fontSize: 16, textAlign: 'center', marginTop: 40 },
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.5)' },
  sheet: {
    backgroundColor: '#1e293b', borderTopLeftRadius: 20, borderTopRightRadius: 20,
    paddingHorizontal: 16, paddingTop: 16, paddingBottom: 40, maxHeight: '75%',
  },
  sheetTitle: { fontSize: 18, fontWeight: '600', color: '#f1f5f9', marginBottom: 12 },
  sectionHeader: { fontSize: 12, color: '#64748b', marginTop: 12, marginBottom: 4, textTransform: 'uppercase' },
  catRow: {
    flexDirection: 'row', alignItems: 'center',
    paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: 'rgba(255,255,255,0.05)',
  },
  catIcon: { fontSize: 20, width: 32 },
  catName: { flex: 1, fontSize: 15, color: '#f1f5f9' },
  checkmark: { color: '#2dd4a7', fontSize: 16, fontWeight: '700' },
});
```

- [ ] **Step 2: Check TypeScript**

```bash
cd netme-mobile && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add netme-mobile/src/screens/TransactionDetailScreen.tsx
git commit -m "feat: add TransactionDetailScreen with category picker and rule prompt"
```

---

### Task 9: Register Screen + Wire Transaction Row Taps

**Files:**
- Modify: `netme-mobile/src/navigation/RootNavigator.tsx`
- Modify: `netme-mobile/src/screens/AccountsScreen.tsx`

**Interfaces:**
- Consumes: `TransactionDetailScreen` with `route.params.transactionId`
- Navigation call: `navigation.navigate('TransactionDetail', { transactionId: txn.id })`

- [ ] **Step 1: Add TransactionDetail to AppStack in RootNavigator.tsx**

In `netme-mobile/src/navigation/RootNavigator.tsx`, add the import:

```ts
import { TransactionDetailScreen } from '../screens/TransactionDetailScreen';
```

In `AppStack`, add a new `Stack.Screen` after the `Settings` screen:

```tsx
<Stack.Screen
  name="TransactionDetail"
  component={TransactionDetailScreen}
  options={{ presentation: 'card', animation: 'slide_from_right' }}
/>
```

- [ ] **Step 2: Find where transactions are rendered in AccountsScreen.tsx**

Open `netme-mobile/src/screens/AccountsScreen.tsx`. Search for `Transaction` list rendering — look for a component that maps transaction items and renders name/amount. It will be wrapped in a `View` or `TouchableOpacity`.

Add `useNavigation` import at the top:

```ts
import { useNavigation } from '@react-navigation/native';
```

Add inside the component, before the return:

```ts
const navigation = useNavigation<any>();
```

Find the transaction row render — it will look something like a `View` with `t.name` and `t.amount`. Wrap it in a `TouchableOpacity` (or change existing `View` to `TouchableOpacity`) with:

```tsx
onPress={() => navigation.navigate('TransactionDetail', { transactionId: t.id })}
```

- [ ] **Step 3: Verify TypeScript**

```bash
cd netme-mobile && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add netme-mobile/src/navigation/RootNavigator.tsx netme-mobile/src/screens/AccountsScreen.tsx
git commit -m "feat: register TransactionDetail screen and wire transaction row taps"
```

---

## Self-Review

**Spec coverage check:**
- ✅ `PATCH /v1/transactions/:id` — Task 3
- ✅ `GET /v1/transactions/:id` — Task 3
- ✅ `category_rules` table — Task 1
- ✅ `POST /v1/rules` with `apply_to_past` — Task 5
- ✅ `GET /v1/rules` — Task 5
- ✅ `DELETE /v1/rules/:id` hard delete — Task 5
- ✅ 500-row cap, most-recent ordering — Task 4 (`ApplyToPast`)
- ✅ `category_id` column on transactions — Task 1 + Task 2
- ✅ `TransactionDetailScreen` with category picker — Task 8
- ✅ Rule prompt: "always ask" after every category change — Task 8
- ✅ Apply-to-past as a second prompt — Task 8
- ✅ Navigation wiring — Task 9

**Placeholder scan:** None found.

**Type consistency:**
- `Transaction.CategoryID *string` (Go) ↔ `Transaction.category_id?: string` (TS) ✅
- `CategoryRule.NormalizedMerchant` (Go) ↔ `CategoryRule.normalized_merchant` (TS) ✅
- `RulesRepo.Upsert` / `ApplyToPast` / `List` / `Delete` signatures consistent across Task 4 (repo), Task 5 (interface), and mock in test ✅
- `TxnRepo.GetTransactionByID` / `PatchTransactionCategory` consistent across Task 2 (repo), Task 3 (interface), and mock in test ✅
