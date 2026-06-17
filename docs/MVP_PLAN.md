# Mobile Personal Finance App MVP Plan

## Document Purpose

This document is a practical build map for an MVP personal finance mobile app inspired by the core value proposition of apps like Copilot Money: connect financial accounts, import transactions, categorize spending, let users correct the data, and show monthly budget progress.

The goal is not to clone any proprietary product, UI, branding, or implementation. The goal is to define a focused, buildable MVP with a Go backend and a mobile-only client.

This document is designed to be used as:

- A product direction document.
- A technical planning document.
- A development checklist.
- A handoff document for another engineer, AI coding agent, or future planning model.
- A living roadmap that can be checked off as work progresses.

---

# 1. Executive Summary

## 1.1 Product Concept

Build a mobile-only personal finance app that helps users answer three questions:

1. Where did my money go this month?
2. Am I over budget in any category?
3. Which transactions need review or correction?

The MVP should focus on transaction ingestion, categorization, user corrections, simple budget tracking, and a clean mobile dashboard.

## 1.2 MVP One-Liner

A mobile app where users connect bank accounts, automatically import transactions, categorize spending, set simple monthly category budgets, and track progress from a dashboard.

## 1.3 Recommended MVP Stack

| Layer | Recommendation |
|---|---|
| Mobile app | React Native, unless iOS-only polish is the priority |
| Backend | Go |
| API style | REST |
| Database | PostgreSQL |
| Background jobs | Postgres job table first, Redis/asynq later if needed |
| Bank aggregator | Plaid first |
| Auth | Clerk, Auth0, Firebase Auth, or custom JWT |
| Token encryption | Application-level encryption with managed secrets |
| Push notifications | APNs/FCM later in MVP polish phase |
| Observability | Structured logs, Sentry, basic metrics |
| Deployment | Render, Fly.io, AWS ECS, or similar simple container platform |

## 1.4 MVP Success Criteria

The MVP is ready for private beta when a user can:

- [ ] Sign up and log in.
- [ ] Connect a bank account through an aggregator.
- [ ] See connected accounts.
- [ ] See imported transactions.
- [ ] View spending by category for the current month.
- [ ] Change a transaction category.
- [ ] Create a rule from a merchant/category correction.
- [ ] Set monthly budgets by category.
- [ ] See budget progress on a dashboard.
- [ ] Exclude transfers from spending totals.
- [ ] Reconnect or handle a broken bank connection.
- [ ] Delete their account and financial data.

---

# 2. Product Boundaries

## 2.1 Build Now

The MVP should include only the features required to prove the core loop.

Core user loop:

```text
Sign up -> connect account -> sync transactions -> review spending -> correct categories -> set budgets -> monitor dashboard
```

Build these features:

- [ ] Mobile onboarding.
- [ ] User authentication.
- [ ] Bank account linking.
- [ ] Connected institution/account list.
- [ ] Transaction import.
- [ ] Transaction list.
- [ ] Transaction detail view.
- [ ] Default category system.
- [ ] Basic auto-categorization.
- [ ] Manual category correction.
- [ ] Merchant-based category rules.
- [ ] Monthly category budgets.
- [ ] Dashboard summary.
- [ ] Transfer/exclusion handling.
- [ ] Basic recurring transaction detection.
- [ ] Basic account reconnect state.
- [ ] Account deletion.

## 2.2 Do Not Build Yet

Avoid scope creep. These features should be deferred.

- [ ] Investment performance.
- [ ] Net-worth graphs.
- [ ] Multi-user household/couple sharing.
- [ ] Web app.
- [ ] Desktop app.
- [ ] Multiple bank aggregators.
- [ ] AI chat assistant.
- [ ] Complex forecasting.
- [ ] Receipt scanning.
- [ ] CSV import/export.
- [ ] Real estate valuation.
- [ ] Crypto.
- [ ] Tax planning.
- [ ] Bill pay or money movement.
- [ ] Financial advice/recommendations.

## 2.3 Product Principles

Use these principles to make decisions during implementation.

1. Data trust is more important than feature count.
2. User corrections must be fast and remembered.
3. Transfers must not double-count as spending.
4. Bank connection errors are normal and need first-class UX.
5. Manual override must beat automation.
6. Keep the first version opinionated and simple.
7. Prefer deterministic logic before AI.
8. Defer investments and net worth until transactions and budgets are stable.

---

# 3. Target User Experience

## 3.1 First-Time User Flow

```text
Open app
  -> Create account
  -> Accept privacy/data explanation
  -> Connect bank
  -> Wait for initial sync
  -> See accounts
  -> See transactions
  -> Review uncategorized transactions
  -> Set monthly budgets
  -> Land on dashboard
```

Checklist:

- [ ] User can understand why account linking is needed.
- [ ] User can connect an institution.
- [ ] User can recover gracefully if linking fails.
- [ ] User can see sync progress.
- [ ] User can skip budget setup and return later.
- [ ] User can complete onboarding in under five minutes.

## 3.2 Returning User Flow

```text
Open app
  -> See monthly spending dashboard
  -> Review new transactions
  -> Correct categories if needed
  -> Check budget progress
  -> Close app
```

Checklist:

- [ ] Home screen loads quickly.
- [ ] User can see if sync is current.
- [ ] User can see transactions needing review.
- [ ] User can see overspent categories.
- [ ] User can access recent transactions from dashboard.

---

# 4. MVP Feature Specification

## 4.1 Authentication

### Goal

Allow users to securely create an account and access their own financial data.

### Requirements

- [ ] User can sign up.
- [ ] User can log in.
- [ ] User can log out.
- [ ] API requests are authenticated.
- [ ] Backend associates every financial record with a user ID.
- [ ] User can delete their account.

### Recommended Initial Approach

Use a managed provider such as Clerk, Auth0, or Firebase Auth unless there is a strong reason to build custom auth. Managed auth reduces early security risk.

If building custom auth:

- [ ] Use email/password or magic link.
- [ ] Hash passwords with Argon2id or bcrypt.
- [ ] Use short-lived access tokens.
- [ ] Use refresh tokens securely.
- [ ] Add rate limiting.
- [ ] Add password reset.

## 4.2 Bank Linking

### Goal

Let the user connect a bank or credit card account.

