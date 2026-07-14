# Aggressive Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the `PlaidRepository` god object into four focused repos, wire proper interfaces throughout backend handlers, reorganize model files, fix mobile type duplication and coupling, and replace inline hex colors with theme constants.

**Architecture:** Backend uses a layered Go arch (handlers → services → repositories). The key change is splitting one 365-line repository into four single-responsibility ones, then defining interfaces for all handler dependencies. Mobile changes are independent — extract a shared Axios instance, fix a missing import, remove a duplicate type, and centralize color constants.

**Tech Stack:** Go 1.21+, Gin, database/sql (Postgres), React Native, TypeScript, Axios, Expo.

## Global Constraints

- No new dependencies (Go modules or npm)
- No DB schema changes
- No HTTP API contract changes (routes, request/response shapes unchanged)
- Preserve all existing behavior exactly — this is structural refactor only
- Backend baseline: `cd netme-backend && go build ./...` must pass; `go test ./...` must pass (handlers, middleware, services packages have tests)
- Mobile baseline: pre-existing `tsconfig.json` TS5098 error is unrelated to this refactor and should remain the only error

---

## File Map

### Backend — files created

| File | Responsibility |
|---|---|
| `internal/repositories/plaid_items.go` | PlaidItemRepository: item CRUD, sync enumeration, token encrypt/decrypt |
| `internal/repositories/accounts.go` | AccountRepository: account CRUD, net worth calc + snapshot |
| `internal/repositories/transactions.go` | TransactionRepository: transaction CRUD + shared `scanTransaction` helper |
| `internal/repositories/events.go` | EventRepository: raw event log, data retention purge |
| `internal/models/account.go` | Account model (moved from plaid.go) |
| `internal/models/transaction.go` | Transaction model (moved from plaid.go) |

### Backend — files deleted

| File | Reason |
|---|---|
| `internal/repositories/plaid.go` | Replaced by the four new repos above |

### Backend — files modified

| File | What changes |
|---|---|
| `internal/models/plaid.go` | Remove Account + Transaction; add PlaidItemEntry type |
| `internal/repositories/budget.go` | `GetTransactionsForMonth` uses shared `scanTransaction` from transactions.go |
| `internal/repositories/interfaces.go` | Add all new interfaces; TxnRepo + RulesRepo moved here from handlers |
| `internal/handlers/transactions.go` | Delete local TxnRepo definition; import from repositories |
| `internal/handlers/rules.go` | Delete local RulesRepo definition; import from repositories |
| `internal/handlers/accounts.go` | AccountsHandler takes AccountLister interface |
| `internal/handlers/analytics.go` | AnalyticsHandler takes NetWorthReader interface |
| `internal/handlers/plaid.go` | PlaidHandler takes PlaidItemGetter + EventLogger interfaces |
| `internal/services/plaid.go` | PlaidService takes four concrete repos instead of *PlaidRepository |
| `internal/jobs/scheduler.go` | Scheduler takes split repos instead of *PlaidRepository |
| `internal/app/app.go` | Construct 4 repos; pass them everywhere |

### Mobile — files created

| File | Responsibility |
|---|---|
| `src/services/api.ts` | Shared Axios instance + interceptors (token injection, refresh on 401) |

### Mobile — files modified

| File | What changes |
|---|---|
| `src/services/authService.ts` | Imports api from api.ts; no longer owns axios instance |
| `src/services/transactionService.ts` | Imports api from api.ts instead of authService |
| `src/services/budgetService.ts` | Imports api from api.ts instead of authService |
| `src/services/analyticsService.ts` | Imports api from api.ts instead of authService |
| `src/services/plaidService.ts` | Imports api from api.ts; getTransactions takes options object |
| `src/context/AuthContext.tsx` | Import AuthResponse from authService |
| `src/screens/AccountsScreen.tsx` | Remove local Transaction interface; import from transactionService |
| `src/screens/*.tsx` (all) | Replace hardcoded hex colors with COLORS constants |

---

## Task 1: Split model files

**Files:**
- Modify: `internal/models/plaid.go`
- Create: `internal/models/account.go`
- Create: `internal/models/transaction.go`

**Interfaces:**
- Produces: `models.Account`, `models.Transaction`, `models.PlaidItemEntry` — used by all subsequent tasks

- [ ] **Step 1: Create `internal/models/account.go`**

```go
package models

import "time"

type Account struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	PlaidItemID      string    `json:"plaid_item_id"`
	PlaidAccountID   string    `json:"plaid_account_id"`
	Name             string    `json:"name"`
	OfficialName     *string   `json:"official_name,omitempty"`
	Type             string    `json:"type"`
	Subtype          *string   `json:"subtype,omitempty"`
	Mask             *string   `json:"mask,omitempty"`
	CurrentBalance   *float64  `json:"current_balance,omitempty"`
	AvailableBalance *float64  `json:"available_balance,omitempty"`
	CurrencyCode     string    `json:"currency_code"`
	InstitutionName  string    `json:"institution_name"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Create `internal/models/transaction.go`**

```go
package models

import "time"

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

- [ ] **Step 3: Update `internal/models/plaid.go`** — remove `Account` and `Transaction`, add `PlaidItemEntry`

Replace the entire file with:

```go
package models

import "time"

type PlaidItem struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	PlaidItemID     string    `json:"plaid_item_id"`
	InstitutionID   *string   `json:"institution_id,omitempty"`
	InstitutionName *string   `json:"institution_name,omitempty"`
	Cursor          *string   `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PlaidItemEntry is the result type for GetAllItemsForSync.
// It pairs a PlaidItem with its decrypted access token.
type PlaidItemEntry struct {
	Item        *PlaidItem
	AccessToken string
}
```

- [ ] **Step 4: Verify build**

```bash
cd netme-backend && go build ./...
```

Expected: `BUILD OK` (no output)

- [ ] **Step 5: Commit**

```bash
git add netme-backend/internal/models/
git commit -m "refactor: split models/plaid.go into account.go, transaction.go; add PlaidItemEntry"
```

---

## Task 2: Create TransactionRepository

**Files:**
- Create: `internal/repositories/transactions.go`
- Modify: `internal/repositories/budget.go` (lines 46–80, `GetTransactionsForMonth`)

**Interfaces:**
- Consumes: `models.Transaction` (Task 1)
- Produces: `*TransactionRepository` with `UpsertTransaction`, `RemoveTransaction`, `GetTransactionsByUserID`, `GetTransactionByID`, `PatchTransactionCategory`; package-private `scanTransaction(scanner) (*models.Transaction, error)`

- [ ] **Step 1: Create `internal/repositories/transactions.go`**

