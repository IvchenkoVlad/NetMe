# Aggressive Refactor — Design Spec

Date: 2026-07-14
Scope: Backend (Go) + Mobile (React Native / TypeScript)
Tests: follow-up only

---

## Goals

Improve code quality, readability, maintainability, and long-term extensibility without changing external behavior. No new dependencies. No new features.

---

## Backend

### 1. Split `PlaidRepository` god object

`internal/repositories/plaid.go` (365 lines, 4 distinct responsibilities) splits into:

| New file | Responsibility |
|---|---|
| `repositories/plaid_items.go` | Plaid item CRUD: `CreateItem`, `GetItemByID`, `GetItemByPlaidItemID`, `GetItemsByUserID`, `GetAllItemsForSync`, `UpdateCursor` |
| `repositories/accounts.go` | Account CRUD: `UpsertAccount`, `GetAccountsByUserID`, `GetAccountByPlaidID` |
| `repositories/transactions.go` | Transaction CRUD: `UpsertTransaction`, `RemoveTransaction`, `GetTransactionsByUserID`, `GetTransactionByID`, `PatchTransactionCategory` |
| `repositories/events.go` | Observability + analytics: `LogRawEvent`, `PurgeOldRawEvents`, `TakeNetWorthSnapshot`, `GetNetWorth`, `GetAllUserIDsWithItems` |

Each file gets its own `*XxxRepository` struct backed by the same `*sql.DB`. The `PlaidRepository` struct and constructor are removed; `NewPlaidRepository` is replaced by four focused constructors.

The anonymous inline `[]struct{ Item *models.PlaidItem; AccessToken string }` return type of `GetAllItemsForSync` becomes a named `PlaidItemEntry` type in `models/plaid.go`.

The 17-column transaction scan (currently copy-pasted in 4 places across `plaid.go` and `repositories/budget.go`) is extracted into a package-private `scanTransaction(*sql.Row) (*models.Transaction, error)` helper in `transactions.go`. The budget repo's `GetTransactionsForMonth` switches to a `*sql.Rows` variant of the same helper.

The encryption helpers (`encryptToken`, `decryptToken`) move to `plaid_items.go` and are kept private — they are only needed there.

### 2. Repository interfaces

`repositories/interfaces.go` currently only defines `UserRepo` and `TokenRepo`. It grows to include:

```
PlaidItemReader   — read-only subset used by Webhook handler
PlaidItemWriter   — full CRUD used by PlaidService
AccountReader     — used by AccountsHandler
AccountWriter     — used by PlaidService
TransactionReader — used by TransactionsHandler (replaces handler-local TxnRepo)
TransactionWriter — used by PlaidService
EventLogger       — used by PlaidService, Scheduler
NetWorthReader    — used by AnalyticsHandler
RulesRepo         — moved here from handlers/rules.go (was handler-local)
```

Handler-local interface definitions in `handlers/transactions.go` (`TxnRepo`) and `handlers/rules.go` (`RulesRepo`) are deleted; handlers import from `repositories`.

Handlers that currently take `*repositories.PlaidRepository` directly (`AccountsHandler`, `AnalyticsHandler`, `PlaidHandler`) are updated to take the appropriate interface(s).

### 3. Model file reorganization

`models/plaid.go` currently contains `PlaidItem`, `Account`, and `Transaction` — three distinct domain concepts sharing a file. Split:

| New file | Types |
|---|---|
| `models/plaid.go` | `PlaidItem`, `PlaidItemEntry` |
| `models/account.go` | `Account` |
| `models/transaction.go` | `Transaction` |
| `models/analytics.go` | `NetWorth`, `AnalyticsOverview`, `TopCategory` (already exists, no change needed) |

No type renames — only file moves.

### 4. `app.go` wiring update

`app.New()` constructs four repos instead of one `PlaidRepository`. Dependency injection updated throughout. `scheduler.go` and `services/plaid.go` updated to take the new split types/interfaces.

---

## Mobile

### 5. Duplicate `Transaction` type removed

`AccountsScreen.tsx` defines a local `Transaction` interface (12 fields) that mirrors the exported `Transaction` from `transactionService.ts`. The local definition is deleted; `AccountsScreen.tsx` imports `Transaction` from `../services/transactionService`.

### 6. `AuthResponse` import fixed

`AuthContext.tsx` uses `AuthResponse` as a type annotation in `persistAuth` without importing it from `authService.ts`. Add the import.

### 7. Hardcoded colors replaced with `COLORS`

`theme.ts` exports `COLORS` but no screen currently uses it. All inline hex literals that match a `COLORS` entry are replaced with the constant across all screen files. Specifically:
- `'#2dd4a7'` → `COLORS.teal`
- `'#fca5a5'` → `COLORS.red`
- `'#4ade80'` → `COLORS.green`
- `'rgba(255,255,255,0.4)'` → `COLORS.muted`
- `'rgba(255,255,255,0.1)'` → `COLORS.mutedLight`

Background color `'#0f172a'` → `COLORS.bg` where used. Navy `'#1e3a5f'` → `COLORS.navy` where used.

StyleSheet values that are already correct or use GLASS are left alone.

### 8. `plaidService.getTransactions` options object

Current signature: `getTransactions(limit, offset, accountId, month)` — all positional, all optional, easy to misuse.

New signature:
```ts
getTransactions(opts?: { limit?: number; offset?: number; accountId?: string; month?: string }): Promise<Transaction[]>
```

All call sites updated (`HomeScreen`, `TransactionsScreen`, `AccountsScreen` transaction modal).

### 9. API instance encapsulation

All service files currently do `const api = authService.api`, accessing a public field on the `AuthService` class. Extract a module `src/services/api.ts` that exports the configured Axios instance and interceptor setup. `AuthService` imports from `api.ts` instead of owning the instance. All other services import from `api.ts` directly, removing the dependency on `authService` for the HTTP client.

---

## Data flow changes

None. All external API contracts (HTTP routes, request/response shapes, DB schema) are unchanged.

---

## Behavior changes

None intentional. The one pre-existing issue this surfaces: `PatchTransactionCategory` returns `nil, nil` when the category doesn't belong to the user (the `EXISTS` subquery silently matches zero rows). This is left as-is — fixing it is a separate concern.

---

## Out of scope

- Tests (follow-up)
- New features
- Dependency changes
- DB schema changes