### MVP Aggregator

Use Plaid first.

### Requirements

- [ ] Backend creates a link token.
- [ ] Mobile app opens Plaid Link.
- [ ] Mobile receives public token.
- [ ] Mobile sends public token to backend.
- [ ] Backend exchanges public token for access token.
- [ ] Backend encrypts and stores access token.
- [ ] Backend stores institution metadata.
- [ ] Backend fetches and stores accounts.
- [ ] Backend queues initial transaction sync.
- [ ] App shows connection status.
- [ ] App supports reconnect/reauth state.

### Non-Requirements for MVP

- [ ] Multi-aggregator fallback.
- [ ] Manual institution search outside the aggregator SDK.
- [ ] Direct bank API integrations.

## 4.3 Accounts

### Goal

Show the user connected accounts and current balances.

### Requirements

- [ ] User can see account name.
- [ ] User can see account type.
- [ ] User can see current balance.
- [ ] User can see institution name.
- [ ] User can hide an account from the UI.
- [ ] Hidden accounts can remain synced.
- [ ] User can remove a linked institution.

### MVP Supported Account Types

- [ ] Checking.
- [ ] Savings.
- [ ] Credit card.

Do not support investment or loan-specific logic yet, even if the aggregator returns those accounts. Store minimally or hide them until supported.

## 4.4 Transactions

### Goal

Import, normalize, display, categorize, and update transactions.

### Requirements

- [ ] Import transactions from aggregator.
- [ ] Store transaction provider ID.
- [ ] Store transaction date.
- [ ] Store authorized date if available.
- [ ] Store merchant/name fields.
- [ ] Store normalized merchant name.
- [ ] Store amount.
- [ ] Store pending status.
- [ ] Store account association.
- [ ] Store category association.
- [ ] Mark transactions needing review.
- [ ] Let user change category.
- [ ] Let user exclude transaction from budgets.
- [ ] Let user search transactions.
- [ ] Let user filter by category.
- [ ] Let user filter by account.
- [ ] Handle pending-to-posted transitions.
- [ ] Handle deleted/removed transactions from provider.

### Amount Sign Convention

Use this convention internally:

```text
Positive amount = money spent / outflow
Negative amount = money received / inflow
```

Examples:

```text
Starbucks purchase: 6.75
Payroll deposit: -3000.00
Refund: -25.00
Credit card payment: 1200.00, but marked as transfer/excluded
```

### Transaction Trust Rules

- [ ] Never silently override a user's manual category correction.
- [ ] User-created rules should beat provider categories.
- [ ] Pending transactions should be visually marked.
- [ ] Pending transactions should not be counted in final monthly spend by default.
- [ ] Transfers should not count as spending.

## 4.5 Categories

### Goal

Group spending into understandable categories.

### Default Categories

Seed these system categories:

- [ ] Income.
- [ ] Rent.
- [ ] Groceries.
- [ ] Restaurants.
- [ ] Coffee.
- [ ] Transport.
- [ ] Gas.
- [ ] Shopping.
- [ ] Entertainment.
- [ ] Travel.
- [ ] Health.
- [ ] Utilities.
- [ ] Insurance.
- [ ] Subscriptions.
- [ ] Education.
- [ ] Fees.
- [ ] Transfers.
- [ ] Uncategorized.

### Category Types

```text
expense
income
transfer
```

### MVP Requirements

- [ ] Categories are available to all users.
- [ ] User can create custom category.
- [ ] User can rename custom category.
- [ ] User can archive/delete custom category if unused or after reassignment.
- [ ] User can assign transaction to category.
- [ ] Category type drives budget calculations.

## 4.6 Categorization Rules

### Goal

Remember user corrections so repeated merchants are categorized correctly.

### Rule Example

```text
If normalized merchant name equals "starbucks", categorize as "Coffee".
```

### Requirements

- [ ] User can change a transaction category.
- [ ] App asks whether to apply change to just this transaction or merchant rule.
- [ ] User can apply rule to future transactions.
- [ ] User can optionally apply rule to past matching transactions.
- [ ] Rules run before provider category mapping.
- [ ] Rules are user-specific.

### Categorization Priority

Use this order:

```text
1. User rule
2. Known merchant mapping
3. Provider category mapping
4. Keyword fallback
5. Uncategorized + needs_review = true
```

## 4.7 Budgets

### Goal

Let users set simple monthly spending targets by category.

### MVP Budget Model

- Monthly only.
- Category-based only.
- No rollover.
- No zero-based/envelope complexity.
- No shared budgets.

### Requirements

- [ ] User can set monthly budget amount per expense category.
- [ ] User can edit budget amount.
- [ ] User can clear budget amount.
- [ ] User can see spent amount per category.
- [ ] User can see remaining amount per category.
- [ ] User can see over-budget categories.
- [ ] User can see total budgeted, total spent, and total remaining.

### Budget Calculation

```text
monthly category spending = sum(transaction.amount)
where:
  transaction.date is within selected month
  transaction.pending = false
  transaction.is_excluded = false
  category.type = expense
```

Income calculation:

```text
monthly income = abs(sum(transaction.amount))
where:
  transaction.date is within selected month
  transaction.pending = false
  transaction.is_excluded = false
  category.type = income
```

## 4.8 Transfers and Exclusions

### Goal

Prevent double-counting and give the user control over what appears in budgets.

### Requirements

- [ ] User can mark transaction as excluded.
- [ ] Excluded transactions do not count in budget totals.
- [ ] Transfer category does not count as expense.
- [ ] Credit card payments are detected as likely transfers.
- [ ] User can override transfer detection.

### MVP Transfer Detection Heuristics

Mark as likely transfer if name/category contains:

```text
payment
credit card payment
autopay
online transfer
ach transfer
transfer to
transfer from
venmo cashout
zelle transfer
```

Caution: Venmo, PayPal, Cash App, and Zelle can represent either spending or transfers. Prefer `needs_review = true` for ambiguous cases.

## 4.9 Recurring Transactions

### Goal

Surface predictable bills/subscriptions.

### MVP Requirements

- [ ] Detect repeated merchant names.
- [ ] Detect approximate monthly frequency.
- [ ] Store average amount.
- [ ] Estimate next expected date.
- [ ] Show upcoming recurring transactions on dashboard.
- [ ] Let user dismiss or ignore a detected recurring series.