```go
package repositories

import (
	"database/sql"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// scanner is satisfied by both *sql.Row and *sql.Rows, allowing one
// scan implementation to serve both single-row and multi-row queries.
type scanner interface {
	Scan(dest ...any) error
}

// scanTransaction reads the standard 17-column transaction projection.
// Column order must match every SELECT that uses this helper.
func scanTransaction(s scanner) (*models.Transaction, error) {
	t := &models.Transaction{}
	err := s.Scan(
		&t.ID, &t.UserID, &t.AccountID, &t.PlaidTransactionID,
		&t.Amount, &t.CurrencyCode, &t.Name, &t.MerchantName,
		&t.Date, &t.AuthorizedDate,
		&t.Category, &t.CategoryDetailed, &t.PaymentChannel,
		&t.Pending, &t.CategoryID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

const txnColumns = `id, user_id, account_id, plaid_transaction_id,
	amount, currency_code, name, merchant_name,
	to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
	category, category_detailed, payment_channel,
	pending, category_id, created_at, updated_at`

func (r *TransactionRepository) UpsertTransaction(t *models.Transaction) error {
	_, err := r.db.Exec(
		`INSERT INTO transactions (user_id, account_id, plaid_transaction_id, amount, currency_code, name, merchant_name, date, authorized_date, category, category_detailed, payment_channel, pending)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 ON CONFLICT (plaid_transaction_id) DO UPDATE SET
		   amount = EXCLUDED.amount,
		   name = EXCLUDED.name,
		   merchant_name = EXCLUDED.merchant_name,
		   pending = EXCLUDED.pending,
		   updated_at = now()`,
		t.UserID, t.AccountID, t.PlaidTransactionID, t.Amount, t.CurrencyCode,
		t.Name, t.MerchantName, t.Date, t.AuthorizedDate,
		t.Category, t.CategoryDetailed, t.PaymentChannel, t.Pending,
	)
	return err
}

func (r *TransactionRepository) RemoveTransaction(plaidTransactionID string) error {
	_, err := r.db.Exec(`DELETE FROM transactions WHERE plaid_transaction_id = $1`, plaidTransactionID)
	return err
}

func (r *TransactionRepository) GetTransactionsByUserID(userID, accountID, month string, limit, offset int) ([]*models.Transaction, error) {
	rows, err := r.db.Query(
		`SELECT `+txnColumns+`
		 FROM transactions
		 WHERE user_id = $1
		   AND ($2 = '' OR account_id::text = $2)
		   AND ($3 = '' OR to_char(date, 'YYYY-MM') = $3)
		 ORDER BY date DESC, created_at DESC
		 LIMIT $4 OFFSET $5`, userID, accountID, month, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []*models.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}

func (r *TransactionRepository) GetTransactionByID(userID, id string) (*models.Transaction, error) {
	row := r.db.QueryRow(
		`SELECT `+txnColumns+`
		 FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	t, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TransactionRepository) PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error) {
	row := r.db.QueryRow(
		`UPDATE transactions SET category_id = $1, updated_at = now()
		 WHERE id = $2 AND user_id = $3
		   AND EXISTS (SELECT 1 FROM categories WHERE id = $1 AND user_id = $3)
		 RETURNING `+txnColumns,
		categoryID, txnID, userID)
	t, err := scanTransaction(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}
```

- [ ] **Step 2: Update `GetTransactionsForMonth` in `internal/repositories/budget.go`**

Replace the `GetTransactionsForMonth` function body to use `scanTransaction`:

```go
func (r *BudgetRepository) GetTransactionsForMonth(userID, month string) ([]*models.Transaction, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, account_id, plaid_transaction_id,
		        amount, currency_code, name, merchant_name,
		        to_char(date, 'YYYY-MM-DD'), to_char(authorized_date, 'YYYY-MM-DD'),
		        category, category_detailed, payment_channel,
		        pending, category_id, created_at, updated_at
		 FROM transactions
		 WHERE user_id = $1 AND to_char(date, 'YYYY-MM') = $2 AND pending = false
		 ORDER BY date DESC`, userID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []*models.Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}
```

- [ ] **Step 3: Verify build and tests**

```bash
cd netme-backend && go build ./... && go test ./...
```

Expected: all tests pass, no build errors.

- [ ] **Step 4: Commit**

```bash
git add netme-backend/internal/repositories/transactions.go netme-backend/internal/repositories/budget.go
git commit -m "refactor: extract TransactionRepository with shared scanTransaction helper"
```

---

## Task 3: Create AccountRepository

**Files:**
- Create: `internal/repositories/accounts.go`

**Interfaces:**
- Consumes: `models.Account`, `models.NetWorth` (analytics.go)
- Produces: `*AccountRepository` with `UpsertAccount`, `GetAccountsByUserID`, `GetAccountByPlaidID`, `GetNetWorth`, `TakeNetWorthSnapshot`

- [ ] **Step 1: Create `internal/repositories/accounts.go`**

```go
package repositories

import (
	"database/sql"

	"github.com/vladyslavivchenko/netme/internal/models"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) UpsertAccount(a *models.Account) error {
	_, err := r.db.Exec(
		`INSERT INTO accounts (user_id, plaid_item_id, plaid_account_id, name, official_name, type, subtype, mask, current_balance, available_balance, currency_code)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (plaid_account_id) DO UPDATE SET
		   name = EXCLUDED.name,
		   current_balance = EXCLUDED.current_balance,
		   available_balance = EXCLUDED.available_balance,
		   updated_at = now()`,
		a.UserID, a.PlaidItemID, a.PlaidAccountID, a.Name, a.OfficialName,
		a.Type, a.Subtype, a.Mask, a.CurrentBalance, a.AvailableBalance, a.CurrencyCode,
	)
	return err
}

func (r *AccountRepository) GetAccountsByUserID(userID string) ([]*models.Account, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.user_id, a.plaid_item_id, a.plaid_account_id, a.name, a.official_name,
		        a.type, a.subtype, a.mask, a.current_balance, a.available_balance, a.currency_code,
		        a.created_at, a.updated_at,
		        COALESCE(pi.institution_name, 'Unknown Bank') AS institution_name
		 FROM accounts a
		 LEFT JOIN plaid_items pi ON pi.id = a.plaid_item_id
		 WHERE a.user_id = $1
		 ORDER BY COALESCE(pi.institution_name, ''), a.type, a.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		a := &models.Account{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.PlaidItemID, &a.PlaidAccountID, &a.Name, &a.OfficialName,
			&a.Type, &a.Subtype, &a.Mask, &a.CurrentBalance, &a.AvailableBalance, &a.CurrencyCode,
			&a.CreatedAt, &a.UpdatedAt, &a.InstitutionName); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *AccountRepository) GetAccountByPlaidID(plaidAccountID, userID string) (*models.Account, error) {
	a := &models.Account{}
	err := r.db.QueryRow(
		`SELECT id, user_id, plaid_item_id, plaid_account_id, name, official_name, type, subtype, mask,
		        current_balance, available_balance, currency_code, created_at, updated_at
		 FROM accounts WHERE plaid_account_id = $1 AND user_id = $2`, plaidAccountID, userID,
	).Scan(&a.ID, &a.UserID, &a.PlaidItemID, &a.PlaidAccountID, &a.Name, &a.OfficialName,
		&a.Type, &a.Subtype, &a.Mask, &a.CurrentBalance, &a.AvailableBalance, &a.CurrencyCode,
		&a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *AccountRepository) GetNetWorth(userID string) (*models.NetWorth, error) {
	var assets, liabilities float64
	err := r.db.QueryRow(
		`SELECT
		   COALESCE(SUM(CASE WHEN type IN ('depository','investment') THEN COALESCE(current_balance,0) ELSE 0 END), 0),
		   COALESCE(SUM(CASE WHEN type IN ('credit','loan')           THEN COALESCE(current_balance,0) ELSE 0 END), 0)
		 FROM accounts WHERE user_id = $1`, userID,
	).Scan(&assets, &liabilities)
	if err != nil {
		return nil, err
	}
	return &models.NetWorth{
		Assets:      assets,
		Liabilities: liabilities,
		NetWorth:    assets - liabilities,
	}, nil
}

func (r *AccountRepository) TakeNetWorthSnapshot(userID string) error {
	nw, err := r.GetNetWorth(userID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO net_worth_snapshots (user_id, assets, liabilities, net_worth, recorded_at)
		 VALUES ($1, $2, $3, $4, CURRENT_DATE)
		 ON CONFLICT (user_id, recorded_at) DO UPDATE
		   SET assets=EXCLUDED.assets, liabilities=EXCLUDED.liabilities, net_worth=EXCLUDED.net_worth`,
		userID, nw.Assets, nw.Liabilities, nw.NetWorth,
	)
	return err
}
```

- [ ] **Step 2: Verify build**

```bash
cd netme-backend && go build ./...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add netme-backend/internal/repositories/accounts.go
git commit -m "refactor: extract AccountRepository with net worth methods"
```

---

## Task 4: Create PlaidItemRepository

**Files:**
- Create: `internal/repositories/plaid_items.go`

**Interfaces:**
- Consumes: `models.PlaidItem`, `models.PlaidItemEntry` (Task 1); `crypto` package
- Produces: `*PlaidItemRepository` with all item CRUD + `GetAllUserIDsWithItems`

- [ ] **Step 1: Create `internal/repositories/plaid_items.go`**

```go
package repositories

