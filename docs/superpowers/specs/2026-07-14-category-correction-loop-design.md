# Category Correction Loop — Design Spec

**Date:** 2026-07-14
**Scope:** Backend (PATCH /v1/transactions/:id, POST/GET/DELETE /v1/rules, GET /v1/transactions/:id) + Mobile (TransactionDetailScreen, category picker, rule creation prompt)

---

## 1. Problem

Users cannot currently correct a transaction's category or teach the app to remember that correction. This is the core product loop: connect bank → see transactions → fix categories → app learns. Without it the MVP is incomplete.

---

## 2. Approach

Two separate concerns, two separate round trips from mobile:

1. **Fix the transaction** — `PATCH /v1/transactions/:id` updates a single transaction's category.
2. **Create a rule** — `POST /v1/rules` upserts a merchant→category rule and optionally backfills past matching transactions inline (capped at 500 rows for MVP).

Rules are a first-class resource with their own endpoints. This keeps the transaction patch clean and allows future rule management UI.

---

## 3. Database

New migration: `0008_create_category_rules.sql`

```sql
CREATE TABLE category_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  normalized_merchant TEXT NOT NULL,
  category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, normalized_merchant)
);
```

- `normalized_merchant` uses the same normalization already applied during Plaid sync (lowercase, strip noise tokens).
- Conflict resolution: `ON CONFLICT (user_id, normalized_merchant) DO UPDATE SET category_id = excluded.category_id, updated_at = now()` — a correction always overwrites the previous rule.
- Hard delete on `DELETE /v1/rules/:id` — rules are user preferences, no audit requirement.

---

## 4. API

### 4.1 GET /v1/transactions/:id

Returns a single transaction by ID, scoped to the authenticated user.

Response: same shape as the list items in `GET /v1/transactions`.

Returns 404 if the transaction does not belong to the user.

### 4.2 PATCH /v1/transactions/:id

Updates a single transaction. For MVP, only `category_id` is patchable.

Request:
```json
{ "category_id": "uuid" }
```

Response: updated transaction object.

Behavior:
- Validates `category_id` exists and belongs to the user.
- Updates `transactions.category_id`.
- Does not touch rules — rule creation is a separate call.
- Returns 404 if transaction does not belong to user.

### 4.3 POST /v1/rules

Creates or overwrites a merchant→category rule.

Request:
```json
{
  "normalized_merchant": "starbucks",
  "category_id": "uuid",
  "apply_to_past": false
}
```

Response:
```json
{
  "rule": { "id": "...", "normalized_merchant": "starbucks", "category": { ... }, "created_at": "..." },
  "updated_count": 0
}
```

Behavior:
- Upserts rule row (insert or update on conflict).
- If `apply_to_past: true`: runs inline update on the 500 most recent matching transactions — `UPDATE transactions SET category_id = $1 WHERE id IN (SELECT id FROM transactions WHERE user_id = $2 AND normalized_merchant_name = $3 AND pending = false ORDER BY date DESC LIMIT 500)`. Returns actual rows updated in `updated_count`.
- `apply_to_past: false`: `updated_count` is 0.

### 4.4 GET /v1/rules

Lists all rules for the authenticated user.

Response:
```json
{
  "rules": [
    { "id": "...", "normalized_merchant": "starbucks", "category": { "id": "...", "name": "Coffee", "icon": "☕", "color": "#..." }, "created_at": "..." }
  ]
}
```

### 4.5 DELETE /v1/rules/:id

Hard deletes a rule. Returns 404 if rule does not belong to user. Returns 204 on success.

---

## 5. Backend Implementation

### Layering

Follow existing handler → service → repository pattern.

New files:
- `internal/models/rules.go` — `CategoryRule` model
- `internal/repositories/rules.go` — `RulesRepository` with methods: `Upsert`, `List`, `Delete`, `ApplyToPast`
- `internal/handlers/rules.go` — HTTP handlers: `CreateRule`, `ListRules`, `DeleteRule`

Extend existing:
- `internal/repositories/plaid.go` (or a new `transactions.go` repo) — add `GetTransactionByID`, `PatchTransactionCategory`
- `internal/handlers/transactions.go` — add `GetTransaction`, `PatchTransaction` handlers

### apply_to_past cap

500 rows inline for MVP. If a user has more than 500 historical transactions from the same merchant, only the 500 most recent are updated. This avoids long-running HTTP requests at private beta scale. No background job needed yet.

---

## 6. Mobile UX

### Navigation

Transaction rows (currently displayed in AccountsScreen or wherever the transaction list lives) get an `onPress` handler that navigates to `TransactionDetailScreen`, passing the transaction ID.

### TransactionDetailScreen

New screen at `src/screens/TransactionDetailScreen.tsx`.

On mount: fetches `GET /v1/transactions/:id`. Shows loading spinner while fetching.

Displays:
- Merchant name (large) + formatted amount
- Date + account name
- Category chip (tappable)
- Pending / excluded badges if applicable

### Category Picker

Bottom sheet (modal). Shows all user categories fetched from `GET /v1/categories`, grouped by type (expense / income / transfer).

On category select:
1. Dismiss sheet
2. Call `PATCH /v1/transactions/:id` with new `category_id`
3. Update local state optimistically
4. Show rule prompt (see below)

### Rule Prompt

After a successful category patch, show an alert/modal:

> **Always categorize "[merchant]" as [category]?**
>
> [Yes] [No]

If **Yes**:
- Show a second confirmation with a checkbox: "Also fix past transactions from this merchant" (default: unchecked)
- On confirm: call `POST /v1/rules` with `apply_to_past` matching the checkbox state
- Show a brief success toast: "Rule saved" (or "Rule saved — N past transactions updated")

If **No**: dismiss, no further action.

### State invalidation

After a successful category patch or rule creation, invalidate (refetch) the transaction list so the corrected category is reflected when the user navigates back.

---

## 7. Error Handling

| Scenario | Behavior |
|---|---|
| PATCH with invalid category_id | 400, show inline error in picker |
| PATCH on another user's transaction | 404, show generic error toast |
| POST /v1/rules with missing fields | 400, show toast |
| Network error on any call | Show retry toast, leave UI in pre-change state |

---

## 8. Out of Scope for This Iteration

- Rule management screen (list/delete rules from Settings) — deferred, GET/DELETE endpoints are built but no mobile UI yet
- `apply_to_past` background job for >500 rows — deferred to post-MVP
- Auto-apply rules during new transaction sync — this should already work if the categorization pipeline checks `category_rules` before falling back to provider categories; that wiring is separate from this feature
- Excluding transactions (`is_excluded`) — separate patch field, deferred
- Needs-review flag management — separate, deferred