### Simple Detection Rule

A recurring series candidate exists if:

```text
same normalized merchant
same user
at least 3 transactions
similar amount within 20 percent
average interval between 25 and 35 days for monthly
```

Do not over-engineer this in MVP.

## 4.10 Dashboard

### Goal

Give the user a useful overview in one screen.

### Dashboard Data

Show:

- [ ] Month-to-date spending.
- [ ] Total budgeted.
- [ ] Remaining budget.
- [ ] Income this month.
- [ ] Top spending categories.
- [ ] Categories over budget.
- [ ] Transactions needing review.
- [ ] Recent transactions.
- [ ] Upcoming recurring transactions.
- [ ] Sync status.

---

# 5. Backend Architecture

## 5.1 Architecture Overview

Use a modular monolith in Go.

```text
Mobile App
   |
   v
Go REST API
   |
   +--> PostgreSQL
   +--> Job runner / worker
   +--> Plaid API
   +--> Push notification service later
```

Avoid microservices for MVP.

## 5.2 Go Project Structure

```text
cmd/
  api/
    main.go
  worker/
    main.go

internal/
  accounts/
    handler.go
    service.go
    repository.go
    model.go
  auth/
  budgets/
  categories/
  config/
  crypto/
  dashboard/
  db/
  institutions/
  jobs/
  logger/
  plaid/
  recurring/
  rules/
  server/
  sync/
  transactions/
  users/
```

## 5.3 Layering Pattern

Use this pattern:

```text
HTTP handler -> service -> repository -> database
```

Rules:

- [ ] HTTP handlers parse input, validate request shape, and return responses.
- [ ] Services contain business logic.
- [ ] Repositories contain database access.
- [ ] Provider clients wrap external services such as Plaid.
- [ ] Workers call services, not handlers.

## 5.4 Recommended Go Libraries

Use pragmatic, common libraries.

| Need | Option |
|---|---|
| HTTP router | chi, gin, or echo |
| SQL access | sqlc or pgx |
| Migrations | goose or golang-migrate |
| Config | envconfig, viper, or plain env parsing |
| Logging | slog or zap |
| Validation | go-playground/validator or manual validation |
| UUID | google/uuid |
| Jobs | Postgres jobs first, asynq later |
| Tests | standard testing package + testify if desired |

A strong default:

```text
chi + pgx + sqlc + goose + slog
```

## 5.5 Backend Binaries

Build two binaries:

```text
api
worker
```

### API Binary

Responsibilities:

- [ ] Serve REST API.
- [ ] Authenticate requests.
- [ ] Create link tokens.
- [ ] Exchange public tokens.
- [ ] Return accounts, transactions, budgets, dashboard.
- [ ] Accept webhooks.
- [ ] Queue jobs.

### Worker Binary

Responsibilities:

- [ ] Process sync jobs.
- [ ] Pull account data.
- [ ] Pull transaction data.
- [ ] Categorize transactions.
- [ ] Detect recurring series.
- [ ] Process provider webhooks.
- [ ] Retry failed jobs.

---

# 6. Database Schema Draft

## 6.1 Users

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  auth_provider TEXT NOT NULL,
  auth_provider_user_id TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## 6.2 Institutions