import (
	"database/sql"

	"github.com/vladyslavivchenko/netme/internal/crypto"
	"github.com/vladyslavivchenko/netme/internal/models"
)

type PlaidItemRepository struct {
	db  *sql.DB
	key []byte // AES-256 key; nil means encryption is disabled
}

func NewPlaidItemRepository(db *sql.DB, key []byte) *PlaidItemRepository {
	return &PlaidItemRepository{db: db, key: key}
}

func (r *PlaidItemRepository) encryptToken(plain string) (string, error) {
	if r.key == nil {
		return plain, nil
	}
	return crypto.Encrypt(r.key, plain)
}

func (r *PlaidItemRepository) decryptToken(stored string) (string, error) {
	if r.key == nil {
		return stored, nil
	}
	return crypto.Decrypt(r.key, stored)
}

func (r *PlaidItemRepository) CreateItem(userID, plaidItemID, accessToken string, institutionID, institutionName *string) (*models.PlaidItem, error) {
	storedToken, err := r.encryptToken(accessToken)
	if err != nil {
		return nil, err
	}
	item := &models.PlaidItem{}
	err = r.db.QueryRow(
		`INSERT INTO plaid_items (user_id, plaid_item_id, access_token, institution_id, institution_name)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, plaid_item_id, institution_id, institution_name, cursor, created_at, updated_at`,
		userID, plaidItemID, storedToken, institutionID, institutionName,
	).Scan(&item.ID, &item.UserID, &item.PlaidItemID, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *PlaidItemRepository) GetItemsByUserID(userID string) ([]*models.PlaidItem, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, plaid_item_id, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.PlaidItem
	for rows.Next() {
		item := &models.PlaidItem{}
		if err := rows.Scan(&item.ID, &item.UserID, &item.PlaidItemID, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PlaidItemRepository) GetItemByID(id string) (*models.PlaidItem, string, error) {
	item := &models.PlaidItem{}
	var storedToken string
	err := r.db.QueryRow(
		`SELECT id, user_id, plaid_item_id, access_token, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE id = $1`, id,
	).Scan(&item.ID, &item.UserID, &item.PlaidItemID, &storedToken, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, "", err
	}
	accessToken, err := r.decryptToken(storedToken)
	return item, accessToken, err
}

func (r *PlaidItemRepository) GetItemByPlaidItemID(plaidItemID string) (*models.PlaidItem, string, error) {
	item := &models.PlaidItem{}
	var storedToken string
	err := r.db.QueryRow(
		`SELECT id, user_id, plaid_item_id, access_token, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE plaid_item_id = $1`, plaidItemID,
	).Scan(&item.ID, &item.UserID, &item.PlaidItemID, &storedToken,
		&item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, "", err
	}
	accessToken, err := r.decryptToken(storedToken)
	return item, accessToken, err
}

func (r *PlaidItemRepository) GetAllItemsForSync(userID string) ([]models.PlaidItemEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, plaid_item_id, access_token, institution_id, institution_name, cursor, created_at, updated_at
		 FROM plaid_items WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.PlaidItemEntry
	for rows.Next() {
		item := &models.PlaidItem{}
		var storedToken string
		if err := rows.Scan(&item.ID, &item.UserID, &item.PlaidItemID, &storedToken, &item.InstitutionID, &item.InstitutionName, &item.Cursor, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		accessToken, err := r.decryptToken(storedToken)
		if err != nil {
			return nil, err
		}
		results = append(results, models.PlaidItemEntry{Item: item, AccessToken: accessToken})
	}
	return results, rows.Err()
}

func (r *PlaidItemRepository) UpdateCursor(itemID, cursor string) error {
	_, err := r.db.Exec(
		`UPDATE plaid_items SET cursor = $1, updated_at = now() WHERE id = $2`,
		cursor, itemID)
	return err
}

func (r *PlaidItemRepository) GetAllUserIDsWithItems() ([]string, error) {
	rows, err := r.db.Query(`SELECT DISTINCT user_id FROM plaid_items ORDER BY user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

- [ ] **Step 2: Verify build**

```bash
cd netme-backend && go build ./...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add netme-backend/internal/repositories/plaid_items.go
git commit -m "refactor: extract PlaidItemRepository"
```

---

## Task 5: Create EventRepository, then delete PlaidRepository

**Files:**
- Create: `internal/repositories/events.go`
- Delete: `internal/repositories/plaid.go`

**Interfaces:**
- Produces: `*EventRepository` with `LogRawEvent`, `PurgeOldRawEvents`

- [ ] **Step 1: Create `internal/repositories/events.go`**

```go
package repositories

import (
	"database/sql"
	"encoding/json"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// LogRawEvent stores any Plaid payload for debugging. userID may be empty for webhook events.
func (r *EventRepository) LogRawEvent(userID, eventType string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	var uid *string
	if userID != "" {
		uid = &userID
	}
	_, _ = r.db.Exec(
		`INSERT INTO plaid_raw_events (user_id, event_type, payload) VALUES ($1, $2, $3)`,
		uid, eventType, string(b),
	)
}

// PurgeOldRawEvents deletes raw event rows older than the given number of days.
func (r *EventRepository) PurgeOldRawEvents(olderThanDays int) (int64, error) {
	res, err := r.db.Exec(
		`DELETE FROM plaid_raw_events WHERE created_at < now() - ($1::int * interval '1 day')`,
		olderThanDays,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

- [ ] **Step 2: Delete `internal/repositories/plaid.go`**

```bash
rm netme-backend/internal/repositories/plaid.go
```

- [ ] **Step 3: Verify build**

```bash
cd netme-backend && go build ./... 2>&1
```

Expected: compile errors referencing `PlaidRepository` in `services/plaid.go`, `handlers/plaid.go`, `handlers/accounts.go`, `handlers/analytics.go`, `jobs/scheduler.go`, `app/app.go`. This is expected — these are fixed in Tasks 7–10.

Since the build is temporarily broken, do NOT commit yet. Proceed to Task 6 immediately.

---

## Task 6: Update interfaces.go

**Files:**
- Modify: `internal/repositories/interfaces.go`

**Interfaces:**
- Produces: all repository interfaces used by handlers and services

- [ ] **Step 1: Replace `internal/repositories/interfaces.go`**

```go
package repositories

import (
	"time"

	"github.com/vladyslavivchenko/netme/internal/models"
)

// ── Auth ──────────────────────────────────────────────────────────────────────

type UserRepo interface {
	CreateUser(email, passwordHash string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateLastLogin(userID string) error
	FindOrCreateGoogleUser(googleID, email string) (*models.User, error)
	DeleteUser(userID string) error
}

type TokenRepo interface {
	CreateRefreshToken(userID, token string, expiresAt time.Time) (*models.RefreshToken, error)
	GetRefreshToken(token string) (*models.RefreshToken, error)
	RevokeRefreshToken(token, userID string) error
	RevokeAllUserTokens(userID string) error
	IsRefreshTokenValid(token string) (bool, error)
}

// ── Accounts ──────────────────────────────────────────────────────────────────

// AccountLister is the read-only subset needed by AccountsHandler.
type AccountLister interface {
	GetAccountsByUserID(userID string) ([]*models.Account, error)
}

// ── Plaid Items ───────────────────────────────────────────────────────────────

// PlaidItemGetter is the subset of PlaidItemRepository needed by PlaidHandler.
type PlaidItemGetter interface {
	GetItemsByUserID(userID string) ([]*models.PlaidItem, error)
	GetItemByPlaidItemID(plaidItemID string) (*models.PlaidItem, string, error)
}

// ── Transactions ──────────────────────────────────────────────────────────────

// TxnRepo is the subset of TransactionRepository needed by TransactionsHandler.
// Moved here from handlers/transactions.go.
type TxnRepo interface {
	GetTransactionsByUserID(userID, accountID, month string, limit, offset int) ([]*models.Transaction, error)
	GetTransactionByID(userID, id string) (*models.Transaction, error)
	PatchTransactionCategory(userID, txnID, categoryID string) (*models.Transaction, error)
}

// ── Rules ─────────────────────────────────────────────────────────────────────

// RulesRepo is the full interface for RulesRepository, used by RulesHandler.
// Moved here from handlers/rules.go.
type RulesRepo interface {
	Upsert(userID, normalizedMerchant, categoryID string) (*models.CategoryRule, error)
	ApplyToPast(userID, normalizedMerchant, categoryID string) (int64, error)
	List(userID string) ([]*models.CategoryRule, error)
	Delete(userID, id string) error
	ApplyCategoryRules(userID string) error
}

// ── Analytics ─────────────────────────────────────────────────────────────────

// NetWorthReader is the subset of AccountRepository needed by AnalyticsHandler.
type NetWorthReader interface {
	GetNetWorth(userID string) (*models.NetWorth, error)
}

// ── Events ────────────────────────────────────────────────────────────────────

// EventLogger is the subset of EventRepository needed by PlaidHandler.
type EventLogger interface {
	LogRawEvent(userID, eventType string, payload any)
}
```

- [ ] **Step 2: Verify compile (still broken due to services/handlers not updated)**

```bash
cd netme-backend && go build ./... 2>&1 | grep -v "PlaidRepository" | head -20
```

Expected: errors only in files not yet updated. Proceed to Tasks 7–10 without committing.

---

## Task 7: Update handlers

**Files:**
- Modify: `internal/handlers/accounts.go`
- Modify: `internal/handlers/analytics.go`
- Modify: `internal/handlers/plaid.go`
- Modify: `internal/handlers/transactions.go`
- Modify: `internal/handlers/rules.go`

- [ ] **Step 1: Update `internal/handlers/accounts.go`**

Replace entire file:

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type AccountsHandler struct {
	repo repositories.AccountLister
}

func NewAccountsHandler(repo repositories.AccountLister) *AccountsHandler {
	return &AccountsHandler{repo: repo}
}

func RegisterAccountRoutes(r *gin.RouterGroup, repo repositories.AccountLister) {
	h := NewAccountsHandler(repo)
	r.Group("/accounts").GET("", h.ListAccounts)
}

func (h *AccountsHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.repo.GetAccountsByUserID(uid(c))
	if err != nil {
		dbErr(c, "failed to load accounts")
		return
	}
	if accounts == nil {
		accounts = []*models.Account{}
	}
	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}
```

- [ ] **Step 2: Update `internal/handlers/analytics.go`**

Replace entire file:

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type AnalyticsHandler struct {
	netWorth repositories.NetWorthReader
	budget   *repositories.BudgetRepository
}

func NewAnalyticsHandler(netWorth repositories.NetWorthReader, budget *repositories.BudgetRepository) *AnalyticsHandler {
	return &AnalyticsHandler{netWorth: netWorth, budget: budget}
}

func RegisterAnalyticsRoutes(r *gin.RouterGroup, netWorth repositories.NetWorthReader, budget *repositories.BudgetRepository) {
	h := NewAnalyticsHandler(netWorth, budget)
	r.GET("/analytics/overview", h.Overview)
}

func (h *AnalyticsHandler) Overview(c *gin.Context) {
	userID := uid(c)
	month := currentMonth()

	nw, err := h.netWorth.GetNetWorth(userID)
	if err != nil {
		dbErr(c, err.Error())
		return
	}

	history, err := h.budget.GetMonthlyHistory(userID, 6)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	if history == nil {
		history = []models.MonthlyTotal{}
	}

	topCats, err := h.budget.GetTopCategories(userID, month, 5)
	if err != nil {
		dbErr(c, err.Error())
		return
	}
	if topCats == nil {
		topCats = []models.TopCategory{}
	}

	c.JSON(http.StatusOK, models.AnalyticsOverview{
		NetWorth:      *nw,
		MonthlyTotals: history,
		TopCategories: topCats,
	})
}
```

- [ ] **Step 3: Update `internal/handlers/plaid.go`** — change struct fields to use interfaces

Replace the struct, constructor, and registration function at the top of the file:

```go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type PlaidHandler struct {
	plaidSvc  *services.PlaidService
	itemRepo  repositories.PlaidItemGetter
	eventRepo repositories.EventLogger
}

func NewPlaidHandler(svc *services.PlaidService, itemRepo repositories.PlaidItemGetter, eventRepo repositories.EventLogger) *PlaidHandler {
	return &PlaidHandler{plaidSvc: svc, itemRepo: itemRepo, eventRepo: eventRepo}
}

func RegisterPlaidRoutes(r *gin.RouterGroup, public *gin.RouterGroup, svc *services.PlaidService, itemRepo repositories.PlaidItemGetter, eventRepo repositories.EventLogger) {
	h := NewPlaidHandler(svc, itemRepo, eventRepo)
	plaid := r.Group("/plaid")
	{
		plaid.POST("/link-token", h.CreateLinkToken)
		plaid.POST("/exchange", h.ExchangeToken)
		plaid.POST("/sync", h.SyncTransactions)
		plaid.GET("/items", h.ListItems)
	}
	public.GET("/plaid/link-page", h.LinkPage)
	public.POST("/plaid/webhook", h.Webhook)
}
```

Then update the handler methods that reference `h.plaidRepo` to use `h.itemRepo` or `h.eventRepo`:

In `ListItems`, change `h.plaidRepo.GetItemsByUserID` → `h.itemRepo.GetItemsByUserID`.

In `Webhook`, change:
- `h.plaidRepo.LogRawEvent(...)` → `h.eventRepo.LogRawEvent(...)`
- In the goroutine: `h.plaidRepo.GetItemByPlaidItemID(...)` → `h.itemRepo.GetItemByPlaidItemID(...)`

The full updated `ListItems` and `Webhook` methods (all other methods are unchanged — they only use `h.plaidSvc`):

```go
func (h *PlaidHandler) ListItems(c *gin.Context) {
	items, err := h.itemRepo.GetItemsByUserID(uid(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("database_error", "failed to load items"))
		return
	}
	if items == nil {
		items = []*models.PlaidItem{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *PlaidHandler) Webhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := h.plaidSvc.WebhookVerifier.Verify(
		c.GetHeader("Plaid-Verification"), body,
	); err != nil {
		c.JSON(http.StatusUnauthorized, errResp("invalid_signature", err.Error()))
		return
	}

	var payload struct {
		WebhookType string `json:"webhook_type"`
		WebhookCode string `json:"webhook_code"`
		ItemID      string `json:"item_id"`
		Error       *struct {
			ErrorType    string `json:"error_type"`
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	h.eventRepo.LogRawEvent("", "webhook_received", map[string]any{
		"type": payload.WebhookType,
		"code": payload.WebhookCode,
		"item": payload.ItemID,
	})

	c.Status(http.StatusOK)

	go func() {
		ctx := context.Background()
		switch payload.WebhookType {
		case "TRANSACTIONS":
			switch payload.WebhookCode {
			case "SYNC_UPDATES_AVAILABLE", "DEFAULT_UPDATE", "INITIAL_UPDATE", "RECURRING_TRANSACTIONS_UPDATE":
				item, _, err := h.itemRepo.GetItemByPlaidItemID(payload.ItemID)
				if err != nil {
					return
				}
				_, _ = h.plaidSvc.SyncItem(ctx, item.UserID, item.ID)
			}
		case "ITEM":
			if payload.Error != nil {
				h.eventRepo.LogRawEvent("", "webhook_item_error", map[string]any{
					"item_id":       payload.ItemID,
					"error_type":    payload.Error.ErrorType,
					"error_code":    payload.Error.ErrorCode,
					"error_message": payload.Error.ErrorMessage,
				})
			}
		}
	}()
}
```

- [ ] **Step 4: Update `internal/handlers/transactions.go`** — delete local `TxnRepo`, import from repositories

Replace the local interface definition and the handler struct/constructor:

```go
package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/models"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type TransactionsHandler struct {
	repo repositories.TxnRepo
}

func NewTransactionsHandler(repo repositories.TxnRepo) *TransactionsHandler {
	return &TransactionsHandler{repo: repo}
}

func RegisterTransactionRoutes(r *gin.RouterGroup, repo repositories.TxnRepo) {
	h := NewTransactionsHandler(repo)
	txns := r.Group("/transactions")
	{
		txns.GET("", h.ListTransactions)
		txns.GET("/:id", h.GetTransaction)
		txns.PATCH("/:id", h.PatchTransaction)
	}
}
```

Keep the handler method bodies (`ListTransactions`, `GetTransaction`, `PatchTransaction`) exactly as they are — no changes needed there.

- [ ] **Step 5: Update `internal/handlers/rules.go`** — delete local `RulesRepo`, import from repositories

Replace the local interface definition and the handler struct/constructor:

```go
package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/repositories"
)

type RulesHandler struct {
	repo repositories.RulesRepo
}

func NewRulesHandler(repo repositories.RulesRepo) *RulesHandler {
	return &RulesHandler{repo: repo}
}

func RegisterRulesRoutes(r *gin.RouterGroup, repo repositories.RulesRepo) {
	h := NewRulesHandler(repo)
	rules := r.Group("/rules")
	{
		rules.POST("", h.CreateRule)
		rules.GET("", h.ListRules)
		rules.DELETE("/:id", h.DeleteRule)
	}
}
```

Keep the handler method bodies exactly as they are.

- [ ] **Step 6: Verify build**

```bash
cd netme-backend && go build ./... 2>&1
```

Expected: remaining errors only in `services/plaid.go`, `jobs/scheduler.go`, `app/app.go`. Proceed without committing.

---

## Task 8: Update PlaidService

**Files:**
- Modify: `internal/services/plaid.go`

- [ ] **Step 1: Replace the `PlaidService` struct and constructor**

The struct changes from one `*PlaidRepository` to four focused repos:

```go
type PlaidService struct {
	client          *plaid.APIClient
	itemRepo        *repositories.PlaidItemRepository
	acctRepo        *repositories.AccountRepository
	txnRepo         *repositories.TransactionRepository
	eventRepo       *repositories.EventRepository
	rulesRepo       *repositories.RulesRepository
	WebhookVerifier *WebhookVerifier
}

func NewPlaidService(
	clientID, secret, env string,
	itemRepo *repositories.PlaidItemRepository,
	acctRepo *repositories.AccountRepository,
	txnRepo *repositories.TransactionRepository,
	eventRepo *repositories.EventRepository,
	rulesRepo *repositories.RulesRepository,
) *PlaidService {
	cfg := plaid.NewConfiguration()
	cfg.AddDefaultHeader("PLAID-CLIENT-ID", clientID)
	cfg.AddDefaultHeader("PLAID-SECRET", secret)

	if env == "production" {
		cfg.UseEnvironment(plaid.Production)
	} else {
		cfg.UseEnvironment(plaid.Sandbox)
	}

	client := plaid.NewAPIClient(cfg)
	return &PlaidService{
		client:          client,
		itemRepo:        itemRepo,
		acctRepo:        acctRepo,
		txnRepo:         txnRepo,
		eventRepo:       eventRepo,
		rulesRepo:       rulesRepo,
		WebhookVerifier: NewWebhookVerifier(client, env),
	}
}
```

- [ ] **Step 2: Update method bodies** — replace all `s.plaidRepo.` calls

`ExchangeAndStore` — replace `s.plaidRepo` with the appropriate split repo:
- `s.plaidRepo.CreateItem(...)` → `s.itemRepo.CreateItem(...)`
- `s.plaidRepo.LogRawEvent(...)` → `s.eventRepo.LogRawEvent(...)`
- `s.plaidRepo.UpsertAccount(...)` → `s.acctRepo.UpsertAccount(...)`

`SyncItem`:
- `s.plaidRepo.GetItemByID(...)` → `s.itemRepo.GetItemByID(...)`
- `s.plaidRepo.UpdateCursor(...)` → `s.itemRepo.UpdateCursor(...)`

`SyncForUser`:
- `s.plaidRepo.GetAllItemsForSync(...)` → `s.itemRepo.GetAllItemsForSync(...)`
- The return type changes from `[]struct{ Item *models.PlaidItem; AccessToken string }` to `[]models.PlaidItemEntry`; update the range variable accordingly: `for _, entry := range items` — `entry.Item` and `entry.AccessToken` field names stay the same.
- `s.plaidRepo.UpdateCursor(...)` → `s.itemRepo.UpdateCursor(...)`

`drainPages`:
- `s.plaidRepo.LogRawEvent(...)` → `s.eventRepo.LogRawEvent(...)`
- `s.plaidRepo.GetAccountByPlaidID(...)` → `s.acctRepo.GetAccountByPlaidID(...)`
- `s.plaidRepo.UpsertTransaction(...)` → `s.txnRepo.UpsertTransaction(...)`
- `s.plaidRepo.RemoveTransaction(...)` → `s.txnRepo.RemoveTransaction(...)`

`RevokeAllItems`:
- `s.plaidRepo.GetAllItemsForSync(...)` → `s.itemRepo.GetAllItemsForSync(...)`

- [ ] **Step 3: Verify build**

```bash
cd netme-backend && go build ./... 2>&1
```

Expected: remaining errors only in `jobs/scheduler.go` and `app/app.go`.

---

## Task 9: Update Scheduler

**Files:**
- Modify: `internal/jobs/scheduler.go`

- [ ] **Step 1: Replace `internal/jobs/scheduler.go`**

```go
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

// Scheduler runs periodic background jobs using in-process tickers.
// Designed for single-instance MVP deployments; replace with a distributed
// queue (e.g. Asynq) when running multiple replicas.
type Scheduler struct {
	plaidSvc  *services.PlaidService
	itemRepo  *repositories.PlaidItemRepository
	acctRepo  *repositories.AccountRepository
	eventRepo *repositories.EventRepository
	log       *slog.Logger
}

func NewScheduler(
	plaidSvc *services.PlaidService,
	itemRepo *repositories.PlaidItemRepository,
	acctRepo *repositories.AccountRepository,
	eventRepo *repositories.EventRepository,
	log *slog.Logger,
) *Scheduler {
	return &Scheduler{
		plaidSvc:  plaidSvc,
		itemRepo:  itemRepo,
		acctRepo:  acctRepo,
		eventRepo: eventRepo,
		log:       log,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	syncTicker := time.NewTicker(24 * time.Hour)
	snapshotTicker := time.NewTicker(24 * time.Hour)
	purgeTicker := time.NewTicker(7 * 24 * time.Hour)

	go s.runSync(ctx)
	go s.runNetWorthSnapshots(ctx)

	for {
		select {
		case <-ctx.Done():
			syncTicker.Stop()
			snapshotTicker.Stop()
			purgeTicker.Stop()
			s.log.Info("scheduler stopped")
			return
		case <-syncTicker.C:
			go s.runSync(ctx)
		case <-snapshotTicker.C:
			go s.runNetWorthSnapshots(ctx)
		case <-purgeTicker.C:
			go s.runDataRetentionPurge()
		}
	}
}

func (s *Scheduler) runSync(ctx context.Context) {
	userIDs, err := s.itemRepo.GetAllUserIDsWithItems()
	if err != nil {
		s.log.Error("daily sync: failed to load users", "err", err)
		return
	}
	s.log.Info("daily sync: starting", "users", len(userIDs))

	added := 0
	for _, uid := range userIDs {
		n, err := s.plaidSvc.SyncForUser(ctx, uid)
		if err != nil {
			s.log.Error("daily sync: user failed", "user_id", uid, "err", err)
			continue
		}
		added += n
	}
	s.log.Info("daily sync: done", "users", len(userIDs), "transactions_added", added)
}

func (s *Scheduler) runDataRetentionPurge() {
	n, err := s.eventRepo.PurgeOldRawEvents(90)
	if err != nil {
		s.log.Error("data retention purge: failed", "err", err)
		return
	}
	s.log.Info("data retention purge: done", "rows_deleted", n)
}

func (s *Scheduler) runNetWorthSnapshots(ctx context.Context) {
	userIDs, err := s.itemRepo.GetAllUserIDsWithItems()
	if err != nil {
		s.log.Error("net worth snapshot: failed to load users", "err", err)
		return
	}
	s.log.Info("net worth snapshot: starting", "users", len(userIDs))

	failed := 0
	for _, uid := range userIDs {
		if err := s.acctRepo.TakeNetWorthSnapshot(uid); err != nil {
			s.log.Error("net worth snapshot: user failed", "user_id", uid, "err", err)
			failed++
		}
	}
	s.log.Info("net worth snapshot: done", "users", len(userIDs), "failed", failed)
}
```

---

## Task 10: Update app.go wiring + final backend commit

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Replace repository construction and wiring in `internal/app/app.go`**

Replace the repository construction block and all downstream usages. The new `New()` function body (keeping the existing logger, db, crypto, JWT, and Google setup unchanged):

```go
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

	plaidKey, err := crypto.ParseKey(os.Getenv("PLAID_TOKEN_ENCRYPTION_KEY"))
	if err != nil {
		return nil, fmt.Errorf("plaid encryption key: %w", err)
	}
	if plaidKey == nil {
		log.Warn("PLAID_TOKEN_ENCRYPTION_KEY not set — Plaid access tokens stored unencrypted")
	}

	userRepo := repositories.NewUserRepository(database)
	tokenRepo := repositories.NewTokenRepository(database)
	itemRepo := repositories.NewPlaidItemRepository(database, plaidKey)
	acctRepo := repositories.NewAccountRepository(database)
	txnRepo := repositories.NewTransactionRepository(database)
	eventRepo := repositories.NewEventRepository(database)
	budgetRepo := repositories.NewBudgetRepository(database)
	rulesRepo := repositories.NewRulesRepository(database)

	jwtSvc := services.NewJWTService(jwtSecret)

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleVerifier := services.NewGoogleIDTokenVerifier(googleClientID)

	plaidSvc := services.NewPlaidService(
		os.Getenv("PLAID_CLIENT_ID"),
		os.Getenv("PLAID_SECRET"),
		os.Getenv("PLAID_ENV"),
		itemRepo,
		acctRepo,
		txnRepo,
		eventRepo,
		rulesRepo,
	)

	authSvc := services.NewAuthService(userRepo, tokenRepo, jwtSvc, googleVerifier)

	authHandler := handlers.NewAuthHandler(authSvc)
	usersHandler := handlers.NewUsersHandler(userRepo, plaidSvc)

	router := gin.Default()
	router.Use(middleware.HTTPSRedirect(os.Getenv("API_ENV")))
	router.Use(middleware.CORSMiddleware())
	router.GET("/healthz", handlers.HealthHandler())

	v1 := router.Group("/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", middleware.RateLimiter(rate.Every(12*time.Second), 5), authHandler.Register)
		auth.POST("/login", middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.Login)
		auth.POST("/refresh", middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.Refresh)
		auth.POST("/google", middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.GoogleAuth)
	}

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(jwtSvc))
	{
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/me", usersHandler.GetMe)
		protected.DELETE("/me", usersHandler.DeleteMe)
		handlers.RegisterAccountRoutes(protected, acctRepo)
		handlers.RegisterTransactionRoutes(protected, txnRepo)
		handlers.RegisterPlaidRoutes(protected, v1, plaidSvc, itemRepo, eventRepo)
		handlers.RegisterBudgetRoutes(protected, budgetRepo)
		handlers.RegisterRulesRoutes(protected, rulesRepo)
		handlers.RegisterAnalyticsRoutes(protected, acctRepo, budgetRepo)
	}

	scheduler := jobs.NewScheduler(plaidSvc, itemRepo, acctRepo, eventRepo, log)
	go scheduler.Start(context.Background())

	return &App{
		db:     database,
		router: router,
		log:    log,
	}, nil
}
```

- [ ] **Step 2: Verify build and tests**

```bash
cd netme-backend && go build ./... && go test ./...
```

Expected: `BUILD OK`, all tests pass (handlers, middleware, services).

- [ ] **Step 3: Commit all backend changes**

```bash
cd netme-backend
git add internal/repositories/events.go \
        internal/repositories/plaid_items.go \
        internal/repositories/accounts.go \
        internal/repositories/transactions.go \
        internal/repositories/interfaces.go \
        internal/handlers/accounts.go \
        internal/handlers/analytics.go \
        internal/handlers/plaid.go \
        internal/handlers/transactions.go \
        internal/handlers/rules.go \
        internal/services/plaid.go \
        internal/jobs/scheduler.go \
        internal/app/app.go
# Note: plaid.go deletion needs git rm
git rm internal/repositories/plaid.go
git commit -m "refactor: split PlaidRepository into focused repos; add repository interfaces; update all wiring"
```

---

## Task 11: Extract shared Axios instance (mobile)

**Files:**
- Create: `netme-mobile/src/services/api.ts`
- Modify: `netme-mobile/src/services/authService.ts`
- Modify: `netme-mobile/src/services/transactionService.ts`
- Modify: `netme-mobile/src/services/budgetService.ts`
- Modify: `netme-mobile/src/services/analyticsService.ts`
- Modify: `netme-mobile/src/services/plaidService.ts`

- [ ] **Step 1: Create `netme-mobile/src/services/api.ts`**

```typescript
import axios, { AxiosInstance } from 'axios';
import { secureStorage } from './secureStorage';

const API_URL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080/v1';

export const api: AxiosInstance = axios.create({
  baseURL: API_URL,
  timeout: 30000,
});

let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

function onRefreshed(token: string) {
  refreshSubscribers.forEach(cb => cb(token));
  refreshSubscribers = [];
}

api.interceptors.request.use(
  async config => {
    const token = await secureStorage.getAccessToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  error => Promise.reject(error),
);

api.interceptors.response.use(
  response => response,
  async error => {
    const original = error.config;

    if (
      error.response?.status === 401 &&
      !original._retry &&
      original.url !== '/auth/refresh'
    ) {
      original._retry = true;

      if (!isRefreshing) {
        isRefreshing = true;
        try {
          const refreshToken = await secureStorage.getRefreshToken();
          if (!refreshToken) throw new Error('No refresh token');

          // Import inline to avoid circular dependency: api.ts ← authService.ts → api.ts
          const { data } = await axios.post<{ access_token: string; refresh_token: string }>(
            `${API_URL}/auth/refresh`,
            { refresh_token: refreshToken },
          );

          await secureStorage.saveAccessToken(data.access_token);
          await secureStorage.saveRefreshToken(data.refresh_token);

          original.headers.Authorization = `Bearer ${data.access_token}`;
          isRefreshing = false;
          onRefreshed(data.access_token);

          return api(original);
        } catch (refreshError) {
          isRefreshing = false;
          if (axios.isAxiosError(refreshError) && refreshError.response) {
            await secureStorage.clearAll();
          }
          throw refreshError;
        }
      } else {
        return new Promise(resolve => {
          refreshSubscribers.push(token => {
            original.headers.Authorization = `Bearer ${token}`;
            resolve(api(original));
          });
        });
      }
    }

    return Promise.reject(error);
  },
);
```

- [ ] **Step 2: Update `netme-mobile/src/services/authService.ts`**

Remove the `axios.create` call, interceptor setup, and the `api` instance. Import from `api.ts` instead. The `AuthService` class keeps all its methods but no longer owns the HTTP client:

```typescript
import { AuthResponse } from './authService'; // removed — AuthResponse is defined here
```

Replace the full file:

```typescript
import { api } from './api';

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: string;
    email: string;
    auth_provider: string;
    auth_provider_user_id?: string;
    created_at: string;
    updated_at: string;
  };
}

class AuthService {
  async register(email: string, password: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/register', { email, password });
    return data;
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/login', { email, password });
    return data;
  }

  async loginWithGoogle(googleIDToken: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/google', { id_token: googleIDToken });
    return data;
  }

  async refresh(refreshToken: string): Promise<AuthResponse> {
    const { data } = await api.post<AuthResponse>('/auth/refresh', { refresh_token: refreshToken });
    return data;
  }

  async logout(refreshToken: string, accessToken: string): Promise<void> {
    try {
      await api.post(
        '/auth/logout',
        { refresh_token: refreshToken },
        { headers: { Authorization: `Bearer ${accessToken}` } },
      );
    } catch (error) {
      console.error('Logout API call failed:', error);
    }
  }

  async deleteAccount(): Promise<void> {
    await api.delete('/me');
  }
}

export const authService = new AuthService();
```

- [ ] **Step 3: Update `netme-mobile/src/services/transactionService.ts`**

Replace `import { authService } from './authService'; const api = authService.api;` with:

```typescript
import { api } from './api';
```

Remove the `const api = authService.api;` line. No other changes.

- [ ] **Step 4: Update `netme-mobile/src/services/budgetService.ts`**

Same change as Step 3: replace the authService import + api assignment with `import { api } from './api';`.

- [ ] **Step 5: Update `netme-mobile/src/services/analyticsService.ts`**

Same change: `import { api } from './api';`, remove `const api = authService.api;`.

- [ ] **Step 6: Update `netme-mobile/src/services/plaidService.ts`**

Same change: `import { api } from './api';`, remove `const api = authService.api;`.

- [ ] **Step 7: Commit**

```bash
git add netme-mobile/src/services/api.ts \
        netme-mobile/src/services/authService.ts \
        netme-mobile/src/services/transactionService.ts \
        netme-mobile/src/services/budgetService.ts \
        netme-mobile/src/services/analyticsService.ts \
        netme-mobile/src/services/plaidService.ts
git commit -m "refactor: extract shared Axios instance into api.ts; remove authService.api coupling"
```

---

## Task 12: Fix AuthResponse import + remove duplicate Transaction type

**Files:**
- Modify: `netme-mobile/src/context/AuthContext.tsx`
- Modify: `netme-mobile/src/screens/AccountsScreen.tsx`

- [ ] **Step 1: Update `AuthContext.tsx`** — add `AuthResponse` import

Change the import line at the top from:

```typescript
import { authService } from '../services/authService';
```

to:

```typescript
import { authService, AuthResponse } from '../services/authService';
```

No other changes to the file.

- [ ] **Step 2: Update `AccountsScreen.tsx`** — delete local `Transaction` interface, import from transactionService

Delete this block from the Types section (lines ~34–42):

```typescript
interface Transaction {
  id: string;
  name: string;
  merchant_name?: string;
  amount: number;
  currency_code: string;
  date: string;
  category?: string;
  pending: boolean;
}
```

Add to the import block at the top of the file:

```typescript
import { Transaction } from '../services/transactionService';
```

- [ ] **Step 3: Commit**

```bash
git add netme-mobile/src/context/AuthContext.tsx \
        netme-mobile/src/screens/AccountsScreen.tsx
git commit -m "fix: import AuthResponse in AuthContext; remove duplicate Transaction type from AccountsScreen"
```

---

## Task 13: Replace hardcoded colors with COLORS constants

**Files:**
- Modify: all files in `netme-mobile/src/screens/` that contain the target hex values

The mapping to apply:

| Hex literal | COLORS constant | Notes |
|---|---|---|
| `'#2dd4a7'` | `COLORS.teal` | Primary brand color |
| `'#fca5a5'` | `COLORS.red` | Debt / over-budget |
| `'#4ade80'` | `COLORS.green` | Income / savings |
| `'rgba(255,255,255,0.4)'` | `COLORS.muted` | Subtitle text |
| `'rgba(255,255,255,0.1)'` | `COLORS.mutedLight` | Hairline separators |
| `'#0f172a'` | `COLORS.bg` | Background (rare explicit use) |
| `'#1e3a5f'` | `COLORS.navy` | Light-background text |

Add `COLORS` to the theme import in every file that uses it. Files already importing `GLASS` just extend the import: `import { GLASS, COLORS } from '../styles/theme';`.

- [ ] **Step 1: Update `HomeScreen.tsx`** — add COLORS import, replace hex literals

Import:
```typescript
import { GLASS, COLORS } from '../styles/theme';
```

In the styles and JSX, replace:
- `color: '#2dd4a7'` → `color: COLORS.teal`
- `tintColor="#2dd4a7"` → `tintColor={COLORS.teal}`
- `color: '#fca5a5'` → `color: COLORS.red`
- `color: '#4ade80'` → `color: COLORS.green`
- `color: 'rgba(255,255,255,0.4)'` → `color: COLORS.muted`
- `backgroundColor: 'rgba(255,255,255,0.1)'` → `backgroundColor: COLORS.mutedLight`
- In the ActivityIndicator: `color="#2dd4a7"` → `color={COLORS.teal}`

- [ ] **Step 2: Update `TransactionsScreen.tsx`**

Same import addition. Replace:
- `'#2dd4a7'` → `COLORS.teal` (tintColor and ActivityIndicator color)
- `'rgba(255,255,255,0.4)'` → `COLORS.muted` (meta text)

- [ ] **Step 3: Update `AccountsScreen.tsx`**

Same import addition. Replace:
- `color="#2dd4a7"` → `color={COLORS.teal}` (ActivityIndicator, RefreshControl)
- `tintColor="..."` → use COLORS constant
- `'#fca5a5'` → `COLORS.red`
- `'rgba(255,255,255,0.4)'` → `COLORS.muted`
- `'rgba(255,255,255,0.1)'` → `COLORS.mutedLight`
- `'#1e3a5f'` in the transaction modal `t` styles → `COLORS.navy`

- [ ] **Step 4: Update `BudgetScreen.tsx`, `ProfileScreen.tsx`, `SettingsScreen.tsx`, `TransactionDetailScreen.tsx`, `LoginScreen.tsx`, `RegisterScreen.tsx`**

For each file: add `COLORS` to the theme import (or add the import if GLASS is not currently imported), then apply the same hex → constant substitutions as above for any occurrences in that file.

- [ ] **Step 5: Commit**

```bash
git add netme-mobile/src/screens/ netme-mobile/src/styles/
git commit -m "refactor: replace hardcoded hex colors with COLORS constants from theme"
```

---

## Task 14: plaidService.getTransactions options object

**Files:**
- Modify: `netme-mobile/src/services/plaidService.ts`
- Modify: `netme-mobile/src/screens/HomeScreen.tsx`
- Modify: `netme-mobile/src/screens/TransactionsScreen.tsx`
- Modify: `netme-mobile/src/screens/AccountsScreen.tsx`

- [ ] **Step 1: Update `plaidService.ts`** — change `getTransactions` signature

Replace:

```typescript
getTransactions: async (limit = 50, offset = 0, accountId = '', month = '') => {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (accountId) params.set('account_id', accountId);
  if (month) params.set('month', month);
  const { data } = await api.get(`/transactions?${params}`);
  return data.transactions || [];
},
```

With:

```typescript
getTransactions: async (opts: { limit?: number; offset?: number; accountId?: string; month?: string } = {}) => {
  const { limit = 50, offset = 0, accountId = '', month = '' } = opts;
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (accountId) params.set('account_id', accountId);
  if (month) params.set('month', month);
  const { data } = await api.get(`/transactions?${params}`);
  return data.transactions || [];
},
```

- [ ] **Step 2: Update call sites**

In `HomeScreen.tsx`, change:
```typescript
plaidService.getTransactions(5),
```
to:
```typescript
plaidService.getTransactions({ limit: 5 }),
```

In `TransactionsScreen.tsx`, change:
```typescript
const page = await plaidService.getTransactions(PAGE_SIZE, offset, '', m);
```
to:
```typescript
const page = await plaidService.getTransactions({ limit: PAGE_SIZE, offset, month: m });
```

In `AccountsScreen.tsx` (inside `AccountTransactionsModal`), change:
```typescript
plaidService.getTransactions(100, 0, account.id)
```
to:
```typescript
plaidService.getTransactions({ limit: 100, accountId: account.id })
```

- [ ] **Step 3: Commit**

```bash
git add netme-mobile/src/services/plaidService.ts \
        netme-mobile/src/screens/HomeScreen.tsx \
        netme-mobile/src/screens/TransactionsScreen.tsx \
        netme-mobile/src/screens/AccountsScreen.tsx
git commit -m "refactor: plaidService.getTransactions takes options object instead of positional params"
```

---

## Final verification

- [ ] **Backend build and tests**

```bash
cd netme-backend && go build ./... && go test ./...
```

Expected: all tests pass, no build errors.

- [ ] **Mobile type check**

```bash
cd netme-mobile && npx tsc --noEmit 2>&1
```

Expected: only the pre-existing `tsconfig.json(2,3): error TS5098` error. No new errors.