```sql
CREATE TABLE institutions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  provider_institution_id TEXT,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Status values:

```text
active
reauth_required
error
removed
```

## 6.3 Linked Items

```sql
CREATE TABLE linked_items (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  institution_id UUID NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  provider_item_id TEXT NOT NULL,
  access_token_encrypted TEXT NOT NULL,
  status TEXT NOT NULL,
  last_synced_at TIMESTAMPTZ,
  last_successful_sync_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(provider, provider_item_id)
);
```

## 6.4 Accounts

```sql
CREATE TABLE accounts (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  linked_item_id UUID REFERENCES linked_items(id) ON DELETE SET NULL,
  provider_account_id TEXT,
  name TEXT NOT NULL,
  official_name TEXT,
  type TEXT NOT NULL,
  subtype TEXT,
  mask TEXT,
  currency TEXT NOT NULL DEFAULT 'USD',
  current_balance NUMERIC(14, 2),
  available_balance NUMERIC(14, 2),
  is_hidden BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, provider_account_id)
);
```

## 6.5 Categories

```sql
CREATE TABLE categories (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
  is_system BOOLEAN NOT NULL DEFAULT false,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

A `NULL user_id` can represent global system categories, or you can copy system categories per user. For MVP simplicity, copying defaults per user is often easier for customization.

## 6.6 Transactions

```sql
CREATE TABLE transactions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  provider_transaction_id TEXT,
  date DATE NOT NULL,
  authorized_date DATE,
  name TEXT NOT NULL,
  merchant_name TEXT,
  normalized_merchant_name TEXT,
  amount NUMERIC(14, 2) NOT NULL,
  currency TEXT NOT NULL DEFAULT 'USD',
  pending BOOLEAN NOT NULL DEFAULT false,
  category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
  original_category TEXT,
  transaction_type TEXT,
  is_transfer BOOLEAN NOT NULL DEFAULT false,
  is_excluded BOOLEAN NOT NULL DEFAULT false,
  needs_review BOOLEAN NOT NULL DEFAULT false,
  fingerprint TEXT,
  removed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, provider_transaction_id)
);
```

Recommended indexes:

```sql
CREATE INDEX idx_transactions_user_date ON transactions(user_id, date DESC);
CREATE INDEX idx_transactions_user_category_date ON transactions(user_id, category_id, date DESC);
CREATE INDEX idx_transactions_account_date ON transactions(account_id, date DESC);
CREATE INDEX idx_transactions_needs_review ON transactions(user_id, needs_review) WHERE needs_review = true;
CREATE INDEX idx_transactions_normalized_merchant ON transactions(user_id, normalized_merchant_name);
```

## 6.7 Category Rules

```sql
CREATE TABLE category_rules (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  match_type TEXT NOT NULL,
  match_value TEXT NOT NULL,
  category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, match_type, match_value)
);
```

Initial match types:

```text
normalized_merchant
transaction_name_contains
```

## 6.8 Budgets

```sql
CREATE TABLE budgets (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  month DATE NOT NULL,
  category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  amount NUMERIC(14, 2) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, month, category_id)
);
```

Store `month` as the first day of the month, for example `2026-06-01`.

## 6.9 Recurring Series

```sql
CREATE TABLE recurring_series (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  normalized_merchant_name TEXT NOT NULL,
  category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
  average_amount NUMERIC(14, 2),
  frequency TEXT NOT NULL,
  next_expected_date DATE,
  confidence NUMERIC(5, 4),
  ignored BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, normalized_merchant_name, frequency)
);
```

## 6.10 Jobs

```sql
CREATE TABLE jobs (
  id UUID PRIMARY KEY,
  type TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 5,
  run_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Job statuses:

```text
queued
running
succeeded
failed
dead
```

Job types:

```text
sync_linked_item
sync_transactions
categorize_account_transactions
detect_recurring_transactions
process_plaid_webhook
refresh_account_balances
```

---

# 7. API Map

## 7.1 API Conventions

Base path:

```text
/v1
```

General conventions:

- [ ] Use JSON request/response bodies.
- [ ] Use authenticated requests except public health checks and provider webhooks.
- [ ] Use request IDs for tracing.
- [ ] Return stable error shapes.
- [ ] Do not leak provider access tokens.

Example error shape:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "Category does not exist"
  }
}
```

## 7.2 Health

```http
GET /healthz
GET /readyz
```

## 7.3 Current User

```http
GET /v1/me
DELETE /v1/me
```

## 7.4 Bank Linking

```http
POST /v1/link/token
POST /v1/link/exchange
GET  /v1/institutions
DELETE /v1/institutions/{institution_id}
```

### POST /v1/link/token

Creates a provider link token for the mobile client.

Response:

```json
{
  "link_token": "..."
}
```

### POST /v1/link/exchange

Request:

```json
{
  "public_token": "..."
}
```

Behavior:

- [ ] Exchange public token for access token.
- [ ] Encrypt and store access token.
- [ ] Fetch institution/account data.
- [ ] Queue initial sync.

## 7.5 Accounts

```http
GET   /v1/accounts
PATCH /v1/accounts/{account_id}
POST  /v1/accounts/sync
```

### PATCH /v1/accounts/{account_id}

Request:

```json
{
  "is_hidden": true
}
```

## 7.6 Transactions

```http
GET   /v1/transactions?month=2026-06&account_id=&category_id=&q=&needs_review=
GET   /v1/transactions/{transaction_id}
PATCH /v1/transactions/{transaction_id}
```

### PATCH /v1/transactions/{transaction_id}

Request:

```json
{
  "category_id": "uuid",
  "is_excluded": false,
  "needs_review": false,
  "create_rule": true,
  "apply_rule_to_past": false
}
```

Behavior:

- [ ] Update transaction.
- [ ] If `create_rule = true`, create/update merchant category rule.
- [ ] If `apply_rule_to_past = true`, recategorize matching historical transactions.

## 7.7 Categories

```http
GET    /v1/categories
POST   /v1/categories
PATCH  /v1/categories/{category_id}
DELETE /v1/categories/{category_id}
```

## 7.8 Budgets

```http
GET /v1/budgets?month=2026-06
PUT /v1/budgets?month=2026-06
GET /v1/budgets/summary?month=2026-06
```

### PUT /v1/budgets?month=2026-06

Request:

```json
{
  "budgets": [
    {
      "category_id": "uuid",
      "amount": "500.00"
    }
  ]
}
```

## 7.9 Dashboard

```http
GET /v1/dashboard?month=2026-06
```

Response shape:

```json
{
  "month": "2026-06",
  "spent": "2450.33",
  "income": "6200.00",
  "budgeted": "4000.00",
  "remaining": "1549.67",
  "needs_review_count": 12,
  "top_categories": [],
  "over_budget_categories": [],
  "recent_transactions": [],
  "upcoming_recurring": [],
  "sync_status": {
    "status": "ok",
    "last_successful_sync_at": "2026-06-16T12:00:00Z"
  }
}
```

## 7.10 Webhooks

```http
POST /v1/webhooks/plaid
```

Webhook behavior:

- [ ] Verify webhook signature if supported/required.
- [ ] Store webhook event metadata.
- [ ] Queue appropriate job.
- [ ] Return quickly.
- [ ] Do not perform heavy sync inline.

---

# 8. Sync Pipeline

## 8.1 Initial Sync

```text
User completes bank linking
  -> Backend exchanges public token
  -> Backend stores encrypted access token
  -> Backend fetches accounts
  -> Backend upserts accounts
  -> Backend queues sync_transactions job
  -> Worker fetches transactions
  -> Worker normalizes transactions
  -> Worker upserts transactions
  -> Worker categorizes transactions
  -> Worker detects recurring transactions
  -> Dashboard becomes available
```

Checklist:

- [ ] Create link token.
- [ ] Exchange public token.
- [ ] Store linked item.
- [ ] Store institution.
- [ ] Fetch accounts.
- [ ] Store accounts.
- [ ] Queue transaction sync.
- [ ] Fetch transactions.
- [ ] Upsert transactions.
- [ ] Categorize transactions.
- [ ] Detect recurring.
- [ ] Update sync metadata.

## 8.2 Ongoing Sync

```text
Provider webhook or scheduled job
  -> Queue sync_linked_item
  -> Fetch added/modified/removed transactions
  -> Upsert changed transactions
  -> Mark removed transactions
  -> Categorize new/changed transactions
  -> Update account balances
  -> Detect recurring changes
  -> Update sync status
```

Checklist:

- [ ] Handle added transactions.
- [ ] Handle modified transactions.
- [ ] Handle removed transactions.
- [ ] Handle pending-to-posted transition.
- [ ] Handle provider errors.
- [ ] Handle reauth-required state.
- [ ] Retry transient failures.
- [ ] Dead-letter repeated failures.

## 8.3 Transaction Normalization

For each incoming transaction:

- [ ] Normalize sign convention.
- [ ] Normalize merchant name.
- [ ] Store original transaction name.
- [ ] Store provider category.
- [ ] Map provider account ID to internal account ID.
- [ ] Create fingerprint.
- [ ] Detect likely transfer.
- [ ] Set initial category.
- [ ] Set `needs_review` if low confidence.

## 8.4 Merchant Normalization

Simple first version:

```text
Original: SQ *BLUE BOTTLE 1234 SAN FRANCISCO CA
Normalized: blue bottle
```

Rules:

- [ ] Lowercase.
- [ ] Trim whitespace.
- [ ] Remove card/network noise.
- [ ] Remove terminal/location numbers.
- [ ] Remove repeated punctuation.
- [ ] Prefer provider merchant name when available.

Potential noise tokens:

```text
sq
pos
debit
card purchase
purchase authorized on
ach
online pmt
terminal
```

---

# 9. Mobile App Plan

## 9.1 Navigation Structure

Suggested tabs:

```text
Home
Transactions
Budgets
Accounts
Settings
```

## 9.2 Screens

### Onboarding Screens

- [ ] Welcome screen.
- [ ] Sign up/login screen.
- [ ] Data/privacy explanation screen.
- [ ] Connect bank screen.
- [ ] Initial sync loading screen.
- [ ] Optional budget setup screen.

### Home Screen

Show:

- [ ] Current month spending.
- [ ] Budget remaining.
- [ ] Income this month.
- [ ] Needs review count.
- [ ] Top categories.
- [ ] Over-budget categories.
- [ ] Recent transactions.
- [ ] Upcoming recurring transactions.
- [ ] Sync status.

### Transactions Screen

Features:

- [ ] Infinite list or paginated list.
- [ ] Search.
- [ ] Filter by account.
- [ ] Filter by category.
- [ ] Filter by needs review.
- [ ] Month selector.
- [ ] Pending indicator.
- [ ] Excluded indicator.

### Transaction Detail Screen

Features:

- [ ] Show amount.
- [ ] Show date.
- [ ] Show account.
- [ ] Show original merchant/name.
- [ ] Show category selector.
- [ ] Toggle excluded.
- [ ] Create category rule option.
- [ ] Apply rule to past transactions option.

### Budgets Screen

Features:

- [ ] Current month selector.
- [ ] Category rows.
- [ ] Budget amount.
- [ ] Spent amount.
- [ ] Remaining amount.
- [ ] Progress indicator.
- [ ] Edit budget amount.

### Accounts Screen

Features:

- [ ] Institution list.
- [ ] Account list.
- [ ] Balance display.
- [ ] Connection status.
- [ ] Reconnect button.
- [ ] Hide account toggle.
- [ ] Remove institution.

### Settings Screen

Features:

- [ ] Profile.
- [ ] Manage categories.
- [ ] Manage connected accounts.
- [ ] Notifications later.
- [ ] Privacy/data export later.
- [ ] Delete account.
- [ ] Log out.

---

# 10. Implementation Milestones

## Milestone 0: Project Setup

Goal: working repositories and development environment.

Backend tasks:

- [ ] Create Go backend repository.
- [ ] Create mobile app repository or monorepo.
- [ ] Add Docker Compose for Postgres.
- [ ] Add environment variable management.
- [ ] Add migration tool.
- [ ] Add basic logging.
- [ ] Add health endpoints.
- [ ] Add CI for tests/linting.
- [ ] Add README with local setup.

Mobile tasks:

- [ ] Create React Native app.
- [ ] Add navigation.
- [ ] Add environment config.
- [ ] Add API client wrapper.
- [ ] Add basic auth screens placeholder.

Acceptance criteria:

- [ ] Backend starts locally.
- [ ] Mobile app starts locally.
- [ ] Mobile app can call backend health endpoint.

## Milestone 1: Auth and User Foundation

Goal: authenticated user can access API.

Backend tasks:

- [ ] Implement auth middleware.
- [ ] Create users table.
- [ ] Create user on first authenticated request or signup callback.
- [ ] Add `GET /v1/me`.
- [ ] Add user-scoped request context.

Mobile tasks:

- [ ] Implement signup/login.
- [ ] Store auth session securely.
- [ ] Call `GET /v1/me`.
- [ ] Add logged-in navigation state.

Acceptance criteria:

- [ ] User can sign up.
- [ ] User can log in.
- [ ] Mobile app displays current user info.
- [ ] Unauthenticated API requests are rejected.

## Milestone 2: Bank Linking and Accounts

Goal: user can connect a bank and see accounts.

Backend tasks:

- [ ] Add Plaid client wrapper.
- [ ] Add institutions table.
- [ ] Add linked_items table.
- [ ] Add accounts table.
- [ ] Implement `POST /v1/link/token`.
- [ ] Implement `POST /v1/link/exchange`.
- [ ] Encrypt provider access tokens.
- [ ] Fetch accounts after exchange.
- [ ] Store institution and account records.
- [ ] Implement `GET /v1/accounts`.
- [ ] Implement `PATCH /v1/accounts/{id}`.

Mobile tasks:

- [ ] Add connect bank screen.
- [ ] Integrate Plaid Link SDK.
- [ ] Send public token to backend.
- [ ] Add accounts screen.
- [ ] Add hide account toggle.

Acceptance criteria:

- [ ] User can connect a bank in sandbox.
- [ ] Backend stores encrypted access token.
- [ ] User can see accounts in app.
- [ ] User can hide an account.

## Milestone 3: Transaction Sync

Goal: user can see imported transactions.

Backend tasks:

- [ ] Add transactions table.
- [ ] Add job table.
- [ ] Implement job enqueue/dequeue.
- [ ] Implement worker binary.
- [ ] Implement initial transaction sync job.
- [ ] Normalize transaction amounts.
- [ ] Normalize merchant names.
- [ ] Upsert transactions.
- [ ] Implement `GET /v1/transactions`.
- [ ] Implement `GET /v1/transactions/{id}`.

Mobile tasks:

- [ ] Add transactions tab.
- [ ] Add transaction list.
- [ ] Add transaction detail screen.
- [ ] Add loading/error states.
- [ ] Add empty state.

Acceptance criteria:

- [ ] After bank linking, transactions sync automatically.
- [ ] User can view transaction list.
- [ ] User can open transaction detail.
- [ ] Duplicate transaction imports are avoided.

## Milestone 4: Categories and Manual Corrections

Goal: user can categorize transactions and the app remembers corrections.

Backend tasks:

- [ ] Add categories table.
- [ ] Seed default categories for new users.
- [ ] Add category_rules table.
- [ ] Implement categorization pipeline.
- [ ] Implement `GET /v1/categories`.
- [ ] Implement category CRUD.
- [ ] Implement `PATCH /v1/transactions/{id}`.
- [ ] Implement create merchant rule from correction.
- [ ] Implement apply rule to past transactions.

Mobile tasks:

- [ ] Add category selector.
- [ ] Add create-rule prompt.
- [ ] Add apply-to-past option.
- [ ] Add needs-review filter.
- [ ] Add category management screen.

Acceptance criteria:

- [ ] User can change a transaction category.
- [ ] User can create a merchant rule.
- [ ] Future matching transactions use the rule.
- [ ] Past matching transactions can be updated on request.

## Milestone 5: Budgets and Dashboard

Goal: user can set monthly budgets and track progress.

Backend tasks:

- [ ] Add budgets table.
- [ ] Implement `GET /v1/budgets`.
- [ ] Implement `PUT /v1/budgets`.
- [ ] Implement budget summary calculation.
- [ ] Implement `GET /v1/budgets/summary`.
- [ ] Implement `GET /v1/dashboard`.
- [ ] Exclude transfers and excluded transactions from spending.

Mobile tasks:

- [ ] Add budgets tab.
- [ ] Add edit budget amount flow.
- [ ] Add dashboard screen.
- [ ] Show month-to-date spending.
- [ ] Show remaining budget.
- [ ] Show top categories.
- [ ] Show needs-review count.

Acceptance criteria:

- [ ] User can set category budgets.
- [ ] User can see spending vs budget.
- [ ] Dashboard displays useful current-month summary.
- [ ] Transfers do not count toward spending.

## Milestone 6: Webhooks, Reauth, and Sync Reliability

Goal: app handles ongoing account updates and connection errors.

Backend tasks:

- [ ] Implement `POST /v1/webhooks/plaid`.
- [ ] Queue sync job from webhook.
- [ ] Store sync status and last error.
- [ ] Handle item errors.
- [ ] Mark linked item as `reauth_required` when needed.
- [ ] Add manual sync endpoint.
- [ ] Add retry logic to jobs.
- [ ] Add dead job status.

Mobile tasks:

- [ ] Show sync status.
- [ ] Show reconnect prompt.
- [ ] Add reconnect flow.
- [ ] Add manual refresh action.

Acceptance criteria:

- [ ] Webhooks trigger transaction updates.
- [ ] Failed connections show clear status.
- [ ] User can reconnect an institution.
- [ ] Sync failures are visible in logs/admin diagnostics.

## Milestone 7: Recurring Transactions

Goal: show predictable bills/subscriptions.

Backend tasks:

- [ ] Add recurring_series table.
- [ ] Implement simple recurring detection.
- [ ] Estimate next expected date.
- [ ] Add upcoming recurring data to dashboard.
- [ ] Allow ignored recurring series.

Mobile tasks:

- [ ] Show upcoming recurring items on dashboard.
- [ ] Add recurring detail or dismiss action.

Acceptance criteria:

- [ ] App detects common monthly recurring transactions.
- [ ] Dashboard shows upcoming recurring items.
- [ ] User can ignore incorrect recurring detection.

## Milestone 8: Privacy, Deletion, and Beta Readiness

Goal: safe enough for private beta.

Backend tasks:

- [ ] Implement account deletion.
- [ ] Delete user financial data on request.
- [ ] Remove/deactivate aggregator item on account deletion.
- [ ] Add audit logs for sensitive operations.
- [ ] Add rate limiting.
- [ ] Add structured error logging.
- [ ] Add Sentry or equivalent.
- [ ] Add backup strategy.
- [ ] Add basic admin/debug tooling.

Mobile tasks:

- [ ] Add delete account screen.
- [ ] Add privacy explanation.
- [ ] Add error reporting UX.
- [ ] Add beta feedback link.

Acceptance criteria:

- [ ] User can delete account and data.
- [ ] App has basic monitoring.
- [ ] Support/debug information is available for sync issues.
- [ ] Private beta users can safely test the app.

---

# 11. Security and Privacy Checklist

## 11.1 Data Handling

- [ ] Do not store bank credentials.
- [ ] Store only aggregator access tokens.
- [ ] Encrypt aggregator access tokens at application level.
- [ ] Use TLS everywhere.
- [ ] Encrypt database storage at rest through cloud provider.
- [ ] Restrict production database access.
- [ ] Use least-privilege service credentials.
- [ ] Do not log access tokens.
- [ ] Do not log full financial payloads unnecessarily.
- [ ] Redact sensitive fields in logs.

## 11.2 Authentication and Authorization

- [ ] Every financial record is scoped by user ID.
- [ ] Every API query filters by authenticated user.
- [ ] Add tests for cross-user access prevention.
- [ ] Use secure session storage on mobile.
- [ ] Add rate limiting on auth-sensitive endpoints.

## 11.3 User Rights and Trust

- [ ] User can delete their account.
- [ ] User can remove connected institutions.
- [ ] User can hide accounts.
- [ ] User can understand what data is imported.
- [ ] User can understand that the app is read-only.
- [ ] Privacy policy is clear before public launch.
- [ ] Terms of service are clear before public launch.

## 11.4 Compliance Boundaries

For MVP, avoid features that create regulated financial advice or money movement.

Do not say:

```text
You should buy this stock.
You should sell this asset.
We will optimize your investments.
We will move money for you.
```

Safer positioning:

```text
Track your spending.
Understand where your money goes.
Monitor budgets.
Review recurring expenses.
See trends in your transactions.
```

---

# 12. Testing Plan

## 12.1 Backend Unit Tests

- [ ] Merchant normalization.
- [ ] Amount normalization.
- [ ] Category rule priority.
- [ ] Transfer detection.
- [ ] Budget calculation.
- [ ] Recurring detection.
- [ ] User authorization checks.

## 12.2 Backend Integration Tests

- [ ] Authenticated API access.
- [ ] Link token creation with mocked provider.
- [ ] Public token exchange with mocked provider.
- [ ] Account upsert.
- [ ] Transaction upsert.
- [ ] Duplicate handling.
- [ ] Transaction category update.
- [ ] Budget summary endpoint.
- [ ] Dashboard endpoint.

## 12.3 Mobile Tests

- [ ] Login flow.
- [ ] Navigation smoke tests.
- [ ] Account list rendering.
- [ ] Transaction list rendering.
- [ ] Transaction category update.
- [ ] Budget editing.
- [ ] Dashboard loading/error states.

## 12.4 Manual QA Scenarios

- [ ] New user signs up and links sandbox bank.
- [ ] User sees accounts.
- [ ] User sees transactions.
- [ ] User changes category.
- [ ] User creates merchant rule.
- [ ] User sets budgets.
- [ ] User marks credit card payment as transfer.
- [ ] User hides an account.
- [ ] User removes institution.
- [ ] User deletes account.
- [ ] Provider webhook triggers sync.
- [ ] Provider reauth state appears correctly.

---

# 13. Observability and Operations

## 13.1 Logs

Log these events:

- [ ] User signup/login events, without sensitive data.
- [ ] Link token creation.
- [ ] Public token exchange success/failure.
- [ ] Account sync start/end.
- [ ] Transaction sync start/end.
- [ ] Number of transactions added/modified/removed.
- [ ] Categorization job start/end.
- [ ] Recurring detection job start/end.
- [ ] Webhook received.
- [ ] Provider errors.
- [ ] Reauth-required events.

## 13.2 Metrics

Track:

- [ ] Active users.
- [ ] Linked institutions per user.
- [ ] Accounts per user.
- [ ] Transactions synced per user.
- [ ] Sync success rate.
- [ ] Sync failure rate.
- [ ] Average sync duration.
- [ ] Categorization confidence/review rate.
- [ ] Number of category corrections.
- [ ] Number of created rules.
- [ ] Budget setup completion rate.

## 13.3 Admin/Debug Tools

At minimum, create internal diagnostics for:

- [ ] User linked items.
- [ ] Institution statuses.
- [ ] Last sync status.
- [ ] Last provider error.
- [ ] Job history.
- [ ] Webhook history.
- [ ] Transaction count by account.

This can be a CLI or protected internal endpoint for MVP.

---

# 14. Important Edge Cases

## 14.1 Pending Transactions

Problem:

A pending transaction can later post with a different provider transaction ID or amount.

MVP approach:

- [ ] Store pending status.
- [ ] Prefer provider transaction lifecycle fields if available.
- [ ] Use fallback fingerprint matching.
- [ ] Avoid counting pending transactions in final budget totals by default.

## 14.2 Duplicate Transactions

Problem:

Same transaction may appear twice due to sync behavior or pending/posting transition.

MVP approach:

- [ ] Unique constraint on provider transaction ID.
- [ ] Fallback fingerprint: account ID + date + amount + normalized merchant.
- [ ] Keep removed_at instead of hard-deleting provider-removed transactions immediately.

## 14.3 Credit Card Payments

Problem:

Credit card purchases already count as spending. Credit card payments should not count again.

MVP approach:

- [ ] Detect likely credit card payments.
- [ ] Mark as transfer or excluded.
- [ ] Let user override.

## 14.4 Refunds

Problem:

Refunds reduce spending but can be categorized inconsistently.

MVP approach:

- [ ] Negative expense-category transactions reduce category spending.
- [ ] Let user recategorize refunds.
- [ ] Do not overcomplicate refund matching in MVP.

## 14.5 Reimbursements

Problem:

User pays for something and gets reimbursed later.

MVP approach:

- [ ] Let user categorize reimbursement as income or exclude both transactions manually.
- [ ] Defer automatic reimbursement matching.

## 14.6 Ambiguous Payment Apps

Problem:

Venmo/PayPal/Cash App/Zelle can be spending, transfer, or reimbursement.

MVP approach:

- [ ] Mark ambiguous payment-app transactions as needs review.
- [ ] Let user create rules.

---

# 15. Suggested Development Timeline

This timeline assumes a small team or solo developer and a focused MVP.

## Week 1: Foundation

- [ ] Backend repo.
- [ ] Mobile repo.
- [ ] Local Postgres.
- [ ] Migrations.
- [ ] Health endpoint.
- [ ] Auth integration.
- [ ] Mobile login.
- [ ] User model.

Deliverable:

```text
Authenticated mobile app can call Go backend.
```

## Week 2: Bank Linking

- [ ] Plaid sandbox setup.
- [ ] Link token endpoint.
- [ ] Plaid Link mobile flow.
- [ ] Public token exchange.
- [ ] Encrypted token storage.
- [ ] Institutions table.
- [ ] Accounts table.
- [ ] Accounts screen.

Deliverable:

```text
User can connect sandbox bank and see accounts.
```

## Week 3: Transaction Sync

- [ ] Jobs table.
- [ ] Worker binary.
- [ ] Initial sync job.
- [ ] Transactions table.
- [ ] Transaction normalization.
- [ ] Transaction list endpoint.
- [ ] Transaction list mobile screen.
- [ ] Transaction detail screen.

Deliverable:

```text
User can see real imported transactions.
```

## Week 4: Categories and Rules

- [ ] Categories table.
- [ ] Default category seeding.
- [ ] Categorization pipeline.
- [ ] Category selector.
- [ ] Manual correction.
- [ ] Category rules table.
- [ ] Merchant rule creation.
- [ ] Needs-review filter.

Deliverable:

```text
User can fix categories and the app remembers corrections.
```

## Week 5: Budgets and Dashboard

- [ ] Budgets table.
- [ ] Budget endpoints.
- [ ] Budget screen.
- [ ] Dashboard endpoint.
- [ ] Dashboard mobile screen.
- [ ] Transfer/exclusion logic.
- [ ] Top categories.
- [ ] Over-budget categories.

Deliverable:

```text
User can set monthly category budgets and see progress.
```

## Week 6: Sync Reliability and Beta Polish

- [ ] Plaid webhooks.
- [ ] Reauth-required status.
- [ ] Manual sync.
- [ ] Job retries.
- [ ] Basic recurring detection.
- [ ] Account deletion.
- [ ] Error tracking.
- [ ] Private beta QA.

Deliverable:

```text
Private beta candidate.
```

---

# 16. Definition of Done

## 16.1 Feature-Level Done

A feature is done when:

- [ ] Backend endpoint or worker logic is implemented.
- [ ] Database migration exists if needed.
- [ ] Mobile UI exists if user-facing.
- [ ] Loading state exists.
- [ ] Error state exists.
- [ ] Empty state exists where relevant.
- [ ] Unit tests exist for business logic.
- [ ] API tests or integration tests exist for critical paths.
- [ ] Logs are sufficient for debugging.
- [ ] User data is scoped correctly.
- [ ] Sensitive data is not logged.

## 16.2 MVP Done

The MVP is done when:

- [ ] New user can onboard successfully.
- [ ] User can link a sandbox/real bank account.
- [ ] User can see accounts.
- [ ] User can see transactions.
- [ ] User can correct transaction categories.
- [ ] User can create category rules.
- [ ] User can set monthly budgets.
- [ ] User can see a dashboard.
- [ ] User can exclude transfers.
- [ ] App handles common sync errors.
- [ ] User can delete account/data.
- [ ] App has basic logs and error monitoring.
- [ ] At least five test users can complete onboarding without developer help.

---

# 17. Backlog After MVP

Prioritize only after the core MVP is stable.

## 17.1 Near-Term Enhancements

- [ ] Better merchant normalization.
- [ ] Better recurring detection.
- [ ] Push notifications for unusual spending.
- [ ] Subscription renewal reminders.
- [ ] CSV export.
- [ ] Manual cash account.
- [ ] Manual transactions.
- [ ] More budget visualizations.
- [ ] Custom category icons.
- [ ] Month-over-month trends.

## 17.2 Medium-Term Enhancements

- [ ] Net worth.
- [ ] Balance history.
- [ ] Loans.
- [ ] Investments balance-only.
- [ ] Multiple aggregators.
- [ ] Android/iOS platform-specific polish.
- [ ] Shared household budgeting.
- [ ] Data export.
- [ ] Basic AI-assisted categorization.

## 17.3 Long-Term Enhancements

- [ ] Investment holdings.
- [ ] Investment performance.
- [ ] Advanced forecasting.
- [ ] Bill calendar.
- [ ] Real estate/manual assets.
- [ ] AI assistant.
- [ ] Advanced anomaly detection.
- [ ] Multi-currency.
- [ ] International bank coverage.

---

# 18. Prompts for Continuing Work with Another Model

Use these prompts to continue development with another AI model or coding agent.

## 18.1 Architecture Prompt

```text
You are helping build a mobile-only personal finance MVP with a Go backend, PostgreSQL database, React Native mobile app, and Plaid as the initial bank aggregator. The MVP scope is account linking, transaction sync, categories, merchant rules, monthly budgets, dashboard, transfer exclusion, and basic recurring detection. Do not add investments, net worth, web app, shared accounts, or AI chat yet.

Use the project plan in this document as the source of truth. Focus on the next unchecked milestone. Prefer simple, production-minded implementation over over-engineering. Preserve user data isolation, avoid logging sensitive financial data, and keep bank sync logic in background workers rather than HTTP handlers.
```

## 18.2 Backend Implementation Prompt

```text
Implement the next backend task for a Go modular monolith personal finance MVP. Use REST APIs, PostgreSQL, migrations, and a handler-service-repository structure. Every financial table must be scoped by user_id. Use background jobs for Plaid sync work. Do not implement out-of-scope features such as investments, net worth, multi-user sharing, or web app support.

Before coding, identify the relevant tables, endpoints, service methods, repository methods, validation rules, and tests. Then implement incrementally.
```

## 18.3 Mobile Implementation Prompt

```text
Implement the next mobile task for a React Native personal finance MVP. The app is mobile-only and communicates with a Go REST API. The main tabs are Home, Transactions, Budgets, Accounts, and Settings. Prioritize clear loading, empty, and error states. Do not add web-specific behavior or out-of-scope product features.

Before coding, identify the required API calls, screen state, navigation behavior, and user interactions. Then implement incrementally.
```

## 18.4 Product Decision Prompt

```text
Given the MVP scope in this document, help decide the simplest product behavior for the next feature. Optimize for reliability, user trust, fast correction flows, and avoiding budget miscalculation. Do not expand scope unless the existing MVP cannot function without the change.
```

## 18.5 QA Prompt

```text
Create a QA checklist for the current milestone of this personal finance MVP. Include happy paths, failure paths, data consistency checks, privacy/security checks, and mobile UX checks. Focus on account linking, transaction sync, categorization, rules, budgets, dashboard, and transfer exclusion.
```

---

# 19. Immediate Next Actions

Start here.

## 19.1 Product Decisions to Lock

- [ ] Confirm mobile framework: React Native, Swift, or Flutter.
- [ ] Confirm backend framework/router: chi, gin, or echo.
- [ ] Confirm auth provider: Clerk, Auth0, Firebase Auth, or custom.
- [ ] Confirm initial bank aggregator: Plaid.
- [ ] Confirm MVP geography: likely US only.
- [ ] Confirm supported account types: checking, savings, credit card.
- [ ] Confirm budget model: monthly category budgets, no rollover.
- [ ] Confirm amount sign convention.
- [ ] Confirm whether pending transactions count in spending. Recommended: no.
- [ ] Confirm deployment target.

## 19.2 First Engineering Tasks

- [ ] Create backend repo.
- [ ] Create mobile repo.
- [ ] Set up local Postgres.
- [ ] Add database migrations.
- [ ] Add users table.
- [ ] Add auth integration.
- [ ] Add `GET /healthz`.
- [ ] Add `GET /v1/me`.
- [ ] Create mobile login flow.
- [ ] Verify mobile can call authenticated backend endpoint.

## 19.3 First Technical Spike

Prove this flow as soon as possible:

```text
User signs up -> connects Plaid sandbox account -> backend stores accounts -> mobile displays accounts
```

After that, prove this flow:

```text
Backend syncs transactions -> mobile displays transactions -> user changes category -> future merchant transactions use corrected category
```

These two loops prove the main technical and product risks.

---

# 20. Final MVP North Star

The first version should not try to be a complete finance platform.

It should do one thing very well:

```text
Help users understand and control monthly spending from automatically imported transaction data.
```

If a proposed feature does not improve that loop, defer it.
